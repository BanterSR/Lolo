package db

import (
	"time"

	"gucooing/lolo/pkg/cache"
)

var (
	basicCache = cache.New[uint32, *UserBasic](3 * time.Second)
)

type UserBasic struct {
	UserId          uint32 `gorm:"primaryKey;not null"`
	NickName        string `gorm:"default:'gucooing'"`
	Level           uint32 `gorm:"default:1"`
	Sign            string `gorm:"default:''"`
	Exp             uint32 `gorm:"default:0"`
	Head            uint32 `gorm:"default:41101"`
	Birthday        string `gorm:"default:''"`
	IsHideBirthday  bool   `gorm:"default:false"`
	PhoneBackground uint32 `gorm:"default:8000"` // 手机背景
}

// 获取玩家基础信息
func GetUserBasic(userId uint32) (*UserBasic, error) {
	if basic, ok := basicCache.Get(userId); ok {
		return basic, nil
	}
	basic := &UserBasic{
		UserId: userId,
	}
	err := db.FirstOrCreate(basic).Error
	if err != nil {
		return nil, err
	}
	basicCache.Set(userId, basic)
	return basic, nil
}

// 更新基础信息
func UpUserBasic(userId uint32, fx func(basic *UserBasic) bool) error {
	basic, err := GetUserBasic(userId)
	if err != nil {
		return err
	}
	if !fx(basic) {
		return nil
	}
	if err = db.Save(basic).Error; err != nil {
		return err
	}
	basicCache.Set(userId, basic)
	return nil
}
