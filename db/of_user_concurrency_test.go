package db

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCreateOFUserConcurrent(t *testing.T) {
	previous := db
	t.Cleanup(func() { db = previous })

	testDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "users.db")+"?_pragma=busy_timeout(5000)"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := testDB.AutoMigrate(&OFUser{}); err != nil {
		t.Fatal(err)
	}
	db = testDB
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	const count = 64
	ids := make(chan uint32, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(sdkUID uint32) {
			defer wg.Done()
			user, err := CreateOFUser(&OFUser{SdkUid: sdkUID})
			if err != nil {
				errs <- err
				return
			}
			ids <- user.UserId
		}(uint32(i))
	}
	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		t.Errorf("CreateOFUser failed: %v", err)
	}
	seen := make(map[uint32]struct{})
	for id := range ids {
		if id < 1000000 {
			t.Errorf("unexpected user id %d", id)
		}
		if _, ok := seen[id]; ok {
			t.Errorf("duplicate user id %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("created %d unique users, want %d", len(seen), count)
	}
}

func TestCreateAccountChainConcurrent(t *testing.T) {
	previous := db
	t.Cleanup(func() { db = previous })

	testDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "accounts.db")+"?_pragma=busy_timeout(5000)"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := testDB.AutoMigrate(&OFQuick{}, &OFQuickCheck{}, &OFGameBasic{}, &OFGame{}); err != nil {
		t.Fatal(err)
	}
	db = testDB
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	const count = 64
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			username := fmt.Sprintf("chain_%d", id)
			if _, err := OrCreateOFQuick(username, "test-password"); err != nil {
				errs <- err
				return
			}
			if _, err := OrCreateOFQuickCheck(fmt.Sprint(id)); err != nil {
				errs <- err
				return
			}
			if _, err := GetGameBasicOrCreate(id); err != nil {
				errs <- err
				return
			}
			if _, err := GetOFGameByUserId(id); err != nil {
				errs <- err
			}
		}(uint32(i))
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("account chain creation failed: %v", err)
	}
}
