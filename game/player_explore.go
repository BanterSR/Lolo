package game

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type exploreDeps struct {
	loadConfig  func() (*model.ExploreConfig, error)
	itemExists  func(uint32) bool
	hasItem     func(*model.Player, uint32, int64) bool
	consumeItem func(*model.Player, uint32, int64) (*proto.ItemDetail, bool)
	grantItem   func(*model.Player, uint32, int64) *proto.ItemDetail
}

var productionExploreDeps = exploreDeps{
	loadConfig: model.LoadExploreConfig,
	itemExists: configuredExploreItem,
	hasItem: func(s *model.Player, id uint32, count int64) bool {
		item := s.GetItemModel().GetItemBaseInfo(id)
		return item != nil && item.Num >= count
	},
	consumeItem: consumeExploreItem,
	grantItem: func(s *model.Player, id uint32, count int64) *proto.ItemDetail {
		ctx := s.AddAllTypeItem(id, count)
		if ctx == nil {
			return nil
		}
		return ctx.AddItemDetail()
	},
}

func configuredExploreItem(id uint32) (ok bool) {
	// gdconf historically panics when its global configuration is not loaded.
	// Treat that the same as a missing row so requests fail closed.
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	conf := gdconf.GetItemConfigure(id)
	if conf == nil {
		return false
	}
	switch proto.EBagItemTag(conf.NewBagItemTag) {
	case proto.EBagItemTag_EBagItemTag_Gift,
		proto.EBagItemTag_EBagItemTag_Fragment,
		proto.EBagItemTag_EBagItemTag_Collection,
		proto.EBagItemTag_EBagItemTag_Material,
		proto.EBagItemTag_EBagItemTag_Food,
		proto.EBagItemTag_EBagItemTag_SpellCard,
		proto.EBagItemTag_EBagItemTag_Item,
		proto.EBagItemTag_EBagItemTag_Fish,
		proto.EBagItemTag_EBagItemTag_Recipe,
		proto.EBagItemTag_EBagItemTag_Baitbox,
		proto.EBagItemTag_EBagItemTag_Quest,
		proto.EBagItemTag_EBagItemTag_StrengthStone,
		proto.EBagItemTag_EBagItemTag_ExpBook,
		proto.EBagItemTag_EBagItemTag_UnlockAbilityItem,
		proto.EBagItemTag_EBagItemTag_CharacterBadge,
		proto.EBagItemTag_EBagItemTag_DyeStuff,
		proto.EBagItemTag_EBagItemTag_PlayerExp,
		proto.EBagItemTag_EBagItemTag_WorldLevel,
		proto.EBagItemTag_EBagItemTag_Agentia,
		proto.EBagItemTag_EBagItemTag_MoonStone,
		proto.EBagItemTag_EBagItemTag_Umbrella,
		proto.EBagItemTag_EBagItemTag_Vitality,
		proto.EBagItemTag_EBagItemTag_Badge,
		proto.EBagItemTag_EBagItemTag_Furniture,
		proto.EBagItemTag_EBagItemTag_Energy,
		proto.EBagItemTag_EBagItemTag_ShowWeapon,
		proto.EBagItemTag_EBagItemTag_ShowArmor,
		proto.EBagItemTag_EBagItemTag_TeleportKey,
		proto.EBagItemTag_EBagItemTag_WallPaper,
		proto.EBagItemTag_EBagItemTag_MoonCard,
		proto.EBagItemTag_EBagItemTag_PhoneCase,
		proto.EBagItemTag_EBagItemTag_Pendant,
		proto.EBagItemTag_EBagItemTag_AvatarFrame,
		proto.EBagItemTag_EBagItemTag_IntimacyGift,
		proto.EBagItemTag_EBagItemTag_MusicNote,
		proto.EBagItemTag_EBagItemTag_MonthlyCard,
		proto.EBagItemTag_EBagItemTag_BattlePassCard,
		proto.EBagItemTag_EBagItemTag_MonthlyGiftCard,
		proto.EBagItemTag_EBagItemTag_BattlePassGiftCard,
		proto.EBagItemTag_EBagItemTag_SeasonalMiniGamesItem,
		proto.EBagItemTag_EBagItemTag_TreasureMap,
		proto.EBagItemTag_EBagItemTag_TreasureMapFrag,
		proto.EBagItemTag_EBagItemTag_Catcher,
		proto.EBagItemTag_EBagItemTag_PetGrid,
		proto.EBagItemTag_EBagItemTag_PetMmaterials,
		proto.EBagItemTag_EBagItemTag_Card,
		proto.EBagItemTag_EBagItemTag_Currency,
		proto.EBagItemTag_EBagItemTag_Head,
		proto.EBagItemTag_EBagItemTag_Weapon,
		proto.EBagItemTag_EBagItemTag_Fashion,
		proto.EBagItemTag_EBagItemTag_Armor,
		proto.EBagItemTag_EBagItemTag_Poster,
		proto.EBagItemTag_EBagItemTag_Inscription,
		proto.EBagItemTag_EBagItemTag_Pet:
		return true
	default:
		return false
	}
}

