package model

import "testing"

func TestAtomicItemTransactionRollsBackGrantsAndAllocator(t *testing.T) {
	items := DefaultItemModel()
	tx, err := items.BeginAtomic()
	if err != nil {
		t.Fatal(err)
	}
	items.NextInstanceIndex()
	if committed := tx.Commit(); committed == nil {
		t.Fatal("commit failed")
	}
	tx, err = items.BeginAtomic()
	if err != nil {
		t.Fatal(err)
	}
	items.NextInstanceIndex()
	items.GetItemBaseMap()[1] = &ItemBaseInfo{ItemId: 1, Num: 4}
	tx.Rollback()
	if items.InstanceIndex != 1 || items.GetItemBaseInfo(1) != nil {
		t.Fatalf("rollback retained mutation: index=%d base=%v", items.InstanceIndex, items.GetItemBaseInfo(1))
	}
	// The mutex must remain usable after rollback.
	next, err := items.BeginAtomic()
	if err != nil {
		t.Fatal(err)
	}
	next.Rollback()
}

func TestAtomicItemTransactionCommitErrorRestoresInventory(t *testing.T) {
	items := DefaultItemModel()
	items.GetItemBaseMap()[1] = &ItemBaseInfo{ItemId: 1, Num: 2}
	tx, err := items.BeginAtomic()
	if err != nil {
		t.Fatal(err)
	}
	items.GetItemBaseInfo(1).Num = 5
	tx.DelBaseItem(1, 6)
	tx.Commit()
	if tx.Error == nil {
		t.Fatal("invalid transaction committed")
	}
	if got := items.GetItemBaseInfo(1).Num; got != 2 {
		t.Fatalf("balance after failed commit = %d, want 2", got)
	}
}
