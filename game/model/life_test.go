package model

import (
	"errors"
	"math"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/config"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

type lifeTestRNG struct {
	floats []float64
	ints   []int
	calls  int
}

func (r *lifeTestRNG) Float64() float64 {
	r.calls++
	if len(r.floats) == 0 {
		return 0
	}
	value := r.floats[0]
	r.floats = r.floats[1:]
	return value
}

func (r *lifeTestRNG) IntN(n int) int {
	r.calls++
	if len(r.ints) == 0 {
		return 0
	}
	value := r.ints[0]
	r.ints = r.ints[1:]
	if value >= n {
		return n - 1
	}
	return value
}

func lifeTestConfig() *LifeConfig {
	cfg := &LifeConfig{
		Levels:       make(map[proto.LifeType]map[uint32]*LifeLevelConfig),
		Recipes:      make(map[proto.LifeType]map[uint32]*LifeRecipe),
		Fish:         make(map[uint32]*LifeFishConfig),
		Skills:       make(map[proto.LifeType]map[uint32]*LifeSkillConfig),
		Achievements: make(map[uint32]*LifeAchievementConfig),
		Rewards:      make(map[uint32][]LifeReward),
		Items: map[uint32]*excel.ItemConfigure{
			11:   {ID: 11, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
			100:  {ID: 100, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)},
			101:  {ID: 101, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
			200:  {ID: 200, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Food)},
			300:  {ID: 300, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Fish)},
			400:  {ID: 400, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Baitbox)},
			500:  {ID: 500, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Item)},
			600:  {ID: 600, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)},
			1006: {ID: 1006, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Weapon)},
		},
		WeaponTypes: map[uint32]proto.EWeaponType{1006: proto.EWeaponType_EWeaponType_FishingRod},
	}
	for _, lifeType := range []proto.LifeType{
		proto.LifeType_LIFE_TYPE_Cooking, proto.LifeType_LIFE_TYPE_Fishing,
		proto.LifeType_LIFE_TYPE_Making, proto.LifeType_LIFE_TYPE_Handicraft,
	} {
		cfg.Levels[lifeType] = make(map[uint32]*LifeLevelConfig)
		cfg.Recipes[lifeType] = make(map[uint32]*LifeRecipe)
		cfg.Skills[lifeType] = make(map[uint32]*LifeSkillConfig)
	}
	cfg.Levels[proto.LifeType_LIFE_TYPE_Cooking][1] = &LifeLevelConfig{
		Level: 1, Proficiency: 100, CostItemID: 101, CostItemCount: 5, ProficiencyPerAction: 10,
	}
	cfg.Levels[proto.LifeType_LIFE_TYPE_Cooking][2] = &LifeLevelConfig{
		Level: 2, Proficiency: 200, CostItemID: 101, CostItemCount: 5, ProficiencyPerAction: 10,
	}
	cfg.Levels[proto.LifeType_LIFE_TYPE_Fishing][1] = &LifeLevelConfig{
		Level: 1, Proficiency: 100, ActionEnergyID: 11, ActionEnergyCount: 2, ProficiencyPerAction: 10,
	}
	cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1] = &LifeRecipe{
		ID: 1, Level: 1, IsShow: true, NeedTime: 4, EnergyID: 11, EnergyCount: 2,
		Materials: []LifeMaterial{{ItemID: 100, Count: 3}}, Probability: 1, OutputID: 200,
		Achievements: []LifeAchievementRef{{ID: 7, RewardID: 70}},
	}
	cfg.Skills[proto.LifeType_LIFE_TYPE_Cooking][1000] = &LifeSkillConfig{
		ID: 1000,
		Levels: map[uint32]*LifeSkillLevelConfig{
			1: {Level: 1, Effect: proto.ELifeSpellEffectType_ELifeSpellEffectType_CookReduceMaterials, Param1: 0.5, NeedLifeLevel: 1, NeedTime: 10, CurrencyID: 101, CurrencyCount: 1},
			2: {Level: 2, Effect: proto.ELifeSpellEffectType_ELifeSpellEffectType_CookingSuccessRateAdd, Param1: 0.1, NeedLifeLevel: 1, CurrencyID: 101, CurrencyCount: 1},
		},
	}
	cfg.Fish[300] = &LifeFishConfig{
		ID: 300, Level: 1, ItemID: 300, IsShow: true, MinSize: 10, MaxSize: 20,
		Achievements: []LifeAchievementRef{{ID: 9, RewardID: 90}},
	}
	cfg.Achievements[7] = &LifeAchievementConfig{ID: 7, LifeType: proto.LifeType_LIFE_TYPE_Cooking, Target: 2, RewardID: 70}
	cfg.Achievements[9] = &LifeAchievementConfig{ID: 9, LifeType: proto.LifeType_LIFE_TYPE_Fishing, Target: 1, RewardID: 90}
	cfg.Rewards[70] = []LifeReward{{ItemID: 600, Min: 2, Max: 2}}
	cfg.Rewards[90] = []LifeReward{{ItemID: 600, Min: 1, Max: 1}}
	return cfg
}

