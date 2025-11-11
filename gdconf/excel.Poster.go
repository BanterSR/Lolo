package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Poster struct {
	all                *excel.AllPosterDatas
	PosterAllMap       map[uint32]*PosterAllInfo
	PosterAllMapByItem map[uint32]*PosterAllInfo
}

type PosterAllInfo struct {
	PosterId           uint32
	PosterInfo         *excel.PosterConfigure
	PosterIllustration *excel.PosterIllustrationConfigure
}

func (g *GameConfig) loadPoster() {
	info := &Poster{
		all:                new(excel.AllPosterDatas),
		PosterAllMap:       make(map[uint32]*PosterAllInfo),
		PosterAllMapByItem: make(map[uint32]*PosterAllInfo),
	}
	g.Excel.Poster = info
	name := "Poster.json"
	ReadJson(g.excelPath, name, &info.all)

	getPosterAllInfo := func(id int32) *PosterAllInfo {
		if info.PosterAllMap[uint32(id)] == nil {
			info.PosterAllMap[uint32(id)] = &PosterAllInfo{
				PosterId: uint32(id),
			}
		}
		return info.PosterAllMap[uint32(id)]
	}

	for _, v := range info.all.GetPoster().GetDatas() {
		getPosterAllInfo(v.ID).PosterInfo = v
		info.PosterAllMapByItem[uint32(v.ItemID)] = getPosterAllInfo(v.ID)
	}
	for _, v := range info.all.GetPosterIllustration().GetDatas() {
		getPosterAllInfo(v.ID).PosterIllustration = v
	}
}

func GetPosterAllInfo(id uint32) *PosterAllInfo {
	return cc.Excel.Poster.PosterAllMap[id]
}

func GetPosterAllInfoByItemId(itemId uint32) *PosterAllInfo {
	return cc.Excel.Poster.PosterAllMapByItem[itemId]
}

func GetPosterAllMap() map[uint32]*PosterAllInfo {
	return cc.Excel.Poster.PosterAllMap
}
