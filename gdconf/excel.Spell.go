package gdconf

import "gucooing/lolo/protocol/excel"

type Spell struct {
	all             *excel.AllSpellDatas
	SpellMap        map[uint32]map[uint32]*excel.SpellConfigure
	SpellLevelUpMap map[int32]*excel.SpellLevelUpInfoConfigure
}

func (g *GameConfig) loadSpell() {
	info := &Spell{
		all:             new(excel.AllSpellDatas),
		SpellMap:        make(map[uint32]map[uint32]*excel.SpellConfigure),
		SpellLevelUpMap: make(map[int32]*excel.SpellLevelUpInfoConfigure),
	}
	g.Excel.Spell = info
	name := "Spell.json"
	ReadJson(g.excelPath, name, &info.all)

	getSpell := func(id uint32) map[uint32]*excel.SpellConfigure {
		if _, ok := info.SpellMap[id]; !ok {
			info.SpellMap[id] = make(map[uint32]*excel.SpellConfigure)
		}
		return info.SpellMap[id]
	}

	for _, v := range info.all.GetSpellInfo().GetDatas() {
		for _, v2 := range v.GetSpellConfigureItems() {
			getSpell(uint32(v.ID))[uint32(v2.Level)] = v2
		}
	}
	for _, v := range info.all.GetSpellLevelUpInfo().GetDatas() {
		info.SpellLevelUpMap[int32(v.ID)] = v
	}
}

func GetSpellLevelUpInfoBySkillId(skillId, level uint32) *excel.SpellLevelUpInfoConfigure {
	levelMap := cc.Excel.Spell.SpellMap[skillId]
	if levelMap == nil {
		return nil
	}
	spellInfo, ok := levelMap[level]
	if !ok {
		return nil
	}
	return cc.Excel.Spell.SpellLevelUpMap[spellInfo.GetCostID()]
}
