package gdconf

import (
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

type CollectionItem struct {
	all          *excel.AllCollectionItemDatas
	ItemMap      map[uint32]*excel.CollectionItemConfigure
	FileMap      map[uint32]*excel.CollectionItemFileConfigure
	ItemItemMap  map[uint32]*excel.CollectionItemItemConfigure
	BookMap      map[uint32]*excel.CollectionItemBookConfigure
	TapeMap      map[uint32]*excel.CollectionItemTapeConfigure
	PortalMap    map[uint32]*excel.CollectionItemPortalConfigure
	DataMap      map[uint32]*excel.CollectionItemDataConfigure
	PhotoDataMap map[uint32]*excel.CollectionPhotoDataConfigure
	PhotoItemMap map[uint32]*excel.CollectItemPhotoItemConfigure
}

func (g *GameConfig) loadCollectionItem() {
	info := &CollectionItem{
		all:          new(excel.AllCollectionItemDatas),
		ItemMap:      make(map[uint32]*excel.CollectionItemConfigure),
		FileMap:      make(map[uint32]*excel.CollectionItemFileConfigure),
		ItemItemMap:  make(map[uint32]*excel.CollectionItemItemConfigure),
		BookMap:      make(map[uint32]*excel.CollectionItemBookConfigure),
		TapeMap:      make(map[uint32]*excel.CollectionItemTapeConfigure),
		PortalMap:    make(map[uint32]*excel.CollectionItemPortalConfigure),
		DataMap:      make(map[uint32]*excel.CollectionItemDataConfigure),
		PhotoDataMap: make(map[uint32]*excel.CollectionPhotoDataConfigure),
		PhotoItemMap: make(map[uint32]*excel.CollectItemPhotoItemConfigure),
	}
	g.Excel.CollectionItem = info
	name := "CollectionItem.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetCollectionItem().GetDatas() {
		info.ItemMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionItemFile().GetDatas() {
		info.FileMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionItemItem().GetDatas() {
		info.ItemItemMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionItemBook().GetDatas() {
		info.BookMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionItemTape().GetDatas() {
		info.TapeMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionItemPortal().GetDatas() {
		info.PortalMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionItemData().GetDatas() {
		info.DataMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectionPhotoData().GetDatas() {
		info.PhotoDataMap[uint32(v.ID)] = v
	}
	for _, v := range info.all.GetCollectItemPhotoItem().GetDatas() {
		info.PhotoItemMap[uint32(v.ID)] = v
	}
}

func GetCollectionItem(id uint32) *excel.CollectionItemConfigure {
	return cc.Excel.CollectionItem.ItemMap[id]
}
func GetCollectionItemFile(id uint32) *excel.CollectionItemFileConfigure {
	return cc.Excel.CollectionItem.FileMap[id]
}
func GetCollectionItemItem(id uint32) *excel.CollectionItemItemConfigure {
	return cc.Excel.CollectionItem.ItemItemMap[id]
}
func GetCollectionItemBook(id uint32) *excel.CollectionItemBookConfigure {
	return cc.Excel.CollectionItem.BookMap[id]
}
func GetCollectionItemTape(id uint32) *excel.CollectionItemTapeConfigure {
	return cc.Excel.CollectionItem.TapeMap[id]
}
func GetCollectionItemPortal(id uint32) *excel.CollectionItemPortalConfigure {
	return cc.Excel.CollectionItem.PortalMap[id]
}
func GetCollectionItemData(id uint32) *excel.CollectionItemDataConfigure {
	return cc.Excel.CollectionItem.DataMap[id]
}
func GetCollectionPhotoData(id uint32) *excel.CollectionPhotoDataConfigure {
	return cc.Excel.CollectionItem.PhotoDataMap[id]
}
func GetCollectItemPhotoItem(id uint32) *excel.CollectItemPhotoItemConfigure {
	return cc.Excel.CollectionItem.PhotoItemMap[id]
}

func GetCollectionReward(info *excel.CollectionItemConfigure) []*excel.RewardItemPoolGroupInfo {
	var rewardId int32
	switch proto.ECollectionType(info.NewCollectionType) {
	case proto.ECollectionType_ECollectionType_CollectFile:
		rewardId = GetCollectionItemFile(uint32(info.GetRelateID())).GetRewardID()
	case proto.ECollectionType_ECollectionType_CollectItem:
		rewardId = GetCollectionItemItem(uint32(info.GetRelateID())).GetRewardID()
	case proto.ECollectionType_ECollectionType_CollectBook:
		rewardId = GetCollectionItemBook(uint32(info.GetRelateID())).GetRewardID()
	case proto.ECollectionType_ECollectionType_CollectTape:
	case proto.ECollectionType_ECollectionType_CollectPortal:
	case proto.ECollectionType_ECollectionType_CollectData:
		rewardId = GetCollectionItemData(uint32(info.GetRelateID())).GetRewardID()
	case proto.ECollectionType_ECollectionType_CollectPhoto:
	case proto.ECollectionType_ECollectionType_CollectCrystal:
	case proto.ECollectionType_ECollectionType_CollectPhotoItem:
		rewardId = GetCollectItemPhotoItem(uint32(info.GetRelateID())).GetRewardID()
	case proto.ECollectionType_ECollectionType_CollectMoonPiece:
	case proto.ECollectionType_ECollectionType_CollectEmotionMoon:

	}

	return GetRewardItemPoolByRewardId(uint32(rewardId))
}