func lifeTestItems() *ItemModel {
	return &ItemModel{ItemBaseInfo: map[uint32]*ItemBaseInfo{
		11:  {ItemId: 11, Num: 20, ItemType: proto.EBagItemTag_EBagItemTag_Currency, PackType: proto.ItemDetail_PackType_Inventory},
		100: {ItemId: 100, Num: 20, ItemType: proto.EBagItemTag_EBagItemTag_Material, PackType: proto.ItemDetail_PackType_Inventory},
		101: {ItemId: 101, Num: 20, ItemType: proto.EBagItemTag_EBagItemTag_Currency, PackType: proto.ItemDetail_PackType_Inventory},
		400: {ItemId: 400, Num: 5, ItemType: proto.EBagItemTag_EBagItemTag_Baitbox, PackType: proto.ItemDetail_PackType_Inventory},
		600: {ItemId: 600, Num: 0, ItemType: proto.EBagItemTag_EBagItemTag_Material, PackType: proto.ItemDetail_PackType_Inventory},
	}}
}

func lifeTestItemsWithRod() *ItemModel {
	items := lifeTestItems()
	items.ItemWeaponMap = map[uint32]*ItemWeaponInfo{
		7: {ItemBaseInfo: &ItemBaseInfo{ItemId: 1006, ItemType: proto.EBagItemTag_EBagItemTag_Weapon}, WeaponId: 1006, InstanceId: 7},
	}
	return items
}

func lifeNum(items *ItemModel, id uint32) int64 {
	info := items.GetItemBaseInfo(id)
	if info == nil {
		return 0
	}
	return info.Num
}

func TestLifeModelJSONRoundTrip(t *testing.T) {
	original := DefaultLifeModel()
	state := original.GetState(proto.LifeType_LIFE_TYPE_Cooking)
	state.Level = 3
	state.Proficiency = 321
	state.Skills[1000] = &LifeSkillState{Lv: 2}
	state.Achievements[7] = &LifeAchieveState{Count: 4, Rewarded: true}
	original.DoOnce("token", func() LifeReceipt {
		return LifeReceipt{Status: proto.StatusCode_StatusCode_OK, LifeType: proto.LifeType_LIFE_TYPE_Cooking, Success: true, Level: 3, Proficiency: 321}
	})
	data, err := sonic.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored LifeModel
	if err := sonic.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.normalize()
	got := restored.GetState(proto.LifeType_LIFE_TYPE_Cooking)
	if got.Level != 3 || got.Proficiency != 321 || got.Skills[1000].Lv != 2 || !got.Achievements[7].Rewarded {
		t.Fatalf("生活技能状态未恢复: %+v", got)
	}
	if receipt := restored.DoOnce("token", func() LifeReceipt { return LifeReceipt{Status: proto.StatusCode_StatusCode_FAIL} }); receipt.Status != proto.StatusCode_StatusCode_OK {
		t.Fatalf("幂等凭证未恢复: %+v", receipt)
	}
}

func TestProduceRecipeAtomicAndIdempotent(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	life := DefaultLifeModel()
	life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Skills[1000] = &LifeSkillState{Lv: 1}
	rng := &lifeTestRNG{floats: []float64{0}}
	receipt := life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Token: "cook-1", Now: 100, RNG: rng})
	if receipt.Status != proto.StatusCode_StatusCode_OK || !receipt.Success {
		t.Fatalf("生产失败: %+v", receipt)
	}
	if lifeNum(items, 11) != 18 || lifeNum(items, 100) != 18 || lifeNum(items, 200) != 1 {
		t.Fatalf("材料或产出数量错误: energy=%d material=%d output=%d", lifeNum(items, 11), lifeNum(items, 100), lifeNum(items, 200))
	}
	state := life.GetState(proto.LifeType_LIFE_TYPE_Cooking)
	if state.Proficiency != 10 || state.Achievements[7].Count != 1 {
		t.Fatalf("熟练度或成就进度错误: %+v", state)
	}
	if len(receipt.ResultItems) != 1 || receipt.ResultItems[0].Tag != proto.EBagItemTag_EBagItemTag_Food {
		t.Fatalf("产出物品标签错误: %+v", receipt.ResultItems)
	}
	calls := rng.calls
	duplicate := life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Token: "cook-1", Now: 100, RNG: rng})
	if duplicate.Status != receipt.Status || !duplicate.Success || rng.calls != calls {
		t.Fatalf("重复请求未命中幂等凭证: %+v", duplicate)
	}
	if lifeNum(items, 11) != 18 || lifeNum(items, 100) != 18 || lifeNum(items, 200) != 1 || state.Proficiency != 10 {
		t.Fatal("重复请求重复扣除或重复产出")
	}
}

func TestProduceRecipeFailureAndMissingConfigDoNotConsume(t *testing.T) {
	cfg := lifeTestConfig()
	cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].Probability = 0
	items := lifeTestItems()
	life := DefaultLifeModel()
	receipt := life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Token: "failed", Now: 100, RNG: &lifeTestRNG{}})
	if receipt.Status != proto.StatusCode_StatusCode_OK || receipt.Success {
		t.Fatalf("概率失败响应错误: %+v", receipt)
	}
	if lifeNum(items, 11) != 20 || lifeNum(items, 100) != 20 || lifeNum(items, 200) != 0 || life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency != 0 {
		t.Fatal("生产失败扣除了材料或增加了熟练度")
	}
	delete(cfg.Items, 200)
	missing := life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Token: "missing", Now: 100, RNG: &lifeTestRNG{}})
	if missing.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND {
		t.Fatalf("缺失产出配置未拒绝: %+v", missing)
	}
	if lifeNum(items, 11) != 20 || lifeNum(items, 100) != 20 || lifeNum(items, 200) != 0 {
		t.Fatal("配置缺失时扣除了材料")
	}
}

