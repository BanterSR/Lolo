package model

import (
	"math"
	"testing"
)

func TestItemGrantsRejectNonPositiveAmounts(t *testing.T) {
	player := &Player{}
	if player.AddAllTypeItem(1, 0) != nil {
		t.Fatal("zero item grant must be rejected")
	}
	if player.AddAllTypeItem(1, -1) != nil {
		t.Fatal("negative item grant must be rejected")
	}
	if player.GetItemModel().AddItemBase(1, 0) != nil {
		t.Fatal("zero base item grant must be rejected")
	}
	if player.GetItemModel().AddItemFashion(0) != nil {
		t.Fatal("fashion id zero must be rejected without a configuration lookup")
	}
}

func TestItemTransactionRejectsInvalidDeductions(t *testing.T) {
	items := DefaultItemModel()
	items.ItemBaseInfo = map[uint32]*ItemBaseInfo{
		1: {ItemId: 1, Num: 10},
	}
	tx, err := items.Begin()
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	tx.DelBaseItem(1, -1)
	if tx.Error == nil {
		t.Fatal("negative deduction must fail")
	}
	tx.Rollback()
	if got := items.GetItemBaseInfo(1).Num; got != 10 {
		t.Fatalf("rollback changed balance to %d", got)
	}
}

func TestItemTransactionRejectsOverflowAndReentrantLock(t *testing.T) {
	items := DefaultItemModel()
	items.ItemBaseInfo = map[uint32]*ItemBaseInfo{
		1: {ItemId: 1, Num: math.MaxInt64},
	}
	tx, err := items.Begin()
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	if _, err := items.Begin(); err == nil {
		t.Fatal("a second transaction must not enter while the first holds the lock")
	}
	tx.DelBaseItem(1, math.MaxInt64-5)
	tx.DelBaseItem(1, 10)
	if tx.Error == nil {
		t.Fatal("overflowing cumulative deduction must fail")
	}
	tx.Rollback()
	if got := items.GetItemBaseInfo(1).Num; got != math.MaxInt64 {
		t.Fatalf("rollback changed balance to %d", got)
	}
}
