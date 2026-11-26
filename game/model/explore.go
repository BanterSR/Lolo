package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	appconfig "gucooing/lolo/config"
	"gucooing/lolo/protocol/proto"
)

const (
	exploreArchiveKey = "explore_state_v1"
	exploreTimeLayout = "2006-01-02 15:04:05"
)

// AddDropItems allocates a server-side drop index and transfers ownership of
// the supplied objects to the scene. Callers must not retain another owner.
func (d *SceneTempPack) AddDropItems(items []EBagItemTag) *DropItem {
	if d.TempAreaPack == nil {
		d.TempAreaPack = make(map[uint32]*DropItem)
	}
	drop := &DropItem{Index: d.index.Add(1), Items: append([]EBagItemTag(nil), items...)}
	d.TempAreaPack[drop.Index] = drop
	return drop
}

// ExploreDefinition is the validated, server-side view of one Explore.json row.
// RewardItemIDs intentionally preserves duplicate rows: each configured row is
// one grant, so no quantity is invented when the source table has no count.
type ExploreDefinition struct {
	ID              uint32
	NameID          uint32
	IconID          uint32
	NeedTime        time.Duration
	QuickItemID     uint32
	QuickItemCount  int64
	RewardItemIDs   []uint32
	NeedSceneID     uint32
	NeedPlayerLevel uint32
	NeedItemID      uint32
	NeedItemCount   int64
	Difficulty      uint32
	MaxTimes        uint32
	StartTime       time.Time
	EndTime         time.Time
	SpecialItemID   uint32
	IsActivity      bool
}

type ExploreIntervalDefinition struct {
	ID             uint32
	NameID         uint32
	IconID         uint32
	RetentionTime  time.Duration
	Time           time.Duration
	NeedItemID     uint32
	CaptainCount   uint32
	MemberCount    uint32
	QuickItemID    uint32
	QuickItemCount int64
}

type ExploreConfig struct {
	areas         map[uint32]ExploreDefinition
	activities    map[uint32]ExploreDefinition
	intervals     map[uint32]ExploreIntervalDefinition
	MaxConcurrent uint32
}

func (c *ExploreConfig) Area(id uint32) (ExploreDefinition, bool) {
	v, ok := c.areas[id]
	return v, ok
}

func (c *ExploreConfig) Activity(id uint32) (ExploreDefinition, bool) {
	v, ok := c.activities[id]
	return v, ok
}

func (c *ExploreConfig) Interval(id uint32) (ExploreIntervalDefinition, bool) {
	v, ok := c.intervals[id]
	return v, ok
}

func (c *ExploreConfig) AreaIDs() []uint32 {
	ids := make([]uint32, 0, len(c.areas))
	for id := range c.areas {
		ids = append(ids, id)
	}
	sortUint32s(ids)
	return ids
}

func (c *ExploreConfig) ActivityIDs() []uint32 {
	ids := make([]uint32, 0, len(c.activities))
	for id := range c.activities {
		ids = append(ids, id)
	}
	sortUint32s(ids)
	return ids
}

func sortUint32s(values []uint32) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

type exploreConfigFile struct {
	ExploreArea struct {
		Datas []exploreAreaRow `json:"datas"`
	} `json:"explore_area"`
	ExploreActivity struct {
		Datas []exploreActivityRow `json:"datas"`
	} `json:"explore_activity"`
	ExploreInterval struct {
		Datas []exploreIntervalRow `json:"datas"`
	} `json:"explore_interval"`
}

type exploreRewardRow struct {
	ItemID int32 `json:"rewar_item_i_d"`
}

type exploreAreaRow struct {
	ID              int32              `json:"i_d"`
	NameID          int32              `json:"name_i_d"`
	IconID          int32              `json:"icon_i_d"`
	NeedTime        int32              `json:"need_time"`
	QuickItem       int32              `json:"quick_need_item"`
	QuickItemCount  int32              `json:"quick_need_item_count"`
	NeedSceneID     int32              `json:"need_scene_i_d"`
	NeedPlayerLevel int32              `json:"need_player_level"`
	NeedItem        int32              `json:"need_item"`
	NeedItemCount   int32              `json:"need_item_count"`
	Items           []exploreRewardRow `json:"explore_area_item"`
	Difficulty      int32              `json:"new_explore_difficulty"`
}

