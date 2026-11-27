package db

import (
	"sync"
	"testing"
)

func TestPersistAndWaitKeepsPlayerWriteOrder(t *testing.T) {
	StartPersist(2, 8)
	defer StopPersist()
	var mu sync.Mutex
	order := make([]int, 0, 4)
	write := func(value int) func() error {
		return func() error {
			mu.Lock()
			order = append(order, value)
			mu.Unlock()
			return nil
		}
	}
	Persist(7, write(1))
	if err := PersistAndWait(7, write(2)); err != nil {
		t.Fatal(err)
	}
	Persist(7, write(3))
	if err := PersistAndWait(7, write(4)); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	for index, value := range order {
		if value != index+1 {
			t.Fatalf("write order = %v, want [1 2 3 4]", order)
		}
	}
}
