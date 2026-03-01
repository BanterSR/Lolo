package gdconf

import "gucooing/lolo/protocol/excel"

type Make struct {
	all           *excel.AllMakeDatas
	MakeLevelMap  map[uint32]*excel.MakeLevelConfigure
	MakeItemMap   map[uint32]*excel.MakeItemConfigure
	MakeSpellMap  map[uint32]*excel.MakeSpellConfigure
	MakeSpellItem map[uint32]map[uint32]*excel.MakeSpellItemsConfigureItem
}

func (g *GameConfig) loadMake() {
	info := &Make{
		all:           new(excel.AllMakeDatas),
		MakeLevelMap:  make(map[uint32]*excel.MakeLevelConfigure),
		MakeItemMap:   make(map[uint32]*excel.MakeItemConfigure),
		MakeSpellMap:  make(map[uint32]*excel.MakeSpellConfigure),
		MakeSpellItem: make(map[uint32]map[uint32]*excel.MakeSpellItemsConfigureItem),
	}
	g.Excel.Make = info
	name := "Make.json"
	ReadJson(g.excelPath, name, &info.all)

	getSpellMap := func(id uint32) map[uint32]*excel.MakeSpellItemsConfigureItem {
		if info.MakeSpellItem[id] == nil {
			info.MakeSpellItem[id] = make(map[uint32]*excel.MakeSpellItemsConfigureItem)
		}
		return info.MakeSpellItem[id]
	}

	for _, v := range info.all.GetMakeLevel().GetDatas() {
		info.MakeLevelMap[uint32(v.GetID())] = v
	}

	for _, v := range info.all.GetMakeItem().GetDatas() {
		info.MakeItemMap[uint32(v.GetID())] = v
	}

	for _, v := range info.all.GetMakeSpell().GetDatas() {
		info.MakeSpellMap[uint32(v.GetID())] = v
		spell := getSpellMap(uint32(v.GetID()))
		for _, v2 := range v.GetMakeSpellItems() {
			spell[uint32(v2.GetLevel())] = v2
		}
	}
}

func GetMakeLevelConfigure(id uint32) *excel.MakeLevelConfigure {
	return cc.Excel.Make.MakeLevelMap[id]
}

func GetMakeItemConfigure(id uint32) *excel.MakeItemConfigure {
	return cc.Excel.Make.MakeItemMap[id]
}

func GetMakeSpellConfigure(id uint32) *excel.MakeSpellConfigure {
	return cc.Excel.Make.MakeSpellMap[id]
}

func GetMakeSpellItemConfigure(spellId, level uint32) *excel.MakeSpellItemsConfigureItem {
	levelMap := cc.Excel.Make.MakeSpellItem[spellId]
	return levelMap[level]
}