type exploreActivityRow struct {
	ID          int32              `json:"i_d"`
	NameID      int32              `json:"name_i_d"`
	IconID      int32              `json:"icon_i_d"`
	StartTime   string             `json:"start_time"`
	EndTime     string             `json:"end_time"`
	NeedTime    int32              `json:"need_time"`
	MaxTimes    int32              `json:"max_times"`
	QuickItem   int32              `json:"quick_need_item"`
	QuickCount  int32              `json:"quick_need_item_count"`
	SpecialItem int32              `json:"special_reward_item_i_d"`
	Items       []exploreRewardRow `json:"explore_activity_item"`
}

type exploreIntervalRow struct {
	ID         int32 `json:"i_d"`
	NameID     int32 `json:"name_i_d"`
	IconID     int32 `json:"icon_i_d"`
	Retention  int32 `json:"retention_time"`
	Time       int32 `json:"time"`
	NeedItem   int32 `json:"need_item_i_d"`
	Captain    int32 `json:"captain_count"`
	Member     int32 `json:"member_count"`
	QuickItem  int32 `json:"quick_need_item"`
	QuickCount int32 `json:"quick_need_item_count"`
}

type globalConfigFile struct {
	Config struct {
		Datas []struct {
			Key    string   `json:"key"`
			Values []string `json:"values"`
		} `json:"datas"`
	} `json:"config"`
}

// LoadExploreConfig reads the same exported files as gdconf without changing
// the shared loader. Structural errors are returned instead of manufacturing
// default explore rows, durations, costs, or rewards.
func LoadExploreConfig() (*ExploreConfig, error) {
	resources := appconfig.GetConfig().Resources
	if resources == nil || resources.ResourcePath == "" {
		return nil, errors.New("resource path is not configured")
	}
	excelPath := filepath.Join(resources.ResourcePath, "Excel")
	var raw exploreConfigFile
	if err := readJSONFile(filepath.Join(excelPath, "Explore.json"), &raw); err != nil {
		return nil, err
	}
	var global globalConfigFile
	if err := readJSONFile(filepath.Join(excelPath, "GlobalConfig.json"), &global); err != nil {
		return nil, err
	}

	cfg := &ExploreConfig{
		areas:      make(map[uint32]ExploreDefinition),
		activities: make(map[uint32]ExploreDefinition),
		intervals:  make(map[uint32]ExploreIntervalDefinition),
	}
	for _, row := range global.Config.Datas {
		if row.Key != "Explore_ProcessIng_MaxCount" {
			continue
		}
		if len(row.Values) == 0 {
			return nil, errors.New("Explore_ProcessIng_MaxCount has no value")
		}
		value, err := strconv.ParseUint(row.Values[0], 10, 32)
		if err != nil || value == 0 {
			return nil, errors.New("Explore_ProcessIng_MaxCount must be positive")
		}
		cfg.MaxConcurrent = uint32(value)
	}
	if cfg.MaxConcurrent == 0 {
		return nil, errors.New("Explore_ProcessIng_MaxCount is missing")
	}

	for _, row := range raw.ExploreArea.Datas {
		def, err := validateAreaRow(row)
		if err != nil {
			return nil, err
		}
		if _, exists := cfg.areas[def.ID]; exists {
			return nil, fmt.Errorf("duplicate explore area %d", def.ID)
		}
		cfg.areas[def.ID] = def
	}
	for _, row := range raw.ExploreActivity.Datas {
		def, err := validateActivityRow(row)
		if err != nil {
			return nil, err
		}
		if _, exists := cfg.activities[def.ID]; exists {
			return nil, fmt.Errorf("duplicate activity explore %d", def.ID)
		}
		cfg.activities[def.ID] = def
	}
	for _, row := range raw.ExploreInterval.Datas {
		if row.ID <= 0 || row.Retention <= 0 || row.Time <= 0 || row.NeedItem <= 0 ||
			row.Captain <= 0 || row.Member <= 0 || !validCostPair(row.QuickItem, row.QuickCount) {
			return nil, fmt.Errorf("invalid explore interval %d", row.ID)
		}
		id := uint32(row.ID)
		if _, exists := cfg.intervals[id]; exists {
			return nil, fmt.Errorf("duplicate explore interval %d", id)
		}
		cfg.intervals[id] = ExploreIntervalDefinition{
			ID: id, NameID: uint32(row.NameID), IconID: uint32(row.IconID),
			RetentionTime: time.Duration(row.Retention) * time.Second,
			Time:          time.Duration(row.Time) * time.Second, NeedItemID: uint32(row.NeedItem),
			CaptainCount: uint32(row.Captain), MemberCount: uint32(row.Member),
			QuickItemID: uint32(row.QuickItem), QuickItemCount: int64(row.QuickCount),
		}
	}
	return cfg, nil
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return nil
}

