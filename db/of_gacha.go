package db

import (
	"gucooing/lolo/protocol/proto"
)

// 抽卡历史记录
type OFGachaRecord struct {
	ID        int64  `gorm:"primary_key;AUTO_INCREMENT"`
	GachaId   uint32 `gorm:"not null;index:gacha_user_id"`
	UserID    uint32 `gorm:"not null;index:gacha_user_id"`
	ItemId    uint32 `gorm:"not null"`
	GachaTime int64  `gorm:"not null"`
}

func (r OFGachaRecord) PlayerGachaRecord() *proto.PlayerGachaRecord {
	return &proto.PlayerGachaRecord{
		GachaId:   r.GachaId,
		ItemId:    r.ItemId,
		GachaTime: r.GachaTime,
	}
}

// 批量写入抽卡记录
func CreateGachaRecords(list []*OFGachaRecord) error {
	return db.Create(&list).Error
}

// 批量拉取抽卡记录
func GetGachaRecords(userId, gachaId, page uint32) ([]*OFGachaRecord, uint32, error) {
	var list []*OFGachaRecord
	all := db.Model(&OFGachaRecord{}).Where("gacha_id = ? AND user_id = ?", gachaId, userId)
	var totalPage int64
	err := all.Count(&totalPage).Error
	if err != nil {
		return nil, 0, err
	}
	err = all.
		Order("id DESC").
		Limit(5).
		Offset(int((page - 1) * 5)).
		Find(&list).Error
	return list, uint32(totalPage), err
}