func TestProduceRecipeValidationBeforeCosts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*LifeConfig, *ItemModel)
		status proto.StatusCode
	}{
		{name: "level", mutate: func(cfg *LifeConfig, _ *ItemModel) { cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].Level = 2 }, status: proto.StatusCode_StatusCode_LifeLvNotEnough},
		{name: "campfire", mutate: func(cfg *LifeConfig, _ *ItemModel) { cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].ToolType = 2 }, status: proto.StatusCode_StatusCode_BAD_REQ},
		{name: "material", mutate: func(_ *LifeConfig, items *ItemModel) { items.ItemBaseInfo[100].Num = 1 }, status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH},
		{name: "formula", mutate: func(cfg *LifeConfig, _ *ItemModel) {
			cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].FormulaItemID = 500
		}, status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := lifeTestConfig()
			items := lifeTestItems()
			test.mutate(cfg, items)
			beforeEnergy, beforeMaterial := lifeNum(items, 11), lifeNum(items, 100)
			receipt := DefaultLifeModel().ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Token: test.name, Now: 100, RNG: &lifeTestRNG{}})
			if receipt.Status != test.status {
				t.Fatalf("状态错误: got=%v want=%v", receipt.Status, test.status)
			}
			if lifeNum(items, 11) != beforeEnergy || lifeNum(items, 100) != beforeMaterial || lifeNum(items, 200) != 0 {
				t.Fatal("校验失败时发生扣除")
			}
		})
	}
}

func TestLifeLevelUpUsesProficiencyAndCost(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	life := DefaultLifeModel()
	state := life.GetState(proto.LifeType_LIFE_TYPE_Cooking)
	state.Proficiency = 99
	receipt := life.LevelUp(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, "level-1")
	if receipt.Status != proto.StatusCode_StatusCode_ProficiencyNotEnough || state.Level != 1 || lifeNum(items, 101) != 20 {
		t.Fatalf("熟练度不足时错误升级: %+v", receipt)
	}
	state.Proficiency = 100
	receipt = life.LevelUp(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, "level-2")
	if receipt.Status != proto.StatusCode_StatusCode_OK || receipt.Level != 2 || lifeNum(items, 101) != 15 {
		t.Fatalf("升级失败: %+v", receipt)
	}
	duplicate := life.LevelUp(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, "level-2")
	if duplicate.Level != 2 || lifeNum(items, 101) != 15 {
		t.Fatal("重复升级请求重复扣费")
	}
	receipt = life.LevelUp(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, "level-3")
	if receipt.Status != proto.StatusCode_StatusCode_LifeMaxLv || state.Level != 2 {
		t.Fatalf("最高等级处理错误: %+v", receipt)
	}
}

func TestLifeSkillLearningAndCompletion(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	life := DefaultLifeModel()
	receipt := life.StudySkill(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1000, false, "skill-1", 100)
	if receipt.Status != proto.StatusCode_StatusCode_OK || lifeNum(items, 101) != 19 {
		t.Fatalf("学习技能失败: %+v", receipt)
	}
	state := life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Skills[1000]
	if state.Lv != 0 || state.StartStudyTime != 100 || state.ReadyAt != 110 {
		t.Fatalf("学习状态错误: %+v", state)
	}
	duplicate := life.StudySkill(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1000, false, "skill-1", 100)
	if duplicate.Status != proto.StatusCode_StatusCode_OK || lifeNum(items, 101) != 19 {
		t.Fatal("重复学习请求重复扣费")
	}
	receipt = life.StudySkill(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1000, false, "skill-2", 105)
	if receipt.Status != proto.StatusCode_StatusCode_IsLearning || lifeNum(items, 101) != 19 {
		t.Fatalf("学习未完成时处理错误: %+v", receipt)
	}
	if !life.RefreshSkills(proto.LifeType_LIFE_TYPE_Cooking, cfg, 110) || state.Lv != 1 || state.StartStudyTime != 0 {
		t.Fatalf("技能到时未完成: %+v", state)
	}
	receipt = life.StudySkill(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1000, true, "skill-3", 110)
	if receipt.Status != proto.StatusCode_StatusCode_OK || state.Lv != 2 || lifeNum(items, 101) != 18 {
		t.Fatalf("强制升级失败: %+v", receipt)
	}
}

