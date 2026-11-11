package projects_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"aion/backend/model"
	"aion/backend/projects"
	"aion/backend/store"
)

var ctx = context.Background()

func setup(t *testing.T) (*projects.Service, *store.Store) {
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
	return projects.New(s), s
}

func must[T any](t *testing.T, value T, err error) T {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func isError(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got error %v; want errors.Is(_, %v)", err, want)
	}
}

func project(t *testing.T, s *projects.Service, name string) model.Project {
	t.Helper()
	p, err := s.CreateProject(ctx, projects.ProjectInput{Name: name})
	return must(t, p, err)
}

func task(t *testing.T, s *projects.Service, p model.Project, title string) model.Task {
	t.Helper()
	v, err := s.CreateTask(ctx, p.ID, projects.TaskInput{Title: title})
	return must(t, v, err)
}

func taskInput(t model.Task) projects.TaskInput {
	return projects.TaskInput{Title: t.Title, Description: t.Description, Status: t.Status, Priority: t.Priority,
		Tags: t.Tags, Subtasks: t.Subtasks, DueDate: t.DueDate}
}

func board(t *testing.T, s *projects.Service, p model.Project, status model.Status, want ...model.Task) {
	t.Helper()
	tasks, err := s.ListTasks(ctx, p.ID, projects.TaskFilter{Status: status})
	check(t, err)
	var gotIDs, wantIDs []string
	for i, v := range tasks {
		gotIDs = append(gotIDs, v.ID)
		if v.Position != i {
			t.Fatalf("%s position = %d, want %d", v.Title, v.Position, i)
		}
	}
	for _, v := range want {
		wantIDs = append(wantIDs, v.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("board IDs = %v, want %v", gotIDs, wantIDs)
	}
}

// Project CRUD, archiving, soft deletion, and restoration preserve project identity and board tasks across reopens.
func TestProjectLifecycleAndPersistence(t *testing.T) {
	s, st := setup(t)
	p := project(t, s, "  工作台  ")
	if p.Name != "工作台" || p.ID == "" || p.CreatedAt.IsZero() || p.Archived {
		t.Fatalf("unexpected project: %+v", p)
	}
	other := project(t, s, "Other")
	updated, err := s.UpdateProject(ctx, p.ID, projects.ProjectInput{Name: "更名", Icon: "α", Description: "介绍"})
	check(t, err)
	if updated.ID != p.ID || !updated.CreatedAt.Equal(p.CreatedAt) || updated.Icon != "α" || updated.Description != "介绍" {
		t.Fatalf("update lost fields/identity: %+v", updated)
	}
	archived, err := s.ArchiveProject(ctx, p.ID, true)
	check(t, err)
	if !archived.Archived {
		t.Fatal("archive flag missing")
	}
	active := false
	list, err := s.ListProjects(ctx, projects.ProjectFilter{Archived: &active})
	check(t, err)
	if len(list) != 1 || list[0].ID != other.ID {
		t.Fatalf("active projects: %+v", list)
	}
	active = true
	list, err = s.ListProjects(ctx, projects.ProjectFilter{Archived: &active})
	check(t, err)
	if len(list) != 1 || list[0].ID != p.ID {
		t.Fatalf("archived projects: %+v", list)
	}
	// Archived boards remain readable and editable.
	v := task(t, s, archived, "Archived task")
	check(t, s.DeleteProject(ctx, p.ID))
	_, err = s.GetProject(ctx, p.ID)
	isError(t, err, model.ErrNotFound)
	isError(t, s.DeleteProject(ctx, p.ID), model.ErrConflict)
	_, err = s.ArchiveProject(ctx, p.ID, false)
	isError(t, err, model.ErrNotFound)
	_, err = s.UpdateProject(ctx, p.ID, projects.ProjectInput{Name: "hidden"})
	isError(t, err, model.ErrNotFound)
	check(t, s.RestoreProject(ctx, p.ID))
	isError(t, s.RestoreProject(ctx, p.ID), model.ErrConflict)
	got, err := s.GetProject(ctx, p.ID)
	check(t, err)
	if !got.Archived || got.Name != "更名" {
		t.Fatalf("restore changed project content: %+v", got)
	}
	_, err = s.ArchiveProject(ctx, p.ID, false)
	check(t, err)
	root := st.Root()
	check(t, st.Close())
	reopened, err := store.Open(root)
	check(t, err)
	t.Cleanup(func() { check(t, reopened.Close()) })
	s = projects.New(reopened)
	got, err = s.GetProject(ctx, p.ID)
	check(t, err)
	if got.Archived || got.Icon != "α" || !got.CreatedAt.Equal(p.CreatedAt) {
		t.Fatalf("persistence: %+v", got)
	}
	board(t, s, got, model.Planned, v)
}

// Task creation and updates normalize tags and subtasks, reject invalid fields without mutation, and track completion timestamps.
func TestTaskContentValidationAndCompletion(t *testing.T) {
	s, _ := setup(t)
	p := project(t, s, "P")
	in := projects.TaskInput{Title: "  中文任务  ", Description: "# Markdown", DueDate: "2028-02-29",
		Tags: []string{" Go ", "中文", "Go"}, Subtasks: []model.Subtask{{Title: "  One  ", Done: true}, {ID: "existing", Title: "Two"}}}
	v, err := s.CreateTask(ctx, p.ID, in)
	check(t, err)
	if v.Title != "中文任务" || v.CompletedAt != nil ||
		!reflect.DeepEqual(v.Tags, []string{"Go", "中文"}) || v.Subtasks[0].ID == "" || v.Subtasks[0].Title != "One" {
		t.Fatalf("normalization: %+v", v)
	}
	got, err := s.GetTask(ctx, v.ID)
	check(t, err)
	if !reflect.DeepEqual(v, got) {
		t.Fatalf("task round trip mismatch: %+v / %+v", v, got)
	}
	bad := []struct {
		name string
		edit func(*projects.TaskInput)
	}{
		{"blank title", func(v *projects.TaskInput) { v.Title = " \n " }},
		{"bad status", func(v *projects.TaskInput) { v.Status = "unknown" }},
		{"bad priority", func(v *projects.TaskInput) { v.Priority = "critical" }},
		{"non leap date", func(v *projects.TaskInput) { v.DueDate = "2026-02-29" }},
		{"short date", func(v *projects.TaskInput) { v.DueDate = "2028-2-29" }},
		{"timestamp date", func(v *projects.TaskInput) { v.DueDate = "2028-02-29T00:00:00Z" }},
		{"blank tag", func(v *projects.TaskInput) { v.Tags = []string{"Go", " "} }},
		{"blank subtask", func(v *projects.TaskInput) { v.Subtasks = []model.Subtask{{Title: " "}} }},
		{"duplicate subtask", func(v *projects.TaskInput) {
			v.Subtasks = []model.Subtask{{ID: "same", Title: "A"}, {ID: "same", Title: "B"}}
		}},
		{"space subtask id", func(v *projects.TaskInput) { v.Subtasks = []model.Subtask{{ID: " ", Title: "A"}} }},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			input := taskInput(v)
			tt.edit(&input)
			_, err := s.CreateTask(ctx, p.ID, input)
			isError(t, err, model.ErrInvalid)
			_, err = s.UpdateTask(ctx, v.ID, input)
			isError(t, err, model.ErrInvalid)
		})
	}
	got, err = s.GetTask(ctx, v.ID)
	check(t, err)
	if !reflect.DeepEqual(v, got) {
		t.Fatal("validation mutated task")
	}
	input := taskInput(v)
	input.Status = ""
	_, err = s.UpdateTask(ctx, v.ID, input)
	isError(t, err, model.ErrInvalid)
	input = taskInput(v)
	input.Status = model.Done
	input.Title = "已完成"
	completed, err := s.UpdateTask(ctx, v.ID, input)
	check(t, err)
	if completed.ID != v.ID || completed.ProjectID != p.ID || !completed.CreatedAt.Equal(v.CreatedAt) || completed.CompletedAt == nil {
		t.Fatalf("completed task: %+v", completed)
	}
	// Editing and reordering an already done task preserves its completion time.
	input = taskInput(completed)
	input.Description = "changed"
	edited, err := s.UpdateTask(ctx, v.ID, input)
	check(t, err)
	moved, err := s.MoveTask(ctx, v.ID, model.Done, 0)
	check(t, err)
	if !edited.CompletedAt.Equal(*completed.CompletedAt) || !moved.CompletedAt.Equal(*completed.CompletedAt) {
		t.Fatal("same-state editing changed completion time")
	}
	reopened, err := s.MoveTask(ctx, v.ID, model.InProgress, 0)
	check(t, err)
	if reopened.CompletedAt != nil {
		t.Fatal("reopened task kept completion time")
	}
	completedAgain, err := s.MoveTask(ctx, v.ID, model.Done, 0)
	check(t, err)
	if completedAgain.CompletedAt == nil || completedAgain.CompletedAt.Before(*completed.CompletedAt) {
		t.Fatal("recompletion timestamp incorrect")
	}
	// Full replacement clears omitted optional content, but cannot alter identity.
	replaced, err := s.UpdateTask(ctx, v.ID, projects.TaskInput{Title: "replacement", Status: model.Planned, Priority: model.Low})
	check(t, err)
	if replaced.Description != "" || replaced.DueDate != "" || len(replaced.Tags) != 0 || len(replaced.Subtasks) != 0 || replaced.CompletedAt != nil || replaced.ID != v.ID || replaced.ProjectID != p.ID {
		t.Fatalf("full replacement failed: %+v", replaced)
	}
	initialDone, err := s.CreateTask(ctx, p.ID, projects.TaskInput{Title: "done on creation", Status: model.Done})
	check(t, err)
	if initialDone.CompletedAt == nil {
		t.Fatal("created done task lacks completion time")
	}
}

