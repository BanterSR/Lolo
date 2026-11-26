package db

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gucooing/lolo/protocol/proto"
)

var (
	ErrInvalidSocialTarget = errors.New("社交目标不能是自己")
	ErrFriendBlack         = errors.New("玩家已被拉黑")
	ErrFriendExists        = errors.New("好友关系已存在")
	ErrFriendNotApply      = errors.New("好友申请不存在")
)

// 好友配置表
type OFFriendInfo struct {
	UserId    uint32    `gorm:"primary_key;not null;index"` // 用户id
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// 好友关系表。申请和好友都使用有向行，好友关系必须同时存在反向行。
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
判断是否已有好友申请
requestUserId - 发起人
recipientId - 接收人
*/
func GetIsFriendApply(requestUserId, recipientId uint32) (int64, error) {
	var count int64
	err := db.Model(&OFFriend{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", requestUserId, recipientId, proto.FriendStatus_FriendStatus_APPLY).
		Count(&count).Error
	return count, err
}

func validateSocialTarget(userId, targetId uint32) error {
	if userId == 0 || targetId == 0 || userId == targetId {
		return ErrInvalidSocialTarget
	}
	return nil
}

func isFriendBlockedTx(tx *gorm.DB, userId, targetId uint32) (bool, error) {
	var count int64
	err := tx.Model(&OFFriendBlack{}).
		Where("(user_id = ? AND black_id = ?) OR (user_id = ? AND black_id = ?)", userId, targetId, targetId, userId).
		Count(&count).Error
	return count > 0, err
}

// IsFriendBlocked 判断任意一方是否拉黑了另一方。
func IsFriendBlocked(userId, targetId uint32) (bool, error) {
	return isFriendBlockedTx(db, userId, targetId)
}

/*
创建好友申请。重复申请不会写入重复行。
requestUserId - 发起人
recipientId - 接收人
*/
func CreateFriendApply(requestUserId, recipientId uint32) error {
	if err := validateSocialTarget(requestUserId, recipientId); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// 先获取写锁，避免 SQLite 上“先读后写”的并发申请互相阻塞。
		apply := &OFFriend{
			UserId:   requestUserId,
			FriendId: recipientId,
			Status:   proto.FriendStatus_FriendStatus_APPLY,
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "friend_id"}},
			DoNothing: true,
		}).Create(apply)
		if result.Error != nil {
			return result.Error
		}

		blocked, err := isFriendBlockedTx(tx, requestUserId, recipientId)
		if err != nil {
			return err
		}
		if blocked {
			return ErrFriendBlack
		}

		var friends int64
		if err = tx.Model(&OFFriend{}).
			Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
				requestUserId, recipientId, recipientId, requestUserId, proto.FriendStatus_FriendStatus_FRIEND).
			Count(&friends).Error; err != nil {
			return err
		}
		if friends != 0 {
			return ErrFriendExists
		}

		var current OFFriend
		if err = tx.Where("user_id = ? AND friend_id = ?", requestUserId, recipientId).First(&current).Error; err != nil {
			return err
		}
		if current.Status == proto.FriendStatus_FriendStatus_FRIEND {
			return ErrFriendExists
		}
		if current.Status != proto.FriendStatus_FriendStatus_APPLY {
			return fmt.Errorf("invalid friend status: %d", current.Status)
		}
		return nil
	})
}

func upsertFriendTx(tx *gorm.DB, userId, friendId uint32) error {
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "friend_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"updated_at",
		}),
	}).Create(&OFFriend{
		UserId:   userId,
		FriendId: friendId,
		Status:   proto.FriendStatus_FriendStatus_FRIEND,
	}).Error
}

func areFriendsTx(tx *gorm.DB, userId, friendId uint32) (bool, error) {
	var count int64
	err := tx.Model(&OFFriend{}).
		Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
			userId, friendId, friendId, userId, proto.FriendStatus_FriendStatus_FRIEND).
		Count(&count).Error
	return count == 2, err
}

/*
被申请玩家处理好友申请。同意时在一个事务内生成双向好友关系。
recipientId - 接收人
requestUserId - 发起人
*/
func FriendHandleApply(recipientId, requestUserId uint32, isAgree bool) error {
	if err := validateSocialTarget(recipientId, requestUserId); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// 先执行条件写获取行锁；后续黑名单校验失败时事务会回滚该写入。
		query := tx.Model(&OFFriend{}).
			Where("user_id = ? AND friend_id = ? AND status = ?",
				requestUserId, recipientId, proto.FriendStatus_FriendStatus_APPLY)
		if isAgree {
			result := query.Update("status", proto.FriendStatus_FriendStatus_FRIEND)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				blocked, blockErr := isFriendBlockedTx(tx, recipientId, requestUserId)
				if blockErr != nil {
					return blockErr
				}
				if blocked {
					return ErrFriendBlack
				}
				// 重复接受已经完成的申请视为成功。
				friends, friendErr := areFriendsTx(tx, recipientId, requestUserId)
				if friendErr != nil {
					return friendErr
				}
				if friends {
					return nil
				}
				return ErrFriendNotApply
			}
		} else {
			result := query.Delete(&OFFriend{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrFriendNotApply
			}
			return nil
		}

		blocked, err := isFriendBlockedTx(tx, recipientId, requestUserId)
		if err != nil {
			return err
		}
		if blocked {
			return ErrFriendBlack
		}

		var request OFFriend
		if err = tx.Where("user_id = ? AND friend_id = ?", requestUserId, recipientId).First(&request).Error; err != nil {
			return err
		}
		if err = upsertFriendTx(tx, request.UserId, request.FriendId); err != nil {
			return err
		}
		if err = upsertFriendTx(tx, request.FriendId, request.UserId); err != nil {
			return err
		}
		// 双方同时申请时，接受其中一条后不能留下另一条过期申请。
		return tx.Where("user_id = ? AND friend_id = ? AND status = ?",
			recipientId, requestUserId, proto.FriendStatus_FriendStatus_APPLY).
			Delete(&OFFriend{}).Error
	})
}

