package model

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

const maxLifeReceipts = 128

type LifeModel struct {
	States       map[uint32]*LifeState  `json:"states,omitempty"`
	Receipts     map[string]LifeReceipt `json:"receipts,omitempty"`
	ReceiptOrder []string               `json:"receiptOrder,omitempty"`
	Generation   uint64                 `json:"generation,omitempty"`
	session      string                 `json:"-"`
	opMu         sync.Mutex             `json:"-"`
}

type LifeState struct {
	Level        uint32                       `json:"level,omitempty"`
	Proficiency  uint32                       `json:"proficiency,omitempty"`
	Skills       map[uint32]*LifeSkillState   `json:"skills,omitempty"`
	Achievements map[uint32]*LifeAchieveState `json:"achievements,omitempty"`
}

type LifeSkillState struct {
	Lv             uint32 `json:"lv,omitempty"`
	StartStudyTime int64  `json:"startStudyTime,omitempty"`
	ReadyAt        int64  `json:"readyAt,omitempty"`
}

type LifeAchieveState struct {
	Count    uint64 `json:"count,omitempty"`
	Rewarded bool   `json:"rewarded,omitempty"`
}

type LifeSkillSnapshot struct {
	SkillId        uint32 `json:"skillId,omitempty"`
	Lv             uint32 `json:"lv,omitempty"`
	StartStudyTime int64  `json:"startStudyTime,omitempty"`
}

type LifeItem struct {
	ItemID uint32            `json:"itemId,omitempty"`
	Count  int64             `json:"count,omitempty"`
	Tag    proto.EBagItemTag `json:"tag,omitempty"`
}

type LifeFishResult struct {
	ID     uint32  `json:"id,omitempty"`
	Length float32 `json:"length,omitempty"`
}

type LifeReceipt struct {
	Status             proto.StatusCode    `json:"status,omitempty"`
	Persist            bool                `json:"-"`
	LifeType           proto.LifeType      `json:"lifeType,omitempty"`
	Success            bool                `json:"success,omitempty"`
	IsCancel           bool                `json:"isCancel,omitempty"`
	Items              []LifeItem          `json:"items,omitempty"`
	ResultItems        []LifeItem          `json:"resultItems,omitempty"`
	SkillReturnItems   []LifeItem          `json:"skillReturnItems,omitempty"`
	IsLuckyReturn      bool                `json:"isLuckyReturn,omitempty"`
	Proficiency        uint32              `json:"proficiency,omitempty"`
	Level              uint32              `json:"level,omitempty"`
	Fish               []LifeFishResult    `json:"fish,omitempty"`
	CostEndurance      uint32              `json:"costEndurance,omitempty"`
	CostTime           uint32              `json:"costTime,omitempty"`
	Skills             []LifeSkillSnapshot `json:"skills,omitempty"`
	RewardedAchieveIDs []uint32            `json:"rewardedAchieveIds,omitempty"`
	AchieveIDs         []uint32            `json:"achieveIds,omitempty"`
}

func (i LifeItem) ItemDetail() *proto.ItemDetail {
	if i.ItemID == 0 || i.Count < 0 {
		return nil
	}
	return &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  i.ItemID,
			ItemTag: i.Tag,
			Item: &proto.ItemInfo_BaseItem{
				BaseItem: &proto.BaseItem{ItemId: i.ItemID, Num: i.Count},
			},
		},
		PackType: proto.PackType_PackType_Inventory,
	}
}

func DefaultLifeModel() *LifeModel {
	m := &LifeModel{
		States:   make(map[uint32]*LifeState),
		Receipts: make(map[string]LifeReceipt),
	}
	for lifeType := proto.LifeType_LifeType_Cooking; lifeType <= proto.LifeType_LifeType_Handicraft; lifeType++ {
		m.States[uint32(lifeType)] = newLifeState()
	}
	return m
}

func newLifeState() *LifeState {
	return &LifeState{
		Level:        1,
		Skills:       make(map[uint32]*LifeSkillState),
		Achievements: make(map[uint32]*LifeAchieveState),
	}
}

func (s *Player) GetLifeModel() *LifeModel {
	if s.Life == nil {
		s.Life = DefaultLifeModel()
	}
	s.Life.normalize()
	return s.Life
}

func (m *LifeModel) normalize() {
	if m.States == nil {
		m.States = make(map[uint32]*LifeState)
	}
	if m.Receipts == nil {
		m.Receipts = make(map[string]LifeReceipt)
	}
	for lifeType := proto.LifeType_LifeType_Cooking; lifeType <= proto.LifeType_LifeType_Handicraft; lifeType++ {
		state := m.States[uint32(lifeType)]
		if state == nil {
			m.States[uint32(lifeType)] = newLifeState()
			continue
		}
		if state.Level == 0 {
			state.Level = 1
		}
		if state.Skills == nil {
			state.Skills = make(map[uint32]*LifeSkillState)
		}
		if state.Achievements == nil {
			state.Achievements = make(map[uint32]*LifeAchieveState)
		}
	}
}

func (m *LifeModel) GetState(lifeType proto.LifeType) *LifeState {
	m.normalize()
	state := m.States[uint32(lifeType)]
	if state == nil {
		state = newLifeState()
		m.States[uint32(lifeType)] = state
	}
	return state
}

func (m *LifeModel) RequestToken(loginUUID string, seq, msgID, id uint32, operation string) string {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if m.session != loginUUID {
		m.Generation++
		m.session = loginUUID
	}
	return fmt.Sprintf("%d:%d:%d:%s:%d", m.Generation, msgID, seq, operation, id)
}

func (m *LifeModel) DoOnce(token string, fn func() LifeReceipt) LifeReceipt {
	return m.doOnce(token, nil, nil, fn)
}

// doOnce 将生活技能状态、背包变更和落库放在同一把锁内。落库失败时恢复
// 调用前的快照且不缓存凭证，客户端重试不会重复扣除或重复发奖。
func (m *LifeModel) doOnce(token string, items *ItemModel, persist func() error, fn func() LifeReceipt) LifeReceipt {
	if m == nil {
		return fn()
	}
	m.opMu.Lock()
	defer m.opMu.Unlock()
	if token != "" {
		if receipt, ok := m.Receipts[token]; ok {
			return receipt
		}
	}
	var lifeBefore map[uint32]*LifeState
	var itemsBefore map[uint32]*ItemBaseInfo
	var receiptsBefore map[string]LifeReceipt
	var orderBefore []string
	if persist != nil {
		lifeBefore = cloneLifeStates(m.States)
		receiptsBefore = cloneLifeReceipts(m.Receipts)
		orderBefore = append([]string(nil), m.ReceiptOrder...)
		if items != nil {
			itemsBefore = cloneLifeItemInfos(items.GetItemBaseMap())
		}
	}
	receipt := fn()
	if receipt.Status == proto.StatusCode_StatusCode_SysErr {
		return receipt
	}
	if token != "" {
		if m.Receipts == nil {
			m.Receipts = make(map[string]LifeReceipt)
		}
		m.ReceiptOrder = append(m.ReceiptOrder, token)
		m.Receipts[token] = receipt
	}
	for len(m.ReceiptOrder) > maxLifeReceipts {
		old := m.ReceiptOrder[0]
		m.ReceiptOrder = m.ReceiptOrder[1:]
		delete(m.Receipts, old)
	}
	if receipt.Persist && persist != nil {
		if err := persist(); err != nil {
			m.States = lifeBefore
			m.Receipts = receiptsBefore
			m.ReceiptOrder = orderBefore
			if items != nil {
				items.ItemBaseInfo = itemsBefore
			}
			return LifeReceipt{Status: proto.StatusCode_StatusCode_SysErr, LifeType: receipt.LifeType}
		}
	}
	return receipt
}

