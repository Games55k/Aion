package backend_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aion/backend"
	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"
)

// Regression: attachment and backup exports used to accept live store paths and case aliases of managed attachments.
func TestExportsRejectProtectedAndCaseAliasedPaths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	app, err := backend.Open(ctx, backend.Options{DataDir: dir, DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "protected export"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "attachment"})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "aion.sqlite")
	payload := []byte("attachment content")
	if err := os.WriteFile(source, payload, 0600); err != nil {
		t.Fatal(err)
	}
	att, err := app.Attachments.Import(ctx, task.ID, source)
	if err != nil {
		t.Fatal(err)
	}
	for _, protected := range []string{
		"aion.sqlite", ".aion.lock", ".backup-auto-state.json", "aion.sqlite-wal", "attachments/escape.bin",
	} {
		dest := filepath.Join(dir, protected)
		if err := app.Attachments.ExportOverwrite(ctx, att.ID, dest); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("attachment export to %s = %v", protected, err)
		}
		if err := app.Backups.ExportOverwrite(ctx, dest); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("backup export to %s = %v", protected, err)
		}
	}
	if got, err := app.Projects.GetTask(ctx, task.ID); err != nil || got.Title != task.Title {
		t.Fatalf("live database was changed: %+v, %v", got, err)
	}

	t.Run("case alias", func(t *testing.T) {
		alias := filepath.Join(dir, "ATTACHMENTS", att.StorageName)
		if _, err := os.Stat(alias); errors.Is(err, os.ErrNotExist) {
			t.Skip("filesystem is case-sensitive")
		} else if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte("replacement content"), 0600); err != nil {
			t.Fatal(err)
		}
		replacement, err := app.Attachments.Import(ctx, task.ID, source)
		if err != nil {
			t.Fatal(err)
		}
		if err := app.Attachments.ExportOverwrite(ctx, replacement.ID, alias); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("attachment export to a case alias = %v", err)
		}
		got, err := os.ReadFile(alias)
		if err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("case alias changed the managed attachment: %q, %v", got, err)
		}
	})
}

// Exporting and restoring across data directories preserves projects, independent trash state, attachments, and settings.
func TestBackupMigrationAcrossDataDirectories(t *testing.T) {
	ctx := context.Background()
	source, err := backend.Open(ctx, backend.Options{DataDir: t.TempDir(), DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	p, err := source.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Aion 成长计划", Icon: "α", Description: "离线项目"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := source.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "记录第一步", Description: "# Markdown\n本地内容", Tags: []string{"成长"}, Subtasks: []model.Subtask{{Title: "开始", Done: true}}, DueDate: "2026-12-01"})
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := source.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "先前删除的任务"})
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(t.TempDir(), "资料.txt")
	if err = os.WriteFile(original, []byte("本地附件\n"), 0600); err != nil {
		t.Fatal(err)
	}
	attachment, err := source.Attachments.Import(ctx, task.ID, original)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(original); err != nil {
		t.Fatal(err)
	}
	if err = source.Settings.SetTheme(ctx, "dark"); err != nil {
		t.Fatal(err)
	}
	if err = source.Projects.DeleteTask(ctx, deleted.ID); err != nil {
		t.Fatal(err)
	}
	if err = source.Projects.DeleteProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "migration.zip")
	if err = source.Backups.Export(ctx, archive); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	target, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	result, err := target.Backups.Restore(ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(result.SafetyBackupPath); err != nil {
		t.Fatal("missing safety backup", err)
	}
	if err = target.Projects.RestoreProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	tasks, err := target.Projects.ListTasks(ctx, p.ID, projects.TaskFilter{})
	if err != nil || len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("restored tasks=%+v err=%v", tasks, err)
	}
	if _, err = target.Projects.GetTask(ctx, deleted.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatal("independently deleted task restored", err)
	}
	path := filepath.Join(t.TempDir(), "restored.txt")
	if err := target.Attachments.ExportOverwrite(ctx, attachment.ID, path); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "本地附件\n" {
		t.Fatal(string(content), err)
	}
	if theme, err := target.Settings.Theme(ctx); err != nil || theme != "dark" {
		t.Fatal(theme, err)
	}
	if err = target.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err = reopened.Projects.GetTask(ctx, task.ID); err != nil {
		t.Fatal("restored database failed reopen", err)
	}
}

