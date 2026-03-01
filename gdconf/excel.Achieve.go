package gdconf

import "gucooing/lolo/protocol/excel"

type Achieve struct {
	all        *excel.AllAchieveDatas
	AchieveMap map[uint32]*excel.AchieveConfigure
}

func (g *GameConfig) loadAchieve() {
	info := &Achieve{
		all:        new(excel.AllAchieveDatas),
		AchieveMap: make(map[uint32]*excel.AchieveConfigure),
	}
	g.Excel.Achieve = info
	name := "Achieve.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetAchieve().GetDatas() {
		info.AchieveMap[uint32(v.GetID())] = v
	}
}

func GetAchieveConfigure(id uint32) *excel.AchieveConfigure {
	return cc.Excel.Achieve.AchieveMap[id]
}

func GetAchieveMap() map[uint32]*excel.AchieveConfigure {
	return cc.Excel.Achieve.AchieveMap
}

func GetAllAchieveConfigure() []*excel.AchieveConfigure {
	return cc.Excel.Achieve.all.GetAchieve().GetDatas()
}
