package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIncrementalMigrationFromV1PreservesDataAndMatchesFreshSchema(t *testing.T) {
	// Upgrading a populated v1 database must apply only v2 DDL, preserve v1 data, and yield identical schema to a fresh v2 store.
	ctx := context.Background()
	v1Dir := t.TempDir()
	dbPath := filepath.Join(v1Dir, "aion.sqlite")
	db, err := openDB(dbPath, "rwc")
	if err != nil {
		t.Fatalf("open v1 db: %v", err)
	}
	if err := ApplySchemaUpTo(ctx, db, MinSchemaVersion); err != nil {
		t.Fatalf("apply v1 schema: %v", err)
	}
	ts := "2026-11-01T10:00:00Z"
	if _, err := db.Exec(`INSERT INTO projects(id,name,icon,description,archived,created_at,updated_at) VALUES('p1','Existing Project','folder','keep me',0,?,?)`, ts, ts); err != nil {
		t.Fatalf("seed v1 project: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO settings(key,value) VALUES('theme','dark')`); err != nil {
		t.Fatalf("seed v1 setting: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close v1 db: %v", err)
	}

	s, err := Open(v1Dir)
	if err != nil {
		t.Fatalf("reopen v1 store as v2: %v", err)
	}
	defer s.Close()

	var version int
	var projName, settingVal string
	err = s.Read(ctx, func(ro *sql.Tx) error {
		if err := ro.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			return err
		}
		if err := ro.QueryRowContext(ctx, "SELECT name FROM projects WHERE id='p1'").Scan(&projName); err != nil {
			return err
		}
		return ro.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='theme'").Scan(&settingVal)
	})
	if err != nil {
		t.Fatalf("verify upgraded data: %v", err)
	}
	if version != SchemaVersion {
		t.Fatalf("user_version = %d, want %d", version, SchemaVersion)
	}
	if projName != "Existing Project" || settingVal != "dark" {
		t.Fatalf("v1 data corrupted: proj=%q setting=%q", projName, settingVal)
	}

	freshDir := t.TempDir()
	fresh, err := Open(freshDir)
	if err != nil {
		t.Fatalf("open fresh v2 store: %v", err)
	}
	defer fresh.Close()

	readSchema := func(target *Store) []string {
		var rows []string
		err := target.Read(ctx, func(ro *sql.Tx) error {
			res, err := ro.QueryContext(ctx, "SELECT type||':'||name||':'||COALESCE(sql,'') FROM sqlite_schema WHERE name NOT GLOB 'sqlite_*' ORDER BY type,name")
			if err != nil {
				return err
			}
			defer res.Close()
			for res.Next() {
				var line string
				if err := res.Scan(&line); err != nil {
					return err
				}
				rows = append(rows, line)
			}
			return res.Err()
		})
		if err != nil {
			t.Fatalf("read schema: %v", err)
		}
		return rows
	}

	if got, want := readSchema(s), readSchema(fresh); !reflect.DeepEqual(got, want) {
		t.Fatalf("upgraded schema differs from fresh schema:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestFailedMigrationRollsBackWithoutCorruptingV1Data(t *testing.T) {
	// A failing v1-to-v2 migration must roll back atomically, keeping user_version at 1 and preserving existing rows.
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "aion.sqlite")
	db, err := openDB(dbPath, "rwc")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := ApplySchemaUpTo(ctx, db, MinSchemaVersion); err != nil {
		t.Fatalf("apply v1 schema: %v", err)
	}
	ts := "2026-11-01T10:00:00Z"
	if _, err := db.Exec(`INSERT INTO projects(id,name,icon,description,archived,created_at,updated_at) VALUES('p1','Intact','folder','',0,?,?)`, ts, ts); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	// Inject a conflicting table so schemaV2 fails midway on CREATE TABLE job_applications.
	if _, err := db.Exec(`CREATE TABLE job_applications(bogus TEXT)`); err != nil {
		t.Fatalf("inject conflict: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	if _, err := Open(root); err == nil {
		t.Fatal("expected migration failure on conflicting table")
	}

	recheck, err := openDB(dbPath, "rw")
	if err != nil {
		t.Fatalf("reopen raw db: %v", err)
	}
	defer recheck.Close()
	var version int
	if err := recheck.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("scan version: %v", err)
	}
	if version != MinSchemaVersion {
		t.Fatalf("version after rollback = %d, want %d", version, MinSchemaVersion)
	}
	var name string
	if err := recheck.QueryRow("SELECT name FROM projects WHERE id='p1'").Scan(&name); err != nil || name != "Intact" {
		t.Fatalf("existing row lost after rollback: name=%q err=%v", name, err)
	}
	var eventsTableCount int
	if err := recheck.QueryRow("SELECT COUNT(*) FROM sqlite_schema WHERE name='job_application_events'").Scan(&eventsTableCount); err != nil || eventsTableCount != 0 {
		t.Fatalf("partial migration left job_application_events behind: count=%d err=%v", eventsTableCount, err)
	}
}

func TestMigrateDatabaseFileRejectsUnversionedOrFutureDB(t *testing.T) {
	// MigrateDatabaseFile must reject empty unversioned files and future schema versions before modifying anything.
	ctx := context.Background()
	unversioned := filepath.Join(t.TempDir(), "empty.sqlite")
	db, err := openDB(unversioned, "rwc")
	if err != nil {
		t.Fatalf("open empty db: %v", err)
	}
	db.Close()
	if err := MigrateDatabaseFile(ctx, unversioned); err == nil {
		t.Fatal("expected error migrating user_version=0 file")
	}

	future := filepath.Join(t.TempDir(), "future.sqlite")
	db2, err := openDB(future, "rwc")
	if err != nil {
		t.Fatalf("open future db: %v", err)
	}
	if _, err := db2.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatalf("set future version: %v", err)
	}
	db2.Close()
	if err := MigrateDatabaseFile(ctx, future); err == nil {
		t.Fatal("expected error migrating future schema version 99")
	}
}