// 获取目标玩家的全部好友
func GetAllFiend(userId uint32) ([]*OFFriend, error) {
	list := make([]*OFFriend, 0)
	err := db.Where("user_id = ? AND status = ?", userId, proto.FriendStatus_FriendStatus_FRIEND).Find(&list).Error
	return list, err
}

// 判断是否存在一致的双向好友关系
func GetIsFiend(userId, friendId uint32) (int64, error) {
	friends, err := areFriendsTx(db, userId, friendId)
	if err != nil || !friends {
		return 0, err
	}
	return 1, nil
}

// 尝试获取可能存在的好友关系
func GetFiend(userId, friendId uint32) (*OFFriend, error) {
	off := &OFFriend{}
	err := db.Where("user_id = ? AND friend_id = ? AND status = ?", userId, friendId, proto.FriendStatus_FriendStatus_FRIEND).
		First(&off).Error
	if err == nil {
		return off, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var count int64
	if err = db.Model(&OFFriendBlack{}).Where("user_id = ? AND black_id = ?", userId, friendId).Count(&count).Error; err != nil {
		return nil, err
	}
	if count != 0 {
		return &OFFriend{
			UserId:   userId,
			FriendId: friendId,
			Status:   proto.FriendStatus_FriendStatus_BLACK,
		}, nil
	}
	return nil, nil
}

// 单向更新好友关系
func UpOFFriend(userId, friendId uint32, fx func(off *OFFriend) bool) error {
	off, err := GetFiend(userId, friendId)
	if err != nil {
		return err
	}
	if off == nil || off.Status != proto.FriendStatus_FriendStatus_FRIEND {
		return ErrFriendNotApply
	}
	if !fx(off) {
		return nil
	}
	return db.Save(off).Error
}

// 删除好友关系，同时删除双向行；私聊历史不受影响。
func DelFiend(userId, friendId uint32) error {
	if err := validateSocialTarget(userId, friendId); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userId, friendId, friendId, userId).Delete(&OFFriend{}).Error
	})
}

// 好友黑名单表
type OFFriendBlack struct {
	UserId    uint32    `gorm:"primary_key;not null;uniqueIndex:black"`
	BlackId   uint32    `gorm:"primary_key;not null;uniqueIndex:black"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// CreateFriendBlack 拉黑玩家；isRemove 为 true 时解除当前玩家对目标的拉黑。
func CreateFriendBlack(userId, blackId uint32, isRemove bool) error {
	if err := validateSocialTarget(userId, blackId); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if isRemove {
			return tx.Where("user_id = ? AND black_id = ?", userId, blackId).Delete(&OFFriendBlack{}).Error
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "black_id"}},
			DoNothing: true,
		}).Create(&OFFriendBlack{UserId: userId, BlackId: blackId}).Error; err != nil {
			return err
		}
		// 拉黑后移除双方好友关系和申请，避免遗留申请被绕过黑名单接受。
		return tx.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userId, blackId, blackId, userId).Delete(&OFFriend{}).Error
	})
}

// 获取全部被拉黑的玩家
func GetAllFriendBlack(userId uint32) ([]*OFFriendBlack, error) {
	list := make([]*OFFriendBlack, 0)
	err := db.Where("user_id = ?", userId).Find(&list).Error
	return list, err
}

/*
判断玩家是否拉黑了目标玩家。
userId - 玩家
friendId - 目标玩家
*/
func IsUserBlack(userId uint32, blackId uint32) (bool, error) {
	var count int64
	err := db.Model(&OFFriendBlack{}).Where("user_id = ? AND black_id = ?", userId, blackId).Count(&count).Error
	return count > 0, err
}

// migrateFriendData 将旧版状态黑名单迁入独立黑名单表，并补齐历史单向好友关系。
// 该迁移可重复执行；遇到自关联等无法安全合并的数据时直接报错并保留原数据。
func migrateFriendData(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		var rows []*OFFriend
		if err := tx.Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if row.UserId == 0 || row.FriendId == 0 || row.UserId == row.FriendId {
				return fmt.Errorf("invalid friend row user_id=%d friend_id=%d", row.UserId, row.FriendId)
			}
			if row.Status != proto.FriendStatus_FriendStatus_BLACK {
				continue
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "black_id"}},
				DoNothing: true,
			}).Create(&OFFriendBlack{UserId: row.UserId, BlackId: row.FriendId}).Error; err != nil {
				return err
			}
		}

		var blacks []*OFFriendBlack
		if err := tx.Find(&blacks).Error; err != nil {
			return err
		}
		for _, black := range blacks {
			if black.UserId == 0 || black.BlackId == 0 || black.UserId == black.BlackId {
				return fmt.Errorf("invalid black row user_id=%d black_id=%d", black.UserId, black.BlackId)
			}
			if err := tx.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
				black.UserId, black.BlackId, black.BlackId, black.UserId).Delete(&OFFriend{}).Error; err != nil {
				return err
			}
		}

		// 旧版可能只写入申请方的好友行，这里补齐反向行并清除反向过期申请。
		if err := tx.Where("status = ?", proto.FriendStatus_FriendStatus_FRIEND).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if err := upsertFriendTx(tx, row.FriendId, row.UserId); err != nil {
				return err
			}
		}
		return nil
	})
}
