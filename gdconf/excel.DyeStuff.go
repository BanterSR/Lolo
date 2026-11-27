package gdconf

import "gucooing/lolo/protocol/excel"

type DyeStuff struct {
	all      *excel.AllDyeStuffDatas
	DataList []*excel.DyeDataConfigure
}

func (g *GameConfig) loadDyeStuff() {
	info := &DyeStuff{
		all:      new(excel.AllDyeStuffDatas),
		DataList: make([]*excel.DyeDataConfigure, 0),
	}
	g.Excel.DyeStuff = info
	ReadJson(g.excelPath, "DyeStuff.json", &info.all)
	for _, v := range info.all.GetDyeData().GetDatas() {
		if v != nil {
			info.DataList = append(info.DataList, v)
		}
	}
}

func GetDyeDataList() []*excel.DyeDataConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.DyeStuff == nil {
		return nil
	}
	return cc.Excel.DyeStuff.DataList
}
