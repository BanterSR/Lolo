package game

import (
	"testing"

	"github.com/gookit/slog"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func TestRouteHandleContainsMaliciousRequestPanic(t *testing.T) {
	previous := log.Game
	log.Game = slog.NewStdLogger()
	defer func() { log.Game = previous }()

	player := &model.Player{UserId: 1, Online: true, LoginUUID: "session"}
	game := &Game{
		userMap: map[uint32]*model.Player{1: player},
		handlerFuncRouteMap: map[uint32]HandlerFunc{
			999: func(*model.Player, *alg.GameMsg) { panic("synthetic malformed request") },
		},
	}
	game.routeHandle(nil, 1, "session", &alg.GameMsg{
		PacketHead: &proto.PacketHead{MsgId: 999},
	})
	if player.Online || !player.NetFreeze {
		t.Fatal("panicking player was not discarded")
	}
	if game.GetUser(1) != nil {
		t.Fatal("panicking player remained in the online map")
	}
}

func TestRouteHandleStillProcessesNormalRequestAfterPanic(t *testing.T) {
	previous := log.Game
	log.Game = slog.NewStdLogger()
	defer func() { log.Game = previous }()

	called := false
	game := &Game{
		userMap: map[uint32]*model.Player{
			1: {UserId: 1, Online: true, LoginUUID: "session"},
		},
		handlerFuncRouteMap: map[uint32]HandlerFunc{
			998: func(*model.Player, *alg.GameMsg) { called = true },
		},
	}
	game.routeHandle(nil, 1, "session", &alg.GameMsg{
		PacketHead: &proto.PacketHead{MsgId: 998},
	})
	if !called {
		t.Fatal("normal request was not processed")
	}
}
