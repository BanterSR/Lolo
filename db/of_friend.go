package db

import (
	"time"

	"gorm.io/gorm"
)

// 好友配置表
type OFFriendInfo struct {
	UserId    uint32    `gorm:"primary_key;not null"` // 用户id
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// 好友申请表
type OFFriendRequest struct {
	SenderUserId  uint32    `gorm:"primary_key;not null;uniqueIndex:request"` // 申请者
	RequestUserId uint32    `gorm:"primary_key;not null;uniqueIndex:request"` // 被申请者
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// 获取向目标玩家申请好友列表
func GetAllFriendApply(userId uint32) ([]*OFFriendRequest, error) {
	list := make([]*OFFriendRequest, 0)
	err := db.Where("request_user_id = ?", userId).Find(&list).Error
	return list, err
}

// 获取目标玩家申请了那些好友列表
func GetAllFriendSenderApply(userId uint32) ([]*OFFriendRequest, error) {
	list := make([]*OFFriendRequest, 0)
	err := db.Where("sender_user_id = ?", userId).Find(&list).Error
	return list, err
}

// 判断是否已有请求
func GetIsFriendApply(userId, senderId uint32) (int64, error) {
	var count int64
	err := db.Where("sender_user_id = ? AND request_user_id = ?", senderId, userId).Count(&count).Error
	return count, err
}

func CreateFriendApply(senderId, requestUserId uint32) error {
	return db.Create(&OFFriendRequest{
		SenderUserId:  senderId,
		RequestUserId: requestUserId,
	}).Error
}

// 好友关系表
type OFFriend struct {
	UserId           uint32    `gorm:"primary_key;not null;uniqueIndex:friend"` // 用户id
	FriendId         uint32    `gorm:"primary_key;not null;uniqueIndex:friend"` // 用户id的好友id
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	Alias            string    `gorm:"default:''"` // 别名
	FriendTag        uint32    `gorm:"default:0"`  // 好友标签
	FriendIntimacy   uint32    `gorm:"default:0"`  // 亲密度
	FriendBackground uint32    `gorm:"default:0"`  // 好友背景
}

// 被申请玩家处理好友申请
func FriendHandleApply(userId, senderId uint32, isAgree bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 获取是否存在该申请
		request := &OFFriendRequest{}
		err := tx.
			Where("sender_user_id = ? AND request_user_id = ?", senderId, userId).
			First(&request).Error
		if err != nil {
			return err
		}
		if isAgree {
			// 同意好友
			tx.Create(&OFFriend{
				UserId:   request.RequestUserId,
				FriendId: request.SenderUserId,
			})
			tx.Create(&OFFriend{
				UserId:   request.SenderUserId,
				FriendId: request.RequestUserId,
			})
			if tx.Error != nil {
				return tx.Error
			}
		}
		return tx.Delete(request).Error
	})
}

// 获取目标玩家的全部好友
func GetAllFiend(userId uint32) ([]*OFFriend, error) {
	list := make([]*OFFriend, 0)
	err := db.Where("user_id = ?", userId).Find(&list).Error
	return list, err
}

// 判断是否存在好友关系
func GetIsFiend(userId, friendId uint32) (int64, error) {
	var count int64
	err := db.Where("user_id = ? AND friend_id = ?", userId, friendId).Count(&count).Error
	return count, err
}

// 删除好友关系
func DelFiend(userId, friendId uint32) error {
	return db.Transaction(func(tx *gorm.DB) error {
		tx.Delete(&OFFriend{
			UserId:   userId,
			FriendId: friendId,
		})
		tx.Delete(&OFFriend{
			UserId:   friendId,
			FriendId: friendId,
		})
		return tx.Error
	})
}

// 好友黑名单表
type OFFriendBlack struct {
	UserId    uint32    `gorm:"primary_key;not null;uniqueIndex:black"`
	BlackId   uint32    `gorm:"primary_key;not null;uniqueIndex:black"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// 拉黑玩家
func CreateFriendBlack(userId, blackId uint32, isRemove bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&OFFriendBlack{
			UserId:  userId,
			BlackId: blackId,
		}).Error; err != nil {
			return err
		}
		if isRemove {
			tx.Delete(&OFFriend{
				UserId:   userId,
				FriendId: blackId,
			})
			tx.Delete(&OFFriend{
				UserId:   blackId,
				FriendId: userId,
			})
		}
		return tx.Error
	})
}

// 获取全部被拉黑的玩家
func GetAllFriendBlack(userId uint32) ([]*OFFriendBlack, error) {
	list := make([]*OFFriendBlack, 0)
	err := db.Where("user_id = ?", userId).Find(&list).Error
	return list, err
}
