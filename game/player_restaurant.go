package game

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/config"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

// 配置加载集中在游戏层，gdconf 暂未提供餐厅/宠物访问器。缺少文件或字段时拒绝操作。
var (
	restaurantConfigProvider = loadRestaurantGameConfig
	activityConfigProvider   = loadActivityConfig
	petConfigProvider        = loadPetConfig
	expressionConfigProvider = loadExpressionConfig
	sceneConfigProvider      = loadSceneConfig

	coreItemConfigure = gdconf.GetItemConfigure
	coreGrantItem     = func(s *model.Player, itemID uint32, count int64) *model.AddItemCtx {
		return s.AddAllTypeItem(itemID, count)
	}
	coreSavePlayer = func(s *model.Player) error { return s.SavePlayerNow() }
	coreRollIntN   = rand.IntN
	coreNow        = time.Now
)

var (
	restaurantConfigOnce sync.Once
	restaurantConfig     *excel.AllRestaurantGameDatas
	restaurantConfigErr  error
	activityConfigOnce   sync.Once
	activityConfig       *excel.AllActivityDatas
	activityConfigErr    error
	petConfigOnce        sync.Once
	petConfig            *excel.AllPetDatas
	petConfigErr         error
	expressionConfigOnce sync.Once
	expressionConfig     *excel.AllExpressionDatas
	expressionConfigErr  error
	sceneConfigOnce      sync.Once
	sceneConfig          *excel.AllSceneDatas
	sceneConfigErr       error
)