func cloneLifeReceipts(receipts map[string]LifeReceipt) map[string]LifeReceipt {
	result := make(map[string]LifeReceipt, len(receipts))
	for token, receipt := range receipts {
		result[token] = receipt
	}
	return result
}

func cloneLifeStates(states map[uint32]*LifeState) map[uint32]*LifeState {
	result := make(map[uint32]*LifeState, len(states))
	for id, state := range states {
		if state == nil {
			result[id] = nil
			continue
		}
		copyState := &LifeState{
			Level: state.Level, Proficiency: state.Proficiency,
			Skills:       make(map[uint32]*LifeSkillState, len(state.Skills)),
			Achievements: make(map[uint32]*LifeAchieveState, len(state.Achievements)),
		}
		for skillID, skill := range state.Skills {
			if skill != nil {
				copySkill := *skill
				copyState.Skills[skillID] = &copySkill
			} else {
				copyState.Skills[skillID] = nil
			}
		}
		for achieveID, achieve := range state.Achievements {
			if achieve != nil {
				copyAchieve := *achieve
				copyState.Achievements[achieveID] = &copyAchieve
			} else {
				copyState.Achievements[achieveID] = nil
			}
		}
		result[id] = copyState
	}
	return result
}

func cloneLifeItemInfos(infos map[uint32]*ItemBaseInfo) map[uint32]*ItemBaseInfo {
	result := make(map[uint32]*ItemBaseInfo, len(infos))
	for id, info := range infos {
		if info != nil {
			copyInfo := *info
			result[id] = &copyInfo
		} else {
			result[id] = nil
		}
	}
	return result
}

func (m *LifeModel) BaseInfo(lifeType proto.LifeType) *proto.LifeBaseInfo {
	state := m.GetState(lifeType)
	return &proto.LifeBaseInfo{LifeType: lifeType, Level: state.Level, ProficiencyValue: state.Proficiency}
}

func (m *LifeModel) SkillInfo(lifeType proto.LifeType, cfg *LifeConfig) []*proto.SingleLifeSkillInfo {
	if cfg == nil {
		return nil
	}
	ids := make([]uint32, 0, len(cfg.Skills[lifeType]))
	for id := range cfg.Skills[lifeType] {
		ids = append(ids, id)
	}
	state := m.GetState(lifeType)
	for id := range state.Skills {
		if _, ok := cfg.Skills[lifeType][id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]*proto.SingleLifeSkillInfo, 0, len(ids))
	for _, id := range ids {
		skill := state.Skills[id]
		if skill == nil {
			result = append(result, &proto.SingleLifeSkillInfo{SkillId: id})
			continue
		}
		result = append(result, &proto.SingleLifeSkillInfo{SkillId: id, Lv: skill.Lv, StartStudyTime: skill.StartStudyTime})
	}
	return result
}

func (m *LifeModel) AchieveInfo(lifeType proto.LifeType, cfg *LifeConfig) *proto.LifeAchieve {
	result := &proto.LifeAchieve{LifeType: lifeType}
	if cfg == nil {
		return result
	}
	rewarded := make([]uint32, 0)
	for id, conf := range cfg.Achievements {
		if conf.LifeType != lifeType {
			continue
		}
		state := m.GetState(lifeType).Achievements[id]
		if state != nil {
			result.LifeAchieve_ = append(result.LifeAchieve_, &proto.Achieve{AchieveId: id, Count: state.Count})
			if state.Rewarded {
				rewarded = append(rewarded, id)
			}
		} else {
			result.LifeAchieve_ = append(result.LifeAchieve_, &proto.Achieve{AchieveId: id})
		}
	}
	sort.Slice(result.LifeAchieve_, func(i, j int) bool { return result.LifeAchieve_[i].AchieveId < result.LifeAchieve_[j].AchieveId })
	sort.Slice(rewarded, func(i, j int) bool { return rewarded[i] < rewarded[j] })
	result.RewardedLifeAchieveIdLst = rewarded
	return result
}

func (m *LifeModel) RefreshSkills(lifeType proto.LifeType, cfg *LifeConfig, now int64) bool {
	return m.refreshSkills(lifeType, cfg, now)
}

func (m *LifeModel) RefreshSkillsAndSave(lifeType proto.LifeType, cfg *LifeConfig, now int64, save func() error) (bool, error) {
	if m == nil {
		return false, nil
	}
	if cfg == nil || save == nil {
		return m.RefreshSkills(lifeType, cfg, now), nil
	}
	m.opMu.Lock()
	defer m.opMu.Unlock()
	before := cloneLifeStates(m.States)
	changed := m.refreshSkills(lifeType, cfg, now)
	if changed {
		if err := save(); err != nil {
			m.States = before
			return false, err
		}
	}
	return changed, nil
}

func (m *LifeModel) refreshSkills(lifeType proto.LifeType, cfg *LifeConfig, now int64) bool {
	if cfg == nil {
		return false
	}
	changed := false
	for id, skill := range m.GetState(lifeType).Skills {
		if skill == nil || skill.StartStudyTime <= 0 {
			continue
		}
		readyAt := skill.ReadyAt
		if readyAt == 0 {
			conf := cfg.Skills[lifeType][id]
			if conf == nil {
				continue
			}
			level := conf.Levels[skill.Lv+1]
			if level == nil || level.NeedTime < 0 || level.NeedTime > math.MaxInt64-skill.StartStudyTime {
				continue
			}
			readyAt = skill.StartStudyTime + level.NeedTime
		}
		if now >= readyAt {
			skill.Lv++
			skill.StartStudyTime = 0
			skill.ReadyAt = 0
			changed = true
		}
	}
	return changed
}

func (m *LifeModel) AddAchievement(lifeType proto.LifeType, id, count uint32) {
	if id == 0 || count == 0 {
		return
	}
	state := m.GetState(lifeType).Achievements[id]
	if state == nil {
		state = &LifeAchieveState{}
		m.GetState(lifeType).Achievements[id] = state
	}
	if ^uint64(0)-state.Count < uint64(count) {
		state.Count = ^uint64(0)
		return
	}
	state.Count += uint64(count)
}

type LifeConfig struct {
	Levels       map[proto.LifeType]map[uint32]*LifeLevelConfig
	Recipes      map[proto.LifeType]map[uint32]*LifeRecipe
	Fish         map[uint32]*LifeFishConfig
	Skills       map[proto.LifeType]map[uint32]*LifeSkillConfig
	Achievements map[uint32]*LifeAchievementConfig
	Rewards      map[uint32][]LifeReward
	Items        map[uint32]*excel.ItemConfigure
	WeaponTypes  map[uint32]proto.EWeaponType
}

type LifeLevelConfig struct {
	Level                uint32
	Proficiency          uint32
	CostItemID           uint32
	CostItemCount        int64
	ActionEnergyID       uint32
	ActionEnergyCount    int64
	ProficiencyPerAction uint32
	Achievements         []LifeAchievementRequirement
}

type LifeAchievementRequirement struct {
	ID          uint32
	Target      uint32
	Proficiency uint32
}

type LifeAchievementRef struct {
	ID       uint32
	RewardID uint32
}

type LifeMaterial struct {
	ItemID uint32
	Count  int64
}

type LifeRecipe struct {
	ID            uint32
	Level         uint32
	IsShow        bool
	NeedTime      int64
	OpeningTime   string
	ClosingTime   string
	EnergyID      uint32
	EnergyCount   int64
	Materials     []LifeMaterial
	Probability   float32
	OutputID      uint32
	FormulaItemID uint32
	Achievements  []LifeAchievementRef
	ToolType      int32
}

type LifeFishConfig struct {
	ID           uint32
	Level        uint32
	ItemID       uint32
	IsShow       bool
	SizeType     proto.ELifeFishSizeType
	MinSize      float32
	MaxSize      float32
	Achievements []LifeAchievementRef
}