func consumeExploreItem(s *model.Player, id uint32, count int64) (*proto.ItemDetail, bool) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, false
	}
	tx.DelBaseItem(id, count)
	if tx.Error != nil {
		tx.Rollback()
		return nil, false
	}
	tx.Commit()
	if tx.PackNotice == nil || len(tx.PackNotice.Items) == 0 {
		return nil, false
	}
	return tx.PackNotice.Items[0], true
}

func exploreDefinition(cfg *model.ExploreConfig, id uint32, activity bool) (model.ExploreDefinition, bool) {
	if activity {
		return cfg.Activity(id)
	}
	return cfg.Area(id)
}

func exploreState(modelState *model.ExploreModel, id uint32, activity bool) *model.PlayerExploreState {
	return modelState.Get(id, activity)
}

func activeExploreCount(modelState *model.ExploreModel) uint32 {
	var count uint32
	for _, states := range []map[uint32]*model.PlayerExploreState{modelState.Areas, modelState.Activities} {
		for _, state := range states {
			if state != nil && state.StartTime != 0 {
				count++
			}
		}
	}
	return count
}

func validExploreTeam(s *model.Player, team [3]uint32) bool {
	if team[0] == 0 {
		return false
	}
	seen := make(map[uint32]struct{}, len(team))
	for _, id := range team {
		if id == 0 {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			return false
		}
		if s.GetCharacterModel().GetCharacterInfo(id) == nil {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

func validateExploreItems(deps exploreDeps, def model.ExploreDefinition) bool {
	if def.NeedItemID != 0 && !deps.itemExists(def.NeedItemID) {
		return false
	}
	if def.QuickItemID != 0 && !deps.itemExists(def.QuickItemID) {
		return false
	}
	for _, id := range def.RewardItemIDs {
		if !deps.itemExists(id) {
			return false
		}
	}
	if def.SpecialItemID != 0 && !deps.itemExists(def.SpecialItemID) {
		return false
	}
	return true
}

func newExploreToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func startExplore(s *model.Player, deps exploreDeps, id uint32, activity bool,
	sceneID, playerLevel uint32, team [3]uint32, now time.Time) (*proto.PlayerExploreInfo, proto.StatusCode) {
	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	def, ok := exploreDefinition(cfg, id, activity)
	if !ok {
		return nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	state, err := s.LoadExploreModel()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	current := exploreState(state, id, activity)
	if current != nil && current.StartTime != 0 {
		return current.PB(def), proto.StatusCode_StatusCode_EXPLORE_IN_PROGRESS
	}
	if activity && (now.Before(def.StartTime) || now.After(def.EndTime)) {
		return nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	if def.MaxTimes != 0 {
		times := uint32(0)
		if current != nil {
			times = current.Times
		}
		if times >= def.MaxTimes {
			return nil, proto.StatusCode_StatusCode_EXPLORE_MAX_TIMES
		}
	}
	if activeExploreCount(state) >= cfg.MaxConcurrent {
		return nil, proto.StatusCode_StatusCode_EXPLORE_IN_PROGRESS
	}
	if sceneID == 0 || (!activity && sceneID != def.NeedSceneID) {
		return nil, proto.StatusCode_StatusCode_SCENE_NOT_UNLOCK
	}
	if playerLevel < def.NeedPlayerLevel {
		return nil, proto.StatusCode_StatusCode_PLAYER_LEVEL_LIMIT
	}
	if !validExploreTeam(s, team) {
		return nil, proto.StatusCode_StatusCode_BAD_REQ
	}
	if !validateExploreItems(deps, def) {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if def.NeedItemID != 0 && !deps.hasItem(s, def.NeedItemID, def.NeedItemCount) {
		return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}
	token, err := newExploreToken()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_BAD_REQ
	}
	current = state.Ensure(id, activity)
	if activity {
		current.Times++
	}
	current.StartTime = now.UnixMilli()
	current.EndTime = current.StartTime + def.NeedTime.Milliseconds()
	current.Token = token
	current.QuickFinish = false
	current.CollectItemIDs = append([]uint32(nil), def.RewardItemIDs...)
	if err := s.SaveExploreModel(state); err != nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	return current.PB(def), proto.StatusCode_StatusCode_OK
}

func quickFinishExplore(s *model.Player, deps exploreDeps, id uint32, activity bool) (*proto.PlayerExploreInfo, *proto.ItemDetail, proto.StatusCode) {
	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	def, ok := exploreDefinition(cfg, id, activity)
	if !ok {
		return nil, nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	state, err := s.LoadExploreModel()
	if err != nil {
		return nil, nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	current := exploreState(state, id, activity)
	if current == nil || current.StartTime == 0 {
		return nil, nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	if current.QuickFinish {
		return current.PB(def), nil, proto.StatusCode_StatusCode_OK
	}
	if def.QuickItemID == 0 || def.QuickItemCount <= 0 || !deps.itemExists(def.QuickItemID) {
		return nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	item, ok := deps.consumeItem(s, def.QuickItemID, def.QuickItemCount)
	if !ok {
		return current.PB(def), nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}
	current.QuickFinish = true
	if err := s.SaveExploreModel(state); err != nil {
		return current.PB(def), item, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	return current.PB(def), item, proto.StatusCode_StatusCode_OK
}

func grantExploreDefinition(s *model.Player, deps exploreDeps, def model.ExploreDefinition) ([]*proto.ItemDetail, bool) {
	ids := append([]uint32(nil), def.RewardItemIDs...)
	if def.SpecialItemID != 0 {
		ids = append([]uint32{def.SpecialItemID}, ids...)
	}
	return grantExploreItems(s, deps, ids)
}

func grantExploreItems(s *model.Player, deps exploreDeps, ids []uint32) ([]*proto.ItemDetail, bool) {
	items := make([]*proto.ItemDetail, 0, len(ids))
	for _, id := range ids {
		item := deps.grantItem(s, id, 1)
		if item == nil {
			return nil, false
		}
		alg.AddList(&items, item)
	}
	return items, true
}

func claimExploreReward(s *model.Player, deps exploreDeps, id uint32, activity bool,
	now time.Time) (*proto.PlayerExploreInfo, []*proto.ItemDetail, proto.StatusCode) {
	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	def, ok := exploreDefinition(cfg, id, activity)
	if !ok {
		return nil, nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	state, err := s.LoadExploreModel()
	if err != nil {
		return nil, nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	current := exploreState(state, id, activity)
	if current == nil || current.StartTime == 0 {
		return nil, nil, proto.StatusCode_StatusCode_EXPLORE_NOT_FINISH
	}
	if !current.QuickFinish && now.UnixMilli() < current.EndTime {
		return current.PB(def), nil, proto.StatusCode_StatusCode_EXPLORE_NOT_FINISH
	}
	if !validateExploreItems(deps, def) {
		return nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	items, ok := grantExploreDefinition(s, deps, def)
	if !ok {
		return current.PB(def), nil, proto.StatusCode_StatusCode_Item_Get_Fail
	}
	current.StartTime, current.EndTime, current.Token, current.QuickFinish = 0, 0, "", false
	current.CompletedTimes++
	if !activity {
		current.Times++
	}
	if err := s.SaveExploreModel(state); err != nil {
		return current.PB(def), items, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	return current.PB(def), items, proto.StatusCode_StatusCode_OK
}

func claimExploreCollectReward(s *model.Player, deps exploreDeps, id uint32, activity bool) (*proto.PlayerExploreInfo, []*proto.ItemDetail, proto.StatusCode) {
	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	def, ok := exploreDefinition(cfg, id, activity)
	if !ok {
		return nil, nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	state, err := s.LoadExploreModel()
	if err != nil {
		return nil, nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	current := exploreState(state, id, activity)
	if current == nil || current.CompletedTimes == 0 {
		return nil, nil, proto.StatusCode_StatusCode_EXPLORE_COLLECT_NOT_FINISH
	}
	if current.CollectReward {
		return current.PB(def), nil, proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	if !validateExploreItems(deps, def) {
		return nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	items, ok := grantExploreItems(s, deps, def.RewardItemIDs)
	if !ok {
		return current.PB(def), nil, proto.StatusCode_StatusCode_Item_Get_Fail
	}
	current.CollectReward = true
	if err := s.SaveExploreModel(state); err != nil {
		return current.PB(def), items, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	return current.PB(def), items, proto.StatusCode_StatusCode_OK
}

func cancelExplore(s *model.Player, id uint32, activity bool) (*proto.PlayerExploreInfo, proto.StatusCode) {
	state, err := s.LoadExploreModel()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	current := exploreState(state, id, activity)
	if current == nil || current.StartTime == 0 {
		return nil, proto.StatusCode_StatusCode_EXPLORE_NOT_EXISTS
	}
	current.StartTime, current.EndTime, current.Token, current.QuickFinish = 0, 0, "", false
	if err := s.SaveExploreModel(state); err != nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	return &proto.PlayerExploreInfo{
		ExploreId: id, Times: current.Times, CollectItemId: append([]uint32(nil), current.CollectItemIDs...),
		CollectReward: current.CollectReward,
	}, proto.StatusCode_StatusCode_OK
}

func (g *Game) explorePlayerContext(s *model.Player) (uint32, uint32, [3]uint32, proto.StatusCode) {
	basic, ok := db.GetGameBasic(s.UserId)
	if !ok {
		return 0, 0, [3]uint32{}, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.CurScene == nil {
		return 0, 0, [3]uint32{}, proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
	}
	team := s.GetTeamModel().GetTeamInfo()
	if team == nil {
		return 0, 0, [3]uint32{}, proto.StatusCode_StatusCode_BAD_REQ
	}
	return basic.Level, scenePlayer.CurScene.GetSceneId(), [3]uint32{team.Char1, team.Char2, team.Char3}, proto.StatusCode_StatusCode_OK
}

func (g *Game) ExploreInit(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.ExploreInitRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		Explore: make([]*proto.PlayerExploreInfo, 0), ActivityExplore: make([]*proto.PlayerExploreInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	cfg, err := productionExploreDeps.loadConfig()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	for _, id := range cfg.AreaIDs() {
		def, _ := cfg.Area(id)
		if !validateExploreItems(productionExploreDeps, def) {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
	}
	for _, id := range cfg.ActivityIDs() {
		def, _ := cfg.Activity(id)
		if !validateExploreItems(productionExploreDeps, def) {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
	}
	state, err := s.LoadExploreModel()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		return
	}
	for _, id := range cfg.AreaIDs() {
		def, _ := cfg.Area(id)
		current := exploreState(state, id, false)
		if current == nil {
			current = &model.PlayerExploreState{ExploreID: id}
		}
		alg.AddList(&rsp.Explore, current.PB(def))
	}
	for _, id := range cfg.ActivityIDs() {
		def, _ := cfg.Activity(id)
		current := exploreState(state, id, true)
		if current == nil {
			current = &model.PlayerExploreState{ExploreID: id}
		}
		alg.AddList(&rsp.ActivityExplore, current.PB(def))
	}
}

func (g *Game) Explore(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ExploreReq)
	rsp := &proto.ExploreRsp{Status: proto.StatusCode_StatusCode_OK, IsActivity: req.IsActivity}
	defer g.send(s, msg.PacketId, rsp)
	level, sceneID, team, status := g.explorePlayerContext(s)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info, status := startExplore(s, productionExploreDeps, req.ExploreId, req.IsActivity, sceneID, level, team, time.Now())
	rsp.Status = status
	rsp.Explore = info
	if status == proto.StatusCode_StatusCode_OK {
		if err := s.SavePlayerNow(); err != nil {
			rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			log.Game.Errorf("UserId:%v 探索开始落库失败:%s", s.UserId, err)
		}
	}
}

func (g *Game) ExploreQuickFinish(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ExploreQuickFinishReq)
	rsp := &proto.ExploreQuickFinishRsp{Status: proto.StatusCode_StatusCode_OK, IsActivity: req.IsActivity}
	defer g.send(s, msg.PacketId, rsp)
	info, item, status := quickFinishExplore(s, productionExploreDeps, req.ExploreId, req.IsActivity)
	rsp.Status, rsp.Explore = status, info
	if item != nil {
		g.PackNoticeByItems(s, []*proto.ItemDetail{item})
	}
	if status == proto.StatusCode_StatusCode_OK {
		if err := s.SavePlayerNow(); err != nil {
			rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			log.Game.Errorf("UserId:%v 探索加速落库失败:%s", s.UserId, err)
		}
	}
}

func (g *Game) ExploreReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ExploreRewardReq)
	rsp := &proto.ExploreRewardRsp{
		Status: proto.StatusCode_StatusCode_OK, IsActivity: req.IsActivity,
		ExploreId: req.ExploreId, RewardItem: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	info, items, status := claimExploreReward(s, productionExploreDeps, req.ExploreId, req.IsActivity, time.Now())
	rsp.Status, rsp.Explore, rsp.RewardItem = status, info, items
	if len(items) > 0 {
		g.PackNoticeByItems(s, items)
	}
	if status == proto.StatusCode_StatusCode_OK {
		if err := s.SavePlayerNow(); err != nil {
			rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			log.Game.Errorf("UserId:%v 探索领奖落库失败:%s", s.UserId, err)
		}
	}
}

func (g *Game) ExploreCancel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ExploreCancelReq)
	rsp := &proto.ExploreCancelRsp{Status: proto.StatusCode_StatusCode_OK, ExploreId: req.ExploreId, IsActivity: req.IsActivity}
	defer g.send(s, msg.PacketId, rsp)
	info, status := cancelExplore(s, req.ExploreId, req.IsActivity)
	rsp.Status, rsp.Explore = status, info
	if status == proto.StatusCode_StatusCode_OK {
		if err := s.SavePlayerNow(); err != nil {
			rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			log.Game.Errorf("UserId:%v 取消探索落库失败:%s", s.UserId, err)
		}
	}
}

func (g *Game) ExploreCollectReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ExploreCollectRewardReq)
	rsp := &proto.ExploreCollectRewardRsp{
		Status: proto.StatusCode_StatusCode_OK, IsActivity: req.IsActivity,
		ExploreId: req.ExploreId, RewardItem: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	info, items, status := claimExploreCollectReward(s, productionExploreDeps, req.ExploreId, req.IsActivity)
	rsp.Status, rsp.Explore, rsp.RewardItem = status, info, items
	if len(items) > 0 {
		g.PackNoticeByItems(s, items)
	}
	if status == proto.StatusCode_StatusCode_OK {
		if err := s.SavePlayerNow(); err != nil {
			rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
			log.Game.Errorf("UserId:%v 探索收集奖励落库失败:%s", s.UserId, err)
		}
	}
}
