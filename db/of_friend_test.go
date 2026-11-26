package db

import (
	"errors"
	"sync"
	"testing"

	"gucooing/lolo/protocol/proto"
)

func TestFriendApplyAcceptDeleteConsistency(t *testing.T) {
	setupSocialTestDB(t)
	if err := CreateFriendApply(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := CreateFriendApply(1, 2); err != nil {
		t.Fatalf("duplicate apply: %v", err)
	}
	var applies int64
	if err := db.Model(&OFFriend{}).Where("status = ?", proto.FriendStatus_FriendStatus_Apply).Count(&applies).Error; err != nil {
		t.Fatal(err)
	}
	if applies != 1 {
		t.Fatalf("apply rows = %d, want 1", applies)
	}
	if err := FriendHandleApply(2, 1, true); err != nil {
		t.Fatal(err)
	}
	if err := FriendHandleApply(2, 1, true); err != nil {
		t.Fatalf("duplicate accept: %v", err)
	}
	for _, pair := range [][2]uint32{{1, 2}, {2, 1}} {
		count, err := GetIsFiend(pair[0], pair[1])
		if err != nil || count != 1 {
			t.Fatalf("GetIsFiend(%d, %d) = %d, %v", pair[0], pair[1], count, err)
		}
	}
	if err := DelFiend(1, 2); err != nil {
		t.Fatal(err)
	}
	var remaining int64
	if err := db.Model(&OFFriend{}).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", 1, 2, 2, 1).
		Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("friend rows after delete = %d, want 0", remaining)
	}
}

func TestFriendRejectRemovesOnlyApplication(t *testing.T) {
	setupSocialTestDB(t)
	if err := CreateFriendApply(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := FriendHandleApply(2, 1, false); err != nil {
		t.Fatal(err)
	}
	count, err := GetIsFriendApply(1, 2)
	if err != nil || count != 0 {
		t.Fatalf("apply after reject = %d, %v", count, err)
	}
	if err = FriendHandleApply(2, 1, false); !errors.Is(err, ErrFriendNotApply) {
		t.Fatalf("duplicate reject error = %v", err)
	}
}

func TestMutualFriendApplicationsCollapseOnAccept(t *testing.T) {
	setupSocialTestDB(t)
	if err := CreateFriendApply(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := CreateFriendApply(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := FriendHandleApply(2, 1, true); err != nil {
		t.Fatal(err)
	}
	var applies int64
	if err := db.Model(&OFFriend{}).Where("status = ?", proto.FriendStatus_FriendStatus_Apply).Count(&applies).Error; err != nil {
		t.Fatal(err)
	}
	if applies != 0 {
		t.Fatalf("stale applies = %d, want 0", applies)
	}
	count, err := GetIsFiend(1, 2)
	if err != nil || count != 1 {
		t.Fatalf("mutual friend = %d, %v", count, err)
	}
}

func TestFriendConcurrentDuplicateOperations(t *testing.T) {
	setupSocialTestDB(t)
	const workers = 16
	run := func(fn func() error) {
		t.Helper()
		var wg sync.WaitGroup
		errs := make(chan error, workers)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- fn()
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	run(func() error { return CreateFriendApply(1, 2) })
	var applies int64
	if err := db.Model(&OFFriend{}).Where("status = ?", proto.FriendStatus_FriendStatus_Apply).Count(&applies).Error; err != nil {
		t.Fatal(err)
	}
	if applies != 1 {
		t.Fatalf("concurrent apply rows = %d, want 1", applies)
	}
	run(func() error { return FriendHandleApply(2, 1, true) })
	var friends int64
	if err := db.Model(&OFFriend{}).Where("status = ?", proto.FriendStatus_FriendStatus_Friend).Count(&friends).Error; err != nil {
		t.Fatal(err)
	}
	if friends != 2 {
		t.Fatalf("concurrent accept friend rows = %d, want 2", friends)
	}
	run(func() error { return DelFiend(2, 1) })
	var remaining int64
	if err := db.Model(&OFFriend{}).Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("concurrent delete rows = %d, want 0", remaining)
	}
}

func TestBlacklistBlocksFriendAndPrivateChatAndCanBeRemoved(t *testing.T) {
	setupSocialTestDB(t)
	makeSocialFriends(t, 1, 2)
	sendPrivateTestMsg(t, 1, 2, 1, "before block")
	if err := CreateFriendBlack(1, 2, false); err != nil {
		t.Fatal(err)
	}
	if err := CreateFriendBlack(1, 2, false); err != nil {
		t.Fatalf("duplicate block: %v", err)
	}
	blocked, err := IsFriendBlocked(2, 1)
	if err != nil || !blocked {
		t.Fatalf("reverse block check = %v, %v", blocked, err)
	}
	count, err := GetIsFiend(1, 2)
	if err != nil || count != 0 {
		t.Fatalf("friend after block = %d, %v", count, err)
	}
	if err = CreateFriendApply(1, 2); !errors.Is(err, ErrFriendBlack) {
		t.Fatalf("blocked apply error = %v", err)
	}
	if err = CreateFriendApply(2, 1); !errors.Is(err, ErrFriendBlack) {
		t.Fatalf("reverse blocked apply error = %v", err)
	}
	if err = FriendHandleApply(2, 1, true); !errors.Is(err, ErrFriendBlack) {
		t.Fatalf("blocked accept error = %v", err)
	}
	for _, pair := range [][2]uint32{{1, 2}, {2, 1}} {
		err = CreateChatPrivateMsg(pair[1], &OFChatPrivateMsg{UserId: pair[0], OFChatMsg: &OFChatMsg{SendTime: 2, Text: "blocked"}})
		if !errors.Is(err, ErrPrivateChatBlocked) {
			t.Fatalf("blocked chat %d -> %d error = %v", pair[0], pair[1], err)
		}
	}
	if err = CreateFriendBlack(1, 2, true); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if err = CreateFriendBlack(1, 2, true); err != nil {
		t.Fatalf("duplicate unblock: %v", err)
	}
	blocked, err = IsFriendBlocked(1, 2)
	if err != nil || blocked {
		t.Fatalf("block after unblock = %v, %v", blocked, err)
	}
	if err = CreateFriendApply(2, 1); err != nil {
		t.Fatalf("apply after unblock: %v", err)
	}
	history, err := GetAllChatPrivateMsg(2, 1)
	if err != nil || len(history) != 1 || history[0].Text != "before block" {
		t.Fatalf("history after block/unblock = %#v, %v", history, err)
	}
}
