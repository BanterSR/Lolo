package game

import (
	"testing"
	"time"

	"github.com/gookit/slog"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/log"
)

func TestGameCloseIsIdempotent(t *testing.T) {
	previous := log.Game
	log.Game = slog.NewStdLogger()
	defer func() { log.Game = previous }()

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	game := &Game{
		userMap:          make(map[uint32]*model.Player),
		doneChan:         make(chan struct{}),
		checkPlayerTimer: timer,
	}
	game.Close()
	game.Close()

	select {
	case <-game.doneChan:
	default:
		t.Fatal("close did not signal the game loop")
	}
}
