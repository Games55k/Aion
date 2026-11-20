package frontend

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aion/backend"
	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
)

func clickWriteControl(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatalf("click %q: %v", label, err)
	}
}

func replaceWriteInput(t *testing.T, tt *ui.Tester, label, value string) {
	t.Helper()
	clickWriteControl(t, tt, label)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(value)
}

func expectWriteGuards(t *testing.T, w *Workbench) {
	t.Helper()
	if w.PrepareClose() {
		t.Fatal("closed while a write was pending")
	}
	w.RequestRestoreBackup("pending-restore.zip")
	w.TriggerRestoreBackup()
	if w.IsRestoring() || !w.confirmRestoreOpen {
		t.Fatal("started restore while a write was pending")
	}
	w.confirmRestoreOpen, w.pendingRestoreZip = false, ""
}

func openWriteTestWorkbench(t *testing.T) (*backend.App, *Workbench, *ui.Tester, chan func(), func(string) func()) {
	t.Helper()
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var arm atomic.Bool
	queue := make(chan func(), 64)
	app, w, tt := openTestWorkbenchAt(t, t.TempDir(), func() time.Time {
		if arm.Swap(false) {
			close(held)
			<-release
		}
		return time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	}, Host{Dispatch: func(f func()) { queue <- f }})
	hold := func(projectID string) func() {
		// CreateTask samples the clock within its write transaction.
		arm.Store(true)
		go func() {
			_, err := app.Projects.CreateTask(context.Background(), projectID, projects.TaskInput{Title: "Hold Store"})
			done <- err
		}()
		<-held
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
	return app, w, tt, queue, hold
}

// Regression: board moves, reorders, and priority changes waited for Store in UI callbacks.
func TestBoardWritesRemainResponsiveUnderStoreLock(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		fail         bool
	}{
		{"move", "移至 已完成", false},
		{"reorder", "在列内下移", false},
		{"priority", "紧急", false},
		{"failed move", "移至 已完成", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, w, tt, queue, ps, task := asyncTestWorkbench(t)
			ctx := context.Background()
			w.ClearSelectedTask()
			if _, err := w.app.Projects.CreateTask(ctx, ps[0].ID, projects.TaskInput{Title: "Neighbor"}); err != nil {
				t.Fatal(err)
			}
			w.ReloadBoardTasks()
			tt.Frame()
			if tc.name == "priority" {
				// Keep the old board visible while updating a separate field in storage.
				if _, err := w.app.Projects.UpdateTask(ctx, task.ID, projects.TaskInput{
					Title: task.Title, Description: "Latest description", Status: task.Status, Priority: task.Priority,
				}); err != nil {
					t.Fatal(err)
				}
			}
			if tc.fail {
				if err := s.Exclusive(ctx, func(db *sql.DB) error {
					_, err := db.Exec(`CREATE TRIGGER reject_move BEFORE UPDATE OF status ON tasks
						BEGIN SELECT RAISE(ABORT, 'move blocked'); END`)
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}

			unlock := holdTestStore(t, s)
			if tc.name == "priority" {
				clickWriteControl(t, tt, "切换优先级 Original")
			} else if err := tt.RightClick("任务卡片 Original"); err != nil {
				t.Fatal(err)
			}
			tt.Advance(time.Second)
			clickWriteControl(t, tt, tc.action)
			tt.Advance(time.Second)
			replaceWriteInput(t, tt, "搜索任务", "can still type")
			if w.searchQuery != "can still type" {
				t.Fatalf("board search = %q", w.searchQuery)
			}
			clickWriteControl(t, tt, "Last project")
			expectWriteGuards(t, w)
			unlock()
			w.WaitForTasks()
			expectWriteGuards(t, w)
			drainTestUI(w, queue)
			if !w.PrepareClose() || w.activeProjectID != ps[2].ID || len(w.boardTasks) != 1 || w.boardTasks[0].Title != "Last board task" {
				t.Fatalf("write affected newer navigation: selected=%q board=%+v", w.activeProjectID, w.boardTasks)
			}
			persisted, err := w.app.Projects.GetTask(ctx, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			switch tc.name {
			case "move":
				if persisted.Status != model.Done {
					t.Fatalf("move = %s", persisted.Status)
				}
			case "reorder":
				if persisted.Position != 1 {
					t.Fatalf("reorder position = %d", persisted.Position)
				}
			case "priority":
				if persisted.Priority != model.Urgent || persisted.Status != task.Status || persisted.Description != "Latest description" {
					t.Fatalf("priority update = %+v", persisted)
				}
			case "failed move":
				tt.Frame()
				if persisted.Status != task.Status || !tt.HasText("移动任务失败") {
					t.Fatalf("failed move was not reported or changed the task: %+v", persisted)
				}
			}
		})
	}
}

// Regression: quick add blocked typing on Store, even when the user only wanted to prepare the next title.
func TestQuickCreateKeepsNewerTitleAndIgnoresDuplicateSubmit(t *testing.T) {
	s, w, tt, queue, ps, _ := asyncTestWorkbench(t)
	w.ClearSelectedTask()
	tt.Frame()
	unlock := holdTestStore(t, s)
	replaceWriteInput(t, tt, "快速添加计划任务", "First submission")
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEnter)
	replaceWriteInput(t, tt, "快速添加计划任务", "Next draft")
	clickWriteControl(t, tt, "Last project")
	unlock()
	drainTestUI(w, queue)
	tasks, err := w.app.Projects.ListTasks(context.Background(), ps[0].ID, projects.TaskFilter{Query: "First submission"})
	if err != nil || len(tasks) != 1 {
		t.Fatalf("quick add = %+v, %v", tasks, err)
	}
	if w.activeProjectID != ps[2].ID || w.quickTaskTitles[model.Planned] != "Next draft" {
		t.Fatalf("quick add replaced newer input or page: %+v, %q", w.quickTaskTitles, w.activeProjectID)
	}
}

// Regression: creating projects, tasks, and job applications blocked form input while Store was held.
func TestCreateFormsKeepNewerInputUnderStoreLock(t *testing.T) {
	for _, tc := range []struct{ kind, field, submit string }{
		{"project", "项目名称输入框", "创建项目"},
		{"task", "新建任务标题输入框", "创建任务"},
		{"job", "创建公司名称", "确认创建"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			s, w, tt, queue, ps, _ := asyncTestWorkbench(t)
			ctx := context.Background()
			w.ClearSelectedTask()
			switch tc.kind {
			case "project":
				w.OpenCreateProjectModal()
			case "task":
				w.OpenCreateTaskModal(model.Planned)
			case "job":
				w.SwitchModule("job_applications")
				w.OpenJobCreateModal()
			}
			drainTestUI(w, queue)
			tt.Frame()
			if tc.kind == "job" {
				replaceWriteInput(t, tt, "创建岗位名称", "Role")
			}
			unlock := holdTestStore(t, s)
			replaceWriteInput(t, tt, tc.field, "First record")
			clickWriteControl(t, tt, tc.submit)
			_ = tt.Click(tc.submit)
			replaceWriteInput(t, tt, tc.field, "Next draft")
			expectWriteGuards(t, w)
			unlock()
			drainTestUI(w, queue)
			switch tc.kind {
			case "project":
				list, err := w.app.Projects.ListProjects(ctx, projects.ProjectFilter{})
				if err != nil || len(list) != len(ps)+1 || !slices.ContainsFunc(list, func(p model.Project) bool { return p.Name == "First record" }) {
					t.Fatalf("project create = %+v, %v", list, err)
				}
				if !w.projectModalOpen || w.projectModalName != "Next draft" || w.activeProjectID != ps[0].ID {
					t.Fatal("project create consumed newer input or navigated")
				}
			case "task":
				list, err := w.app.Projects.ListTasks(ctx, ps[0].ID, projects.TaskFilter{})
				if err != nil || len(list) != 2 || !slices.ContainsFunc(list, func(task model.Task) bool { return task.Title == "First record" }) {
					t.Fatalf("task create = %+v, %v", list, err)
				}
				if !w.taskModalOpen || w.taskModalTitle != "Next draft" {
					t.Fatal("task create consumed newer input")
				}
			case "job":
				list, err := w.app.JobApplications.List(ctx)
				if err != nil || len(list) != 1 || list[0].CompanyName != "First record" {
					t.Fatalf("job create = %+v, %v", list, err)
				}
				if !w.jobCreateOpen || w.jobCreateCompany != "Next draft" {
					t.Fatal("job create consumed newer input")
				}
			}
		})
	}
}

