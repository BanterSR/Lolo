package db

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"gucooing/lolo/protocol/proto"
)

func TestPrivateChatUnreadPerRecipientAndHistoryAfterFriendDelete(t *testing.T) {
	setupSocialTestDB(t)
	makeSocialFriends(t, 10, 2)
	sendPrivateTestMsg(t, 10, 2, 1, "first")
	sendPrivateTestMsg(t, 10, 2, 2, "second")
	sendPrivateTestMsg(t, 2, 10, 3, "reply")

	private, err := GetChatPrivate(10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if private.UserID1 != 2 || private.UserID2 != 10 {
		t.Fatalf("private users = %d, %d, want 2, 10", private.UserID1, private.UserID2)
	}
	if private.UnreadUser1 != 2 || private.UnreadUser2 != 1 {
		t.Fatalf("unread = %d/%d, want 2/1", private.UnreadUser1, private.UnreadUser2)
	}

	history, err := GetAllChatPrivateMsg(10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3", len(history))
	}
	for i, want := range []string{"first", "second", "reply"} {
		if history[i].Text != want || history[i].SendTime != int64(i+1) {
			t.Fatalf("history[%d] = %#v, want %q at %d", i, history[i], want, i+1)
		}
	}
	private, err = GetChatPrivate(2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if private.UnreadUser1 != 2 || private.UnreadUser2 != 0 || !private.IsNewMsg {
		t.Fatalf("unread after user 10 read = %d/%d/%v", private.UnreadUser1, private.UnreadUser2, private.IsNewMsg)
	}

	if err = DelFiend(10, 2); err != nil {
		t.Fatal(err)
	}
	history, err = GetAllChatPrivateMsg(2, 10)
	if err != nil || len(history) != 3 {
		t.Fatalf("history after friend delete = %d, %v", len(history), err)
	}
	private, err = GetChatPrivate(10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if private.HasUnread(2) || private.HasUnread(10) || private.IsNewMsg {
		t.Fatalf("unread after both reads = %#v", private)
	}
	if err = CreateChatPrivateMsg(2, &OFChatPrivateMsg{UserId: 10, OFChatMsg: &OFChatMsg{SendTime: 4, Text: "not allowed"}}); !errors.Is(err, ErrPrivateChatNotAllowed) {
		t.Fatalf("chat after friend delete error = %v", err)
	}

	history, err = GetAllChatPrivateMsg(99, 2)
	if err != nil || len(history) != 0 {
		t.Fatalf("third party history = %d, %v", len(history), err)
	}
	var conversations int64
	if err = db.Model(&OFChatPrivate{}).Count(&conversations).Error; err != nil {
		t.Fatal(err)
	}
	if conversations != 1 {
		t.Fatalf("history query created conversations: %d", conversations)
	}
}

func TestPrivateChatUnreadSurvivesReopen(t *testing.T) {
	previous := db
	path := filepath.Join(t.TempDir(), "reopen.db")
	t.Cleanup(func() {
		if db != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		}
		db = previous
	})
	gdb := openSocialTestGorm(t, path)
	db = gdb
	makeSocialFriends(t, 1, 2)
	sendPrivateTestMsg(t, 1, 2, 1, "to-two")
	sendPrivateTestMsg(t, 1, 2, 2, "to-two-again")
	sendPrivateTestMsg(t, 2, 1, 3, "to-one")
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err = sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	gdb = openSocialTestGorm(t, path)
	db = gdb
	private, err := GetChatPrivate(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if private.UnreadUser1 != 1 || private.UnreadUser2 != 2 {
		t.Fatalf("unread after reopen = %d/%d, want 1/2", private.UnreadUser1, private.UnreadUser2)
	}
	if _, err = GetAllChatPrivateMsg(2, 1); err != nil {
		t.Fatal(err)
	}
	private, err = GetChatPrivate(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if private.UnreadUser1 != 1 || private.UnreadUser2 != 0 || !private.IsNewMsg {
		t.Fatalf("unread after receiver read = %#v", private)
	}
}

func TestPrivateChatConcurrentSendsShareOneConversation(t *testing.T) {
	setupSocialTestDB(t)
	makeSocialFriends(t, 1, 2)
	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errs <- CreateChatPrivateMsg(2, &OFChatPrivateMsg{
				UserId: 1,
				OFChatMsg: &OFChatMsg{
					SendTime: int64(index + 1),
					Text:     fmt.Sprintf("message-%d", index),
				},
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var conversations int64
	if err := db.Model(&OFChatPrivate{}).Count(&conversations).Error; err != nil {
		t.Fatal(err)
	}
	if conversations != 1 {
		t.Fatalf("conversations = %d, want 1", conversations)
	}
	var messages int64
	if err := db.Model(&OFChatPrivateMsg{}).Count(&messages).Error; err != nil {
		t.Fatal(err)
	}
	if messages != workers {
		t.Fatalf("messages = %d, want %d", messages, workers)
	}
	private, err := GetChatPrivate(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if private.UnreadUser1 != 0 || private.UnreadUser2 != workers {
		t.Fatalf("unread = %d/%d, want 0/%d", private.UnreadUser1, private.UnreadUser2, workers)
	}
}

func TestPrivateChatRequiresMutualFriendAndDoesNotCreateConversation(t *testing.T) {
	setupSocialTestDB(t)
	if err := CreateFriendApply(1, 2); err != nil {
		t.Fatal(err)
	}
	err := CreateChatPrivateMsg(2, &OFChatPrivateMsg{UserId: 1, OFChatMsg: &OFChatMsg{SendTime: 1, Text: "one-way"}})
	if !errors.Is(err, ErrPrivateChatNotAllowed) {
		t.Fatalf("one-way friend chat error = %v", err)
	}
	var conversations int64
	if err = db.Model(&OFChatPrivate{}).Count(&conversations).Error; err != nil {
		t.Fatal(err)
	}
	if conversations != 0 {
		t.Fatalf("failed chat created %d conversations", conversations)
	}
}

func TestPrivateChatHistoryReturnsLatest100InSendOrder(t *testing.T) {
	setupSocialTestDB(t)
	makeSocialFriends(t, 1, 2)
	for i := 1; i <= 105; i++ {
		sendPrivateTestMsg(t, 1, 2, int64(i), fmt.Sprintf("message-%03d", i))
	}
	history, err := GetAllChatPrivateMsg(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 100 {
		t.Fatalf("history length = %d, want 100", len(history))
	}
	for i, msg := range history {
		wantTime := int64(i + 6)
		if msg.SendTime != wantTime || msg.Text != fmt.Sprintf("message-%03d", wantTime) {
			t.Fatalf("history[%d] = %#v, want send time %d", i, msg, wantTime)
		}
	}
	private, err := GetChatPrivate(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if private.HasUnread(2) || private.HasUnread(1) || private.IsNewMsg {
		t.Fatalf("unread after receiver history read = %#v", private)
	}
}

func TestFriendMigrationRejectsSelfRelationship(t *testing.T) {
	setupSocialTestDB(t)
	if err := db.Create(&OFFriend{UserId: 1, FriendId: 1, Status: proto.FriendStatus_FriendStatus_Friend}).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateFriendData(db); err == nil {
		t.Fatal("self friend migration unexpectedly succeeded")
	}
	var count int64
	if err := db.Model(&OFFriend{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("dirty friend row was lost: count=%d err=%v", count, err)
	}
}

func TestChatMigrationRejectsSelfConversation(t *testing.T) {
	setupSocialTestDB(t)
	if err := db.Create(&OFChatPrivate{UserID1: 1, UserID2: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateChatData(db); err == nil {
		t.Fatal("self conversation migration unexpectedly succeeded")
	}
	var count int64
	if err := db.Model(&OFChatPrivate{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("dirty conversation was lost: count=%d err=%v", count, err)
	}
}
