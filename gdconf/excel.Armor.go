package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Armor struct {
	all               *excel.AllArmorDatas
	ArmorAllMap       map[uint32]*ArmorAllInfo
	ArmorAllMapByItem map[uint32]*ArmorAllInfo
}

type ArmorAllInfo struct {
	ArmorId   uint32
	ArmorInfo *excel.ArmorConfigure
}

func (g *GameConfig) loadArmor() {
	info := &Armor{
		all:               new(excel.AllArmorDatas),
		ArmorAllMap:       make(map[uint32]*ArmorAllInfo),
		ArmorAllMapByItem: make(map[uint32]*ArmorAllInfo),
	}
	g.Excel.Armor = info
	name := "Armor.json"
	ReadJson(g.excelPath, name, &info.all)

	getArmorAllInfo := func(id int32) *ArmorAllInfo {
		if info.ArmorAllMap[uint32(id)] == nil {
			info.ArmorAllMap[uint32(id)] = &ArmorAllInfo{
				ArmorId: uint32(id),
			}
		}
		return info.ArmorAllMap[uint32(id)]
	}

	for _, v := range info.all.GetArmor().GetDatas() {
		getArmorAllInfo(v.ID).ArmorInfo = v
		info.ArmorAllMapByItem[uint32(v.ItemID)] = getArmorAllInfo(v.ID)
	}
}

func GetArmorAllInfo(id uint32) *ArmorAllInfo {
	return cc.Excel.Armor.ArmorAllMap[id]
}

func GetArmorAllInfoByItemId(itemId uint32) *ArmorAllInfo {
	return cc.Excel.Armor.ArmorAllMapByItem[itemId]
}

func GetArmorAllMap() map[uint32]*ArmorAllInfo {
	return cc.Excel.Armor.ArmorAllMap
}