// Permanently deleting a trashed project purges its metadata and collects unreferenced attachment files while soft deletion retains them.
func TestPurgeCoordinatesMetadataAndFiles(t *testing.T) {
	ctx := context.Background()
	a, err := backend.Open(ctx, backend.Options{DataDir: t.TempDir(), DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	p, err := a.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Purge"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := a.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "Attachment"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "file.txt")
	if err = os.WriteFile(path, []byte("keep until permanent deletion"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := a.Attachments.Import(ctx, task.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(a.DataDir(), "attachments", file.StorageName)
	if err = a.PermanentlyDelete(ctx, projects.TrashProject, p.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatal("live project purged", err)
	}
	if err = a.Projects.DeleteProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Attachments.GC(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(managed); err != nil {
		t.Fatal("GC removed recycled file", err)
	}
	if err = a.PermanentlyDelete(ctx, projects.TrashProject, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(managed); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("orphan not removed", err)
	}
	if err = a.ClearTrash(ctx); err != nil {
		t.Fatal(err)
	}
}

// Module registration rejects blank names and trims surrounding whitespace while Settings round-trips typed preferences.
func TestModuleRegistrationAndSettingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	a, err := backend.Open(ctx, backend.Options{DataDir: t.TempDir(), DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	if err := a.Modules.Register(backend.ModuleDescriptor{ID: "notes", Name: "   ", Version: 1}); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("blank module name = %v, want ErrInvalid", err)
	}
	if err := a.Modules.Register(backend.ModuleDescriptor{ID: "notes", Name: " 笔记 ", Version: 1}); err != nil {
		t.Fatalf("register trimmed module failed: %v", err)
	}
	mods := a.Modules.List()
	if len(mods) != 3 || mods[0].ID != "projects" || mods[1].ID != "job_applications" || mods[2].Name != "笔记" {
		t.Fatalf("unexpected modules: %+v", mods)
	}
	if err := a.Settings.Set(ctx, "notes.sidebar_width", 280); err != nil {
		t.Fatal(err)
	}
	var width int
	if err := a.Settings.Get(ctx, "notes.sidebar_width", &width); err != nil || width != 280 {
		t.Fatalf("Settings.Get width = %d, err = %v", width, err)
	}
}

// Regression: trash operations reported existing tasks or applications in the wrong state as missing.
func TestTrashErrorsDistinguishMissingFromStateConflict(t *testing.T) {
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{DataDir: t.TempDir(), DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	project, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Project"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.Projects.CreateTask(ctx, project.ID, projects.TaskInput{Title: "Task"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{CompanyName: "Company", PositionName: "Role"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.JobApplications.Close(ctx, jobapplications.CloseInput{ID: job.ID, Outcome: model.JobOutcomeRejected}); err != nil {
		t.Fatal(err)
	}

	// Exercise the task before permanently deleting its project.
	for _, tc := range []struct {
		kind, id   string
		softDelete func(context.Context, string) error
	}{
		{projects.TrashTask, task.ID, app.Projects.DeleteTask},
		{projects.TrashProject, project.ID, app.Projects.DeleteProject},
		{jobapplications.TrashKind, job.ID, app.JobApplications.Delete},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			expect := func(operation string, got, want error) {
				t.Helper()
				if !errors.Is(got, want) {
					t.Errorf("%s = %v, want %v", operation, got, want)
				}
			}
			const missing = "missing"
			expect("delete missing", tc.softDelete(ctx, missing), model.ErrNotFound)
			expect("restore missing", app.RestoreTrash(ctx, tc.kind, missing), model.ErrNotFound)
			expect("purge missing", app.PermanentlyDelete(ctx, tc.kind, missing), model.ErrNotFound)
			expect("restore live", app.RestoreTrash(ctx, tc.kind, tc.id), model.ErrConflict)
			expect("purge live", app.PermanentlyDelete(ctx, tc.kind, tc.id), model.ErrConflict)
			if err := tc.softDelete(ctx, tc.id); err != nil {
				t.Fatalf("delete live: %v", err)
			}
			expect("delete already recycled", tc.softDelete(ctx, tc.id), model.ErrConflict)
			if err := app.RestoreTrash(ctx, tc.kind, tc.id); err != nil {
				t.Fatalf("restore recycled: %v", err)
			}
			if err := tc.softDelete(ctx, tc.id); err != nil {
				t.Fatalf("delete restored: %v", err)
			}
			if err := app.PermanentlyDelete(ctx, tc.kind, tc.id); err != nil {
				t.Fatalf("purge recycled: %v", err)
			}
			expect("restore purged", app.RestoreTrash(ctx, tc.kind, tc.id), model.ErrNotFound)
		})
	}
}

// Task restore reports a recycled parent and independently recycled tasks can still be purged.
func TestTaskTrashOperationsWithDeletedParent(t *testing.T) {
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{DataDir: t.TempDir(), DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	project, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Project"})
	if err != nil {
		t.Fatal(err)
	}
	live, err := app.Projects.CreateTask(ctx, project.ID, projects.TaskInput{Title: "Live task"})
	if err != nil {
		t.Fatal(err)
	}
	recycled, err := app.Projects.CreateTask(ctx, project.ID, projects.TaskInput{Title: "Independently recycled task"})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Projects.DeleteTask(ctx, recycled.ID); err != nil {
		t.Fatal(err)
	}
	if err := app.Projects.DeleteProject(ctx, project.ID); err != nil {
		t.Fatal(err)
	}

	if err := app.Projects.DeleteTask(ctx, live.ID); !errors.Is(err, model.ErrParentDeleted) {
		t.Errorf("delete live child = %v, want ErrParentDeleted", err)
	}
	if err := app.Projects.DeleteTask(ctx, recycled.ID); !errors.Is(err, model.ErrConflict) {
		t.Errorf("delete recycled child = %v, want ErrConflict", err)
	}
	for _, task := range []model.Task{live, recycled} {
		if err := app.RestoreTrash(ctx, projects.TrashTask, task.ID); !errors.Is(err, model.ErrParentDeleted) {
			t.Errorf("restore %q = %v, want ErrParentDeleted", task.Title, err)
		}
	}
	if err := app.PermanentlyDelete(ctx, projects.TrashTask, live.ID); !errors.Is(err, model.ErrConflict) {
		t.Errorf("purge live child = %v, want ErrConflict", err)
	}
	if err := app.PermanentlyDelete(ctx, projects.TrashTask, recycled.ID); err != nil {
		t.Fatalf("purge independently recycled child: %v", err)
	}
}

// Global recycle bin aggregates projects, tasks, and job applications while dispatching restores, cascades, and attachment GC cleanly.
func TestCrossModuleTrashDispatchAndAttachmentCleanup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	a, err := backend.Open(ctx, backend.Options{DataDir: dir, DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	p, err := a.Projects.CreateProject(ctx, projects.ProjectInput{Name: "工作项目", Icon: "folder"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := a.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "带附件任务"})
	if err != nil {
		t.Fatal(err)
	}
	attFile := filepath.Join(t.TempDir(), "spec.txt")
	if err := os.WriteFile(attFile, []byte("attachment-data"), 0600); err != nil {
		t.Fatal(err)
	}
	att, err := a.Attachments.Import(ctx, task.ID, attFile)
	if err != nil {
		t.Fatal(err)
	}

	app1, err := a.JobApplications.Create(ctx, jobapplications.CreateInput{CompanyName: "星辰科技", PositionName: "Go 工程师", AppliedDate: "2026-11-01"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.JobApplications.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app1.ID, ToStage: model.JobStageInterview1}); err != nil {
		t.Fatal(err)
	}
	app2, err := a.JobApplications.Create(ctx, jobapplications.CreateInput{CompanyName: "海岳科技", PositionName: "架构师", AppliedDate: "2026-11-02"})
	if err != nil {
		t.Fatal(err)
	}

	if err := a.Projects.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := a.JobApplications.Delete(ctx, app1.ID); err != nil {
		t.Fatal(err)
	}
	if err := a.JobApplications.Delete(ctx, app2.ID); err != nil {
		t.Fatal(err)
	}

	trash, err := a.ListTrash(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(trash) != 3 {
		t.Fatalf("trash len = %d, want 3 (%+v)", len(trash), trash)
	}

	// Restoring a job application preserves full timeline history.
	if err := a.RestoreTrash(ctx, jobapplications.TrashKind, app1.ID); err != nil {
		t.Fatal(err)
	}
	events, err := a.JobApplications.ListEvents(ctx, app1.ID)
	if err != nil || len(events) != 2 {
		t.Fatalf("restored events = %+v, err = %v", events, err)
	}

	// Permanently deleting app2 removes only app2 and keeps task intact.
	if err := a.PermanentlyDelete(ctx, jobapplications.TrashKind, app2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.JobApplications.Get(ctx, app2.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected app2 deleted, err=%v", err)
	}

	// Re-delete app1 and inject a failure in the job module after project cleanup starts.
	if err := a.JobApplications.Delete(ctx, app1.ID); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "aion.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TRIGGER fail_clear BEFORE DELETE ON job_applications BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := a.ClearTrash(ctx); err == nil {
		t.Fatal("expected metadata deletion failure")
	}
	if items, err := a.ListTrash(ctx); err != nil || len(items) != 2 {
		t.Fatalf("failed clear lost trash entries: %+v, %v", items, err)
	}
	if _, err := db.Exec("DROP TRIGGER fail_clear"); err != nil {
		t.Fatal(err)
	}

	// A later GC failure still reports committed metadata deletion.
	attDir := filepath.Join(dir, "attachments")
	if err := os.Chmod(attDir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(attDir, 0700)

	err = a.ClearTrash(ctx)
	var cleanupErr *backend.AttachmentCleanupError
	if !errors.As(err, &cleanupErr) {
		t.Fatalf("ClearTrash err = %v, want *AttachmentCleanupError", err)
	}
	trashAfter, err := a.ListTrash(ctx)
	if err != nil || len(trashAfter) != 0 {
		t.Fatalf("expected empty trash metadata after partial-success ClearTrash, got %+v err=%v", trashAfter, err)
	}
	if _, statErr := os.Stat(filepath.Join(attDir, att.StorageName)); statErr != nil {
		t.Fatalf("expected unremovable attachment file to remain during chmod 0500, got %v", statErr)
	}
}
