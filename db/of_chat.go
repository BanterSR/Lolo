package db

import (
	"time"
)

// 玩家私聊数据库
type OFChatPrivate struct {
	ID        int64     `gorm:"primary_key;auto_increment"`
	UserID1   uint32    `gorm:"uniqueIndex:user"`
	UserID2   uint32    `gorm:"uniqueIndex:user"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type OFChatMsg struct {
	UserId     uint32 `gorm:"not null"`
	SendTime   int64  `gorm:"not null"`
	Text       string
	Expression uint32
}

type OFChatPrivateMsg struct {
	ID        int64 `gorm:"primary_key;auto_increment"`
	PrivateID int64 `gorm:"not null"` // 房间号
	*OFChatMsg
}
