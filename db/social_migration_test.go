package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type migrationFriendRow struct {
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	UserId           uint32    `gorm:"primary_key;not null;uniqueIndex:friend"`
	FriendId         uint32    `gorm:"primary_key;not null;uniqueIndex:friend"`
	Status           int32     `gorm:"default:0"`
	Alias            string    `gorm:"default:''"`
	FriendTag        uint32    `gorm:"default:0"`
	FriendIntimacy   uint32    `gorm:"default:0"`
	FriendBackground uint32    `gorm:"default:0"`
}

func (migrationFriendRow) TableName() string { return "of_friend" }

type migrationBlackRow struct {
	UserId    uint32    `gorm:"primary_key;not null;uniqueIndex:black"`
	BlackId   uint32    `gorm:"primary_key;not null;uniqueIndex:black"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (migrationBlackRow) TableName() string { return "of_friend_black" }

type migrationPrivateRow struct {
	ID        int64     `gorm:"primary_key;auto_increment"`
	UserID1   uint32    `gorm:"uniqueIndex:user"`
	UserID2   uint32    `gorm:"uniqueIndex:user"`
	IsNewMsg  bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (migrationPrivateRow) TableName() string { return "of_chat_private" }

type migrationPrivateMsgRow struct {
	ID         int64  `gorm:"primary_key;auto_increment"`
	PrivateID  int64  `gorm:"not null;uniqueIndex:user_msg"`
	UserId     uint32 `gorm:"not null;uniqueIndex:user_msg"`
	SendTime   int64  `gorm:"not null"`
	Text       string
	Expression uint32
}

func (migrationPrivateMsgRow) TableName() string { return "of_chat_private_msg" }

func TestSocialMigrationPreservesLegacyDataAndIsRepeatable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = legacy.AutoMigrate(&migrationFriendRow{}, &migrationBlackRow{}, &migrationPrivateRow{}, &migrationPrivateMsgRow{}); err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`INSERT INTO of_friend (user_id, friend_id, status) VALUES (1, 3, 3), (4, 5, 2), (5, 4, 1), (1, 2, 2), (2, 1, 2)`,
		`INSERT INTO of_chat_private (id, user_id1, user_id2, is_new_msg) VALUES (1, 2, 1, 1), (2, 1, 2, 0), (3, 7, 8, 1)`,
		`INSERT INTO of_chat_private_msg (id, private_id, user_id, send_time, text, expression) VALUES
			(1, 1, 1, 10, 'legacy-a', 0), (2, 2, 2, 20, 'legacy-b', 0)`,
	}
	for _, statement := range statements {
		if err = legacy.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	legacySQL, err := legacy.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err = legacySQL.Close(); err != nil {
		t.Fatal(err)
	}

	previous := db
	t.Cleanup(func() {
		StopPersist()
		if db != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		}
		db = previous
	})
	openMigrated := func() {
		t.Helper()
		if err = NewDB(&Option{
			Type:            Sqlite,
			Dsn:             path,
			MaxIdleConns:    4,
			MaxOpenConns:    4,
			ConnMaxLifetime: 0,
			PersistShardNum: 1,
			PersistBufSize:  8,
		}); err != nil {
			t.Fatal(err)
		}
	}
	verify := func() {
		t.Helper()
		var blacks int64
		if err = db.Model(&OFFriendBlack{}).Where("user_id = ? AND black_id = ?", 1, 3).Count(&blacks).Error; err != nil || blacks != 1 {
			t.Fatalf("legacy black migration = %d, %v", blacks, err)
		}
		var blackFriends int64
		if err = db.Model(&OFFriend{}).
			Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", 1, 3, 3, 1).
			Count(&blackFriends).Error; err != nil || blackFriends != 0 {
			t.Fatalf("friend rows remained after black migration = %d, %v", blackFriends, err)
		}
		var repaired int64
		if err = db.Model(&OFFriend{}).
			Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
				4, 5, 5, 4, 2).Count(&repaired).Error; err != nil || repaired != 2 {
			t.Fatalf("one-way friend repair = %d, %v", repaired, err)
		}

		var conversations int64
		if err = db.Model(&OFChatPrivate{}).Count(&conversations).Error; err != nil || conversations != 2 {
			t.Fatalf("merged conversations = %d, %v", conversations, err)
		}
		private, privateErr := GetChatPrivate(1, 2)
		if privateErr != nil {
			t.Fatal(privateErr)
		}
		if private.UserID1 != 1 || private.UserID2 != 2 || private.UnreadUser1 != 1 || private.UnreadUser2 != 0 {
			t.Fatalf("merged private row = %#v", private)
		}
		empty, emptyErr := GetChatPrivate(7, 8)
		if emptyErr != nil {
			t.Fatal(emptyErr)
		}
		if empty.UnreadUser1 != 1 || empty.UnreadUser2 != 1 {
			t.Fatalf("legacy unread without messages = %#v", empty)
		}
		var messages int64
		if err = db.Model(&OFChatPrivateMsg{}).Count(&messages).Error; err != nil || messages != 2 {
			t.Fatalf("legacy messages = %d, %v", messages, err)
		}
		indexes, indexErr := db.Migrator().GetIndexes(&OFChatPrivateMsg{})
		if indexErr != nil {
			t.Fatal(indexErr)
		}
		for _, index := range indexes {
			unique, uniqueOK := index.Unique()
			primaryKey, primaryKeyOK := index.PrimaryKey()
			if uniqueOK && unique && !(primaryKeyOK && primaryKey) &&
				(sameIndexColumns(index, "private_id", "user_id") || sameIndexColumns(index, "user_id", "private_id")) {
				t.Fatalf("legacy unique message index still exists: %s", index.Name())
			}
		}
	}

	openMigrated()
	verify()
	if err = Close(); err != nil {
		t.Fatal(err)
	}
	openMigrated()
	verify()
	history, historyErr := GetAllChatPrivateMsg(1, 2)
	if historyErr != nil {
		t.Fatal(historyErr)
	}
	if len(history) != 2 || history[0].Text != "legacy-a" || history[1].Text != "legacy-b" {
		t.Fatalf("merged legacy history = %#v", history)
	}
}
