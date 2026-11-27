package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
)

func (g *Game) AbyssInfo(s *model.Player, msg *alg.GameMsg) {
	rsp := defaultActivityService().abyssInfo(s)
	defer g.send(s, msg.PacketId, rsp)
}
