package gdconf

import "gucooing/lolo/protocol/excel"

type QuickObtaining struct {
	all     *excel.AllQuickObtainingDatas
	DataMap map[uint32]*excel.QuickObtainingConfigure
}

func (g *GameConfig) loadQuickObtaining() {
	info := &QuickObtaining{
		all:     new(excel.AllQuickObtainingDatas),
		DataMap: make(map[uint32]*excel.QuickObtainingConfigure),
	}
	g.Excel.QuickObtaining = info
	ReadJson(g.excelPath, "QuickObtaining.json", &info.all)
	for _, v := range info.all.GetQuickObtaining().GetDatas() {
		if v != nil && v.GetID() != 0 {
			info.DataMap[uint32(v.GetID())] = v
		}
	}
}

func GetQuickObtainingConfigure(id uint32) *excel.QuickObtainingConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.QuickObtaining == nil {
		return nil
	}
	return cc.Excel.QuickObtaining.DataMap[id]
}
