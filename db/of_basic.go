package db

import (
	"time"

	"gucooing/lolo/pkg/cache"
	"gucooing/lolo/protocol/proto"
)

var (
	basicCache = cache.New[uint32, *OFGameBasic](5 * time.Second)
)

type OFGameBasic struct {
	UserId          uint32         `gorm:"primaryKey;not null"`
	NickName        string         `gorm:"default:'gucooing'"`
	Level           uint32         `gorm:"default:1"`
	Exp             uint32         `gorm:"default:0"`
	Head            uint32         `gorm:"default:41101"`
	LastLoginTime   int64          `gorm:"default:0"`    // 上次登录时间
	Sex             proto.ESexType `gorm:"default:0"`    // 性别
	PhoneBackground uint32         `gorm:"default:8000"` // 手机背景
	Sign            string         `gorm:"default:''"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	Birthday        string         `gorm:"default:''"`
	IsHideBirthday  bool           `gorm:"default:false"`
	AvatarFrame     uint32         `gorm:"default:0"`
}

// 获取玩家基础信息
func GetGameBasic(userId uint32) (*OFGameBasic, error) {
	if basic, ok := basicCache.Get(userId); ok {
		return basic, nil
	}
	basic := &OFGameBasic{
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
func UpGameBasic(userId uint32, fx func(basic *OFGameBasic) bool) error {
	basic, err := GetGameBasic(userId)
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