func validCostPair(id, count int32) bool {
	return (id == 0 && count == 0) || (id > 0 && count > 0)
}

func rewardIDs(rows []exploreRewardRow) ([]uint32, error) {
	ids := make([]uint32, 0, len(rows))
	for _, row := range rows {
		if row.ItemID <= 0 {
			return nil, errors.New("explore reward item id must be positive")
		}
		ids = append(ids, uint32(row.ItemID))
	}
	return ids, nil
}

func validateAreaRow(row exploreAreaRow) (ExploreDefinition, error) {
	if row.ID <= 0 || row.NeedTime <= 0 || row.NeedSceneID <= 0 || row.NeedPlayerLevel < 0 ||
		!validCostPair(row.QuickItem, row.QuickItemCount) || !validCostPair(row.NeedItem, row.NeedItemCount) {
		return ExploreDefinition{}, fmt.Errorf("invalid explore area %d", row.ID)
	}
	ids, err := rewardIDs(row.Items)
	if err != nil || len(ids) == 0 {
		return ExploreDefinition{}, fmt.Errorf("explore area %d has no valid rewards", row.ID)
	}
	return ExploreDefinition{
		ID: uint32(row.ID), NameID: uint32(row.NameID), IconID: uint32(row.IconID),
		NeedTime:    time.Duration(row.NeedTime) * time.Second,
		QuickItemID: uint32(row.QuickItem), QuickItemCount: int64(row.QuickItemCount),
		RewardItemIDs: ids, NeedSceneID: uint32(row.NeedSceneID),
		NeedPlayerLevel: uint32(row.NeedPlayerLevel), NeedItemID: uint32(row.NeedItem),
		NeedItemCount: int64(row.NeedItemCount), Difficulty: uint32(row.Difficulty),
	}, nil
}

func validateActivityRow(row exploreActivityRow) (ExploreDefinition, error) {
	if row.ID <= 0 || row.NeedTime <= 0 || row.MaxTimes <= 0 || row.SpecialItem < 0 ||
		!validCostPair(row.QuickItem, row.QuickCount) {
		return ExploreDefinition{}, fmt.Errorf("invalid activity explore %d", row.ID)
	}
	start, err := time.ParseInLocation(exploreTimeLayout, row.StartTime, time.Local)
	if err != nil {
		return ExploreDefinition{}, fmt.Errorf("activity explore %d start time: %w", row.ID, err)
	}
	end, err := time.ParseInLocation(exploreTimeLayout, row.EndTime, time.Local)
	if err != nil || !end.After(start) {
		return ExploreDefinition{}, fmt.Errorf("activity explore %d end time is invalid", row.ID)
	}
	ids, err := rewardIDs(row.Items)
	if err != nil || (len(ids) == 0 && row.SpecialItem == 0) {
		return ExploreDefinition{}, fmt.Errorf("activity explore %d has no valid rewards", row.ID)
	}
	return ExploreDefinition{
		ID: uint32(row.ID), NameID: uint32(row.NameID), IconID: uint32(row.IconID),
		NeedTime:    time.Duration(row.NeedTime) * time.Second,
		QuickItemID: uint32(row.QuickItem), QuickItemCount: int64(row.QuickCount),
		RewardItemIDs: ids, MaxTimes: uint32(row.MaxTimes), StartTime: start, EndTime: end,
		SpecialItemID: uint32(row.SpecialItem), IsActivity: true,
	}, nil
}

// ExploreModel is persisted inside Player.ArchiveMap. This keeps the feature
// self-contained while still using the existing whole-player JSON snapshot.
type ExploreModel struct {
	Areas      map[uint32]*PlayerExploreState `json:"areas,omitempty"`
	Activities map[uint32]*PlayerExploreState `json:"activities,omitempty"`
}

type PlayerExploreState struct {
	ExploreID      uint32   `json:"exploreId,omitempty"`
	Times          uint32   `json:"times,omitempty"`
	CompletedTimes uint32   `json:"completedTimes,omitempty"`
	StartTime      int64    `json:"startTime,omitempty"`
	EndTime        int64    `json:"endTime,omitempty"`
	Token          string   `json:"token,omitempty"`
	QuickFinish    bool     `json:"quickFinish,omitempty"`
	CollectItemIDs []uint32 `json:"collectItemIds,omitempty"`
	CollectReward  bool     `json:"collectReward,omitempty"`
}

