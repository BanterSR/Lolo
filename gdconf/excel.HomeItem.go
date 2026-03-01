package gdconf

import "gucooing/lolo/protocol/excel"

type HomeItem struct {
	all           *excel.AllHomeItemDatas
	HomeItemMap   map[uint32]*excel.HomeItemConfigure
	HomeThemeMap  map[uint32]*excel.HomeThemeConfigure
	MusicaItemMap map[uint32]*excel.MusicaItemConfigure
	RoomDecorMap  map[uint32]*excel.RoomDecorConfigure
}

func (g *GameConfig) loadHomeItem() {
	info := &HomeItem{
		all:           new(excel.AllHomeItemDatas),
		HomeItemMap:   make(map[uint32]*excel.HomeItemConfigure),
		HomeThemeMap:  make(map[uint32]*excel.HomeThemeConfigure),
		MusicaItemMap: make(map[uint32]*excel.MusicaItemConfigure),
		RoomDecorMap:  make(map[uint32]*excel.RoomDecorConfigure),
	}
	g.Excel.HomeItem = info
	name := "HomeItem.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetHomeItem().GetDatas() {
		info.HomeItemMap[uint32(v.GetID())] = v
	}

	for _, v := range info.all.GetHomeTheme().GetDatas() {
		info.HomeThemeMap[uint32(v.GetID())] = v
	}

	for _, v := range info.all.GetMusicaItem().GetDatas() {
		info.MusicaItemMap[uint32(v.GetID())] = v
	}

	for _, v := range info.all.GetRoomDecor().GetDatas() {
		info.RoomDecorMap[uint32(v.GetID())] = v
	}
}

func GetHomeItemConfigure(id uint32) *excel.HomeItemConfigure {
	return cc.Excel.HomeItem.HomeItemMap[id]
}

func GetHomeThemeConfigure(id uint32) *excel.HomeThemeConfigure {
	return cc.Excel.HomeItem.HomeThemeMap[id]
}

func GetMusicaItemConfigure(id uint32) *excel.MusicaItemConfigure {
	return cc.Excel.HomeItem.MusicaItemMap[id]
}

func GetRoomDecorConfigure(id uint32) *excel.RoomDecorConfigure {
	return cc.Excel.HomeItem.RoomDecorMap[id]
}
