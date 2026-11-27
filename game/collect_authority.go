package game

import (
	"math"
	"math/rand/v2"
	"reflect"
	"strconv"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

const gatherLuckyProbabilityConfig = "GatherLuckyFinishBaseProbability"

// currentPlayerScene 返回玩家当前实际所在的场景。接口中的 typed nil 也按缺失处理。
func (g *Game) currentPlayerScene(s *model.Player) (*ScenePlayer, uint32, bool) {
	if g == nil || s == nil {
		return nil, 0, false
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.Player != s || nilSceneInterface(scenePlayer.CurScene) {
		return nil, 0, false
	}
	return scenePlayer, scenePlayer.CurScene.GetSceneId(), true
}

func nilSceneInterface(scene PlayerSceneInterface) bool {
	if scene == nil {
		return true
	}
	value := reflect.ValueOf(scene)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// playerSceneInfo 保留 SceneModel 原有的惰性初始化行为，但显式存入 nil 的场景数据视为损坏。
func playerSceneInfo(s *model.Player, sceneId uint32) *model.SceneInfo {
	if s == nil {
		return nil
	}
	sceneModel := s.GetSceneModel()
	if info, ok := sceneModel.GetSceneMap()[sceneId]; ok {
		if info == nil || info.SceneId != sceneId {
			return nil
		}
		return info
	}
	return sceneModel.GetSceneInfo(sceneId)
}

func findGatherGroup(conf *excel.GatherConfigure, reward int32) (*excel.GatherGroupInfo, bool) {
	if conf == nil {
		return nil, false
	}
	var matched *excel.GatherGroupInfo
	for _, group := range conf.GetGatherGroupInfo() {
		if group == nil || group.GetReward() != reward {
			continue
		}
		if matched != nil {
			// 同一奖励出现多个分组时，工具类型和计数类型可能不同，不能猜测。
			return nil, false
		}
		matched = group
	}
	return matched, matched != nil
}

func gatherLuckyProbability(gatherType uint32) (int, bool) {
	conf := gdconf.GetGlobalConfigConfigure(gatherLuckyProbabilityConfig)
	if conf == nil || int(gatherType) >= len(conf.GetValues()) {
		return 0, false
	}
	probability, err := strconv.Atoi(conf.GetValues()[gatherType])
	if err != nil || probability < 0 || probability > 100 {
		return 0, false
	}
	return probability, true
}

func rollGatherLucky(probability int, roll func(int) int) bool {
	if probability < 0 || probability > 100 {
		return false
	}
	value := roll(100)
	return value >= 0 && value < 100 && value < probability
}

func selectGatherRewards(conf *excel.GatherRewardConfigure, lucky bool) ([]*excel.GatherRewardGroupInfo, bool) {
	if conf == nil {
		return nil, false
	}
	rewards := make([]*excel.GatherRewardGroupInfo, 0, len(conf.GetGatherRewardGroupInfo()))
	for _, reward := range conf.GetGatherRewardGroupInfo() {
		if reward != nil && reward.GetLucky() == lucky {
			rewards = append(rewards, reward)
		}
	}
	return rewards, len(rewards) > 0
}

// gatherToolMatches 校验当前队伍角色确实持有配置要求的采集工具实例。
func gatherToolMatches(s *model.Player, scenePlayer *ScenePlayer, weaponType proto.EWeaponType) bool {
	if _, ok := proto.EWeaponType_name[int32(weaponType)]; !ok {
		return false
	}
	team := scenePlayer.CurScene.GetTeam()
	if team == nil {
		team = s.GetTeamModel().GetTeamInfo()
	}
	if team == nil {
		return false
	}
	characterId := team.Char1
	if characterId == 0 {
		characterId = team.Char2
	}
	if characterId == 0 {
		characterId = team.Char3
	}
	character := s.GetCharacterModel().GetCharacterInfo(characterId)
	if character == nil {
		return false
	}
	if weaponType == proto.EWeaponType_EWeaponType_All {
		// 赶海配置使用 All，表示不限制手持工具，但仍要求当前角色属于玩家。
		return true
	}
	if character.GatherWeapon == 0 {
		return false
	}
	for instanceId, item := range s.GetItemModel().GetItemWeaponMap() {
		if item == nil || item.WeaponId != character.GatherWeapon {
			continue
		}
		weapon, _, ok := validWeaponInstance(s, instanceId)
		if !ok || weapon != item {
			continue
		}
		conf := gdconf.GetWeaponAllInfo(item.WeaponId)
		if conf != nil && conf.WeaponInfo != nil &&
			proto.EWeaponType(conf.WeaponInfo.GetNewWeaponType()) == weaponType {
			return true
		}
	}
	return false
}

func gatherWithinLimit(limit *model.GatherLimit, lucky bool) bool {
	if limit == nil || limit.GatherNum == math.MaxUint32 {
		return false
	}
	// 上限一律以配置为准，0 表示禁止；不再把旧存档的 0 当作无限。
	if limit.GatherNum >= limit.GatherLimitNum {
		return false
	}
	if lucky && limit.GatherNum >= limit.LuckyGatherLimitNum {
		return false
	}
	return true
}

type itemReward struct {
	itemId uint32
	count  int64
}

// isBaseRewardTag 与 ItemModel.AddAllTypeItem 的基础物品分支保持一致。
func isBaseRewardTag(tag proto.EBagItemTag) bool {
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
		proto.EBagItemTag_EBagItemTag_Currency:
		return true
	default:
		return false
	}
}

func validateItemRewards(s *model.Player, rewards []itemReward) bool {
	if s == nil {
		return false
	}
	totals := make(map[uint32]int64, len(rewards))
	for _, reward := range rewards {
		if reward.itemId == 0 || reward.count <= 0 {
			return false
		}
		conf := gdconf.GetItemConfigure(reward.itemId)
		if conf == nil || !isBaseRewardTag(proto.EBagItemTag(conf.GetNewBagItemTag())) {
			return false
		}
		total := totals[reward.itemId]
		if total > math.MaxInt64-reward.count {
			return false
		}
		totals[reward.itemId] = total + reward.count
	}
	for itemId, total := range totals {
		item := s.GetItemModel().GetItemBaseInfo(itemId)
		if item != nil && (item.Num < 0 || item.Num > math.MaxInt64-total) {
			return false
		}
	}
	return true
}

func addItemRewards(s *model.Player, rewards []itemReward) []*proto.ItemDetail {
	items := make([]*proto.ItemDetail, 0, len(rewards))
	for _, reward := range rewards {
		item := s.AddAllTypeItem(reward.itemId, reward.count)
		if item == nil || item.EBagItemTag == nil {
			return nil
		}
		detail := item.AddItemDetail()
		if detail == nil {
			return nil
		}
		items = append(items, detail)
	}
	return items
}

func collectionRewardCount(minCount, maxCount int32, roll func(int64) int64) (int64, bool) {
	if minCount < 0 || maxCount < minCount {
		return 0, false
	}
	span := int64(maxCount) - int64(minCount) + 1
	if span <= 0 {
		return 0, false
	}
	offset := roll(span)
	if offset < 0 || offset >= span {
		return 0, false
	}
	return int64(minCount) + offset, true
}

func collectionRewards(s *model.Player, conf *excel.CollectionItemConfigure) ([]itemReward, bool) {
	rewardList := gdconf.GetCollectionReward(conf)
	rewards := make([]itemReward, 0, len(rewardList))
	for _, reward := range rewardList {
		if reward == nil {
			return nil, false
		}
		count, ok := collectionRewardCount(reward.GetItemMinCount(), reward.GetItemMaxCount(), rand.Int64N)
		if !ok {
			return nil, false
		}
		rewards = append(rewards, itemReward{itemId: uint32(reward.GetItemID()), count: count})
	}
	return rewards, validateItemRewards(s, rewards)
}