func TestLifeAchievementRewardOnce(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	life := DefaultLifeModel()
	life.AddAchievement(proto.LifeType_LIFE_TYPE_Cooking, 7, 2)
	receipt := life.ClaimAchievement(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 7, "reward-1", &lifeTestRNG{})
	if receipt.Status != proto.StatusCode_StatusCode_OK || lifeNum(items, 600) != 2 {
		t.Fatalf("成就领奖失败: %+v", receipt)
	}
	duplicate := life.ClaimAchievement(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 7, "reward-1", &lifeTestRNG{})
	if duplicate.Status != proto.StatusCode_StatusCode_OK || lifeNum(items, 600) != 2 {
		t.Fatal("重复领奖请求重复发奖")
	}
	receipt = life.ClaimAchievement(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 7, "reward-2", &lifeTestRNG{})
	if receipt.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD || lifeNum(items, 600) != 2 {
		t.Fatalf("重复领奖未拒绝: %+v", receipt)
	}
	delete(cfg.Rewards, 70)
	life2 := DefaultLifeModel()
	life2.AddAchievement(proto.LifeType_LIFE_TYPE_Cooking, 7, 2)
	receipt = life2.ClaimAchievement(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 7, "reward-missing", &lifeTestRNG{})
	if receipt.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || life2.GetState(proto.LifeType_LIFE_TYPE_Cooking).Achievements[7].Rewarded || lifeNum(items, 600) != 2 {
		t.Fatalf("奖励配置缺失时错误发奖: %+v", receipt)
	}
}

func TestFishingValidatesEquipmentAndBait(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItemsWithRod()
	life := DefaultLifeModel()
	receipt := life.Fish(items, cfg, FishingOptions{Token: "fish-1", Now: 100, PondID: 1, BoxID: 400, RodInstanceID: 7, RNG: &lifeTestRNG{floats: []float64{0, 0, 0}}})
	if receipt.Status != proto.StatusCode_StatusCode_OK || !receipt.Success || len(receipt.Fish) != 1 || receipt.Fish[0].Length != 10 {
		t.Fatalf("钓鱼失败: %+v", receipt)
	}
	if lifeNum(items, 11) != 18 || lifeNum(items, 400) != 4 || lifeNum(items, 300) != 1 || life.GetState(proto.LifeType_LIFE_TYPE_Fishing).Proficiency != 10 {
		t.Fatal("钓鱼消耗或产出错误")
	}
	duplicate := life.Fish(items, cfg, FishingOptions{Token: "fish-1", Now: 100, PondID: 1, BoxID: 400, RodInstanceID: 7, RNG: &lifeTestRNG{}})
	if !duplicate.Success || lifeNum(items, 11) != 18 || lifeNum(items, 400) != 4 || lifeNum(items, 300) != 1 {
		t.Fatal("重复钓鱼请求重复执行")
	}
	cfg.WeaponTypes[1006] = proto.EWeaponType_EWeaponType_Sword
	beforeEnergy := lifeNum(items, 11)
	receipt = life.Fish(items, cfg, FishingOptions{Token: "fish-bad-rod", Now: 100, PondID: 1, BoxID: 400, RodInstanceID: 7, RNG: &lifeTestRNG{}})
	if receipt.Status != proto.StatusCode_StatusCode_BAD_REQ || lifeNum(items, 11) != beforeEnergy {
		t.Fatalf("错误鱼竿未拒绝: %+v", receipt)
	}
	cfg.WeaponTypes[1006] = proto.EWeaponType_EWeaponType_FishingRod
	cfg.Items[400].NewBagItemTag = int32(proto.EBagItemTag_EBagItemTag_Currency)
	receipt = life.Fish(items, cfg, FishingOptions{Token: "fish-bad-bait", Now: 100, PondID: 1, BoxID: 400, RodInstanceID: 7, RNG: &lifeTestRNG{}})
	if receipt.Status != proto.StatusCode_StatusCode_BAD_REQ || lifeNum(items, 11) != beforeEnergy {
		t.Fatalf("错误鱼饵未拒绝: %+v", receipt)
	}
}

func TestLifeRequestTokenScopesSessionsAndOperations(t *testing.T) {
	life := DefaultLifeModel()
	first := life.RequestToken("session-a", 1, 10, 20, "produce")
	if again := life.RequestToken("session-a", 1, 10, 20, "produce"); again != first {
		t.Fatal("同一请求token不稳定")
	}
	if other := life.RequestToken("session-a", 1, 10, 21, "produce"); other == first {
		t.Fatal("不同生产ID共用token")
	}
	if reconnect := life.RequestToken("session-b", 0, 10, 20, "produce"); reconnect == first {
		t.Fatal("重连后的请求复用旧token")
	}
}

func TestLifeTimeAvailable(t *testing.T) {
	now := int64(100)
	if ok, valid := lifeTimeAvailable("1970-01-01 08:00:00", "1970-01-01 08:02:00", now); !ok || !valid {
		t.Fatal("有效时间窗口被拒绝")
	}
	if ok, valid := lifeTimeAvailable("1970-01-01 08:02:00", "", now); ok || !valid {
		t.Fatal("未开放时间窗口被接受")
	}
	if _, valid := lifeTimeAvailable("bad", "", now); valid {
		t.Fatal("非法时间配置被接受")
	}
}

