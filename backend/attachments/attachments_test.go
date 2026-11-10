package attachments_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aion/backend/attachments"
	"aion/backend/model"
	"aion/backend/store"
	"golang.org/x/sys/unix"
)

var ctx = context.Background()

func fixture(t *testing.T) (*store.Store, *attachments.Service) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	exec(t, s, `INSERT INTO projects(id,name,icon,description,created_at,updated_at) VALUES('p','project','','',?,?)`, model.Timestamp(), model.Timestamp())
	exec(t, s, `INSERT INTO tasks(id,project_id,title,description,status,priority,created_at,updated_at) VALUES('t','p','task','','planned','none',?,?)`, model.Timestamp(), model.Timestamp())
	return s, attachments.New(s)
}

func exec(t *testing.T, s *store.Store, query string, args ...any) {
	t.Helper()
	if err := s.Write(ctx, func(tx *sql.Tx) error { _, err := tx.ExecContext(ctx, query, args...); return err }); err != nil {
		t.Fatal(err)
	}
}

func source(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func entries(t *testing.T, s *store.Store) int {
	t.Helper()
	files, err := os.ReadDir(s.AttachmentsDir())
	if err != nil {
		t.Fatal(err)
	}
	return len(files)
}

// Importing attachments copies files into managed storage with unique storage names and survives source deletion.
func TestImportSurvivesOriginalDeletionAndSameNames(t *testing.T) {
	_, service := fixture(t)
	original := source(t, "相同名称.txt", "first contents")
	a, err := service.Import(ctx, "t", original)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	b, err := service.Import(ctx, "t", source(t, "相同名称.txt", "second contents"))
	if err != nil {
		t.Fatal(err)
	}
	if a.StorageName == b.StorageName || a.ID == b.ID || a.Name != b.Name {
		t.Fatalf("bad names: %+v %+v", a, b)
	}
	hash := sha256.Sum256([]byte("first contents"))
	if a.SHA256 != hex.EncodeToString(hash[:]) || a.Size != 14 || a.MediaType != "text/plain; charset=utf-8" {
		t.Fatalf("bad metadata: %+v", a)
	}
	for _, tc := range []struct {
		attachment model.Attachment
		want       string
	}{{a, "first contents"}, {b, "second contents"}} {
		path, err := service.ManagedPath(ctx, tc.attachment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got := read(t, path); got != tc.want {
			t.Fatalf("contents = %q", got)
		}
		export := filepath.Join(t.TempDir(), "saved.txt")
		if err := service.Export(ctx, tc.attachment.ID, export); err != nil {
			t.Fatal(err)
		}
		if got := read(t, export); got != tc.want {
			t.Fatalf("export = %q", got)
		}
		if err := service.Export(ctx, tc.attachment.ID, export); !errors.Is(err, model.ErrConflict) {
			t.Fatalf("overwrite error = %v", err)
		}
		if got := read(t, export); got != tc.want {
			t.Fatalf("overwrite changed file: %q", got)
		}
	}
	list, err := service.List(ctx, "t")
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %+v, %v", list, err)
	}
}

// Rejected or aborted attachment imports leave no orphan files or database rows behind.
func TestFailedImportsLeaveNoFilesOrRows(t *testing.T) {
	s, service := fixture(t)
	dir := t.TempDir()
	regular := source(t, "normal.txt", "data")
	oversize := filepath.Join(dir, "large.bin")
	if f, err := os.Create(oversize); err != nil {
		t.Fatal(err)
	} else {
		err = f.Truncate(attachments.MaxFileBytes + 1)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "missing"), oversize, link, fifo, dir, source(t, "unsafe\\name", "data")} {
		if _, err := service.Import(ctx, "t", path); err == nil {
			t.Errorf("accepted unsafe/missing source %q", path)
		}
	}
	if _, err := service.Import(ctx, "missing-task", regular); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("missing task error = %v", err)
	}
	exec(t, s, `CREATE TRIGGER reject_attachment BEFORE INSERT ON attachments BEGIN SELECT RAISE(ABORT,'injected insert failure'); END`)
	if _, err := service.Import(ctx, "t", regular); err == nil {
		t.Fatal("insert failure succeeded")
	}
	if n := entries(t, s); n != 0 {
		t.Fatalf("failed imports left %d files", n)
	}
	list, err := service.List(ctx, "t")
	if err != nil || len(list) != 0 {
		t.Fatalf("failed imports left rows: %+v %v", list, err)
	}
}

