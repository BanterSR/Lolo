package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
)

func (g *Game) SupplyBoxInfo(s *model.Player, msg *alg.GameMsg) {
	rsp, _ := defaultActivityService().supplyBoxInfo(s)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) SupplyBoxReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().supplyBoxReward(s)
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}
