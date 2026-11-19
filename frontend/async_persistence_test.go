package frontend

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"aion/backend/attachments"
	"aion/backend/backup"
	"aion/backend/jobapplications"
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
	app.Projects, app.JobApplications, app.Attachments, app.Backups = projects.New(s), jobapplications.New(s), attachments.New(s), backup.New(s)
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

// Discarding while a save is in flight must rebase later edits on that write's result, keeping its title.
func TestAsyncDiscardThenEditKeepsInFlightSaveAsBaseline(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "飞行中的新标题")
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if w.editTitle != "" || w.saveState != "validation_error" {
		t.Fatalf("title not cleared before discard: %q %q", w.editTitle, w.saveState)
	}
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("任务详情描述输入框"); err != nil {
		t.Fatal(err)
	}
	tt.Type("落地前补充的描述")
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Title != "飞行中的新标题" || saved.Description != "落地前补充的描述" {
		t.Fatalf("post-discard edit reverted the in-flight save: %+v %v", saved, err)
	}
	if w.editTitle != saved.Title || w.editDescription != saved.Description || w.dirty || w.saving {
		t.Fatalf("draft diverged from persisted task: title=%q desc=%q dirty=%v saving=%v", w.editTitle, w.editDescription, w.dirty, w.saving)
	}
}

