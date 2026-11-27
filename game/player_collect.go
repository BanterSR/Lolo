package game

import (
	"math/rand/v2"
	"slices"
	"sync"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

var (
	collectMu         sync.Mutex
	collectSavePlayer = func(s *model.Player) error { return s.SavePlayerNow() }
)

func persistCollectReward(s *model.Player, before model.ItemModelSnapshot) bool {
	if !s.Online {
		return true
	}
	if err := collectSavePlayer(s); err != nil {
		before.Restore()
		return false
	}
	return true
}

func (g *Game) GetCollectItemIds(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetCollectItemIdsReq)
	rsp := &proto.GetCollectItemIdsRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		ItemIds: make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, scene := range s.GetSceneModel().GetSceneMap() {
		if scene == nil {
			continue
		}
		for _, collection := range scene.GetCollections() {
			if collection == nil {
				continue
			}
			for _, v := range collection.ItemMap {
				if v != nil {
					alg.AddLists(&rsp.ItemIds, v.ItemId)
				}
			}
		}
	}
}

func (g *Game) Collecting(s *model.Player, msg *alg.GameMsg) {
	collectMu.Lock()
	defer collectMu.Unlock()
	req := msg.Body.(*proto.CollectingReq)
	rsp := &proto.CollectingRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		Collections: make([]*proto.CollectionData, 0),
		Items:       make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer, sceneId, ok := g.currentPlayerScene(s)
	if !ok || gdconf.GetSceneInfo(sceneId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	conf := gdconf.GetCollectionItem(req.GetItemId())
	sceneInfo := playerSceneInfo(s, sceneId)
	if conf == nil || scenePlayer == nil || sceneInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	info := sceneInfo.GetCollectionInfo(proto.ECollectionType(conf.NewCollectionType))
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}

	inventory := model.SnapshotItemModel(s.GetItemModel())
	tx, err := s.GetItemModel().BeginAtomic()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	defer tx.Rollback()
	if info.ItemMap == nil {
		info.ItemMap = make(map[uint32]*model.PBCollectionRewardData)
	}
	if _, ok := info.ItemMap[req.GetItemId()]; ok {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rewards, ok := collectionRewards(s, conf)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	items := addItemRewards(s, rewards)
	if items == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	info.ItemMap[req.GetItemId()] = &model.PBCollectionRewardData{
		ItemId: req.GetItemId(),
		Status: proto.RewardStatus_Reward,
	}
	rsp.Collections = append(rsp.Collections, info.CollectionData())
	rsp.Items = append(rsp.Items, items...)
	if tx.Commit() == nil || !persistCollectReward(s, inventory) {
		delete(info.ItemMap, req.GetItemId())
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
	}
}

func (g *Game) CollectionReward(s *model.Player, msg *alg.GameMsg) {
	collectMu.Lock()
	defer collectMu.Unlock()
	req := msg.Body.(*proto.CollectionRewardReq)
	rsp := &proto.CollectionRewardRsp{
		Status:               proto.StatusCode_StatusCode_OK,
		CollectionRewardData: nil,
		Items:                make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer, sceneId, ok := g.currentPlayerScene(s)
	if !ok || gdconf.GetSceneInfo(sceneId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	conf := gdconf.GetCollectionItem(req.GetItemId())
	sceneInfo := playerSceneInfo(s, sceneId)
	if conf == nil || scenePlayer == nil || sceneInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	info := sceneInfo.GetCollectionInfo(proto.ECollectionType(conf.NewCollectionType))
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}

	inventory := model.SnapshotItemModel(s.GetItemModel())
	tx, err := s.GetItemModel().BeginAtomic()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	defer tx.Rollback()
	itemInfo, ok := info.ItemMap[req.GetItemId()]
	if !ok || itemInfo == nil ||
		itemInfo.Status != proto.RewardStatus_NotReward {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rewards, ok := collectionRewards(s, conf)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	items := addItemRewards(s, rewards)
	if items == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	itemInfo.Status = proto.RewardStatus_Reward
	rsp.CollectionRewardData = itemInfo.PBCollectionRewardData()
	rsp.Items = append(rsp.Items, items...)
	if tx.Commit() == nil || !persistCollectReward(s, inventory) {
		itemInfo.Status = proto.RewardStatus_NotReward
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
	}
}

func (g *Game) Gather(s *model.Player, msg *alg.GameMsg) {
	collectMu.Lock()
	defer collectMu.Unlock()
	req := msg.Body.(*proto.GatherReq)
	rsp := &proto.GatherRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		Index:            req.GetGatherItem().GetIndex(),
		Items:            make([]*proto.ItemDetail, 0),
		GroupGatherLimit: new(proto.GroupGatherLimit),
		SceneGatherLimit: new(proto.SceneGatherLimit),
		ItemLevel:        0,
	}
	defer g.send(s, msg.PacketId, rsp)
	badReq := func() {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
	}
	if req.GetGatherItem() == nil {
		badReq()
		return
	}
	scenePlayer, sceneId, ok := g.currentPlayerScene(s)
	sceneConf := gdconf.GetSceneInfo(sceneId)
	if !ok || scenePlayer == nil || sceneConf == nil {
		badReq()
		return
	}
	point := sceneConf.GatherPointInfo(req.GetGatherItem().GetIndex())
	if point == nil || point.GetGatherID() <= 0 {
		badReq()
		return
	}
	gatherConf := gdconf.GetGatherConfigure(uint32(point.GetGatherID()))
	if gatherConf == nil || gatherConf.ID <= 0 || uint32(gatherConf.ID) != uint32(point.GetGatherID()) {
		badReq()
		return
	}
	group, matched := findGatherGroup(gatherConf, req.GetGatherItem().GetReward())
	if !matched || group.GetNewGatherType() < 0 || group.GetNewWeaponType() < 0 {
		badReq()
		return
	}
	gatherType := uint32(group.GetNewGatherType())
	if _, _, ok := gdconf.GatherRuleLimits(gdconf.GetSceneGatherRuleConfigure(sceneId), gatherType); !ok {
		badReq()
		return
	}
	weaponType := proto.EWeaponType(group.GetNewWeaponType())
	rewardConf := gdconf.GetGatherRewardConfigure(req.GetGatherItem().GetReward())
	if rewardConf == nil {
		badReq()
		return
	}
	probability, ok := gatherLuckyProbability(gatherType)
	if !ok {
		badReq()
		return
	}
	// is_lucky 只是客户端展示，奖励分支完全由服务端概率决定。
	lucky := rollGatherLucky(probability, rand.IntN)
	rewardGroups, ok := selectGatherRewards(rewardConf, lucky)
	if !ok {
		badReq()
		return
	}
	sceneInfo := playerSceneInfo(s, sceneId)
	if sceneInfo == nil {
		badReq()
		return
	}

	inventory := model.SnapshotItemModel(s.GetItemModel())
	tx, err := s.GetItemModel().BeginAtomic()
	if err != nil {
		badReq()
		return
	}
	defer tx.Rollback()
	if !gatherToolMatches(s, scenePlayer, weaponType) {
		badReq()
		return
	}
	limits := sceneInfo.GetGatherLimits()
	limit, exists := limits[gatherType]
	if !exists || limit == nil || limit.GatherType != gatherType {
		badReq()
		return
	}
	if !gatherWithinLimit(limit, lucky) {
		badReq()
		return
	}
	rewards := make([]itemReward, 0, len(rewardGroups))
	for _, reward := range rewardGroups {
		rewards = append(rewards, itemReward{
			itemId: uint32(reward.GetItemID()),
			count:  int64(reward.GetCount()),
		})
	}
	if !validateItemRewards(s, rewards) {
		badReq()
		return
	}
	items := addItemRewards(s, rewards)
	if items == nil {
		badReq()
		return
	}
	previousCount := limit.GatherNum
	limit.GatherNum++
	rsp.Items = append(rsp.Items, items...)
	rsp.SceneGatherLimit = sceneInfo.SceneGatherLimit()
	rsp.GroupGatherLimit.GatherLimit = limit.GatherLimit()
	if tx.Commit() == nil || !persistCollectReward(s, inventory) {
		limit.GatherNum = previousCount
		badReq()
		return
	}
	g.PackNoticeByItems(s, items)
}

func (g *Game) TreasureBoxOpen(s *model.Player, msg *alg.GameMsg) {
	tempPackMu.Lock()
	defer tempPackMu.Unlock()
	req := msg.Body.(*proto.TreasureBoxOpenReq)
	rsp := &proto.TreasureBoxOpenRsp{
		Status:          proto.StatusCode_StatusCode_OK,
		Items:           make([]*proto.ItemDetail, 0),
		NextRefreshTime: 0,
	}
	defer g.send(s, msg.PacketId, rsp)

	if req.GetLocType() != proto.TreasureBoxLocType_TreasureBoxLocType_DUNGEON {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	dungeon, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok || dungeon == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	if dungeon.Info == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	box := dungeon.Info.GetTreasureBoxes()[req.TreasureBoxIndex]
	stateBefore := proto.TreasureBoxState_TreasureBoxState_Close
	if box != nil {
		stateBefore = box.State
	}
	var opened *model.DungeonTreasureRuntime
	items, status := g.dungeonRuntime().openTreasure(s, dungeon, req.TreasureBoxIndex)
	if status == proto.StatusCode_StatusCode_OK && s.Online {
		if err := tempPackSavePlayer(s); err != nil {
			if box != nil {
				box.State = stateBefore
			}
			if dungeon.Runtime != nil {
				opened = dungeon.Runtime.Treasures[req.TreasureBoxIndex]
				delete(dungeon.Runtime.Treasures, req.TreasureBoxIndex)
				if opened != nil && opened.Drop != nil {
					delete(dungeon.GetTempPack().TempAreaPack, opened.Drop.Index)
				}
			}
			status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			items = nil
		}
	}
	rsp.Status = status
	rsp.Items = items
	if box := dungeon.Info.GetTreasureBoxes()[req.TreasureBoxIndex]; box != nil {
		rsp.NextRefreshTime = box.NextRefreshTime
	}
}

func (g *Game) TreasureBoxPickup(s *model.Player, msg *alg.GameMsg) {
	tempPackMu.Lock()
	defer tempPackMu.Unlock()
	req := msg.Body.(*proto.TreasureBoxPickupReq)
	rsp := &proto.TreasureBoxPickupRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	if req.GetLocType() != proto.TreasureBoxLocType_TreasureBoxLocType_DUNGEON {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	dungeon, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok || dungeon == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	inventory := model.SnapshotItemModel(s.GetItemModel())
	var drop *model.DropItem
	var before []model.EBagItemTag
	var treasure *model.DungeonTreasureRuntime
	if dungeon.Runtime != nil {
		treasure = dungeon.Runtime.Treasures[req.BoxIndex]
	}
	if treasure != nil {
		drop = treasure.Drop
		if drop != nil {
			before = append([]model.EBagItemTag(nil), drop.Items...)
		}
	}
	pickedBefore := treasure != nil && treasure.Picked
	items, status := g.dungeonRuntime().pickupTreasure(s, dungeon, req.BoxIndex, req.PickIndex)
	if status == proto.StatusCode_StatusCode_OK && s.Online {
		if err := tempPackSavePlayer(s); err != nil {
			inventory.Restore()
			if drop != nil {
				drop.Items = before
			}
			if treasure != nil {
				treasure.Picked = pickedBefore
			}
			status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			items = nil
		}
	}
	rsp.Status = status
	rsp.Items = items
	if status == proto.StatusCode_StatusCode_OK && len(items) > 0 {
		g.PackNoticeByItems(s, items)
	}
}

func (g *Game) GetCollectMoonInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCollectMoonInfoReq)
	rsp := &proto.GetCollectMoonInfoRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		SceneId:          req.SceneId,
		CollectedMoonIds: make([]uint32, 0),
		EmotionMoons:     make([]*proto.EmotionMoonInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	sceneInfo := playerSceneInfo(s, req.GetSceneId())
	if gdconf.GetSceneInfo(req.GetSceneId()) == nil || sceneInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	info := sceneInfo.GetCollectionInfo(proto.ECollectionType_ECollectionType_CollectMoonPiece)
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.CollectedMoonIds = info.CollectedMoonIds
}

func (g *Game) CollectMoon(s *model.Player, msg *alg.GameMsg) {
	collectMu.Lock()
	defer collectMu.Unlock()
	req := msg.Body.(*proto.CollectMoonReq)
	rsp := &proto.CollectMoonRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		MoonId:  req.MoonId,
		Rewards: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer, sceneId, ok := g.currentPlayerScene(s)
	if !ok || scenePlayer == nil || gdconf.GetSceneInfo(sceneId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	conf := gdconf.GetCollectionItem(req.GetMoonId())
	sceneInfo := playerSceneInfo(s, sceneId)
	if conf == nil || sceneInfo == nil || conf.NewCollectionType != int32(proto.ECollectionType_ECollectionType_CollectMoonPiece) {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	info := sceneInfo.GetCollectionInfo(proto.ECollectionType(conf.NewCollectionType))
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	inventory := model.SnapshotItemModel(s.GetItemModel())
	tx, err := s.GetItemModel().BeginAtomic()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	defer tx.Rollback()
	// 判断
	if slices.Contains(info.CollectedMoonIds, req.MoonId) {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rewards, ok := collectionRewards(s, conf)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	items := addItemRewards(s, rewards)
	if items == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	previous := append([]uint32(nil), info.CollectedMoonIds...)
	alg.AddSlice(&info.CollectedMoonIds, req.MoonId)
	rsp.Rewards = append(rsp.Rewards, items...)
	if tx.Commit() == nil || !persistCollectReward(s, inventory) {
		info.CollectedMoonIds = previous
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	g.PackNoticeByItems(s, items)
}