func newExploreModel() *ExploreModel {
	return &ExploreModel{
		Areas: make(map[uint32]*PlayerExploreState), Activities: make(map[uint32]*PlayerExploreState),
	}
}

func (s *Player) LoadExploreModel() (*ExploreModel, error) {
	raw := s.GetArchive().GetArchiveValue(exploreArchiveKey)
	state := newExploreModel()
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), state); err != nil {
			return nil, fmt.Errorf("decode explore state: %w", err)
		}
	}
	if state.Areas == nil {
		state.Areas = make(map[uint32]*PlayerExploreState)
	}
	if state.Activities == nil {
		state.Activities = make(map[uint32]*PlayerExploreState)
	}
	state.normalize()
	return state, nil
}

func (s *Player) SaveExploreModel(state *ExploreModel) error {
	if state == nil {
		state = newExploreModel()
	}
	state.normalize()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	s.GetArchive().SetArchiveMap(exploreArchiveKey, string(data))
	return nil
}

func (m *ExploreModel) normalize() {
	if m == nil {
		return
	}
	for _, states := range []map[uint32]*PlayerExploreState{m.Areas, m.Activities} {
		for id, state := range states {
			if state == nil {
				delete(states, id)
				continue
			}
			state.ExploreID = id
			if state.StartTime < 0 || (state.StartTime > 0 && (state.Token == "" || state.EndTime <= state.StartTime)) {
				state.StartTime, state.EndTime, state.Token, state.QuickFinish = 0, 0, "", false
			}
			if state.StartTime == 0 {
				state.EndTime, state.Token, state.QuickFinish = 0, "", false
			}
		}
	}
}

func (m *ExploreModel) Get(id uint32, activity bool) *PlayerExploreState {
	states := m.Areas
	if activity {
		states = m.Activities
	}
	return states[id]
}

func (m *ExploreModel) Ensure(id uint32, activity bool) *PlayerExploreState {
	states := m.Areas
	if activity {
		states = m.Activities
	}
	if states == nil {
		states = make(map[uint32]*PlayerExploreState)
		if activity {
			m.Activities = states
		} else {
			m.Areas = states
		}
	}
	state := states[id]
	if state == nil {
		state = &PlayerExploreState{ExploreID: id}
		states[id] = state
	}
	return state
}

func (s *PlayerExploreState) PB(def ExploreDefinition) *proto.PlayerExploreInfo {
	ids := s.CollectItemIDs
	if len(ids) == 0 {
		ids = def.RewardItemIDs
	}
	return &proto.PlayerExploreInfo{
		ExploreId: s.ExploreID, Times: s.Times, StartTime: s.StartTime,
		IsQuickFinish: s.QuickFinish, CollectItemId: append([]uint32(nil), ids...),
		CollectReward: s.CollectReward,
	}
}

// DropReward is one resolved Reward.json item and its configured count range.
type DropReward struct {
	ItemID uint32
	Count  int64
}

type rewardConfigFile struct {
	RewardPool struct {
		Datas []rewardPoolRow `json:"datas"`
	} `json:"reward_pool"`
	RewardItemPool struct {
		Datas []rewardItemPoolRow `json:"datas"`
	} `json:"reward_item_pool"`
	WorldLevelPool struct {
		Datas []worldLevelPoolRow `json:"datas"`
	} `json:"world_level_pool"`
}

type rewardPoolRow struct {
	ID     int32 `json:"i_d"`
	Groups []struct {
		PoolID int32 `json:"reward_pool_group_i_d"`
	} `json:"reward_pool_group"`
}

type rewardItemPoolRow struct {
	ID     int32 `json:"i_d"`
	Groups []struct {
		ItemID   int32 `json:"item_i_d"`
		MinCount int32 `json:"item_min_count"`
		MaxCount int32 `json:"item_max_count"`
		MinLevel int32 `json:"min_level_limit"`
		MaxLevel int32 `json:"max_level_limit"`
	} `json:"reward_item_pool_group"`
}

type worldLevelPoolRow struct {
	ID     int32 `json:"i_d"`
	Groups []struct {
		WorldLevel int32 `json:"world_level"`
		PoolID     int32 `json:"pool_i_d"`
	} `json:"world_level_pool_group"`
}

type resolvedRewardTable struct {
	pools  map[uint32]rewardPoolRow
	items  map[uint32]rewardItemPoolRow
	worlds map[uint32]worldLevelPoolRow
}

var dropRewardCache sync.Map // resource path + mtime + size -> *resolvedRewardTable

