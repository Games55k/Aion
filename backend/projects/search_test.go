package projects

import (
	"context"
	"testing"

	"aion/backend/model"
	"aion/backend/store"
)

// SearchTasks queries non-deleted tasks across projects, excludes archived projects by default, and filters by due date, status, priority, tags, and text.
func TestSearchTasksCrossProjectFiltering(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s)

	p1, err := svc.CreateProject(ctx, ProjectInput{Name: "Active A"})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := svc.CreateProject(ctx, ProjectInput{Name: "Active B"})
	if err != nil {
		t.Fatal(err)
	}
	pArch, err := svc.CreateProject(ctx, ProjectInput{Name: "Archived C"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ArchiveProject(ctx, pArch.ID, true); err != nil {
		t.Fatal(err)
	}

	t1, err := svc.CreateTask(ctx, p1.ID, TaskInput{
		Title:    "Ship Focus View",
		Status:   model.InProgress,
		Priority: model.Urgent,
		Tags:     []string{"ui", "v0.2"},
		DueDate:  "2026-11-28",
	})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := svc.CreateTask(ctx, p2.ID, TaskInput{
		Title:    "Write CLI docs",
		Status:   model.Planned,
		Priority: model.High,
		Tags:     []string{"docs", "v0.2"},
		DueDate:  "2026-12-10",
	})
	if err != nil {
		t.Fatal(err)
	}
	tTrashed, err := svc.CreateTask(ctx, p2.ID, TaskInput{
		Title:    "Trashed task",
		Status:   model.Planned,
		Priority: model.Urgent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, tTrashed.ID); err != nil {
		t.Fatal(err)
	}
	tArch, err := svc.CreateTask(ctx, pArch.ID, TaskInput{
		Title:    "Archived task",
		Status:   model.Planned,
		Priority: model.High,
		Tags:     []string{"v0.2"},
		DueDate:  "2026-11-27",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Default excludes archived and soft-deleted tasks.
	got, err := svc.SearchTasks(ctx, SearchTasksFilter{Tags: []string{"v0.2"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != t1.ID || got[1].ID != t2.ID {
		t.Fatalf("unexpected default cross-project search results: %+v", got)
	}

	// DueBefore filters inclusively and excludes tasks without due dates.
	got, err = svc.SearchTasks(ctx, SearchTasksFilter{DueBefore: "2026-11-30"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != t1.ID {
		t.Fatalf("unexpected due_before results: %+v", got)
	}

	// IncludeArchived includes archived project's tasks.
	got, err = svc.SearchTasks(ctx, SearchTasksFilter{IncludeArchived: true, Tags: []string{"v0.2"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 tasks including archived, got %d (%+v)", len(got), got)
	}
	foundArch := false
	for _, item := range got {
		if item.ID == tArch.ID {
			foundArch = true
		}
	}
	if !foundArch {
		t.Fatalf("expected archived task %s in IncludeArchived results", tArch.ID)
	}
}