// Soft-deleted parent tasks and projects block attachment operations while retaining managed files until permanent deletion.
func TestDeletedParentsHideOperationsButRetainFiles(t *testing.T) {
	for _, parent := range []string{"tasks", "projects"} {
		t.Run(parent, func(t *testing.T) {
			s, service := fixture(t)
			original := source(t, "data.txt", "retained")
			a, err := service.Import(ctx, "t", original)
			if err != nil {
				t.Fatal(err)
			}
			exec(t, s, `UPDATE `+parent+` SET deleted_at=?`, model.Timestamp())
			want := model.ErrNotFound
			if parent == "projects" {
				want = model.ErrParentDeleted
			}
			if _, err := service.Import(ctx, "t", original); !errors.Is(err, want) {
				t.Fatalf("Import error = %v", err)
			}
			if _, err := service.List(ctx, "t"); !errors.Is(err, want) {
				t.Fatalf("List error = %v", err)
			}
			if _, err := service.ManagedPath(ctx, a.ID); !errors.Is(err, want) {
				t.Fatalf("ManagedPath error = %v", err)
			}
			if err := service.Export(ctx, a.ID, filepath.Join(t.TempDir(), "export")); !errors.Is(err, want) {
				t.Fatalf("Export error = %v", err)
			}
			if err := service.Remove(ctx, a.ID); !errors.Is(err, want) {
				t.Fatalf("Remove error = %v", err)
			}
			if n, err := service.GC(ctx); err != nil || n != 0 {
				t.Fatalf("GC removed soft deleted attachment: %d %v", n, err)
			}
			if got := read(t, filepath.Join(s.AttachmentsDir(), a.StorageName)); got != "retained" {
				t.Fatalf("retained file = %q", got)
			}
			exec(t, s, `UPDATE `+parent+` SET deleted_at=NULL`)
			if _, err := service.ManagedPath(ctx, a.ID); err != nil {
				t.Fatalf("restored path: %v", err)
			}
			exec(t, s, `DELETE FROM `+parent)
			if n, err := service.GC(ctx); err != nil || n != 1 {
				t.Fatalf("purged GC = %d %v", n, err)
			}
			if entries(t, s) != 0 {
				t.Fatal("purged file survived GC")
			}
		})
	}
}

// Removing an attachment and running GC deletes only unreferenced regular files without following symlinks.
func TestRemoveAndGCPreserveOtherReferencesAndNeverFollowLinks(t *testing.T) {
	s, service := fixture(t)
	a, err := service.Import(ctx, "t", source(t, "a", "a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := service.Import(ctx, "t", source(t, "b", "b"))
	if err != nil {
		t.Fatal(err)
	}
	outside := source(t, "outside", "outside")
	if err := os.Symlink(outside, filepath.Join(s.AttachmentsDir(), "symlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(s.AttachmentsDir(), "directory"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := service.Remove(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ManagedPath(ctx, a.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("removed path error = %v", err)
	}
	if n, err := service.GC(ctx); err != nil || n != 1 {
		t.Fatalf("GC = %d %v", n, err)
	}
	if read(t, outside) != "outside" {
		t.Fatal("GC changed external file")
	}
	if _, err := os.Lstat(filepath.Join(s.AttachmentsDir(), "symlink")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ManagedPath(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
}

// ManagedPath and Export reject stored attachment names that traverse directories or point to symlinks.
func TestManagedPathsRejectTraversalAndSymlinks(t *testing.T) {
	for _, badName := range []string{"../outside", "/absolute", "..\\outside", "link"} {
		t.Run(badName, func(t *testing.T) {
			s, service := fixture(t)
			a, err := service.Import(ctx, "t", source(t, "a", "a"))
			if err != nil {
				t.Fatal(err)
			}
			outside := source(t, "outside", "outside")
			if err := os.Symlink(outside, filepath.Join(s.AttachmentsDir(), "link")); err != nil {
				t.Fatal(err)
			}
			exec(t, s, `UPDATE attachments SET storage_name=? WHERE id=?`, badName, a.ID)
			if _, err := service.ManagedPath(ctx, a.ID); err == nil {
				t.Fatal("resolved unsafe stored name")
			}
			if err := service.Export(ctx, a.ID, filepath.Join(t.TempDir(), "saved")); err == nil {
				t.Fatal("exported unsafe stored name")
			}
		})
	}
}

// Failed attachment exports preserve existing target files and clean up partial staging copies.
func TestExportFailuresPreserveExistingFilesAndCleanPartialCopies(t *testing.T) {
	s, service := fixture(t)
	a, err := service.Import(ctx, "t", source(t, "data", "contents"))
	if err != nil {
		t.Fatal(err)
	}
	outside := source(t, "outside", "protected")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := service.Export(ctx, a.ID, link); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("symlink export = %v", err)
	}
	if read(t, outside) != "protected" {
		t.Fatal("export overwrote symlink target")
	}
	if err := service.Export(ctx, a.ID, filepath.Join(s.AttachmentsDir(), "unexpected")); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("export to store = %v", err)
	}
	managed, err := service.ManagedPath(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managed, []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "partial")
	if err := service.Export(ctx, a.ID, destination); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("corrupt export = %v", err)
	}
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial export survived: %v", err)
	}
	if err := os.Remove(managed); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ManagedPath(ctx, a.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing managed file = %v", err)
	}
}
