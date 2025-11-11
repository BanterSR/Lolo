package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Weapon struct {
	all                *excel.AllWeaponDatas
	WeaponAllMap       map[uint32]*WeaponAllInfo
	WeaponAllMapByItem map[uint32]*WeaponAllInfo
}

type WeaponAllInfo struct {
	WeaponId   uint32
	WeaponInfo *excel.WeaponConfigure
}

func (g *GameConfig) loadWeapon() {
	info := &Weapon{
		all:                new(excel.AllWeaponDatas),
		WeaponAllMap:       make(map[uint32]*WeaponAllInfo),
		WeaponAllMapByItem: make(map[uint32]*WeaponAllInfo),
	}
	g.Excel.Weapon = info
	name := "Weapon.json"
	ReadJson(g.excelPath, name, &info.all)

	getWeaponAllInfo := func(id int32) *WeaponAllInfo {
		if info.WeaponAllMap[uint32(id)] == nil {
			info.WeaponAllMap[uint32(id)] = &WeaponAllInfo{
				WeaponId: uint32(id),
			}
		}
		return info.WeaponAllMap[uint32(id)]
	}

	for _, v := range info.all.GetWeapon().GetDatas() {
		getWeaponAllInfo(v.ID).WeaponInfo = v
		info.WeaponAllMapByItem[uint32(v.ItemID)] = getWeaponAllInfo(v.ID)
	}
}

func GetWeaponAllInfo(id uint32) *WeaponAllInfo {
	return cc.Excel.Weapon.WeaponAllMap[id]
}

func GetWeaponAllInfoByItemId(itemId uint32) *WeaponAllInfo {
	return cc.Excel.Weapon.WeaponAllMapByItem[itemId]
}

func GetWeaponAllMap() map[uint32]*WeaponAllInfo {
	return cc.Excel.Weapon.WeaponAllMap
}
