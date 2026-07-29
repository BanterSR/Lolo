package game

import (
	"errors"
	"fmt"
	"gucooing/lolo/pkg/cache"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/config"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/proto"
)

type Game struct {
	router              *gin.Engine // http 服务器
	gateTaskChan        chan GateTask
	userMap             map[uint32]*model.Player
	handlerFuncRouteMap map[uint32]HandlerFunc
	botCache            *cache.Cache[uint32, BotInterface]
	onlyUserId          atomic.Uint32
	wordInfo            *WordInfo
	worldTask           *model.WorldTask
	chatInfo            *ChatInfo
	checkPlayerTimer    *time.Timer
	gmChan              chan bool
	doneChan            chan struct{}
}

type GateTask interface {
	UserID() uint32
}

// 玩家消息
type PlayerMsg struct {
	UserId uint32
	UUID   string
	Conn   ofnet.Conn
	*alg.GameMsg
}

func (p *PlayerMsg) UserID() uint32 { return p.UserId }

// 下线指定玩家
type KillPlayer struct {
	UserId     uint32
	UUID       string
	Reason     proto.PlayerOfflineReason // 下线原因
	KillPlayer bool                      // 是否完整下线玩家
}

func (k *KillPlayer) UserID() uint32 { return k.UserId }

// ErrPlayerOffline 目标玩家不在线(未加载到内存),操作未执行。
var ErrPlayerOffline = errors.New("玩家不在线")

// FuncTask 通用玩家任务:把"对在线玩家的操作"投递到 game 主循环上串行执行,
// 让 GM/HTTP/bot 回调等外部 goroutine 无需加锁即可安全操作 Player。
type FuncTask struct {
	UserId uint32
	Fn     func(s *model.Player) (any, error) // 在主循环上执行,可直接读写 Player,返回结果与错误
	reply  chan funcTaskResult                // 非nil时回传执行结果
}

// funcTaskResult FuncTask 的执行结果
type funcTaskResult struct {
	msg any
	err error
}

func (f *FuncTask) UserID() uint32 { return f.UserId }

func NewGame(router *gin.Engine) *Game {
	conf := config.GetGame()
	log.NewGame()
	g := &Game{
		router:       router,
		worldTask:    model.NewWorldTask(),
		gateTaskChan: make(chan GateTask, conf.MsgChanSize),
		userMap:      make(map[uint32]*model.Player, 1000),
		doneChan:     make(chan struct{}),
		botCache:     cache.New[uint32, BotInterface](0),
	}
	g.newRouter()
	// 初始化场景配置
	model.ChannelTick = time.Duration(alg.MaxInt(int(model.ChannelTick.Milliseconds()), gdconf.GetConstant().ChannelTick)) * time.Millisecond
	model.OneSTickCount = int64(time.Second / model.ChannelTick)

	go g.gameMainLoop()
	return g
}

// 游戏主线程
func (g *Game) gameMainLoop() {
	runtime.LockOSThread()
	g.checkPlayerTimer = time.NewTimer(3 * time.Minute) // 3分钟检查一次玩家
	syncWorldTimer := time.NewTimer(model.ChannelTick)
	defer func() {
		log.Game.Info("game主线程停止")
		runtime.UnlockOSThread()
		if err := recover(); err != nil {
			log.Game.Error("----------------------------------------------------------------------------")
			log.Game.Error("出现未知错误请将当前控制台信息粘贴到 https://github.com/BanterSR/Lolo/issues 进行反馈")
			log.Game.Error("!!! GAME MAIN LOOP PANIC !!!")
			log.Game.Errorf("error: %s", err)
			log.Game.Errorf("Stack trace: %s", string(debug.Stack()))
			log.Game.Error("----------------------------------------------------------------------------")
			g.Close()
		}
	}()
	for {
		select {
		case <-g.doneChan:
			return
		case <-syncWorldTimer.C:
			g.worldTask.Tick()
			syncWorldTimer.Reset(model.ChannelTick)
		case task := <-g.gateTaskChan:
			g.gateTask(task)
		case <-g.checkPlayerTimer.C:
			g.checkPlayer()
		}
	}
}

func (g *Game) gateTask(task GateTask) {
	switch t := task.(type) {
	case *PlayerMsg:
		g.routeHandle(t.Conn, t.UserId, t.UUID, t.GameMsg)
	case *KillPlayer:
		g.donePlayer(t)
	case *FuncTask:
		g.funcTask(t)
	}
}

// funcTask 在主循环上执行外部投递的玩家操作。
func (g *Game) funcTask(t *FuncTask) {
	player := g.GetUser(t.UserId)
	if player == nil {
		if t.reply != nil {
			t.reply <- funcTaskResult{err: fmt.Errorf("[PlayerID:%d]%s", t.UserId, ErrPlayerOffline)}
		}
		return
	}
	msg, err := t.Fn(player)
	if t.reply != nil {
		t.reply <- funcTaskResult{msg: msg, err: err}
	}
}

