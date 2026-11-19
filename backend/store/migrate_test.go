package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIncrementalMigrationFromV1PreservesDataAndMatchesFreshSchema(t *testing.T) {
	// Upgrading a populated v1 database must preserve v1 data and yield the same schema as a fresh store.
	ctx := context.Background()
	v1Dir := t.TempDir()
	dbPath := filepath.Join(v1Dir, "aion.sqlite")
	db, err := openDB(dbPath, "rwc")
	if err != nil {
		t.Fatalf("open v1 db: %v", err)
	}
	if err := migrateDB(ctx, db, 1); err != nil {
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
		t.Fatalf("upgrade v1 store: %v", err)
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
		t.Fatalf("open fresh store: %v", err)
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
	if err := migrateDB(ctx, db, 1); err != nil {
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
	if version != 1 {
		t.Fatalf("version after rollback = %d, want 1", version)
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

// Opening a v2 store drops the obsolete column while preserving the job and its timeline.
func TestMigrationFromV2DropsFollowUpDate(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db, err := openDB(filepath.Join(root, "aion.sqlite"), "rwc")
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateDB(ctx, db, 2); err != nil {
		db.Close()
		t.Fatal(err)
	}
	ts := "2026-11-01T10:00:00Z"
	_, err = db.Exec(`INSERT INTO job_applications(id,company_name,position_name,applied_date,stage,lifecycle,next_follow_up_date,notes,created_at,updated_at)
		VALUES('job','Acme','Engineer','2026-11-01','applied','active','2026-11-10','keep me',?,?)`, ts, ts)
	if err == nil {
		_, err = db.Exec(`INSERT INTO job_application_events(id,application_id,event_type,note,occurred_at,created_at)
			VALUES('event','job','note_added','timeline note',?,?)`, ts, ts)
	}
	if closeErr := db.Close(); err != nil || closeErr != nil {
		t.Fatalf("seed legacy store: %v, close: %v", err, closeErr)
	}

	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Read(ctx, func(tx *sql.Tx) error {
		var remaining int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM pragma_table_info('job_applications') WHERE name = 'next_follow_up_date'").Scan(&remaining); err != nil {
			return err
		}
		var notes, eventNote string
		if err := tx.QueryRowContext(ctx, `SELECT j.notes, e.note FROM job_applications j JOIN job_application_events e ON e.application_id=j.id WHERE j.id='job' AND e.id='event'`).Scan(&notes, &eventNote); err != nil {
			return err
		}
		if remaining != 0 || notes != "keep me" || eventNote != "timeline note" {
			t.Fatalf("migration left %d obsolete columns, job notes %q and event note %q", remaining, notes, eventNote)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