// Moving tasks across columns and reordering subsets keep dense zero-based positions without disturbing other tasks.
func TestMoveAndSubsetReorderPreserveOtherTasks(t *testing.T) {
	s, _ := setup(t)
	p := project(t, s, "P")
	a, b, c, d, e := task(t, s, p, "A"), task(t, s, p, "B"), task(t, s, p, "C"), task(t, s, p, "D"), task(t, s, p, "E")
	other := project(t, s, "Other")
	f := task(t, s, other, "F")
	check(t, s.ReorderTasks(ctx, p.ID, model.Planned, []string{d.ID, b.ID}))
	board(t, s, p, model.Planned, a, d, c, b, e)
	_, err := s.MoveTask(ctx, c.ID, model.Planned, 0)
	check(t, err)
	board(t, s, p, model.Planned, c, a, d, b, e)
	_, err = s.MoveTask(ctx, c.ID, model.InProgress, 0)
	check(t, err)
	board(t, s, p, model.Planned, a, d, b, e)
	board(t, s, p, model.InProgress, c)
	_, err = s.MoveTask(ctx, d.ID, model.InProgress, 0)
	check(t, err)
	board(t, s, p, model.InProgress, d, c)
	board(t, s, p, model.Planned, a, b, e)
	_, err = s.MoveTask(ctx, a.ID, model.Planned, 2)
	check(t, err)
	board(t, s, p, model.Planned, b, e, a)
	_, err = s.MoveTask(ctx, a.ID, model.Planned, 3)
	isError(t, err, model.ErrInvalid)
	_, err = s.MoveTask(ctx, a.ID, model.Done, -1)
	isError(t, err, model.ErrInvalid)
	_, err = s.MoveTask(ctx, a.ID, "unknown", 0)
	isError(t, err, model.ErrInvalid)
	for _, ids := range [][]string{{a.ID, a.ID}, {a.ID, f.ID}, {d.ID}, {""}, {"missing"}} {
		isError(t, s.ReorderTasks(ctx, p.ID, model.Planned, ids), model.ErrInvalid)
	}
	board(t, s, p, model.Planned, b, e, a)
	board(t, s, other, model.Planned, f)
	check(t, s.ReorderTasks(ctx, p.ID, model.Planned, []string{a.ID, b.ID, e.ID}))
	board(t, s, p, model.Planned, a, b, e)
	check(t, s.ReorderTasks(ctx, p.ID, model.Planned, nil))
	input := taskInput(c)
	input.Status = model.Planned
	_, err = s.UpdateTask(ctx, c.ID, input)
	check(t, err)
	board(t, s, p, model.Planned, a, b, e, c)
	board(t, s, p, model.InProgress, d)
	check(t, s.DeleteTask(ctx, b.ID))
	board(t, s, p, model.Planned, a, e, c)
	check(t, s.RestoreTask(ctx, b.ID))
	board(t, s, p, model.Planned, a, b, e, c)
}

