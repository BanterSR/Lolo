package game

import (
	"runtime"
	"runtime/debug"
	"time"

	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/config"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
)

type Game struct {
	gameMsgChan         chan *GameMsg
	killUserChan        chan uint32
	userMap             map[uint32]*model.Player
	handlerFuncRouteMap map[uint32]HandlerFunc
	wordInfo            *WordInfo
	checkPlayerTimer    *time.Timer
	doneChan            chan struct{}
}

type GameMsg struct {
	UserId uint32
	Conn   ofnet.Conn
	*alg.GameMsg
}

func NewGame() *Game {
	conf := config.GetGame()
	log.NewGame()
	g := &Game{
		gameMsgChan:  make(chan *GameMsg, conf.MsgChanSize),
		killUserChan: make(chan uint32, 100),
		userMap:      make(map[uint32]*model.Player),
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
		log.Game.Infof("game主线程停止")
		runtime.UnlockOSThread()
		if err := recover(); err != nil {
			log.Game.Error("!!! GAME MAIN LOOP PANIC !!!")
			log.Game.Errorf("error: %s", err)
			log.Game.Errorf("Stack trace: %s", string(debug.Stack()))
		}
	}()
	for {
		select {
		case <-g.doneChan:
			return
		case msg := <-g.gameMsgChan:
			g.RouteHandle(msg.Conn, msg.UserId, msg.GameMsg)
		case <-g.checkPlayerTimer.C:
			g.checkPlayer()
		case userId := <-g.killUserChan:
			g.kickPlayer(userId)
		}
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
	playerList := make([]*model.Player, 0)
	for _, player := range g.userMap {
		if player.IsOffline() {
			g.kickPlayer(player.UserId)
		}
		playerList = append(playerList, player)
	}
	go func() {
		defer g.checkPlayerTimer.Reset(3 * time.Minute)
		for _, player := range playerList {
			if player.IsSave() {
				player.SavePlayer()
			}
		}
	}()
}

func (g *Game) kickPlayer(userId uint32) {
	player := g.GetUser(userId)
	if player == nil {
		return
	}
	// 退出世界
	g.getWordInfo().killScenePlayer(player)
	log.Game.Debugf("玩家:%v 离线", userId)
}

func (g *Game) GetGameMsgChan() chan *GameMsg {
	return g.gameMsgChan
}

func (g *Game) GetKillUserChan() chan uint32 {
	return g.killUserChan
}

func (g *Game) Close() {
	close(g.doneChan)
	g.checkPlayer()
	log.Game.Infof("game退出完成")
}
