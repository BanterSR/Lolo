package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Fashion struct {
	all                 *excel.AllFashionDatas
	FashionAllMap       map[uint32]*FashionAllInfo
	FashionAllMapByItem map[uint32]*FashionAllInfo
}

type FashionAllInfo struct {
	FashionId   uint32
	FashionInfo *excel.FashionInfoConfigure
}

func (g *GameConfig) loadFashion() {
	info := &Fashion{
		all:                 new(excel.AllFashionDatas),
		FashionAllMap:       make(map[uint32]*FashionAllInfo),
		FashionAllMapByItem: make(map[uint32]*FashionAllInfo),
	}
	g.Excel.Fashion = info
	name := "Fashion.json"
	ReadJson(g.excelPath, name, &info.all)

	getFashionAllInfo := func(id int32) *FashionAllInfo {
		if info.FashionAllMap[uint32(id)] == nil {
			info.FashionAllMap[uint32(id)] = &FashionAllInfo{
				FashionId: uint32(id),
			}
		}
		return info.FashionAllMap[uint32(id)]
	}

	for _, v := range info.all.GetFashionInfo().GetDatas() {
		getFashionAllInfo(v.ID).FashionInfo = v
		info.FashionAllMapByItem[uint32(v.ItemID)] = getFashionAllInfo(v.ID)
	}
}

func GetFashionAllInfo(id uint32) *FashionAllInfo {
	return cc.Excel.Fashion.FashionAllMap[id]
}

func GetFashionAllInfoByItemId(itemId uint32) *FashionAllInfo {
	return cc.Excel.Fashion.FashionAllMapByItem[itemId]
}

func GetFashionAllMap() map[uint32]*FashionAllInfo {
	return cc.Excel.Fashion.FashionAllMap
}
