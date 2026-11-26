package db

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func openSocialTestGorm(t *testing.T, path string) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(path+"?_pragma=busy_timeout(5000)"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	if err = gdb.AutoMigrate(&OFFriend{}, &OFFriendBlack{}, &OFChatPrivate{}, &OFChatPrivateMsg{}); err != nil {
		_ = sqlDB.Close()
		t.Fatal(err)
	}
	if err = migrateFriendData(gdb); err != nil {
		_ = sqlDB.Close()
		t.Fatal(err)
	}
	if err = migrateChatData(gdb); err != nil {
		_ = sqlDB.Close()
		t.Fatal(err)
	}
	return gdb
}

func setupSocialTestDB(t *testing.T) {
	t.Helper()
	previous := db
	path := filepath.Join(t.TempDir(), "social.db")
	gdb := openSocialTestGorm(t, path)
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	db = gdb
	t.Cleanup(func() {
		_ = sqlDB.Close()
		db = previous
	})
}

func makeSocialFriends(t *testing.T, userID, friendID uint32) {
	t.Helper()
	if err := CreateFriendApply(userID, friendID); err != nil {
		t.Fatal(err)
	}
	if err := FriendHandleApply(friendID, userID, true); err != nil {
		t.Fatal(err)
	}
}

func sendPrivateTestMsg(t *testing.T, sender, receiver uint32, sendTime int64, text string) {
	t.Helper()
	if err := CreateChatPrivateMsg(receiver, &OFChatPrivateMsg{
		UserId: sender,
		OFChatMsg: &OFChatMsg{
			SendTime: sendTime,
			Text:     text,
		},
	}); err != nil {
		t.Fatal(err)
	}
}
