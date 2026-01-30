package gdconf

import "gucooing/lolo/protocol/excel"

type Gather struct {
	all          *excel.AllGatherDatas
	Gathers      map[uint32]*excel.GatherConfigure
	GatherReward map[int32]*excel.GatherRewardConfigure
}

func (g *GameConfig) loadGather() {
	info := &Gather{
		all:          new(excel.AllGatherDatas),
		Gathers:      make(map[uint32]*excel.GatherConfigure),
		GatherReward: make(map[int32]*excel.GatherRewardConfigure),
	}
	g.Excel.Gather = info
	name := "Gather.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetGather().GetDatas() {
		info.Gathers[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetGatherReward().GetDatas() {
		info.GatherReward[v.ID] = v
	}
}

func GetGatherConfigure(id uint32) *excel.GatherConfigure {
	return cc.Excel.Gather.Gathers[id]
}

func GetGatherRewardConfigure(id int32) *excel.GatherRewardConfigure {
	return cc.Excel.Gather.GatherReward[id]
}
