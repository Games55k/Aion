package backend_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aion/backend"
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