// A failed in-flight write must not leak its content through the queued post-discard save: untouched
// fields revert to the committed values while edits made after the discard still persist.
func TestAsyncDiscardThenEditExcludesFailedSaveFromFollowUpWrite(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	if err := s.Exclusive(context.Background(), func(db *sql.DB) error {
		_, err := db.Exec(`CREATE TRIGGER tasks_reject_failed_title BEFORE UPDATE ON tasks
			WHEN NEW.title = '注定失败的新标题'
			BEGIN SELECT RAISE(ABORT, 'first write must fail'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "注定失败的新标题")
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if w.saveState != "saving" {
		t.Fatalf("discard during save must keep the saving state: %q", w.saveState)
	}
	if err := tt.Click("任务详情描述输入框"); err != nil {
		t.Fatal(err)
	}
	tt.Type("落地前补充的描述")
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Title != "Original" || saved.Description != "落地前补充的描述" {
		t.Fatalf("follow-up save leaked the failed write or lost new input: %+v %v", saved, err)
	}
	if w.editTitle != "Original" || w.editDescription != "落地前补充的描述" || w.dirty || w.saving || w.saveState != "saved" {
		t.Fatalf("draft diverged after rebase: title=%q desc=%q dirty=%v saving=%v state=%q",
			w.editTitle, w.editDescription, w.dirty, w.saving, w.saveState)
	}
}

// An invalid draft after the discard must also rebase: untouched fields must not keep the
// failed write's values while the user's invalid input and validation state survive.
func TestAsyncDiscardThenInvalidDraftExcludesFailedSave(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	if err := s.Exclusive(context.Background(), func(db *sql.DB) error {
		_, err := db.Exec(`CREATE TRIGGER tasks_reject_failed_desc BEFORE UPDATE ON tasks
			WHEN NEW.description = '注定失败的描述'
			BEGIN SELECT RAISE(ABORT, 'first write must fail'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	unlock := holdTestStore(t, s)
	if err := tt.Click("任务详情描述输入框"); err != nil {
		t.Fatal(err)
	}
	tt.Type("注定失败的描述")
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	unlock()
	drainTestUI(w, queue)
	if w.editTitle != "" || w.editDescription != "" || w.saveState != "validation_error" {
		t.Fatalf("invalid draft kept the failed baseline: title=%q desc=%q state=%q",
			w.editTitle, w.editDescription, w.saveState)
	}
	tt.Type("补齐的新标题")
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Title != "补齐的新标题" || saved.Description != "" {
		t.Fatalf("follow-up save leaked the failed write: %+v %v", saved, err)
	}
}

// Tag edits made after the discard must merge additively: a failed tag from the rolled-back
// write must not ride along when the user only added a new tag afterwards.
func TestAsyncDiscardThenTagEditExcludesFailedTags(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	if err := s.Exclusive(context.Background(), func(db *sql.DB) error {
		_, err := db.Exec(`CREATE TRIGGER tasks_reject_failed_tag BEFORE UPDATE ON tasks
			WHEN NEW.tags LIKE '%"failed-tag"%'
			BEGIN SELECT RAISE(ABORT, 'first write must fail'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	addTag := func(tag string) {
		t.Helper()
		if err := tt.Click("添加标签输入框"); err != nil {
			t.Fatal(err)
		}
		tt.Type(tag)
		tt.Key(0, ui.KeyEnter)
	}
	unlock := holdTestStore(t, s)
	addTag("failed-tag")
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	addTag("keep-tag")
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || len(saved.Tags) != 1 || saved.Tags[0] != "keep-tag" {
		t.Fatalf("failed tag leaked or kept tag lost: %+v %v", saved, err)
	}
	if len(w.editTags) != 1 || w.editTags[0] != "keep-tag" || w.dirty || w.saving {
		t.Fatalf("draft diverged after tag rebase: %+v dirty=%v saving=%v", w.editTags, w.dirty, w.saving)
	}
}

// Text typed into the tag input after 放弃修改 must survive the in-flight save's completion callback.
func TestAsyncDiscardKeepsTagDraftTypedAfterDiscard(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "飞行中的新标题")
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if w.newTagInput != "" {
		t.Fatalf("discard left a tag draft: %q", w.newTagInput)
	}
	if err := tt.Click("添加标签输入框"); err != nil {
		t.Fatal(err)
	}
	tt.Type("未提交的新标签")
	unlock()
	drainTestUI(w, queue)
	if w.newTagInput != "未提交的新标签" {
		t.Fatalf("save callback wiped the tag draft typed after discard: %q", w.newTagInput)
	}
	saved, err := w.app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || saved.Title != "飞行中的新标题" {
		t.Fatalf("in-flight save lost: %+v %v", saved, err)
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

func asyncJobTestWorkbench(t *testing.T) (*store.Store, *Workbench, *ui.Tester, chan func(), model.JobApplication, model.JobApplication) {
	t.Helper()
	s, w, tt, queue, _, _ := asyncTestWorkbench(t)
	w.ClearSelectedTask()
	ctx := context.Background()
	first, err := w.app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "初始公司A",
		PositionName:   "后端工程师",
		Stage:          model.JobStageApplied,
		ApplicationURL: "https://example.com/a",
		Notes:          "初始备注",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "初始公司B",
		PositionName: "架构师",
		Stage:        model.JobStageInterview1,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SwitchModule("job_applications")
	drainTestUI(w, queue)
	w.SelectJobApplication(first.ID)
	drainTestUI(w, queue)
	tt.Frame()
	return s, w, tt, queue, first, second
}

func typeJobDetailField(t *testing.T, tt *ui.Tester, label, value string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatalf("click %q: %v", label, err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(value)
	tt.Frame()
}

// Job detail edits coalesce to the latest draft under held Store locks while deferring module switching and detail closing until saved.
func TestJobAsyncLatestDraftCoalescingAndSwitchModuleDeferral(t *testing.T) {
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情公司名称", "第一次修改")
	typeJobDetailField(t, tt, "详情公司名称", "最终公司名称")
	if w.editJobCompany != "最终公司名称" || !w.jobSaving || !w.jobDirty {
		t.Fatalf("expected coalesced local draft while saving, company=%q saving=%v dirty=%v", w.editJobCompany, w.jobSaving, w.jobDirty)
	}
	if w.PrepareClose() {
		t.Fatal("PrepareClose must return false while job auto-save is pending")
	}

	// Switching module and closing details wait for the in-flight + coalesced save to settle.
	w.SwitchModule("projects")
	if w.activeModule != "job_applications" {
		t.Fatalf("SwitchModule jumped ahead of pending job save: module=%q", w.activeModule)
	}
	w.CloseJobDetail()
	unlock()
	w.WaitForTasks()
	if w.PrepareClose() {
		t.Fatal("PrepareClose must wait until UI dispatch drains final save confirmation")
	}
	drainTestUI(w, queue)

	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.CompanyName != "最终公司名称" {
		t.Fatalf("expected coalesced job draft saved, got %+v err=%v", saved, err)
	}
	if w.jobDetailOpen || w.jobSaving || w.jobDirty || !w.PrepareClose() {
		t.Fatalf("expected closed detail and clean PrepareClose after drain, open=%v saving=%v dirty=%v", w.jobDetailOpen, w.jobSaving, w.jobDirty)
	}
}

// Invalid job detail drafts block detail close, module switching, navigation, and app close until fixed or discarded.
func TestJobAsyncInvalidDraftBlocksCloseSwitchAndPrepareClose(t *testing.T) {
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情公司名称", "合法新公司")
	typeJobDetailField(t, tt, "详情投递链接", "ftp://invalid.example.com")
	unlock()
	drainTestUI(w, queue)

	if w.jobSaveState != "validation_error" || !w.jobDirty || w.editJobURL != "ftp://invalid.example.com" {
		t.Fatalf("expected validation error preserving newer invalid URL draft, state=%q dirty=%v url=%q", w.jobSaveState, w.jobDirty, w.editJobURL)
	}
	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.CompanyName != "合法新公司" {
		t.Fatalf("expected prior valid draft persisted before invalid edit, got %+v err=%v", saved, err)
	}

	w.CloseJobDetail()
	w.SwitchModule("projects")
	w.NavigateTo(NavSettings)
	if !w.jobDetailOpen || w.activeModule != "job_applications" || w.navPage != NavJobAll || w.PrepareClose() {
		t.Fatalf("invalid job draft failed to block close/switch/nav/PrepareClose: open=%v mod=%q page=%v", w.jobDetailOpen, w.activeModule, w.navPage)
	}

	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if w.jobDirty || w.editJobURL != "https://example.com/a" || !w.PrepareClose() {
		t.Fatalf("discard failed to revert to acknowledged state: dirty=%v url=%q", w.jobDirty, w.editJobURL)
	}
}

// Failed background saves preserve the job draft for Retry and Discard during an in-flight write reverts to the persisted baseline.
func TestJobAsyncSaveErrorRetryAndDiscardDuringFlight(t *testing.T) {
	// Part 1: failed DB write retains dirty state until 重试保存 writes cleanly after reopening store.
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	typeJobDetailField(t, tt, "详情公司名称", "失败重试公司")
	w.SwitchModule("projects")
	drainTestUI(w, queue)
	if w.jobSaveState != "error" || !w.jobDirty || w.editJobCompany != "失败重试公司" || w.activeModule != "job_applications" {
		t.Fatalf("expected save error preserving draft and canceling switch, state=%q dirty=%v company=%q mod=%q", w.jobSaveState, w.jobDirty, w.editJobCompany, w.activeModule)
	}
	reopened, err := store.Open(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	w.app.JobApplications = jobapplications.New(reopened)
	tt.Frame()
	if err := tt.Click("重试保存"); err != nil {
		t.Fatal(err)
	}
	drainTestUI(w, queue)
	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.CompanyName != "失败重试公司" || w.jobDirty || w.activeModule != "job_applications" {
		t.Fatalf("expected retry to save cleanly without replaying canceled module switch, saved=%+v err=%v dirty=%v mod=%q", saved, err, w.jobDirty, w.activeModule)
	}

	// Part 2: discarding while an in-flight write is blocked drops queued edits and reverts cleanly once store unlocks.
	s2, w2, tt2, queue2, jobA, _ := asyncJobTestWorkbench(t)
	unlock := holdTestStore(t, s2)
	typeJobDetailField(t, tt2, "详情公司名称", "在途写入")
	typeJobDetailField(t, tt2, "详情公司名称", "排队写入")
	typeJobDetailField(t, tt2, "详情公司名称", "   ")
	if err := tt2.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if !w2.discardJobAfterSave {
		t.Fatal("expected discardJobAfterSave flagged while worker still saving")
	}
	unlock()
	drainTestUI(w2, queue2)
	persistedA, err := w2.app.JobApplications.Get(context.Background(), jobA.ID)
	if err != nil || persistedA.CompanyName != "在途写入" {
		t.Fatalf("expected discarded queued draft ignored, got %+v err=%v", persistedA, err)
	}
	if w2.jobDirty || w2.editJobCompany != "在途写入" {
		t.Fatalf("expected local detail reverted to persisted baseline, dirty=%v company=%q", w2.jobDirty, w2.editJobCompany)
	}
}

// A failed in-flight job write must not leak its content through the queued post-discard save.
func TestJobAsyncDiscardThenEditExcludesFailedSaveFromFollowUpWrite(t *testing.T) {
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	if err := s.Exclusive(context.Background(), func(db *sql.DB) error {
		_, err := db.Exec(`CREATE TRIGGER jobs_reject_failed_company BEFORE UPDATE ON job_applications
			WHEN NEW.company_name = '注定失败的新公司'
			BEGIN SELECT RAISE(ABORT, 'first write must fail'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情公司名称", "注定失败的新公司")
	if err := tt.Click("详情公司名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if w.jobSaveState != "saving" {
		t.Fatalf("discard during save must keep the saving state: %q", w.jobSaveState)
	}
	typeJobDetailField(t, tt, "详情通用备注", "落地前补充的备注")
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.CompanyName != "初始公司A" || saved.Notes != "落地前补充的备注" {
		t.Fatalf("follow-up save leaked the failed write or lost new input: %+v %v", saved, err)
	}
	if w.editJobCompany != "初始公司A" || w.editJobNotes != "落地前补充的备注" || w.jobDirty || w.jobSaving || w.jobSaveState != "saved" {
		t.Fatalf("draft diverged after rebase: company=%q notes=%q dirty=%v saving=%v state=%q",
			w.editJobCompany, w.editJobNotes, w.jobDirty, w.jobSaving, w.jobSaveState)
	}
}

// A job draft that turns invalid after the discard must also rebase on the committed values.
func TestJobAsyncDiscardThenInvalidDraftExcludesFailedSave(t *testing.T) {
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	if err := s.Exclusive(context.Background(), func(db *sql.DB) error {
		_, err := db.Exec(`CREATE TRIGGER jobs_reject_failed_company2 BEFORE UPDATE ON job_applications
			WHEN NEW.company_name = '注定失败的新公司'
			BEGIN SELECT RAISE(ABORT, 'first write must fail'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情公司名称", "注定失败的新公司")
	if err := tt.Click("详情公司名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("详情岗位名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	unlock()
	drainTestUI(w, queue)
	if w.editJobCompany != "初始公司A" || w.editJobPosition != "" || w.jobSaveState != "validation_error" {
		t.Fatalf("invalid draft kept the failed baseline: company=%q position=%q state=%q",
			w.editJobCompany, w.editJobPosition, w.jobSaveState)
	}
	// The error banner pushes the position field out of the scroll viewport in
	// headless tests; drive the same draft edit the UI would submit directly.
	w.editJobPosition = "补齐的新岗位"
	w.TouchJobDetailField()
	drainTestUI(w, queue)
	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.CompanyName != "初始公司A" || saved.PositionName != "补齐的新岗位" {
		t.Fatalf("follow-up save leaked the failed write: %+v %v", saved, err)
	}
}

// A closure reason queued by the rebase must stay pending until a write carrying it is
// acknowledged; later note-only saves must keep retrying it instead of silently dropping it.
func TestJobAsyncRebaseKeepsPendingClosureReason(t *testing.T) {
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	if err := s.Exclusive(context.Background(), func(db *sql.DB) error {
		_, err := db.Exec(`UPDATE job_applications SET lifecycle='closed', outcome='other',
			closure_reason='旧结束原因', closed_at='2026-01-01T00:00:00Z' WHERE id = ?`, job1.ID)
		if err != nil {
			return err
		}
		_, err = db.Exec(`CREATE TRIGGER jobs_reject_reason_change BEFORE UPDATE ON job_applications
			WHEN NEW.closure_reason != '旧结束原因'
			BEGIN SELECT RAISE(ABORT, 'reason write must fail'); END;`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	w.CloseJobDetail()
	drainTestUI(w, queue)
	w.SelectJobApplication(job1.ID)
	drainTestUI(w, queue)
	tt.Frame()
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情结束原因", "原因甲")
	if err := tt.Click("详情公司名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	typeJobDetailField(t, tt, "详情结束原因", "原因乙")
	unlock()
	drainTestUI(w, queue)
	if w.jobSaveState != "error" {
		t.Fatalf("expected the rebased reason write to fail visibly, state=%q", w.jobSaveState)
	}
	// The error banner pushes the notes field out of the scroll viewport in headless
	// tests; drive the same draft edit the UI would submit directly.
	w.editJobNotes = "新备注"
	w.TouchJobDetailField()
	drainTestUI(w, queue)
	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	// The trigger rejects every reason change, so the notes ride along in the same
	// failed write and must not land without the reason.
	if err != nil || saved.ClosureReason != "旧结束原因" || saved.Notes != "初始备注" {
		t.Fatalf("notes-only save mishandled the pending reason: %+v %v", saved, err)
	}
	if w.editJobClosureReason != "原因乙" || w.editJobNotes != "新备注" || w.jobSaveState != "error" || !w.jobDirty {
		t.Fatalf("pending reason dropped or failure masked: reason=%q notes=%q state=%q dirty=%v",
			w.editJobClosureReason, w.editJobNotes, w.jobSaveState, w.jobDirty)
	}
}

// Discarding job changes while a save is in flight must rebase later edits on that write's result, keeping its company.
func TestJobAsyncDiscardThenEditKeepsInFlightSaveAsBaseline(t *testing.T) {
	s, w, tt, queue, job1, _ := asyncJobTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情公司名称", "飞行中的新公司")
	if err := tt.Click("详情公司名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if w.editJobCompany != "" || w.jobSaveState != "validation_error" {
		t.Fatalf("company not cleared before discard: %q %q", w.editJobCompany, w.jobSaveState)
	}
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	typeJobDetailField(t, tt, "详情通用备注", "落地前补充的备注")
	unlock()
	drainTestUI(w, queue)
	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.CompanyName != "飞行中的新公司" || saved.Notes != "落地前补充的备注" {
		t.Fatalf("post-discard edit reverted the in-flight save: %+v %v", saved, err)
	}
	if w.editJobCompany != saved.CompanyName || w.editJobNotes != saved.Notes || w.jobDirty || w.jobSaving {
		t.Fatalf("draft diverged from persisted application: company=%q notes=%q dirty=%v saving=%v", w.editJobCompany, w.editJobNotes, w.jobDirty, w.jobSaving)
	}
}

// Stale background loads cannot overwrite newer job selection or filter views, and inline workflow transitions do not clobber unsaved detail text edits.
func TestJobAsyncStaleLoadAndInlineVsDetailIsolation(t *testing.T) {
	s, w, tt, queue, job1, job2 := asyncJobTestWorkbench(t)

	// Hold store during inline workflow stage transition while detail modal holds an unsaved validation-blocked draft.
	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情投递链接", "ftp://unsaved.invalid")
	w.editJobNotes = "尚未保存的详情手写备注"
	w.TouchJobDetailField()
	w.ChangeJobStage(job1.ID, model.JobStageInterview1)
	unlock()
	drainTestUI(w, queue)
	tt.Frame()

	saved, err := w.app.JobApplications.Get(context.Background(), job1.ID)
	if err != nil || saved.Stage != model.JobStageInterview1 || saved.Notes != "初始备注" {
		t.Fatalf("expected inline stage transition without flushing invalid unsaved draft, got %+v err=%v", saved, err)
	}
	if w.editJobNotes != "尚未保存的详情手写备注" || w.detailJob.Stage != model.JobStageInterview1 || len(w.detailJobEvents) < 2 {
		t.Fatalf("expected inline stage update without clobbering unsaved detail notes, notes=%q stage=%q events=%d", w.editJobNotes, w.detailJob.Stage, len(w.detailJobEvents))
	}

	// Discard invalid draft, then stale load from opening job1 cannot replace newer job2 selection/navigation.
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	w.CloseJobDetail()
	drainTestUI(w, queue)
	unlock2 := holdTestStore(t, s)
	w.SelectJobApplication(job1.ID)
	w.SwitchModule("projects")
	w.SwitchModule("job_applications")
	w.SelectJobApplication(job2.ID)
	unlock2()
	drainTestUI(w, queue)
	if w.selectedJobID != job2.ID || w.detailJob == nil || w.detailJob.ID != job2.ID {
		t.Fatalf("expected newer job2 selection to win over stale load, selected=%q detail=%+v", w.selectedJobID, w.detailJob)
	}
}

// Restoring a backup is blocked while a job detail draft is actively saving and succeeds once saving drains.
func TestJobAsyncRestoreBackupGuardWhileSaving(t *testing.T) {
	s, w, tt, queue, _, _ := asyncJobTestWorkbench(t)
	zipPath := filepath.Join(t.TempDir(), "manual-backup.zip")
	if err := w.app.Backups.Export(context.Background(), zipPath); err != nil {
		t.Fatal(err)
	}

	unlock := holdTestStore(t, s)
	typeJobDetailField(t, tt, "详情公司名称", "保存中阻断恢复")
	w.RequestRestoreBackup(zipPath)
	w.TriggerRestoreBackup()
	if w.IsRestoring() || !w.confirmRestoreOpen {
		t.Fatalf("expected TriggerRestoreBackup blocked while jobSaving is true, restoring=%v confirm=%v", w.IsRestoring(), w.confirmRestoreOpen)
	}

	unlock()
	drainTestUI(w, queue)
	w.TriggerRestoreBackup()
	if !w.IsRestoring() {
		t.Fatal("expected restore to start once job save drained")
	}
	drainTestUI(w, queue)
	if w.IsRestoring() || w.jobDetailOpen {
		t.Fatalf("expected completed restore to reset restoring & job detail state, restoring=%v open=%v", w.IsRestoring(), w.jobDetailOpen)
	}
}

// An earlier timeline write must not consume a later submission, clear newer text, or race a restore.
func TestQueuedTimelineNotesPreserveNewInput(t *testing.T) {
	for _, tc := range []struct {
		kind          model.JobEventType
		input, submit string
	}{
		{model.JobEventNoteAdded, "时间线备注输入", "添加备注"},
		{model.JobEventStageBackfilled, "补录说明", "补录历史事件"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			s, w, tt, queue, job, _ := asyncJobTestWorkbench(t)
			if tc.kind == model.JobEventStageBackfilled {
				if err := tt.Click("切换补录历史节点"); err != nil {
					t.Fatal(err)
				}
			}
			unlock := holdTestStore(t, s)
			for _, note := range []string{"first", "first", "second"} {
				typeJobDetailField(t, tt, tc.input, note)
				if err := tt.Click(tc.submit); err != nil {
					t.Fatal(err)
				}
			}
			typeJobDetailField(t, tt, tc.input, "third draft")
			w.RequestRestoreBackup("unused.zip")
			w.TriggerRestoreBackup()
			if w.IsRestoring() || w.PrepareClose() {
				t.Fatal("restore or close raced queued timeline writes")
			}
			unlock()
			drainTestUI(w, queue)
			draft := w.jobNoteInput
			if tc.kind == model.JobEventStageBackfilled {
				draft = w.jobBackfillNote
			}
			if draft != "third draft" || !w.PrepareClose() {
				t.Fatalf("unexpected draft after writes: %q", draft)
			}
			events, err := w.app.JobApplications.ListEvents(context.Background(), job.ID)
			if err != nil {
				t.Fatal(err)
			}
			notes := map[string]int{}
			for _, event := range events {
				if event.EventType == tc.kind {
					notes[event.Note]++
				}
			}
			if len(notes) != 2 || notes["first"] != 1 || notes["second"] != 1 {
				t.Fatalf("wrong persisted notes: %v", notes)
			}
		})
	}
}

// A selection queued before a new edit must leave the edited record and its invalid draft open.
func TestQueuedJobSelectionPreservesNewDraft(t *testing.T) {
	s, w, tt, queue, first, second := asyncJobTestWorkbench(t)
	unlock := holdTestStore(t, s)
	w.SelectJobApplication(second.ID)
	typeJobDetailField(t, tt, "详情投递链接", "ftp://invalid.example")
	unlock()
	drainTestUI(w, queue)
	if w.selectedJobID != first.ID || !w.jobDirty || w.editJobURL != "ftp://invalid.example" {
		t.Fatal("stale selection replaced the new draft")
	}
}

// Regression: discarding during an in-flight task or job save dropped a subsequent close or navigation and left afterSave armed for a later save.
func TestAsyncDiscardThenNavigateDoesNotDropOrLeakAfterSave(t *testing.T) {
	s, w, tt, queue, ps, _ := asyncTestWorkbench(t)
	task2, err := w.app.Projects.CreateTask(context.Background(), ps[0].ID, projects.TaskInput{
		Title:    "Second task",
		Status:   model.Planned,
		Priority: model.Low,
	})
	if err != nil {
		t.Fatal(err)
	}

	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "In flight task")
	typeTaskTitle(t, tt, "   ")
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	w.SelectProject(ps[1].ID)
	unlock()
	drainTestUI(w, queue)
	tt.Frame()
	if w.activeProjectID != ps[1].ID {
		t.Fatalf("expected deferred project switch after discard-while-saving, got %q", w.activeProjectID)
	}

	w.SelectProject(ps[0].ID)
	drainTestUI(w, queue)
	w.SelectTask(task2.ID)
	drainTestUI(w, queue)
	tt.Frame()
	typeTaskTitle(t, tt, "Independent task edit")
	drainTestUI(w, queue)
	if w.activeProjectID != ps[0].ID || w.selectedTaskID != task2.ID {
		t.Fatalf("stale afterSave leaked into unrelated task save: project=%q selected=%q", w.activeProjectID, w.selectedTaskID)
	}

	s2, w2, tt2, queue2, _, job2 := asyncJobTestWorkbench(t)
	unlock2 := holdTestStore(t, s2)
	typeJobDetailField(t, tt2, "详情公司名称", "在途公司")
	typeJobDetailField(t, tt2, "详情公司名称", "   ")
	if err := tt2.Click("放弃修改"); err != nil {
		t.Fatal(err)
	}
	w2.CloseJobDetail()
	unlock2()
	drainTestUI(w2, queue2)
	tt2.Frame()
	if w2.jobDetailOpen || w2.selectedJobID != "" {
		t.Fatalf("expected job detail closed after discard-while-saving finished, open=%v selected=%q", w2.jobDetailOpen, w2.selectedJobID)
	}

	w2.SelectJobApplication(job2.ID)
	drainTestUI(w2, queue2)
	tt2.Frame()
	typeJobDetailField(t, tt2, "详情公司名称", "独立求职修改")
	drainTestUI(w2, queue2)
	if !w2.jobDetailOpen || w2.selectedJobID != job2.ID {
		t.Fatalf("stale afterSave leaked into unrelated job save: open=%v selected=%q", w2.jobDetailOpen, w2.selectedJobID)
	}
}

// Regression: SoftDeleteTask lacked an in-flight save guard and left pending save state behind when deleting the active task.
func TestAsyncSoftDeleteTaskGuardWhileSaving(t *testing.T) {
	s, w, tt, queue, _, task := asyncTestWorkbench(t)
	unlock := holdTestStore(t, s)
	typeTaskTitle(t, tt, "Saving before delete")
	w.SoftDeleteTask(task.ID)
	if w.selectedTaskID != task.ID {
		t.Fatal("SoftDeleteTask deleted the task while an async save was still in flight")
	}
	unlock()
	drainTestUI(w, queue)
	tt.Frame()

	typeTaskTitle(t, tt, "   ")
	w.SoftDeleteTask(task.ID)
	drainTestUI(w, queue)
	if w.selectedTaskID != "" || w.pendingSave != nil || w.afterSave != nil || w.discardAfterSave {
		t.Fatalf("expected active task save state cleared on delete, selected=%q pending=%v", w.selectedTaskID, w.pendingSave)
	}
	if _, err := w.app.Projects.GetTask(context.Background(), task.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected deleted task not found, got %v", err)
	}
}
