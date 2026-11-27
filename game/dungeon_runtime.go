package game

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/proto"
)

type dungeonMonster struct {
	UnitID       uint32
	RewardPoolID uint32
}

type dungeonConfig struct {
	ID              uint32
	NeedPlayerLevel uint32
	TeamSize        uint32
	DailyLimit      uint32
	CanReplay       bool
	PreDungeonID    uint32
	CheckQuestID    uint32
	FightTime       int64
	OpenWeekdays    []int32
	RewardID        uint32
	StarRewardID    uint32
	StarTimer1      int64
	StarTimer2      int64
	StarTimer3      int64
	Monsters        map[uint32]dungeonMonster
	MonstersByIndex bool
	Treasures       map[uint32]uint32
}

type dungeonReward struct {
	ItemID   uint32
	MinCount int64
	MaxCount int64
	ItemType proto.EBagItemTag
	MinLevel uint32
	MaxLevel uint32
}

type dungeonService struct {
	loadConfig         func(uint32) (*dungeonConfig, error)
	loadReward         func(uint32) ([]dungeonReward, error)
	loadTreasureReward func(uint32) ([]dungeonReward, error)
	playerLevel        func(*model.Player) (uint32, bool)
	addReward          func(*model.Player, dungeonReward) *proto.ItemDetail
	validateRewards    func(*model.Player, []dungeonReward) bool
	save               func(*model.Player) error
	now                func() time.Time
}

var dungeonTokenCounter atomic.Uint64

var authoritativeDungeons = &dungeonService{
	loadConfig:         loadDungeonConfig,
	loadReward:         loadDungeonReward,
	loadTreasureReward: loadDungeonTreasureReward,
	playerLevel:        dungeonPlayerLevel,
	addReward:          addDungeonReward,
	validateRewards:    validateDungeonRewardItems,
	save:               func(s *model.Player) error { return s.SavePlayerNow() },
	now:                time.Now,
}

func (g *Game) dungeonRuntime() *dungeonService {
	if g != nil && g.dungeons != nil {
		return g.dungeons
	}
	return authoritativeDungeons
}