type LifeSkillConfig struct {
	ID     uint32
	Levels map[uint32]*LifeSkillLevelConfig
}

type LifeSkillLevelConfig struct {
	Level         uint32
	Effect        proto.ELifeSpellEffectType
	Param1        float32
	Param2        float32
	NeedLifeLevel uint32
	NeedTime      int64
	CurrencyID    uint32
	CurrencyCount int64
	Materials     []LifeMaterial
}

type LifeAchievementConfig struct {
	ID       uint32
	LifeType proto.LifeType
	Target   uint32
	RewardID uint32
}

type LifeReward struct {
	ItemID uint32
	Min    int64
	Max    int64
}

func LoadLifeConfig(resourcePath string) (*LifeConfig, error) {
	cfg := &LifeConfig{
		Levels:       make(map[proto.LifeType]map[uint32]*LifeLevelConfig),
		Recipes:      make(map[proto.LifeType]map[uint32]*LifeRecipe),
		Fish:         make(map[uint32]*LifeFishConfig),
		Skills:       make(map[proto.LifeType]map[uint32]*LifeSkillConfig),
		Achievements: make(map[uint32]*LifeAchievementConfig),
		Rewards:      make(map[uint32][]LifeReward),
		Items:        make(map[uint32]*excel.ItemConfigure),
		WeaponTypes:  make(map[uint32]proto.EWeaponType),
	}
	for _, item := range gdconf.GetAllItemConfigure() {
		cfg.Items[uint32(item.ID)] = item
	}
	for id, weapon := range gdconf.GetWeaponAllMap() {
		if weapon != nil && weapon.WeaponInfo != nil {
			cfg.WeaponTypes[id] = proto.EWeaponType(weapon.WeaponInfo.NewWeaponType)
		}
	}
	excelPath := filepath.Join(resourcePath, "Excel")
	var fishing excel.AllFishingDatas
	var cooking excel.AllCookingDatas
	var handicraft excel.AllHandicraftDatas
	var making excel.AllMakeDatas
	for name, out := range map[string]any{
		"Fishing.json": &fishing, "Cooking.json": &cooking,
		"Handicraft.json": &handicraft, "Make.json": &making,
	} {
		if err := readLifeJSON(filepath.Join(excelPath, name), out); err != nil {
			return nil, err
		}
	}
	for _, lifeType := range []proto.LifeType{
		proto.LifeType_LifeType_Cooking, proto.LifeType_LifeType_Fishing,
		proto.LifeType_LifeType_Making, proto.LifeType_LifeType_Handicraft,
	} {
		cfg.Levels[lifeType] = make(map[uint32]*LifeLevelConfig)
		cfg.Recipes[lifeType] = make(map[uint32]*LifeRecipe)
		cfg.Skills[lifeType] = make(map[uint32]*LifeSkillConfig)
	}
	for _, row := range fishing.GetFishingLevel().GetDatas() {
		reqs := make([]LifeAchievementRequirement, 0, len(row.FishingLevelQuest))
		for _, item := range row.FishingLevelQuest {
			reqs = append(reqs, LifeAchievementRequirement{ID: uint32(item.AchieveConditionID), Target: uint32(item.CompleteLimit), Proficiency: uint32(item.Proficiency)})
		}
		cfg.addLevel(proto.LifeType_LifeType_Fishing, uint32(row.ID), uint32(row.UpgradeLevelByProficiency), uint32(row.UpLevelNeedItemID), int64(row.UpLevelNeedItemCount), uint32(row.NeedItemID), int64(row.NeedItemCount), reqs)
	}
	for _, row := range cooking.GetCookLevel().GetDatas() {
		reqs := make([]LifeAchievementRequirement, 0, len(row.CookLevelQuest))
		for _, item := range row.CookLevelQuest {
			reqs = append(reqs, LifeAchievementRequirement{ID: uint32(item.AchieveConditionID), Target: uint32(item.CompleteLimit), Proficiency: uint32(item.Proficiency)})
		}
		cfg.addLevel(proto.LifeType_LifeType_Cooking, uint32(row.ID), uint32(row.UpgradeLevelByProficiency), uint32(row.NeedItemID), int64(row.NeedItemCount), 0, 0, reqs)
	}
	for _, row := range making.GetMakeLevel().GetDatas() {
		reqs := make([]LifeAchievementRequirement, 0, len(row.MakeLevelAchieve))
		for _, item := range row.MakeLevelAchieve {
			reqs = append(reqs, LifeAchievementRequirement{ID: uint32(item.AchieveConditionID), Target: uint32(item.CompleteLimit), Proficiency: uint32(item.Proficiency)})
		}
		cfg.addLevel(proto.LifeType_LifeType_Making, uint32(row.ID), uint32(row.UpgradeLevelByProficiency), uint32(row.NeedItemID), int64(row.NeedItemCount), 0, 0, reqs)
	}
	for _, row := range handicraft.GetHandicraftLevel().GetDatas() {
		reqs := make([]LifeAchievementRequirement, 0, len(row.HandicraftLevelAchieve))
		for _, item := range row.HandicraftLevelAchieve {
			reqs = append(reqs, LifeAchievementRequirement{ID: uint32(item.AchieveConditionID), Target: uint32(item.CompleteLimit), Proficiency: uint32(item.Proficiency)})
		}
		cfg.addLevel(proto.LifeType_LifeType_Handicraft, uint32(row.ID), uint32(row.UpgradeLevelByProficiency), uint32(row.NeedItemID), int64(row.NeedItemCount), 0, 0, reqs)
	}
	for _, row := range fishing.GetFishingInfo().GetDatas() {
		refs := make([]LifeAchievementRef, 0, len(row.FishAchieveRewards))
		for _, item := range row.FishAchieveRewards {
			refs = append(refs, LifeAchievementRef{ID: uint32(item.AchieveConditionID), RewardID: uint32(item.RewardID)})
		}
		cfg.Fish[uint32(row.ID)] = &LifeFishConfig{ID: uint32(row.ID), Level: uint32(row.NeedLevel), ItemID: uint32(row.ItemID), IsShow: row.IsShow, MinSize: row.MinSize, MaxSize: row.MaxSize, Achievements: refs}
		cfg.registerAchievements(proto.LifeType_LifeType_Fishing, refs)
	}
	for _, row := range cooking.GetCookFood().GetDatas() {
		refs := make([]LifeAchievementRef, 0, len(row.CookFoodQuestRewards))
		for _, item := range row.CookFoodQuestRewards {
			refs = append(refs, LifeAchievementRef{ID: uint32(item.AchieveConditionID), RewardID: uint32(item.RewardID)})
		}
		materials := make([]LifeMaterial, 0, len(row.CookFoodItems))
		for _, item := range row.CookFoodItems {
			materials = append(materials, LifeMaterial{ItemID: uint32(item.FoodItemID), Count: int64(item.FoodItemCount)})
		}
		cfg.Recipes[proto.LifeType_LifeType_Cooking][uint32(row.ID)] = &LifeRecipe{
			ID: uint32(row.ID), Level: uint32(row.CookLevel), IsShow: row.IsShow, NeedTime: int64(row.NeedTime),
			OpeningTime: row.OpeningTime, ClosingTime: row.ClosingTime, EnergyID: uint32(row.NeedItemID), EnergyCount: int64(row.NeedItemCount),
			Materials: materials, Probability: row.SuccessfulProbability, OutputID: uint32(row.GetItemID), FormulaItemID: uint32(row.NeedFormulaItemID),
			Achievements: refs, ToolType: row.NewCookToolType,
		}
		cfg.registerAchievements(proto.LifeType_LifeType_Cooking, refs)
	}
	for _, row := range making.GetMakeItem().GetDatas() {
		refs := make([]LifeAchievementRef, 0, len(row.MakeAchieveRewards))
		for _, item := range row.MakeAchieveRewards {
			refs = append(refs, LifeAchievementRef{ID: uint32(item.AchieveConditionID), RewardID: uint32(item.RewardID)})
		}
		materials := make([]LifeMaterial, 0, len(row.MakeItems))
		for _, item := range row.MakeItems {
			materials = append(materials, LifeMaterial{ItemID: uint32(item.ItemID), Count: int64(item.ItemCount)})
		}
		cfg.Recipes[proto.LifeType_LifeType_Making][uint32(row.ID)] = &LifeRecipe{
			ID: uint32(row.ID), Level: uint32(row.MakeLevel), IsShow: row.IsShow, NeedTime: int64(row.NeedTime),
			OpeningTime: row.OpeningTime, ClosingTime: row.ClosingTime, EnergyID: uint32(row.NeedItemID), EnergyCount: int64(row.NeedItemCount),
			Materials: materials, Probability: row.SuccessfulProbability, OutputID: uint32(row.GetItemID), FormulaItemID: uint32(row.NeedFormulaItemID),
			Achievements: refs, ToolType: row.NewMakeToolType,
		}
		cfg.registerAchievements(proto.LifeType_LifeType_Making, refs)
	}
	for _, row := range handicraft.GetHandicraftItem().GetDatas() {
		refs := make([]LifeAchievementRef, 0, len(row.HandicraftAchieveRewards))
		for _, item := range row.HandicraftAchieveRewards {
			refs = append(refs, LifeAchievementRef{ID: uint32(item.AchieveConditionID), RewardID: uint32(item.RewardID)})
		}
		materials := make([]LifeMaterial, 0, len(row.HandicraftItems))
		for _, item := range row.HandicraftItems {
			materials = append(materials, LifeMaterial{ItemID: uint32(item.ItemID), Count: int64(item.ItemCount)})
		}
		cfg.Recipes[proto.LifeType_LifeType_Handicraft][uint32(row.ID)] = &LifeRecipe{
			ID: uint32(row.ID), Level: uint32(row.HandicraftLevel), IsShow: row.IsShow, NeedTime: int64(row.NeedTime),
			OpeningTime: row.OpeningTime, ClosingTime: row.ClosingTime, EnergyID: uint32(row.NeedItemID), EnergyCount: int64(row.NeedItemCount),
			Materials: materials, Probability: row.SuccessfulProbability, OutputID: uint32(row.GetItemID), FormulaItemID: uint32(row.NeedFormulaItemID),
			Achievements: refs, ToolType: row.NewHandicraftToolType,
		}
		cfg.registerAchievements(proto.LifeType_LifeType_Handicraft, refs)
	}
	for _, row := range fishing.GetFishingSpell().GetDatas() {
		skill := cfg.ensureSkill(proto.LifeType_LifeType_Fishing, uint32(row.ID))
		for _, item := range row.FishingSpellItems {
			cfg.addSkill(skill, item.Level, item.NewLifeSpellEffectType, item.Param1, item.Param2, item.NeedFishLevel, item.NeedTime, item.NeedCurrencyID, item.NeedCurrencyCount, fishingMaterials(item.FishingNeedItem))
		}
	}
	for _, row := range cooking.GetCookSpell().GetDatas() {
		skill := cfg.ensureSkill(proto.LifeType_LifeType_Cooking, uint32(row.ID))
		for _, item := range row.CookSpellItems {
			cfg.addSkill(skill, item.Level, item.NewLifeSpellEffectType, item.SpellParam1, item.SpellParam2, item.NeedCookLevel, item.NeedTime, item.NeedCurrencyID, item.NeedCurrencyCount, cookingMaterials(item.CookNeedItem))
		}
	}
	for _, row := range making.GetMakeSpell().GetDatas() {
		skill := cfg.ensureSkill(proto.LifeType_LifeType_Making, uint32(row.ID))
		for _, item := range row.MakeSpellItems {
			cfg.addSkill(skill, item.Level, item.NewLifeSpellEffectType, item.SpellParam1, item.SpellParam2, item.NeedMakeLevel, item.NeedTime, item.NeedCurrencyID, item.NeedCurrencyCount, makeMaterials(item.MakeNeedItem))
		}
	}
	for _, row := range handicraft.GetHandicraftSpell().GetDatas() {
		skill := cfg.ensureSkill(proto.LifeType_LifeType_Handicraft, uint32(row.ID))
		for _, item := range row.HandicraftSpellItems {
			cfg.addSkill(skill, item.Level, item.NewLifeSpellEffectType, item.SpellParam1, item.SpellParam2, item.NeedHandicraftLevel, item.NeedTime, item.NeedCurrencyID, item.NeedCurrencyCount, handicraftMaterials(item.HandicraftNeedItem))
		}
	}
	rewardIDs := make(map[uint32]struct{})
	for _, conf := range cfg.Achievements {
		if conf.RewardID != 0 {
			rewardIDs[conf.RewardID] = struct{}{}
		}
	}
	for id := range rewardIDs {
		for _, reward := range gdconf.GetRewardItemPoolByRewardId(id) {
			if reward != nil {
				cfg.Rewards[id] = append(cfg.Rewards[id], LifeReward{ItemID: uint32(reward.ItemID), Min: int64(reward.ItemMinCount), Max: int64(reward.ItemMaxCount)})
			}
		}
	}
	for _, lifeType := range []proto.LifeType{
		proto.LifeType_LifeType_Cooking, proto.LifeType_LifeType_Fishing,
		proto.LifeType_LifeType_Making, proto.LifeType_LifeType_Handicraft,
	} {
		if len(cfg.Levels[lifeType]) == 0 {
			return nil, fmt.Errorf("生活技能配置缺失: %s", lifeType)
		}
	}
	return cfg, nil
}

func readLifeJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := sonic.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}

func (c *LifeConfig) addLevel(lifeType proto.LifeType, level, proficiency, costID uint32, costCount int64, energyID uint32, energyCount int64, reqs []LifeAchievementRequirement) {
	perAction := uint32(0)
	for _, req := range reqs {
		if req.ID == 0 {
			continue
		}
		if req.Proficiency > 0 && (perAction == 0 || req.Proficiency < perAction) {
			perAction = req.Proficiency
		}
		conf := c.Achievements[req.ID]
		if conf == nil {
			conf = &LifeAchievementConfig{ID: req.ID, LifeType: lifeType}
			c.Achievements[req.ID] = conf
		}
		if conf.Target == 0 {
			conf.Target = req.Target
		}
	}
	c.Levels[lifeType][level] = &LifeLevelConfig{
		Level: level, Proficiency: proficiency, CostItemID: costID, CostItemCount: costCount,
		ActionEnergyID: energyID, ActionEnergyCount: energyCount, ProficiencyPerAction: perAction, Achievements: reqs,
	}
}

func (c *LifeConfig) registerAchievements(lifeType proto.LifeType, refs []LifeAchievementRef) {
	for _, ref := range refs {
		if ref.ID == 0 {
			continue
		}
		conf := c.Achievements[ref.ID]
		if conf == nil {
			conf = &LifeAchievementConfig{ID: ref.ID, LifeType: lifeType}
			c.Achievements[ref.ID] = conf
		}
		if conf.Target == 0 {
			if achieve := gdconf.GetAchieveConfigure(ref.ID); achieve != nil && achieve.CountParam > 0 {
				conf.Target = uint32(achieve.CountParam)
			}
		}
		if conf.RewardID == 0 {
			conf.RewardID = ref.RewardID
		}
	}
}

func (c *LifeConfig) ensureSkill(lifeType proto.LifeType, id uint32) *LifeSkillConfig {
	skill := c.Skills[lifeType][id]
	if skill == nil {
		skill = &LifeSkillConfig{ID: id, Levels: make(map[uint32]*LifeSkillLevelConfig)}
		c.Skills[lifeType][id] = skill
	}
	return skill
}

