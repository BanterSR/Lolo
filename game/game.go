package game

import (
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
	userMap             *cache.Cache[uint32, *model.Player]
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

// SnapshotPlayer 请求某在线玩家的实时快照:在主线程内只读取少量高价值字段(不整体序列化),避免高频序列化开销与并发竞争
type SnapshotPlayer struct {
	UserId uint32
	Reply  chan map[string]any
}

func (s *SnapshotPlayer) UserID() uint32 { return s.UserId }

func NewGame(router *gin.Engine) *Game {
	conf := config.GetGame()
	log.NewGame()
	g := &Game{
		router:       router,
		worldTask:    model.NewWorldTask(),
		gateTaskChan: make(chan GateTask, conf.MsgChanSize),
		userMap:      cache.New[uint32, *model.Player](0),
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
	case *SnapshotPlayer:
		if player := g.GetUser(t.UserId); player != nil {
			t.Reply <- g.livePlayerSnapshot(player)
		} else {
			t.Reply <- nil
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
	player, ok := g.userMap.Get(userId)
	if !ok {
		return nil
	}
	return player
}

// OnlinePlayerCount 当前内存中加载的玩家数(供 mcp 只读查询)
func (g *Game) OnlinePlayerCount() int {
	count := 0
	g.userMap.Range(func(_ uint32, _ *model.Player) bool {
		count++
		return true
	})
	return count
}

// IsPlayerOnline 玩家是否在内存中(供 mcp 只读查询)
func (g *Game) IsPlayerOnline(userId uint32) bool {
	_, ok := g.userMap.Get(userId)
	return ok
}

// LivePlayerInfo 取在线玩家的实时快照(供 mcp 只读查询)。
// 经游戏主线程读取:唯一写线程内只挑少量高价值字段,既避免并发竞争,又避免高频全量序列化开销。
// 玩家不在线或超时返回 false。
func (g *Game) LivePlayerInfo(userId uint32) (map[string]any, bool) {
	if !g.IsPlayerOnline(userId) {
		return nil, false
	}
	reply := make(chan map[string]any, 1)
	select {
	case g.gateTaskChan <- &SnapshotPlayer{UserId: userId, Reply: reply}:
	case <-time.After(2 * time.Second):
		return nil, false
	}
	select {
	case info := <-reply:
		return info, info != nil
	case <-time.After(2 * time.Second):
		return nil, false
	}
}

// livePlayerSnapshot 在主线程内构造紧凑实时快照:仅读取少量简单字段,不整体序列化玩家
func (g *Game) livePlayerSnapshot(p *model.Player) map[string]any {
	h := map[string]any{
		"userId":     p.UserId,
		"nickName":   p.NickName,
		"online":     p.Online,
		"activeTime": p.ActiveTime.Unix(),
	}
	if p.Scene != nil {
		h["worldLevel"] = p.Scene.GetWorldLevel()
	}
	if p.Character != nil {
		h["characterCount"] = len(p.Character.CharacterMap)
	}
	if p.Item != nil {
		h["itemCount"] = len(p.Item.ItemBaseInfo)
	}
	if p.Team != nil && p.Team.TeamInfo != nil {
		h["team"] = []uint32{p.Team.TeamInfo.Char1, p.Team.TeamInfo.Char2, p.Team.TeamInfo.Char3}
	}
	// 当前场景与位置(世界层实时状态,主线程内读取避免竞争)
	if sp := g.getWordInfo().getScenePlayer(p); sp != nil {
		h["channelId"] = sp.ChannelId
		if sp.CurScene != nil {
			h["sceneId"] = sp.CurScene.GetSceneId()
			if pos := sp.CurScene.GetPos(); pos != nil {
				// 坐标为定点数(见 gdconf.ConfigVector3ToProtoVector3,已放大),decimalPlaces 表示小数位
				h["pos"] = map[string]any{"x": pos.GetX(), "y": pos.GetY(), "z": pos.GetZ(), "decimalPlaces": pos.GetDecimalPlaces()}
			}
		}
	}
	return h
}

func (g *Game) checkPlayer() {
	defer g.checkPlayerTimer.Reset(3 * time.Minute)
	playerList := make([]*model.Player, 0)
	g.userMap.Range(func(key uint32, player *model.Player) bool {
		if player.IsOffline() {
			g.kickPlayer(player)
			playerList = append(playerList, player)
		}
		if player.IsSave() {
			player.SavePlayer()
		}
		return true
	})
	for _, player := range playerList {
		g.userMap.Del(player.UserId)
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
	g.userMap.Range(func(key uint32, player *model.Player) bool {
		g.send(player, 0, &proto.PlayerOfflineRsp{
			Status:             proto.StatusCode_StatusCode_Ok,
			Reason:             proto.PlayerOfflineReason_PlayerOfflineReason_ServerShutdown,
			ServerNextOpenTime: 0,
		})
		g.kickPlayer(player)
		return true
	})
	log.Game.Infof("game退出完成")
}
