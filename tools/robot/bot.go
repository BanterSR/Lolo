package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

const handshakeTimeout = 10 * time.Second

var (
	errTimeout      = errors.New("等待响应超时")
	errDisconnected = errors.New("连接已断开")
)

// 每类失败仅打印首个原因到 stderr，避免刷屏又不至于让用户对着计数器猜。
var (
	logSdkErrOnce      sync.Once
	logConnErrOnce     sync.Once
	logLoginErrOnce    sync.Once
	logSceneAnchorOnce sync.Once
)

func logOnce(o *sync.Once, format string, args ...any) {
	o.Do(func() { fmt.Fprintf(os.Stderr, "[robot] "+format+"\n", args...) })
}

// Bot 是单个纯协议机器人：SDK 登录 -> 连接网关 -> VerifyLoginToken -> PlayerLogin -> 场景循环。
type Bot struct {
	id     int
	userId uint32 // 网关下发的游戏 UserId，用于在 SceneDataNotice 中定位自己
	cfg    *Config
	m      *Metrics

	conn *Client

	mu       sync.Mutex
	packetId uint32
	inflight map[uint32]time.Time // packetId -> 发送时刻，用于计算 RTT

	hsCh     chan *alg.GameMsg // 握手响应投递通道
	dead     chan struct{}     // 读协程退出信号
	deadOnce sync.Once
}

func newBot(id int, cfg *Config, m *Metrics) *Bot {
	return &Bot{
		id:       id,
		cfg:      cfg,
		m:        m,
		inflight: make(map[uint32]time.Time),
		hsCh:     make(chan *alg.GameMsg, 16),
		dead:     make(chan struct{}),
	}
}

func (b *Bot) username() string { return fmt.Sprintf("%s%d", b.cfg.Prefix, b.id) }

func (b *Bot) nextPacketId() uint32 {
	b.mu.Lock()
	b.packetId++
	id := b.packetId
	b.mu.Unlock()
	return id
}

// send 发送请求并登记 inflight，返回后由读协程按 packetId 回填 RTT。
// 仅用于响应会回显 packetId 的请求。
func (b *Bot) send(msg pb.Message) error {
	id := b.nextPacketId()
	b.mu.Lock()
	b.inflight[id] = time.Now()
	b.mu.Unlock()
	atomic.AddInt64(&b.m.reqSent, 1)
	if err := b.conn.Send(id, msg); err != nil {
		b.mu.Lock()
		delete(b.inflight, id)
		b.mu.Unlock()
		return err
	}
	return nil
}

// sendUntracked 发送但不登记 inflight，用于响应不回显 packetId 的请求
// （如 VerifyLoginTokenReq 的响应固定以 packetId=0 返回，靠消息类型相关联）。
func (b *Bot) sendUntracked(msg pb.Message) error {
	id := b.nextPacketId()
	atomic.AddInt64(&b.m.reqSent, 1)
	return b.conn.Send(id, msg)
}

// readLoop 是该连接唯一的读方：回填 RTT、投递握手响应、排空通知。
func (b *Bot) readLoop() {
	defer b.deadOnce.Do(func() { close(b.dead) })
	for {
		msg, err := b.conn.Recv()
		if err != nil {
			return
		}
		atomic.AddInt64(&b.m.rspRecv, 1)

		if msg.PacketId != 0 {
			b.mu.Lock()
			t, ok := b.inflight[msg.PacketId]
			if ok {
				delete(b.inflight, msg.PacketId)
			}
			b.mu.Unlock()
			if ok {
				b.m.observeRTT(time.Since(t))
			}
		}

		switch msg.Body.(type) {
		case *proto.VerifyLoginTokenRsp, *proto.PlayerLoginRsp, *proto.PlayerMainDataRsp, *proto.SceneDataNotice:
			select {
			case b.hsCh <- msg:
			default:
			}
		}
	}
}

// waitHS 等待满足 match 的握手响应，或超时/断线。
func (b *Bot) waitHS(match func(*alg.GameMsg) bool) (*alg.GameMsg, error) {
	timer := time.NewTimer(handshakeTimeout)
	defer timer.Stop()
	for {
		select {
		case msg := <-b.hsCh:
			if match(msg) {
				return msg, nil
			}
		case <-b.dead:
			return nil, errDisconnected
		case <-timer.C:
			return nil, errTimeout
		}
	}
}

// sweepInflight 回收超时未响应的请求，计入 timeout。
func (b *Bot) sweepInflight() {
	now := time.Now()
	b.mu.Lock()
	for id, t := range b.inflight {
		if now.Sub(t) > 10*time.Second {
			delete(b.inflight, id)
			atomic.AddInt64(&b.m.rspTimeout, 1)
		}
	}
	b.mu.Unlock()
}