// Listing tasks filters by project, case-insensitive query, tags, priority, and status while matching literal special characters.
func TestProjectScopedSearchAndFilters(t *testing.T) {
	s, _ := setup(t)
	p := project(t, s, "P")
	other := project(t, s, "Other")
	inputs := []projects.TaskInput{
		{Title: "ÉCOLE 中文", Description: "notes", Tags: []string{"Go", "后台"}, Priority: model.High},
		{Title: "B", Description: "École in description 100%_literal", Tags: []string{"Go"}, Priority: model.Low},
		{Title: "C", Description: "none", Tags: []string{"École", "后台"}, Priority: model.High, Status: model.Done},
	}
	var tasks []model.Task
	for _, in := range inputs {
		v, err := s.CreateTask(ctx, p.ID, in)
		check(t, err)
		tasks = append(tasks, v)
	}
	_, err := s.CreateTask(ctx, other.ID, inputs[0])
	check(t, err)
	deleted, err := s.CreateTask(ctx, p.ID, inputs[0])
	check(t, err)
	check(t, s.DeleteTask(ctx, deleted.ID))
	cases := []struct {
		filter projects.TaskFilter
		want   []int
	}{
		{projects.TaskFilter{}, []int{0, 1, 2}},
		{projects.TaskFilter{Query: "éCoLe"}, []int{0, 1, 2}},
		{projects.TaskFilter{Query: "中文"}, []int{0}},
		{projects.TaskFilter{Query: "后台"}, []int{0, 2}},
		{projects.TaskFilter{Query: "%_"}, []int{1}},
		{projects.TaskFilter{Query: "[missing]"}, nil},
		{projects.TaskFilter{Tags: []string{"Go", "后台"}}, []int{0}},
		{projects.TaskFilter{Tags: []string{"go"}}, nil},
		{projects.TaskFilter{Priority: model.High}, []int{0, 2}},
		{projects.TaskFilter{Status: model.Done}, []int{2}},
		{projects.TaskFilter{Status: model.Done, Priority: model.High, Tags: []string{" École ", "后台"}}, []int{2}},
		{projects.TaskFilter{Query: "école", Tags: []string{" Go "}, Priority: model.Low, Status: model.Planned}, []int{1}},
	}
	for _, tt := range cases {
		got, err := s.ListTasks(ctx, p.ID, tt.filter)
		check(t, err)
		var gotIDs, wantIDs []string
		for _, v := range got {
			gotIDs = append(gotIDs, v.ID)
		}
		for _, index := range tt.want {
			wantIDs = append(wantIDs, tasks[index].ID)
		}
		if !reflect.DeepEqual(gotIDs, wantIDs) {
			t.Fatalf("filter %+v got %v, want %v", tt.filter, gotIDs, wantIDs)
		}
		// The in-memory matcher used by the board must agree with the query.
		var matched []string
		for _, v := range tasks {
			if tt.filter.Matches(v) {
				matched = append(matched, v.ID)
			}
		}
		if !reflect.DeepEqual(matched, wantIDs) {
			t.Fatalf("filter %+v Matches %v, want %v", tt.filter, matched, wantIDs)
		}
	}
	board(t, s, p, model.Planned, tasks[0], tasks[1])
	for _, f := range []projects.TaskFilter{{Status: "bad"}, {Priority: "bad"}, {Tags: []string{" "}}} {
		_, err := s.ListTasks(ctx, p.ID, f)
		isError(t, err, model.ErrInvalid)
	}
}

