package gdconf

import "gucooing/lolo/protocol/excel"

type CharacterPractice struct {
	all                  *excel.AllCharacterPracticeDatas
	CharacterPracticeMap map[uint32]*excel.CharacterPracticeConfigure
}

func (g *GameConfig) loadCharacterPractice() {
	info := &CharacterPractice{
		all:                  new(excel.AllCharacterPracticeDatas),
		CharacterPracticeMap: make(map[uint32]*excel.CharacterPracticeConfigure),
	}
	g.Excel.CharacterPractice = info
	name := "CharacterPractice.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetCharacterPractice().GetDatas() {
		info.CharacterPracticeMap[uint32(v.GetID())] = v
	}
}

// GetCharacterPractice 官方推荐练度(按角色id):推荐武器/练度副本/目标等级
func GetCharacterPractice(id uint32) *excel.CharacterPracticeConfigure {
	return cc.Excel.CharacterPractice.CharacterPracticeMap[id]
}