// Run 执行完整生命周期，阻塞直到 ctx 取消或连接断开。
func (b *Bot) Run(ctx context.Context) {
	if !b.login() {
		// 登录失败后退避,避免瞬时重试风暴反复猛敲(尤其服务端已过载)时把失败数刷爆。
		sleepCtx(ctx, b.cfg.RetryBackoff.D())
		return
	}
	defer b.conn.Close()

	// 场景循环
	b.runScenario(ctx)

	// 若因 socket 断开而非主动取消退出，计一次断线
	select {
	case <-b.dead:
		if ctx.Err() == nil {
			atomic.AddInt64(&b.m.disconnects, 1)
		}
	default:
	}
}

// login 完成 SDK 登录 → 连接网关 → VerifyLoginToken → PlayerLogin。
// 成功返回 true 且读协程已启动、连接交由 Run 管理；任一步失败自行关闭连接并返回 false。
func (b *Bot) login() bool {
	uid, gateToken, err := SdkLogin(b.cfg.Sdk, b.username(), b.cfg.Password)
	if err != nil {
		atomic.AddInt64(&b.m.sdkLoginFail, 1)
		logOnce(&logSdkErrOnce, "SDK 登录失败（账号 %s）: %v", b.username(), err)
		return false
	}
	atomic.AddInt64(&b.m.sdkLoginOK, 1)

	conn, err := Dial(b.cfg.Gate, 10*time.Second, b.m)
	if err != nil {
		atomic.AddInt64(&b.m.connectFail, 1)
		logOnce(&logConnErrOnce, "网关连接失败（%s）: %v", b.cfg.Gate, err)
		return false
	}
	atomic.AddInt64(&b.m.connectOK, 1)
	b.conn = conn
	go b.readLoop()

	ok := false
	defer func() {
		if !ok {
			conn.Close() // 握手中途失败则关闭；成功则由 Run 的 defer 负责
		}
	}()

	// VerifyLoginToken —— 必须是首个包（响应以 packetId=0 返回，不做 RTT 跟踪）
	if err := b.sendUntracked(&proto.VerifyLoginTokenReq{
		AccountType: 1,
		SdkUid:      uid,
		LoginToken:  gateToken,
		ChannelCode: "lolo",
		DeviceUuid:  b.username(),
	}); err != nil {
		atomic.AddInt64(&b.m.loginFail, 1)
		return false
	}
	vmsg, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.VerifyLoginTokenRsp)
		return ok
	})
	if err != nil {
		atomic.AddInt64(&b.m.loginFail, 1)
		logOnce(&logLoginErrOnce, "VerifyLoginToken 无响应: %v", err)
		return false
	}
	b.userId = vmsg.Body.(*proto.VerifyLoginTokenRsp).GetUserId()
	if b.userId == 0 {
		atomic.AddInt64(&b.m.loginFail, 1)
		logOnce(&logLoginErrOnce, "VerifyLoginToken 被拒（UserId=0，检查 SdkUid/GateToken）")
		return false
	}

	// PlayerLogin
	if err := b.send(&proto.PlayerLoginReq{
		Lang:            "zh",
		ClientVersion:   "1.0.0",
		ResourceVersion: "1.0.0",
		DeviceUuid:      b.username(),
		DeviceModel:     "robot",
		OsName:          "windows",
		OsVer:           "10",
		Network:         "wifi",
	}); err != nil {
		atomic.AddInt64(&b.m.loginFail, 1)
		return false
	}
	pmsg, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.PlayerLoginRsp)
		return ok
	})
	if err != nil {
		atomic.AddInt64(&b.m.loginFail, 1)
		logOnce(&logLoginErrOnce, "PlayerLogin 无响应: %v", err)
		return false
	}
	if rsp := pmsg.Body.(*proto.PlayerLoginRsp); rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		atomic.AddInt64(&b.m.loginFail, 1)
		logOnce(&logLoginErrOnce, "PlayerLogin 被拒: status=%v", rsp.GetStatus())
		return false
	} else if rsp.PlayerName == "" {
		// 改名
		b.send(&proto.ChangeNickNameReq{
			NickName: b.username(),
			Birthday: "1990-01-01",
		})
	}

	atomic.AddInt64(&b.m.loginOK, 1)

	ok = true
	return true
}

// sleepCtx 睡眠 d，但 ctx 取消时立即返回。
func sleepCtx(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