func TestLifeConfigIntegration(t *testing.T) {
	resourcePath := os.Getenv("LOLO_LIFE_RESOURCE")
	dataPath := os.Getenv("LOLO_LIFE_DATA")
	if resourcePath == "" || dataPath == "" {
		t.Skip("设置 LOLO_LIFE_RESOURCE 和 LOLO_LIFE_DATA 后运行真实配置集成测试")
	}
	config.DefaultConfig.Resources.ResourcePath = resourcePath
	config.DefaultConfig.Resources.DataPath = dataPath
	log.NewApp()
	defer log.Close()
	gdconf.LoadGameConfig()
	cfg, err := LoadLifeConfig(resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, lifeType := range []proto.LifeType{
		proto.LifeType_LIFE_TYPE_Cooking, proto.LifeType_LIFE_TYPE_Fishing,
		proto.LifeType_LIFE_TYPE_Making, proto.LifeType_LIFE_TYPE_Handicraft,
	} {
		if len(cfg.Levels[lifeType]) != 10 || len(cfg.Skills[lifeType]) == 0 {
			t.Fatalf("真实生活配置不完整 type=%v levels=%d skills=%d", lifeType, len(cfg.Levels[lifeType]), len(cfg.Skills[lifeType]))
		}
	}
	for _, lifeType := range []proto.LifeType{proto.LifeType_LIFE_TYPE_Cooking, proto.LifeType_LIFE_TYPE_Making, proto.LifeType_LIFE_TYPE_Handicraft} {
		for id, recipe := range cfg.Recipes[lifeType] {
			if !recipe.IsShow {
				continue
			}
			for _, itemID := range []uint32{recipe.EnergyID, recipe.OutputID, recipe.FormulaItemID} {
				if itemID == 0 {
					continue
				}
				if _, status := lifeBaseItem(cfg, itemID); status != proto.StatusCode_StatusCode_OK {
					t.Fatalf("配方物品配置非法 type=%v id=%d item=%d status=%v", lifeType, id, itemID, status)
				}
			}
			for _, material := range recipe.Materials {
				if _, status := lifeBaseItem(cfg, material.ItemID); status != proto.StatusCode_StatusCode_OK {
					t.Fatalf("配方材料配置非法 type=%v id=%d item=%d status=%v", lifeType, id, material.ItemID, status)
				}
			}
		}
	}
	for id, fish := range cfg.Fish {
		if fish.IsShow {
			if _, status := lifeBaseItem(cfg, fish.ItemID); status != proto.StatusCode_StatusCode_OK {
				t.Fatalf("鱼类物品配置非法 id=%d item=%d status=%v", id, fish.ItemID, status)
			}
		}
	}
	for id, achievement := range cfg.Achievements {
		if achievement.RewardID == 0 {
			continue
		}
		rewards := cfg.Rewards[achievement.RewardID]
		if len(rewards) == 0 {
			t.Fatalf("成就奖励配置缺失 id=%d reward=%d", id, achievement.RewardID)
		}
		for _, reward := range rewards {
			if _, status := lifeBaseItem(cfg, reward.ItemID); status != proto.StatusCode_StatusCode_OK || reward.Min <= 0 || reward.Max < reward.Min {
				t.Fatalf("成就奖励非法 id=%d reward=%+v", id, reward)
			}
		}
	}
	for _, lifeType := range []proto.LifeType{
		proto.LifeType_LIFE_TYPE_Cooking, proto.LifeType_LIFE_TYPE_Fishing,
		proto.LifeType_LIFE_TYPE_Making, proto.LifeType_LIFE_TYPE_Handicraft,
	} {
		skillIDs := make([]uint32, 0, len(cfg.Skills[lifeType]))
		for id := range cfg.Skills[lifeType] {
			skillIDs = append(skillIDs, id)
		}
		sort.Slice(skillIDs, func(i, j int) bool { return skillIDs[i] < skillIDs[j] })
		if len(skillIDs) == 0 || cfg.Skills[lifeType][skillIDs[0]].Levels[1] == nil {
			t.Fatalf("真实技能一级配置缺失 type=%v", lifeType)
		}
		skillID := skillIDs[0]
		level := cfg.Skills[lifeType][skillID].Levels[1]
		items := &ItemModel{ItemBaseInfo: make(map[uint32]*ItemBaseInfo)}
		addIntegrationItem(items, cfg, level.CurrencyID, level.CurrencyCount)
		for _, material := range level.Materials {
			addIntegrationItem(items, cfg, material.ItemID, material.Count)
		}
		life := DefaultLifeModel()
		life.GetState(lifeType).Level = level.NeedLifeLevel
		now := time.Now().Unix()
		receipt := life.StudySkill(items, cfg, lifeType, skillID, false, "integration-skill", now)
		if receipt.Status != proto.StatusCode_StatusCode_OK {
			t.Fatalf("真实技能学习失败 type=%v skill=%d receipt=%+v", lifeType, skillID, receipt)
		}
		skill := life.GetState(lifeType).Skills[skillID]
		if level.NeedTime > 0 {
			if skill.Lv != 0 || skill.StartStudyTime != now || skill.ReadyAt != now+level.NeedTime {
				t.Fatalf("真实技能计时状态错误 type=%v skill=%+v", lifeType, skill)
			}
			life.RefreshSkills(lifeType, cfg, skill.ReadyAt)
		}
		if skill.Lv != 1 || skill.StartStudyTime != 0 {
			t.Fatalf("真实技能未完成 type=%v skill=%+v", lifeType, skill)
		}
	}
	for _, test := range []struct {
		lifeType proto.LifeType
		recipeID uint32
	}{
		{proto.LifeType_LIFE_TYPE_Cooking, 5012001},
		{proto.LifeType_LIFE_TYPE_Making, 5041001},
		{proto.LifeType_LIFE_TYPE_Handicraft, 5052001},
	} {
		items := &ItemModel{ItemBaseInfo: make(map[uint32]*ItemBaseInfo)}
		recipe := cfg.Recipes[test.lifeType][test.recipeID]
		if recipe == nil {
			t.Fatalf("真实配方缺失 type=%v id=%d", test.lifeType, test.recipeID)
		}
		addIntegrationItem(items, cfg, recipe.EnergyID, recipe.EnergyCount)
		for _, material := range recipe.Materials {
			addIntegrationItem(items, cfg, material.ItemID, material.Count)
		}
		if recipe.FormulaItemID != 0 {
			addIntegrationItem(items, cfg, recipe.FormulaItemID, 1)
		}
		receipt := DefaultLifeModel().ProduceRecipe(items, cfg, test.lifeType, test.recipeID, LifeProductionOptions{
			Token: "integration", Now: time.Now().Unix(), RNG: &lifeTestRNG{floats: []float64{0}}, HasCampFire: true,
		})
		expectedOutput := int64(1)
		if recipe.FormulaItemID == recipe.OutputID {
			expectedOutput = 2
		}
		if receipt.Status != proto.StatusCode_StatusCode_OK || !receipt.Success || lifeNum(items, recipe.OutputID) != expectedOutput {
			t.Fatalf("真实配方生产失败 type=%v id=%d receipt=%+v", test.lifeType, test.recipeID, receipt)
		}
	}
	achievement := cfg.Achievements[150120011]
	if achievement == nil || achievement.Target == 0 {
		t.Fatal("真实烹饪成就配置缺失")
	}
	rewardLife := DefaultLifeModel()
	rewardLife.AddAchievement(proto.LifeType_LIFE_TYPE_Cooking, achievement.ID, achievement.Target)
	rewardItems := &ItemModel{ItemBaseInfo: make(map[uint32]*ItemBaseInfo)}
	rewardReceipt := rewardLife.ClaimAchievement(rewardItems, cfg, proto.LifeType_LIFE_TYPE_Cooking, achievement.ID, "integration-reward", &lifeTestRNG{})
	if rewardReceipt.Status != proto.StatusCode_StatusCode_OK || len(rewardReceipt.ResultItems) == 0 {
		t.Fatalf("真实成就领奖失败: %+v", rewardReceipt)
	}
	items := &ItemModel{ItemBaseInfo: make(map[uint32]*ItemBaseInfo)}
	addIntegrationItem(items, cfg, 11, 10)
	addIntegrationItem(items, cfg, 6021, 1)
	items.ItemWeaponMap = map[uint32]*ItemWeaponInfo{
		1: {ItemBaseInfo: &ItemBaseInfo{ItemId: 1006001, ItemType: proto.EBagItemTag_EBagItemTag_Weapon}, WeaponId: 1006001, InstanceId: 1},
	}
	receipt := DefaultLifeModel().Fish(items, cfg, FishingOptions{
		Token: "integration-fish", Now: time.Now().Unix(), RNG: &lifeTestRNG{floats: []float64{0, 0, 0}}, PondID: 1, BoxID: 6021, RodInstanceID: 1,
	})
	if receipt.Status != proto.StatusCode_StatusCode_OK || !receipt.Success || len(receipt.Fish) != 1 {
		t.Fatalf("真实钓鱼配置生产失败: %+v", receipt)
	}
}

func addIntegrationItem(items *ItemModel, cfg *LifeConfig, id uint32, count int64) {
	if id == 0 || count <= 0 {
		return
	}
	conf := cfg.Items[id]
	if conf == nil {
		return
	}
	items.ItemBaseInfo[id] = &ItemBaseInfo{
		ItemId: id, Num: count, ItemType: proto.EBagItemTag(conf.NewBagItemTag), PackType: proto.ItemDetail_PackType_Inventory,
	}
}

func TestLifePersistenceFailureRollsBackAndAllowsRetry(t *testing.T) {
	tests := []struct {
		name string
		run  func(*LifeModel, *ItemModel, *LifeConfig, func() error) LifeReceipt
	}{
		{name: "produce", run: func(life *LifeModel, items *ItemModel, cfg *LifeConfig, save func() error) LifeReceipt {
			return life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Token: "retry", Now: 100, RNG: &lifeTestRNG{}, Persist: save})
		}},
		{name: "fish", run: func(life *LifeModel, items *ItemModel, cfg *LifeConfig, save func() error) LifeReceipt {
			return life.Fish(items, cfg, FishingOptions{Token: "retry", Now: 100, PondID: 1, BoxID: 400, RodInstanceID: 7, RNG: &lifeTestRNG{}, Persist: save})
		}},
		{name: "level", run: func(life *LifeModel, items *ItemModel, cfg *LifeConfig, save func() error) LifeReceipt {
			if life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Level == 1 {
				life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency = 100
			}
			return life.LevelUp(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, "retry", save)
		}},
		{name: "study", run: func(life *LifeModel, items *ItemModel, cfg *LifeConfig, save func() error) LifeReceipt {
			return life.StudySkill(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1000, false, "retry", 100, save)
		}},
		{name: "achievement", run: func(life *LifeModel, items *ItemModel, cfg *LifeConfig, save func() error) LifeReceipt {
			if life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Achievements[7] == nil {
				life.AddAchievement(proto.LifeType_LIFE_TYPE_Cooking, 7, 2)
			}
			return life.ClaimAchievement(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 7, "retry", &lifeTestRNG{}, save)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := lifeTestConfig()
			items := lifeTestItems()
			if test.name == "fish" {
				items = lifeTestItemsWithRod()
			}
			life := DefaultLifeModel()
			if test.name == "level" || test.name == "achievement" {
				if test.name == "level" {
					life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency = 100
				} else {
					life.AddAchievement(proto.LifeType_LIFE_TYPE_Cooking, 7, 2)
				}
			}
			beforeLife := cloneLifeStates(life.States)
			beforeItems := cloneLifeItemInfos(items.ItemBaseInfo)
			fail := true
			saves := 0
			save := func() error {
				saves++
				if _, ok := life.Receipts["retry"]; !ok {
					t.Error("落库快照缺少本次请求凭证")
				}
				if fail {
					return errors.New("save failed")
				}
				return nil
			}
			receipt := test.run(life, items, cfg, save)
			if receipt.Status != proto.StatusCode_StatusCode_SYS_ERR || receipt.Success {
				t.Fatalf("保存失败未回报系统错误: %+v", receipt)
			}
			if !reflect.DeepEqual(life.States, beforeLife) || !reflect.DeepEqual(items.ItemBaseInfo, beforeItems) {
				t.Fatalf("保存失败后内存未回滚: life=%+v items=%+v", life.States, items.ItemBaseInfo)
			}
			if _, ok := life.Receipts["retry"]; ok {
				t.Fatal("保存失败的请求被缓存为成功凭证")
			}
			fail = false
			receipt = test.run(life, items, cfg, save)
			if receipt.Status != proto.StatusCode_StatusCode_OK {
				t.Fatalf("回滚后同一请求无法重试: %+v", receipt)
			}
			if saves != 2 {
				t.Fatalf("保存调用次数错误: %d", saves)
			}
			beforeItems = cloneLifeItemInfos(items.ItemBaseInfo)
			beforeLife = cloneLifeStates(life.States)
			receipt = test.run(life, items, cfg, save)
			if receipt.Status != proto.StatusCode_StatusCode_OK || saves != 2 ||
				!reflect.DeepEqual(life.States, beforeLife) || !reflect.DeepEqual(items.ItemBaseInfo, beforeItems) {
				t.Fatal("重试成功后的并发重放未命中凭证")
			}
		})
	}
}