// ResolveDropRewards resolves world-level, reward, and direct item pools from
// Reward.json. Missing pools/items are errors; there is no random fallback.
func ResolveDropRewards(rewardID, worldLevel, playerLevel uint32) ([]DropReward, error) {
	if rewardID == 0 {
		return nil, errors.New("drop reward id is missing")
	}
	table, err := loadResolvedRewardTable()
	if err != nil {
		return nil, err
	}
	if world, ok := table.worlds[rewardID]; ok {
		groups := make([]int32, 0, len(world.Groups))
		for _, group := range world.Groups {
			if group.WorldLevel >= 0 && uint32(group.WorldLevel) == worldLevel {
				groups = append(groups, group.PoolID)
			}
		}
		if len(groups) == 0 {
			for _, group := range world.Groups {
				if group.WorldLevel == 0 {
					groups = append(groups, group.PoolID)
				}
			}
		}
		if rewards, err := table.resolveItemPools(groups, playerLevel); err != nil {
			return nil, err
		} else if len(rewards) > 0 {
			return rewards, nil
		}
	}
	if pool, ok := table.pools[rewardID]; ok {
		groups := make([]int32, 0, len(pool.Groups))
		for _, group := range pool.Groups {
			groups = append(groups, group.PoolID)
		}
		rewards, err := table.resolveItemPools(groups, playerLevel)
		if err != nil {
			return nil, err
		}
		if len(rewards) > 0 {
			return rewards, nil
		}
	}
	if _, ok := table.items[rewardID]; ok {
		rewards, err := table.resolveItemPools([]int32{int32(rewardID)}, playerLevel)
		if err != nil {
			return nil, err
		}
		if len(rewards) > 0 {
			return rewards, nil
		}
	}
	return nil, fmt.Errorf("reward pool %d has no valid drops", rewardID)
}

func loadResolvedRewardTable() (*resolvedRewardTable, error) {
	resources := appconfig.GetConfig().Resources
	if resources == nil || resources.ResourcePath == "" {
		return nil, errors.New("resource path is not configured")
	}
	path := filepath.Join(resources.ResourcePath, "Excel", "Reward.json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat Reward.json: %w", err)
	}
	key := fmt.Sprintf("%s|%d|%d", path, info.ModTime().UnixNano(), info.Size())
	if cached, ok := dropRewardCache.Load(key); ok {
		return cached.(*resolvedRewardTable), nil
	}
	var raw rewardConfigFile
	if err := readJSONFile(path, &raw); err != nil {
		return nil, err
	}
	table := &resolvedRewardTable{
		pools: make(map[uint32]rewardPoolRow), items: make(map[uint32]rewardItemPoolRow),
		worlds: make(map[uint32]worldLevelPoolRow),
	}
	for _, row := range raw.RewardPool.Datas {
		if row.ID > 0 {
			table.pools[uint32(row.ID)] = row
		}
	}
	for _, row := range raw.RewardItemPool.Datas {
		if row.ID > 0 {
			table.items[uint32(row.ID)] = row
		}
	}
	for _, row := range raw.WorldLevelPool.Datas {
		if row.ID > 0 {
			table.worlds[uint32(row.ID)] = row
		}
	}
	dropRewardCache.Store(key, table)
	return table, nil
}

func (t *resolvedRewardTable) resolveItemPools(ids []int32, playerLevel uint32) ([]DropReward, error) {
	out := make([]DropReward, 0)
	for _, id := range ids {
		pool, ok := t.items[uint32(id)]
		if !ok {
			return nil, fmt.Errorf("reward item pool %d is missing", id)
		}
		for _, group := range pool.Groups {
			if group.ItemID <= 0 {
				return nil, fmt.Errorf("reward item pool %d has an invalid item", id)
			}
			if playerLevel != 0 && group.MinLevel > 0 && playerLevel < uint32(group.MinLevel) {
				continue
			}
			if playerLevel != 0 && group.MaxLevel > 0 && playerLevel > uint32(group.MaxLevel) {
				continue
			}
			if group.MinCount <= 0 || group.MaxCount < group.MinCount {
				return nil, fmt.Errorf("reward item pool %d has an invalid count", id)
			}
			count := int64(group.MinCount)
			if group.MaxCount > group.MinCount {
				count = int64(group.MinCount) + rand.Int64N(int64(group.MaxCount-group.MinCount)+1)
			}
			out = append(out, DropReward{ItemID: uint32(group.ItemID), Count: count})
		}
	}
	return out, nil
}
