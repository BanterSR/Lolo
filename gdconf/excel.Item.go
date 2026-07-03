package gdconf

import (
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
	"math/rand"
)

type Item struct {
	all                 *excel.AllItemDatas
	ItemMap             map[uint32]*excel.ItemConfigure
	ItemByNewBagItemTag map[proto.EBagItemTag][]*excel.ItemConfigure
}

func (g *GameConfig) loadItem() {
	info := &Item{
		all:                 new(excel.AllItemDatas),
		ItemMap:             make(map[uint32]*excel.ItemConfigure),
		ItemByNewBagItemTag: make(map[proto.EBagItemTag][]*excel.ItemConfigure),
	}
	g.Excel.Item = info
	name := "Item.json"
	ReadJson(g.excelPath, name, &info.all)
	for _, v := range info.all.GetItem().GetDatas() {
		info.ItemMap[uint32(v.ID)] = v

		if info.ItemByNewBagItemTag[proto.EBagItemTag(v.NewBagItemTag)] == nil {
			info.ItemByNewBagItemTag[proto.EBagItemTag(v.NewBagItemTag)] = make([]*excel.ItemConfigure, 0)
		}
		info.ItemByNewBagItemTag[proto.EBagItemTag(v.NewBagItemTag)] = append(
			info.ItemByNewBagItemTag[proto.EBagItemTag(v.NewBagItemTag)], v)
	}
}

func GetItemConfigure(id uint32) *excel.ItemConfigure {
	return cc.Excel.Item.ItemMap[id]
}

func GetItemByNewBagItemTagAll() map[proto.EBagItemTag][]*excel.ItemConfigure {
	return cc.Excel.Item.ItemByNewBagItemTag
}

func GetItemByNewBagItemTag(tag proto.EBagItemTag) []*excel.ItemConfigure {
	return cc.Excel.Item.ItemByNewBagItemTag[tag]
}

func RandItemByNewBagItemTag(tag proto.EBagItemTag) *excel.ItemConfigure {
	list := cc.Excel.Item.ItemByNewBagItemTag[tag]
	return list[rand.Intn(len(list))]
}

func GetAllItemConfigure() []*excel.ItemConfigure {
	return cc.Excel.Item.all.GetItem().GetDatas()
}
