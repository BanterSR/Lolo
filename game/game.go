package game

import (
	"runtime"
	"runtime/debug"
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
	wordInfo            *WordInfo
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

func NewGame(router *gin.Engine) *Game {
	conf := config.GetGame()
	log.NewGame()
	g := &Game{
		router:       router,
		gateTaskChan: make(chan GateTask, conf.MsgChanSize),
		userMap:      make(map[uint32]*model.Player, 1000),
		doneChan:     make(chan struct{}),
	}
	g.newRouter()
	// 初始化场景配置
	channelTick = time.Duration(alg.MaxInt(int(channelTick.Milliseconds()), gdconf.GetConstant().ChannelTick)) * time.Millisecond
	oneSTickCount = int(time.Second / channelTick)

	go g.gameMainLoop()
	return g
}

// 游戏主线程
func (g *Game) gameMainLoop() {
	runtime.LockOSThread()
	g.checkPlayerTimer = time.NewTimer(3 * time.Minute) // 3分钟检查一次玩家
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
		player.LoginUUID != player2.LoginUUID {
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
		player.Conn = nil
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