func TestLifeFailedProductionStillPersistsSkillCompletion(t *testing.T) {
	cfg := lifeTestConfig()
	cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].Probability = 0
	items := lifeTestItems()
	life := DefaultLifeModel()
	skill := &LifeSkillState{StartStudyTime: 100, ReadyAt: 110}
	life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Skills[1000] = skill
	saves := 0
	receipt := life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{
		Token: "failed-with-skill", Now: 110, RNG: &lifeTestRNG{},
		Persist: func() error {
			saves++
			if skill.Lv != 1 {
				t.Error("保存时技能尚未完成")
			}
			return nil
		},
	})
	if receipt.Status != proto.StatusCode_StatusCode_OK || receipt.Success || saves != 1 || skill.Lv != 1 ||
		lifeNum(items, 11) != 20 || lifeNum(items, 100) != 20 {
		t.Fatalf("生产失败时的技能完成未正确保存: receipt=%+v saves=%d skill=%+v", receipt, saves, skill)
	}
}

func TestLifeReceiptIsIncludedInPersistenceSnapshot(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	life := DefaultLifeModel()
	var saved LifeModel
	receipt := life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{
		Token: "persisted", Now: 100, RNG: &lifeTestRNG{},
		Persist: func() error {
			data, err := sonic.Marshal(life)
			if err != nil {
				return err
			}
			return sonic.Unmarshal(data, &saved)
		},
	})
	if receipt.Status != proto.StatusCode_StatusCode_OK || !receipt.Success {
		t.Fatalf("生产失败: %+v", receipt)
	}
	saves := 0
	replay := saved.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{
		Token: "persisted", Now: 100, RNG: &lifeTestRNG{},
		Persist: func() error { saves++; return nil },
	})
	if replay.Status != proto.StatusCode_StatusCode_OK || !replay.Success || saves != 0 ||
		lifeNum(items, 11) != 18 || lifeNum(items, 100) != 17 || lifeNum(items, 200) != 1 {
		t.Fatalf("落库凭证无法在重启后阻止重放: receipt=%+v saves=%d", replay, saves)
	}
}

