package db

import (
	"errors"
	"sync"

	"gorm.io/gorm"
)

var accountCreateMu sync.Mutex

// 快游戏sdk
type OFQuick struct {
	ID        uint32 `gorm:"primarykey;autoIncrement;index"`
	Username  string `gorm:"unique;index"`
	Password  string `gorm:"not null"`
	RegDevice string `gorm:"not null"`
	UserToken string // 网关登录
	AuthToken string `gorm:"not null"` // 自动登录
}

func CreateOFQuick(username string, password string) (*OFQuick, error) {
	q := &OFQuick{
		Username: username,
		Password: password,
	}
	err := db.Create(q).Error
	return q, err
}

func GetOFQuick(id uint32) (*OFQuick, error) {
	q := new(OFQuick)
	err := db.Where("id = ?", id).First(q).Error
	return q, err
}

func OrCreateOFQuick(username, password string) (*OFQuick, error) {
	accountCreateMu.Lock()
	defer accountCreateMu.Unlock()

	q := &OFQuick{
		Username: username,
		Password: password,
	}
	err := db.Where("username = ?", username).First(q).Error
	if err == nil {
		return q, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err = db.Create(q).Error; err != nil {
		return nil, err
	}
	return q, nil
}

func UpOFQuick(q *OFQuick) error {
	return db.Save(q).Error
}
