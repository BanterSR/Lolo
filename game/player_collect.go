package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) Collecting(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.CollectingRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		Collections: make([]*proto.CollectionData, 0),
		Items:       make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetCollectMoonInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCollectMoonInfoReq)
	rsp := &proto.GetCollectMoonInfoRsp{
		Status:           proto.StatusCode_StatusCode_Ok,
		SceneId:          req.SceneId,
		CollectedMoonIds: make([]uint32, 0),
		EmotionMoons:     make([]*proto.EmotionMoonInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) CollectMoon(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CollectMoonReq)
	rsp := &proto.CollectMoonRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		MoonId:  req.MoonId,
		Rewards: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
