package db

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gucooing/lolo/pkg/alg"
	"time"
)

type OFGame struct {
	UserId    uint32    `gorm:"primaryKey;not null;index:user_id"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	BinData   GzipBin
	Basic     *OFGameBasic `gorm:"foreignKey:UserId"`
}

type GzipBin []byte

func (g *GzipBin) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal Gzip value:", value))
	}
	data, err := alg.UnGzip(bytes)
	if err != nil {
		return err
	}
	*g = data
	return err
}

func (g GzipBin) Value() (driver.Value, error) {
	if len(g) == 0 {
		return nil, nil
	}
	return alg.CompGzip(g)
}

// GetOFGameByUserId 使用UserId拉取数据 如果不存在就添加
func GetOFGameByUserId(userId uint32) (*OFGame, error) {
	user := &OFGame{
		UserId: userId,
	}
	err := db.FirstOrCreate(&user).Error
	return user, err
}

// 更新账号数据
func SaveOFGame(userId uint32, fx func(user *OFGame) bool) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		info := new(OFGame)
		if tx.Where("user_id = ?", userId).First(info); tx.Error != nil {
			return tx.Error
		}
		if !fx(info) {
			return nil
		}
		if tx.Save(info).Error != nil {
			return tx.Error
		}

		return tx.Error
	})

	return err
}

// 判断玩家是否存在
func IsUserExists(userId uint32) bool {
	var count int64
	db.Model(&OFGame{}).Where("user_id = ?", userId).Count(&count)
	return count > 0
}
