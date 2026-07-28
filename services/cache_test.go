package services

import (
	"sync"
	"testing"
)

func TestXwTableConcurrentAccess(t *testing.T) {
	table := NewXwTable()
	const workers = 32
	const iterations = 200

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				table.Incrby("counter", 1)
				table.SetExString("message", 60, "ok")
				_ = table.GetInt("counter")
				_ = table.GetString("message")
				_ = table.KeyIsExpire("counter")
			}
		}()
	}
	wg.Wait()

	if got, want := table.GetInt("counter"), int64(workers*iterations); got != want {
		t.Fatalf("counter = %d, want %d", got, want)
	}
}

func TestXwTableExpireKeepsValueTypeForDeletion(t *testing.T) {
	table := NewXwTable()
	table.SetExInt("number", 0, 7)
	if err := table.Expire("number", 60); err != nil {
		t.Fatalf("Expire() error = %v", err)
	}
	if err := table.DelKey("number"); err != nil {
		t.Fatalf("DelKey() error = %v", err)
	}
	if got := table.GetInt("number"); got != 0 {
		t.Fatalf("GetInt() = %d after deletion, want 0", got)
	}
}

func TestXwTableSwitchingValueTypeRemovesOldValue(t *testing.T) {
	table := NewXwTable()
	table.SetExInt("value", 0, 7)
	table.SetExString("value", 0, "seven")

	if got := table.GetInt("value"); got != 0 {
		t.Fatalf("stale integer value = %d, want 0", got)
	}
	if got := table.GetString("value"); got != "seven" {
		t.Fatalf("GetString() = %q, want seven", got)
	}
}
