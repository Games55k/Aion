package backend_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aion/backend"
	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"
)

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
	path, err := target.Attachments.ManagedPath(ctx, attachment.ID)
	if err != nil {
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
	managed, err := a.Attachments.ManagedPath(ctx, file.ID)
	if err != nil {
		t.Fatal(err)
	}
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

	// Re-delete app1, lock attachments dir to simulate GC failure, and verify ClearTrash reports partial cleanup error after clearing DB metadata.
	if err := a.JobApplications.Delete(ctx, app1.ID); err != nil {
		t.Fatal(err)
	}
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