func loadDungeonConfig(dungeonID uint32) (cfg *dungeonConfig, err error) {
	// gdconf 的旧 getter 在配置未完整初始化时会解引用 nil。这里把这种配置错误
	// 转换成拒绝进入，而不是让请求 panic 后落入默认奖励逻辑。
	defer func() {
		if recovered := recover(); recovered != nil {
			cfg = nil
			err = fmt.Errorf("副本配置初始化不完整: %v", recovered)
		}
	}()
	row := gdconf.GetDungeonConfigure(dungeonID)
	if row == nil || row.ID <= 0 || (uint32(row.ID) != dungeonID && uint32(row.DungeonID) != dungeonID) {
		return nil, errors.New("副本配置不存在或 ID 不匹配")
	}
	scene := gdconf.GetDungeonSceneInfo(dungeonID)
	if scene == nil || scene.Info == nil {
		return nil, errors.New("副本场景配置不存在")
	}
	if len(scene.Info.GetBorn()) == 0 {
		return nil, errors.New("副本出生点配置不存在")
	}
	for _, born := range scene.Info.GetBorn() {
		if born == nil || born.Position == nil || born.Rotation == nil {
			return nil, errors.New("副本出生点配置无效")
		}
	}
	if row.UseCharacterMaxNumber <= 0 {
		return nil, errors.New("副本队伍人数配置无效")
	}
	if row.UseCharacterPoolID != 0 || row.UnUseCharacterPoolID != 0 {
		return nil, errors.New("副本引用了未加载的角色池配置")
	}
	if row.DengeonQuestID != 0 && gdconf.GetDungeonQuestConfigure(uint32(row.DengeonQuestID)) == nil {
		return nil, errors.New("副本任务配置不存在")
	}
	if row.CheckQuestID != 0 && gdconf.GetQuestInfos()[uint32(row.CheckQuestID)] == nil {
		return nil, errors.New("副本前置任务配置不存在")
	}
	for _, weekday := range row.OpenDay {
		if weekday < 0 || weekday > 7 {
			return nil, errors.New("副本开放星期配置无效")
		}
	}
	cfg = &dungeonConfig{
		ID:              dungeonID,
		NeedPlayerLevel: uint32(max(row.NeedPlayerLevel, 0)),
		TeamSize:        uint32(row.UseCharacterMaxNumber),
		DailyLimit:      uint32(max(row.RespawnTimes, 0)),
		CanReplay:       row.CanRePlay,
		// DungeonID is a logical configuration key, not a prerequisite dungeon.
		// Enter prerequisites are expressed by CheckQuestID below.
		PreDungeonID: 0,
		CheckQuestID: uint32(max(row.CheckQuestID, 0)),
		FightTime:    int64(max(row.FightTime, 0)),
		OpenWeekdays: append([]int32(nil), row.OpenDay...),
		RewardID:     uint32(max(row.RewardID, 0)),
		StarRewardID: uint32(max(row.StarRewardID, 0)),
		StarTimer1:   int64(max(row.StarTimer1, 0)),
		StarTimer2:   int64(max(row.StarTimer2, 0)),
		StarTimer3:   int64(max(row.StarTimer3, 0)),
		Monsters:     make(map[uint32]dungeonMonster),
		Treasures:    make(map[uint32]uint32),
	}
	for _, treasure := range scene.Info.GetCollectionTreasureInfos() {
		if treasure == nil || treasure.Index < 0 || treasure.ID <= 0 {
			return nil, errors.New("副本宝箱配置无效")
		}
		index := uint32(treasure.Index)
		if _, exists := cfg.Treasures[index]; exists {
			return nil, errors.New("副本宝箱索引重复")
		}
		if gdconf.GetTreasureBoxConfigure(uint32(treasure.ID)) == nil {
			return nil, errors.New("副本宝箱配置不存在")
		}
		cfg.Treasures[index] = uint32(treasure.ID)
	}
	sceneMonsters := scene.Info.GetMonsterInfos()
	cfg.MonstersByIndex = len(sceneMonsters) > 0
	if len(sceneMonsters) > 0 && len(row.MonsterIDs) > 0 {
		return nil, errors.New("副本场景怪物与 Dungeon.MonsterIDs 同时存在，无法确定权威遭遇")
	}
	if len(sceneMonsters) > 0 {
		for _, monster := range sceneMonsters {
			if monster == nil || monster.Index < 0 || monster.UnitID <= 0 {
				return nil, errors.New("副本遭遇配置无效")
			}
			index := uint32(monster.Index)
			if _, exists := cfg.Monsters[index]; exists {
				return nil, errors.New("副本遭遇索引重复")
			}
			character := gdconf.GetMonsterCharacterConfigure(uint32(monster.UnitID))
			if character == nil {
				return nil, errors.New("副本遭遇怪物配置不存在")
			}
			cfg.Monsters[index] = dungeonMonster{
				UnitID:       uint32(monster.UnitID),
				RewardPoolID: uint32(max(character.RewardPoolID, 0)),
			}
		}
	} else {
		// 部分配置只有 Dungeon.MonsterIDs。没有场景索引时按怪物 ID 结算，
		// 每个配置怪物每轮仍只能结算一次，不能据此生成额外实例。
		for _, monsterID := range row.MonsterIDs {
			if monsterID <= 0 || gdconf.GetMonsterCharacterConfigure(uint32(monsterID)) == nil {
				return nil, errors.New("副本怪物配置不存在")
			}
			if _, exists := cfg.Monsters[uint32(monsterID)]; exists {
				return nil, errors.New("副本怪物配置重复")
			}
			character := gdconf.GetMonsterCharacterConfigure(uint32(monsterID))
			cfg.Monsters[uint32(monsterID)] = dungeonMonster{
				UnitID:       uint32(monsterID),
				RewardPoolID: uint32(max(character.RewardPoolID, 0)),
			}
		}
	}
	return cfg, nil
}

func loadDungeonReward(rewardID uint32) (rewards []dungeonReward, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			rewards = nil
			err = fmt.Errorf("奖励配置初始化不完整: %v", recovered)
		}
	}()
	if rewardID == 0 {
		return nil, nil
	}
	pool := gdconf.GetRewardPool(rewardID)
	if pool == nil || pool.ID <= 0 || uint32(pool.ID) != rewardID {
		return nil, errors.New("奖励池配置不存在或 ID 不匹配")
	}
	for _, group := range pool.GetRewardPoolGroup() {
		if group == nil || group.RewardPoolGroupID <= 0 {
			return nil, errors.New("奖励池分组配置无效")
		}
		itemPool := gdconf.GetRewardItemPool(uint32(group.RewardPoolGroupID))
		if itemPool == nil || itemPool.ID <= 0 || uint32(itemPool.ID) != uint32(group.RewardPoolGroupID) {
			return nil, errors.New("奖励物品池配置不存在或 ID 不匹配")
		}
		for _, item := range itemPool.GetRewardItemPoolGroup() {
			if item == nil || item.ItemID <= 0 || item.ItemMinCount <= 0 || item.ItemMaxCount < item.ItemMinCount ||
				item.MinLevelLimit < 0 || item.MaxLevelLimit < 0 ||
				(item.MaxLevelLimit > 0 && item.MaxLevelLimit < item.MinLevelLimit) {
				return nil, errors.New("奖励物品配置无效")
			}
			itemConfig := gdconf.GetItemConfigure(uint32(item.ItemID))
			if itemConfig == nil {
				return nil, errors.New("奖励物品配置不存在")
			}
			rewards = append(rewards, dungeonReward{
				ItemID:   uint32(item.ItemID),
				MinCount: int64(item.ItemMinCount),
				MaxCount: int64(item.ItemMaxCount),
				ItemType: proto.EBagItemTag(itemConfig.NewBagItemTag),
				MinLevel: uint32(item.MinLevelLimit),
				MaxLevel: uint32(item.MaxLevelLimit),
			})
		}
	}
	return rewards, nil
}

