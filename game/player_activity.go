package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/config"
	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

const (
	activityScopeLifetime = "lifetime"
	activityScopePeriod   = "period"
	activityScopeDay      = "day"
	activityScopeWeek     = "week"
)

type activityConfigRow struct {
	ID              uint32 `json:"i_d"`
	ActivityID      uint32 `json:"activity_i_d"`
	NewActivityType int32  `json:"new_activity_type"`
	OpeningTime     string `json:"opening_time"`
	ClosingTime     string `json:"closing_time"`
	openUnix        int64
	closeUnix       int64
}

func (r activityConfigRow) templateID() uint32 {
	if r.ActivityID != 0 {
		return r.ActivityID
	}
	return r.ID
}

type activityItem struct {
	ID  uint32
	Num int64
}

type activitySignConfig struct {
	ID    uint32                  `json:"i_d"`
	Items []activitySignDayConfig `json:"activity_sign_item"`
}

type activitySignDayConfig struct {
	Day     uint32               `json:"days"`
	Rewards []activitySignReward `json:"activity_sign_reward_item"`
}

type activitySignReward struct {
	ItemID  uint32 `json:"item_i_d"`
	ItemNum int64  `json:"item_num"`
}

type activityQuestConfig struct {
	ID           uint32                       `json:"i_d"`
	RewardID     uint32                       `json:"activity_quest_reward_i_d"`
	AchieveItems []activityQuestAchieveConfig `json:"activity_quest_achieve_items"`
}

type activityQuestAchieveConfig struct {
	Day       uint32               `json:"days"`
	AchieveID uint32               `json:"achieve_i_d"`
	Rewards   []activityDirectItem `json:"activity_quest_item"`
}

type activityQuestRewardConfig struct {
	ID    uint32                        `json:"i_d"`
	Items []activityQuestProgressReward `json:"activity_quest_reward_item"`
}

type activityQuestProgressReward struct {
	Index            uint32 `json:"index"`
	ConditionItemID  uint32 `json:"condition_item_i_d"`
	ConditionItemNum int64  `json:"condition_item_num"`
	RewardItemID     uint32 `json:"reward_item_i_d"`
	RewardItemNum    int64  `json:"reward_item_num"`
}

type activityDirectItem struct {
	ItemID  uint32 `json:"item_i_d"`
	ItemNum int64  `json:"item_num"`
}

type activityRegularConfig struct {
	ID                 uint32 `json:"i_d"`
	DailyPoolID        uint32 `json:"daily_pool_i_d"`
	DailyRefreshPoolID uint32 `json:"daily_refresh_pool_i_d"`
	RandomPoolID       uint32 `json:"random_pool_i_d"`
	PoolID             uint32 `json:"pool_i_d"`
}

type activityRegularPoolConfig struct {
	ID    uint32                         `json:"i_d"`
	Items []activityRegularAchieveConfig `json:"activity_regular_pool_item"`
}

type activityRegularAchieveConfig struct {
	AchieveID uint32               `json:"achieve_i_d"`
	Days      uint32               `json:"days"`
	Rewards   []activityDirectItem `json:"activity_regular_pool_reward_item"`
}

type activityGiftConfig struct {
	ID     uint32                      `json:"i_d"`
	ShopID uint32                      `json:"shop_i_d"`
	Items  []activityGiftAchieveConfig `json:"activity_gift_item"`
}

type activityGiftAchieveConfig struct {
	AchieveID uint32               `json:"achieve_i_d"`
	Rewards   []activityDirectItem `json:"a_activity_gift_reward_item"`
}

type battlePassConfig struct {
	ID            uint32                  `json:"i_d"`
	ShopID        uint32                  `json:"shop_i_d"`
	MaxExp        uint32                  `json:"max_exp"`
	DayQuestID    uint32                  `json:"day_quest_i_d"`
	WeekQuestID   uint32                  `json:"week_quest_i_d"`
	SeasonQuestID uint32                  `json:"season_quest_i_d"`
	Levels        []battlePassLevelConfig `json:"battle_pass_level_ups"`
}

type battlePassLevelConfig struct {
	Level   uint32               `json:"level"`
	Exp     uint32               `json:"exp"`
	Rewards []activityDirectItem `json:"battle_pass_reward"`
}

type battlePassQuestConfig struct {
	ID    uint32                `json:"i_d"`
	Items []battlePassQuestItem `json:"battle_pass_queset_item"`
}

type battlePassQuestItem struct {
	AchieveID uint32 `json:"achieve_i_d"`
	Exp       uint32 `json:"exp"`
}

type activityTurntableConfig struct {
	ID                  uint32 `json:"i_d"`
	PoolID              uint32 `json:"turntable_pool_i_d"`
	CostItemID          uint32 `json:"cost_item_i_d"`
	FirstOneNeedItemNum int64  `json:"first_one_need_item_num"`
	OneNeedItemNum      int64  `json:"one_need_item_num"`
}

type activityTurntablePoolConfig struct {
	ID    uint32                      `json:"i_d"`
	Items []activityTurntablePoolItem `json:"turntable_pool_item"`
}

type activityTurntablePoolItem struct {
	Index    uint32 `json:"index"`
	ItemID   uint32 `json:"item_i_d"`
	ItemNum  int64  `json:"item_num"`
	LimitNum uint32 `json:"limit_num"`
	Weight   uint32 `json:"weight"`
	Special  bool   `json:"is_special"`
}