// Restoring a soft-deleted project restores its active tasks while keeping independently deleted tasks in Trash.
func TestTrashProjectRestoreKeepsIndependentTaskDeletion(t *testing.T) {
	s, st := setup(t)
	p := project(t, s, "P")
	a := task(t, s, p, "visible")
	b := task(t, s, p, "independently deleted")
	check(t, s.DeleteTask(ctx, b.ID))
	check(t, s.DeleteProject(ctx, p.ID))
	// Deleting the project must not set or overwrite any child deleted_at.
	check(t, st.Read(ctx, func(tx *sql.Tx) error {
		var deleted sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT deleted_at FROM tasks WHERE id=?`, a.ID).Scan(&deleted); err != nil {
			return err
		}
		if deleted.Valid {
			t.Error("project deletion marked a live task deleted")
		}
		return nil
	}))
	trash, err := s.ListTrash(ctx)
	check(t, err)
	if len(trash) != 2 || trash[0].Kind != projects.TrashProject || trash[0].ID != p.ID || trash[1].ID != b.ID || trash[1].ProjectID != p.ID || trash[1].DeletedAt.IsZero() {
		t.Fatalf("trash entries: %+v", trash)
	}
	_, err = s.GetTask(ctx, a.ID)
	isError(t, err, model.ErrParentDeleted)
	_, err = s.ListTasks(ctx, p.ID, projects.TaskFilter{})
	isError(t, err, model.ErrParentDeleted)
	_, err = s.CreateTask(ctx, p.ID, projects.TaskInput{Title: "hidden parent"})
	isError(t, err, model.ErrParentDeleted)
	_, err = s.UpdateTask(ctx, a.ID, taskInput(a))
	isError(t, err, model.ErrParentDeleted)
	_, err = s.MoveTask(ctx, a.ID, model.Done, 0)
	isError(t, err, model.ErrParentDeleted)
	isError(t, s.ReorderTasks(ctx, p.ID, model.Planned, []string{a.ID}), model.ErrParentDeleted)
	isError(t, s.DeleteTask(ctx, a.ID), model.ErrParentDeleted)
	isError(t, s.RestoreTask(ctx, b.ID), model.ErrParentDeleted)
	isError(t, s.RestoreTask(ctx, a.ID), model.ErrParentDeleted)
	check(t, s.RestoreTrash(ctx, projects.TrashProject, p.ID))
	board(t, s, p, model.Planned, a)
	_, err = s.GetTask(ctx, b.ID)
	isError(t, err, model.ErrNotFound)
	trash, err = s.ListTrash(ctx)
	check(t, err)
	if len(trash) != 1 || trash[0].ID != b.ID {
		t.Fatalf("independent task lost from trash: %+v", trash)
	}
	check(t, s.RestoreTrash(ctx, projects.TrashTask, b.ID))
	board(t, s, p, model.Planned, a, b)
	isError(t, s.RestoreTask(ctx, b.ID), model.ErrConflict)
	trash, err = s.ListTrash(ctx)
	check(t, err)
	if len(trash) != 0 {
		t.Fatal("trash not empty after restore")
	}
}

func addAttachment(t *testing.T, st *store.Store, v model.Task) string {
	t.Helper()
	id := model.NewID()
	path := filepath.Join(st.AttachmentsDir(), id)
	check(t, os.WriteFile(path, []byte("keep for GC"), 0600))
	check(t, st.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO attachments(id,task_id,name,storage_name,media_type,size,sha256,created_at) VALUES(?,?,?,?,?,?,?,?)`,
			id, v.ID, "test.txt", id, "text/plain", 11, "unused-in-project-tests", model.Timestamp())
		return err
	}))
	return path
}

