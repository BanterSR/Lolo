package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Ability struct {
	all            *excel.AllAbilityDatas
	AbilityMap     map[uint32]*excel.AbilityConfigure
	AbilityItemMap map[uint32]*excel.AbilityConfigure
}

func (g *GameConfig) loadAbility() {
	info := &Ability{
		all:            new(excel.AllAbilityDatas),
		AbilityMap:     make(map[uint32]*excel.AbilityConfigure),
		AbilityItemMap: make(map[uint32]*excel.AbilityConfigure),
	}
	g.Excel.Ability = info
	name := "Ability.json"
	ReadJson(g.excelPath, name, &info.all)
	for _, v := range info.all.GetAbility().GetDatas() {
		info.AbilityMap[uint32(v.ID)] = v
		info.AbilityItemMap[uint32(v.NeedItemID)] = v
	}
}

func GetAbilityByItemId(itemId uint32) *excel.AbilityConfigure {
	return cc.Excel.Ability.AbilityItemMap[itemId]
}
