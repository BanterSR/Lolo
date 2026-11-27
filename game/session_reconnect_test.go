package game

import (
	"testing"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type sessionTestConn struct {
	lifeTestConn
	closed bool
}

func (c *sessionTestConn) Close() { c.closed = true }

func newSessionTestGame(t *testing.T) (*Game, *model.Player, *sessionTestConn, *int) {
	t.Helper()
	previous := sessionSavePlayer
	saves := 0
	sessionSavePlayer = func(*model.Player) error { saves++; return nil }
	t.Cleanup(func() { sessionSavePlayer = previous })
	conn := &sessionTestConn{}
	player := &model.Player{UserId: 42, LoginUUID: "session-a", Conn: conn, Online: true}
	scene := &model.ScenePlayerInfo{Player: player, SceneId: 1, Team: &model.TeamInfo{Char1: 1}, TempPack: model.NewSceneTempPack()}
	game := &Game{userMap: map[uint32]*model.Player{player.UserId: player}}
	game.getWordInfo().allScenePlayer.Store(player.UserId, &ScenePlayer{Player: player, CurScene: scene, ChannelId: 1})
	return game, player, conn, &saves
}

func TestDisconnectReconnectKeepsCachedStateAndIgnoresStaleSession(t *testing.T) {
	game, player, oldConn, saves := newSessionTestGame(t)
	game.donePlayer(&KillPlayer{UserId: player.UserId, UUID: "session-a", Reason: proto.PlayerOfflineReason_PlayerOfflineReason_NONE})
	if !player.Online || !player.NetFreeze || player.LoginUUID != "session-a" || !oldConn.closed || *saves != 1 ||
		game.getWordInfo().getScenePlayer(player) == nil {
		t.Fatalf("disconnect did not preserve reconnectable state: player=%+v closed=%v saves=%d", player, oldConn.closed, *saves)
	}

	newConn := &sessionTestConn{}
	player.Init(newConn, "session-b")
	if scenePlayer := game.getWordInfo().addScenePlayer(player); scenePlayer == nil || scenePlayer.NetFreeze {
		t.Fatal("reconnect did not unfreeze the existing scene player")
	}
	game.donePlayer(&KillPlayer{UserId: player.UserId, UUID: "session-a", Reason: proto.PlayerOfflineReason_PlayerOfflineReason_ANOTHER_LOGIN})
	if newConn.closed || player.NetFreeze || player.LoginUUID != "session-b" || *saves != 1 {
		t.Fatalf("stale disconnect affected the new session: player=%+v closed=%v saves=%d", player, newConn.closed, *saves)
	}

	game.handlerFuncRouteMap = map[uint32]HandlerFunc{cmd.PlayerPingReq: game.PlayerPing}
	game.routeHandle(newConn, player.UserId, "session-b", &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 9, MsgId: cmd.PlayerPingReq, SeqId: 1},
		Body:       &proto.PlayerPingReq{ClientTimeMs: 123},
	})
	if rsp, ok := newConn.sent[len(newConn.sent)-1].(*proto.PlayerPingRsp); !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_OK || rsp.GetClientTimeMs() != 123 {
		t.Fatalf("reconnected session could not route protocol traffic: %+v", newConn.sent)
	}
	before := len(newConn.sent)
	game.routeHandle(oldConn, player.UserId, "session-a", &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 10, MsgId: cmd.PlayerPingReq, SeqId: 2},
		Body:       &proto.PlayerPingReq{ClientTimeMs: 456},
	})
	if len(oldConn.sent) != 0 || len(newConn.sent) != before {
		t.Fatal("stale UUID was allowed to route a request after reconnect")
	}
}

func TestDuplicateLoginOfflinesOldSessionBeforeRebind(t *testing.T) {
	game, player, oldConn, saves := newSessionTestGame(t)
	game.offlinePlayer(player, proto.PlayerOfflineReason_PlayerOfflineReason_ANOTHER_LOGIN)
	if !oldConn.closed || !player.NetFreeze || *saves != 1 {
		t.Fatalf("duplicate login did not close/save old session: closed=%v frozen=%v saves=%d", oldConn.closed, player.NetFreeze, *saves)
	}
	if len(oldConn.sent) != 1 {
		t.Fatalf("old session did not receive duplicate-login notice: %+v", oldConn.sent)
	}
	offline, ok := oldConn.sent[0].(*proto.PlayerOfflineRsp)
	if !ok || offline.GetReason() != proto.PlayerOfflineReason_PlayerOfflineReason_ANOTHER_LOGIN {
		t.Fatalf("wrong duplicate-login notice: %+v", oldConn.sent[0])
	}

	newConn := &sessionTestConn{}
	player.Init(newConn, "session-b")
	game.getWordInfo().addScenePlayer(player)
	game.donePlayer(&KillPlayer{UserId: player.UserId, UUID: "session-a", Reason: proto.PlayerOfflineReason_PlayerOfflineReason_NONE})
	if newConn.closed || player.NetFreeze || !player.Online || player.LoginUUID != "session-b" || *saves != 1 {
		t.Fatalf("old cleanup removed the replacement session: player=%+v closed=%v saves=%d", player, newConn.closed, *saves)
	}
}
