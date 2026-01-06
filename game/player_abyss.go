package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) AbyssInfo(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.AbyssInfoReq)
	rsp := &proto.AbyssInfoRsp{
		Status:             proto.StatusCode_StatusCode_Ok,
		AbyssInfo:          new(proto.AbyssInfo),
		InProgressSeasonId: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
}
