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

var (
	playerNum int64 = 0
)

type Game struct {
	router              *gin.Engine // http 服务器
	taskChan            chan TaskInterface
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

func (g *Game) LoadPlayerNum() int64 { return atomic.LoadInt64(&playerNum) }

type TaskInterface interface {
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
	Fn     func(s *model.Player) // 在主循环上执行,可直接读写 Player,返回结果与错误
	reply  chan error            // 非nil时回传执行结果
}

func (f *FuncTask) UserID() uint32 { return f.UserId }

func NewGame(router *gin.Engine) *Game {
	conf := config.GetGame()
	log.NewGame()
	g := &Game{
		router:    router,
		worldTask: model.NewWorldTask(),
		taskChan:  make(chan TaskInterface, conf.MsgChanSize),
		userMap:   make(map[uint32]*model.Player, 1000),
		doneChan:  make(chan struct{}),
		botCache:  cache.New[uint32, BotInterface](0),
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
		case task := <-g.taskChan:
			g.taskRun(task)
		case <-g.checkPlayerTimer.C:
			g.checkPlayer()
		}
	}
}

func (g *Game) taskRun(task TaskInterface) {
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
			t.reply <- fmt.Errorf("[PlayerID:%d]%s", t.UserId, ErrPlayerOffline)
		}
		return
	}
	t.Fn(player)
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
		atomic.AddInt64(&playerNum, -1)
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

func (g *Game) GetGateTask() chan TaskInterface {
	return g.taskChan
}

// 获取排队中的任务数量
func (g *Game) TaskNum() int {
	return len(g.taskChan)
}

// PostPlayerFunc 投递一个针对在线玩家的操作到 game 主循环异步执行(不等待结果)。
// 仅供非主循环 goroutine(GM/HTTP/bot 回调等)调用;严禁在主循环内部调用,否则可能自锁。
func (g *Game) PostPlayerFunc(userId uint32, fn func(s *model.Player)) {
	g.taskChan <- &FuncTask{UserId: userId, Fn: fn}
}

// InvokePlayerFunc 投递操作并等待主循环执行完成,返回 fn 的结果与错误。
// 玩家不在线时返回 ErrPlayerOffline;timeout<=0 表示一直等待。
// 阻塞的是调用方 goroutine,不是主循环。仅供非主循环 goroutine 调用。
func (g *Game) InvokePlayerFunc(userId uint32, fn func(s *model.Player), timeout time.Duration) error {
	reply := make(chan error, 1)
	g.taskChan <- &FuncTask{UserId: userId, Fn: fn, reply: reply}
	if timeout <= 0 {
		err := <-reply
		return err
	}
	select {
	case err := <-reply:
		return err
	case <-time.After(timeout):
		return errors.New("game主循环执行超时")
	}
}

func (g *Game) Close() {
	close(g.doneChan)
	g.checkPlayer()
	for _, player := range g.userMap {
		g.send(player, 0, &proto.PlayerOfflineRsp{
			Status:             proto.StatusCode_StatusCode_Ok,
			Reason:             proto.PlayerOfflineReason_PlayerOfflineReason_ServerShutdown,
			ServerNextOpenTime: time.Now().Add(10 * time.Minute).Unix(),
		})
		g.kickPlayer(player)
	}
	log.Game.Infof("game退出完成")
}