func (g *Game) send(s *model.Player, packetId uint32, payloadMsg pb.Message) {
	if s.NetFreeze {
		return
	}
	s.Conn.Send(packetId, payloadMsg)
}

func (g *Game) GetUser(userId uint32) *model.Player {
	player, ok := g.userMap[userId]
	if !ok {
		return nil
	}
	return player
}

func (g *Game) checkPlayer() {
	defer g.checkPlayerTimer.Reset(3 * time.Minute)
	playerList := make([]*model.Player, 0)
	for _, player := range g.userMap {
		if player.IsOffline() {
			g.kickPlayer(player)
			playerList = append(playerList, player)
		}
		if player.IsSave() {
			player.SavePlayer()
		}
	}
	for _, player := range playerList {
		delete(g.userMap, player.UserId)
	}
}

// gate侧通知下线
func (g *Game) donePlayer(k *KillPlayer) {
	player := g.GetUser(k.UserId)
	if player == nil || !player.Online ||
		player.LoginUUID != k.UUID {
		return
	}
	g.offlinePlayer(player, k.Reason)
	if k.KillPlayer {
		player.Online = false
		// 退出世界
		g.getWordInfo().killScenePlayer(player)
		// 退出聊天频道
		g.getChatInfo().killChannelUser(player)
		log.Game.Debugf("玩家:%v 离线", player.UserId)
	}
}

// 仅做客户端下线
func (g *Game) offlinePlayer(player *model.Player, reason proto.PlayerOfflineReason) {
	player2 := g.GetUser(player.UserId)
	if player2 == nil || !player2.Online ||
		player.LoginUUID != player2.LoginUUID || player.NetFreeze {
		return
	}
	if reason != proto.PlayerOfflineReason_PlayerOfflineReason_None {
		g.send(player, 0, &proto.PlayerOfflineRsp{
			Status:             proto.StatusCode_StatusCode_Ok,
			Reason:             reason,
			ServerNextOpenTime: 0,
		})
	}
	if player.Conn != nil {
		player.Conn.Close()
	}
	player.NetFreeze = true
	scenePlayer := g.getWordInfo().getScenePlayer(player)
	if scenePlayer != nil {
		scenePlayer.NetFreeze = true
	}
}

// 彻底移除玩家
func (g *Game) kickPlayer(player *model.Player) {
	player2 := g.GetUser(player.UserId)
	if player2 == nil || !player2.Online ||
		player.LoginUUID != player2.LoginUUID {
		return
	}
	player.Online = false
	// 退出世界
	g.getWordInfo().killScenePlayer(player)
	// 退出聊天频道
	g.getChatInfo().killChannelUser(player)
	log.Game.Debugf("玩家:%v 离线", player.UserId)
}

func (g *Game) GetGateTask() chan GateTask {
	return g.gateTaskChan
}

// PostPlayerFunc 投递一个针对在线玩家的操作到 game 主循环异步执行(不等待结果)。
// 仅供非主循环 goroutine(GM/HTTP/bot 回调等)调用;严禁在主循环内部调用,否则可能自锁。
func (g *Game) PostPlayerFunc(userId uint32, fn func(s *model.Player)) {
	g.gateTaskChan <- &FuncTask{UserId: userId, Fn: func(s *model.Player) (any, error) {
		fn(s)
		return nil, nil
	}}
}

// InvokePlayerFunc 投递操作并等待主循环执行完成,返回 fn 的结果与错误。
// 玩家不在线时返回 ErrPlayerOffline;timeout<=0 表示一直等待。
// 阻塞的是调用方 goroutine,不是主循环。仅供非主循环 goroutine 调用。
func (g *Game) InvokePlayerFunc(userId uint32, fn func(s *model.Player) (any, error), timeout time.Duration) (any, error) {
	reply := make(chan funcTaskResult, 1)
	g.gateTaskChan <- &FuncTask{UserId: userId, Fn: fn, reply: reply}
	if timeout <= 0 {
		res := <-reply
		return res.msg, res.err
	}
	select {
	case res := <-reply:
		return res.msg, res.err
	case <-time.After(timeout):
		return nil, errors.New("game主循环执行超时")
	}
}

func (g *Game) Close() {
	close(g.doneChan)
	g.checkPlayer()
	for _, player := range g.userMap {
		g.send(player, 0, &proto.PlayerOfflineRsp{
			Status:             proto.StatusCode_StatusCode_Ok,
			Reason:             proto.PlayerOfflineReason_PlayerOfflineReason_ServerShutdown,
			ServerNextOpenTime: 0,
		})
		g.kickPlayer(player)
	}
	log.Game.Infof("game退出完成")
}
