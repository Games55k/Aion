package attachments_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aion/backend/attachments"
	"aion/backend/backup"
	"aion/backend/model"
	"aion/backend/projects"
	"aion/backend/store"
)

func importStages(root string) []string {
	paths, _ := filepath.Glob(filepath.Join(root, ".attachment-import-*"))
	return paths
}

func pauseImport(t *testing.T, s *store.Store, service *attachments.Service, ctx context.Context, contents string) (string, func() (model.Attachment, error)) {
	t.Helper()
	src := source(t, "pending.txt", contents)
	copyCtx := &pauseCopyContext{
		Context: ctx,
		started: make(chan struct{}),
		release: make(chan struct{}),
		ready: func() bool {
			for _, path := range importStages(s.Root()) {
				if info, err := os.Stat(filepath.Join(path, "payload")); err == nil && info.Size() > 0 {
					return true
				}
			}
			return false
		},
	}
	var imported model.Attachment
	var importErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		imported, importErr = service.Import(copyCtx, "t", src)
	}()
	resume := sync.OnceFunc(func() { close(copyCtx.release) })
	t.Cleanup(func() {
		resume()
		<-done
	})
	select {
	case <-copyCtx.started:
	case <-done:
		t.Fatalf("import ended before its staged copy could pause: %v", importErr)
	case <-time.After(5 * time.Second):
		t.Fatal("import did not reach the staged copy")
	}
	stages := importStages(s.Root())
	if len(stages) != 1 {
		t.Fatalf("expected one active import staging directory, got %v", stages)
	}
	return filepath.Join(stages[0], "payload"), func() (model.Attachment, error) {
		resume()
		<-done
		return imported, importErr
	}
}

func whileCopyPaused(t *testing.T, work func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- work() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("unrelated Store operation blocked during attachment copy")
	}
}

func expectEmptyImportStorage(t *testing.T, s *store.Store) {
	t.Helper()
	if stages := importStages(s.Root()); len(stages) != 0 {
		t.Fatalf("import leaked staging directories: %v", stages)
	}
	if n := entries(t, s); n != 0 {
		t.Fatalf("import left %d managed files", n)
	}
	var count int
	err := s.Read(ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM attachments").Scan(&count)
	})
	if err != nil || count != 0 {
		t.Fatalf("import left %d attachment rows, error %v", count, err)
	}
}

// Regression: GC and unrelated writes used to wait for an attachment copy to finish.
func TestImportCopyAllowsConcurrentGarbageCollection(t *testing.T) {
	s, service := fixture(t)
	old, err := service.Import(ctx, "t", source(t, "old.txt", "orphan after removal"))
	if err != nil {
		t.Fatal(err)
	}
	contents := strings.Repeat("pending attachment\n", 4096)
	staged, finish := pauseImport(t, s, service, ctx, contents)

	whileCopyPaused(t, func() error {
		if _, err := s.ValidateExportTarget(staged, true); !errors.Is(err, model.ErrInvalid) {
			return fmt.Errorf("a staged import must be protected from export: %v", err)
		}
		if err := service.Remove(ctx, old.ID); err != nil {
			return err
		}
		if n, err := service.GC(ctx); err != nil || n != 1 {
			return fmt.Errorf("GC removed %d files, error %v", n, err)
		}
		if _, err := os.Stat(staged); err != nil {
			return fmt.Errorf("staged copy lost during GC: %w", err)
		}
		if listed, err := service.List(ctx, "t"); err != nil || len(listed) != 0 {
			return fmt.Errorf("import became visible before publication: %+v, %v", listed, err)
		}
		return nil
	})

	imported, err := finish()
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(s.AttachmentsDir(), imported.StorageName)); got != contents {
		t.Fatalf("published file = %d bytes, want %d matching bytes", len(got), len(contents))
	}
	if listed, err := service.List(ctx, "t"); err != nil || len(listed) != 1 || listed[0].ID != imported.ID {
		t.Fatalf("published metadata = %+v, %v", listed, err)
	}
	if stages := importStages(s.Root()); len(stages) != 0 {
		t.Fatalf("successful import leaked staging directories: %v", stages)
	}
}

// Canceling a staged copy removes its temporary files and leaves no attachment metadata.
func TestCancelDuringImportCopyCleansStaging(t *testing.T) {
	s, service := fixture(t)
	copyCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	_, finish := pauseImport(t, s, service, copyCtx, "pending data")
	cancel()
	if _, err := finish(); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled import = %v", err)
	}
	expectEmptyImportStorage(t, s)
}

// An import rechecks task and project visibility after an unlocked copy.
func TestImportRechecksVisibilityBeforePublication(t *testing.T) {
	for _, tc := range []struct {
		name string
		want error
	}{
		{"task", model.ErrNotFound},
		{"project", model.ErrParentDeleted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, service := fixture(t)
			_, finish := pauseImport(t, s, service, ctx, "pending data")
			p := projects.New(s)
			whileCopyPaused(t, func() error {
				if tc.name == "task" {
					return p.DeleteTask(ctx, "t")
				}
				return p.DeleteProject(ctx, "p")
			})
			if _, err := finish(); !errors.Is(err, tc.want) {
				t.Fatalf("import to a deleted %s = %v, want %v", tc.name, err, tc.want)
			}
			expectEmptyImportStorage(t, s)
		})
	}
}

// Restore leaves active staging alone and a copy started before restore cannot publish to the replacement database.
func TestRestoreDuringImportCopyDiscardsOldStaging(t *testing.T) {
	s, service := fixture(t)
	backups := backup.New(s)
	archive := filepath.Join(t.TempDir(), "source.zip")
	if err := backups.Export(ctx, archive); err != nil {
		t.Fatal(err)
	}
	exec(t, s, `UPDATE tasks SET title='changed after backup' WHERE id='t'`)
	staged, finish := pauseImport(t, s, service, ctx, "old session copy")
	whileCopyPaused(t, func() error {
		if _, err := backups.Restore(ctx, archive); err != nil {
			return err
		}
		if _, err := os.Stat(staged); err != nil {
			return fmt.Errorf("restore removed the active staging file: %w", err)
		}
		return nil
	})

	if _, err := finish(); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("import started before restore = %v, want conflict", err)
	}
	expectEmptyImportStorage(t, s)
	task, err := projects.New(s).GetTask(ctx, "t")
	if err != nil || task.Title != "task" {
		t.Fatalf("expected restored task: %+v, %v", task, err)
	}
	if _, err := service.Import(ctx, "t", source(t, "new.txt", "new import after restore")); err != nil {
		t.Fatalf("new import after restore: %v", err)
	}
}

// A failure at COMMIT rolls back the attachment metadata and both filesystem links.
func TestFailedImportCommitCleansStagingAndPublishedFile(t *testing.T) {
	s, service := fixture(t)
	exec(t, s, `CREATE TABLE attachment_commit_guard (
		task_id TEXT REFERENCES tasks(id) DEFERRABLE INITIALLY DEFERRED
	)`)
	exec(t, s, `CREATE TRIGGER reject_attachment_commit AFTER INSERT ON attachments
		BEGIN INSERT INTO attachment_commit_guard VALUES('missing-task'); END`)
	src := source(t, "commit.txt", "contents")
	if _, err := service.Import(ctx, "t", src); err == nil || !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		t.Fatalf("expected deferred foreign key failure, got %v", err)
	}
	expectEmptyImportStorage(t, s)

	exec(t, s, "DROP TRIGGER reject_attachment_commit")
	if _, err := service.Import(ctx, "t", src); err != nil {
		t.Fatalf("retry after commit failure: %v", err)
	}
}