func loadDungeonTreasureReward(boxID uint32) (rewards []dungeonReward, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			rewards = nil
			err = fmt.Errorf("宝箱配置初始化不完整: %v", recovered)
		}
	}()
	box := gdconf.GetTreasureBoxConfigure(boxID)
	if box == nil || box.ID <= 0 || uint32(box.ID) != boxID || box.RelateID <= 0 {
		return nil, errors.New("宝箱配置或奖励引用不存在")
	}
	return loadDungeonReward(uint32(box.RelateID))
}

func dungeonPlayerLevel(s *model.Player) (uint32, bool) {
	basic, ok := db.GetGameBasic(s.UserId)
	if !ok || basic == nil {
		return 0, false
	}
	return basic.Level, true
}

func addDungeonReward(s *model.Player, reward dungeonReward) *proto.ItemDetail {
	ctx := s.AddAllTypeItem(reward.ItemID, reward.Count())
	if ctx == nil {
		return nil
	}
	return ctx.AddItemDetail()
}

func isDungeonBaseRewardTag(tag proto.EBagItemTag) bool {
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

func validateDungeonRewardItems(s *model.Player, rewards []dungeonReward) bool {
	totals := make(map[uint32]int64, len(rewards))
	for _, reward := range rewards {
		if reward.ItemID == 0 || reward.MinCount <= 0 || reward.MaxCount < reward.MinCount {
			return false
		}
		conf := gdconf.GetItemConfigure(reward.ItemID)
		if conf == nil || !isDungeonBaseRewardTag(proto.EBagItemTag(conf.GetNewBagItemTag())) {
			return false
		}
		total := totals[reward.ItemID]
		if total > math.MaxInt64-reward.MaxCount {
			return false
		}
		totals[reward.ItemID] = total + reward.MaxCount
	}
	if s == nil {
		return true
	}
	for itemID, total := range totals {
		item := s.GetItemModel().GetItemBaseInfo(itemID)
		if item != nil && (item.Num < 0 || item.Num > math.MaxInt64-total) {
			return false
		}
	}
	return true
}

func (d *dungeonService) rewardItemsValid(s *model.Player, rewards []dungeonReward) bool {
	return d.validateRewards == nil || d.validateRewards(s, rewards)
}

func (r dungeonReward) Count() int64 {
	if r.MaxCount <= r.MinCount {
		return r.MinCount
	}
	return r.MinCount + rand.Int64N(r.MaxCount-r.MinCount+1)
}

func dungeonRewardsAtLevel(rewards []dungeonReward, level uint32) []dungeonReward {
	filtered := make([]dungeonReward, 0, len(rewards))
	for _, reward := range rewards {
		if reward.MinLevel > 0 && level < reward.MinLevel {
			continue
		}
		if reward.MaxLevel > 0 && level > reward.MaxLevel {
			continue
		}
		filtered = append(filtered, reward)
	}
	return filtered
}

func nextDungeonToken() uint64 {
	return dungeonTokenCounter.Add(1)
}

func dungeonServerDay(t time.Time) int64 {
	const serverOffset = int64(8 * 60 * 60)
	return (t.Unix() + serverOffset) / (24 * 60 * 60)
}

func (d *dungeonService) config(id uint32) (*dungeonConfig, proto.StatusCode) {
	if d.loadConfig == nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	cfg, err := d.loadConfig(id)
	if err != nil || cfg == nil || cfg.ID != id {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	return cfg, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) rewards(id uint32) ([]dungeonReward, proto.StatusCode) {
	if d.loadReward == nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	rewards, err := d.loadReward(id)
	if err != nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	return rewards, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) validateConfigRewards(cfg *dungeonConfig) proto.StatusCode {
	clearRewards, status := d.rewards(cfg.RewardID)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	starRewards, status := d.rewards(cfg.StarRewardID)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	for _, rewards := range [][]dungeonReward{clearRewards, starRewards} {
		if !d.rewardItemsValid(nil, rewards) {
			return proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
	}
	for _, monster := range cfg.Monsters {
		rewards, status := d.rewards(monster.RewardPoolID)
		if status != proto.StatusCode_StatusCode_OK {
			return status
		}
		if !d.rewardItemsValid(nil, rewards) {
			return proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
	}
	// Treasure rows include collection/riddle/game boxes whose rewards are not
	// RewardPool references. Validate each box authoritatively when it is opened
	// instead of rejecting the whole dungeon or fabricating loot for those rows.
	return proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) enterDungeon(s *model.Player, active *model.SceneDungeon, dungeonID uint32, team model.TeamInfo) (*model.DungeonRuntime, proto.StatusCode) {
	if s == nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	if active != nil {
		return nil, proto.StatusCode_StatusCode_EXPLORE_IN_PROGRESS
	}
	cfg, status := d.config(dungeonID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if status := d.validateConfigRewards(cfg); status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	level, ok := d.playerLevel(s)
	if !ok {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	if level < cfg.NeedPlayerLevel {
		return nil, proto.StatusCode_StatusCode_PLAYER_LEVEL_LIMIT
	}
	if len(cfg.OpenWeekdays) > 0 {
		weekday := int32(d.now().Weekday())
		open := false
		for _, allowed := range cfg.OpenWeekdays {
			// Configuration may number Sunday as either 0 (time.Weekday) or 7.
			if weekday == allowed || (weekday == 0 && allowed == 7) {
				open = true
				break
			}
		}
		if !open {
			return nil, proto.StatusCode_StatusCode_Activity_Close
		}
	}
	characters := []uint32{team.Char1, team.Char2, team.Char3}
	seen := make(map[uint32]bool, len(characters))
	count := 0
	for _, characterID := range characters {
		if characterID == 0 {
			continue
		}
		if seen[characterID] {
			return nil, proto.StatusCode_StatusCode_DungeonTeamError
		}
		character := s.GetCharacterModel().GetCharacterInfo(characterID)
		if character == nil || character.CharacterId != characterID {
			return nil, proto.StatusCode_StatusCode_CharNotExist
		}
		seen[characterID] = true
		count++
	}
	if count == 0 {
		return nil, proto.StatusCode_StatusCode_DungeonTeamError
	}
	if count > int(cfg.TeamSize) {
		return nil, proto.StatusCode_StatusCode_DungeonTeamFull
	}
	info := s.GetDungeonModel().GetDungeonInfo(dungeonID)
	if cfg.PreDungeonID != 0 {
		previous := s.GetDungeonModel().GetDungeonMap()[cfg.PreDungeonID]
		if previous == nil || previous.FinishTimes == 0 {
			return nil, proto.StatusCode_StatusCode_DungeonUnlock
		}
	}
	if cfg.CheckQuestID != 0 {
		quest := s.GetQuestModel().GetQuestMap()[cfg.CheckQuestID]
		if quest == nil || (quest.Status != proto.QuestStatus_QuestStatus_Completed && quest.Status != proto.QuestStatus_QuestStatus_Finish) {
			return nil, proto.StatusCode_StatusCode_DungeonUnlock
		}
	}
	if !cfg.CanReplay && (info.FinishTimes > 0 || info.FirstClearRewardClaimed) {
		return nil, proto.StatusCode_StatusCode_DungeonCantReward
	}
	now := d.now()
	day := dungeonServerDay(now)
	if cfg.DailyLimit > 0 && info.LastEnterDay == day && info.DailyEnterCount >= cfg.DailyLimit {
		return nil, proto.StatusCode_StatusCode_EXPLORE_NUM_LIMIT
	}
	if info.EnterTimes == math.MaxUint32 || (info.LastEnterDay == day && info.DailyEnterCount == math.MaxUint32) {
		return nil, proto.StatusCode_StatusCode_SYS_ERR
	}
	monsters := make(map[uint32]model.DungeonMonsterRuntime, len(cfg.Monsters))
	for index, monster := range cfg.Monsters {
		monsters[index] = model.DungeonMonsterRuntime{UnitID: monster.UnitID, RewardPoolID: monster.RewardPoolID}
	}
	runtime := &model.DungeonRuntime{
		Token:           nextDungeonToken(),
		DungeonID:       dungeonID,
		State:           model.DungeonRuntimeEntered,
		MonstersByIndex: cfg.MonstersByIndex,
		Monsters:        monsters,
		DeadMonsters:    make(map[uint32]bool),
		Encounters:      make(map[uint32]proto.BattleState),
		Drops:           make(map[uint32]*model.DropItem),
		Treasures:       make(map[uint32]*model.DungeonTreasureRuntime),
	}
	// The counter is process-local. Skip tokens present in persisted progress so
	// a server restart cannot make a new round look like an already claimed one.
	for runtime.Token == 0 || runtime.Token == info.LastRunToken || runtime.Token == info.RewardClaimedToken {
		runtime.Token = nextDungeonToken()
	}
	info.TeamInfo = team
	info.EnterTimes++
	if info.LastEnterDay != day {
		info.DailyEnterCount = 0
		info.LastEnterDay = day
	}
	info.DailyEnterCount++
	info.LastEnterTime = now.Unix()
	info.LastRunToken = runtime.Token
	info.ResetTreasureBoxes()
	return runtime, proto.StatusCode_StatusCode_OK
}

func validDungeonRuntime(s *model.Player, sd *model.SceneDungeon) (*model.DungeonRuntime, proto.StatusCode) {
	if sd == nil || sd.Info == nil || sd.Runtime == nil || sd.ScenePlayerInfo == nil || sd.Player == nil {
		return nil, proto.StatusCode_StatusCode_PlayerNotInDungeon
	}
	if s == nil || sd.Player.UserId != s.UserId || sd.Info.DungeonID != sd.Runtime.DungeonID ||
		sd.Runtime.Token == 0 || sd.Info.LastRunToken != sd.Runtime.Token ||
		sd.Runtime.State > model.DungeonRuntimeClaimed {
		return nil, proto.StatusCode_StatusCode_PlayerNotInDungeon
	}
	return sd.Runtime, proto.StatusCode_StatusCode_OK
}

func (g *Game) getAuthorityDungeonScene(s *model.Player) (*model.SceneDungeon, proto.StatusCode) {
	if g == nil || s == nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
	}
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		return nil, proto.StatusCode_StatusCode_PlayerNotInDungeon
	}
	if _, status := validDungeonRuntime(s, sd); status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	return sd, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) start(s *model.Player, sd *model.SceneDungeon) proto.StatusCode {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	if runtime.State != model.DungeonRuntimeEntered {
		return proto.StatusCode_StatusCode_BAD_REQ
	}
	cfg, status := d.config(runtime.DungeonID)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	if status := d.validateConfigRewards(cfg); status != proto.StatusCode_StatusCode_OK {
		return status
	}
	now := d.now()
	runtime.StartedAt = now.Unix()
	runtime.State = model.DungeonRuntimeRunning
	sd.Info.LastEnterTime = runtime.StartedAt
	return proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) end(s *model.Player, sd *model.SceneDungeon) proto.StatusCode {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	if runtime.State != model.DungeonRuntimeRunning {
		return proto.StatusCode_StatusCode_ChallengeNotStart
	}
	cfg, status := d.config(runtime.DungeonID)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	now := d.now()
	elapsed := now.Unix() - runtime.StartedAt
	if elapsed < 0 {
		return proto.StatusCode_StatusCode_BAD_REQ
	}
	if cfg.FightTime > 0 && elapsed > cfg.FightTime {
		return proto.StatusCode_StatusCode_EXPLORE_NOT_FINISH
	}
	runtime.EndedAt = now.Unix()
	runtime.Stars = dungeonStars(cfg, elapsed)
	runtime.State = model.DungeonRuntimeFinished
	return proto.StatusCode_StatusCode_OK
}

func dungeonStars(cfg *dungeonConfig, elapsed int64) uint32 {
	stars := uint32(0)
	if cfg.StarTimer1 > 0 && elapsed <= int64(cfg.StarTimer1) {
		stars = 1
	}
	if cfg.StarTimer2 > 0 && elapsed <= int64(cfg.StarTimer2) {
		stars = 2
	}
	if cfg.StarTimer3 > 0 && elapsed <= int64(cfg.StarTimer3) {
		stars = 3
	}
	return stars
}

func (d *dungeonService) allMonstersDead(runtime *model.DungeonRuntime) bool {
	for index := range runtime.Monsters {
		if !runtime.DeadMonsters[index] {
			return false
		}
	}
	return true
}

func (d *dungeonService) monsterDead(s *model.Player, sd *model.SceneDungeon, monsterID, monsterIndex uint32) (*model.DropItem, proto.StatusCode) {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if runtime.State != model.DungeonRuntimeRunning {
		return nil, proto.StatusCode_StatusCode_ChallengeNotStart
	}
	monsterKey := monsterIndex
	monster, ok := runtime.Monsters[monsterKey]
	if runtime.MonstersByIndex {
		if !ok || monster.UnitID != monsterID {
			return nil, proto.StatusCode_StatusCode_DungeonCheat
		}
	} else {
		monsterKey = monsterID
		monster, ok = runtime.Monsters[monsterKey]
		if !ok {
			return nil, proto.StatusCode_StatusCode_DungeonCheat
		}
	}
	if runtime.DeadMonsters[monsterKey] {
		return nil, proto.StatusCode_StatusCode_DungeonCheat
	}
	rewards, status := d.rewards(monster.RewardPoolID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if !d.rewardItemsValid(nil, rewards) {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if !d.rewardItemsValid(s, rewards) {
		return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}
	runtime.DeadMonsters[monsterKey] = true
	if runtime.MonstersByIndex {
		runtime.Encounters[monsterKey] = proto.BattleState_Finish
	}
	if len(rewards) == 0 {
		return nil, proto.StatusCode_StatusCode_OK
	}
	items := make([]model.EBagItemTag, 0, len(rewards))
	for _, reward := range rewards {
		count := reward.Count()
		items = append(items, &model.ItemBaseInfo{
			ItemId:   reward.ItemID,
			Num:      count,
			ItemType: reward.ItemType,
			PackType: proto.ItemDetail_PackType_TempStorageArea,
		})
	}
	drop := sd.GetTempPack().NewDrop(items)
	if drop == nil {
		delete(runtime.DeadMonsters, monsterKey)
		if runtime.MonstersByIndex {
			delete(runtime.Encounters, monsterKey)
		}
		return nil, proto.StatusCode_StatusCode_SYS_ERR
	}
	runtime.Drops[drop.Index] = drop
	return drop, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) encounterState(s *model.Player, sd *model.SceneDungeon, encounterID uint32) (proto.BattleState, proto.StatusCode) {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return proto.BattleState_Start, status
	}
	if runtime.State != model.DungeonRuntimeEntered && runtime.State != model.DungeonRuntimeRunning &&
		runtime.State != model.DungeonRuntimeFinished && runtime.State != model.DungeonRuntimeClaimed {
		return proto.BattleState_Start, proto.StatusCode_StatusCode_ChallengeNotStart
	}
	if !runtime.MonstersByIndex {
		return proto.BattleState_Start, proto.StatusCode_StatusCode_DungeonCheat
	}
	if _, ok := runtime.Monsters[encounterID]; !ok {
		return proto.BattleState_Start, proto.StatusCode_StatusCode_DungeonCheat
	}
	if runtime.DeadMonsters[encounterID] {
		return proto.BattleState_Finish, proto.StatusCode_StatusCode_OK
	}
	if state, ok := runtime.Encounters[encounterID]; ok {
		return state, proto.StatusCode_StatusCode_OK
	}
	return proto.BattleState_Start, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) updateEncounter(s *model.Player, sd *model.SceneDungeon, encounterID uint32, state proto.BattleState) proto.StatusCode {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return status
	}
	if runtime.State != model.DungeonRuntimeRunning {
		return proto.StatusCode_StatusCode_ChallengeNotStart
	}
	if !runtime.MonstersByIndex {
		return proto.StatusCode_StatusCode_DungeonCheat
	}
	if _, ok := runtime.Monsters[encounterID]; !ok {
		return proto.StatusCode_StatusCode_DungeonCheat
	}
	if state != proto.BattleState_Start && state != proto.BattleState_Reset &&
		state != proto.BattleState_Finish {
		return proto.StatusCode_StatusCode_DungeonCheat
	}
	if runtime.DeadMonsters[encounterID] && state != proto.BattleState_Finish {
		return proto.StatusCode_StatusCode_DungeonCheat
	}
	if state == proto.BattleState_Finish && !runtime.DeadMonsters[encounterID] {
		return proto.StatusCode_StatusCode_DungeonCheat
	}
	runtime.Encounters[encounterID] = state
	return proto.StatusCode_StatusCode_OK
}

type dungeonInventorySnapshot struct {
	items *model.ItemModel
	base  map[uint32]*model.ItemBaseInfo
}

func snapshotDungeonInventory(s *model.Player) dungeonInventorySnapshot {
	items := s.GetItemModel()
	snapshot := dungeonInventorySnapshot{items: items}
	if items.ItemBaseInfo != nil {
		snapshot.base = make(map[uint32]*model.ItemBaseInfo, len(items.ItemBaseInfo))
		for id, item := range items.ItemBaseInfo {
			if item == nil {
				snapshot.base[id] = nil
				continue
			}
			copy := *item
			snapshot.base[id] = &copy
		}
	}
	return snapshot
}

func (s dungeonInventorySnapshot) restore() {
	s.items.ItemBaseInfo = s.base
}

func (d *dungeonService) grantRewards(s *model.Player, rewards []dungeonReward) ([]*proto.ItemDetail, proto.StatusCode) {
	if d.addReward == nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	snapshot := snapshotDungeonInventory(s)
	items := make([]*proto.ItemDetail, 0, len(rewards))
	for _, reward := range rewards {
		item := d.addReward(s, reward)
		if item == nil {
			snapshot.restore()
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		items = append(items, item)
	}
	return items, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) finish(s *model.Player, sd *model.SceneDungeon) ([]*proto.ItemDetail, proto.StatusCode) {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if runtime.State == model.DungeonRuntimeClaimed {
		return nil, proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	if runtime.State != model.DungeonRuntimeFinished {
		return nil, proto.StatusCode_StatusCode_ChallengeNotEnd
	}
	cfg, status := d.config(runtime.DungeonID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if !d.allMonstersDead(runtime) {
		return nil, proto.StatusCode_StatusCode_EXPLORE_NOT_FINISH
	}
	clearRewards, status := d.rewards(cfg.RewardID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	starRewards, status := d.rewards(cfg.StarRewardID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	info := sd.Info
	if info.LastRunToken != runtime.Token || info.RewardClaimedToken == runtime.Token {
		return nil, proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	if info.FinishTimes == math.MaxUint32 {
		return nil, proto.StatusCode_StatusCode_SYS_ERR
	}
	grantNewStarReward := cfg.StarRewardID != 0 && runtime.Stars > 0 && info.StarRewardClaimed&starMask(runtime.Stars) != starMask(runtime.Stars)
	grantedRewards := append([]dungeonReward(nil), clearRewards...)
	if grantNewStarReward {
		grantedRewards = append(grantedRewards, starRewards...)
	}
	if !d.rewardItemsValid(nil, grantedRewards) {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if !d.rewardItemsValid(s, grantedRewards) {
		return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}

	// Grant and persist in one logical settlement. Persisting the claim before
	// adding items would permanently lose rewards if the process died between
	// those operations; saving after both lets a failed save retry safely.
	inventory := snapshotDungeonInventory(s)
	items, status := d.grantRewards(s, grantedRewards)
	if status != proto.StatusCode_StatusCode_OK {
		inventory.restore()
		return nil, status
	}
	previous := *info
	info.FinishTimes++
	info.LastFinishTime = runtime.EndedAt - runtime.StartedAt
	info.FirstClearRewardClaimed = true
	if runtime.Stars > info.Stars {
		info.Stars = runtime.Stars
	}
	info.RewardClaimedToken = runtime.Token
	if grantNewStarReward {
		info.StarRewardClaimed |= starMask(runtime.Stars)
	}
	if d.save != nil {
		if err := d.save(s); err != nil {
			*info = previous
			inventory.restore()
			return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		}
	}
	runtime.State = model.DungeonRuntimeClaimed
	return items, proto.StatusCode_StatusCode_OK
}

func starMask(stars uint32) uint32 {
	if stars >= 3 {
		return 0b111
	}
	return (1 << stars) - 1
}

func clearDungeonRuntime(sd *model.SceneDungeon) {
	if sd == nil {
		return
	}
	if runtime := sd.Runtime; runtime != nil {
		runtime.Token = 0
		runtime.Monsters = nil
		runtime.DeadMonsters = nil
		runtime.Encounters = nil
		runtime.Drops = nil
		runtime.Treasures = nil
	}
	sd.Runtime = nil
	if sd.ScenePlayerInfo != nil {
		if sd.TempPack != nil {
			sd.TempPack.TempPack = nil
			sd.TempPack.TempAreaPack = nil
		}
		sd.TempPack = model.NewSceneTempPack()
	}
}

func (d *dungeonService) exit(s *model.Player, sd *model.SceneDungeon) proto.StatusCode {
	if _, status := validDungeonRuntime(s, sd); status != proto.StatusCode_StatusCode_OK {
		return status
	}
	d.abandon(s, sd)
	return proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) abandon(s *model.Player, sd *model.SceneDungeon) {
	if sd == nil {
		return
	}
	if _, status := validDungeonRuntime(s, sd); status == proto.StatusCode_StatusCode_OK && sd.Info.ExitTimes < math.MaxUint32 {
		sd.Info.ExitTimes++
	}
	clearDungeonRuntime(sd)
}

// cleanupDungeonPlayer is the disconnect/shutdown hook. It deliberately does
// not switch scenes: the caller is already removing the player from the world.
func (g *Game) cleanupDungeonPlayer(s *model.Player) {
	if g == nil || s == nil {
		return
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		return
	}
	if sd, ok := scenePlayer.CurScene.(*model.SceneDungeon); ok {
		g.dungeonRuntime().abandon(s, sd)
	}
}

func (d *dungeonService) pickup(s *model.Player, sd *model.SceneDungeon, dropIndex uint32, pickIndex int32) ([]*proto.ItemDetail, proto.StatusCode) {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if runtime.State == model.DungeonRuntimeEntered {
		return nil, proto.StatusCode_StatusCode_ChallengeNotStart
	}
	if runtime.Drops[dropIndex] == nil || runtime.Drops[dropIndex] != sd.GetTempPack().GetTempAreaPack(dropIndex) {
		return nil, proto.StatusCode_StatusCode_GACHA_ALREADY_FULL_PICK
	}
	return d.pickupDrop(s, sd, dropIndex, pickIndex)
}

func (d *dungeonService) pickupDrop(s *model.Player, sd *model.SceneDungeon, dropIndex uint32, pickIndex int32) ([]*proto.ItemDetail, proto.StatusCode) {
	pack := sd.GetTempPack()
	drop := pack.GetTempAreaPack(dropIndex)
	if pack == nil || drop == nil || len(drop.Items) == 0 {
		return nil, proto.StatusCode_StatusCode_GACHA_ALREADY_FULL_PICK
	}
	originalItems := append([]model.EBagItemTag(nil), drop.Items...)
	restoreDrop := func() {
		drop.Items = originalItems
		pack.TempAreaPack[dropIndex] = drop
	}
	picked := pack.Pickup(dropIndex, pickIndex)
	if len(picked) == 0 {
		restoreDrop()
		return nil, proto.StatusCode_StatusCode_GACHA_ALREADY_FULL_PICK
	}
	rewards := make([]dungeonReward, 0, len(picked))
	for _, detail := range picked {
		if detail == nil || detail.MainItem == nil || detail.MainItem.ItemId == 0 {
			restoreDrop()
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		base := detail.MainItem.GetBaseItem()
		if base == nil || base.ItemId != detail.MainItem.ItemId || base.Num <= 0 {
			restoreDrop()
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		rewards = append(rewards, dungeonReward{
			ItemID:   detail.MainItem.ItemId,
			MinCount: base.Num,
			MaxCount: base.Num,
			ItemType: detail.MainItem.ItemTag,
		})
	}
	if !d.rewardItemsValid(nil, rewards) {
		restoreDrop()
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if !d.rewardItemsValid(s, rewards) {
		restoreDrop()
		return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}
	items, status := d.grantRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		restoreDrop()
		return nil, status
	}
	return items, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) openTreasure(s *model.Player, sd *model.SceneDungeon, index uint32) ([]*proto.ItemDetail, proto.StatusCode) {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if runtime.State != model.DungeonRuntimeRunning {
		return nil, proto.StatusCode_StatusCode_ChallengeNotStart
	}
	cfg, status := d.config(runtime.DungeonID)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	box := sd.Info.GetTreasureBoxes()[index]
	if box == nil || box.Index != index || cfg.Treasures[index] != box.BoxId {
		return nil, proto.StatusCode_StatusCode_TREASURE_BOX_NOT_EXIST
	}
	if treasure := runtime.Treasures[index]; treasure != nil {
		if treasure.Picked {
			return nil, proto.StatusCode_StatusCode_AlreadyGotReward
		}
		if treasure.Drop == nil {
			return nil, proto.StatusCode_StatusCode_TREASURE_BOX_NOT_EXIST
		}
		return treasure.Drop.PbDropItem().Items, proto.StatusCode_StatusCode_OK
	}
	if box.State != proto.TreasureBoxState_TreasureBoxState_Close {
		return nil, proto.StatusCode_StatusCode_BAD_REQ
	}
	rewards, err := d.loadTreasureReward(box.BoxId)
	if err != nil || len(rewards) == 0 {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if !d.rewardItemsValid(nil, rewards) {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if !d.rewardItemsValid(s, rewards) {
		return nil, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
	}
	items := make([]model.EBagItemTag, 0, len(rewards))
	for _, reward := range rewards {
		items = append(items, &model.ItemBaseInfo{
			ItemId:   reward.ItemID,
			Num:      reward.Count(),
			ItemType: reward.ItemType,
			PackType: proto.ItemDetail_PackType_TempStorageArea,
		})
	}
	drop := sd.GetTempPack().NewDrop(items)
	if drop == nil {
		return nil, proto.StatusCode_StatusCode_SYS_ERR
	}
	box.State = proto.TreasureBoxState_TreasureBoxState_Open
	runtime.Treasures[index] = &model.DungeonTreasureRuntime{Drop: drop}
	return drop.PbDropItem().Items, proto.StatusCode_StatusCode_OK
}

func (d *dungeonService) pickupTreasure(s *model.Player, sd *model.SceneDungeon, index uint32, pickIndex int32) ([]*proto.ItemDetail, proto.StatusCode) {
	runtime, status := validDungeonRuntime(s, sd)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if runtime.State == model.DungeonRuntimeEntered {
		return nil, proto.StatusCode_StatusCode_ChallengeNotStart
	}
	treasure := runtime.Treasures[index]
	if treasure == nil || treasure.Drop == nil {
		return nil, proto.StatusCode_StatusCode_TREASURE_BOX_NOT_EXIST
	}
	if treasure.Picked {
		return nil, proto.StatusCode_StatusCode_AlreadyGotReward
	}
	if treasure.Drop != sd.GetTempPack().GetTempAreaPack(treasure.Drop.Index) {
		return nil, proto.StatusCode_StatusCode_TREASURE_BOX_NOT_EXIST
	}
	items, status := d.pickupDrop(s, sd, treasure.Drop.Index, pickIndex)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	treasure.Picked = len(treasure.Drop.Items) == 0
	return items, proto.StatusCode_StatusCode_OK
}
