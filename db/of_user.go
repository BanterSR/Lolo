package db

import (
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

type OFUser struct {
	UserId    uint32 `gorm:"primarykey;autoIncrement"`
	SdkUid    uint32 `gorm:"unique"`
	Token     string
	DeviceId  string // 设备码
	ChannelId string
	Ban       bool
	BanTime   time.Time
	BanText   string
	Game      *OFGame `gorm:"foreignKey:UserId"`
}

var ofUserCreateMu sync.Mutex

func (u *OFUser) BeforeCreate(tx *gorm.DB) (err error) {
	if u.UserId == 0 {
		var lastUser OFUser
		tx.Order("user_id desc").First(&lastUser)
		if lastUser.UserId < 1000000 {
			u.UserId = 1000000
		} else {
			u.UserId = lastUser.UserId + 1
		}
	}
	return nil
}

// GetOFUserByUserId 使用UserId拉取数据
func GetOFUserByUserId(userId uint32) (*OFUser, error) {
	user := &OFUser{UserId: userId}
	tx := db.Where("user_id = ?", userId).First(user)
	return user, tx.Error
}

// GetOFUserBySdkUid 使用SdkUid拉取数据
func GetOFUserBySdkUid(sdkUid uint32) (*OFUser, error) {
	user := &OFUser{}
	err := db.Where("sdk_uid = ?", sdkUid).First(user).Error
	return user, err
}

func CreateOFUser(user *OFUser) (*OFUser, error) {
	ofUserCreateMu.Lock()
	defer ofUserCreateMu.Unlock()

	existing := new(OFUser)
	err := db.Where("sdk_uid = ?", user.SdkUid).First(existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return user, err
	}
	if user.UserId == 0 {
		var lastUser OFUser
		if err := db.Order("user_id desc").First(&lastUser).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return user, err
		}
		if lastUser.UserId < 1000000 {
			user.UserId = 1000000
		} else {
			user.UserId = lastUser.UserId + 1
		}
	}
	if err = db.Create(user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func SaveOFUser(sdkUid uint32, fx func(user *OFUser) bool) error {
	tx := db.Begin()
	info := new(OFUser)
	if tx.Where("sdk_uid = ?", sdkUid).First(info); tx.Error != nil {
		tx.Rollback()
		return tx.Error
	}
	if !fx(info) {
		tx.Rollback()
		return nil
	}
	if tx.Save(info).Error != nil {
		tx.Rollback()
		return tx.Error
	}

	return tx.Commit().Error
}
