package frontend

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"aion/backend/attachments"
	"aion/backend/backup"
	"aion/backend/model"
	"aion/backend/projects"
	"aion/backend/store"
	"github.com/egoist/mygo/ui"
)

func asyncTestWorkbench(t *testing.T) (*store.Store, *Workbench, *ui.Tester, chan func(), []model.Project, model.Task) {
	t.Helper()
	app, w, tt := openTestWorkbench(t, t.TempDir(), Host{})
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	app.Projects, app.Attachments, app.Backups = projects.New(s), attachments.New(s), backup.New(s)
	var ps []model.Project
	for _, name := range []string{"First project", "Second project", "Last project"} {
		p, err := app.Projects.CreateProject(context.Background(), projects.ProjectInput{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		ps = append(ps, p)
	}
	task, err := app.Projects.CreateTask(context.Background(), ps[0].ID, projects.TaskInput{Title: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Projects.CreateTask(context.Background(), ps[2].ID, projects.TaskInput{Title: "Last board task"}); err != nil {
		t.Fatal(err)
	}
	w.ReloadProjects()
	w.SelectProject(ps[0].ID)
	w.SelectTask(task.ID)
	tt.Frame()
	queue := make(chan func(), 64)
	w.host.Dispatch = func(fn func()) { queue <- fn }
	return s, w, tt, queue, ps, task
}

func holdTestStore(t *testing.T, s *store.Store) func() {
	t.Helper()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.Exclusive(context.Background(), func(*sql.DB) error { close(entered); <-release; return nil })
	}()
	<-entered
	var once sync.Once
	unlock := func() {
		once.Do(func() {
			close(release)
			if err := <-done; err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(unlock)
	return unlock
}

func drainTestUI(w *Workbench, queue chan func()) {
	for {
		w.WaitForTasks()
		n := len(queue)
		if n == 0 {
			return
		}
		for i := 0; i < n; i++ {
			(<-queue)()
		}
	}
}

func typeTaskTitle(t *testing.T, tt *ui.Tester, title string) {
	t.Helper()
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(title)
}

// Regression: typing and leaving a draft used to block under Store; only the latest draft and navigation intent may survive.
func TestAsyncDraftAndNavigationUnderStoreLock(t *testing.T) {
	s, w, tt, queue, ps, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "First write")
	typeTaskTitle(t, tt, "Latest draft")
	if w.editTitle != "Latest draft" || !w.saving || !w.dirty {
		t.Fatalf("input did not remain local: %+v", w.pendingSave)
	}
	if w.PrepareClose() {
		t.Fatal("closed with an unacknowledged write")
	}
	w.SelectProject(ps[1].ID)
	w.SelectProject(ps[2].ID)
	if w.activeProjectID != ps[0].ID {
		t.Fatal("discarded draft before persistence")
	}
	w.RequestRestoreBackup("unused.zip")
	w.TriggerRestoreBackup()
	if w.IsRestoring() {
		t.Fatal("restore raced a pending save")
	}
	tt.Frame()
	unlock()
	w.WaitForTasks()
	if w.PrepareClose() {
		t.Fatal("closed before dispatched save acknowledgement")
	}
	drainTestUI(w, queue)
	persisted, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || persisted.Title != "Latest draft" {
		t.Fatalf("lost/out-of-order save: %+v, %v", persisted, err)
	}
	if w.activeProjectID != ps[2].ID || len(w.boardTasks) != 1 || w.boardTasks[0].Title != "Last board task" {
		t.Fatalf("wrong final navigation: %s, %+v", w.activeProjectID, w.boardTasks)
	}
	if w.saving || w.dirty || !w.PrepareClose() {
		t.Fatal("completed save still blocks close")
	}
}

// Regression: an older successful write must not erase newer invalid input or navigate away from that draft.
func TestAsyncSavePreservesNewerInvalidDraft(t *testing.T) {
	s, w, tt, queue, ps, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "Committed")
	typeTaskTitle(t, tt, "   ")
	w.SelectProject(ps[1].ID)
	unlock()
	drainTestUI(w, queue)
	if w.editTitle != "   " || !w.dirty || w.saveState != "validation_error" || w.activeProjectID != ps[0].ID {
		t.Fatal("older save replaced invalid draft")
	}
	persisted, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || persisted.Title != "Committed" {
		t.Fatalf("wrong baseline: %+v %v", persisted, err)
	}
	if w.PrepareClose() {
		t.Fatal("closed with invalid input")
	}
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if w.editTitle != "Committed" || w.dirty || !w.PrepareClose() {
		t.Fatal("discard failed to restore acknowledged baseline")
	}
}

// Failed background persistence keeps the draft and cancels deferred navigation; retry writes the same draft safely.
func TestAsyncSaveFailureAndRetry(t *testing.T) {
	s, w, tt, queue, ps, task := asyncTestWorkbench(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	typeTaskTitle(t, tt, "Retry me")
	w.SelectProject(ps[1].ID)
	drainTestUI(w, queue)
	if w.saveState != "error" || !w.dirty || w.editTitle != "Retry me" || w.activeProjectID != ps[0].ID {
		t.Fatal("failed save lost the draft")
	}
	reopened, err := store.Open(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	w.app.Projects = projects.New(reopened)
	tt.Frame()
	if err := tt.Click("重试保存"); err != nil {
		t.Fatal(err)
	}
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Title != "Retry me" || w.dirty || w.activeProjectID != ps[0].ID {
		t.Fatalf("retry failed or replayed canceled navigation: %+v, %v", saved, err)
	}
}

// Navigation and Settings input stay responsive while Store is held; late loads cannot undo newer navigation or form edits.
func TestAsyncNavigationRejectsStaleResults(t *testing.T) {
	s, w, tt, queue, ps, _ := asyncTestWorkbench(t)
	w.ClearSelectedTask()
	tt.Frame()
	unlock := holdTestStore(t, s)
	for _, label := range []string{"Second project", "Last project", "设置"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if err := tt.Click("自动备份目录输入框"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("/tmp/new-draft-directory")
	unlock()
	w.WaitForTasks()
	// Deliberately deliver old pages last.
	var callbacks []func()
	for len(queue) > 0 {
		callbacks = append(callbacks, <-queue)
	}
	for i := len(callbacks) - 1; i >= 0; i-- {
		callbacks[i]()
	}
	if w.navPage != NavSettings || w.activeProjectID != ps[2].ID || len(w.boardTasks) != 0 || w.backupDirInput != "/tmp/new-draft-directory" {
		t.Fatal("late load overwrote current page or draft")
	}
	if err := tt.Click("Last project"); err != nil {
		t.Fatal(err)
	}
	drainTestUI(w, queue)
	if len(w.boardTasks) != 1 || w.boardTasks[0].ProjectID != ps[2].ID {
		t.Fatalf("wrong board after loading: %+v", w.boardTasks)
	}
}

// Discard during an in-flight write drops queued edits and restores the last write that actually committed.
func TestAsyncDiscardWhileSaving(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "In flight")
	typeTaskTitle(t, tt, "Queued")
	typeTaskTitle(t, tt, "   ")
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Title != "In flight" || w.editTitle != saved.Title || w.dirty || w.saving {
		t.Fatalf("discard raced persistence: %+v %v", saved, err)
	}
}

// Closing details during a status save must still refresh the board after the deferred close invalidates old loads.
func TestAsyncStatusSaveThenCloseRefreshesBoard(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	if err := tt.Click("任务详情状态"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("已完成"); err != nil {
		t.Fatal(err)
	}
	typeTaskTitle(t, tt, "Status plus coalesced content")
	if err := tt.Click("关闭详情弹窗"); err != nil {
		t.Fatal(err)
	}
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Status != model.Done {
		t.Fatalf("status not persisted: %+v %v", saved, err)
	}
	if w.detailExpanded || len(w.TasksInColumn(model.Done)) != 1 || len(w.TasksInColumn(model.Planned)) != 0 {
		t.Fatal("closed details left board in old column")
	}
}
