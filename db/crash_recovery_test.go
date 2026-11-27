package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWALRecoveryAfterAbruptProcessExit(t *testing.T) {
	if os.Getenv("LOLO_WAL_CRASH_HELPER") == "1" {
		path := os.Getenv("LOLO_WAL_CRASH_DB")
		conn, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)")
		if err != nil {
			os.Exit(2)
		}
		if _, err = conn.Exec("CREATE TABLE crash_rows (id INTEGER PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
			os.Exit(2)
		}
		tx, err := conn.Begin()
		if err != nil {
			os.Exit(2)
		}
		if _, err = tx.Exec("INSERT INTO crash_rows(value) VALUES (?)", "committed"); err != nil {
			os.Exit(2)
		}
		if err = tx.Commit(); err != nil {
			os.Exit(2)
		}
		tx, err = conn.Begin()
		if err != nil {
			os.Exit(2)
		}
		if _, err = tx.Exec("INSERT INTO crash_rows(value) VALUES (?)", "uncommitted"); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}

	path := filepath.Join(t.TempDir(), "crash.db")
	cmd := exec.Command(os.Args[0], "-test.run=^TestWALRecoveryAfterAbruptProcessExit$")
	cmd.Env = append(os.Environ(), "LOLO_WAL_CRASH_HELPER=1", "LOLO_WAL_CRASH_DB="+path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("crash helper failed: %v\n%s", err, output)
	}

	conn, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var integrity string
	if err := conn.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check = %q, want ok", integrity)
	}
	var count int
	if err := conn.QueryRow("SELECT count(*) FROM crash_rows").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("recovered %d rows, want only the committed row", count)
	}
	var value string
	if err := conn.QueryRow("SELECT value FROM crash_rows").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "committed" {
		t.Fatalf("recovered value = %q, want committed", value)
	}
	if _, err := conn.Exec("INSERT INTO crash_rows(value) VALUES (?)", "after-recovery"); err != nil {
		t.Fatalf("database not writable after recovery: %v", err)
	}
	var recoveredID int64
	if err := conn.QueryRow("SELECT id FROM crash_rows WHERE value = ?", "after-recovery").Scan(&recoveredID); err != nil {
		t.Fatal(err)
	}
	if recoveredID < 1 {
		t.Fatal(fmt.Sprintf("invalid recovered id %d", recoveredID))
	}
}
