package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Shop struct {
	all  *excel.AllShopDatas
	Info map[uint32]*excel.ShopInfoConfigure
	Grid map[uint32]map[uint32]*excel.ShopGridConfigureItem
	Pool map[uint32]map[uint32]*excel.ShopPoolConfigureItem
}

func (g *GameConfig) loadShop() {
	info := &Shop{
		all:  new(excel.AllShopDatas),
		Info: make(map[uint32]*excel.ShopInfoConfigure),
		Grid: make(map[uint32]map[uint32]*excel.ShopGridConfigureItem),
		Pool: make(map[uint32]map[uint32]*excel.ShopPoolConfigureItem),
	}
	g.Excel.Shop = info
	name := "Shop.json"
	ReadJson(g.excelPath, name, &info.all)
	for _, v := range info.all.GetInfo().GetDatas() {
		info.Info[uint32(v.ID)] = v
	}

	for _, v := range info.all.GetGrid().GetDatas() {
		grid := make(map[uint32]*excel.ShopGridConfigureItem)
		for _, v2 := range v.GetItems() {
			grid[uint32(v2.GridID)] = v2
		}
		info.Grid[uint32(v.ID)] = grid
	}

	for _, v := range info.all.GetPool().GetDatas() {
		pool := make(map[uint32]*excel.ShopPoolConfigureItem)
		for _, v2 := range v.GetItems() {
			pool[uint32(v2.Index)] = v2
		}
		info.Pool[uint32(v.ID)] = pool
	}
}

func GetShopInfo(shopId uint32) *excel.ShopInfoConfigure {
	return cc.Excel.Shop.Info[shopId]
}

func GetAllShopInfo() []*excel.ShopInfoConfigure {
	return cc.Excel.Shop.all.GetInfo().GetDatas()
}

func GetGrids(grid uint32) map[uint32]*excel.ShopGridConfigureItem {
	return cc.Excel.Shop.Grid[grid]
}

func GetGridsByShopId(shopId uint32) map[uint32]*excel.ShopGridConfigureItem {
	info := GetShopInfo(shopId)
	return GetGrids(uint32(info.GetGridID()))
}

func GetGrid(shopId, gridID uint32) *excel.ShopGridConfigureItem {
	ls := GetGridsByShopId(shopId)
	if ls == nil {
		return nil
	}
	return ls[gridID]
}

func GetPools(poolId uint32) map[uint32]*excel.ShopPoolConfigureItem {
	return cc.Excel.Shop.Pool[poolId]
}

func GetPoolByGrid(shopId, gridID, itemId uint32) *excel.ShopPoolConfigureItem {
	grid := GetGrid(shopId, gridID)
	if grid == nil {
		return nil
	}
	ls := GetPools(uint32(grid.GetShopPoolID()))
	for _, v := range ls {
		if uint32(v.ItemID) == itemId {
			return v
		}
	}
	return nil
}
