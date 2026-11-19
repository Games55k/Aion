package backend_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"aion/backend"
	"aion/backend/model"
	"aion/backend/projects"
)

// Regression: project and task lifecycle writes ignored the backend clock and used system time.
func TestProjectAndTaskLifecycleUseBackendClock(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 2, 3, 4, 123456789, time.FixedZone("test offset", 8*60*60))
	app, err := backend.Open(ctx, backend.Options{
		DataDir: t.TempDir(), DisableScheduler: true, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	db, err := sql.Open("sqlite", filepath.Join(app.DataDir(), "aion.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const projectTimes = "SELECT created_at, updated_at, deleted_at FROM projects WHERE id = ?"
	const taskTimes = "SELECT created_at, updated_at, deleted_at FROM tasks WHERE id = ?"
	expectTimes := func(operation, query, id string, createdAt time.Time, deleted bool) {
		t.Helper()
		var created, updated string
		var deletedAt sql.NullString
		if err := db.QueryRowContext(ctx, query, id).Scan(&created, &updated, &deletedAt); err != nil {
			t.Fatal(err)
		}
		wantCreated, wantUpdated := createdAt.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano)
		wantDeleted := sql.NullString{}
		if deleted {
			wantDeleted = sql.NullString{String: wantUpdated, Valid: true}
		}
		if created != wantCreated || updated != wantUpdated || deletedAt != wantDeleted {
			t.Fatalf("%s stored (%q, %q, %+v), want (%q, %q, %+v)", operation,
				created, updated, deletedAt, wantCreated, wantUpdated, wantDeleted)
		}
	}
	advance := func() { now = now.Add(time.Minute) }

	projectCreated := now
	p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Project"})
	if err != nil {
		t.Fatal(err)
	}
	expectTimes("create project", projectTimes, p.ID, projectCreated, false)
	advance()
	if _, err := app.Projects.UpdateProject(ctx, p.ID, projects.ProjectInput{Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	expectTimes("update project", projectTimes, p.ID, projectCreated, false)
	advance()
	if _, err := app.Projects.ArchiveProject(ctx, p.ID, true); err != nil {
		t.Fatal(err)
	}
	expectTimes("archive project", projectTimes, p.ID, projectCreated, false)

	advance()
	taskCreated := now
	task, err := app.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "Task", Status: model.Done})
	if err != nil {
		t.Fatal(err)
	}
	expectTimes("create task", taskTimes, task.ID, taskCreated, false)
	var completed string
	if err := db.QueryRowContext(ctx, "SELECT completed_at FROM tasks WHERE id = ?", task.ID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if want := now.UTC().Format(time.RFC3339Nano); completed != want {
		t.Fatalf("completed_at = %q, want %q", completed, want)
	}

	advance()
	if _, err := app.Projects.UpdateTask(ctx, task.ID, projects.TaskInput{Title: "Edited", Status: model.Done, Priority: model.NoPriority}); err != nil {
		t.Fatal(err)
	}
	expectTimes("update task", taskTimes, task.ID, taskCreated, false)
	advance()
	if err := app.Projects.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	expectTimes("delete task", taskTimes, task.ID, taskCreated, true)
	advance()
	if err := app.RestoreTrash(ctx, projects.TrashTask, task.ID); err != nil {
		t.Fatal(err)
	}
	expectTimes("restore task", taskTimes, task.ID, taskCreated, false)

	advance()
	if err := app.Projects.DeleteProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	expectTimes("delete project", projectTimes, p.ID, projectCreated, true)
	advance()
	if err := app.RestoreTrash(ctx, projects.TrashProject, p.ID); err != nil {
		t.Fatal(err)
	}
	expectTimes("restore project", projectTimes, p.ID, projectCreated, false)
}

// Regression: task moves and reorders ignored the backend clock for both the moved task and its reindexed neighbors.
func TestTaskColumnWritesShareBackendTimestamp(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 2, 3, 4, 123456789, time.UTC)
	calls := 0
	app, err := backend.Open(ctx, backend.Options{
		DataDir: t.TempDir(), DisableScheduler: true,
		Now: func() time.Time {
			calls++
			now = now.Add(time.Second)
			return now
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Board"})
	if err != nil {
		t.Fatal(err)
	}
	create := func(title string, status model.Status) model.Task {
		t.Helper()
		task, err := app.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: title, Status: status})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	first := create("First", model.Planned)
	second := create("Second", model.Planned)
	third := create("Third", model.Planned)
	doneA := create("Done A", model.Done)
	doneB := create("Done B", model.Done)
	untouched := create("In progress", model.InProgress)

	expectUpdated := func(want time.Time, tasks ...model.Task) {
		t.Helper()
		for _, task := range tasks {
			stored, err := app.Projects.GetTask(ctx, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !stored.UpdatedAt.Equal(want) {
				t.Errorf("%s updated_at = %v, want %v", task.Title, stored.UpdatedAt, want)
			}
		}
	}
	expectClockRead := func(operation string, before int) {
		t.Helper()
		if calls != before+1 {
			t.Errorf("%s read the clock %d times, want 1", operation, calls-before)
		}
	}

	before := calls
	moved, err := app.Projects.MoveTask(ctx, first.ID, model.Done, 0)
	if err != nil {
		t.Fatal(err)
	}
	expectClockRead("move", before)
	moveTime := now
	expectUpdated(moveTime, first, second, third, doneA, doneB)
	expectUpdated(untouched.UpdatedAt, untouched)
	if moved.CompletedAt == nil || !moved.CompletedAt.Equal(moveTime) {
		t.Errorf("move completed_at = %v, want %v", moved.CompletedAt, moveTime)
	}

	before = calls
	if err := app.Projects.ReorderTasks(ctx, p.ID, model.Done, []string{doneB.ID, first.ID}); err != nil {
		t.Fatal(err)
	}
	expectClockRead("reorder", before)
	reorderTime := now
	expectUpdated(reorderTime, first, doneB)
	expectUpdated(moveTime, doneA)

	before = calls
	updated, err := app.Projects.UpdateTask(ctx, second.ID, projects.TaskInput{Title: "Second", Status: model.Done, Priority: model.NoPriority})
	if err != nil {
		t.Fatal(err)
	}
	expectClockRead("change status", before)
	expectUpdated(now, second, third)
	expectUpdated(reorderTime, first, doneB)
	if updated.CompletedAt == nil || !updated.CompletedAt.Equal(now) {
		t.Errorf("status change completed_at = %v, want %v", updated.CompletedAt, now)
	}
}