func TestLifeConcurrentReplayAndTokenGeneration(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	life := DefaultLifeModel()
	const count = 32
	start := make(chan struct{})
	receipts := make(chan LifeReceipt, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			receipts <- life.ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{
				Token: "concurrent", Now: 100, RNG: &lifeTestRNG{},
			})
		}()
	}
	close(start)
	wg.Wait()
	close(receipts)
	for receipt := range receipts {
		if receipt.Status != proto.StatusCode_StatusCode_OK || !receipt.Success {
			t.Fatalf("并发重放返回错误: %+v", receipt)
		}
	}
	if lifeNum(items, 11) != 18 || lifeNum(items, 100) != 17 || lifeNum(items, 200) != 1 || life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency != 10 {
		t.Fatalf("并发重放重复扣除、产出或增加熟练度: energy=%d material=%d output=%d proficiency=%d receipts=%d",
			lifeNum(items, 11), lifeNum(items, 100), lifeNum(items, 200), life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency, len(life.Receipts))
	}

	life = DefaultLifeModel()
	tokens := make(chan string, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(seq uint32) {
			defer wg.Done()
			tokens <- life.RequestToken("session", seq, 10, 20, "produce")
		}(uint32(i))
	}
	wg.Wait()
	close(tokens)
	seen := make(map[string]bool)
	for token := range tokens {
		if seen[token] {
			t.Fatalf("不同封包序号生成了相同token: %s", token)
		}
		seen[token] = true
	}
}

