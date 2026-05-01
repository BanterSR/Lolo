package game

import (
	"math/rand/v2"
	"slices"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
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
	info.ItemMap[req.ItemId] = &model.PBCollectionRewardData{
		ItemId: req.ItemId,
		Status: proto.RewardStatus_RewardStatus_Reward,
	}
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

func (g *Game) CollectionReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CollectionRewardReq)
	rsp := &proto.CollectionRewardRsp{
		Status:               proto.StatusCode_StatusCode_Ok,
		CollectionRewardData: nil,
		Items:                make([]*proto.ItemDetail, 0),
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
	itemInfo, ok := info.ItemMap[req.ItemId]
	if !ok ||
		itemInfo.Status != proto.RewardStatus_RewardStatus_NotReward {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	// 获取奖励
	for _, reward := range gdconf.GetCollectionReward(conf) {
		alg.AddList(&rsp.Items,
			s.AddAllTypeItem(
				uint32(reward.ItemID),
				int64(rand.Int32N(reward.ItemMaxCount)+reward.ItemMinCount),
			).
				AddItemDetail())
	}
	itemInfo.Status = proto.RewardStatus_RewardStatus_Reward
	rsp.CollectionRewardData = itemInfo.PBCollectionRewardData()
}

func (g *Game) Gather(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GatherReq)
	rsp := &proto.GatherRsp{
		Status:           proto.StatusCode_StatusCode_Ok,
		Index:            req.GetGatherItem().GetIndex(),
		Items:            make([]*proto.ItemDetail, 0),
		GroupGatherLimit: new(proto.GroupGatherLimit),
		SceneGatherLimit: new(proto.SceneGatherLimit),
		ItemLevel:        0,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	conf := gdconf.GetSceneInfo(scenePlayer.SceneId).GatherPointInfo(req.GetGatherItem().GetIndex())
	gatherConf := gdconf.GetGatherConfigure(uint32(conf.GetGatherID()))
	rewardConf := gdconf.GetGatherRewardConfigure(req.GetGatherItem().GetReward())
	if gatherConf == nil || rewardConf == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	var t uint32
	rewards := make([]*excel.GatherRewardGroupInfo, 0)
	for _, info := range gatherConf.GatherGroupInfo {
		if info.Reward == req.GetGatherItem().GetReward() {
			t = uint32(info.NewGatherType)
			break
		}
	}
	for _, info := range rewardConf.GetGatherRewardGroupInfo() {
		if info.Lucky == req.GetGatherItem().GetIsLucky() {
			alg.AddList(&rewards, info)
		}
	}

	packItems := make([]*proto.ItemDetail, 0)
	for _, reward := range rewards {
		item := s.AddAllTypeItem(uint32(reward.ItemID), int64(reward.Count))
		alg.AddList(&rsp.Items, item.AddItemDetail())
		alg.AddList(&packItems, item.AddItemDetail())
	}
	g.PackNoticeByItems(s, packItems)

	sceneInfo := s.GetSceneModel().GetSceneInfo(scenePlayer.SceneId)
	info := sceneInfo.GetGatherLimit(t)
	info.GatherNum++

	rsp.SceneGatherLimit = sceneInfo.SceneGatherLimit()
	rsp.GroupGatherLimit.GatherLimit = info.GatherLimit()
}

func (g *Game) TreasureBoxOpen(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TreasureBoxOpenReq)
	rsp := &proto.TreasureBoxOpenRsp{
		Status:          proto.StatusCode_StatusCode_Ok,
		Items:           make([]*proto.ItemDetail, 0),
		NextRefreshTime: 0,
	}
	defer g.send(s, msg.PacketId, rsp)

	if req.GetLocType() != proto.TreasureBoxLocType_TreasureBoxLocType_Dungeon {
		return
	}

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	runtime := g.getDungeonRuntime(scenePlayer)
	if runtime == nil || !runtime.InDungeon {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	if runtime.TreasureBoxes == nil {
		runtime.TreasureBoxes = g.newDungeonTreasureBoxes(runtime.DungeonId)
	}

	boxIndex := req.GetTreasureBoxIndex()
	if boxIndex == 0 {
		boxIndex = 2
	}
	box, ok := runtime.TreasureBoxes[boxIndex]
	if !ok {
		box = &proto.TreasureBoxData{
			Index:           boxIndex,
			BoxId:           runtime.DungeonId*100 + boxIndex,
			Type:            proto.ETreasureBoxType_ETreasureBoxType_Normal,
			State:           proto.TreasureBoxState_TreasureBoxState_Close,
			NextRefreshTime: 0,
			Rewards:         make([]*proto.ItemDetail, 0),
		}
		runtime.TreasureBoxes[boxIndex] = box
	}
	box.State = proto.TreasureBoxState_TreasureBoxState_Open
	if len(box.Rewards) == 0 {
		box.Rewards = g.buildDungeonOpenRewards(runtime.DungeonId)
	}
	rsp.Items = append(rsp.Items, box.GetRewards()...)
	rsp.NextRefreshTime = box.GetNextRefreshTime()

	if !runtime.FinalBoxOpened {
		runtime.FinalBoxOpened = true
		g.send(s, 0, &proto.DungeonOpenFinalBoxNotice{
			Status:    proto.StatusCode_StatusCode_Ok,
			DungeonId: runtime.DungeonId,
		})
	}
}

func (g *Game) TreasureBoxPickup(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TreasureBoxPickupReq)
	rsp := &proto.TreasureBoxPickupRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	if req.GetLocType() != proto.TreasureBoxLocType_TreasureBoxLocType_Dungeon {
		return
	}

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	runtime := g.getDungeonRuntime(scenePlayer)
	if runtime == nil || !runtime.InDungeon {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	if runtime.TreasureBoxes == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if runtime.PickedBoxes == nil {
		runtime.PickedBoxes = make(map[uint32]bool)
	}

	boxIndex := req.GetBoxIndex()
	if boxIndex == 0 {
		boxIndex = 2
	}
	box := runtime.TreasureBoxes[boxIndex]
	if box == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if runtime.PickedBoxes[boxIndex] {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if len(box.Rewards) == 0 {
		box.State = proto.TreasureBoxState_TreasureBoxState_Open
		box.Rewards = g.buildDungeonOpenRewards(runtime.DungeonId)
		if !runtime.FinalBoxOpened {
			runtime.FinalBoxOpened = true
			g.send(s, 0, &proto.DungeonOpenFinalBoxNotice{
				Status:    proto.StatusCode_StatusCode_Ok,
				DungeonId: runtime.DungeonId,
			})
		}
	}

	packItems := make([]*proto.ItemDetail, 0, len(box.Rewards))
	for _, reward := range box.Rewards {
		if reward == nil || reward.GetMainItem() == nil {
			continue
		}
		itemId := reward.GetMainItem().GetItemId()
		if itemId == 0 {
			continue
		}
		num := int64(1)
		if base := reward.GetMainItem().GetBaseItem(); base != nil && base.GetNum() > 0 {
			num = base.GetNum()
		}

		addCtx := s.AddAllTypeItem(itemId, num)
		if addCtx == nil || addCtx.EBagItemTag == nil {
			continue
		}
		if item := addCtx.AddItemDetail(); item != nil {
			alg.AddList(&rsp.Items, item)
		}
		if item := addCtx.EBagItemTag.ItemDetail(); item != nil {
			alg.AddList(&packItems, item)
		}
	}
	if len(packItems) > 0 {
		g.PackNoticeByItems(s, packItems)
	}
	runtime.PickedBoxes[boxIndex] = true
}

func (g *Game) buildDungeonOpenRewards(dungeonId uint32) []*proto.ItemDetail {
	conf := gdconf.GetDungeonConfigure(dungeonId)
	rewards := make([]*proto.ItemDetail, 0)
	if conf != nil {
		rewardId := uint32(conf.GetRewardID())
		if rewardId != 0 && gdconf.GetRewardPool(rewardId) != nil {
			for _, reward := range gdconf.GetRewardItemPoolByRewardId(rewardId) {
				itemId := uint32(reward.GetItemID())
				num := scaleDungeonRewardNum(reward)
				item := g.buildTempRewardItemDetail(itemId, num)
				if item != nil {
					alg.AddList(&rewards, item)
				}
			}
		}
	}

	// 保底一份，确保副本最终箱子至少有可拾取奖励。
	if len(rewards) == 0 {
		if item := g.buildTempRewardItemDetail(6039, 4); item != nil {
			alg.AddList(&rewards, item)
		}
	}
	return rewards
}

func (g *Game) buildTempRewardItemDetail(itemId uint32, num int64) *proto.ItemDetail {
	if itemId == 0 || num <= 0 {
		return nil
	}
	itemConf := gdconf.GetItemConfigure(itemId)
	if itemConf == nil {
		return nil
	}
	return &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  itemId,
			ItemTag: proto.EBagItemTag(itemConf.GetNewBagItemTag()),
			Item: &proto.ItemInfo_BaseItem{
				BaseItem: &proto.BaseItem{
					ItemId: itemId,
					Num:    num,
				},
			},
		},
		PackType: proto.PackType_PackType_TempStorageArea,
	}
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
	info := s.GetSceneModel().GetSceneInfo(req.SceneId).
		GetCollectionInfo(proto.ECollectionType_ECollectionType_CollectMoonPiece)
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	rsp.CollectedMoonIds = info.CollectedMoonIds
}

func (g *Game) CollectMoon(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CollectMoonReq)
	rsp := &proto.CollectMoonRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		MoonId:  req.MoonId,
		Rewards: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	conf := gdconf.GetCollectionItem(req.MoonId)
	if conf == nil || scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	info := s.GetSceneModel().GetSceneInfo(scenePlayer.SceneId).
		GetCollectionInfo(proto.ECollectionType(conf.NewCollectionType))
	// 判断
	if slices.Contains(info.CollectedMoonIds, req.MoonId) {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	alg.AddSlice(&info.CollectedMoonIds, req.MoonId)
	// 获取奖励
	item := s.AddAllTypeItem(124, 5)
	g.PackNoticeByItems(s, []*proto.ItemDetail{item.ItemDetail()})
	alg.AddList(&rsp.Rewards, item.AddItemDetail())
}
