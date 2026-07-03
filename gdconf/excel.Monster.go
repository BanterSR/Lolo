package gdconf

import "gucooing/lolo/protocol/excel"

type Monster struct {
	all                          *excel.AllMonsterDatas
	MonsterCharacterMap          map[uint32]*excel.MonsterCharacterConfigure
	MonsterGrowthModelMap        map[uint32]*excel.MonsterCharacterGrowthModelConfigure
	MonsterGrowthLevelMap        map[uint32]map[uint32]*excel.MonsterCharacterGrowthModelLevelInfo
	MonsterDungeonGrowthModelMap map[uint32]*excel.MonsterDungeonCharacterGrowthModelConfigure
	MonsterDungeonGrowthLevelMap map[uint32]map[uint32]*excel.MonsterDungeonCharacterGrowthModelLevelInfo
}

func (g *GameConfig) loadMonster() *Monster {
	info := &Monster{
		all:                          new(excel.AllMonsterDatas),
		MonsterCharacterMap:          make(map[uint32]*excel.MonsterCharacterConfigure),
		MonsterGrowthModelMap:        make(map[uint32]*excel.MonsterCharacterGrowthModelConfigure),
		MonsterGrowthLevelMap:        make(map[uint32]map[uint32]*excel.MonsterCharacterGrowthModelLevelInfo),
		MonsterDungeonGrowthModelMap: make(map[uint32]*excel.MonsterDungeonCharacterGrowthModelConfigure),
		MonsterDungeonGrowthLevelMap: make(map[uint32]map[uint32]*excel.MonsterDungeonCharacterGrowthModelLevelInfo),
	}
	g.Excel.Monster = info
	name := "Monster.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetMonsterCharacterData().GetDatas() {
		info.MonsterCharacterMap[uint32(v.GetID())] = v
	}
	for _, v := range info.all.GetMonsterGrowthModel().GetDatas() {
		id := uint32(v.GetID())
		info.MonsterGrowthModelMap[id] = v
		info.MonsterGrowthLevelMap[id] = make(map[uint32]*excel.MonsterCharacterGrowthModelLevelInfo)
		for _, lv := range v.GetLevelGroupInfo() {
			info.MonsterGrowthLevelMap[id][uint32(lv.GetLevel())] = lv
		}
	}
	for _, v := range info.all.GetMonsterDungeonGrowthModel().GetDatas() {
		id := uint32(v.GetID())
		info.MonsterDungeonGrowthModelMap[id] = v
		info.MonsterDungeonGrowthLevelMap[id] = make(map[uint32]*excel.MonsterDungeonCharacterGrowthModelLevelInfo)
		for _, lv := range v.GetDungeonLevelGroupInfo() {
			info.MonsterDungeonGrowthLevelMap[id][uint32(lv.GetLevel())] = lv
		}
	}

	return info
}

func GetMonsterCharacterConfigure(id uint32) *excel.MonsterCharacterConfigure {
	info := cc.Excel.Monster
	if info == nil {
		return nil
	}
	return info.MonsterCharacterMap[id]
}

func GetMonsterGrowthModelConfigure(id uint32) *excel.MonsterCharacterGrowthModelConfigure {
	info := cc.Excel.Monster
	if info == nil {
		return nil
	}
	return info.MonsterGrowthModelMap[id]
}

func GetMonsterGrowthModelLevelInfo(id, level uint32) *excel.MonsterCharacterGrowthModelLevelInfo {
	info := cc.Excel.Monster
	if info == nil {
		return nil
	}
	levelMap := info.MonsterGrowthLevelMap[id]
	if levelMap == nil {
		return nil
	}
	return levelMap[level]
}

func GetMonsterDungeonGrowthModelConfigure(id uint32) *excel.MonsterDungeonCharacterGrowthModelConfigure {
	info := cc.Excel.Monster
	if info == nil {
		return nil
	}
	return info.MonsterDungeonGrowthModelMap[id]
}

func GetMonsterDungeonGrowthModelLevelInfo(id, level uint32) *excel.MonsterDungeonCharacterGrowthModelLevelInfo {
	info := cc.Excel.Monster
	if info == nil {
		return nil
	}
	levelMap := info.MonsterDungeonGrowthLevelMap[id]
	if levelMap == nil {
		return nil
	}
	return levelMap[level]
}

func GetAllMonsterCharacterConfigure() []*excel.MonsterCharacterConfigure {
	info := cc.Excel.Monster
	if info == nil {
		return nil
	}
	return info.all.GetMonsterCharacterData().GetDatas()
}