func (c *LifeConfig) addSkill(skill *LifeSkillConfig, level int32, effect int32, param1, param2 float32, needLife, needTime, currencyID, currencyCount int32, materials []LifeMaterial) {
	skill.Levels[uint32(level)] = &LifeSkillLevelConfig{
		Level: uint32(level), Effect: proto.ELifeSpellEffectType(effect), Param1: param1, Param2: param2,
		NeedLifeLevel: uint32(needLife), NeedTime: int64(needTime), CurrencyID: uint32(currencyID),
		CurrencyCount: int64(currencyCount), Materials: materials,
	}
}

func fishingMaterials(rows []*excel.FishingNeedItems) []LifeMaterial {
	result := make([]LifeMaterial, 0, len(rows))
	for _, row := range rows {
		result = append(result, LifeMaterial{ItemID: uint32(row.NeedItemID), Count: int64(row.NeedItemCount)})
	}
	return result
}

func cookingMaterials(rows []*excel.CookNeedItems) []LifeMaterial {
	result := make([]LifeMaterial, 0, len(rows))
	for _, row := range rows {
		result = append(result, LifeMaterial{ItemID: uint32(row.NeedItemID), Count: int64(row.NeedItemCount)})
	}
	return result
}

func makeMaterials(rows []*excel.MakeNeedItems) []LifeMaterial {
	result := make([]LifeMaterial, 0, len(rows))
	for _, row := range rows {
		result = append(result, LifeMaterial{ItemID: uint32(row.NeedItemID), Count: int64(row.NeedItemCount)})
	}
	return result
}

func handicraftMaterials(rows []*excel.HandicraftNeedItems) []LifeMaterial {
	result := make([]LifeMaterial, 0, len(rows))
	for _, row := range rows {
		result = append(result, LifeMaterial{ItemID: uint32(row.NeedItemID), Count: int64(row.NeedItemCount)})
	}
	return result
}

type LifeRNG interface {
	IntN(int) int
	Float64() float64
}

type LifeProductionOptions struct {
	Token       string
	Now         int64
	RNG         LifeRNG
	HasCampFire bool
	Persist     func() error
}

func lifeRandom(rng LifeRNG) float64 {
	if rng == nil {
		return rand.Float64()
	}
	return rng.Float64()
}

func lifeIntn(rng LifeRNG, n int) int {
	if rng == nil {
		return rand.IntN(n)
	}
	return rng.IntN(n)
}

func ValidLifeType(lifeType proto.LifeType) bool {
	return lifeType >= proto.LifeType_LifeType_Cooking && lifeType <= proto.LifeType_LifeType_Handicraft
}

