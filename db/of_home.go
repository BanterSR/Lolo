package db

import (
	"errors"

	"gorm.io/gorm"
)

// 家园数据
type OFHome struct {
	ID                     uint64 `gorm:"primarykey;autoIncrement"`
	UserID                 uint32 `gorm:"index;uniqueIndex:idx_id"`
	SceneID                uint32 `gorm:"index;uniqueIndex:idx_id"`
	GardenName             string
	LikesNum               int64
	AccessPlayerNum        int64
	LeftLikeNum            uint32
	IsOpen                 bool
	Password               string
	PasswordExpireTime     int64
	GardenFurnitureInfoMap []byte
	PlacedCharacterMap     []byte
}

// 获取/创建家园
func GetOFHome(userId, sceneId uint32) (*OFHome, error) {
	accountCreateMu.Lock()
	defer accountCreateMu.Unlock()

	h := &OFHome{
		UserID:  userId,
		SceneID: sceneId,
	}
	err := db.Where("user_id = ? AND scene_id = ?", userId, sceneId).First(h).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Create(h).Error
	}
	return h, err
}

// 更新家园
func SaveOFHome(h *OFHome) error {
	return db.Save(h).Error
}
