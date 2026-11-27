package db

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gucooing/lolo/protocol/proto"
)

var (
	ErrPrivateChatBlocked    = errors.New("私聊已被黑名单阻止")
	ErrPrivateChatNotAllowed = errors.New("只能与双向好友私聊")
	ErrChatPrivateForbidden  = errors.New("无权读取该私聊记录")
)

// 聊天消息基础结构
type OFChatMsg struct {
	SendTime   int64 `gorm:"not null"`
	Text       string
	Expression uint32
}

// 玩家私聊数据库。UserID1/UserID2 始终按大小排序，未读数分别属于会话双方。
type OFChatPrivate struct {
	ID          int64     `gorm:"primary_key;auto_increment"`
	UserID1     uint32    `gorm:"uniqueIndex:user"`
	UserID2     uint32    `gorm:"uniqueIndex:user"`
	UnreadUser1 uint32    `gorm:"not null;default:0"`
	UnreadUser2 uint32    `gorm:"not null;default:0"`
	IsNewMsg    bool      `gorm:"default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (p *OFChatPrivate) GetSubUserID(mainUserId uint32) uint32 {
	if p == nil {
		return 0
	}
	if p.UserID1 == mainUserId {
		return p.UserID2
	}
	if p.UserID2 == mainUserId {
		return p.UserID1
	}
	return 0
}

// GetUnreadCount 返回指定会话参与者的未读消息数，非参与者始终返回 0。
func (p *OFChatPrivate) GetUnreadCount(userId uint32) uint32 {
	if p == nil {
		return 0
	}
	switch userId {
	case p.UserID1:
		return p.UnreadUser1
	case p.UserID2:
		return p.UnreadUser2
	default:
		return 0
	}
}

func (p *OFChatPrivate) HasUnread(userId uint32) bool {
	return p.GetUnreadCount(userId) > 0
}

func getPrivateIndex(userId1, userId2 uint32) (uint32, uint32) {
	if userId1 > userId2 {
		return userId2, userId1
	}
	return userId1, userId2
}

// GetChatPrivate 查询一个已存在的私聊会话，不会在查询历史时创建空会话。
func GetChatPrivate(userId1, userId2 uint32) (*OFChatPrivate, error) {
	if err := validateSocialTarget(userId1, userId2); err != nil {
		return nil, err
	}
	userId1, userId2 = getPrivateIndex(userId1, userId2)
	private := &OFChatPrivate{}
	err := db.Where("user_id1 = ? AND user_id2 = ?", userId1, userId2).First(private).Error
	return private, err
}

func getOrCreateChatPrivateTx(tx *gorm.DB, userId1, userId2 uint32) (*OFChatPrivate, error) {
	userId1, userId2 = getPrivateIndex(userId1, userId2)
	private := &OFChatPrivate{UserID1: userId1, UserID2: userId2}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id1"}, {Name: "user_id2"}},
		DoNothing: true,
	}).Create(private).Error; err != nil {
		return nil, err
	}
	// 使用当前读取得并发冲突后已提交的会话行，并锁住它以串行更新双方未读数。
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id1 = ? AND user_id2 = ?", userId1, userId2).First(private).Error; err != nil {
		return nil, err
	}
	return private, nil
}

// 获取已创建的全部私聊频道
func GetAllChatPrivate(userId uint32) ([]*OFChatPrivate, error) {
	list := make([]*OFChatPrivate, 0)
	err := db.Where("user_id1 = ? OR user_id2 = ?", userId, userId).
		Order("created_at ASC, id ASC").Find(&list).Error
	return list, err
}

// 私聊记录表。同一发送者可以在同一会话中发送多条消息。
type OFChatPrivateMsg struct {
	ID        int64  `gorm:"primary_key;auto_increment"`
	PrivateID int64  `gorm:"not null;index:private_msg"`
	UserId    uint32 `gorm:"not null"`
	*OFChatMsg
}

// 获取私聊全部聊天记录。userId1 是读取方，读取成功后只清除该接收者的未读。
func GetAllChatPrivateMsg(userId1, userId2 uint32) ([]*OFChatPrivateMsg, error) {
	if err := validateSocialTarget(userId1, userId2); err != nil {
		return nil, err
	}
	readerID := userId1
	userId1, userId2 = getPrivateIndex(userId1, userId2)
	list := make([]*OFChatPrivateMsg, 0)
	err := db.Transaction(func(tx *gorm.DB) error {
		readerColumn := "unread_user2"
		otherColumn := "unread_user1"
		if readerID == userId1 {
			readerColumn = "unread_user1"
			otherColumn = "unread_user2"
		}
		// 先写未读状态以获取 SQLite 写锁；查询失败时整个事务回滚。
		result := tx.Model(&OFChatPrivate{}).
			Where("user_id1 = ? AND user_id2 = ?", userId1, userId2).
			Updates(map[string]interface{}{
				readerColumn: 0,
				"is_new_msg": gorm.Expr(otherColumn + " > 0"),
			})
		if result.Error != nil {
			return result.Error
		}

		private := &OFChatPrivate{}
		err := tx.Where("user_id1 = ? AND user_id2 = ?", userId1, userId2).First(private).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if (private.UserID1 != userId1 && private.UserID1 != userId2) ||
			(private.UserID2 != userId1 && private.UserID2 != userId2) ||
			private.UserID1 == private.UserID2 {
			return ErrChatPrivateForbidden
		}

		// 先取最新 100 条，再反转为与保存顺序一致的升序返回。
		if err = tx.Where("private_id = ?", private.ID).
			Order("send_time DESC, id DESC").Limit(100).Find(&list).Error; err != nil {
			return err
		}
		for left, right := 0, len(list)-1; left < right; left, right = left+1, right-1 {
			list[left], list[right] = list[right], list[left]
		}
		return nil
	})
	return list, err
}

/*
写入私聊消息。黑名单和双向好友校验与消息、未读更新处于同一事务。
userId 接收方
msg.UserId 发送方
*/
func CreateChatPrivateMsg(userId uint32, msg *OFChatPrivateMsg) error {
	if msg == nil || msg.OFChatMsg == nil {
		return errors.New("私聊消息不能为空")
	}
	if err := validateSocialTarget(msg.UserId, userId); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// 先创建/锁定会话以获取写锁；校验失败时事务回滚，不会留下空会话。
		private, err := getOrCreateChatPrivateTx(tx, msg.UserId, userId)
		if err != nil {
			return err
		}
		blocked, err := isFriendBlockedTx(tx, msg.UserId, userId)
		if err != nil {
			return err
		}
		if blocked {
			return ErrPrivateChatBlocked
		}

		// 锁定双方好友行，使拉黑删除关系与发送私聊形成确定的先后顺序。
		friends := make([]OFFriend, 0, 2)
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
				msg.UserId, userId, userId, msg.UserId, proto.FriendStatus_FriendStatus_FRIEND).
			Find(&friends).Error; err != nil {
			return err
		}
		if len(friends) != 2 {
			return ErrPrivateChatNotAllowed
		}
		blocked, err = isFriendBlockedTx(tx, msg.UserId, userId)
		if err != nil {
			return err
		}
		if blocked {
			return ErrPrivateChatBlocked
		}

		msg.PrivateID = private.ID
		if err = tx.Create(msg).Error; err != nil {
			return err
		}

		recipientColumn := "unread_user1"
		if userId == private.UserID2 {
			recipientColumn = "unread_user2"
		}
		return tx.Model(&OFChatPrivate{}).Where("id = ?", private.ID).Updates(map[string]interface{}{
			recipientColumn: gorm.Expr(recipientColumn + " + 1"),
			"is_new_msg":    true,
		}).Error
	})
}

// 系统消息记录表
type OFChatSystemMsg struct {
	ID int64 `gorm:"primary_key;auto_increment;index:system_msg_id"`
	*OFChatMsg
}

type legacyChatPrivate struct {
	ID          int64
	UserID1     uint32
	UserID2     uint32
	UnreadUser1 uint32
	UnreadUser2 uint32
	IsNewMsg    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// prepareChatMigration 在 AutoMigrate 添加索引前合并重复/反向会话，保留全部消息。
func prepareChatMigration(gdb *gorm.DB) error {
	if !gdb.Migrator().HasTable(&OFChatPrivate{}) {
		return nil
	}
	return gdb.Transaction(func(tx *gorm.DB) error {
		rows := make([]*legacyChatPrivate, 0)
		if err := tx.Table("of_chat_private").Find(&rows).Error; err != nil {
			return err
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		canonical := make(map[[2]uint32]*legacyChatPrivate)
		hasMessages := tx.Migrator().HasTable(&OFChatPrivateMsg{})

		unreadFor := func(row *legacyChatPrivate, userID uint32) uint32 {
			if row.UserID1 == userID {
				return row.UnreadUser1
			}
			return row.UnreadUser2
		}
		for _, row := range rows {
			if row.UserID1 == 0 || row.UserID2 == 0 || row.UserID1 == row.UserID2 {
				return fmt.Errorf("invalid private chat row id=%d user_id1=%d user_id2=%d", row.ID, row.UserID1, row.UserID2)
			}
			user1, user2 := getPrivateIndex(row.UserID1, row.UserID2)
			key := [2]uint32{user1, user2}
			if current, ok := canonical[key]; ok {
				if hasMessages {
					if err := tx.Table("of_chat_private_msg").
						Where("private_id = ?", row.ID).Update("private_id", current.ID).Error; err != nil {
						return err
					}
				}
				current.UnreadUser1 += unreadFor(row, current.UserID1)
				current.UnreadUser2 += unreadFor(row, current.UserID2)
				current.IsNewMsg = current.IsNewMsg || row.IsNewMsg || current.UnreadUser1 > 0 || current.UnreadUser2 > 0
				if row.CreatedAt.Before(current.CreatedAt) {
					current.CreatedAt = row.CreatedAt
				}
				if row.UpdatedAt.After(current.UpdatedAt) {
					current.UpdatedAt = row.UpdatedAt
				}
				updates := map[string]interface{}{
					"is_new_msg": current.IsNewMsg,
					"created_at": current.CreatedAt,
					"updated_at": current.UpdatedAt,
				}
				if tx.Migrator().HasColumn("of_chat_private", "unread_user1") {
					updates["unread_user1"] = current.UnreadUser1
					updates["unread_user2"] = current.UnreadUser2
				}
				if err := tx.Table("of_chat_private").Where("id = ?", current.ID).Updates(updates).Error; err != nil {
					return err
				}
				if err := tx.Exec("DELETE FROM of_chat_private WHERE id = ?", row.ID).Error; err != nil {
					return err
				}
				continue
			}
			canonical[key] = row
		}

		// 所有反向重复行删除后再统一排序，避免更新时撞上另一方向的唯一索引。
		for _, row := range canonical {
			user1, user2 := getPrivateIndex(row.UserID1, row.UserID2)
			if user1 == row.UserID1 && user2 == row.UserID2 {
				continue
			}
			if err := tx.Table("of_chat_private").Where("id = ?", row.ID).Updates(map[string]interface{}{
				"user_id1": user1,
				"user_id2": user2,
			}).Error; err != nil {
				return err
			}
			if tx.Migrator().HasColumn("of_chat_private", "unread_user1") {
				if err := tx.Table("of_chat_private").Where("id = ?", row.ID).Updates(map[string]interface{}{
					"unread_user1": row.UnreadUser2,
					"unread_user2": row.UnreadUser1,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func sameIndexColumns(index gorm.Index, columns ...string) bool {
	actual := index.Columns()
	if len(actual) != len(columns) {
		return false
	}
	for i := range columns {
		if actual[i] != columns[i] {
			return false
		}
	}
	return true
}

// migrateChatData 迁移旧版共享未读标记，并移除会错误限制同一发送者消息数的唯一索引。
func migrateChatData(gdb *gorm.DB) error {
	if err := prepareChatMigration(gdb); err != nil {
		return err
	}
	return gdb.Transaction(func(tx *gorm.DB) error {
		indexes, err := tx.Migrator().GetIndexes(&OFChatPrivateMsg{})
		if err != nil {
			return err
		}
		for _, index := range indexes {
			unique, uniqueOK := index.Unique()
			primaryKey, primaryKeyOK := index.PrimaryKey()
			if !uniqueOK || !unique || (primaryKeyOK && primaryKey) {
				continue
			}
			// 旧版 user_msg 唯一索引是 (private_id, user_id)，会阻止同一人发送第二条消息。
			if sameIndexColumns(index, "private_id", "user_id") || sameIndexColumns(index, "user_id", "private_id") {
				if err = tx.Migrator().DropIndex(&OFChatPrivateMsg{}, index.Name()); err != nil {
					return err
				}
			}
		}

		privates := make([]*OFChatPrivate, 0)
		if err = tx.Order("id ASC").Find(&privates).Error; err != nil {
			return err
		}
		for _, private := range privates {
			if private.UserID1 == 0 || private.UserID2 == 0 || private.UserID1 >= private.UserID2 {
				return fmt.Errorf("invalid private chat row id=%d user_id1=%d user_id2=%d", private.ID, private.UserID1, private.UserID2)
			}
			if private.UnreadUser1 == 0 && private.UnreadUser2 == 0 && private.IsNewMsg {
				var latest OFChatPrivateMsg
				if err = tx.Where("private_id = ?", private.ID).
					Order("send_time DESC, id DESC").Limit(1).Find(&latest).Error; err != nil {
					return err
				}
				switch {
				case latest.ID == 0:
					// 旧数据没有消息可推断接收者，保留未读提示并交给双方读取。
					private.UnreadUser1 = 1
					private.UnreadUser2 = 1
				case latest.UserId == private.UserID1:
					private.UnreadUser2 = 1
				case latest.UserId == private.UserID2:
					private.UnreadUser1 = 1
				default:
					return fmt.Errorf("private chat %d has message from non-participant %d", private.ID, latest.UserId)
				}
			}
			private.IsNewMsg = private.UnreadUser1 > 0 || private.UnreadUser2 > 0
			if err = tx.Model(&OFChatPrivate{}).Where("id = ?", private.ID).
				Select("unread_user1", "unread_user2", "is_new_msg").Updates(private).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
