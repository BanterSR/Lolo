package gdconf

import (
	"math"

	"gucooing/lolo/protocol/excel"
)

const GatherTypeCount = 6

type Gather struct {
	all          *excel.AllGatherDatas
	sceneAll     *excel.AllSceneDatas
	Gathers      map[uint32]*excel.GatherConfigure
	GatherReward map[int32]*excel.GatherRewardConfigure
	GatherRule   map[int32]*excel.GatherRuleConfigure
	Scenes       map[int32]*excel.SceneConfigure
}

func (g *GameConfig) loadGather() {
	info := &Gather{
		all:          new(excel.AllGatherDatas),
		sceneAll:     new(excel.AllSceneDatas),
		Gathers:      make(map[uint32]*excel.GatherConfigure),
		GatherReward: make(map[int32]*excel.GatherRewardConfigure),
		GatherRule:   make(map[int32]*excel.GatherRuleConfigure),
		Scenes:       make(map[int32]*excel.SceneConfigure),
	}
	g.Excel.Gather = info
	ReadJson(g.excelPath, "Gather.json", &info.all)
	// 场景的 GatherRuleID 位于 Scene.json，而非 Config/ScenesConfigAsset.json。
	ReadJson(g.excelPath, "Scene.json", &info.sceneAll)

	for _, v := range info.all.GetGather().GetDatas() {
		if v != nil {
			info.Gathers[uint32(v.ID)] = v
		}
	}
	for _, v := range info.all.GetGatherReward().GetDatas() {
		if v != nil {
			info.GatherReward[v.ID] = v
		}
	}
	for _, v := range info.all.GetGatherRule().GetDatas() {
		if v != nil {
			info.GatherRule[v.ID] = v
		}
	}
	for _, v := range info.sceneAll.GetScene().GetDatas() {
		if v != nil {
			info.Scenes[v.ID] = v
		}
	}
}

func GetGatherConfigure(id uint32) *excel.GatherConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Gather == nil {
		return nil
	}
	return cc.Excel.Gather.Gathers[id]
}

func GetGatherRewardConfigure(id int32) *excel.GatherRewardConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Gather == nil {
		return nil
	}
	return cc.Excel.Gather.GatherReward[id]
}

func GetGatherRuleConfigure(id int32) *excel.GatherRuleConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Gather == nil {
		return nil
	}
	return cc.Excel.Gather.GatherRule[id]
}

func GetSceneGatherRuleConfigure(sceneId uint32) *excel.GatherRuleConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Gather == nil || sceneId > math.MaxInt32 {
		return nil
	}
	scene := cc.Excel.Gather.Scenes[int32(sceneId)]
	if scene == nil || scene.GetGatherRuleID() < 0 {
		return nil
	}
	return GetGatherRuleConfigure(scene.GetGatherRuleID())
}

// GatherRuleLimits 按 NewGatherType 0..5 返回普通上限和幸运完成上限。
// 配置中的零是有效上限，不代表无限；缺失、负值或类型越界均返回 false。
func GatherRuleLimits(rule *excel.GatherRuleConfigure, gatherType uint32) (uint32, uint32, bool) {
	if rule == nil || gatherType >= GatherTypeCount {
		return 0, 0, false
	}
	normal := [...]int32{
		rule.GetWoodNum(), rule.GetFlowerNum(), rule.GetStoneNum(),
		rule.GetButterflyNum(), rule.GetWaterNum(), rule.GetPickNum(),
	}
	lucky := [...]int32{
		rule.GetLuckyFinishwoodNum(), rule.GetLuckyFinishflowerNum(), rule.GetLuckyFinishstoneNum(),
		rule.GetLuckyFinishbutterflyNum(), rule.GetLuckyFinishwaterNum(), rule.GetLuckyFinishpickNum(),
	}
	if normal[gatherType] < 0 || lucky[gatherType] < 0 {
		return 0, 0, false
	}
	return uint32(normal[gatherType]), uint32(lucky[gatherType]), true
}
