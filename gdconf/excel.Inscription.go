package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Inscription struct {
	all                     *excel.AllInscriptionDatas
	InscriptionAllMap       map[uint32]*InscriptionAllInfo
	InscriptionAllMapByItem map[uint32]*InscriptionAllInfo
}

type InscriptionAllInfo struct {
	InscriptionId   uint32
	InscriptionInfo *excel.InscriptionConfigure
}

func (g *GameConfig) loadInscription() {
	info := &Inscription{
		all:                     new(excel.AllInscriptionDatas),
		InscriptionAllMap:       make(map[uint32]*InscriptionAllInfo),
		InscriptionAllMapByItem: make(map[uint32]*InscriptionAllInfo),
	}
	g.Excel.Inscription = info
	name := "Inscription.json"
	ReadJson(g.excelPath, name, &info.all)

	getInscriptionAllInfo := func(id int32) *InscriptionAllInfo {
		if info.InscriptionAllMap[uint32(id)] == nil {
			info.InscriptionAllMap[uint32(id)] = &InscriptionAllInfo{
				InscriptionId: uint32(id),
			}
		}
		return info.InscriptionAllMap[uint32(id)]
	}

	for _, v := range info.all.GetInscription().GetDatas() {
		getInscriptionAllInfo(v.ID).InscriptionInfo = v
		info.InscriptionAllMapByItem[uint32(v.ItemID)] = getInscriptionAllInfo(v.ID)
	}
}

func GetInscriptionAllInfo(id uint32) *InscriptionAllInfo {
	return cc.Excel.Inscription.InscriptionAllMap[id]
}

func GetInscriptionAllInfoByItemId(itemId uint32) *InscriptionAllInfo {
	return cc.Excel.Inscription.InscriptionAllMapByItem[itemId]
}

func GetInscriptionAllMap() map[uint32]*InscriptionAllInfo {
	return cc.Excel.Inscription.InscriptionAllMap
}
