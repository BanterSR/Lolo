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
	petMap := s.Item.GetItemPetInfoMap()
	for _, pet := range petMap {
		alg.AddLists(&rsp.Pets, pet.GetPbPetInstance())
	}
	rsp.TotalNum = uint32(len(petMap))
	rsp.EndIndex = uint32(len(petMap))
}

func (g *Game) ChangePet(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangePetReq)
	rsp := &proto.ChangePetRsp{
		Status:        proto.StatusCode_StatusCode_Ok,
		PetInstanceId: req.PetInstanceId,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdataPet)
	}()
	_, ok := s.GetItemModel().GetItemPetInfoMap()[req.PetInstanceId]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PetNotFound
		return
	}
	s.GetSceneModel().CurPetInstanceId = req.PetInstanceId
}
