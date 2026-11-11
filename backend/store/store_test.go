package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aion/backend/model"
)

// Store enforces a single-directory lock, rolls back failed write transactions, and persists committed rows across reopens.
func TestTransactionsReopenAndLock(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if other, err := Open(root); err == nil {
		other.Close()
		t.Fatal("same directory accepted twice")
	}
	sentinel := errors.New("rollback")
	err = s.Write(ctx, func(tx *sql.Tx) error {
		_, e := tx.Exec("INSERT INTO settings VALUES('app.test','1')")
		if e != nil {
			return e
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	err = s.Read(ctx, func(tx *sql.Tx) error {
		var count int
		if e := tx.QueryRow("SELECT count(*) FROM settings").Scan(&count); e != nil {
			return e
		}
		if count != 0 {
			t.Fatal("rollback lost")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Write(ctx, func(tx *sql.Tx) error { _, e := tx.Exec("INSERT INTO settings VALUES('app.test','1')"); return e }); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Read(ctx, func(tx *sql.Tx) error {
		var value string
		e := tx.QueryRow("SELECT value FROM settings WHERE key='app.test'").Scan(&value)
		if value != "1" {
			t.Fatal(value)
		}
		return e
	}); err != nil {
		t.Fatal(err)
	}
}

// ReplaceDatabaseLocked atomically swaps the live SQLite database with a verified snapshot.
func TestReplaceDatabase(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	snapshot := filepath.Join(t.TempDir(), "snapshot.sqlite")
	if err = s.Exclusive(ctx, func(db *sql.DB) error {
		if _, e := db.Exec("VACUUM INTO ?", snapshot); e != nil {
			return e
		}
		_, e := db.Exec("INSERT INTO settings VALUES('app.new','true')")
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if err = s.Exclusive(ctx, func(_ *sql.DB) error { return s.ReplaceDatabaseLocked(ctx, snapshot) }); err != nil {
		t.Fatal(err)
	}
	if err = s.Read(ctx, func(tx *sql.Tx) error {
		var count int
		e := tx.QueryRow("SELECT count(*) FROM settings").Scan(&count)
		if count != 0 {
			t.Fatal("snapshot not restored")
		}
		return e
	}); err != nil {
		t.Fatal(err)
	}
}

// Opening a database with a newer schema version fails without rewriting its contents.
func TestFutureSchemaIsRejectedWithoutRewriting(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Exclusive(context.Background(), func(db *sql.DB) error { _, e := db.Exec("PRAGMA user_version=99"); return e }); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if newer, err := Open(root); err == nil {
		newer.Close()
		t.Fatal("future schema accepted")
	}
	db, err := openDB(filepath.Join(root, "aion.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 99 {
		t.Fatal(version, err)
	}
}

// Regression: Store methods return an error instead of panicking with a nil pointer dereference if s.db is nil after a failed reopen.
func TestStoreGuardsNilDatabaseHandle(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.db.Close(); err != nil {
		t.Fatal(err)
	}
	s.db = nil
	ctx := context.Background()
	if err := s.Read(ctx, func(_ *sql.Tx) error { return nil }); err == nil {
		t.Fatal("Read with nil db succeeded")
	}
	if err := s.Write(ctx, func(_ *sql.Tx) error { return nil }); err == nil {
		t.Fatal("Write with nil db succeeded")
	}
	if err := s.Exclusive(ctx, func(_ *sql.DB) error { return nil }); err == nil {
		t.Fatal("Exclusive with nil db succeeded")
	}
}

// Open rejects a symlinked lock file instead of following it outside the data directory.
func TestOpenRejectsSymlinkLockFile(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside-lock")
	if err := os.WriteFile(outside, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".aion.lock")); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(root); err == nil {
		s.Close()
		t.Fatal("Open followed symlink lock file")
	}
}

// Regression: ValidateBackupDir and ValidateExportTarget reject SQLite sidecar paths, trailing separators, and unsafe roots without creating directories under protected paths.
func TestValidatePathsRejectSidecarsAndTrailingSeparators(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	rejectedBackupDirs := []string{
		root,
		filepath.Join(root, "attachments"),
		filepath.Join(root, "attachments", "nested"),
		filepath.Join(root, ".hidden"),
		filepath.Join(root, "aion.sqlite"),
		filepath.Join(root, "aion.sqlite-journal"),
		filepath.Join(root, "aion.sqlite-wal"),
		filepath.Join(root, "aion.sqlite-shm"),
	}
	for _, dir := range rejectedBackupDirs {
		if _, err := s.ValidateBackupDir(dir); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("ValidateBackupDir(%q) = %v, want ErrInvalid", dir, err)
		}
	}
	if _, err := s.ValidateBackupDir(filepath.Join(root, "backups")); err != nil {
		t.Fatalf("ValidateBackupDir(backups) failed: %v", err)
	}

	outDir := t.TempDir()
	for _, dest := range []string{outDir + "/", outDir + "\\"} {
		if _, err := s.ValidateExportTarget(dest, false); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("ValidateExportTarget(%q) = %v, want ErrInvalid", dest, err)
		}
	}
	sidecarExport := filepath.Join(root, "aion.sqlite-journal", "out.zip")
	if _, err := s.ValidateExportTarget(sidecarExport, true); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("ValidateExportTarget(%q) = %v, want ErrInvalid", sidecarExport, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "aion.sqlite-journal")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ValidateExportTarget created sidecar directory: %v", err)
	}
}
