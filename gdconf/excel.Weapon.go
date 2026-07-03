package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Weapon struct {
	all              *excel.AllWeaponDatas
	WeaponAllMap     map[uint32]*WeaponAllInfo
	PropertyGroupMap map[uint32]*excel.WeaponPropertyConfigure
	RandomProperty   map[int32]*excel.WeaponRandomPropertyConfigure
	RandomValue      map[int32]*excel.WeaponRandomValueConfigure
}

type WeaponAllInfo struct {
	WeaponId      uint32
	WeaponInfo    *excel.WeaponConfigure
	PropertyGroup map[uint32]*excel.WeaponPropertyGroupInfo // index
}

func (g *GameConfig) loadWeapon() {
	info := &Weapon{
		all:              new(excel.AllWeaponDatas),
		WeaponAllMap:     make(map[uint32]*WeaponAllInfo),
		PropertyGroupMap: make(map[uint32]*excel.WeaponPropertyConfigure),
		RandomProperty:   make(map[int32]*excel.WeaponRandomPropertyConfigure),
		RandomValue:      make(map[int32]*excel.WeaponRandomValueConfigure),
	}
	g.Excel.Weapon = info
	name := "Weapon.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetWeaponProperty().GetDatas() {
		info.PropertyGroupMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetWeaponRandomProperty().GetDatas() {
		info.RandomProperty[v.ID] = v
	}
	for _, v := range info.all.GetWeaponRandomValue().GetDatas() {
		info.RandomValue[v.ID] = v
	}

	getWeaponAllInfo := func(id int32) *WeaponAllInfo {
		if info.WeaponAllMap[uint32(id)] == nil {
			info.WeaponAllMap[uint32(id)] = &WeaponAllInfo{
				WeaponId:      uint32(id),
				PropertyGroup: make(map[uint32]*excel.WeaponPropertyGroupInfo),
			}
		}
		return info.WeaponAllMap[uint32(id)]
	}

	for _, v := range info.all.GetWeapon().GetDatas() {
		if v.ID != v.ItemID {
			continue
		}
		weaponInfo := getWeaponAllInfo(v.ID)
		weaponInfo.WeaponInfo = v
		// 添加参数组
		if property, ok := info.PropertyGroupMap[uint32(v.WeaponPropertyID)]; ok {
			for index, group := range property.GetWeaponPropertyGroupInfo() {
				weaponInfo.PropertyGroup[uint32(index)+1] = group
			}
		}
		// 添加随机组
	}
}

func GetWeaponAllInfo(id uint32) *WeaponAllInfo {
	return cc.Excel.Weapon.WeaponAllMap[id]
}

func GetWeaponAllMap() map[uint32]*WeaponAllInfo {
	return cc.Excel.Weapon.WeaponAllMap
}

func GetWeaponRandomPropertyConfigure(id int32) *excel.WeaponRandomPropertyConfigure {
	return cc.Excel.Weapon.RandomProperty[id]
}

func GetWeaponRandomValueConfigure(id int32) *excel.WeaponRandomValueConfigure {
	return cc.Excel.Weapon.RandomValue[id]
}
