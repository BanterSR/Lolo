package gdconf

import "gucooing/lolo/protocol/excel"

type Dungeon struct {
	all             *excel.AllDungeonDatas
	DungeonMap      map[uint32]*excel.DungeonConfigure
	DungeonQuestMap map[uint32]*excel.DungeonQuestConfigure
}

func (g *GameConfig) loadDungeon() {
	info := &Dungeon{
		all:             new(excel.AllDungeonDatas),
		DungeonMap:      make(map[uint32]*excel.DungeonConfigure),
		DungeonQuestMap: make(map[uint32]*excel.DungeonQuestConfigure),
	}
	g.Excel.Dungeon = info
	name := "Dungeon.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetDungeon().GetDatas() {
		id := uint32(v.GetID())
		if id != 0 {
			info.DungeonMap[id] = v
		}
		// Some rows also carry a logical DungeonID. Keep both keys.
		dungeonId := uint32(v.GetDungeonID())
		if dungeonId != 0 {
			info.DungeonMap[dungeonId] = v
		}
	}

	for _, v := range info.all.GetDungeonQuest().GetDatas() {
		id := uint32(v.GetID())
		if id != 0 {
			info.DungeonQuestMap[id] = v
		}
	}
}

func GetDungeonConfigure(id uint32) *excel.DungeonConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Dungeon == nil {
		return nil
	}
	return cc.Excel.Dungeon.DungeonMap[id]
}

func GetDungeonQuestConfigure(id uint32) *excel.DungeonQuestConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Dungeon == nil {
		return nil
	}
	return cc.Excel.Dungeon.DungeonQuestMap[id]
}
