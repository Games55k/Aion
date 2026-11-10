package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aion/backend/store"
)

var testCtx = context.Background()

func testStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func seed(t *testing.T, s *store.Store, id, contents string) {
	t.Helper()
	now := "2026-11-09T08:00:00Z"
	h := sha256.Sum256([]byte(contents))
	err := s.Exclusive(testCtx, func(db *sql.DB) error {
		if err := os.WriteFile(filepath.Join(s.AttachmentsDir(), id+"-file"), []byte(contents), 0600); err != nil {
			return err
		}
		_, err := db.Exec(`INSERT INTO projects(id,name,icon,description,archived,created_at,updated_at,deleted_at) VALUES(?,?,'p','description',1,?,?,?)`, id, id, now, now, now)
		if err != nil {
			return err
		}
		_, err = db.Exec(`INSERT INTO tasks(id,project_id,title,description,status,priority,tags,subtasks,due_date,position,created_at,updated_at,completed_at,deleted_at) VALUES(?,?,'task','markdown','done','high','["one","two"]','[{"id":"sub","title":"check","done":true}]','2026-12-01',3,?,?,?,?)`, id+"-task", id, now, now, now, now)
		if err != nil {
			return err
		}
		_, err = db.Exec(`INSERT INTO attachments(id,task_id,name,storage_name,media_type,size,sha256,created_at) VALUES(?,?,'image.png',?,'image/png',?,?,?)`, id+"-a", id+"-task", id+"-file", len(contents), hex.EncodeToString(h[:]), now)
		if err != nil {
			return err
		}
		_, err = db.Exec("INSERT INTO settings(key,value) VALUES('theme',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertSeed(t *testing.T, s *store.Store, id, contents string) {
	t.Helper()
	err := s.Read(testCtx, func(tx *sql.Tx) error {
		var name, deleted, theme, tags, subtasks, due, completed string
		var archived int
		if err := tx.QueryRow("SELECT name,archived,deleted_at FROM projects WHERE id=?", id).Scan(&name, &archived, &deleted); err != nil {
			return err
		}
		if name != id || archived != 1 || deleted == "" {
			t.Errorf("project state lost: %s %d %s", name, archived, deleted)
		}
		if err := tx.QueryRow("SELECT tags,subtasks,due_date,completed_at,deleted_at FROM tasks WHERE id=?", id+"-task").Scan(&tags, &subtasks, &due, &completed, &deleted); err != nil {
			return err
		}
		if tags != "[\"one\",\"two\"]" || !strings.Contains(subtasks, "check") || due != "2026-12-01" || completed == "" || deleted == "" {
			t.Error("task/trash data lost")
		}
		if err := tx.QueryRow("SELECT value FROM settings WHERE key='theme'").Scan(&theme); err != nil {
			return err
		}
		if theme != id {
			t.Errorf("theme=%s", theme)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.AttachmentsDir(), id+"-file"))
	if err != nil || string(b) != contents {
		t.Fatalf("attachment: %s %v", b, err)
	}
}

func exportFixture(t *testing.T) (string, *store.Store) {
	t.Helper()
	s := testStore(t)
	seed(t, s, "source", "source attachment")
	path := filepath.Join(t.TempDir(), "manual.zip")
	if err := New(s).Export(testCtx, path); err != nil {
		t.Fatal(err)
	}
	return path, s
}

// Backups exclude orphan/history files and can restore both a snapshot and its pre-restore safety copy.
func TestRoundTripAndSafety(t *testing.T) {
	_, src := exportFixture(t)
	// Only references are included; history backups and orphaned files are absent.
	if err := os.WriteFile(filepath.Join(src.AttachmentsDir(), "orphan"), []byte("unreferenced"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src.Root(), "backups"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src.Root(), "backups", "history.zip"), []byte("history"), 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "second.zip")
	if err := New(src).Export(testCtx, source); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if len(z.File) != 3 {
		t.Fatalf("got %d archive entries", len(z.File))
	}
	target := testStore(t)
	seed(t, target, "live", "current attachment")
	service := New(target)
	result, err := service.Restore(testCtx, source)
	if err != nil {
		t.Fatal(err)
	}
	if result.SafetyBackupPath == "" {
		t.Fatal("no safety backup")
	}
	assertSeed(t, target, "source", "source attachment")
	if _, err = os.Stat(filepath.Join(target.AttachmentsDir(), "live-file")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old attachment remains: %v", err)
	}
	if _, err = service.Restore(testCtx, result.SafetyBackupPath); err != nil {
		t.Fatal(err)
	}
	assertSeed(t, target, "live", "current attachment")
}

type zipItem struct {
	name string
	data []byte
	mode os.FileMode
}

func zipItems(t *testing.T, path string) []zipItem {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var out []zipItem
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, zipItem{f.Name, b, f.Mode()})
	}
	return out
}
func writeItems(t *testing.T, items []zipItem) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "modified.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, item := range items {
		h := &zip.FileHeader{Name: item.name, Method: zip.Deflate}
		h.SetMode(item.mode)
		w, err := z.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
func editManifest(t *testing.T, items []zipItem, fn func(*manifest)) {
	t.Helper()
	for i := range items {
		if items[i].name == "manifest.json" {
			var m manifest
			if err := json.Unmarshal(items[i].data, &m); err != nil {
				t.Fatal(err)
			}
			fn(&m)
			items[i].data, _ = json.Marshal(m)
			return
		}
	}
	t.Fatal("manifest missing")
}
func updateDB(t *testing.T, items []zipItem, statement string) {
	t.Helper()
	for i := range items {
		if items[i].name != "database.sqlite" {
			continue
		}
		path := filepath.Join(t.TempDir(), "db.sqlite")
		if err := os.WriteFile(path, items[i].data, 0600); err != nil {
			t.Fatal(err)
		}
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(statement); err != nil {
			db.Close()
			t.Fatal(err)
		}
		db.Close()
		items[i].data, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		b := items[i].data
		h := sha256.Sum256(b)
		editManifest(t, items, func(m *manifest) {
			for n := range m.Files {
				if m.Files[n].Path == "database.sqlite" {
					m.Files[n].Size = int64(len(b))
					m.Files[n].SHA256 = hex.EncodeToString(h[:])
				}
			}
		})
		return
	}
	t.Fatal("database missing")
}

// Restore validates archive entries, manifest hashes, and SQLite schema before touching live files or creating a safety backup.
func TestRejectBeforeLiveChanges(t *testing.T) {
	source, _ := exportFixture(t)
	cases := []struct {
		name string
		edit func([]zipItem) []zipItem
	}{
		{"traversal", func(x []zipItem) []zipItem { return append(x, zipItem{"../escape", []byte("x"), 0600}) }},
		{"absolute", func(x []zipItem) []zipItem { return append(x, zipItem{"/absolute", []byte("x"), 0600}) }},
		{"backslash", func(x []zipItem) []zipItem { return append(x, zipItem{"attachments/..\\escape", []byte("x"), 0600}) }},
		{"nested", func(x []zipItem) []zipItem { return append(x, zipItem{"attachments/../escape", []byte("x"), 0600}) }},
		{"duplicate-entry", func(x []zipItem) []zipItem { return append(x, x[0]) }},
		{"symlink", func(x []zipItem) []zipItem { x[0].mode = os.ModeSymlink | 0777; return x }},
		{"directory", func(x []zipItem) []zipItem { return append(x, zipItem{"attachments/", nil, os.ModeDir | 0700}) }},
		{"hash", func(x []zipItem) []zipItem { x[1].data = []byte("corrupt"); return x }},
		{"unsupported-format", func(x []zipItem) []zipItem { editManifest(t, x, func(m *manifest) { m.Version++ }); return x }},
		{"unsupported-schema", func(x []zipItem) []zipItem { editManifest(t, x, func(m *manifest) { m.SchemaVersion++ }); return x }},
		{"unsupported-db-version", func(x []zipItem) []zipItem { updateDB(t, x, "PRAGMA user_version=999"); return x }},
		{"bad-json", func(x []zipItem) []zipItem { updateDB(t, x, "UPDATE tasks SET tags='bad'"); return x }},
		{"bad-subtasks", func(x []zipItem) []zipItem { updateDB(t, x, "UPDATE tasks SET subtasks='[42]'"); return x }},
		{"bad-time", func(x []zipItem) []zipItem { updateDB(t, x, "UPDATE projects SET created_at='never'"); return x }},
		{"bad-date", func(x []zipItem) []zipItem { updateDB(t, x, "UPDATE tasks SET due_date='2026-02-31'"); return x }},
		{"foreign-key", func(x []zipItem) []zipItem { updateDB(t, x, "PRAGMA foreign_keys=OFF;DELETE FROM projects"); return x }},
		{"unexpected-schema", func(x []zipItem) []zipItem {
			updateDB(t, x, "CREATE TRIGGER unexpected AFTER INSERT ON projects BEGIN DELETE FROM tasks; END")
			return x
		}},
		{"missing-reference", func(x []zipItem) []zipItem {
			editManifest(t, x, func(m *manifest) { m.Files = m.Files[:1] })
			return append(x[:1], x[2:]...)
		}},
		{"db-hash-disagrees", func(x []zipItem) []zipItem {
			updateDB(t, x, "UPDATE attachments SET sha256='"+strings.Repeat("0", 64)+"'")
			return x
		}},
		{"unsafe-db-reference", func(x []zipItem) []zipItem {
			updateDB(t, x, "UPDATE attachments SET storage_name='../outside'")
			return x
		}},
		{"invalid-db", func(x []zipItem) []zipItem {
			x[0].data = []byte("not sqlite")
			h := sha256.Sum256(x[0].data)
			editManifest(t, x, func(m *manifest) {
				m.Files[0].Size = int64(len(x[0].data))
				m.Files[0].SHA256 = hex.EncodeToString(h[:])
			})
			return x
		}},
		{"resource-limit", func(x []zipItem) []zipItem {
			editManifest(t, x, func(m *manifest) { m.Files[0].Size = MaxFileBytes + 1 })
			return x
		}},
		{"duplicate-record", func(x []zipItem) []zipItem {
			editManifest(t, x, func(m *manifest) { m.Files[1] = m.Files[0] })
			return x
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := zipItems(t, source)
			modified := writeItems(t, tc.edit(items))
			s := testStore(t)
			seed(t, s, "live", "must survive")
			result, err := New(s).Restore(testCtx, modified)
			if err == nil {
				t.Fatal("accepted invalid archive")
			}
			if result.SafetyBackupPath != "" {
				t.Fatal("invalid archive reached live mutation phase")
			}
			assertSeed(t, s, "live", "must survive")
			if _, err := os.Stat(filepath.Join(s.Root(), "backups")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("created safety directory before validation: %v", err)
			}
		})
	}
}

// If database replacement fails during restore, live data and attachments are rolled back while preserving the pre-restore safety backup.
func TestReplacementFailureRollsBack(t *testing.T) {
	source, _ := exportFixture(t)
	s := testStore(t)
	seed(t, s, "live", "survives failed replacement")
	service := New(s)
	injected := errors.New("injected database replacement failure")
	service.replaceDatabase = func(context.Context, string) error { return injected }
	result, err := service.Restore(testCtx, source)
	if !errors.Is(err, injected) || result.SafetyBackupPath == "" {
		t.Fatalf("%+v %v", result, err)
	}
	assertSeed(t, s, "live", "survives failed replacement")
	if _, err = os.Stat(filepath.Join(s.AttachmentsDir(), "source-file")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("new attachment survived rollback")
	}
	if _, err = New(s).Restore(testCtx, result.SafetyBackupPath); err != nil {
		t.Fatal(err)
	}
	assertSeed(t, s, "live", "survives failed replacement")
}

// If creating the pre-restore safety backup fails, Restore aborts without altering live data.
func TestSafetyFailureLeavesLiveUntouched(t *testing.T) {
	source, _ := exportFixture(t)
	s := testStore(t)
	seed(t, s, "live", "untouched")
	if err := os.WriteFile(filepath.Join(s.Root(), "backups"), []byte("directory blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := New(s).Restore(testCtx, source)
	if err == nil || result.SafetyBackupPath != "" {
		t.Fatalf("%+v %v", result, err)
	}
	assertSeed(t, s, "live", "untouched")
}

// Export refuses to overwrite existing archives, export corrupt or symlinked attachments, or write inside the managed attachments directory.
func TestExportRefusesInvalidLiveFilesAndOverwrite(t *testing.T) {
	s := testStore(t)
	seed(t, s, "live", "hello")
	service := New(s)
	destination := filepath.Join(t.TempDir(), "backup.zip")
	if err := service.Export(testCtx, destination); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(destination)
	if err := service.Export(testCtx, destination); err == nil {
		t.Fatal("overwrote existing archive")
	}
	after, _ := os.ReadFile(destination)
	if string(before) != string(after) {
		t.Fatal("destination changed")
	}
	path := filepath.Join(s.AttachmentsDir(), "live-file")
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "bad.zip")
	if err := service.Export(testCtx, bad); err == nil {
		t.Fatal("exported hash mismatch")
	}
	if _, err := os.Stat(bad); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial destination published")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(destination, path); err != nil {
		t.Fatal(err)
	}
	if err := service.Export(testCtx, bad); err == nil {
		t.Fatal("exported symlink")
	}
	if err := service.Export(testCtx, filepath.Join(s.AttachmentsDir(), "export.zip")); err == nil {
		t.Fatal("exported into live attachments")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := service.Export(testCtx, bad); err == nil {
		t.Fatal("exported missing reference")
	}
}

// Automatic backups rotate only owned archives beyond Keep, preserve manual backups, and keep schedule state across restores and reopens.
func TestAutoPolicyRotationPersistenceAndRestore(t *testing.T) {
	s := testStore(t)
	seed(t, s, "live", "auto payload")
	service := New(s)
	p, err := service.GetPolicy(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	p.Keep = 2
	p.Directory = t.TempDir()
	if err = service.SetPolicy(testCtx, p); err != nil {
		t.Fatal(err)
	}
	manual := filepath.Join(p.Directory, "auto-manual.zip")
	if err = service.Export(testCtx, manual); err != nil {
		t.Fatal(err)
	}
	var paths []string
	for i := 0; i < 4; i++ {
		path, err := service.RunDue(testCtx)
		if err != nil || path == "" {
			t.Fatalf("auto creation: %s %v", path, err)
		}
		paths = append(paths, path)
		if i != 3 {
			st, err := service.loadState()
			if err != nil {
				t.Fatal(err)
			}
			st.LastSuccess = time.Now().Add(-48 * time.Hour)
			if err = service.saveState(st); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, path := range paths[:2] {
		if _, err = os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("old owned auto archive retained")
		}
	}
	for _, path := range append(paths[2:], manual) {
		if _, err = os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	status, err := New(s).Status(testCtx)
	if err != nil || status.LastSuccess.IsZero() || status.LastError != "" {
		t.Fatalf("persisted status: %+v %v", status, err)
	}
	// Restore a manual archive predating all successful runs. State remains new.
	if _, err = service.Restore(testCtx, manual); err != nil {
		t.Fatal(err)
	}
	after, err := service.Status(testCtx)
	if err != nil || !after.LastSuccess.Equal(status.LastSuccess) {
		t.Fatalf("restore rewound auto state: %+v %v", after, err)
	}
	if path, err := New(s).RunDue(testCtx); err != nil || path != "" {
		t.Fatalf("unexpected duplicate automatic run: %s %v", path, err)
	}
	// Store reopen retains both policy and execution state.
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err = New(reopened).Status(testCtx)
	if err != nil || !after.LastSuccess.Equal(status.LastSuccess) {
		t.Fatalf("reopen state: %+v %v", after, err)
	}
}

// Failed automatic backup runs preserve prior archives, record LastError, and respect disabling the policy.
func TestAutoFailurePreservesFilesAndRecordsError(t *testing.T) {
	s := testStore(t)
	seed(t, s, "live", "payload")
	service := New(s)
	p, _ := service.GetPolicy(testCtx)
	p.Keep = 1
	if err := service.SetPolicy(testCtx, p); err != nil {
		t.Fatal(err)
	}
	first, err := service.RunDue(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := service.loadState()
	st.LastSuccess = time.Now().Add(-48 * time.Hour)
	if err = service.saveState(st); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.AttachmentsDir(), "live-file"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RunDue(testCtx); err == nil {
		t.Fatal("automatic corruption accepted")
	}
	if _, err = os.Stat(first); err != nil {
		t.Fatal("rotation ran after failed export")
	}
	status, err := New(s).Status(testCtx)
	if err != nil || status.LastError == "" || !status.LastSuccess.Equal(st.LastSuccess) {
		t.Fatalf("failure status: %+v %v", status, err)
	}
	// Disabled policy prevents attempts even with a stale last success.
	p.Enabled = false
	if err = service.SetPolicy(testCtx, p); err != nil {
		t.Fatal(err)
	}
	if path, err := service.RunDue(testCtx); err != nil || path != "" {
		t.Fatalf("disabled backup ran: %s %v", path, err)
	}
}

// Automatic backup rotation verifies file hashes so manually replaced files in the backup directory are never deleted.
func TestRotationPreservesReplacedFile(t *testing.T) {
	s := testStore(t)
	service := New(s)
	p, _ := service.GetPolicy(testCtx)
	p.Keep = 1
	if err := service.SetPolicy(testCtx, p); err != nil {
		t.Fatal(err)
	}
	first, err := service.RunDue(testCtx)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(first, []byte("manual replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	st, _ := service.loadState()
	st.LastSuccess = time.Now().Add(-48 * time.Hour)
	if err = service.saveState(st); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RunDue(testCtx); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(first)
	if err != nil || string(b) != "manual replacement" {
		t.Fatal("deleted replaced/manual file")
	}
}

// Concurrent RunDue calls serialize through the Store lock so only a single due archive is created.
func TestConcurrentRunDueCreatesOnce(t *testing.T) {
	s := testStore(t)
	var wg sync.WaitGroup
	paths := make(chan string, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); path, err := New(s).RunDue(testCtx); paths <- path; errs <- err }()
	}
	wg.Wait()
	close(paths)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	for path := range paths {
		if path != "" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("created %d archives", count)
	}
}
