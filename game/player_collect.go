package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
	"math/rand/v2"
)

func (g *Game) GetCollectItemIds(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetCollectItemIdsReq)
	rsp := &proto.GetCollectItemIdsRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		ItemIds: make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, scene := range s.GetSceneModel().GetSceneMap() {
		for _, collection := range scene.GetCollections() {
			for _, v := range collection.ItemMap {
				alg.AddLists(&rsp.ItemIds, v.ItemId)
			}
		}
	}
}

func (g *Game) Collecting(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CollectingReq)
	rsp := &proto.CollectingRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		Collections: make([]*proto.CollectionData, 0),
		Items:       make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	conf := gdconf.GetCollectionItem(req.ItemId)
	if conf == nil || scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	info := s.GetSceneModel().GetSceneInfo(scenePlayer.SceneId).
		GetCollectionInfo(proto.ECollectionType(conf.NewCollectionType))
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if _, ok := info.ItemMap[req.ItemId]; ok {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	info.ItemMap[req.ItemId] = &model.PBCollectionRewardData{ItemId: req.ItemId}
	alg.AddList(&rsp.Collections, info.CollectionData())
	// 获取奖励
	for _, reward := range gdconf.GetCollectionReward(conf) {
		alg.AddList(&rsp.Items,
			s.AddAllTypeItem(
				uint32(reward.ItemID),
				int64(rand.Int32N(reward.ItemMaxCount)+reward.ItemMinCount),
			).
				AddItemDetail())
	}
}

func (g *Game) Gather(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GatherReq)
	rsp := &proto.GatherRsp{
		Status:           proto.StatusCode_StatusCode_Ok,
		Index:            0,
		Items:            make([]*proto.ItemDetail, 0),
		GroupGatherLimit: nil,
		SceneGatherLimit: nil,
		ItemLevel:        0,
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) TreasureBoxOpen(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.TreasureBoxOpenReq)
	rsp := &proto.TreasureBoxOpenRsp{
		Status:          proto.StatusCode_StatusCode_Ok,
		Items:           make([]*proto.ItemDetail, 0),
		NextRefreshTime: 0,
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