func (m *LifeModel) ProduceRecipe(items *ItemModel, cfg *LifeConfig, lifeType proto.LifeType, recipeID uint32, opts LifeProductionOptions) LifeReceipt {
	return m.doOnce(opts.Token, items, opts.Persist, func() LifeReceipt {
		receipt := LifeReceipt{Status: proto.StatusCode_StatusCode_Ok, LifeType: lifeType}
		if !ValidLifeType(lifeType) || cfg == nil || items == nil {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		recipe := cfg.Recipes[lifeType][recipeID]
		if recipe == nil || !recipe.IsShow {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		state := m.GetState(lifeType)
		if recipe.Level > state.Level {
			receipt.Status = proto.StatusCode_StatusCode_LifeLvNotEnough
			return receipt
		}
		if ok, valid := lifeTimeAvailable(recipe.OpeningTime, recipe.ClosingTime, opts.Now); !valid {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		} else if !ok {
			receipt.Status = proto.StatusCode_StatusCode_ActivityClose
			return receipt
		}
		if lifeType == proto.LifeType_LifeType_Cooking && recipe.ToolType == int32(proto.ELifeCookToolType_ELifeCookToolType_Bbqstove) && !opts.HasCampFire {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		if recipe.FormulaItemID != 0 {
			if _, status := lifeBaseItem(cfg, recipe.FormulaItemID); status != proto.StatusCode_StatusCode_Ok {
				receipt.Status = status
				return receipt
			}
			if info := items.GetItemBaseInfo(recipe.FormulaItemID); info == nil || info.Num < 1 {
				receipt.Status = proto.StatusCode_StatusCode_ItemNotEnough
				return receipt
			}
		}
		level := cfg.Levels[lifeType][state.Level]
		if level == nil || (level.ProficiencyPerAction == 0 && cfg.Levels[lifeType][state.Level+1] != nil) {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		if _, status := lifeBaseItem(cfg, recipe.OutputID); status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		if recipe.Probability < 0 || recipe.Probability > 1 || math.IsNaN(float64(recipe.Probability)) {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		if recipe.NeedTime < 0 || !validLifeEnergy(cfg, recipe.EnergyID, recipe.EnergyCount) {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		for _, material := range recipe.Materials {
			if material.Count < 0 || (material.Count > 0 && !validLifeItem(cfg, material.ItemID)) {
				receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
				return receipt
			}
		}
		receipt.Persist = m.RefreshSkills(lifeType, cfg, opts.Now)
		effects := m.skillEffects(lifeType, cfg)
		chance := float64(recipe.Probability) + effects.successRate
		if chance > 1 {
			chance = 1
		}
		if lifeRandom(opts.RNG) >= chance {
			receipt.Proficiency = state.Proficiency
			receipt.Level = state.Level
			return receipt
		}
		costs := make([]LifeItem, 0, len(recipe.Materials)+1)
		if recipe.EnergyID != 0 || recipe.EnergyCount != 0 {
			costs = append(costs, LifeItem{ItemID: recipe.EnergyID, Count: recipe.EnergyCount})
		}
		for _, material := range recipe.Materials {
			count := material.Count
			if effects.reduceMaterials > 0 {
				count = int64(math.Ceil(float64(count) * (1 - effects.reduceMaterials)))
			}
			if count > 0 {
				costs = append(costs, LifeItem{ItemID: material.ItemID, Count: count})
			}
		}
		costs = mergeLifeItems(costs)
		outputCount := int64(1)
		if effects.extraChance > 0 && lifeRandom(opts.RNG) < effects.extraChance {
			outputCount++
		}
		outputs := tagLifeItems(cfg, []LifeItem{{ItemID: recipe.OutputID, Count: outputCount}})
		returns := make([]LifeItem, 0)
		if effects.returnChance > 0 && effects.returnCount > 0 && lifeRandom(opts.RNG) < effects.returnChance {
			for _, material := range recipe.Materials {
				count := effects.returnCount
				if count > material.Count {
					count = material.Count
				}
				if count > 0 {
					returns = append(returns, LifeItem{ItemID: material.ItemID, Count: count})
				}
			}
		}
		returns = tagLifeItems(cfg, returns)
		additions := append([]LifeItem(nil), outputs...)
		additions = append(additions, returns...)
		if effects.restoreEnergy > 0 {
			additions = append(additions, LifeItem{ItemID: recipe.EnergyID, Count: int64(effects.restoreEnergy)})
		}
		additions = tagLifeItems(cfg, mergeLifeItems(additions))
		tx, status := stageLifeCosts(items, cfg, costs)
		if status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		if status = validateLifeBalance(items, cfg, costs, additions); status != proto.StatusCode_StatusCode_Ok {
			tx.Rollback()
			receipt.Status = status
			return receipt
		}
		tx.Commit()
		applyLifeAdditions(items, cfg, additions)
		receipt.Success = true
		receipt.Persist = true
		receipt.Items = changedLifeItems(items, cfg, costs, additions)
		receipt.ResultItems = outputs
		receipt.SkillReturnItems = returns
		receipt.IsLuckyReturn = len(returns) > 0
		receipt.CostTime = reducedLifeTime(recipe.NeedTime, effects.timeReduction)
		receipt.Proficiency = addProficiency(state.Proficiency, level.ProficiencyPerAction)
		state.Proficiency = receipt.Proficiency
		for _, ref := range recipe.Achievements {
			m.AddAchievement(lifeType, ref.ID, uint32(outputCount))
			receipt.AchieveIDs = append(receipt.AchieveIDs, ref.ID)
		}
		receipt.Level = state.Level
		return receipt
	})
}

type FishingOptions struct {
	Token         string
	Now           int64
	RNG           LifeRNG
	PondID        uint32
	BoxID         uint32
	RodInstanceID uint32
	Persist       func() error
}

func (m *LifeModel) Fish(items *ItemModel, cfg *LifeConfig, opts FishingOptions) LifeReceipt {
	return m.doOnce(opts.Token, items, opts.Persist, func() LifeReceipt {
		receipt := LifeReceipt{Status: proto.StatusCode_StatusCode_Ok, LifeType: proto.LifeType_LifeType_Fishing}
		if cfg == nil || items == nil || opts.PondID == 0 || opts.BoxID == 0 || opts.RodInstanceID == 0 {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		rod := items.GetItemWeaponInfo(opts.RodInstanceID)
		if rod == nil {
			receipt.Status = proto.StatusCode_StatusCode_ItemNotEnough
			return receipt
		}
		weaponType, ok := cfg.WeaponTypes[rod.WeaponId]
		if !ok {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		if weaponType != proto.EWeaponType_EWeaponType_FishingRod {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		bait, status := lifeBaseItem(cfg, opts.BoxID)
		if status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		if proto.EBagItemTag(bait.NewBagItemTag) != proto.EBagItemTag_EBagItemTag_Baitbox {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		if info := items.GetItemBaseInfo(opts.BoxID); info == nil || info.Num < 1 {
			receipt.Status = proto.StatusCode_StatusCode_ItemNotEnough
			return receipt
		}
		state := m.GetState(proto.LifeType_LifeType_Fishing)
		level := cfg.Levels[proto.LifeType_LifeType_Fishing][state.Level]
		if level == nil || (level.ProficiencyPerAction == 0 && cfg.Levels[proto.LifeType_LifeType_Fishing][state.Level+1] != nil) {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		if !validLifeEnergy(cfg, level.ActionEnergyID, level.ActionEnergyCount) {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		receipt.Persist = m.RefreshSkills(proto.LifeType_LifeType_Fishing, cfg, opts.Now)
		effects := m.skillEffects(proto.LifeType_LifeType_Fishing, cfg)
		if lifeRandom(opts.RNG) >= 1+effects.successRate {
			receipt.Proficiency = state.Proficiency
			receipt.Level = state.Level
			return receipt
		}
		fishList := make([]*LifeFishConfig, 0)
		for _, fish := range cfg.Fish {
			if fish != nil && fish.IsShow && fish.Level <= state.Level {
				if _, itemStatus := lifeBaseItem(cfg, fish.ItemID); itemStatus == proto.StatusCode_StatusCode_Ok &&
					validLifeFishSize(fish.MinSize, fish.MaxSize) {
					fishList = append(fishList, fish)
				}
			}
		}
		if len(fishList) == 0 {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		sort.Slice(fishList, func(i, j int) bool { return fishList[i].ID < fishList[j].ID })
		fish := fishList[lifeIntn(opts.RNG, len(fishList))]
		outputCount := int64(1)
		if effects.extraChance > 0 && lifeRandom(opts.RNG) < effects.extraChance {
			outputCount++
		}
		costs := []LifeItem{{ItemID: opts.BoxID, Count: 1, Tag: proto.EBagItemTag(bait.NewBagItemTag)}}
		if level.ActionEnergyID != 0 || level.ActionEnergyCount != 0 {
			costs = append(costs, LifeItem{ItemID: level.ActionEnergyID, Count: level.ActionEnergyCount})
		}
		costs = mergeLifeItems(costs)
		outputs := tagLifeItems(cfg, []LifeItem{{ItemID: fish.ItemID, Count: outputCount}})
		tx, status := stageLifeCosts(items, cfg, costs)
		if status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		if status = validateLifeBalance(items, cfg, costs, outputs); status != proto.StatusCode_StatusCode_Ok {
			tx.Rollback()
			receipt.Status = status
			return receipt
		}
		tx.Commit()
		applyLifeAdditions(items, cfg, outputs)
		for i := int64(0); i < outputCount; i++ {
			length := fish.MinSize + (fish.MaxSize-fish.MinSize)*float32(lifeRandom(opts.RNG))
			receipt.Fish = append(receipt.Fish, LifeFishResult{ID: fish.ID, Length: length})
		}
		receipt.Success = true
		receipt.Persist = true
		receipt.Items = changedLifeItems(items, cfg, costs, outputs)
		receipt.ResultItems = outputs
		receipt.CostEndurance = uint32(level.ActionEnergyCount)
		receipt.Proficiency = addProficiency(state.Proficiency, level.ProficiencyPerAction)
		state.Proficiency = receipt.Proficiency
		for _, ref := range fish.Achievements {
			m.AddAchievement(proto.LifeType_LifeType_Fishing, ref.ID, uint32(outputCount))
			receipt.AchieveIDs = append(receipt.AchieveIDs, ref.ID)
		}
		receipt.Level = state.Level
		return receipt
	})
}

func optionalLifePersist(persist []func() error) func() error {
	if len(persist) == 0 {
		return nil
	}
	return persist[0]
}

func (m *LifeModel) LevelUp(items *ItemModel, cfg *LifeConfig, lifeType proto.LifeType, token string, persist ...func() error) LifeReceipt {
	return m.doOnce(token, items, optionalLifePersist(persist), func() LifeReceipt {
		receipt := LifeReceipt{Status: proto.StatusCode_StatusCode_Ok, LifeType: lifeType}
		if !ValidLifeType(lifeType) || cfg == nil || items == nil {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		state := m.GetState(lifeType)
		level := cfg.Levels[lifeType][state.Level]
		if level == nil {
			receipt.Status = proto.StatusCode_StatusCode_LifeMaxLv
			return receipt
		}
		if cfg.Levels[lifeType][state.Level+1] == nil {
			receipt.Status = proto.StatusCode_StatusCode_LifeMaxLv
			return receipt
		}
		if state.Proficiency < level.Proficiency {
			receipt.Status = proto.StatusCode_StatusCode_ProficiencyNotEnough
			return receipt
		}
		costs := make([]LifeItem, 0, 1)
		if level.CostItemID != 0 || level.CostItemCount != 0 {
			costs = append(costs, LifeItem{ItemID: level.CostItemID, Count: level.CostItemCount})
		}
		tx, status := stageLifeCosts(items, cfg, costs)
		if status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		tx.Commit()
		state.Level++
		receipt.Persist = true
		receipt.Level = state.Level
		receipt.Proficiency = state.Proficiency
		receipt.Items = changedLifeItems(items, cfg, costs, nil)
		return receipt
	})
}

func (m *LifeModel) StudySkill(items *ItemModel, cfg *LifeConfig, lifeType proto.LifeType, skillID uint32, force bool, token string, now int64, persist ...func() error) LifeReceipt {
	return m.doOnce(token, items, optionalLifePersist(persist), func() LifeReceipt {
		receipt := LifeReceipt{Status: proto.StatusCode_StatusCode_Ok, LifeType: lifeType}
		if !ValidLifeType(lifeType) || cfg == nil || items == nil {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		skillConf := cfg.Skills[lifeType][skillID]
		if skillConf == nil {
			receipt.Status = proto.StatusCode_StatusCode_PlayerLifeSkillNotExist
			return receipt
		}
		receipt.Persist = m.RefreshSkills(lifeType, cfg, now)
		state := m.GetState(lifeType)
		skill := state.Skills[skillID]
		if skill != nil && skill.StartStudyTime != 0 {
			receipt.Status = proto.StatusCode_StatusCode_IsLearning
			return receipt
		}
		current := uint32(0)
		if skill != nil {
			current = skill.Lv
		}
		level := skillConf.Levels[current+1]
		if level == nil {
			receipt.Status = proto.StatusCode_StatusCode_LifeSkillMaxLv
			return receipt
		}
		if level.NeedTime < 0 {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		if level.NeedLifeLevel > state.Level {
			receipt.Status = proto.StatusCode_StatusCode_LifeLvNotEnough
			return receipt
		}
		learning := 0
		for _, value := range state.Skills {
			if value != nil && value.StartStudyTime != 0 {
				learning++
			}
		}
		if learning >= 2 {
			receipt.Status = proto.StatusCode_StatusCode_CantLearningTwoSkill
			return receipt
		}
		costs := make([]LifeItem, 0, len(level.Materials)+1)
		if level.CurrencyID != 0 || level.CurrencyCount != 0 {
			costs = append(costs, LifeItem{ItemID: level.CurrencyID, Count: level.CurrencyCount})
		}
		for _, material := range level.Materials {
			costs = append(costs, LifeItem{ItemID: material.ItemID, Count: material.Count})
		}
		tx, status := stageLifeCosts(items, cfg, costs)
		if status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		tx.Commit()
		if skill == nil {
			skill = &LifeSkillState{}
			state.Skills[skillID] = skill
		}
		if level.NeedTime > 0 && !force {
			skill.StartStudyTime = now
			skill.ReadyAt = now + level.NeedTime
		} else {
			skill.Lv = current + 1
			skill.StartStudyTime = 0
			skill.ReadyAt = 0
		}
		receipt.Items = changedLifeItems(items, cfg, costs, nil)
		receipt.Persist = true
		receipt.Skills = m.skillSnapshots(lifeType, cfg)
		return receipt
	})
}

func (m *LifeModel) ClaimAchievement(items *ItemModel, cfg *LifeConfig, lifeType proto.LifeType, achieveID uint32, token string, rng LifeRNG, persist ...func() error) LifeReceipt {
	return m.doOnce(token, items, optionalLifePersist(persist), func() LifeReceipt {
		receipt := LifeReceipt{Status: proto.StatusCode_StatusCode_Ok, LifeType: lifeType}
		if !ValidLifeType(lifeType) || cfg == nil || items == nil || achieveID == 0 {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		conf := cfg.Achievements[achieveID]
		if conf == nil || conf.LifeType != lifeType || conf.Target == 0 || conf.RewardID == 0 {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		state := m.GetState(lifeType).Achievements[achieveID]
		if state == nil {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		if state.Rewarded {
			receipt.Status = proto.StatusCode_StatusCode_RepeatGetReward
			return receipt
		}
		if state.Count < uint64(conf.Target) {
			receipt.Status = proto.StatusCode_StatusCode_BadReq
			return receipt
		}
		rewards := cfg.Rewards[conf.RewardID]
		if len(rewards) == 0 {
			receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
			return receipt
		}
		additions := make([]LifeItem, 0, len(rewards))
		for _, reward := range rewards {
			if _, status := lifeBaseItem(cfg, reward.ItemID); status != proto.StatusCode_StatusCode_Ok {
				receipt.Status = status
				return receipt
			}
			if reward.Min <= 0 || reward.Max < reward.Min {
				receipt.Status = proto.StatusCode_StatusCode_CfgNotFound
				return receipt
			}
			count := reward.Min
			if reward.Max > reward.Min {
				count += int64(lifeIntn(rng, int(reward.Max-reward.Min)+1))
			}
			additions = append(additions, LifeItem{ItemID: reward.ItemID, Count: count})
		}
		additions = tagLifeItems(cfg, mergeLifeItems(additions))
		if status := validateLifeBalance(items, cfg, nil, additions); status != proto.StatusCode_StatusCode_Ok {
			receipt.Status = status
			return receipt
		}
		applyLifeAdditions(items, cfg, additions)
		state.Rewarded = true
		receipt.Persist = true
		receipt.ResultItems = additions
		receipt.Items = changedLifeItems(items, cfg, nil, additions)
		receipt.RewardedAchieveIDs = []uint32{achieveID}
		return receipt
	})
}

func (m *LifeModel) skillSnapshots(lifeType proto.LifeType, cfg *LifeConfig) []LifeSkillSnapshot {
	infos := m.SkillInfo(lifeType, cfg)
	result := make([]LifeSkillSnapshot, 0, len(infos))
	for _, info := range infos {
		result = append(result, LifeSkillSnapshot{SkillId: info.SkillId, Lv: info.Lv, StartStudyTime: info.StartStudyTime})
	}
	return result
}

type lifeEffects struct {
	successRate     float64
	extraChance     float64
	reduceMaterials float64
	returnChance    float64
	returnCount     int64
	restoreEnergy   float64
	timeReduction   float64
}

func (m *LifeModel) skillEffects(lifeType proto.LifeType, cfg *LifeConfig) lifeEffects {
	var result lifeEffects
	for id, state := range m.GetState(lifeType).Skills {
		if state == nil || state.Lv == 0 || state.StartStudyTime != 0 {
			continue
		}
		skill := cfg.Skills[lifeType][id]
		if skill == nil {
			continue
		}
		level := skill.Levels[state.Lv]
		if level == nil {
			continue
		}
		switch level.Effect {
		case proto.ELifeSpellEffectType_ELifeSpellEffectType_CookingSuccessRateAdd,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_FishingSuccessRateAdd,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_HandicraftSuccessRateAdd,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_MakeSuccessRateAdd:
			result.successRate += float64(level.Param1)
		case proto.ELifeSpellEffectType_ELifeSpellEffectType_FoodAddGains,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_FishingAddGains:
			result.extraChance += float64(level.Param1)
		case proto.ELifeSpellEffectType_ELifeSpellEffectType_CookReduceMaterials,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_MakeReduceMaterials,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_HandicraftReduceMaterials:
			result.reduceMaterials += float64(level.Param1)
		case proto.ELifeSpellEffectType_ELifeSpellEffectType_HandicraftReturnMaterials:
			result.returnChance += float64(level.Param1)
			if count := int64(level.Param2); count > result.returnCount {
				result.returnCount = count
			}
		case proto.ELifeSpellEffectType_ELifeSpellEffectType_HandicraftRestoreEnergy,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_MakeRestoreEnergy:
			result.restoreEnergy += float64(level.Param1)
		case proto.ELifeSpellEffectType_ELifeSpellEffectType_FoodTimeAdd,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_ReduceFishingTime,
			proto.ELifeSpellEffectType_ELifeSpellEffectType_HandicraftProcessRateAdd:
			result.timeReduction += float64(level.Param1)
		}
	}
	if math.IsNaN(result.successRate) || math.IsInf(result.successRate, 0) || result.successRate < 0 {
		result.successRate = 0
	}
	if math.IsNaN(result.extraChance) || math.IsInf(result.extraChance, 0) || result.extraChance < 0 {
		result.extraChance = 0
	} else if result.extraChance > 1 {
		result.extraChance = 1
	}
	if math.IsNaN(result.returnChance) || math.IsInf(result.returnChance, 0) || result.returnChance < 0 {
		result.returnChance = 0
	} else if result.returnChance > 1 {
		result.returnChance = 1
	}
	if math.IsNaN(result.restoreEnergy) || math.IsInf(result.restoreEnergy, 0) || result.restoreEnergy < 0 {
		result.restoreEnergy = 0
	}
	if math.IsNaN(result.reduceMaterials) || math.IsInf(result.reduceMaterials, 0) || result.reduceMaterials < 0 {
		result.reduceMaterials = 0
	} else if result.reduceMaterials > 1 {
		result.reduceMaterials = 1
	}
	if math.IsNaN(result.timeReduction) || math.IsInf(result.timeReduction, 0) || result.timeReduction < 0 {
		result.timeReduction = 0
	} else if result.timeReduction > 1 {
		result.timeReduction = 1
	}
	return result
}

func validLifeFishSize(min, max float32) bool {
	return !math.IsNaN(float64(min)) && !math.IsNaN(float64(max)) &&
		!math.IsInf(float64(min), 0) && !math.IsInf(float64(max), 0) && min >= 0 && max >= min
}

func reducedLifeTime(base int64, reduction float64) uint32 {
	if base <= 0 {
		return 0
	}
	value := int64(math.Ceil(float64(base) * (1 - reduction)))
	if value < 0 {
		return 0
	}
	return uint32(value)
}

func lifeTimeAvailable(opening, closing string, now int64) (bool, bool) {
	location := time.FixedZone("CST", 8*60*60)
	parse := func(value string) (int64, bool) {
		if value == "" {
			return 0, true
		}
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, location)
		return parsed.Unix(), err == nil
	}
	start, valid := parse(opening)
	if !valid {
		return false, false
	}
	end, valid := parse(closing)
	if !valid {
		return false, false
	}
	if start != 0 && now < start {
		return false, true
	}
	if end != 0 && now > end {
		return false, true
	}
	return true, true
}

func addProficiency(value, amount uint32) uint32 {
	if ^uint32(0)-value < amount {
		return ^uint32(0)
	}
	return value + amount
}

func lifeBaseItem(cfg *LifeConfig, id uint32) (*excel.ItemConfigure, proto.StatusCode) {
	if id == 0 || cfg == nil {
		return nil, proto.StatusCode_StatusCode_CfgNotFound
	}
	item := cfg.Items[id]
	if item == nil {
		return nil, proto.StatusCode_StatusCode_CfgNotFound
	}
	switch proto.EBagItemTag(item.NewBagItemTag) {
	case proto.EBagItemTag_EBagItemTag_AbilityItem,
		proto.EBagItemTag_EBagItemTag_Gift,
		proto.EBagItemTag_EBagItemTag_IntimacyGift,
		proto.EBagItemTag_EBagItemTag_Catcher,
		proto.EBagItemTag_EBagItemTag_Currency,
		proto.EBagItemTag_EBagItemTag_Material,
		proto.EBagItemTag_EBagItemTag_Food,
		proto.EBagItemTag_EBagItemTag_Fish,
		proto.EBagItemTag_EBagItemTag_Item,
		proto.EBagItemTag_EBagItemTag_Furniture,
		proto.EBagItemTag_EBagItemTag_Baitbox,
		proto.EBagItemTag_EBagItemTag_Quest,
		proto.EBagItemTag_EBagItemTag_DyeStuff,
		proto.EBagItemTag_EBagItemTag_StrengthStone,
		proto.EBagItemTag_EBagItemTag_ExpBook:
		return item, proto.StatusCode_StatusCode_Ok
	default:
		return nil, proto.StatusCode_StatusCode_CfgNotFound
	}
}

func validLifeItem(cfg *LifeConfig, id uint32) bool {
	_, status := lifeBaseItem(cfg, id)
	return status == proto.StatusCode_StatusCode_Ok
}

func validLifeEnergy(cfg *LifeConfig, id uint32, count int64) bool {
	if count < 0 || (id == 0) != (count == 0) {
		return false
	}
	return count == 0 || validLifeItem(cfg, id)
}

func stageLifeCosts(items *ItemModel, cfg *LifeConfig, costs []LifeItem) (*ItemTransaction, proto.StatusCode) {
	for _, cost := range costs {
		if cost.Count <= 0 {
			return nil, proto.StatusCode_StatusCode_CfgNotFound
		}
		if _, status := lifeBaseItem(cfg, cost.ItemID); status != proto.StatusCode_StatusCode_Ok {
			return nil, status
		}
	}
	tx, err := items.Begin()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_SysErr
	}
	for _, cost := range costs {
		tx.DelBaseItem(cost.ItemID, cost.Count)
		if tx.Error != nil {
			tx.Rollback()
			return nil, proto.StatusCode_StatusCode_ItemNotEnough
		}
	}
	return tx, proto.StatusCode_StatusCode_Ok
}

func validateLifeBalance(items *ItemModel, cfg *LifeConfig, costs, additions []LifeItem) proto.StatusCode {
	delta := make(map[uint32]int64)
	for _, list := range [][]LifeItem{costs, additions} {
		for _, item := range list {
			if item.Count <= 0 {
				return proto.StatusCode_StatusCode_CfgNotFound
			}
			if _, status := lifeBaseItem(cfg, item.ItemID); status != proto.StatusCode_StatusCode_Ok {
				return status
			}
			if item.Count > 0 && delta[item.ItemID] > math.MaxInt64-item.Count {
				return proto.StatusCode_StatusCode_SysErr
			}
			delta[item.ItemID] += item.Count
		}
	}
	for id, change := range delta {
		current := int64(0)
		if info := items.GetItemBaseInfo(id); info != nil {
			current = info.Num
		}
		if change > 0 && current > math.MaxInt64-change {
			return proto.StatusCode_StatusCode_SysErr
		}
		current += change
		if current < 0 {
			return proto.StatusCode_StatusCode_ItemNotEnough
		}
	}
	return proto.StatusCode_StatusCode_Ok
}

func applyLifeAdditions(items *ItemModel, cfg *LifeConfig, additions []LifeItem) {
	list := items.GetItemBaseMap()
	for _, addition := range additions {
		if addition.Count <= 0 {
			continue
		}
		info := list[addition.ItemID]
		if info == nil {
			item := cfg.Items[addition.ItemID]
			info = &ItemBaseInfo{
				ItemId: addition.ItemID, ItemType: proto.EBagItemTag(item.NewBagItemTag),
				PackType: proto.PackType_PackType_Inventory,
			}
			list[addition.ItemID] = info
		}
		info.Num += addition.Count
	}
}

func changedLifeItems(items *ItemModel, cfg *LifeConfig, costs, additions []LifeItem) []LifeItem {
	ids := make(map[uint32]struct{})
	for _, list := range [][]LifeItem{costs, additions} {
		for _, item := range list {
			ids[item.ItemID] = struct{}{}
		}
	}
	result := make([]LifeItem, 0, len(ids))
	for id := range ids {
		info := items.GetItemBaseInfo(id)
		if info == nil {
			continue
		}
		result = append(result, LifeItem{ItemID: id, Count: info.Num, Tag: info.ItemType})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ItemID < result[j].ItemID })
	return result
}

func tagLifeItems(cfg *LifeConfig, items []LifeItem) []LifeItem {
	for i := range items {
		if items[i].Tag != proto.EBagItemTag_EBagItemTag_None && cfg.Items[items[i].ItemID] != nil {
			continue
		}
		if item := cfg.Items[items[i].ItemID]; item != nil {
			items[i].Tag = proto.EBagItemTag(item.NewBagItemTag)
		}
	}
	return items
}

func mergeLifeItems(items []LifeItem) []LifeItem {
	byID := make(map[uint32]LifeItem)
	for _, item := range items {
		if item.Count == 0 {
			continue
		}
		current := byID[item.ItemID]
		current.ItemID = item.ItemID
		current.Tag = item.Tag
		current.Count += item.Count
		byID[item.ItemID] = current
	}
	result := make([]LifeItem, 0, len(byID))
	for _, item := range byID {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ItemID < result[j].ItemID })
	return result
}