func countRows(t *testing.T, st *store.Store, table string, want int) {
	t.Helper()
	check(t, st.Read(ctx, func(tx *sql.Tx) error {
		var got int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM `+table).Scan(&got); err != nil {
			return err
		}
		if got != want {
			t.Errorf("%s count = %d, want %d", table, got, want)
		}
		return nil
	}))
}

// Permanently deleting items or clearing Trash cascades only to trashed projects, tasks, and attachment metadata.
func TestPermanentDeleteCascadeAndClearTrash(t *testing.T) {
	s, st := setup(t)
	p := project(t, s, "deleted project")
	a := task(t, s, p, "live child")
	b := task(t, s, p, "deleted child")
	physical := addAttachment(t, st, a)
	addAttachment(t, st, b)
	isError(t, s.PermanentlyDelete(ctx, projects.TrashProject, p.ID), model.ErrConflict)
	isError(t, s.PermanentlyDelete(ctx, projects.TrashTask, a.ID), model.ErrConflict)
	check(t, s.DeleteTask(ctx, b.ID))
	check(t, s.DeleteProject(ctx, p.ID))
	check(t, s.PermanentlyDelete(ctx, projects.TrashProject, p.ID))
	countRows(t, st, "projects", 0)
	countRows(t, st, "tasks", 0)
	countRows(t, st, "attachments", 0)
	if _, err := os.Stat(physical); err != nil {
		t.Fatalf("project purge removed managed file outside attachment GC: %v", err)
	}
	live := project(t, s, "live")
	keep := task(t, s, live, "keep")
	keepPath := addAttachment(t, st, keep)
	remove := task(t, s, live, "remove individually")
	addAttachment(t, st, remove)
	check(t, s.DeleteTask(ctx, remove.ID))
	check(t, s.PermanentlyDelete(ctx, projects.TrashTask, remove.ID))
	board(t, s, live, model.Planned, keep)
	countRows(t, st, "attachments", 1)
	remove = task(t, s, live, "clear individually")
	addAttachment(t, st, remove)
	check(t, s.DeleteTask(ctx, remove.ID))
	archived := project(t, s, "archived")
	_, err := s.ArchiveProject(ctx, archived.ID, true)
	check(t, err)
	archivedTask := task(t, s, archived, "archived task")
	removedProject := project(t, s, "clear project")
	child := task(t, s, removedProject, "clear child")
	addAttachment(t, st, child)
	check(t, s.DeleteProject(ctx, removedProject.ID))
	check(t, s.ClearTrash(ctx))
	countRows(t, st, "projects", 2)
	countRows(t, st, "tasks", 2)
	countRows(t, st, "attachments", 1)
	board(t, s, live, model.Planned, keep)
	board(t, s, archived, model.Planned, archivedTask)
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatal(err)
	}
	trash, err := s.ListTrash(ctx)
	check(t, err)
	if len(trash) != 0 {
		t.Fatalf("trash after clear: %+v", trash)
	}
	check(t, s.ClearTrash(ctx))
}

// ReorderTasks runs in a single transaction so a mid-flight database error rolls back all position changes.
func TestReorderRollsBackAllChangesOnDatabaseFailure(t *testing.T) {
	s, st := setup(t)
	p := project(t, s, "P")
	a, b, c := task(t, s, p, "A"), task(t, s, p, "B"), task(t, s, p, "C")
	check(t, st.Write(ctx, func(tx *sql.Tx) error {
		// Reverse order updates C first, then A. Abort the second write to
		// prove the first write is rolled back by Store.Write.
		_, err := tx.ExecContext(ctx, `CREATE TRIGGER reject_reorder BEFORE UPDATE OF position ON tasks
			WHEN NEW.title='A' AND NEW.position<>OLD.position BEGIN SELECT RAISE(ABORT,'injected failure'); END`)
		return err
	}))
	if err := s.ReorderTasks(ctx, p.ID, model.Planned, []string{c.ID, b.ID, a.ID}); err == nil {
		t.Fatal("injected failure did not fail reorder")
	}
	board(t, s, p, model.Planned, a, b, c)
	for _, original := range []model.Task{a, b, c} {
		got, err := s.GetTask(ctx, original.ID)
		check(t, err)
		if !reflect.DeepEqual(got, original) {
			t.Fatalf("transaction leaked mutation: %+v", got)
		}
	}
}

// Regression: RFC3339Nano timestamps with variable fractional digits must sort chronologically in project and Trash queries.
func TestRFC3339NanoChronologicalOrdering(t *testing.T) {
	s, st := setup(t)
	earlier := project(t, s, "earlier")
	later := project(t, s, "later")
	check(t, st.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE projects SET created_at=CASE id WHEN ? THEN ? ELSE ? END`,
			earlier.ID, "2026-11-09T00:00:00.1Z", "2026-11-09T00:00:00.11Z")
		return err
	}))
	list, err := s.ListProjects(ctx, projects.ProjectFilter{})
	check(t, err)
	if len(list) != 2 || list[0].ID != earlier.ID || list[1].ID != later.ID {
		t.Fatalf("projects not chronologically sorted: %+v", list)
	}
	check(t, s.DeleteProject(ctx, earlier.ID))
	check(t, s.DeleteProject(ctx, later.ID))
	check(t, st.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE projects SET deleted_at=created_at`)
		return err
	}))
	trash, err := s.ListTrash(ctx)
	check(t, err)
	if len(trash) != 2 || trash[0].ID != later.ID || trash[1].ID != earlier.ID {
		t.Fatalf("trash not chronologically sorted: %+v", trash)
	}
}

// Concurrent task creations and moves across service instances serialize through Store and maintain dense positions.
func TestConcurrentServicesKeepAllTasksAndDensePositions(t *testing.T) {
	s, st := setup(t)
	p := project(t, s, "P")
	const n = 16
	var wg sync.WaitGroup
	created := make(chan model.Task, n)
	errs := make(chan error, 2*n)
	for range n {
		wg.Go(func() {
			// Multiple services must serialize through the same Store lock.
			v, err := projects.New(st).CreateTask(ctx, p.ID, projects.TaskInput{Title: "concurrent"})
			if err != nil {
				errs <- err
				return
			}
			created <- v
		})
	}
	wg.Wait()
	close(created)
	var all []model.Task
	for v := range created {
		all = append(all, v)
		wg.Go(func() {
			_, err := projects.New(st).MoveTask(ctx, v.ID, model.Done, 0)
			if err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if len(all) != n {
		t.Fatalf("created %d tasks, want %d", len(all), n)
	}
	done, err := s.ListTasks(ctx, p.ID, projects.TaskFilter{Status: model.Done})
	check(t, err)
	if len(done) != n {
		t.Fatalf("moved %d tasks, want %d", len(done), n)
	}
	seen := make(map[string]bool)
	for i, v := range done {
		if v.Position != i || v.CompletedAt == nil || seen[v.ID] {
			t.Fatalf("lost order, identity or completion time: %+v", v)
		}
		seen[v.ID] = true
	}
	board(t, s, p, model.Planned)
}

// UpdateTask changing Status updates content fields, appends to the destination column, and keeps dense positions in both columns.
func TestUpdateTaskStatusChangeReindexesBothColumns(t *testing.T) {
	s, _ := setup(t)
	p := project(t, s, "P")
	a := task(t, s, p, "A")
	b := task(t, s, p, "B")
	c, err := s.CreateTask(ctx, p.ID, projects.TaskInput{Title: "C", Status: model.Done})
	check(t, err)

	updated, err := s.UpdateTask(ctx, a.ID, projects.TaskInput{
		Title:       "A-done",
		Description: "moved via update",
		Status:      model.Done,
		Priority:    model.Urgent,
		Tags:        []string{"release"},
		DueDate:     "2026-12-31",
	})
	check(t, err)
	if updated.Title != "A-done" || updated.Description != "moved via update" || updated.Priority != model.Urgent ||
		updated.Status != model.Done || updated.Position != 1 || updated.CompletedAt == nil || updated.DueDate != "2026-12-31" {
		t.Fatalf("unexpected updated task: %+v", updated)
	}
	board(t, s, p, model.Planned, b)
	board(t, s, p, model.Done, c, updated)

	reopened, err := s.UpdateTask(ctx, updated.ID, projects.TaskInput{
		Title:    "A-reopened",
		Status:   model.InProgress,
		Priority: model.Medium,
	})
	check(t, err)
	if reopened.CompletedAt != nil || reopened.Status != model.InProgress || reopened.Position != 0 {
		t.Fatalf("unexpected reopened task: %+v", reopened)
	}
	board(t, s, p, model.Done, c)
	board(t, s, p, model.InProgress, reopened)
}

// CountTasks reports live tasks per project and status, skipping trashed tasks and projects without tasks.
func TestCountTasksGroupsLiveTasksByProjectAndStatus(t *testing.T) {
	s, _ := setup(t)
	p := project(t, s, "P")
	empty := project(t, s, "Empty")
	for _, status := range []model.Status{model.Planned, model.Planned, model.Done} {
		_, err := s.CreateTask(ctx, p.ID, projects.TaskInput{Title: "task", Status: status})
		check(t, err)
	}
	trashed, err := s.CreateTask(ctx, p.ID, projects.TaskInput{Title: "trashed", Status: model.InProgress})
	check(t, err)
	check(t, s.DeleteTask(ctx, trashed.ID))
	got, err := s.CountTasks(ctx)
	check(t, err)
	want := map[string]map[model.Status]int{p.ID: {model.Planned: 2, model.Done: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CountTasks = %v, want %v (project %s has no tasks)", got, want, empty.ID)
	}
}