type abyssSeasonConfig struct {
	ID            uint32 `json:"i_d"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	SeasonStage   uint32 `json:"season_stage"`
	SeasonAchieve uint32 `json:"season_achieve"`
	openUnix      int64
	closeUnix     int64
}

type abyssStageConfig struct {
	ID    uint32                  `json:"i_d"`
	Items []abyssStageGroupConfig `json:"abyss_stage_group_info"`
}

type abyssStageGroupConfig struct {
	DungeonID uint32 `json:"dungeon_i_d"`
	Challenge bool   `json:"is_challenge_dungeon"`
}

type abyssSeasonAchieveConfig struct {
	ID    uint32                   `json:"i_d"`
	Items []abyssSeasonAchieveItem `json:"season_achieve_group_info"`
}

type abyssSeasonAchieveItem struct {
	AchieveID uint32 `json:"achieve_i_d"`
	RewardID  uint32 `json:"reward_i_d"`
}

type achieveConfig struct {
	ID         uint32 `json:"i_d"`
	CountParam uint64 `json:"count_param"`
	Condition  int32  `json:"new_achieve_condition"`
}

type achieveQuestConfig struct {
	ID       uint32 `json:"i_d"`
	GroupID  uint32 `json:"achieve_group_i_d"`
	RewardID uint32 `json:"reward_i_d"`
}

type achieveQuestGroupConfig struct {
	ID    uint32                  `json:"i_d"`
	Items []achieveQuestGroupItem `json:"achieve_quest_group_info"`
}

type achieveQuestGroupItem struct {
	AchieveID uint32 `json:"achieve_condition_i_d"`
	RewardID  uint32 `json:"reward_i_d"`
}

type rewardPoolConfig struct {
	ID     uint32                  `json:"i_d"`
	Groups []rewardPoolGroupConfig `json:"reward_pool_group"`
}

type rewardPoolGroupConfig struct {
	ID uint32 `json:"reward_pool_group_i_d"`
}

type rewardItemPoolConfig struct {
	ID     uint32                      `json:"i_d"`
	Groups []rewardItemPoolGroupConfig `json:"reward_item_pool_group"`
}

type rewardItemPoolGroupConfig struct {
	ItemID       uint32 `json:"item_i_d"`
	ItemMinCount int64  `json:"item_min_count"`
	ItemMaxCount int64  `json:"item_max_count"`
	MinLevel     uint32 `json:"min_level_limit"`
	MaxLevel     uint32 `json:"max_level_limit"`
}

type globalConfigRow struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type itemConfigRow struct {
	ID  uint32 `json:"i_d"`
	Tag int32  `json:"new_bag_item_tag"`
}

type achievementSpec struct {
	TemplateID uint32
	Scope      string
}

type activityConfigs struct {
	Activities          map[uint32]activityConfigRow
	Signs               map[uint32]activitySignConfig
	Quests              map[uint32]activityQuestConfig
	QuestRewards        map[uint32]activityQuestRewardConfig
	Regulars            map[uint32]activityRegularConfig
	RegularPools        map[uint32]activityRegularPoolConfig
	Gifts               map[uint32]activityGiftConfig
	BattlePasses        map[uint32]battlePassConfig
	BattlePassQuests    map[uint32]battlePassQuestConfig
	Turntables          map[uint32]activityTurntableConfig
	TurntablePools      map[uint32]activityTurntablePoolConfig
	AbyssSeasons        map[uint32]abyssSeasonConfig
	AbyssStages         map[uint32]abyssStageConfig
	AbyssAchieves       map[uint32]abyssSeasonAchieveConfig
	Achieves            map[uint32]achieveConfig
	AchievesByCondition map[int32][]uint32
	AchieveQuestByGroup map[uint32]achieveQuestConfig
	AchieveQuestGroups  map[uint32]achieveQuestGroupConfig
	AchieveRewards      map[uint32]uint32
	RewardPools         map[uint32]rewardPoolConfig
	RewardItemPools     map[uint32]rewardItemPoolConfig
	Globals             map[string][]string
	Items               map[uint32]itemConfigRow
	AchievementSpecs    map[uint32][]achievementSpec
}

func newActivityConfigs() *activityConfigs {
	return &activityConfigs{
		Activities:          make(map[uint32]activityConfigRow),
		Signs:               make(map[uint32]activitySignConfig),
		Quests:              make(map[uint32]activityQuestConfig),
		QuestRewards:        make(map[uint32]activityQuestRewardConfig),
		Regulars:            make(map[uint32]activityRegularConfig),
		RegularPools:        make(map[uint32]activityRegularPoolConfig),
		Gifts:               make(map[uint32]activityGiftConfig),
		BattlePasses:        make(map[uint32]battlePassConfig),
		BattlePassQuests:    make(map[uint32]battlePassQuestConfig),
		Turntables:          make(map[uint32]activityTurntableConfig),
		TurntablePools:      make(map[uint32]activityTurntablePoolConfig),
		AbyssSeasons:        make(map[uint32]abyssSeasonConfig),
		AbyssStages:         make(map[uint32]abyssStageConfig),
		AbyssAchieves:       make(map[uint32]abyssSeasonAchieveConfig),
		Achieves:            make(map[uint32]achieveConfig),
		AchievesByCondition: make(map[int32][]uint32),
		AchieveQuestByGroup: make(map[uint32]achieveQuestConfig),
		AchieveQuestGroups:  make(map[uint32]achieveQuestGroupConfig),
		AchieveRewards:      make(map[uint32]uint32),
		RewardPools:         make(map[uint32]rewardPoolConfig),
		RewardItemPools:     make(map[uint32]rewardItemPoolConfig),
		Globals:             make(map[string][]string),
		Items:               make(map[uint32]itemConfigRow),
		AchievementSpecs:    make(map[uint32][]achievementSpec),
	}
}

func activityConfigWarn(format string, args ...any) {
	if log.App != nil {
		log.App.Warnf(format, args...)
	}
}

func loadActivityConfigs(excelPath string) *activityConfigs {
	c := newActivityConfigs()
	read := func(name, key string, dst any) {
		path := filepath.Join(excelPath, name)
		data, err := os.ReadFile(path)
		if err != nil {
			activityConfigWarn("活动配置读取失败 %s: %v", name, err)
			return
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(data, &envelope); err != nil {
			activityConfigWarn("活动配置解析失败 %s: %v", name, err)
			return
		}
		raw, ok := envelope[key]
		if !ok {
			activityConfigWarn("活动配置缺少 %s", key)
			return
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			activityConfigWarn("活动配置解析失败 %s.%s: %v", name, key, err)
		}
	}

	var activities struct {
		Datas []activityConfigRow `json:"datas"`
	}
	read("Activity.json", "activity", &activities)
	for _, row := range activities.Datas {
		if row.ID == 0 {
			continue
		}
		row.openUnix, _ = parseActivityTime(row.OpeningTime)
		row.closeUnix, _ = parseActivityTime(row.ClosingTime)
		c.Activities[row.ID] = row
	}

	var signs struct {
		Datas []activitySignConfig `json:"datas"`
	}
	read("ActivitySign.json", "activity_sign", &signs)
	for _, row := range signs.Datas {
		if row.ID != 0 {
			c.Signs[row.ID] = row
		}
	}

	var quests struct {
		Quests struct {
			Datas []activityQuestConfig `json:"datas"`
		} `json:"activity_quest"`
		Rewards struct {
			Datas []activityQuestRewardConfig `json:"datas"`
		} `json:"activity_quest_reward"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "ActivityQuest.json"))
		if err == nil {
			if err := json.Unmarshal(data, &quests); err != nil {
				activityConfigWarn("活动配置解析失败 ActivityQuest.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 ActivityQuest.json: %v", err)
		}
	}
	for _, row := range quests.Quests.Datas {
		if row.ID != 0 {
			c.Quests[row.ID] = row
		}
	}
	for _, row := range quests.Rewards.Datas {
		if row.ID != 0 {
			c.QuestRewards[row.ID] = row
		}
	}

	var regulars struct {
		Regular struct {
			Datas []activityRegularConfig `json:"datas"`
		} `json:"activity_regular"`
		Pools struct {
			Datas []activityRegularPoolConfig `json:"datas"`
		} `json:"activity_regular_pool"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "ActivityRegular.json"))
		if err == nil {
			if err := json.Unmarshal(data, &regulars); err != nil {
				activityConfigWarn("活动配置解析失败 ActivityRegular.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 ActivityRegular.json: %v", err)
		}
	}
	for _, row := range regulars.Regular.Datas {
		if row.ID != 0 {
			c.Regulars[row.ID] = row
		}
	}
	for _, row := range regulars.Pools.Datas {
		if row.ID != 0 {
			c.RegularPools[row.ID] = row
		}
	}

	var gifts struct {
		Datas []activityGiftConfig `json:"datas"`
	}
	read("ActivityGift.json", "activity_gift", &gifts)
	for _, row := range gifts.Datas {
		if row.ID != 0 {
			c.Gifts[row.ID] = row
		}
	}

	var battlePass struct {
		Passes struct {
			Datas []battlePassConfig `json:"datas"`
		} `json:"battle_pass"`
		Quests struct {
			Datas []battlePassQuestConfig `json:"datas"`
		} `json:"battle_pass_queset"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "BattlePass.json"))
		if err == nil {
			if err := json.Unmarshal(data, &battlePass); err != nil {
				activityConfigWarn("活动配置解析失败 BattlePass.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 BattlePass.json: %v", err)
		}
	}
	for _, row := range battlePass.Passes.Datas {
		if row.ID != 0 {
			sort.Slice(row.Levels, func(i, j int) bool { return row.Levels[i].Level < row.Levels[j].Level })
			c.BattlePasses[row.ID] = row
		}
	}
	for _, row := range battlePass.Quests.Datas {
		if row.ID != 0 {
			c.BattlePassQuests[row.ID] = row
		}
	}

	var turntables struct {
		Tables struct {
			Datas []activityTurntableConfig `json:"datas"`
		} `json:"activity_turntable"`
		Pools struct {
			Datas []activityTurntablePoolConfig `json:"datas"`
		} `json:"turntable_pool"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "ActivityTurntable.json"))
		if err == nil {
			if err := json.Unmarshal(data, &turntables); err != nil {
				activityConfigWarn("活动配置解析失败 ActivityTurntable.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 ActivityTurntable.json: %v", err)
		}
	}
	for _, row := range turntables.Tables.Datas {
		if row.ID != 0 {
			c.Turntables[row.ID] = row
		}
	}
	for _, row := range turntables.Pools.Datas {
		if row.ID != 0 {
			c.TurntablePools[row.ID] = row
		}
	}

	var abyss struct {
		Seasons struct {
			Datas []abyssSeasonConfig `json:"datas"`
		} `json:"abyss_season"`
		Stages struct {
			Datas []abyssStageConfig `json:"datas"`
		} `json:"abyss_stage"`
		Achieves struct {
			Datas []abyssSeasonAchieveConfig `json:"datas"`
		} `json:"season_achieve"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "Abyss.json"))
		if err == nil {
			if err := json.Unmarshal(data, &abyss); err != nil {
				activityConfigWarn("活动配置解析失败 Abyss.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 Abyss.json: %v", err)
		}
	}
	for _, row := range abyss.Seasons.Datas {
		if row.ID != 0 {
			row.openUnix, _ = parseActivityTime(row.StartTime)
			row.closeUnix, _ = parseActivityTime(row.EndTime)
			c.AbyssSeasons[row.ID] = row
		}
	}
	for _, row := range abyss.Stages.Datas {
		if row.ID != 0 {
			c.AbyssStages[row.ID] = row
		}
	}
	for _, row := range abyss.Achieves.Datas {
		if row.ID != 0 {
			c.AbyssAchieves[row.ID] = row
		}
	}

	var achieves struct {
		Datas []achieveConfig `json:"datas"`
	}
	read("Achieve.json", "achieve", &achieves)
	for _, row := range achieves.Datas {
		if row.ID == 0 {
			continue
		}
		c.Achieves[row.ID] = row
		c.AchievesByCondition[row.Condition] = append(c.AchievesByCondition[row.Condition], row.ID)
	}

	var achieveQuests struct {
		Quests struct {
			Datas []achieveQuestConfig `json:"datas"`
		} `json:"achieve_quest"`
		Groups struct {
			Datas []achieveQuestGroupConfig `json:"datas"`
		} `json:"achieve_quest_group"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "AchieveQuest.json"))
		if err == nil {
			if err := json.Unmarshal(data, &achieveQuests); err != nil {
				activityConfigWarn("活动配置解析失败 AchieveQuest.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 AchieveQuest.json: %v", err)
		}
	}
	for _, row := range achieveQuests.Quests.Datas {
		if row.GroupID != 0 {
			c.AchieveQuestByGroup[row.GroupID] = row
		}
	}
	for _, row := range achieveQuests.Groups.Datas {
		if row.ID == 0 {
			continue
		}
		c.AchieveQuestGroups[row.ID] = row
		for _, item := range row.Items {
			if item.AchieveID == 0 || item.RewardID == 0 {
				continue
			}
			if old, ok := c.AchieveRewards[item.AchieveID]; ok && old != item.RewardID {
				delete(c.AchieveRewards, item.AchieveID)
				continue
			}
			c.AchieveRewards[item.AchieveID] = item.RewardID
		}
	}

	var rewards struct {
		Pools struct {
			Datas []rewardPoolConfig `json:"datas"`
		} `json:"reward_pool"`
		Items struct {
			Datas []rewardItemPoolConfig `json:"datas"`
		} `json:"reward_item_pool"`
	}
	{
		data, err := os.ReadFile(filepath.Join(excelPath, "Reward.json"))
		if err == nil {
			if err := json.Unmarshal(data, &rewards); err != nil {
				activityConfigWarn("活动配置解析失败 Reward.json: %v", err)
			}
		} else {
			activityConfigWarn("活动配置读取失败 Reward.json: %v", err)
		}
	}
	for _, row := range rewards.Pools.Datas {
		if row.ID != 0 {
			c.RewardPools[row.ID] = row
		}
	}
	for _, row := range rewards.Items.Datas {
		if row.ID != 0 {
			c.RewardItemPools[row.ID] = row
		}
	}

	var globals struct {
		Datas []globalConfigRow `json:"datas"`
	}
	read("GlobalConfig.json", "config", &globals)
	for _, row := range globals.Datas {
		if row.Key != "" {
			c.Globals[row.Key] = row.Values
		}
	}

	var items struct {
		Datas []itemConfigRow `json:"datas"`
	}
	read("Item.json", "item", &items)
	for _, row := range items.Datas {
		if row.ID != 0 {
			c.Items[row.ID] = row
		}
	}

	c.buildAchievementSpecs()
	return c
}

func parseActivityTime(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

func (c *activityConfigs) buildAchievementSpecs() {
	add := func(id, template uint32, scope string) {
		if id == 0 || template == 0 {
			return
		}
		for _, spec := range c.AchievementSpecs[id] {
			if spec.TemplateID == template && spec.Scope == scope {
				return
			}
		}
		c.AchievementSpecs[id] = append(c.AchievementSpecs[id], achievementSpec{TemplateID: template, Scope: scope})
	}
	for _, quest := range c.Quests {
		for _, item := range quest.AchieveItems {
			add(item.AchieveID, quest.ID, activityScopePeriod)
		}
	}
	for _, regular := range c.Regulars {
		for _, poolID := range []uint32{regular.DailyPoolID, regular.DailyRefreshPoolID, regular.RandomPoolID, regular.PoolID} {
			pool := c.RegularPools[poolID]
			scope := activityScopePeriod
			if poolID == regular.DailyPoolID || poolID == regular.DailyRefreshPoolID {
				scope = activityScopeDay
			}
			for _, item := range pool.Items {
				add(item.AchieveID, regular.ID, scope)
			}
		}
	}
	for _, gift := range c.Gifts {
		for _, item := range gift.Items {
			add(item.AchieveID, gift.ID, activityScopePeriod)
		}
	}
	for _, pass := range c.BattlePasses {
		for _, group := range []struct {
			id    uint32
			scope string
		}{{pass.DayQuestID, activityScopeDay}, {pass.WeekQuestID, activityScopeWeek}, {pass.SeasonQuestID, activityScopePeriod}} {
			for _, item := range c.BattlePassQuests[group.id].Items {
				add(item.AchieveID, pass.ID, group.scope)
			}
		}
	}
}

type activityStatusError struct {
	status proto.StatusCode
	cause  error
}

func (e *activityStatusError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.status.String()
}

func activityErr(status proto.StatusCode, format string, args ...any) error {
	return &activityStatusError{status: status, cause: fmt.Errorf(format, args...)}
}

func ActivityErrorStatus(err error) proto.StatusCode {
	var statusErr *activityStatusError
	if errors.As(err, &statusErr) {
		return statusErr.status
	}
	return proto.StatusCode_StatusCode_SYS_ERR
}

type activityItemGranter func(*model.Player, []activityItem) ([]*proto.ItemDetail, error)

type activityService struct {
	conf        *activityConfigs
	now         func() int64
	randomInt   func(int) int
	save        func(*model.Player) error
	playerLevel func(*model.Player) uint32
	grant       activityItemGranter
	location    *time.Location
}

var (
	rewardActivityConfigOnce sync.Once
	rewardActivityConfigData *activityConfigs
)

func defaultActivityService() *activityService {
	rewardActivityConfigOnce.Do(func() {
		resources := config.GetResources()
		rewardActivityConfigData = loadActivityConfigs(filepath.Join(resources.GetResourcePath(), "Excel"))
	})
	return &activityService{
		conf:        rewardActivityConfigData,
		now:         func() int64 { return time.Now().Unix() },
		randomInt:   rand.IntN,
		save:        func(s *model.Player) error { return s.SavePlayerNow() },
		playerLevel: defaultActivityPlayerLevel,
		location:    time.Local,
	}
}

func defaultActivityPlayerLevel(s *model.Player) uint32 {
	if basic, ok := db.GetGameBasic(s.UserId); ok {
		return basic.Level
	}
	return 0
}

func (a *activityService) date(now int64) time.Time {
	location := a.location
	if location == nil {
		location = time.Local
	}
	return time.Unix(now, 0).In(location)
}

func (a *activityService) dayToken(now int64) string {
	return a.date(now).Format("2006-01-02")
}

func (a *activityService) weekToken(now int64) string {
	year, week := a.date(now).ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func (a *activityService) commit(s *model.Player, fn func() ([]*proto.ItemDetail, error)) ([]*proto.ItemDetail, error) {
	snapshot, err := sonic.Marshal(s)
	if err != nil {
		return nil, activityErr(proto.StatusCode_StatusCode_SYS_ERR, "序列化活动事务失败: %w", err)
	}
	items, err := fn()
	if err == nil {
		err = a.save(s)
	}
	if err != nil {
		var restored model.Player
		if restoreErr := sonic.Unmarshal(snapshot, &restored); restoreErr != nil {
			log.Game.Errorf("UserId:%v 活动事务回滚失败:%v", s.UserId, restoreErr)
		} else {
			restored.Conn = s.Conn
			restored.Online = s.Online
			restored.NetFreeze = s.NetFreeze
			restored.Created = s.Created
			restored.ActiveTime = s.ActiveTime
			restored.LastSaveTime = s.LastSaveTime
			restored.SaveLock = s.SaveLock
			restored.UserId = s.UserId
			restored.NickName = s.NickName
			restored.LoginUUID = s.LoginUUID
			*s = restored
		}
		return nil, err
	}
	return items, nil
}

func (a *activityService) consume(s *model.Player, itemID uint32, count int64) (*proto.PackNotice, error) {
	if itemID == 0 || count <= 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "活动消耗配置缺失")
	}
	if _, ok := a.conf.Items[itemID]; !ok {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "消耗物品配置缺失:%d", itemID)
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, activityErr(proto.StatusCode_StatusCode_SYS_ERR, "创建物品事务失败:%w", err)
	}
	tx.DelBaseItem(itemID, count)
	if tx.Error != nil {
		tx.Rollback()
		return nil, &activityStatusError{status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, cause: tx.Error}
	}
	tx.Commit()
	if tx.Error != nil {
		tx.Rollback()
		return nil, &activityStatusError{status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, cause: tx.Error}
	}
	return tx.PackNotice, nil
}

func activityStackableItem(tag proto.EBagItemTag) bool {
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
		proto.EBagItemTag_EBagItemTag_MonthlyCard,
		proto.EBagItemTag_EBagItemTag_PhoneCase,
		proto.EBagItemTag_EBagItemTag_Pendant,
		proto.EBagItemTag_EBagItemTag_AvatarFrame,
		proto.EBagItemTag_EBagItemTag_IntimacyGift,
		proto.EBagItemTag_EBagItemTag_MusicNote,
		proto.EBagItemTag_EBagItemTag_BattlePassCard,
		proto.EBagItemTag_EBagItemTag_MonthlyGiftCard,
		proto.EBagItemTag_EBagItemTag_BattlePassGiftCard,
		proto.EBagItemTag_EBagItemTag_SeasonalMiniGamesItem,
		proto.EBagItemTag_EBagItemTag_TreasureMap,
		proto.EBagItemTag_EBagItemTag_TreasureMapFrag,
		proto.EBagItemTag_EBagItemTag_PetGrid,
		proto.EBagItemTag_EBagItemTag_PetMmaterials,
		proto.EBagItemTag_EBagItemTag_Currency:
		return true
	default:
		return false
	}
}

func (a *activityService) grantItems(s *model.Player, items []activityItem) ([]*proto.ItemDetail, error) {
	if len(items) == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "活动奖励配置缺失")
	}
	for _, item := range items {
		if item.ID == 0 || item.Num <= 0 {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "活动奖励配置无效")
		}
		if _, ok := a.conf.Items[item.ID]; !ok {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励物品配置缺失:%d", item.ID)
		}
	}
	granter := a.grant
	if granter == nil {
		granter = func(s *model.Player, items []activityItem) ([]*proto.ItemDetail, error) {
			details := make([]*proto.ItemDetail, 0, len(items))
			for _, item := range items {
				conf := a.conf.Items[item.ID]
				tag := proto.EBagItemTag(conf.Tag)
				if activityStackableItem(tag) {
					bag := s.GetItemModel().GetItemBaseMap()
					info := bag[item.ID]
					if info == nil {
						info = &model.ItemBaseInfo{ItemId: item.ID, ItemType: tag, PackType: proto.ItemDetail_PackType_Inventory}
						bag[item.ID] = info
					}
					if info.Num > int64(^uint64(0)>>1)-item.Num {
						return nil, activityErr(proto.StatusCode_StatusCode_SYS_ERR, "奖励物品数量溢出:%d", item.ID)
					}
					info.Num += item.Num
					details = append(details, info.ItemDetail())
					continue
				}
				ctx := s.AddAllTypeItem(item.ID, item.Num)
				if ctx == nil {
					return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励物品无法发放:%d", item.ID)
				}
				detail := ctx.AddItemDetail()
				if detail == nil {
					return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励物品详情缺失:%d", item.ID)
				}
				details = append(details, detail)
			}
			return details, nil
		}
	}
	return granter(s, items)
}

func (a *activityService) resolveReward(s *model.Player, rewardID uint32) ([]activityItem, error) {
	if rewardID == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励池ID缺失")
	}
	level := a.playerLevel(s)
	items := make([]activityItem, 0)
	pool, ok := a.conf.RewardPools[rewardID]
	if !ok {
		itemPool, exists := a.conf.RewardItemPools[rewardID]
		if !exists {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励池配置缺失:%d", rewardID)
		}
		poolItems, err := a.resolveRewardItemPool(itemPool, level)
		if err != nil {
			return nil, err
		}
		return poolItems, nil
	}
	for _, group := range pool.Groups {
		itemPool, exists := a.conf.RewardItemPools[group.ID]
		if !exists {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励物品池缺失:%d", group.ID)
		}
		poolItems, err := a.resolveRewardItemPool(itemPool, level)
		if err != nil {
			return nil, err
		}
		items = append(items, poolItems...)
	}
	if len(items) == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励池为空:%d", rewardID)
	}
	return items, nil
}

func (a *activityService) resolveRewardItemPool(pool rewardItemPoolConfig, level uint32) ([]activityItem, error) {
	items := make([]activityItem, 0, len(pool.Groups))
	for _, group := range pool.Groups {
		if group.ItemID == 0 {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励池物品ID缺失")
		}
		if group.MinLevel > 0 && level < group.MinLevel {
			continue
		}
		if group.MaxLevel > 0 && level > group.MaxLevel {
			continue
		}
		minCount, maxCount := group.ItemMinCount, group.ItemMaxCount
		if minCount <= 0 {
			minCount = 1
		}
		if maxCount <= 0 {
			maxCount = minCount
		}
		if maxCount < minCount {
			minCount, maxCount = maxCount, minCount
		}
		count := minCount
		if maxCount > minCount {
			span := maxCount - minCount
			if span >= int64(^uint(0)>>1) {
				return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励数量范围无效")
			}
			count = minCount + int64(a.randomInt(int(span)+1))
		}
		items = append(items, activityItem{ID: group.ItemID, Num: count})
	}
	if len(items) == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "奖励物品池为空或不满足等级:%d", pool.ID)
	}
	return items, nil
}

func (a *activityService) directItems(items []activityDirectItem) ([]activityItem, error) {
	result := make([]activityItem, 0, len(items))
	for _, item := range items {
		if item.ItemID == 0 || item.ItemNum <= 0 {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "活动奖励条目无效")
		}
		result = append(result, activityItem{ID: item.ItemID, Num: item.ItemNum})
	}
	if len(result) == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "活动奖励条目为空")
	}
	return result, nil
}

func (c *activityConfigs) resolveActivity(id uint32, activityType proto.EActivityType, now int64) (activityConfigRow, proto.StatusCode) {
	if row, ok := c.Activities[id]; ok {
		if proto.EActivityType(row.NewActivityType) != activityType {
			return activityConfigRow{}, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		if !row.open(now) {
			return activityConfigRow{}, proto.StatusCode_StatusCode_Activity_Close
		}
		return row, proto.StatusCode_StatusCode_OK
	}
	matches := make([]activityConfigRow, 0, 1)
	closedTemplate := false
	for _, row := range c.Activities {
		if proto.EActivityType(row.NewActivityType) != activityType || row.templateID() != id {
			continue
		}
		closedTemplate = true
		if row.open(now) {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches[0], proto.StatusCode_StatusCode_OK
	}
	if len(matches) > 1 {
		return activityConfigRow{}, proto.StatusCode_StatusCode_BAD_REQ
	}
	if closedTemplate {
		return activityConfigRow{}, proto.StatusCode_StatusCode_Activity_Close
	}
	return activityConfigRow{}, proto.StatusCode_StatusCode_CFG_NOT_FOUND
}

func (r activityConfigRow) open(now int64) bool {
	if r.openUnix == 0 || now < r.openUnix {
		return false
	}
	if r.ClosingTime != "" && r.closeUnix == 0 {
		return false
	}
	return r.closeUnix == 0 || now <= r.closeUnix
}

func activityPeriodKey(row activityConfigRow) string {
	return fmt.Sprintf("%d:%d", row.ID, row.openUnix)
}

func activityOpenDay(row activityConfigRow, now int64) uint32 {
	if now < row.openUnix {
		return 0
	}
	return uint32((now-row.openUnix)/86400) + 1
}

func (a *activityService) period(s *model.Player, row activityConfigRow) *model.ActivityPeriodState {
	return s.GetActivityModel().GetPeriod(activityPeriodKey(row))
}

func (a *activityService) achievementTarget(id uint32) (uint64, bool) {
	conf, ok := a.conf.Achieves[id]
	if !ok || conf.CountParam == 0 {
		return 0, false
	}
	return conf.CountParam, true
}

func (a *activityService) achievementProgress(s *model.Player, row activityConfigRow, achieveID uint32, now int64) uint64 {
	scope := activityScopeLifetime
	for _, spec := range a.conf.AchievementSpecs[achieveID] {
		if spec.TemplateID == row.templateID() {
			scope = spec.Scope
			break
		}
	}
	if scope == activityScopeLifetime {
		return s.GetActivityModel().AchievementProgress(achieveID)
	}
	key := fmt.Sprintf("achieve:%d", achieveID)
	switch scope {
	case activityScopeDay:
		key += ":d:" + a.dayToken(now)
	case activityScopeWeek:
		key += ":w:" + a.weekToken(now)
	}
	return a.period(s, row).Counter(key)
}

func (a *activityService) requireAchievement(s *model.Player, row activityConfigRow, achieveID uint32, now int64) error {
	target, ok := a.achievementTarget(achieveID)
	if !ok {
		return activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "成就配置缺失:%d", achieveID)
	}
	if a.achievementProgress(s, row, achieveID, now) < target {
		return activityErr(proto.StatusCode_StatusCode_Quest_Not_Finish, "成就未完成:%d", achieveID)
	}
	return nil
}

func (a *activityService) signIn(s *model.Player, req *proto.ActivitySignInReq) (*proto.ActivitySignInRsp, []*proto.ItemDetail) {
	rsp := &proto.ActivitySignInRsp{Day: req.GetDay(), Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityId(), proto.EActivityType_EActivityType_SevenDaySign, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	conf, ok := a.conf.Signs[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	state := a.period(s, row)
	signState := state.GetSignIn()
	today := a.dayToken(now)
	loginDay := signState.LoginDay
	if signState.LastLoginDay != today {
		if signState.LastLoginDay == "" {
			loginDay = 1
		} else if signState.LoginDay < ^uint32(0) {
			loginDay++
		}
	}
	if req.GetDay() == 0 || req.GetDay() > 31 || req.GetDay() > loginDay {
		rsp.Status = proto.StatusCode_StatusCode_ChallengeNotStart
		return rsp, nil
	}
	var rewards []activityItem
	for _, day := range conf.Items {
		if day.Day != req.GetDay() {
			continue
		}
		var err error
		rewards, err = a.directItemsFromSign(day.Rewards)
		if err != nil {
			rsp.Status = ActivityErrorStatus(err)
			return rsp, nil
		}
		break
	}
	if len(rewards) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	for day := uint32(1); day < req.GetDay(); day++ {
		if !state.IsClaimed(fmt.Sprintf("sign:%d", day)) {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			return rsp, nil
		}
	}
	claimKey := fmt.Sprintf("sign:%d", req.GetDay())
	if state.IsClaimed(claimKey) {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.Claim(claimKey)
		if signState.InitTime == 0 {
			signState.InitTime = s.Created.Unix()
			if s.Created.IsZero() || signState.InitTime <= 0 {
				signState.InitTime = now
			}
		}
		signState.LoginDay = loginDay
		signState.LastLoginDay = today
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	for _, day := range conf.Items {
		if state.IsClaimed(fmt.Sprintf("sign:%d", day.Day)) {
			rsp.RewardId |= uint32(1) << day.Day
		}
	}
	return rsp, items
}

func (a *activityService) directItemsFromSign(items []activitySignReward) ([]activityItem, error) {
	result := make([]activityItem, 0, len(items))
	for _, item := range items {
		if item.ItemID == 0 || item.ItemNum <= 0 {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "签到奖励配置无效")
		}
		result = append(result, activityItem{ID: item.ItemID, Num: item.ItemNum})
	}
	if len(result) == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "签到奖励配置为空")
	}
	return result, nil
}

func (a *activityService) activityQuestReward(s *model.Player, req *proto.ActivityQuestRewardReq) (*proto.ActivityQuestRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.ActivityQuestRewardRsp{Param: req.GetParam(), IsProgress: req.GetIsProgress(), Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityId(), proto.EActivityType_EActivityType_SevenDayQuest, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	conf, ok := a.conf.Quests[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	state := a.period(s, row)
	if req.GetIsProgress() {
		rewardConf, ok := a.conf.QuestRewards[conf.RewardID]
		if !ok {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return rsp, nil
		}
		var selected *activityQuestProgressReward
		for i := range rewardConf.Items {
			if rewardConf.Items[i].Index == req.GetParam() {
				selected = &rewardConf.Items[i]
				break
			}
		}
		if selected == nil || selected.ConditionItemID == 0 || selected.ConditionItemNum <= 0 || selected.RewardItemID == 0 || selected.RewardItemNum <= 0 {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return rsp, nil
		}
		if _, ok := a.conf.Items[selected.ConditionItemID]; !ok {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return rsp, nil
		}
		have := s.GetItemModel().GetItemBaseInfo(selected.ConditionItemID)
		if have == nil || have.Num < selected.ConditionItemNum {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return rsp, nil
		}
		claimKey := fmt.Sprintf("quest:progress:%d", req.GetParam())
		if state.IsClaimed(claimKey) {
			rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
			return rsp, nil
		}
		rewards := []activityItem{{ID: selected.RewardItemID, Num: selected.RewardItemNum}}
		items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
			state.Claim(claimKey)
			return a.grantItems(s, rewards)
		})
		if err != nil {
			rsp.Status = ActivityErrorStatus(err)
			return rsp, nil
		}
		rsp.Rewards, rsp.RewardId = items, conf.RewardID
		return rsp, items
	}

	if req.GetParam() == 0 || req.GetParam() > activityOpenDay(row, now) {
		rsp.Status = proto.StatusCode_StatusCode_ChallengeNotStart
		return rsp, nil
	}
	dayConfigured := false
	allFinished := true
	claimable := make([]activityQuestAchieveConfig, 0)
	for _, item := range conf.AchieveItems {
		if item.Day != req.GetParam() {
			continue
		}
		dayConfigured = true
		if err := a.requireAchievement(s, row, item.AchieveID, now); err != nil {
			allFinished = false
			continue
		}
		if state.IsClaimed(fmt.Sprintf("quest:achieve:%d", item.AchieveID)) {
			continue
		}
		claimable = append(claimable, item)
	}
	if !dayConfigured {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if len(claimable) == 0 {
		if allFinished {
			rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		} else {
			rsp.Status = proto.StatusCode_StatusCode_Quest_Not_Finish
		}
		return rsp, nil
	}
	totals := make(map[uint32]int64)
	ids := make([]uint32, 0, len(totals))
	for _, item := range claimable {
		for _, reward := range item.Rewards {
			if reward.ItemID == 0 || reward.ItemNum <= 0 {
				rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
				return rsp, nil
			}
			if _, ok := totals[reward.ItemID]; !ok {
				ids = append(ids, reward.ItemID)
			}
			totals[reward.ItemID] += reward.ItemNum
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	rewards := make([]activityItem, 0, len(ids))
	for _, id := range ids {
		rewards = append(rewards, activityItem{ID: id, Num: totals[id]})
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		for _, item := range claimable {
			state.Claim(fmt.Sprintf("quest:achieve:%d", item.AchieveID))
		}
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) activityRegularReward(s *model.Player, req *proto.ActivityRegularRewardReq) (*proto.ActivityRegularRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.ActivityRegularRewardRsp{Param: req.GetParam(), PoolType: req.GetPoolType(), Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityId(), proto.EActivityType_EActivityType_ActivityRegular, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	conf, ok := a.conf.Regulars[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	var poolID uint32
	switch req.GetPoolType() {
	case 1:
		poolID = conf.DailyPoolID
	case 2:
		poolID = conf.RandomPoolID
	case 3:
		poolID = conf.PoolID
	case 4:
		poolID = conf.DailyRefreshPoolID
	default:
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return rsp, nil
	}
	pool, ok := a.conf.RegularPools[poolID]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	var selected *activityRegularAchieveConfig
	for i := range pool.Items {
		if pool.Items[i].AchieveID == req.GetParam() {
			selected = &pool.Items[i]
			break
		}
	}
	if selected == nil || selected.Days > activityOpenDay(row, now) {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if err := a.requireAchievement(s, row, selected.AchieveID, now); err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rewards, err := a.directItems(selected.Rewards)
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	state := a.period(s, row)
	claimKey := fmt.Sprintf("regular:%d:%d", req.GetPoolType(), req.GetParam())
	if req.GetPoolType() == 1 || req.GetPoolType() == 4 {
		claimKey += ":d:" + a.dayToken(now)
	}
	if state.IsClaimed(claimKey) {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.Claim(claimKey)
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) activityGiftReward(s *model.Player, req *proto.ActivityGiftRewardReq) (*proto.ActivityGiftRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.ActivityGiftRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityId(), proto.EActivityType_EActivityType_ActivityGift, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	conf, ok := a.conf.Gifts[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	var selected *activityGiftAchieveConfig
	for i := range conf.Items {
		if conf.Items[i].AchieveID == req.GetAchieveId() {
			selected = &conf.Items[i]
			break
		}
	}
	if selected == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if err := a.requireAchievement(s, row, selected.AchieveID, now); err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	state := a.period(s, row)
	if !state.GetGift().HasBuy {
		rsp.Status = proto.StatusCode_StatusCode_Not_Payment
		return rsp, nil
	}
	claimKey := fmt.Sprintf("gift:%d", req.GetAchieveId())
	if state.IsClaimed(claimKey) {
		rsp.Status = proto.StatusCode_StatusCode_ActivityGift_Rewarded
		return rsp, nil
	}
	rewards, err := a.directItems(selected.Rewards)
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.Claim(claimKey)
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) validateBattlePass(pass battlePassConfig) error {
	if pass.MaxExp == 0 || len(pass.Levels) == 0 {
		return activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令等级配置缺失")
	}
	for i, level := range pass.Levels {
		if level.Level != uint32(i+1) || level.Exp == 0 || len(level.Rewards) < 2 {
			return activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令等级奖励配置无效")
		}
	}
	return nil
}

func battlePassMaxTotal(pass battlePassConfig) uint32 {
	var total uint64
	for _, level := range pass.Levels {
		total += uint64(level.Exp)
	}
	if total > uint64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(total)
}

func battlePassLevelFromTotal(pass battlePassConfig, total uint32) (uint32, uint32) {
	remaining := total
	var level uint32
	for _, conf := range pass.Levels {
		if remaining < conf.Exp {
			break
		}
		remaining -= conf.Exp
		level++
	}
	return level, remaining
}

func (a *activityService) addBattlePassExp(s *model.Player, row activityConfigRow, pass battlePassConfig, add uint32, now int64) (*model.ActivityBattlePassState, error) {
	if add == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令经验配置缺失")
	}
	state := a.period(s, row).GetBattlePass()
	maxTotal := battlePassMaxTotal(pass)
	if state.TotalExp >= maxTotal {
		return nil, activityErr(proto.StatusCode_StatusCode_BattlePass_MaxLevel, "战令已满级")
	}
	next := uint64(state.TotalExp) + uint64(add)
	if next > uint64(maxTotal) {
		next = uint64(maxTotal)
	}
	state.TotalExp = uint32(next)
	state.Level, state.CurrentExp = battlePassLevelFromTotal(pass, state.TotalExp)
	week := a.weekToken(now)
	if state.WeekKey != week {
		state.WeekKey = week
		state.WeekExp = 0
	}
	if uint64(state.WeekExp)+uint64(add) <= uint64(^uint32(0)) {
		state.WeekExp += add
	}
	return state, nil
}

func (a *activityService) battlePassBuyExp(s *model.Player, req *proto.ActivityBattlePassBuyExpReq) (*proto.ActivityBattlePassBuyExpRsp, []*proto.ItemDetail) {
	rsp := &proto.ActivityBattlePassBuyExpRsp{ActivityBattlePassId: req.GetActivityBattlePassId(), Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityBattlePassId(), proto.EActivityType_EActivityType_BattlePass, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	pass, ok := a.conf.BattlePasses[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if err := a.validateBattlePass(pass); err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	currency, cost, exp, err := a.battlePassBuyExpConfig()
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	state := a.period(s, row).GetBattlePass()
	if state.TotalExp >= battlePassMaxTotal(pass) {
		rsp.Status = proto.StatusCode_StatusCode_BattlePass_MaxLevel
		return rsp, nil
	}
	var pack []*proto.ItemDetail
	_, err = a.commit(s, func() ([]*proto.ItemDetail, error) {
		notice, err := a.consume(s, currency, cost)
		if err != nil {
			return nil, err
		}
		pack = notice.Items
		if _, err := a.addBattlePassExp(s, row, pass, exp, now); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.BattlePassLv = state.Level
	rsp.TotalExp = state.TotalExp
	rsp.CurrentExp = state.CurrentExp
	return rsp, pack
}

func (a *activityService) battlePassBuyExpConfig() (uint32, int64, uint32, error) {
	values := a.conf.Globals["BattlePass_BuyExp_Currency"]
	if len(values) == 0 {
		return 0, 0, 0, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令购买经验配置缺失")
	}
	parts := strings.Split(values[0], ":")
	if len(parts) != 3 {
		return 0, 0, 0, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令购买经验配置无效")
	}
	currency, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	cost, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	exp, err3 := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 32)
	if err1 != nil || err2 != nil || err3 != nil || currency == 0 || cost <= 0 || exp == 0 {
		return 0, 0, 0, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令购买经验配置无效")
	}
	return uint32(currency), cost, uint32(exp), nil
}

func (a *activityService) battlePassLevelReward(s *model.Player, req *proto.GetActivityBattlePassLevelRewardReq) (*proto.GetActivityBattlePassLevelRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.GetActivityBattlePassLevelRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityBattlePassId(), proto.EActivityType_EActivityType_BattlePass, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	pass, ok := a.conf.BattlePasses[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if err := a.validateBattlePass(pass); err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	var level *battlePassLevelConfig
	for i := range pass.Levels {
		if pass.Levels[i].Level == req.GetLevel() {
			level = &pass.Levels[i]
			break
		}
	}
	if level == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	period := a.period(s, row)
	state := period.GetBattlePass()
	if state.Level < req.GetLevel() {
		rsp.Status = proto.StatusCode_StatusCode_BattlePass_LevelLimit
		return rsp, nil
	}
	index := 1
	kind := "paid"
	if req.GetIsGetFreeReward() {
		index = 0
		kind = "free"
	} else if !state.HasBuy {
		rsp.Status = proto.StatusCode_StatusCode_BattlePass_Not_Buy
		return rsp, nil
	}
	if index >= len(level.Rewards) {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	rewards, err := a.directItems(level.Rewards[index : index+1])
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	claimKey := fmt.Sprintf("bp:level:%d:%s", req.GetLevel(), kind)
	if period.IsClaimed(claimKey) {
		rsp.Status = proto.StatusCode_StatusCode_BattlePass_LvRewarded
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		period.Claim(claimKey)
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) battlePassQuestReward(s *model.Player, req *proto.GetActivityBattlePassQuestRewardReq) (*proto.GetActivityBattlePassQuestRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.GetActivityBattlePassQuestRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityBattlePassId(), proto.EActivityType_EActivityType_BattlePass, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	pass, ok := a.conf.BattlePasses[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if err := a.validateBattlePass(pass); err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	var exp uint32
	var scope string
	for _, group := range []struct {
		id    uint32
		scope string
	}{{pass.DayQuestID, activityScopeDay}, {pass.WeekQuestID, activityScopeWeek}, {pass.SeasonQuestID, activityScopePeriod}} {
		for _, item := range a.conf.BattlePassQuests[group.id].Items {
			if item.AchieveID == req.GetAchieveId() {
				if item.Exp == 0 {
					rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
					return rsp, nil
				}
				exp, scope = item.Exp, group.scope
				break
			}
		}
		if exp != 0 {
			break
		}
	}
	if exp == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if err := a.requireAchievement(s, row, req.GetAchieveId(), now); err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	state := a.period(s, row)
	claimKey := fmt.Sprintf("bp:quest:%d", req.GetAchieveId())
	switch scope {
	case activityScopeDay:
		claimKey += ":d:" + a.dayToken(now)
	case activityScopeWeek:
		claimKey += ":w:" + a.weekToken(now)
	}
	if state.IsClaimed(claimKey) {
		rsp.Status = proto.StatusCode_StatusCode_BattlePass_QuestRewarded
		return rsp, nil
	}
	_, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.Claim(claimKey)
		_, err := a.addBattlePassExp(s, row, pass, exp, now)
		return nil, err
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	bp := state.GetBattlePass()
	rsp.AddExp = exp
	rsp.CurrentLv = bp.Level
	rsp.CurrentWeekExp = bp.WeekExp
	rsp.CurrentTotalExp = bp.TotalExp
	rsp.CurrentExp = bp.CurrentExp
	return rsp, nil
}

func (a *activityService) monthCardReward(s *model.Player) (*proto.MonthCardRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.MonthCardRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	state := s.GetActivityModel().GetMonthCard()
	rsp.MonthCardRewardDays = state.RewardDays
	if state.OverDueTime == 0 {
		rsp.Status = proto.StatusCode_StatusCode_Not_Payment
		return rsp, nil
	}
	if now >= state.OverDueTime {
		rsp.Status = proto.StatusCode_StatusCode_FAIL
		return rsp, nil
	}
	day := a.dayToken(now)
	if state.LastClaimDay == day {
		rsp.Status = proto.StatusCode_StatusCode_MOON_CARD_DAYS_LIMIT
		return rsp, nil
	}
	values := a.conf.Globals["MoonCard_Rveryday_Currency"]
	if len(values) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	parts := strings.Split(values[0], ":")
	if len(parts) != 2 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	itemID, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	count, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err1 != nil || err2 != nil || itemID == 0 || count <= 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if state.RewardDays == ^uint32(0) {
		rsp.Status = proto.StatusCode_StatusCode_MOON_CARD_DAYS_LIMIT
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.LastClaimDay = day
		state.RewardDays++
		return a.grantItems(s, []activityItem{{ID: uint32(itemID), Num: count}})
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.MonthCardRewardDays = state.RewardDays
	return rsp, items
}

func (a *activityService) supplyBoxInterval() (int64, error) {
	values := a.conf.Globals["SupplyBox_Random_Time"]
	if len(values) < 2 {
		return 0, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱冷却配置缺失")
	}
	low, err1 := strconv.ParseInt(strings.TrimSpace(values[0]), 10, 64)
	high, err2 := strconv.ParseInt(strings.TrimSpace(values[1]), 10, 64)
	if err1 != nil || err2 != nil || low <= 0 || high <= 0 {
		return 0, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱冷却配置无效")
	}
	if low > high {
		low, high = high, low
	}
	span := high - low
	if span >= int64(^uint(0)>>1) || low > int64(^uint64(0)>>1)/60 {
		return 0, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱冷却配置范围无效")
	}
	return (low + int64(a.randomInt(int(span)+1))) * 60, nil
}

func (a *activityService) supplyBoxInfo(s *model.Player) (*proto.SupplyBoxInfoRsp, []*proto.ItemDetail) {
	rsp := &proto.SupplyBoxInfoRsp{Status: proto.StatusCode_StatusCode_OK}
	interval, err := a.supplyBoxInterval()
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	now := a.now()
	state := s.GetActivityModel().GetSupplyBox()
	if state.NextRewardTime == 0 {
		base := s.Created.Unix()
		if s.Created.IsZero() || base <= 0 {
			base = now
		}
		_, err = a.commit(s, func() ([]*proto.ItemDetail, error) {
			state.NextRewardTime = base + interval
			return nil, nil
		})
		if err != nil {
			rsp.Status = ActivityErrorStatus(err)
			return rsp, nil
		}
	}
	rsp.NextRewardTime = state.NextRewardTime
	return rsp, nil
}

func (a *activityService) supplyBoxRewards(s *model.Player) ([]activityItem, error) {
	if values := a.conf.Globals["SupplyBox_RewardID"]; len(values) > 0 {
		id, err := strconv.ParseUint(strings.TrimSpace(values[0]), 10, 32)
		if err != nil || id == 0 {
			return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱奖励池配置无效")
		}
		return a.resolveReward(s, uint32(id))
	}
	values := a.conf.Globals["SupplyBox_Reward"]
	if len(values) == 0 {
		values = a.conf.Globals["SupplyBox_Rewards"]
	}
	if len(values) == 0 {
		return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱奖励配置缺失")
	}
	items := make([]activityItem, 0)
	for _, value := range values {
		for _, entry := range strings.Split(value, ",") {
			parts := strings.Split(strings.TrimSpace(entry), ":")
			if len(parts) != 2 {
				return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱奖励配置无效")
			}
			id, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
			count, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err1 != nil || err2 != nil || id == 0 || count <= 0 {
				return nil, activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "补给箱奖励配置无效")
			}
			items = append(items, activityItem{ID: uint32(id), Num: count})
		}
	}
	return items, nil
}

func (a *activityService) supplyBoxReward(s *model.Player) (*proto.SupplyBoxRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.SupplyBoxRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	interval, err := a.supplyBoxInterval()
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rewards, err := a.supplyBoxRewards(s)
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	now := a.now()
	state := s.GetActivityModel().GetSupplyBox()
	if state.NextRewardTime != 0 && now < state.NextRewardTime {
		rsp.Status = proto.StatusCode_StatusCode_Interval_Reward_Is_Get
		rsp.NextRewardTime = state.NextRewardTime
		return rsp, nil
	}
	base := state.NextRewardTime
	if base == 0 {
		base = s.Created.Unix()
		if s.Created.IsZero() || base <= 0 {
			base = now
		}
		if now < base {
			rsp.Status = proto.StatusCode_StatusCode_Interval_Reward_Is_Get
			rsp.NextRewardTime = base + interval
			return rsp, nil
		}
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.NextRewardTime = now + interval
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.NextRewardTime = state.NextRewardTime
	rsp.Items = items
	return rsp, items
}

func (a *activityService) activityTurntableDraw(s *model.Player, req *proto.ActivityTurntableDrawReq) (*proto.ActivityTurntableDrawRsp, []*proto.ItemDetail) {
	rsp := &proto.ActivityTurntableDrawRsp{ActivityId: req.GetActivityId(), Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityId(), proto.EActivityType_EActivityType_ActivityTurntable, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp, nil
	}
	conf, ok := a.conf.Turntables[row.templateID()]
	if !ok || conf.CostItemID == 0 || conf.FirstOneNeedItemNum <= 0 || conf.OneNeedItemNum <= 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	pool, ok := a.conf.TurntablePools[conf.PoolID]
	if !ok || len(pool.Items) == 0 || req.GetDrawTimes() == 0 || req.GetDrawTimes() > 10 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		if req.GetDrawTimes() > 10 {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		}
		return rsp, nil
	}
	seen := make(map[uint32]bool)
	for _, item := range pool.Items {
		if item.Index == 0 || seen[item.Index] || item.ItemID == 0 || item.ItemNum <= 0 || item.Weight == 0 {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return rsp, nil
		}
		seen[item.Index] = true
	}
	state := a.period(s, row).GetTurntable()
	used := make(map[uint32]uint32, len(state.GetUsed()))
	for index, count := range state.GetUsed() {
		used[index] = count
	}
	plan := make([]activityTurntablePoolItem, 0, req.GetDrawTimes())
	for i := uint32(0); i < req.GetDrawTimes(); i++ {
		available := make([]activityTurntablePoolItem, 0, len(pool.Items))
		var totalWeight uint64
		for _, item := range pool.Items {
			if item.LimitNum > 0 && used[item.Index] >= item.LimitNum {
				continue
			}
			available = append(available, item)
			totalWeight += uint64(item.Weight)
		}
		if len(available) == 0 {
			break
		}
		if totalWeight == 0 || totalWeight >= uint64(^uint(0)>>1) {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return rsp, nil
		}
		roll := uint64(a.randomInt(int(totalWeight)))
		var selected *activityTurntablePoolItem
		for j := range available {
			weight := uint64(available[j].Weight)
			if roll < weight {
				selected = &available[j]
				break
			}
			roll -= weight
		}
		if selected == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return rsp, nil
		}
		used[selected.Index]++
		plan = append(plan, *selected)
	}
	if len(plan) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_TURNTABLE_NO_REWARD_REMAIN
		return rsp, nil
	}
	cost := conf.OneNeedItemNum * int64(len(plan))
	if state.DrawCount == 0 {
		cost += conf.FirstOneNeedItemNum - conf.OneNeedItemNum
	}
	rewards := make([]activityItem, 0, len(plan))
	for _, item := range plan {
		rsp.RewardIndexes = append(rsp.RewardIndexes, item.Index)
		rewards = append(rewards, activityItem{ID: item.ItemID, Num: item.ItemNum})
	}
	var pack []*proto.ItemDetail
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		notice, err := a.consume(s, conf.CostItemID, cost)
		if err != nil {
			return nil, err
		}
		pack = append(pack, notice.Items...)
		items, err := a.grantItems(s, rewards)
		if err != nil {
			return nil, err
		}
		state.DrawCount += uint32(len(plan))
		for _, item := range plan {
			state.GetUsed()[item.Index]++
		}
		for i := len(plan) - 1; i >= 0; i-- {
			item := plan[i]
			record := model.ActivityTurntableRecord{Index: item.Index, ItemID: item.ItemID, ItemNum: uint32(item.ItemNum), DrawTime: now}
			state.Records = append([]model.ActivityTurntableRecord{record}, state.Records...)
		}
		pack = append(pack, items...)
		return items, nil
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, pack
}

func (a *activityService) activityTurntableRecord(s *model.Player, req *proto.ActivityTurntableRecordReq) *proto.ActivityTurntableRecordRsp {
	rsp := &proto.ActivityTurntableRecordRsp{ActivityId: req.GetActivityId(), Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	row, status := a.conf.resolveActivity(req.GetActivityId(), proto.EActivityType_EActivityType_ActivityTurntable, now)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return rsp
	}
	conf, ok := a.conf.Turntables[row.templateID()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp
	}
	if _, ok := a.conf.TurntablePools[conf.PoolID]; !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp
	}
	page := req.GetPage()
	if page == 0 {
		page = 1
	}
	rsp.Page = page
	records := a.period(s, row).GetTurntable().Records
	rsp.TotalPage = uint32((len(records) + 9) / 10)
	offset := (uint64(page) - 1) * 10
	if offset >= uint64(len(records)) {
		return rsp
	}
	start := int(offset)
	end := start + 10
	if end > len(records) {
		end = len(records)
	}
	for _, record := range records[start:end] {
		rsp.Records = append(rsp.Records, &proto.TurntableDrawRecord{
			Index: record.Index, ItemId: record.ItemID, ItemNum: record.ItemNum, DrawTime: record.DrawTime,
		})
	}
	return rsp
}

func (a *activityService) abyssQuestReward(s *model.Player, req *proto.AbyssQuestRewardReq) (*proto.AbyssQuestRewardRsp, []*proto.ItemDetail) {
	rsp := &proto.AbyssQuestRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	now := a.now()
	season, ok := a.conf.AbyssSeasons[req.GetSeasonId()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if season.openUnix == 0 || now < season.openUnix || (season.EndTime != "" && season.closeUnix == 0) || (season.closeUnix != 0 && now > season.closeUnix) {
		rsp.Status = proto.StatusCode_StatusCode_Activity_Close
		return rsp, nil
	}
	achieves, ok := a.conf.AbyssAchieves[season.SeasonAchieve]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	var rewardID uint32
	for _, item := range achieves.Items {
		if item.AchieveID == req.GetConditionId() {
			rewardID = item.RewardID
			break
		}
	}
	if rewardID == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	target, ok := a.achievementTarget(req.GetConditionId())
	if !ok || s.GetActivityModel().AchievementProgress(req.GetConditionId()) < target {
		rsp.Status = proto.StatusCode_StatusCode_Quest_Not_Finish
		return rsp, nil
	}
	rewards, err := a.resolveReward(s, rewardID)
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	row := activityConfigRow{ID: req.GetSeasonId(), openUnix: season.openUnix}
	state := a.period(s, row)
	claimKey := fmt.Sprintf("abyss:%d", req.GetConditionId())
	if state.IsClaimed(claimKey) {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.Claim(claimKey)
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) getOneAchieveAward(s *model.Player, req *proto.GetOneAchieveAwardReq) (*proto.GetOneAchieveAwardRsp, []*proto.ItemDetail) {
	rsp := &proto.GetOneAchieveAwardRsp{Status: proto.StatusCode_StatusCode_OK}
	target, ok := a.achievementTarget(req.GetAchieveId())
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	if s.GetActivityModel().AchievementProgress(req.GetAchieveId()) < target {
		rsp.Status = proto.StatusCode_StatusCode_Quest_Not_Finish
		return rsp, nil
	}
	rewardID, ok := a.conf.AchieveRewards[req.GetAchieveId()]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	rewards, err := a.resolveReward(s, rewardID)
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	if s.GetActivityModel().GetRewardedAchievements()[req.GetAchieveId()] {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		s.GetActivityModel().GetRewardedAchievements()[req.GetAchieveId()] = true
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) getAchieveGroupAward(s *model.Player, req *proto.GetAchieveGroupAwardReq) (*proto.GetAchieveGroupAwardRsp, []*proto.ItemDetail) {
	rsp := &proto.GetAchieveGroupAwardRsp{Status: proto.StatusCode_StatusCode_OK}
	quest, ok := a.conf.AchieveQuestByGroup[req.GetGroupId()]
	if !ok || quest.RewardID == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	group, ok := a.conf.AchieveQuestGroups[req.GetGroupId()]
	if !ok || len(group.Items) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp, nil
	}
	for _, item := range group.Items {
		target, ok := a.achievementTarget(item.AchieveID)
		if !ok || s.GetActivityModel().AchievementProgress(item.AchieveID) < target {
			rsp.Status = proto.StatusCode_StatusCode_Quest_Not_Finish
			return rsp, nil
		}
	}
	rewards, err := a.resolveReward(s, quest.RewardID)
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	if s.GetActivityModel().GetRewardedAchieveGroups()[req.GetGroupId()] {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return rsp, nil
	}
	items, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		s.GetActivityModel().GetRewardedAchieveGroups()[req.GetGroupId()] = true
		return a.grantItems(s, rewards)
	})
	if err != nil {
		rsp.Status = ActivityErrorStatus(err)
		return rsp, nil
	}
	rsp.Rewards = items
	return rsp, items
}

func (a *activityService) recordAchievement(s *model.Player, achieveID uint32, delta uint64, now int64) error {
	if delta == 0 {
		return nil
	}
	if _, ok := a.conf.Achieves[achieveID]; !ok {
		return activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "成就配置缺失:%d", achieveID)
	}
	specs := a.conf.AchievementSpecs[achieveID]
	if len(specs) == 0 {
		s.GetActivityModel().AddAchievementProgress(achieveID, delta)
		return nil
	}
	recorded := false
	for _, spec := range specs {
		for _, candidate := range a.conf.Activities {
			if candidate.templateID() != spec.TemplateID || !candidate.open(now) {
				continue
			}
			key := fmt.Sprintf("achieve:%d", achieveID)
			switch spec.Scope {
			case activityScopeDay:
				key += ":d:" + a.dayToken(now)
			case activityScopeWeek:
				key += ":w:" + a.weekToken(now)
			}
			a.period(s, candidate).AddCounter(key, delta)
			recorded = true
		}
	}
	if !recorded {
		return activityErr(proto.StatusCode_StatusCode_Activity_Close, "成就所属活动未开放")
	}
	return nil
}

func (a *activityService) touchActivityLogin(s *model.Player, now int64) error {
	today := a.dayToken(now)
	changed := false
	for _, row := range a.conf.Activities {
		if proto.EActivityType(row.NewActivityType) != proto.EActivityType_EActivityType_SevenDaySign || !row.open(now) {
			continue
		}
		state := a.period(s, row).GetSignIn()
		if state.LastLoginDay != today {
			changed = true
			break
		}
	}
	if !changed {
		return nil
	}
	_, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		for _, row := range a.conf.Activities {
			if proto.EActivityType(row.NewActivityType) != proto.EActivityType_EActivityType_SevenDaySign || !row.open(now) {
				continue
			}
			state := a.period(s, row).GetSignIn()
			if state.LastLoginDay == today {
				continue
			}
			if state.LastLoginDay == "" {
				state.LoginDay = 1
			} else if state.LoginDay < ^uint32(0) {
				state.LoginDay++
			}
			state.LastLoginDay = today
			if state.InitTime == 0 {
				state.InitTime = s.Created.Unix()
				if s.Created.IsZero() || state.InitTime <= 0 {
					state.InitTime = now
				}
			}
		}
		return nil, nil
	})
	return err
}

func TouchActivityLogin(s *model.Player) error {
	return defaultActivityService().touchActivityLogin(s, time.Now().Unix())
}

func RecordAchieveProgress(s *model.Player, achieveID uint32, delta uint64) error {
	return defaultActivityService().recordAchievement(s, achieveID, delta, time.Now().Unix())
}

func RecordAchieveCondition(s *model.Player, condition proto.EAchieveCondition, delta uint64) error {
	service := defaultActivityService()
	now := time.Now().Unix()
	var errs []error
	for _, achieveID := range service.conf.AchievesByCondition[int32(condition)] {
		if err := service.recordAchievement(s, achieveID, delta, now); err != nil {
			var statusErr *activityStatusError
			if errors.As(err, &statusErr) && statusErr.status == proto.StatusCode_StatusCode_Activity_Close {
				continue
			}
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *activityService) recordGiftPurchased(s *model.Player, activityID uint32, now int64) error {
	row, status := a.conf.resolveActivity(activityID, proto.EActivityType_EActivityType_ActivityGift, now)
	if status != proto.StatusCode_StatusCode_OK {
		return activityErr(status, "活动未开放")
	}
	if _, ok := a.conf.Gifts[row.templateID()]; !ok {
		return activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "活动礼物配置缺失")
	}
	state := a.period(s, row).GetGift()
	if state.HasBuy {
		return activityErr(proto.StatusCode_StatusCode_ActivityGift_HasBuy, "活动礼物已购买")
	}
	_, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.HasBuy = true
		return nil, nil
	})
	return err
}

func (a *activityService) recordBattlePassPurchased(s *model.Player, activityID uint32, now int64) error {
	row, status := a.conf.resolveActivity(activityID, proto.EActivityType_EActivityType_BattlePass, now)
	if status != proto.StatusCode_StatusCode_OK {
		return activityErr(status, "活动未开放")
	}
	pass, ok := a.conf.BattlePasses[row.templateID()]
	if !ok {
		return activityErr(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "战令配置缺失")
	}
	if err := a.validateBattlePass(pass); err != nil {
		return err
	}
	state := a.period(s, row).GetBattlePass()
	if state.HasBuy {
		return activityErr(proto.StatusCode_StatusCode_BattlePass_HasBuy, "战令已购买")
	}
	_, err := a.commit(s, func() ([]*proto.ItemDetail, error) {
		state.HasBuy = true
		return nil, nil
	})
	return err
}

func (a *activityService) activityIDForShop(shopID uint32, activityType proto.EActivityType, now int64) (uint32, error) {
	var selected uint32
	for _, row := range a.conf.Activities {
		if proto.EActivityType(row.NewActivityType) != activityType || !row.open(now) {
			continue
		}
		var configuredShop uint32
		switch activityType {
		case proto.EActivityType_EActivityType_ActivityGift:
			configuredShop = a.conf.Gifts[row.templateID()].ShopID
		case proto.EActivityType_EActivityType_BattlePass:
			configuredShop = a.conf.BattlePasses[row.templateID()].ShopID
		}
		if configuredShop != shopID {
			continue
		}
		if selected != 0 {
			return 0, activityErr(proto.StatusCode_StatusCode_BAD_REQ, "商店对应多个开放活动")
		}
		selected = row.ID
	}
	if selected == 0 {
		return 0, activityErr(proto.StatusCode_StatusCode_Activity_Close, "商店对应活动未开放")
	}
	return selected, nil
}

func RecordActivityGiftShopPurchased(s *model.Player, shopID uint32) error {
	service := defaultActivityService()
	activityID, err := service.activityIDForShop(shopID, proto.EActivityType_EActivityType_ActivityGift, time.Now().Unix())
	if err != nil {
		return err
	}
	return service.recordGiftPurchased(s, activityID, time.Now().Unix())
}

func RecordBattlePassShopPurchased(s *model.Player, shopID uint32) error {
	service := defaultActivityService()
	activityID, err := service.activityIDForShop(shopID, proto.EActivityType_EActivityType_BattlePass, time.Now().Unix())
	if err != nil {
		return err
	}
	return service.recordBattlePassPurchased(s, activityID, time.Now().Unix())
}

func RecordActivityGiftPurchased(s *model.Player, activityID uint32) error {
	return defaultActivityService().recordGiftPurchased(s, activityID, time.Now().Unix())
}

func RecordBattlePassPurchased(s *model.Player, activityID uint32) error {
	return defaultActivityService().recordBattlePassPurchased(s, activityID, time.Now().Unix())
}

func ActivateMonthCard(s *model.Player, overDueTime int64) error {
	if overDueTime <= time.Now().Unix() {
		return activityErr(proto.StatusCode_StatusCode_BAD_REQ, "月卡到期时间无效")
	}
	service := defaultActivityService()
	_, err := service.commit(s, func() ([]*proto.ItemDetail, error) {
		s.GetActivityModel().GetMonthCard().OverDueTime = overDueTime
		return nil, nil
	})
	return err
}

func (a *activityService) activeAbyssSeason(now int64) (abyssSeasonConfig, bool) {
	var selected abyssSeasonConfig
	found := false
	for _, season := range a.conf.AbyssSeasons {
		if season.openUnix == 0 || now < season.openUnix || (season.EndTime != "" && season.closeUnix == 0) || (season.closeUnix != 0 && now > season.closeUnix) {
			continue
		}
		if !found || season.openUnix > selected.openUnix || (season.openUnix == selected.openUnix && season.ID > selected.ID) {
			selected = season
			found = true
		}
	}
	return selected, found
}

func (a *activityService) abyssInfo(s *model.Player) *proto.AbyssInfoRsp {
	rsp := &proto.AbyssInfoRsp{Status: proto.StatusCode_StatusCode_OK, AbyssInfo: &proto.AbyssInfo{PlayerId: s.UserId, SeasonInfo: make(map[uint32]*proto.AbyssSeasonInfo)}}
	season, ok := a.activeAbyssSeason(a.now())
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_Activity_Close
		return rsp
	}
	rsp.InProgressSeasonId = season.ID
	info := &proto.AbyssSeasonInfo{SeasonId: season.ID, DungeonInfo: make(map[uint32]*proto.AbyssDungeonInfo), DungeonInfo_0: make(map[uint32]*proto.AbyssDungeonInfo)}
	stage, ok := a.conf.AbyssStages[season.SeasonStage]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return rsp
	}
	dungeons := s.GetDungeonModel().GetDungeonMap()
	for _, group := range stage.Items {
		if group.DungeonID == 0 {
			continue
		}
		dungeonInfo := &proto.AbyssDungeonInfo{DungeonId: group.DungeonID, IsChallenge: group.Challenge}
		if dungeon := dungeons[group.DungeonID]; dungeon != nil {
			dungeonInfo.FinishTime = uint32(dungeon.LastFinishTime)
		}
		if group.Challenge {
			info.DungeonInfo_0[group.DungeonID] = dungeonInfo
		} else {
			info.DungeonInfo[group.DungeonID] = dungeonInfo
		}
	}
	rsp.AbyssInfo.SeasonInfo[season.ID] = info
	return rsp
}

func (a *activityService) monthCardNotice(s *model.Player) *proto.MonthCardNotice {
	state := s.GetActivityModel().GetMonthCard()
	return &proto.MonthCardNotice{
		Status:        proto.StatusCode_StatusCode_OK,
		OverDueTime:   state.OverDueTime,
		IsRewardToday: state.LastClaimDay == a.dayToken(a.now()),
	}
}

func (g *Game) MonthCardNotice(s *model.Player) {
	g.send(s, 0, defaultActivityService().monthCardNotice(s))
}

func (g *Game) sendActivityItems(s *model.Player, items []*proto.ItemDetail) {
	if len(items) > 0 {
		g.send(s, 0, &proto.PackNotice{Status: proto.StatusCode_StatusCode_OK, Items: items})
	}
}

func (g *Game) ActivitySignIn(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().signIn(s, msg.Body.(*proto.ActivitySignInReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivityQuestReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().activityQuestReward(s, msg.Body.(*proto.ActivityQuestRewardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivityRegularReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().activityRegularReward(s, msg.Body.(*proto.ActivityRegularRewardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivityGiftReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().activityGiftReward(s, msg.Body.(*proto.ActivityGiftRewardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivityBattlePassBuyExp(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().battlePassBuyExp(s, msg.Body.(*proto.ActivityBattlePassBuyExpReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetActivityBattlePassLevelReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().battlePassLevelReward(s, msg.Body.(*proto.GetActivityBattlePassLevelRewardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetActivityBattlePassQuestReward(s *model.Player, msg *alg.GameMsg) {
	rsp, _ := defaultActivityService().battlePassQuestReward(s, msg.Body.(*proto.GetActivityBattlePassQuestRewardReq))
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) MonthCardReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().monthCardReward(s)
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivityTurntableDraw(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().activityTurntableDraw(s, msg.Body.(*proto.ActivityTurntableDrawReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivityTurntableRecord(s *model.Player, msg *alg.GameMsg) {
	rsp := defaultActivityService().activityTurntableRecord(s, msg.Body.(*proto.ActivityTurntableRecordReq))
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) AbyssQuestReward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().abyssQuestReward(s, msg.Body.(*proto.AbyssQuestRewardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetAchieveGroupAward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().getAchieveGroupAward(s, msg.Body.(*proto.GetAchieveGroupAwardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetOneAchieveAward(s *model.Player, msg *alg.GameMsg) {
	rsp, items := defaultActivityService().getOneAchieveAward(s, msg.Body.(*proto.GetOneAchieveAwardReq))
	defer g.sendActivityItems(s, items)
	defer g.send(s, msg.PacketId, rsp)
}