func loadExcelConfig[T any](name string) (*T, error) {
	var data T
	path := filepath.Join(config.GetResources().GetResourcePath(), "Excel", name)
	bin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := sonic.Unmarshal(bin, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func loadRestaurantGameConfig() (*excel.AllRestaurantGameDatas, error) {
	restaurantConfigOnce.Do(func() {
		restaurantConfig, restaurantConfigErr = loadExcelConfig[excel.AllRestaurantGameDatas]("RestaurantGame.json")
	})
	return restaurantConfig, restaurantConfigErr
}

func loadActivityConfig() (*excel.AllActivityDatas, error) {
	activityConfigOnce.Do(func() {
		activityConfig, activityConfigErr = loadExcelConfig[excel.AllActivityDatas]("Activity.json")
	})
	return activityConfig, activityConfigErr
}

func loadPetConfig() (*excel.AllPetDatas, error) {
	petConfigOnce.Do(func() {
		petConfig, petConfigErr = loadExcelConfig[excel.AllPetDatas]("Pet.json")
	})
	return petConfig, petConfigErr
}

func loadExpressionConfig() (*excel.AllExpressionDatas, error) {
	expressionConfigOnce.Do(func() {
		expressionConfig, expressionConfigErr = loadExcelConfig[excel.AllExpressionDatas]("Expression.json")
	})
	return expressionConfig, expressionConfigErr
}

func loadSceneConfig() (*excel.AllSceneDatas, error) {
	sceneConfigOnce.Do(func() {
		sceneConfig, sceneConfigErr = loadExcelConfig[excel.AllSceneDatas]("Scene.json")
	})
	return sceneConfig, sceneConfigErr
}

type restaurantRules struct {
	activityID       uint32
	dungeons         []*excel.RestaurantDungeonConfigure
	dungeonByID      map[uint32]*excel.RestaurantDungeonConfigure
	bags             map[uint32]*excel.RestaurantDeskBagGroupInfos
	foods            map[uint32]*excel.RestaurantGameFoodGroupInfos
	foodsByLevelType map[restaurantFoodKey][]uint32
	pools            map[uint32]*excel.RestaurantRandomPoolConfigure
	quests           map[uint32]*excel.RestaurantGameQuestGroupInfo
	coinItemID       uint32
}

type restaurantFoodKey struct {
	level int32
	type_ int32
}

func buildRestaurantRules(activityID uint32) (*restaurantRules, proto.StatusCode) {
	conf, err := restaurantConfigProvider()
	if err != nil || conf == nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	rules := &restaurantRules{
		activityID:       activityID,
		dungeonByID:      make(map[uint32]*excel.RestaurantDungeonConfigure),
		bags:             make(map[uint32]*excel.RestaurantDeskBagGroupInfos),
		foods:            make(map[uint32]*excel.RestaurantGameFoodGroupInfos),
		foodsByLevelType: make(map[restaurantFoodKey][]uint32),
		pools:            make(map[uint32]*excel.RestaurantRandomPoolConfigure),
		quests:           make(map[uint32]*excel.RestaurantGameQuestGroupInfo),
	}
	for _, dungeon := range conf.GetRestaurantDungeon().GetDatas() {
		if dungeon == nil || dungeon.ID <= 0 || uint32(dungeon.ActivityID) != activityID || len(dungeon.RestaurantDungeonGroupInfo) == 0 {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		if _, exists := rules.dungeonByID[uint32(dungeon.ID)]; exists {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		rules.dungeons = append(rules.dungeons, dungeon)
		rules.dungeonByID[uint32(dungeon.ID)] = dungeon
	}
	if len(rules.dungeons) == 0 {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	sort.Slice(rules.dungeons, func(i, j int) bool { return rules.dungeons[i].ID < rules.dungeons[j].ID })
	for _, dungeon := range rules.dungeons {
		if dungeon.UnlockDungeonID != 0 {
			if _, ok := rules.dungeonByID[uint32(dungeon.UnlockDungeonID)]; !ok {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
		}
		for _, group := range dungeon.RestaurantDungeonGroupInfo {
			if group == nil || group.FoodLevel <= 0 || group.FoodType <= 0 || group.OrderQuantity <= 0 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
		}
	}
	for _, bagConf := range conf.GetRestaurantDeskBag().GetDatas() {
		if bagConf == nil || uint32(bagConf.ID) != activityID || len(bagConf.RestaurantDeskBagGroupInfo) == 0 {
			continue
		}
		for _, bag := range bagConf.RestaurantDeskBagGroupInfo {
			if bag == nil || bag.BagID <= 0 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if _, exists := rules.bags[uint32(bag.BagID)]; exists {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if bag.UnlockDungonID != 0 {
				if _, ok := rules.dungeonByID[uint32(bag.UnlockDungonID)]; !ok {
					return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
				}
			}
			rules.bags[uint32(bag.BagID)] = bag
		}
	}
	if len(rules.bags) == 0 {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	for _, foodConf := range conf.GetRestaurantGameFood().GetDatas() {
		if foodConf == nil || uint32(foodConf.ID) != activityID {
			continue
		}
		for _, food := range foodConf.RestaurantGameFoodGroupInfo {
			if food == nil || food.FoodID <= 0 || food.FoodLevel <= 0 || food.FoodType <= 0 ||
				food.RecyclingPriceMin < 0 || food.RecyclingPriceMax < food.RecyclingPriceMin {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if _, exists := rules.foods[uint32(food.FoodID)]; exists {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			rules.foods[uint32(food.FoodID)] = food
			key := restaurantFoodKey{level: food.FoodLevel, type_: food.FoodType}
			rules.foodsByLevelType[key] = append(rules.foodsByLevelType[key], uint32(food.FoodID))
		}
	}
	for id := range rules.foods {
		food := rules.foods[id]
		if food.SyntheticTarget != 0 {
			if _, ok := rules.foods[uint32(food.SyntheticTarget)]; !ok {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
		}
	}
	for key := range rules.foodsByLevelType {
		sort.Slice(rules.foodsByLevelType[key], func(i, j int) bool { return rules.foodsByLevelType[key][i] < rules.foodsByLevelType[key][j] })
	}
	for _, pool := range conf.GetRestaurantRandomPool().GetDatas() {
		if pool == nil || uint32(pool.ActivityID) != activityID {
			continue
		}
		if pool.ID <= 0 || pool.RateMin < 0 || pool.RateMax < pool.RateMin || len(pool.RestaurantRandomPoolGroupInfo) == 0 {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		if _, exists := rules.pools[uint32(pool.ID)]; exists {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		for _, group := range pool.RestaurantRandomPoolGroupInfo {
			if group == nil {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if _, ok := rules.foods[uint32(group.FoodId)]; !ok {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
		}
		rules.pools[uint32(pool.ID)] = pool
	}
	for _, questConf := range conf.GetRestaurantGameQuest().GetDatas() {
		if questConf == nil || uint32(questConf.ID) != activityID {
			continue
		}
		for _, quest := range questConf.RestaurantGameQuestGroupInfo {
			if quest == nil || quest.GameQuestID <= 0 || quest.GameCoin < 0 || quest.GameIntegral < 0 ||
				quest.GameCoinID <= 0 || quest.Param <= 0 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if quest.NewRestaurantGameQuestType < 0 || quest.NewRestaurantGameQuestType > 1 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if quest.GameIntegral > 0 && quest.GameIntegralItem <= 0 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			if rules.coinItemID != 0 && rules.coinItemID != uint32(quest.GameCoinID) {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			rules.coinItemID = uint32(quest.GameCoinID)
			if _, exists := rules.quests[uint32(quest.GameQuestID)]; exists {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			rules.quests[uint32(quest.GameQuestID)] = quest
		}
	}
	if len(rules.quests) == 0 || rules.coinItemID == 0 {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	return rules, proto.StatusCode_StatusCode_OK
}

func restaurantActivityStatus(activityID uint32, now time.Time) proto.StatusCode {
	conf, err := activityConfigProvider()
	if err != nil || conf == nil {
		return proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	for _, activity := range conf.GetActivity().GetDatas() {
		if activity == nil || uint32(activity.ID) != activityID {
			continue
		}
		opening, err := time.ParseInLocation("2006-01-02 15:04:05", activity.OpeningTime, time.Local)
		if err != nil || activity.ClosingTime == "" {
			return proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		closing, err := time.ParseInLocation("2006-01-02 15:04:05", activity.ClosingTime, time.Local)
		if err != nil || closing.Before(opening) {
			return proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		if now.Before(opening) || !now.Before(closing) {
			return proto.StatusCode_StatusCode_Activity_Close
		}
		return proto.StatusCode_StatusCode_OK
	}
	return proto.StatusCode_StatusCode_CFG_NOT_FOUND
}

func restaurantDay(now time.Time) int32 {
	_, offset := now.Zone()
	return int32((now.Unix() + int64(offset)) / 86400)
}

func restaurantTeamValid(s *model.Player) bool {
	team := s.GetTeamModel().GetTeamInfo()
	if team == nil {
		return false
	}
	count := 0
	for _, id := range []uint32{team.Char1, team.Char2, team.Char3} {
		if id == 0 {
			continue
		}
		if s.GetCharacterModel().GetCharacterInfo(id) == nil {
			return false
		}
		count++
	}
	return count > 0
}

func restaurantGridUnlocked(rules *restaurantRules, activity *model.RestaurantActivityState, index uint32) bool {
	bag := rules.bags[index]
	if bag == nil {
		return false
	}
	if bag.UnlockDungonID == 0 {
		return true
	}
	dungeon := activity.Dungeons[uint32(bag.UnlockDungonID)]
	return dungeon != nil && dungeon.Cleared
}

func newRestaurantDungeonState(conf *excel.RestaurantDungeonConfigure, rules *restaurantRules) (*model.RestaurantDungeonState, error) {
	state := &model.RestaurantDungeonState{
		Grids:      make(map[uint32]uint32, len(rules.bags)),
		Customers:  make([]model.RestaurantCustomerState, 0, len(conf.RestaurantDungeonGroupInfo)),
		Operations: make(map[string]model.RestaurantOperationRecord),
	}
	for index := range rules.bags {
		state.Grids[index] = 0
	}
	for _, group := range conf.RestaurantDungeonGroupInfo {
		foods := rules.foodsByLevelType[restaurantFoodKey{level: group.FoodLevel, type_: group.FoodType}]
		if len(foods) == 0 {
			return nil, errors.New("restaurant customer has no configured food")
		}
		state.Customers = append(state.Customers, model.RestaurantCustomerState{Food: foods[coreRollIntN(len(foods))]})
	}
	return state, nil
}

func ensureRestaurantContext(s *model.Player, activityID uint32, now time.Time) (*restaurantRules, *model.RestaurantActivityState, *excel.RestaurantDungeonConfigure, *model.RestaurantDungeonState, proto.StatusCode, bool) {
	if status := restaurantActivityStatus(activityID, now); status != proto.StatusCode_StatusCode_OK {
		return nil, nil, nil, nil, status, false
	}
	if !restaurantTeamValid(s) {
		return nil, nil, nil, nil, proto.StatusCode_StatusCode_DungeonTeamError, false
	}
	rules, status := buildRestaurantRules(activityID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, nil, nil, nil, status, false
	}
	day := restaurantDay(now)
	activity := s.GetRestaurantModel().Activities[activityID]
	dirty := false
	if activity == nil {
		activity = &model.RestaurantActivityState{
			Day:      day,
			Dungeons: make(map[uint32]*model.RestaurantDungeonState),
			Quests:   make(map[uint32]*model.RestaurantQuestState),
		}
		s.GetRestaurantModel().Activities[activityID] = activity
		dirty = true
	}
	if activity.Day != day {
		for id, conf := range rules.dungeonByID {
			if conf.IsRefresh {
				delete(activity.Dungeons, id)
			}
		}
		activity.Day = day
		dirty = true
	}
	if activity.Dungeons == nil {
		activity.Dungeons = make(map[uint32]*model.RestaurantDungeonState)
		dirty = true
	}
	if activity.Quests == nil {
		activity.Quests = make(map[uint32]*model.RestaurantQuestState)
		dirty = true
	}
	for id, quest := range rules.quests {
		state := activity.Quests[id]
		if state == nil {
			state = &model.RestaurantQuestState{Status: proto.QuestStatus_QuestStatus_Locked, Day: day}
			activity.Quests[id] = state
			dirty = true
		}
		if quest.IsRefresh && state.Day != day {
			state.Status = proto.QuestStatus_QuestStatus_Locked
			state.Progress = 0
			state.Rewarded = false
			state.Day = day
			dirty = true
		}
		if quest.NewRestaurantGameQuestType == int32(proto.ERestaurantGameQuest_ERestaurantGameQuest_RestaurantGameDailyLogin) &&
			state.Status == proto.QuestStatus_QuestStatus_Locked {
			state.Status = proto.QuestStatus_QuestStatus_Completed
			state.Progress = 1
			state.Day = day
			dirty = true
		}
		if quest.NewRestaurantGameQuestType == int32(proto.ERestaurantGameQuest_ERestaurantGameQuest_RestaurantGameClearLevel) &&
			state.Status == proto.QuestStatus_QuestStatus_Locked {
			if dungeon := activity.Dungeons[uint32(quest.Param)]; dungeon != nil && dungeon.Cleared {
				state.Status = proto.QuestStatus_QuestStatus_Completed
				state.Progress = 1
				dirty = true
			}
		}
	}
	selected := rules.dungeonByID[activity.CurrentDungeonID]
	if selected == nil || activity.Dungeons[activity.CurrentDungeonID] == nil || activity.Dungeons[activity.CurrentDungeonID].Cleared {
		selected = nil
		for _, conf := range rules.dungeons {
			if conf.UnlockDungeonID != 0 {
				prerequisite := activity.Dungeons[uint32(conf.UnlockDungeonID)]
				if prerequisite == nil || !prerequisite.Cleared {
					continue
				}
			}
			dungeon := activity.Dungeons[uint32(conf.ID)]
			if dungeon == nil || !dungeon.Cleared {
				selected = conf
				break
			}
		}
		if selected == nil {
			selected = rules.dungeons[len(rules.dungeons)-1]
		}
		activity.CurrentDungeonID = uint32(selected.ID)
		dirty = true
	}
	dungeon := activity.Dungeons[activity.CurrentDungeonID]
	if dungeon == nil {
		var err error
		dungeon, err = newRestaurantDungeonState(selected, rules)
		if err != nil {
			return nil, nil, nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND, false
		}
		activity.Dungeons[activity.CurrentDungeonID] = dungeon
		dirty = true
	}
	if len(dungeon.Customers) != len(selected.RestaurantDungeonGroupInfo) || dungeon.Grids == nil {
		return nil, nil, nil, nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND, false
	}
	return rules, activity, selected, dungeon, proto.StatusCode_StatusCode_OK, dirty
}

func restaurantGridList(rules *restaurantRules, activity *model.RestaurantActivityState, dungeon *model.RestaurantDungeonState) []*proto.RestaurantGameGridInfo {
	indices := make([]uint32, 0, len(rules.bags))
	for index := range rules.bags {
		indices = append(indices, index)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })
	list := make([]*proto.RestaurantGameGridInfo, 0, len(indices))
	for _, index := range indices {
		list = append(list, &proto.RestaurantGameGridInfo{
			Index:    index,
			Food:     dungeon.Grids[index],
			IsUnLock: restaurantGridUnlocked(rules, activity, index),
		})
	}
	return list
}

func restaurantDungeonInfo(rules *restaurantRules, activity *model.RestaurantActivityState, conf *excel.RestaurantDungeonConfigure, dungeon *model.RestaurantDungeonState) *proto.RestaurantGameDungeonInfo {
	info := &proto.RestaurantGameDungeonInfo{
		DungeonId:    uint32(conf.ID),
		CustomerList: make([]*proto.RestaurantGameCustomerInfo, 0, len(dungeon.Customers)),
		QuestionInfo: make([]*proto.RestaurantGameQuestInfo, 0, len(rules.quests)),
	}
	for index, customer := range dungeon.Customers {
		info.CustomerList = append(info.CustomerList, &proto.RestaurantGameCustomerInfo{
			CustomerIndex: uint32(index),
			Food:          customer.Food,
			Status:        customer.Done,
		})
	}
	ids := make([]uint32, 0, len(rules.quests))
	for id := range rules.quests {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		quest := activity.Quests[id]
		if quest == nil {
			continue
		}
		info.QuestionInfo = append(info.QuestionInfo, &proto.RestaurantGameQuestInfo{QuestId: id, Status: quest.Status})
	}
	return info
}

func saveCoreMutation(s *model.Player, rspStatus *proto.StatusCode) {
	if err := coreSavePlayer(s); err != nil {
		*rspStatus = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		log.Game.Errorf("UserId:%v 核心玩法落库失败:%s", s.UserId, err.Error())
	}
}

func coreCosts(items map[uint32]int64) []model.ItemBaseInfo {
	costs := make([]model.ItemBaseInfo, 0, len(items))
	for id, count := range items {
		costs = append(costs, model.ItemBaseInfo{ItemId: id, Num: count})
	}
	sort.Slice(costs, func(i, j int) bool { return costs[i].ItemId < costs[j].ItemId })
	return costs
}

func spendCoreItems(s *model.Player, costs map[uint32]int64) (*model.ItemTransaction, proto.StatusCode) {
	if len(costs) == 0 {
		return nil, proto.StatusCode_StatusCode_OK
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}
	for _, cost := range coreCosts(costs) {
		if cost.Num <= 0 {
			tx.Rollback()
			return nil, proto.StatusCode_StatusCode_BAD_REQ
		}
		tx.DelBaseItem(cost.ItemId, cost.Num)
		if tx.Error != nil {
			tx.Rollback()
			return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
		}
	}
	tx.Commit()
	return tx, proto.StatusCode_StatusCode_OK
}

func isCoreStackableTag(tag proto.EBagItemTag) bool {
	switch tag {
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
		proto.EBagItemTag_EBagItemTag_Vitality,
		proto.EBagItemTag_EBagItemTag_Badge,
		proto.EBagItemTag_EBagItemTag_Energy,
		proto.EBagItemTag_EBagItemTag_TeleportKey,
		proto.EBagItemTag_EBagItemTag_WallPaper,
		proto.EBagItemTag_EBagItemTag_PhoneCase,
		proto.EBagItemTag_EBagItemTag_Pendant,
		proto.EBagItemTag_EBagItemTag_AvatarFrame,
		proto.EBagItemTag_EBagItemTag_IntimacyGift,
		proto.EBagItemTag_EBagItemTag_MusicNote,
		proto.EBagItemTag_EBagItemTag_Catcher,
		proto.EBagItemTag_EBagItemTag_PetGrid,
		proto.EBagItemTag_EBagItemTag_PetMmaterials,
		proto.EBagItemTag_EBagItemTag_Currency:
		return true
	default:
		return false
	}
}

func grantCoreRewards(s *model.Player, rewards map[uint32]int64) ([]*proto.ItemDetail, proto.StatusCode) {
	items := make([]*proto.ItemDetail, 0, len(rewards))
	for _, reward := range coreCosts(rewards) {
		if reward.Num <= 0 {
			continue
		}
		conf := coreItemConfigure(reward.ItemId)
		if conf == nil || !isCoreStackableTag(proto.EBagItemTag(conf.NewBagItemTag)) {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
	}
	for _, reward := range coreCosts(rewards) {
		if reward.Num <= 0 {
			continue
		}
		ctx := coreGrantItem(s, reward.ItemId, reward.Num)
		if ctx == nil || ctx.EBagItemTag == nil {
			return nil, proto.StatusCode_StatusCode_Item_Get_Fail
		}
		if detail := ctx.AddItemDetail(); detail != nil {
			items = append(items, detail)
		}
	}
	return items, proto.StatusCode_StatusCode_OK
}

func restaurantOperationKey(activityID, dungeonID uint32, operateType proto.RestaurantGameOperateType, poolID uint32, grids []uint32) string {
	canonical := append([]uint32(nil), grids...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i] < canonical[j] })
	h := sha256.New()
	fmt.Fprintf(h, "%d:%d:%d:%d:", activityID, dungeonID, operateType, poolID)
	for _, grid := range canonical {
		fmt.Fprintf(h, "%d,", grid)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func restaurantOperationMatches(record model.RestaurantOperationRecord, day int32, dungeon *model.RestaurantDungeonState, grids []uint32) bool {
	if record.Day != day || len(record.Grids) != len(grids) {
		return false
	}
	for _, index := range grids {
		food, ok := record.Grids[index]
		if !ok || dungeon.Grids[index] != food {
			return false
		}
	}
	return true
}

func validateRestaurantGrids(rules *restaurantRules, activity *model.RestaurantActivityState, dungeon *model.RestaurantDungeonState, grids []uint32, empty bool) ([]uint32, proto.StatusCode) {
	if len(grids) == 0 {
		return nil, proto.StatusCode_StatusCode_BAD_REQ
	}
	seen := make(map[uint32]struct{}, len(grids))
	result := append([]uint32(nil), grids...)
	for _, index := range result {
		if _, exists := seen[index]; exists {
			return nil, proto.StatusCode_StatusCode_BAD_REQ
		}
		seen[index] = struct{}{}
		if !restaurantGridUnlocked(rules, activity, index) {
			return nil, proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_GRID_NOT_EXIST
		}
		food := dungeon.Grids[index]
		if empty && food != 0 {
			return nil, proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_GRID_FULL
		}
		if !empty && food == 0 {
			return nil, proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_GRID_EMPTY
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, proto.StatusCode_StatusCode_OK
}

func (g *Game) RestaurantGameInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.RestaurantGameInfoReq)
	rsp := &proto.RestaurantGameInfoRsp{Status: proto.StatusCode_StatusCode_OK, ActivityId: req.ActivityId}
	defer g.send(s, msg.PacketId, rsp)
	rules, activity, conf, dungeon, status, dirty := ensureRestaurantContext(s, req.ActivityId, coreNow())
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	rsp.GridList = restaurantGridList(rules, activity, dungeon)
	rsp.DungeonInfo = restaurantDungeonInfo(rules, activity, conf, dungeon)
	if dirty {
		saveCoreMutation(s, &rsp.Status)
	}
}

func (g *Game) RestaurantGameOperate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.RestaurantGameOperateReq)
	rsp := &proto.RestaurantGameOperateRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		ActivityId:  req.ActivityId,
		Grid:        make([]*proto.RestaurantGameGridInfo, 0),
		ItemDetails: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	rules, activity, conf, dungeon, status, _ := ensureRestaurantContext(s, req.ActivityId, coreNow())
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	if dungeon.Cleared {
		rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_CUSTOMER_FINISH
		return
	}
	day := restaurantDay(coreNow())
	key := restaurantOperationKey(req.ActivityId, uint32(conf.ID), req.Type, req.BuyPoolId, req.Grid)
	if record, exists := dungeon.Operations[key]; exists && restaurantOperationMatches(record, day, dungeon, req.Grid) {
		rsp.Grid = restaurantGridList(rules, activity, dungeon)
		return
	}
	switch req.Type {
	case proto.RestaurantGameOperateType_RestaurantGameOperateType_Buy:
		pool := rules.pools[req.BuyPoolId]
		if pool == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
		grids, status := validateRestaurantGrids(rules, activity, dungeon, req.Grid, true)
		if status != proto.StatusCode_StatusCode_OK {
			rsp.Status = status
			return
		}
		foods := make([]uint32, 0, len(pool.RestaurantRandomPoolGroupInfo))
		for _, group := range pool.RestaurantRandomPoolGroupInfo {
			foods = append(foods, uint32(group.FoodId))
		}
		costs := make(map[uint32]int64)
		selectedFoods := make(map[uint32]uint32, len(grids))
		for _, index := range grids {
			selectedFoods[index] = foods[coreRollIntN(len(foods))]
			price := int64(pool.RateMin)
			if pool.RateMax > pool.RateMin {
				price += int64(coreRollIntN(int(pool.RateMax-pool.RateMin) + 1))
			}
			costs[rules.coinItemID] += price
		}
		tx, status := spendCoreItems(s, costs)
		if status != proto.StatusCode_StatusCode_OK {
			rsp.Status = status
			return
		}
		for index, food := range selectedFoods {
			dungeon.Grids[index] = food
		}
		if tx != nil {
			rsp.ItemDetails = append(rsp.ItemDetails, tx.PackNotice.Items...)
			g.send(s, 0, tx.PackNotice)
		}
	case proto.RestaurantGameOperateType_RestaurantGameOperateType_Mix:
		grids, status := validateRestaurantGrids(rules, activity, dungeon, req.Grid, false)
		if status != proto.StatusCode_StatusCode_OK {
			rsp.Status = status
			return
		}
		if len(grids) != 2 {
			rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_MIX_ERR
			return
		}
		foodID := dungeon.Grids[grids[0]]
		if dungeon.Grids[grids[1]] != foodID {
			rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_MIX_ERR
			return
		}
		food := rules.foods[foodID]
		if food == nil || food.SyntheticTarget == 0 {
			rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_FOOD_MAX_LEVEL
			return
		}
		target := uint32(food.SyntheticTarget)
		if rules.foods[target] == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
		dungeon.Grids[grids[0]] = target
		dungeon.Grids[grids[1]] = 0
	case proto.RestaurantGameOperateType_RestaurantGameOperateType_Recycle:
		grids, status := validateRestaurantGrids(rules, activity, dungeon, req.Grid, false)
		if status != proto.StatusCode_StatusCode_OK {
			rsp.Status = status
			return
		}
		rewards := make(map[uint32]int64)
		for _, index := range grids {
			food := rules.foods[dungeon.Grids[index]]
			if food == nil {
				rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
				return
			}
			price := int64(food.RecyclingPriceMin)
			if food.RecyclingPriceMax > food.RecyclingPriceMin {
				price += int64(coreRollIntN(int(food.RecyclingPriceMax-food.RecyclingPriceMin) + 1))
			}
			rewards[rules.coinItemID] += price
		}
		items, status := grantCoreRewards(s, rewards)
		if status != proto.StatusCode_StatusCode_OK {
			rsp.Status = status
			return
		}
		for _, index := range grids {
			dungeon.Grids[index] = 0
		}
		rsp.ItemDetails = append(rsp.ItemDetails, items...)
	default:
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	record := model.RestaurantOperationRecord{Day: day, Grids: make(map[uint32]uint32, len(req.Grid))}
	for _, index := range req.Grid {
		record.Grids[index] = dungeon.Grids[index]
	}
	dungeon.Operations[key] = record
	rsp.Grid = restaurantGridList(rules, activity, dungeon)
	saveCoreMutation(s, &rsp.Status)
}

func (g *Game) RestaurantGameEat(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.RestaurantGameEatReq)
	rsp := &proto.RestaurantGameEatRsp{
		Status:        proto.StatusCode_StatusCode_OK,
		CustomerIndex: req.CustomerIndex,
		ActivityId:    req.ActivityId,
		Grid:          make([]*proto.RestaurantGameGridInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	rules, activity, conf, dungeon, status, _ := ensureRestaurantContext(s, req.ActivityId, coreNow())
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	if dungeon.Cleared || int(req.CustomerIndex) >= len(dungeon.Customers) {
		rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_CUSTOMER_FINISH
		return
	}
	customer := &dungeon.Customers[req.CustomerIndex]
	if customer.Done {
		rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_CUSTOMER_FINISH
		return
	}
	if !restaurantGridUnlocked(rules, activity, req.Grid) {
		rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_GRID_NOT_EXIST
		return
	}
	foodID := dungeon.Grids[req.Grid]
	if foodID == 0 {
		rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_GRID_EMPTY
		return
	}
	if foodID != customer.Food || rules.foods[foodID] == nil {
		rsp.Status = proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_FOOD_ERR
		return
	}
	dungeon.Grids[req.Grid] = 0
	customer.Served++
	group := conf.RestaurantDungeonGroupInfo[req.CustomerIndex]
	if customer.Served >= uint32(group.OrderQuantity) {
		customer.Done = true
	}
	allDone := true
	for _, customer := range dungeon.Customers {
		if !customer.Done {
			allDone = false
			break
		}
	}
	if allDone && !dungeon.Cleared {
		dungeon.Cleared = true
		for id, quest := range rules.quests {
			if quest.NewRestaurantGameQuestType != int32(proto.ERestaurantGameQuest_ERestaurantGameQuest_RestaurantGameClearLevel) || quest.Param != int32(conf.ID) {
				continue
			}
			state := activity.Quests[id]
			if state != nil && !state.Rewarded {
				state.Status = proto.QuestStatus_QuestStatus_Completed
				state.Progress = 1
			}
		}
	}
	rsp.Grid = restaurantGridList(rules, activity, dungeon)
	rsp.DungeonInfo = restaurantDungeonInfo(rules, activity, conf, dungeon)
	saveCoreMutation(s, &rsp.Status)
}

func (g *Game) RestaurantGameQuestReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.RestaurantGameQuestRewardReq)
	rsp := &proto.RestaurantGameQuestRewardRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		ActivityId:  req.ActivityId,
		ItemDetails: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	rules, activity, _, _, status, _ := ensureRestaurantContext(s, req.ActivityId, coreNow())
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	questConf := rules.quests[req.QuestId]
	quest := activity.Quests[req.QuestId]
	if questConf == nil || quest == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	if quest.Rewarded || quest.Status == proto.QuestStatus_QuestStatus_Finish {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	if quest.Status != proto.QuestStatus_QuestStatus_Completed {
		rsp.Status = proto.StatusCode_StatusCode_Quest_Not_Finish
		return
	}
	rewards := make(map[uint32]int64)
	if questConf.GameCoin > 0 {
		rewards[uint32(questConf.GameCoinID)] += int64(questConf.GameCoin)
	}
	if questConf.GameIntegral > 0 {
		rewards[uint32(questConf.GameIntegralItem)] += int64(questConf.GameIntegral)
	}
	items, status := grantCoreRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	quest.Status = proto.QuestStatus_QuestStatus_Finish
	quest.Rewarded = true
	rsp.Quest = &proto.RestaurantGameQuestInfo{QuestId: req.QuestId, Status: quest.Status}
	rsp.ItemDetails = append(rsp.ItemDetails, items...)
	saveCoreMutation(s, &rsp.Status)
}
