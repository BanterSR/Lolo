package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) ExploreInit(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.ExploreInitRsp{
		Status:          proto.StatusCode_StatusCode_OK,
		Explore:         make([]*proto.PlayerExploreInfo, 0),
		ActivityExplore: make([]*proto.PlayerExploreInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