func TestLifeSkillRefreshSaveFailureRollsBack(t *testing.T) {
	cfg := lifeTestConfig()
	life := DefaultLifeModel()
	skill := &LifeSkillState{StartStudyTime: 100, ReadyAt: 110}
	life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Skills[1000] = skill
	changed, err := life.RefreshSkillsAndSave(proto.LifeType_LIFE_TYPE_Cooking, cfg, 110, func() error {
		return errors.New("save failed")
	})
	skill = life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Skills[1000]
	if err == nil || changed || skill.Lv != 0 || skill.StartStudyTime != 100 || skill.ReadyAt != 110 {
		t.Fatalf("技能完成保存失败后未回滚: changed=%v err=%v skill=%+v", changed, err, skill)
	}
	changed, err = life.RefreshSkillsAndSave(proto.LifeType_LIFE_TYPE_Cooking, cfg, 110, func() error { return nil })
	if err != nil || !changed || skill.Lv != 1 || skill.StartStudyTime != 0 || skill.ReadyAt != 0 {
		t.Fatalf("技能完成重试失败: changed=%v err=%v skill=%+v", changed, err, skill)
	}
}

func TestLifeRejectsInvalidRewardAndProductionConfig(t *testing.T) {
	t.Run("nan probability", func(t *testing.T) {
		cfg := lifeTestConfig()
		cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].Probability = float32(math.NaN())
		items := lifeTestItems()
		receipt := DefaultLifeModel().ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Now: 100, RNG: &lifeTestRNG{}})
		if receipt.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || lifeNum(items, 11) != 20 || lifeNum(items, 100) != 20 {
			t.Fatalf("非法概率未拒绝: %+v", receipt)
		}
	})
	t.Run("invalid energy", func(t *testing.T) {
		cfg := lifeTestConfig()
		cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking][1].EnergyCount = -1
		items := lifeTestItems()
		receipt := DefaultLifeModel().ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Now: 100, RNG: &lifeTestRNG{}})
		if receipt.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || lifeNum(items, 100) != 20 {
			t.Fatalf("非法精力配置未拒绝: %+v", receipt)
		}
	})
	t.Run("invalid fish size", func(t *testing.T) {
		cfg := lifeTestConfig()
		cfg.Fish[300].MaxSize = 5
		items := lifeTestItemsWithRod()
		receipt := DefaultLifeModel().Fish(items, cfg, FishingOptions{Now: 100, PondID: 1, BoxID: 400, RodInstanceID: 7, RNG: &lifeTestRNG{}})
		if receipt.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || lifeNum(items, 11) != 20 || lifeNum(items, 400) != 5 {
			t.Fatalf("非法鱼体型配置未拒绝: %+v", receipt)
		}
	})
	t.Run("invalid reward", func(t *testing.T) {
		cfg := lifeTestConfig()
		cfg.Rewards[70][0].Min = 0
		items := lifeTestItems()
		life := DefaultLifeModel()
		life.AddAchievement(proto.LifeType_LIFE_TYPE_Cooking, 7, 2)
		receipt := life.ClaimAchievement(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 7, "invalid-reward", &lifeTestRNG{})
		if receipt.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || lifeNum(items, 600) != 0 || life.GetState(proto.LifeType_LIFE_TYPE_Cooking).Achievements[7].Rewarded {
			t.Fatalf("非法奖励配置未拒绝: %+v", receipt)
		}
	})
}

func TestLifeOutputOverflowDoesNotConsume(t *testing.T) {
	cfg := lifeTestConfig()
	items := lifeTestItems()
	items.ItemBaseInfo[200] = &ItemBaseInfo{ItemId: 200, Num: math.MaxInt64, ItemType: proto.EBagItemTag_EBagItemTag_Food}
	receipt := DefaultLifeModel().ProduceRecipe(items, cfg, proto.LifeType_LIFE_TYPE_Cooking, 1, LifeProductionOptions{Now: 100, RNG: &lifeTestRNG{}})
	if receipt.Status != proto.StatusCode_StatusCode_SYS_ERR || lifeNum(items, 11) != 20 || lifeNum(items, 100) != 20 || lifeNum(items, 200) != math.MaxInt64 {
		t.Fatalf("产出溢出时未保持原子性: %+v", receipt)
	}
}
