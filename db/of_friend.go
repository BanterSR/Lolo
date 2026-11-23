package db

import (
	"errors"
	"gucooing/lolo/protocol/proto"
	"time"

	"gorm.io/gorm"
)

// 好友配置表
type OFFriendInfo struct {
	UserId    uint32    `gorm:"primary_key;not null;index"` // 用户id
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// 好友关系表
type OFFriend struct {
	CreatedAt        time.Time          `gorm:"autoCreateTime"`
	UpdatedAt        time.Time          `gorm:"autoUpdateTime"`
	UserId           uint32             `gorm:"primary_key;not null;uniqueIndex:friend"` // 用户id
	FriendId         uint32             `gorm:"primary_key;not null;uniqueIndex:friend"` // 好友id / 被申请好友的id
	Status           proto.FriendStatus `gorm:"default:0"`                               // 好友关系
	Alias            string             `gorm:"default:''"`                              // 别名
	FriendTag        uint32             `gorm:"default:0"`                               // 好友标签
	FriendIntimacy   uint32             `gorm:"default:0"`                               // 亲密度
	FriendBackground uint32             `gorm:"default:0"`                               // 好友背景
}

// 获取向目标玩家申请好友列表
func GetAllFriendApply(userId uint32) ([]*OFFriend, error) {
	list := make([]*OFFriend, 0)
	err := db.Where("friend_id = ? AND status = ?", userId, proto.FriendStatus_FriendStatus_APPLY).Find(&list).Error
	return list, err
}

// 获取目标玩家申请了那些好友列表
func GetAllFriendSenderApply(userId uint32) ([]*OFFriend, error) {
	list := make([]*OFFriend, 0)
	err := db.Where("user_id = ? AND status = ?", userId, proto.FriendStatus_FriendStatus_APPLY).Find(&list).Error
	return list, err
}

// 获取玩家对应好友关系玩家列表
func GetAllFriendByStatus(userId uint32, status proto.FriendStatus) ([]*OFFriend, error) {
	list := make([]*OFFriend, 0)
	err := db.Where("user_id = ? AND status = ?", userId, status).Find(&list).Error
	return list, err
}

/*
判断是否已有好友关系
requestUserId - 发起人
recipientId - 接收人
*/
func GetIsFriendApply(requestUserId, recipientId uint32) (int64, error) {
	var count int64
	err := db.Model(&OFFriend{}).Where("user_id = ? AND friend_id = ?", requestUserId, recipientId).Count(&count).Error
	return count, err
}

/*
创建好友申请
requestUserId - 发起人
recipientId - 接收人
*/
func CreateFriendApply(requestUserId, recipientId uint32) error {
	return db.Create(&OFFriend{
		UserId:   requestUserId,
		FriendId: recipientId,
		Status:   proto.FriendStatus_FriendStatus_APPLY,
	}).Error
}

/*
被申请玩家处理好友申请
requestUserId - 发起人
recipientId - 接收人
*/
func FriendHandleApply(recipientId, requestUserId uint32, isAgree bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 获取是否存在该申请
		request := &OFFriend{}
		err := tx.
			Where("user_id = ? AND friend_id = ? AND status = ?", requestUserId, recipientId, proto.FriendStatus_FriendStatus_APPLY).
			First(&request).Error
		if err != nil {
			return err
		}
		if isAgree {
			// 同意好友
			request.Status = proto.FriendStatus_FriendStatus_FRIEND
			if err := tx.Save(request).Error; err != nil {
				return err
			}
			if err := tx.Create(&OFFriend{
				UserId:   request.FriendId,
				FriendId: request.UserId,
				Status:   proto.FriendStatus_FriendStatus_FRIEND,
			}).Error; err != nil {
				return err
			}
			return nil
		}
		return tx.Delete(request).Error
	})
}

// 获取目标玩家的全部好友
func GetAllFiend(userId uint32) ([]*OFFriend, error) {
	list := make([]*OFFriend, 0)
	err := db.Where("user_id = ? AND status = ?", userId, proto.FriendStatus_FriendStatus_FRIEND).Find(&list).Error
	return list, err
}

// 判断是否存在好友关系
func GetIsFiend(userId, friendId uint32) (int64, error) {
	var count int64
	err := db.Model(&OFFriend{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", userId, friendId, proto.FriendStatus_FriendStatus_FRIEND).
		Count(&count).Error
	return count, err
}

// 尝试获取可能存在的好友关系
func GetFiend(userId, friendId uint32) (*OFFriend, error) {
	off := &OFFriend{}
	err := db.Where("user_id = ? AND friend_id = ? AND status = ?", userId, friendId, proto.FriendStatus_FriendStatus_FRIEND).
		First(&off).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return off, err
}

// 单向更新好友关系
func UpOFFriend(userId, friendId uint32, fx func(off *OFFriend) bool) error {
	off, err := GetFiend(userId, friendId)
	if err != nil {
		return err
	}
	if !fx(off) {
		return nil
	}
	if err = db.Save(off).Error; err != nil {
		return err
	}
	return nil
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
		} else {
			tx.Where("user_id = ? AND friend_id = ?", userId, blackId).
				Update("status", proto.FriendStatus_FriendStatus_BLACK)
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

/*
判断玩家是否被目标玩家拉黑
userId - 玩家
friendId - 目标玩家
*/
func IsUserBlack(userId uint32, blackId uint32) (bool, error) {
	var count int64
	err := db.Model(&OFFriendBlack{}).Where("user_id = ? AND black_id = ?", userId, blackId).Count(&count).Error
	return count > 0, err
}
