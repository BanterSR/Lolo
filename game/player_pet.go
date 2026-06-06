package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetPetReq(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetPetReq)
	rsp := &proto.GetPetRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Pets:     make([]*proto.PetInstance, 0),
		TotalNum: 0,
		EndIndex: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
}