// A failed background create keeps the form available and a retry creates exactly one record.
func TestFailedBackgroundCreateKeepsDraftForRetry(t *testing.T) {
	s, w, tt, queue, _, _ := asyncTestWorkbench(t)
	ctx := context.Background()
	w.ClearSelectedTask()
	w.SwitchModule("job_applications")
	drainTestUI(w, queue)
	w.OpenJobCreateModal()
	tt.Frame()
	if err := s.Exclusive(ctx, func(db *sql.DB) error {
		_, err := db.Exec(`CREATE TRIGGER reject_job BEFORE INSERT ON job_applications
			BEGIN SELECT RAISE(ABORT, 'create blocked'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	replaceWriteInput(t, tt, "创建公司名称", "Retry company")
	replaceWriteInput(t, tt, "创建岗位名称", "Role")
	unlock := holdTestStore(t, s)
	clickWriteControl(t, tt, "确认创建")
	unlock()
	drainTestUI(w, queue)
	if !w.jobCreateOpen || w.jobCreateCompany != "Retry company" || !strings.Contains(w.jobCreateErr, "create blocked") {
		t.Fatalf("failed create lost its form or error: %q, %q", w.jobCreateCompany, w.jobCreateErr)
	}
	if err := s.Exclusive(ctx, func(db *sql.DB) error {
		_, err := db.Exec("DROP TRIGGER reject_job")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	clickWriteControl(t, tt, "确认创建")
	drainTestUI(w, queue)
	list, err := w.app.JobApplications.List(ctx)
	if err != nil || len(list) != 1 || list[0].CompanyName != "Retry company" || w.jobCreateOpen {
		t.Fatalf("retry = %+v, %v", list, err)
	}
}

// Opening a task during a move loads the committed task rather than exposing its old status for editing.
func TestTaskSelectionDuringBoardWriteUsesCommittedState(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	w.ClearSelectedTask()
	tt.Frame()
	unlock := holdTestStore(t, s)
	if err := tt.RightClick("任务卡片 Original"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	clickWriteControl(t, tt, "移至 已完成")
	tt.Advance(time.Second)
	clickWriteControl(t, tt, "任务卡片 Original")
	if w.detailModalOpen() {
		t.Fatal("opened a task before its pending move completed")
	}
	unlock()
	drainTestUI(w, queue)
	if !w.detailModalOpen() || w.detailTask.ID != task.ID || w.detailTask.Status != model.Done {
		t.Fatalf("deferred selection did not load the moved task: %+v", w.detailTask)
	}
}

// Regression: recovering an item blocked navigation, and a late parent error could open a modal on a different page.
func TestTrashRestoreRemainsResponsiveUnderStoreLock(t *testing.T) {
	for _, parentDeleted := range []bool{false, true} {
		name := "restored"
		if parentDeleted {
			name = "deleted parent"
		}
		t.Run(name, func(t *testing.T) {
			app, w, tt, queue, hold := openWriteTestWorkbench(t)
			ctx := context.Background()
			p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Trashed task project"})
			if err != nil {
				t.Fatal(err)
			}
			next, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Last project"})
			if err != nil {
				t.Fatal(err)
			}
			task, err := app.Projects.CreateTask(ctx, p.ID, projects.TaskInput{Title: "Original"})
			if err != nil {
				t.Fatal(err)
			}
			if err := app.Projects.DeleteTask(ctx, task.ID); err != nil {
				t.Fatal(err)
			}
			if parentDeleted {
				if err := app.Projects.DeleteProject(ctx, p.ID); err != nil {
					t.Fatal(err)
				}
			}
			w.ReloadProjects()
			w.NavigateTo(NavTrash)
			drainTestUI(w, queue)
			tt.Frame()
			unlock := hold(next.ID)
			clickWriteControl(t, tt, "恢复 Original")
			clickWriteControl(t, tt, "Last project")
			unlock()
			drainTestUI(w, queue)
			if w.activeProjectID != next.ID || w.trashParentErrModalOpen {
				t.Fatal("restore acknowledgement interrupted newer navigation")
			}
			items, err := app.ListTrash(ctx)
			stillDeleted := slices.ContainsFunc(items, func(item model.TrashItem) bool { return item.ID == task.ID })
			if err != nil || stillDeleted != parentDeleted {
				t.Fatalf("unexpected trash after restore: %+v, %v", items, err)
			}
		})
	}
}

// Regression: saving themes or backup policies blocked input and could reload an older Settings draft.
func TestSettingsWritesCoalesceAndKeepUnsubmittedInput(t *testing.T) {
	ctx := context.Background()
	app, w, tt, queue, hold := openWriteTestWorkbench(t)
	p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Lock owner"})
	if err != nil {
		t.Fatal(err)
	}
	w.NavigateTo(NavSettings)
	drainTestUI(w, queue)
	tt.Frame()
	invalid, second, draft := app.DataDir(), t.TempDir(), t.TempDir()

	unlock := hold(p.ID)
	clickWriteControl(t, tt, "浅色")
	clickWriteControl(t, tt, "深色")
	for _, directory := range []string{invalid, second} {
		replaceWriteInput(t, tt, "自动备份目录输入框", directory)
		tt.Key(0, ui.KeyEnter)
	}
	replaceWriteInput(t, tt, "自动备份目录输入框", draft)
	expectWriteGuards(t, w)
	unlock()
	drainTestUI(w, queue)
	if theme, err := app.Settings.Theme(ctx); err != nil || theme != ThemeDark {
		t.Fatalf("latest theme = %q, %v", theme, err)
	}
	if policy, err := app.Backups.GetPolicy(ctx); err != nil || policy.Directory != second {
		t.Fatalf("latest submitted backup policy = %+v, %v", policy, err)
	}
	if strings.TrimSpace(w.backupDirInput) != draft || !w.PrepareClose() {
		t.Fatalf("settings completion replaced the unsubmitted input: %q", w.backupDirInput)
	}
	tt.Frame()
	if !tt.HasText("保存自动备份策略失败") {
		t.Fatal("expected invalid first policy to report its failure")
	}
}

// Auto-update check preference writes coalesce under a held Store lock, guard close/restore while pending, and sync the latest policy to Host once committed.
func TestUpdateCheckPolicyWritesCoalesceUnderStoreLock(t *testing.T) {
	ctx := context.Background()
	app, w, tt, queue, hold := openWriteTestWorkbench(t)
	var synced []backend.UpdateCheckPolicy
	host := w.host
	host.SetUpdateCheckPolicy = func(pol backend.UpdateCheckPolicy) {
		synced = append(synced, pol)
	}
	w.SetHost(host)

	p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Update policy lock"})
	if err != nil {
		t.Fatal(err)
	}
	w.NavigateTo(NavSettings)
	drainTestUI(w, queue)
	tt.Frame()
	tt.Scroll(500, 300, 0, 800)
	tt.Frame()

	unlock := hold(p.ID)
	// Toggle off, then back on, then change interval to 6h while Store is held.
	clickWriteControl(t, tt, "自动检查更新开关")
	clickWriteControl(t, tt, "自动检查更新开关")
	clickWriteControl(t, tt, "自动检查频率选择")
	clickWriteControl(t, tt, "每 6 小时")
	expectWriteGuards(t, w)
	unlock()
	drainTestUI(w, queue)

	pol, err := app.Settings.UpdateCheckPolicy(ctx)
	if err != nil || !pol.Enabled || pol.Interval != 6*time.Hour {
		t.Fatalf("expected coalesced 6h enabled policy, got %+v, err=%v", pol, err)
	}
	if len(synced) == 0 {
		t.Fatal("expected Host.SetUpdateCheckPolicy invoked after commit")
	}
	if last := synced[len(synced)-1]; !last.Enabled || last.Interval != 6*time.Hour {
		t.Fatalf("expected final synced policy 6h enabled, got %+v", synced)
	}
	if !w.PrepareClose() {
		t.Fatal("expected PrepareClose permitted after update check policy writes drained")
	}
}
