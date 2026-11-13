package frontend

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"aion/backend"
	"aion/backend/attachments"
	"aion/backend/backup"
	"aion/backend/model"
	"aion/backend/projects"
	"aion/backend/store"

	"github.com/egoist/mygo/ui"
)

func openTestWorkbench(t *testing.T, dataDir string, host Host) (*backend.App, *Workbench, *ui.Tester) {
	t.Helper()
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{
		DataDir:          dataDir,
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatalf("backend.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Close()
	})
	wb := New(app, host)
	tt := ui.NewTesterWith(wb.View, 1440, 900, ui.TesterOptions{
		Preferences: ui.Preferences{ReduceMotion: true},
	})
	return app, wb, tt
}

func writeSamplePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 120, B: 220, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
}

// Project actions persist edits, isolate boards, and support archive/restore through the UI.
func TestProjectLifecycle(t *testing.T) {
	dataDir := t.TempDir()
	app, wb, tt := openTestWorkbench(t, dataDir, Host{})

	// Create first project via UI modal
	if err := tt.Click("新建项目"); err != nil {
		t.Fatalf("click 新建项目: %v", err)
	}
	if err := tt.Click("项目名称输入框"); err != nil {
		t.Fatalf("click 项目名称输入框: %v", err)
	}
	tt.Type("Aion 桌面研发")
	if err := tt.Click("图标 代码"); err != nil {
		t.Fatalf("click 图标 代码: %v", err)
	}
	if err := tt.Click("项目简介输入框"); err != nil {
		t.Fatalf("click 项目简介输入框: %v", err)
	}
	tt.Type("构建 macOS 本地优先项目管理中心")
	if err := tt.Click("创建项目"); err != nil {
		t.Fatalf("click 创建项目: %v", err)
	}

	if len(wb.activeProjects) != 1 || wb.activeProjects[0].Name != "Aion 桌面研发" || wb.activeProjects[0].Icon != "code" {
		t.Fatalf("unexpected active projects after create: %+v", wb.activeProjects)
	}
	if wb.navPage != NavProjectBoard {
		t.Fatalf("expected navigation to project board, got %s", wb.navPage)
	}

	// Edit project via UI modal
	if err := tt.Click("编辑项目"); err != nil {
		t.Fatalf("click 编辑项目: %v", err)
	}
	wb.projectModalName = "Aion 核心研发"
	tt.Frame()
	if err := tt.Click("图标 发布"); err != nil {
		t.Fatalf("click 图标 发布: %v", err)
	}
	if err := tt.Click("保存修改"); err != nil {
		t.Fatalf("click 保存修改: %v", err)
	}
	if wb.ActiveProject() == nil || wb.ActiveProject().Name != "Aion 核心研发" || wb.ActiveProject().Icon != "rocket" {
		t.Fatalf("project not updated: %+v", wb.ActiveProject())
	}
	persisted, err := app.Projects.GetProject(context.Background(), wb.activeProjectID)
	if err != nil || persisted.Name != "Aion 核心研发" || persisted.Icon != "rocket" {
		t.Fatalf("project update not persisted in DB: %+v err=%v", persisted, err)
	}

	// Create second project and verify switching projects isolates board data
	wb.OpenCreateProjectModal()
	wb.projectModalName = "个人读书计划"
	wb.projectModalIcon = "book"
	wb.SubmitProjectModal()
	tt.Frame()

	proj2ID := wb.activeProjectID
	if _, err := wb.QuickCreateTask(model.Planned, "阅读 DDIA"); err != nil {
		t.Fatalf("QuickCreateTask on proj2: %v", err)
	}
	tt.Frame()
	if len(wb.boardTasks) != 1 {
		t.Fatalf("expected 1 task in proj2, got %d", len(wb.boardTasks))
	}

	// Switch back to first project via sidebar
	if err := tt.Click("Aion 核心研发"); err != nil {
		t.Fatalf("click Aion 核心研发 in sidebar: %v", err)
	}
	if len(wb.boardTasks) != 0 {
		t.Fatalf("expected 0 tasks in proj1, got %d", len(wb.boardTasks))
	}

	// Archive second project, view it in Archived Projects, and restore it
	wb.SelectProject(proj2ID)
	tt.Frame()
	if err := tt.Click("归档项目"); err != nil {
		t.Fatalf("click 归档项目: %v", err)
	}
	if len(wb.archivedProjects) != 1 || len(wb.activeProjects) != 1 {
		t.Fatalf("expected 1 active and 1 archived project, got %d active, %d archived", len(wb.activeProjects), len(wb.archivedProjects))
	}

	if err := tt.Click("已归档项目"); err != nil {
		t.Fatalf("click 已归档项目: %v", err)
	}
	if err := tt.Click("恢复归档 个人读书计划"); err != nil {
		t.Fatalf("click 恢复归档: %v", err)
	}
	if len(wb.archivedProjects) != 0 || len(wb.activeProjects) != 2 {
		t.Fatalf("expected 2 active and 0 archived projects after unarchive")
	}
}

// Kanban actions reorder within a column, move across columns by click and drag, filter tasks, and persist across restarts.
func TestKanbanBoardDragMoveFilterAndPersistence(t *testing.T) {
	dataDir := t.TempDir()
	app1, wb1, tt1 := openTestWorkbench(t, dataDir, Host{})

	// Create project
	wb1.OpenCreateProjectModal()
	wb1.projectModalName = "看板测试项目"
	wb1.SubmitProjectModal()
	tt1.Frame()

	// Create 3 tasks via modal and quick-add
	if err := tt1.Click("+ 新建任务"); err != nil {
		t.Fatalf("click + 新建任务: %v", err)
	}
	wb1.taskModalTitle = "设计数据库 Schema"
	wb1.taskModalPriority = string(model.Urgent)
	wb1.taskModalTags = "后端, SQLite"
	wb1.taskModalDesc = "包含 projects 与 tasks 表"
	wb1.SubmitTaskModal()
	tt1.Frame()

	t2, err := wb1.QuickCreateTask(model.Planned, "实现 MyGo 看板视图")
	if err != nil {
		t.Fatalf("create task 2: %v", err)
	}
	t3, err := wb1.QuickCreateTask(model.Planned, "编写自动化测试")
	if err != nil {
		t.Fatalf("create task 3: %v", err)
	}
	wb1.ClearSelectedTask()
	tt1.Frame()

	// Verify all 3 are in Planned
	if got := len(wb1.TasksInColumn(model.Planned)); got != 3 {
		t.Fatalf("expected 3 planned tasks, got %d", got)
	}

	// Hover task 3 to reveal its quick-action bar and reorder within Planned column using the Up button
	if r, ok := tt1.Find("任务卡片 编写自动化测试"); !ok {
		t.Fatalf("could not find card 编写自动化测试")
	} else {
		tt1.Move(r.X+r.W/2, r.Y+16)
	}
	if err := tt1.Click("上移 编写自动化测试"); err != nil {
		t.Fatalf("click 上移: %v", err)
	}
	planned := wb1.TasksInColumn(model.Planned)
	if planned[1].ID != t3.ID || planned[2].ID != t2.ID {
		t.Fatalf("unexpected order after step up: %v, %v", planned[1].Title, planned[2].Title)
	}

	// Hover task 2 and move it to InProgress via quick move button
	if r, ok := tt1.Find("任务卡片 实现 MyGo 看板视图"); !ok {
		t.Fatalf("could not find card 实现 MyGo 看板视图")
	} else {
		tt1.Move(r.X+r.W/2, r.Y+16)
	}
	if err := tt1.Click("移至进行中 实现 MyGo 看板视图"); err != nil {
		t.Fatalf("click 移至进行中: %v", err)
	}
	if len(wb1.TasksInColumn(model.InProgress)) != 1 {
		t.Fatalf("expected 1 in_progress task")
	}

	// Drag task 3 from Planned column and drop onto Done column
	cardRect, ok := tt1.Find("任务卡片 编写自动化测试")
	if !ok {
		t.Fatalf("could not find card 编写自动化测试")
	}
	doneColRect, ok := tt1.Find("看板列 已完成")
	if !ok {
		t.Fatalf("could not find column 已完成")
	}
	fromX, fromY := cardRect.X+cardRect.W/2, cardRect.Y+16
	toX, toY := doneColRect.X+doneColRect.W/2, doneColRect.Y+doneColRect.H/2
	tt1.Press(fromX, fromY)
	for step := 1; step <= 4; step++ {
		f := float32(step) / 4
		tt1.Move(fromX+(toX-fromX)*f, fromY+(toY-fromY)*f)
	}
	tt1.Release(toX, toY)

	if len(wb1.TasksInColumn(model.Done)) != 1 {
		t.Fatalf("expected drag-and-drop to move task 3 to Done column, got %d in Done", len(wb1.TasksInColumn(model.Done)))
	}

	// Add shared and distinct tags on task 2 to verify multi-tag AND intersection
	wb1.SelectTask(t2.ID)
	tt1.Frame()
	if err := tt1.Click("添加标签输入框"); err != nil {
		t.Fatalf("click 添加标签输入框: %v", err)
	}
	tt1.Type("后端")
	if err := tt1.Click("添加标签"); err != nil {
		t.Fatalf("click 添加标签: %v", err)
	}
	wb1.ClearSelectedTask()
	tt1.Frame()

	// Test search via UI input
	if err := tt1.Click("搜索任务"); err != nil {
		t.Fatalf("click 搜索任务: %v", err)
	}
	tt1.Type("SQLite")
	if len(wb1.filteredTasks) != 1 || wb1.filteredTasks[0].Title != "设计数据库 Schema" {
		t.Fatalf("expected search for SQLite to match 1 task, got %d", len(wb1.filteredTasks))
	}

	if err := tt1.Click("清除筛选"); err != nil {
		t.Fatalf("click 清除筛选: %v", err)
	}
	if len(wb1.filteredTasks) != 3 {
		t.Fatalf("expected 3 tasks after clearing filters, got %d", len(wb1.filteredTasks))
	}

	// Test priority filter via UI select
	if err := tt1.Click("按优先级筛选"); err != nil {
		t.Fatalf("click 按优先级筛选: %v", err)
	}
	tt1.Key(0, ui.KeyDown)
	tt1.Key(0, ui.KeyEnter)
	if len(wb1.filteredTasks) != 1 || wb1.filteredTasks[0].Priority != model.Urgent {
		t.Fatalf("expected urgent priority filter to match 1 task, got %+v", wb1.filteredTasks)
	}
	if err := tt1.Click("清除筛选"); err != nil {
		t.Fatalf("click 清除筛选 after priority: %v", err)
	}

	// Filter by multi-tag AND pills
	if err := tt1.Click("筛选标签 后端"); err != nil {
		t.Fatalf("click 筛选标签 后端: %v", err)
	}
	if len(wb1.filteredTasks) != 2 {
		t.Fatalf("expected 2 tasks for tag 后端, got %d", len(wb1.filteredTasks))
	}
	if err := tt1.Click("筛选标签 SQLite"); err != nil {
		t.Fatalf("click 筛选标签 SQLite: %v", err)
	}
	if len(wb1.filteredTasks) != 1 || wb1.filteredTasks[0].Title != "设计数据库 Schema" {
		t.Fatalf("expected 1 task for AND tags [后端, SQLite], got %+v", wb1.filteredTasks)
	}
	if err := tt1.Click("清除筛选"); err != nil {
		t.Fatalf("click 清除筛选 after tags: %v", err)
	}

	// Close first app instance and reopen to verify SQLite persistence
	if err := wb1.FlushPending(); err != nil {
		t.Fatalf("FlushPending: %v", err)
	}
	if err := app1.Close(); err != nil {
		t.Fatalf("close app1: %v", err)
	}

	_, wb2, _ := openTestWorkbench(t, dataDir, Host{})
	if len(wb2.TasksInColumn(model.Planned)) != 1 ||
		len(wb2.TasksInColumn(model.InProgress)) != 1 ||
		len(wb2.TasksInColumn(model.Done)) != 1 {
		t.Fatalf("persisted column counts mismatch after restart")
	}
}

// Detail actions validate edits, render Markdown, persist subtask checks, and use managed attachments.
func TestTaskDetailMarkdownSubtasksAndAttachments(t *testing.T) {
	dataDir := t.TempDir()
	srcDir := t.TempDir()

	imgPath := filepath.Join(srcDir, "architecture.png")
	writeSamplePNG(t, imgPath)
	docPath := filepath.Join(srcDir, "notes.txt")
	if err := os.WriteFile(docPath, []byte("offline-first notes"), 0o600); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}
	exportDest := filepath.Join(srcDir, "exported-notes.txt")

	var openedPaths, revealedPaths []string
	host := Host{
		PickOpenFiles: func(_ string, _ bool, _ ...string) ([]string, error) {
			return []string{imgPath}, nil
		},
		PickSaveFile: func(_ string, _ string, _ ...string) (string, error) {
			return exportDest, nil
		},
		OpenPath: func(path string) error {
			openedPaths = append(openedPaths, path)
			return nil
		},
		RevealPath: func(path string) error {
			revealedPaths = append(revealedPaths, path)
			return nil
		},
	}

	app, wb, tt := openTestWorkbench(t, dataDir, host)
	wb.OpenCreateProjectModal()
	wb.projectModalName = "详情与附件项目"
	wb.SubmitProjectModal()

	task, err := wb.QuickCreateTask(model.Planned, "完善任务详情面板")
	if err != nil {
		t.Fatalf("QuickCreateTask: %v", err)
	}
	wb.SelectTask(task.ID)
	tt.Frame()

	// 1. Empty title validation guard
	wb.editTitle = "   "
	if err := wb.SaveSelectedTask(); err == nil {
		t.Fatalf("expected validation error when title is empty")
	}
	wb.editTitle = "完善任务详情与附件"
	if err := wb.SaveSelectedTask(); err != nil {
		t.Fatalf("expected valid title save to succeed: %v", err)
	}
	tt.Frame()

	// 2. Markdown toolbar formatting
	if err := tt.Click("格式 H1"); err != nil {
		t.Fatalf("click 格式 H1: %v", err)
	}
	if err := tt.Click("格式 代码块"); err != nil {
		t.Fatalf("click 格式 代码块: %v", err)
	}
	if wb.editDescription == "" {
		t.Fatalf("expected Markdown toolbar to insert formatting into description")
	}

	// 3. Subtasks: add 2 subtasks, check both, verify parent task remains Planned
	wb.newSubtaskTitle = "支持图片缩略图预览"
	if err := tt.Click("添加子任务"); err != nil {
		t.Fatalf("click 添加子任务 1: %v", err)
	}
	wb.newSubtaskTitle = "支持拖入文件导入"
	if err := tt.Click("添加子任务"); err != nil {
		t.Fatalf("click 添加子任务 2: %v", err)
	}
	if len(wb.editSubtasks) != 2 {
		t.Fatalf("expected 2 subtasks, got %d", len(wb.editSubtasks))
	}

	if err := tt.Click("勾选子任务 支持图片缩略图预览"); err != nil {
		t.Fatalf("check subtask 1: %v", err)
	}
	if err := tt.Click("勾选子任务 支持拖入文件导入"); err != nil {
		t.Fatalf("check subtask 2: %v", err)
	}
	savedTask, err := app.Projects.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("reload task after checking subtasks: %v", err)
	}
	if len(savedTask.Subtasks) != 2 || !savedTask.Subtasks[0].Done || !savedTask.Subtasks[1].Done {
		t.Fatalf("subtask checks were not persisted: %+v", savedTask.Subtasks)
	}
	if savedTask.Status != model.Planned {
		t.Fatalf("checking all subtasks must NOT auto-complete parent task, got status %s", savedTask.Status)
	}

	// 4. Attachments: import image via button and document via native DropFiles
	if err := tt.Click("+ 导入附件"); err != nil {
		t.Fatalf("click + 导入附件: %v", err)
	}
	dropRect, ok := tt.Find("附件拖拽上传区")
	if !ok {
		t.Fatalf("could not find 附件拖拽上传区")
	}
	if !tt.DropFiles(dropRect.X+10, dropRect.Y+10, docPath) {
		t.Fatalf("DropFiles was not accepted by attachment dropzone")
	}
	if len(wb.detailAttachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(wb.detailAttachments))
	}

	// Delete original source files and verify managed copies still work
	_ = os.Remove(imgPath)
	_ = os.Remove(docPath)
	tt.Frame()

	for _, att := range wb.detailAttachments {
		if IsImageAttachment(att) {
			if bmp := wb.AttachmentBitmap(att); bmp == nil {
				t.Fatalf("expected decoded bitmap from managed storage after source file removal")
			}
		}
	}

	// Scroll the task detail modal down so attachment action buttons are in view
	if panelRect, ok := tt.Find("任务详情弹窗"); ok {
		tt.Scroll(panelRect.X+panelRect.W/2, panelRect.Y+panelRect.H/2, 0, 320)
	} else {
		t.Fatal("could not find 任务详情弹窗")
	}

	// Open, Reveal in Finder, and Export attachment via UI buttons
	if err := tt.Click("打开附件 notes.txt"); err != nil {
		t.Fatalf("click 打开附件: %v", err)
	}
	if err := tt.Click("在 Finder 中显示 notes.txt"); err != nil {
		t.Fatalf("click 在 Finder 中显示: %v", err)
	}
	if err := tt.Click("导出附件 notes.txt"); err != nil {
		t.Fatalf("click 导出附件: %v", err)
	}
	if len(openedPaths) != 1 || len(revealedPaths) != 1 {
		t.Fatalf("expected OpenPath and RevealPath to be called once each")
	}
	exportedBytes, err := os.ReadFile(exportDest)
	if err != nil || string(exportedBytes) != "offline-first notes" {
		t.Fatalf("unexpected exported attachment content: %q (err=%v)", string(exportedBytes), err)
	}

	// Remove attachment via UI and verify cleanup
	if err := tt.Click("移除附件 notes.txt"); err != nil {
		t.Fatalf("click 移除附件 notes.txt: %v", err)
	}
	if len(wb.detailAttachments) != 1 {
		t.Fatalf("expected 1 attachment remaining after removal, got %d (task=%s)", len(wb.detailAttachments), task.ID)
	}
}

// Settings and trash actions persist preferences, surface errors, and restore data through confirmation dialogs.
func TestRecycleBinAndSettingsBackupRestore(t *testing.T) {
	dataDir := t.TempDir()
	backupDir := t.TempDir()
	manualZip := filepath.Join(backupDir, "manual-full.zip")

	var appliedThemes []string
	host := Host{
		PickSaveFile: func(_ string, _ string, _ ...string) (string, error) {
			return manualZip, nil
		},
		PickOpenFiles: func(_ string, _ bool, _ ...string) ([]string, error) {
			return []string{manualZip}, nil
		},
		SetTheme: func(th string) {
			appliedThemes = append(appliedThemes, th)
		},
	}

	app, wb, tt := openTestWorkbench(t, dataDir, host)

	// Create project and two tasks
	wb.OpenCreateProjectModal()
	wb.projectModalName = "备份与回收站测试项目"
	wb.SubmitProjectModal()
	projID := wb.activeProjectID

	task1, err := wb.QuickCreateTask(model.Planned, "独立删除的任务")
	if err != nil {
		t.Fatalf("create task1: %v", err)
	}
	_, err = wb.QuickCreateTask(model.InProgress, "随项目删除的任务")
	if err != nil {
		t.Fatalf("create task2: %v", err)
	}
	wb.ClearSelectedTask()
	tt.Frame()

	// 1. Export a full manual backup in Settings and verify error surfacing on invalid backup dir
	wb.NavigateTo(NavSettings)
	tt.Frame()

	// Switch theme to Dark via UI Segmented control
	if err := tt.Click("深色"); err != nil {
		t.Fatalf("click 深色: %v", err)
	}
	if got, _ := app.Settings.Theme(context.Background()); got != ThemeDark {
		t.Fatalf("expected persisted dark theme, got %s", got)
	}
	if !slices.Contains(appliedThemes, ThemeDark) {
		t.Fatalf("expected Host.SetTheme to be invoked with dark")
	}

	// Reject store root as automatic backup directory and surface an error toast
	wb.backupDirInput = dataDir
	tt.Frame()
	if err := tt.Click("保存策略"); err != nil {
		t.Fatalf("click 保存策略: %v", err)
	}
	if !tt.HasText("保存自动备份策略失败") {
		t.Fatalf("expected error toast when saving protected backup directory")
	}
	wb.ReloadSettings()
	if err := tt.Click("Close"); err != nil {
		t.Fatalf("dismiss toast: %v", err)
	}

	// Trigger manual backup export via UI button
	if err := tt.Click("导出完整备份 (.zip)"); err != nil {
		t.Fatalf("click 导出完整备份: %v", err)
	}
	if _, err := os.Stat(manualZip); err != nil {
		t.Fatalf("expected exported manual backup zip at %s: %v", manualZip, err)
	}

	// 2. Test Recycle Bin: soft-delete task1, then soft-delete project
	wb.SoftDeleteTask(task1.ID)
	wb.SoftDeleteProject(projID)
	wb.NavigateTo(NavTrash)
	tt.Frame()

	if len(wb.trashItems) < 2 {
		t.Fatalf("expected project and tasks in trash, got %d", len(wb.trashItems))
	}

	// Try restoring task1 while its parent project is still in Trash -> triggers ErrParentDeleted modal
	if err := tt.Click("恢复 独立删除的任务"); err != nil {
		t.Fatalf("click 恢复 独立删除的任务: %v", err)
	}
	if !wb.trashParentErrModalOpen {
		t.Fatalf("expected ErrParentDeleted warning modal to open")
	}
	// Click "一并恢复项目与任务" in modal
	if err := tt.Click("一并恢复项目与任务"); err != nil {
		t.Fatalf("click 一并恢复项目与任务: %v", err)
	}
	if len(wb.activeProjects) != 1 || len(wb.trashItems) != 0 {
		t.Fatalf("expected project and tasks to be restored from trash, got activeProjects=%d trash=%d", len(wb.activeProjects), len(wb.trashItems))
	}

	// Soft-delete project again and Clear Trash with confirmation modal
	wb.SoftDeleteProject(projID)
	wb.NavigateTo(NavTrash)
	tt.Frame()
	if err := tt.Click("清空回收站"); err != nil {
		t.Fatalf("click 清空回收站: %v", err)
	}
	if err := tt.Click("确认清空回收站"); err != nil {
		t.Fatalf("click 确认清空回收站: %v", err)
	}
	if len(wb.trashItems) != 0 || len(wb.activeProjects) != 0 {
		t.Fatalf("expected all data cleared after ClearTrash")
	}

	// 3. Restore from the exported backup ZIP via UI confirmation modal
	wb.NavigateTo(NavSettings)
	tt.Frame()
	tt.Scroll(500, 300, 0, 320)
	if err := tt.Click("从备份包恢复 (.zip)..."); err != nil {
		t.Fatalf("click 从备份包恢复: %v", err)
	}
	if !wb.confirmRestoreOpen {
		t.Fatalf("expected restore confirmation modal to open")
	}
	if err := tt.Click("确认覆盖并恢复"); err != nil {
		t.Fatalf("click 确认覆盖并恢复: %v", err)
	}

	if len(wb.activeProjects) != 1 || wb.activeProjects[0].Name != "备份与回收站测试项目" {
		t.Fatalf("expected project restored from backup zip, got %+v", wb.activeProjects)
	}
	if len(wb.boardTasks) != 2 {
		t.Fatalf("expected 2 tasks restored from backup zip, got %d", len(wb.boardTasks))
	}
	if wb.lastRestoreResult == nil || wb.lastRestoreResult.SafetyBackupPath == "" {
		t.Fatalf("expected pre-restore safety backup path to be recorded")
	}
	if _, err := os.Stat(wb.lastRestoreResult.SafetyBackupPath); err != nil {
		t.Fatalf("expected safety backup file to exist on disk: %v", err)
	}
}

// Export populated board and settings snapshots for manual visual inspection when requested.
func TestVisualSnapshotRender(t *testing.T) {
	outPath := os.Getenv("AION_SAVE_SNAPSHOT")
	settingsOutPath := os.Getenv("AION_SAVE_SETTINGS_SNAPSHOT")
	if outPath == "" && settingsOutPath == "" {
		t.Skip("set AION_SAVE_SNAPSHOT or AION_SAVE_SETTINGS_SNAPSHOT to export a PNG for visual inspection")
	}
	dataDir := t.TempDir()
	srcDir := t.TempDir()
	imgPath := filepath.Join(srcDir, "preview.png")
	writeSamplePNG(t, imgPath)

	_, wb, _ := openTestWorkbench(t, dataDir, Host{
		OpenURL:         func(string) error { return nil },
		CheckForUpdates: func() {},
		UpdateInfo: UpdateCapability{
			Version:          "0.1.0",
			AvailableVersion: "0.2.0",
			LastCheck:        time.Date(2026, 11, 11, 15, 30, 0, 0, time.Local),
			IsDev:            false,
			CanSelfUpdate:    true,
			ReleasesURL:      defaultReleasesURL,
		},
	})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "Aion 1.0 桌面端发布"
	wb.projectModalIcon = "rocket"
	wb.projectModalDesc = "macOS 本地优先（Local-First）个人数字化管理中心"
	wb.SubmitProjectModal()

	t1, _ := wb.QuickCreateTask(model.Planned, "完成 .app 打包与图标验证")
	_, _ = wb.QuickCreateTask(model.InProgress, "实现三列看板与拖拽排序")
	_, _ = wb.QuickCreateTask(model.Done, "完成 SQLite 事务存储与全量备份")

	wb.SelectTask(t1.ID)
	wb.editPriority = string(model.High)
	wb.editTags = []string{"macOS", "MyGo", "V1"}
	wb.editDescription = "## 交付清单\n- 支持三列看板拖拽\n- 支持本地受管附件与缩略图"
	wb.editSubtasks = []model.Subtask{
		{ID: "s1", Title: "集成 Aion 品牌 SVG 图标", Done: true},
		{ID: "s2", Title: "执行无头 UI 回归测试", Done: false},
	}
	_ = wb.SaveSelectedTask()
	wb.ImportAttachmentsToSelectedTask([]string{imgPath})
	wb.pendingToasts = nil

	if outPath != "" {
		img := ui.Render(wb.View, 1440, 900, 1.0)
		f, err := os.Create(outPath)
		if err != nil {
			t.Fatalf("create snapshot file: %v", err)
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			t.Fatalf("encode png: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("close snapshot file: %v", err)
		}
	}

	if settingsOutPath != "" {
		wb.ClearSelectedTask()
		wb.NavigateTo(NavSettings)
		wb.pendingToasts = nil
		img := ui.Render(wb.View, 1440, 1280, 1.0)
		f, err := os.Create(settingsOutPath)
		if err != nil {
			t.Fatalf("create settings snapshot file: %v", err)
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			t.Fatalf("encode settings png: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("close settings snapshot file: %v", err)
		}
	}
}

// Regression: closing a task or navigating away after a failed save previously discarded the unsaved draft.
func TestReviewFailedSaveMustKeepDraft(t *testing.T) {
	dataDir := t.TempDir()
	_, wb, tt := openTestWorkbench(t, dataDir, Host{})

	wb.OpenCreateProjectModal()
	wb.projectModalName = "草稿保护"
	wb.SubmitProjectModal()

	task, err := wb.QuickCreateTask(model.Planned, "原始标题")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	task2, err := wb.QuickCreateTask(model.Planned, "第二个任务")
	if err != nil {
		t.Fatalf("create task2: %v", err)
	}

	wb.SelectTask(task.ID)
	wb.editTitle = "   "
	wb.editDescription = "这段未保存正文不可以在关闭面板时静默丢失"

	// Close detail panel must be blocked when FlushPending fails
	wb.ClearSelectedTask()
	if wb.selectedTaskID != task.ID || wb.detailTask == nil {
		t.Fatalf("closing detail panel after failed save must not discard task selection")
	}
	if wb.editDescription != "这段未保存正文不可以在关闭面板时静默丢失" {
		t.Fatalf("closing detail panel after failed save must keep draft description")
	}

	// Switching task, project, or page must also be blocked while draft is invalid
	wb.SelectTask(task2.ID)
	if wb.selectedTaskID != task.ID {
		t.Fatalf("switching task after failed save must be blocked")
	}
	wb.NavigateTo(NavSettings)
	if wb.navPage != NavProjectBoard {
		t.Fatalf("navigating away after failed save must be blocked")
	}
	if wb.PrepareClose() {
		t.Fatalf("PrepareClose must return false when FlushPending fails")
	}

	// Explicit "放弃修改" button in UI restores the saved state and allows closing
	tt.Frame()
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatalf("click 放弃修改: %v", err)
	}
	if wb.editTitle != "原始标题" || wb.dirty {
		t.Fatalf("expected draft to revert to saved title after 放弃修改, got title=%q dirty=%v", wb.editTitle, wb.dirty)
	}
	wb.ClearSelectedTask()
	if wb.selectedTaskID != "" {
		t.Fatalf("expected detail panel to close after discarding invalid draft")
	}
}

// Regression: exporting over an existing backup or attachment previously removed the target file before the new export succeeded.
func TestReviewFailedExportMustPreservePreviousBackup(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	destZip := filepath.Join(outDir, "aion-backup.zip")
	oldBytes := []byte("existing user backup that must survive a failed overwrite")
	if err := os.WriteFile(destZip, oldBytes, 0o600); err != nil {
		t.Fatalf("seed existing backup: %v", err)
	}

	app, wb, _ := openTestWorkbench(t, dataDir, Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "导出失败保护"
	wb.SubmitProjectModal()

	srcAtt := filepath.Join(t.TempDir(), "doc.txt")
	if err := os.WriteFile(srcAtt, []byte("attachment body"), 0o600); err != nil {
		t.Fatalf("write src attachment: %v", err)
	}
	task, err := wb.QuickCreateTask(model.Planned, "带附件任务")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	wb.SelectTask(task.ID)
	wb.ImportAttachmentsToSelectedTask([]string{srcAtt})
	if len(wb.detailAttachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(wb.detailAttachments))
	}
	att := wb.detailAttachments[0]

	destAtt := filepath.Join(outDir, "exported-doc.txt")
	oldAttBytes := []byte("existing exported attachment that must survive")
	if err := os.WriteFile(destAtt, oldAttBytes, 0o600); err != nil {
		t.Fatalf("seed existing exported attachment: %v", err)
	}

	// Close the backend to simulate an export failure.
	if err := app.Close(); err != nil {
		t.Fatalf("close backend: %v", err)
	}

	if err := wb.ExportManualBackup(destZip); err == nil {
		t.Fatalf("expected ExportManualBackup to fail after backend close")
	}

	got, err := os.ReadFile(destZip)
	if err != nil {
		t.Fatalf("existing backup file was deleted on failed export: %v", err)
	}
	if !bytes.Equal(got, oldBytes) {
		t.Fatalf("existing backup content changed on failed export: got %q", got)
	}

	if err := wb.ExportAttachment(att, destAtt); err == nil {
		t.Fatalf("expected ExportAttachment to fail after backend close")
	}
	gotAtt, err := os.ReadFile(destAtt)
	if err != nil {
		t.Fatalf("existing attachment file was deleted on failed export: %v", err)
	}
	if !bytes.Equal(gotAtt, oldAttBytes) {
		t.Fatalf("existing attachment content changed on failed export: got %q", gotAtt)
	}
}

// Regression: long-running backup and restore actions must run asynchronously, block duplicate triggers, and clear busy state on cancel.
func TestReviewAsyncBackgroundTaskAndCancel(t *testing.T) {
	dataDir := t.TempDir()
	dispatched := make(chan func(), 16)
	host := Host{
		Dispatch: func(fn func()) {
			dispatched <- fn
		},
	}

	_, wb, _ := openTestWorkbench(t, dataDir, host)

	started := make(chan struct{})
	release := make(chan struct{})
	releaseOnce := false
	t.Cleanup(func() {
		if !releaseOnce {
			close(release)
		}
	})

	ok := wb.runAsyncTask("正在执行耗时备份...", func(ctx context.Context) func() {
		close(started)
		select {
		case <-ctx.Done():
			return func() {}
		case <-release:
			return func() {}
		}
	})
	if !ok {
		t.Fatalf("expected first async task to start")
	}

	<-started
	if !wb.IsBusy() {
		t.Fatalf("expected workbench to report busy state while background task is running")
	}

	// Duplicate trigger must be rejected while busy
	if wb.runAsyncTask("重复任务", func(_ context.Context) func() { return nil }) {
		t.Fatalf("expected duplicate async task to be rejected while busy")
	}

	// Cancel busy task and drain dispatched UI callback
	wb.CancelBusyTask()
	wb.WaitForTasks()
	for len(dispatched) > 0 {
		fn := <-dispatched
		fn()
	}
	if wb.IsBusy() {
		t.Fatalf("expected workbench to clear busy state after task cancellation and UI dispatch")
	}
	releaseOnce = true
	close(release)
}

// Regression: exporting an attachment or backup previously allowed overwriting live store files or case-aliased managed attachments.
func TestExportRejectsProtectedAndCaseAliasedPaths(t *testing.T) {
	dir := t.TempDir()
	app, wb, _ := openTestWorkbench(t, dir, Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "protected export"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "attachment")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	source := filepath.Join(t.TempDir(), "aion.sqlite")
	payload := []byte("attachment content, not the application database")
	if err := os.WriteFile(source, payload, 0600); err != nil {
		t.Fatal(err)
	}
	wb.ImportAttachmentsToSelectedTask([]string{source})
	if len(wb.detailAttachments) != 1 {
		t.Fatal("attachment import failed")
	}
	dest := filepath.Join(dir, "aion.sqlite")
	exportErr := wb.ExportAttachment(wb.detailAttachments[0], dest)
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if exportErr == nil || bytes.Equal(got, payload) {
		t.Fatalf("live database overwritten: export error=%v, bytes=%q", exportErr, got)
	}

	for _, protected := range []string{
		filepath.Join(dir, ".aion.lock"),
		filepath.Join(dir, ".backup-auto-state.json"),
		filepath.Join(dir, "aion.sqlite-wal"),
		filepath.Join(dir, "attachments", "escape.bin"),
	} {
		if err := wb.ExportAttachment(wb.detailAttachments[0], protected); err == nil {
			t.Fatalf("expected attachment export to protected path %s to be rejected", protected)
		}
		if err := wb.ExportManualBackup(protected); err == nil {
			t.Fatalf("expected backup export to protected path %s to be rejected", protected)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "ATTACHMENTS")); err == nil {
		otherSrc := filepath.Join(t.TempDir(), "replacement.txt")
		otherPayload := []byte("different exported content")
		if err := os.WriteFile(otherSrc, otherPayload, 0600); err != nil {
			t.Fatal(err)
		}
		replacement, err := app.Attachments.Import(context.Background(), task.ID, otherSrc)
		if err != nil {
			t.Fatal(err)
		}
		original := wb.detailAttachments[0]
		aliasErr := wb.ExportAttachment(replacement, filepath.Join(dir, "ATTACHMENTS", original.StorageName))
		managedBytes, err := os.ReadFile(filepath.Join(dir, "attachments", original.StorageName))
		if err != nil {
			t.Fatal(err)
		}
		if aliasErr == nil || bytes.Equal(managedBytes, otherPayload) {
			t.Fatalf("case alias overwrote managed attachment: error=%v, content=%q", aliasErr, managedBytes)
		}
	}
}

// Regression: rendering Settings or task attachment previews previously acquired the Store lock and froze the UI during backups.
func TestRenderDoesNotWaitForStoreLock(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "render lock check"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "with attachment")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	imgPath := filepath.Join(t.TempDir(), "thumb.png")
	writeSamplePNG(t, imgPath)
	wb.ImportAttachmentsToSelectedTask([]string{imgPath})

	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	app.Backups = backup.New(s)
	app.Attachments = attachments.New(s)

	locked := make(chan struct{})
	release := make(chan struct{})
	lockDone := make(chan error, 1)
	go func() {
		lockDone <- s.Exclusive(context.Background(), func(_ *sql.DB) error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	tt.Frame()
	wb.navPage = NavSettings
	tt.Frame()

	close(release)
	if err := <-lockDone; err != nil {
		t.Fatal(err)
	}
}

// Regression: restoring a backup previously flushed pre-restore drafts or allowed queued UI writes to overwrite restored rows.
func TestRestoreBlocksPreRestoreDraftAndBusyEdits(t *testing.T) {
	app, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "async restore"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "snapshot title")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	wb.editDescription = "snapshot description"
	if err := wb.SaveSelectedTask(); err != nil {
		t.Fatal(err)
	}
	zip := filepath.Join(t.TempDir(), "snapshot.zip")
	if err := wb.ExportManualBackup(zip); err != nil {
		t.Fatal(err)
	}
	wb.editDescription = "unsaved or post-backup description from the old database"
	dispatched := make(chan func(), 1)
	wb.host.Dispatch = func(fn func()) { dispatched <- fn }
	wb.RequestRestoreBackup(zip)
	wb.TriggerRestoreBackup()
	wb.WaitForTasks()
	if !wb.IsBusy() {
		t.Fatal("restore should still be awaiting UI completion")
	}
	wb.editTitle = "user edits while restore is busy"
	saveErr := wb.SaveSelectedTask()
	wb.RunDueBackupNow()
	exportErr := wb.ExportManualBackup(filepath.Join(t.TempDir(), "blocked.zip"))
	wb.NavigateTo(NavAllProjects)
	(<-dispatched)()
	if exportErr == nil {
		t.Fatal("expected ExportManualBackup to be blocked while restore is awaiting UI completion")
	}
	if wb.selectedTaskID != "" || wb.dirty || wb.editDescription != "" {
		t.Fatalf("expected restore completion to clear pre-restore draft state, got selected=%q dirty=%v desc=%q", wb.selectedTaskID, wb.dirty, wb.editDescription)
	}
	restored, err := app.Projects.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Description != "snapshot description" {
		t.Fatalf("restored data overwritten by old UI state: save error=%v, description=%q", saveErr, restored.Description)
	}
}

// Regression: board reloads previously decoded all project images eagerly and retained bitmaps after tasks were purged.
func TestThirdReviewDeletedProjectImagesMustLeaveCache(t *testing.T) {
	app, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "image cache"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "unopened image task")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "photo.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2048, 2048))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := app.Attachments.Import(context.Background(), task.ID, path); err != nil {
			t.Fatal(err)
		}
	}
	// Refresh a board without opening the task or displaying a single image.
	wb.ReloadBoardTasks()
	var decodedBytes int
	for _, bmp := range wb.bitmapCache {
		width, height := bmp.Size()
		decodedBytes += width * height * 4
	}
	if len(wb.bitmapCache) != 0 {
		t.Fatalf("unopened images should not be eagerly decoded in ReloadBoardTasks: cached=%d, RGBA bytes=%d", len(wb.bitmapCache), decodedBytes)
	}

	// Even after rendering/decoding an image, deleting the task and clearing trash must evict cached bitmaps.
	wb.SelectTask(task.ID)
	_ = wb.AttachmentBitmap(wb.detailAttachments[0])
	if len(wb.bitmapCache) != 1 {
		t.Fatalf("expected 1 on-demand cached bitmap, got %d", len(wb.bitmapCache))
	}
	wb.SoftDeleteTask(task.ID)
	wb.ConfirmClearTrash()
	entries, err := os.ReadDir(filepath.Join(app.DataDir(), "attachments"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("expected GC to remove attachments")
	}
	if len(wb.bitmapCache) != 0 {
		t.Fatalf("permanently deleted images still retained: cached=%d", len(wb.bitmapCache))
	}
}

// Regression: hovering over a sidebar project item with progress badges previously caused 0/1 text to bleed through action icons.
func TestSidebarProjectHoverActions(t *testing.T) {
	dataDir := t.TempDir()
	_, wb, tt := openTestWorkbench(t, dataDir, Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "测试项目"
	wb.SubmitProjectModal()
	_, err := wb.QuickCreateTask(model.Planned, "任务 1")
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	// Find the project row in the sidebar and move pointer over it
	rect, found := tt.Find("测试项目")
	if !found {
		t.Fatal("could not find 测试项目 in accessibility tree")
	}
	tt.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()

	// Verify clicking the hover action works
	if err := tt.Click("编辑项目: 测试项目"); err != nil {
		t.Fatalf("click edit project icon: %v", err)
	}
	if !wb.projectModalOpen || wb.projectModalEditID != wb.activeProjectID {
		t.Fatalf("expected project modal open for %s, got open=%v editID=%s", wb.activeProjectID, wb.projectModalOpen, wb.projectModalEditID)
	}
}

// Regression: toasts enqueued by a frame's click handler previously stayed queued past ToastHost instead of rendering immediately.
func TestSameFrameToastRendersImmediately(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})

	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	if err := tt.Click("复制路径"); err != nil {
		t.Fatalf("click 复制路径: %v", err)
	}
	if got := tt.Clipboard(); got != wb.app.DataDir() {
		t.Fatalf("expected clipboard %q, got %q", wb.app.DataDir(), got)
	}
	if len(wb.pendingToasts) != 0 {
		t.Fatalf("expected pendingToasts to drain before ToastHost in same frame, got %d remaining", len(wb.pendingToasts))
	}
	if !tt.HasText("已复制数据目录路径") {
		t.Fatalf("expected copy toast to render in the same frame")
	}
}

// Regression: RestoreOrphanTaskWithParent previously ignored task-restore failures and reported success after restoring only the parent project.
func TestRestoreOrphanTaskWithParentSurfacesTaskError(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "孤儿恢复项目"
	wb.SubmitProjectModal()
	projID := wb.activeProjectID

	wb.SoftDeleteProject(projID)
	wb.NavigateTo(NavTrash)
	tt.Frame()

	// Point orphan modal at a deleted parent project with a missing task ID so parent restore succeeds and task restore fails.
	wb.trashOrphanTask = &model.TrashItem{
		ID:        "missing-task-id",
		Kind:      projects.TrashTask,
		Name:      "不存在的任务",
		ProjectID: projID,
	}
	wb.trashParentErrModalOpen = true
	tt.Frame()

	if err := tt.Click("一并恢复项目与任务"); err != nil {
		t.Fatalf("click 一并恢复项目与任务: %v", err)
	}
	if wb.trashParentErrModalOpen || wb.trashOrphanTask != nil {
		t.Fatalf("expected orphan modal to close after parent project restored")
	}
	if _, err := app.Projects.GetProject(context.Background(), projID); err != nil {
		t.Fatalf("expected parent project to be restored, got err=%v", err)
	}
	if !tt.HasText("恢复任务失败（所属项目已恢复）") {
		t.Fatalf("expected error toast when restoring task failed")
	}
	if tt.HasText("已恢复项目与任务") {
		t.Fatalf("must not show success toast when task restore failed")
	}
}

// Regression: ConfirmPermanentDelete and ConfirmClearTrash previously cleared confirmation modal state even when rejected because another task was busy.
func TestConfirmPermanentDeleteRetainsModalWhenBusy(t *testing.T) {
	dispatched := make(chan func(), 16)
	app, wb, _ := openTestWorkbench(t, t.TempDir(), Host{
		Dispatch: func(fn func()) {
			dispatched <- fn
		},
	})

	wb.OpenCreateProjectModal()
	wb.projectModalName = "忙碌删除保护"
	wb.SubmitProjectModal()
	projID := wb.activeProjectID
	wb.SoftDeleteProject(projID)

	started := make(chan struct{})
	release := make(chan struct{})
	releaseOnce := false
	t.Cleanup(func() {
		if !releaseOnce {
			close(release)
		}
	})
	if !wb.runAsyncTask("阻塞后台任务", func(context.Context) func() {
		close(started)
		<-release
		return nil
	}) {
		t.Fatal("failed to start blocking async task")
	}
	<-started

	item := wb.trashItems[0]
	wb.confirmPermDeleteItem = &item
	wb.confirmPermDeleteOpen = true
	wb.ConfirmPermanentDelete()
	if !wb.confirmPermDeleteOpen || wb.confirmPermDeleteItem == nil || wb.confirmPermDeleteItem.ID != item.ID {
		t.Fatalf("expected permanent delete modal state to remain intact while busy")
	}

	wb.confirmClearTrashOpen = true
	wb.ConfirmClearTrash()
	if !wb.confirmClearTrashOpen {
		t.Fatalf("expected clear trash modal state to remain intact while busy")
	}

	releaseOnce = true
	close(release)
	wb.WaitForTasks()
	for len(dispatched) > 0 {
		(<-dispatched)()
	}

	// Once idle, ConfirmPermanentDelete executes and prunes projectNames.
	wb.ConfirmPermanentDelete()
	wb.WaitForTasks()
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	if wb.confirmPermDeleteOpen || wb.confirmPermDeleteItem != nil {
		t.Fatalf("expected permanent delete modal state cleared after execution")
	}
	if _, exists := wb.projectNames[projID]; exists {
		t.Fatalf("expected permanently deleted project ID %s to be pruned from projectNames", projID)
	}
	if items, err := app.Projects.ListTrash(context.Background()); err != nil || len(items) != 0 {
		t.Fatalf("expected trash empty after permanent delete, got items=%+v err=%v", items, err)
	}
}

// Regression: applyRestoredState previously leaked pre-restore projectNames, filters, modals, timestamps, and stale board tasks when active projects were empty.
func TestApplyRestoredStateResetsPreRestoreWorkbenchState(t *testing.T) {
	app, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})

	// Create an archived-only backup (0 active projects) so restore tests empty-active-project cleanup.
	wb.OpenCreateProjectModal()
	wb.projectModalName = "归档快照项目"
	wb.SubmitProjectModal()
	archivedProjID := wb.activeProjectID
	if _, err := wb.QuickCreateTask(model.Planned, "归档项目任务"); err != nil {
		t.Fatal(err)
	}
	wb.ClearSelectedTask()
	wb.ArchiveProject(archivedProjID)

	zipPath := filepath.Join(t.TempDir(), "archived-only.zip")
	if err := wb.ExportManualBackup(zipPath); err != nil {
		t.Fatal(err)
	}

	// Pollute workbench state with another project, filters, modals, timestamps, and async errors before restore.
	wb.OpenCreateProjectModal()
	wb.projectModalName = "恢复前临时项目"
	wb.SubmitProjectModal()
	tempProjID := wb.activeProjectID
	wb.searchQuery = "stale-query"
	wb.priorityFilter = string(model.High)
	wb.tagFilters = []string{"stale-tag"}
	wb.quickTaskTitles[model.Planned] = "draft quick task"
	wb.trashSearch = "stale-trash"
	wb.trashKindFilter = projects.TrashTask
	wb.projectModalOpen = true
	wb.taskModalOpen = true
	wb.trashParentErrModalOpen = true
	wb.confirmClearTrashOpen = true
	wb.lastSavedAt = time.Now()
	wb.RecordAsyncError(errors.New("pre-restore async error"))

	// Make old active project match archivedProjID to test switching to 0 active projects cleanly.
	wb.SelectProject(archivedProjID)
	wb.RequestRestoreBackup(zipPath)
	wb.TriggerRestoreBackup()

	if wb.activeProjectID != "" {
		t.Fatalf("expected empty activeProjectID when restored backup has 0 active projects, got %q", wb.activeProjectID)
	}
	if len(wb.boardTasks) != 0 || len(wb.filteredTasks) != 0 || len(wb.allBoardTags) != 0 || len(wb.taskAttachments) != 0 {
		t.Fatalf("expected board task caches cleared on empty-active restore, got board=%d filtered=%d", len(wb.boardTasks), len(wb.filteredTasks))
	}
	if _, leaked := wb.projectNames[tempProjID]; leaked {
		t.Fatalf("pre-restore project name %s leaked into restored projectNames", tempProjID)
	}
	if wb.projectNames[archivedProjID] != "归档快照项目" {
		t.Fatalf("expected restored archived project in projectNames, got %+v", wb.projectNames)
	}
	if wb.searchQuery != "" || wb.priorityFilter != "" || len(wb.tagFilters) != 0 || len(wb.quickTaskTitles) != 0 {
		t.Fatalf("expected board filters and quick inputs reset after restore")
	}
	if wb.trashSearch != "" || wb.trashKindFilter != "" || wb.pendingRestoreZip != "" {
		t.Fatalf("expected trash filters and pendingRestoreZip reset after restore")
	}
	if wb.projectModalOpen || wb.taskModalOpen || wb.trashParentErrModalOpen || wb.confirmClearTrashOpen || wb.confirmRestoreOpen {
		t.Fatalf("expected all modals closed after restore")
	}
	if !wb.lastSavedAt.IsZero() || wb.LastAsyncError() != "" {
		t.Fatalf("expected lastSavedAt and lastAsyncErr cleared after restore, got savedAt=%v err=%q", wb.lastSavedAt, wb.LastAsyncError())
	}
	if _, err := app.Projects.GetProject(context.Background(), tempProjID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected pre-restore project to be absent in restored DB, got err=%v", err)
	}
}

// Regression: Settings previously kept showing a stale async backup error banner after RunDueBackupNow succeeded.
func TestRunDueBackupNowClearsLastAsyncError(t *testing.T) {
	dispatched := make(chan func(), 8)
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		Dispatch: func(fn func()) {
			dispatched <- fn
		},
	})

	wb.NavigateTo(NavSettings)
	wb.RecordAsyncError(errors.New("simulated scheduled backup failure"))
	tt.Frame()
	if !tt.HasText("最近一次自动备份失败") {
		t.Fatalf("expected Settings to display async backup failure banner")
	}

	if err := tt.Click("立即检查到期备份"); err != nil {
		t.Fatalf("click 立即检查到期备份: %v", err)
	}
	wb.WaitForTasks()
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	tt.Frame()

	if wb.LastAsyncError() != "" {
		t.Fatalf("expected lastAsyncErr cleared after successful RunDueBackupNow, got %q", wb.LastAsyncError())
	}
	if tt.HasText("最近一次自动备份失败") {
		t.Fatalf("expected failure banner to disappear after RunDueBackupNow succeeded")
	}
}

// Regression: MoveOrReorderTaskOnto and StepTaskInColumn previously failed silently on invalid drafts, and FlushPending errors without saveErr produced empty messages.
func TestDirtyDraftBlocksReorderAndCardDropWithToast(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "排序草稿保护"
	wb.SubmitProjectModal()

	t1, err := wb.QuickCreateTask(model.Planned, "任务一")
	if err != nil {
		t.Fatal(err)
	}
	t2, err := wb.QuickCreateTask(model.Planned, "任务二")
	if err != nil {
		t.Fatal(err)
	}

	wb.SelectTask(t2.ID)
	wb.editTitle = "   "
	wb.detailExpanded = false
	wb.pendingToasts = nil

	wb.StepTaskInColumn(t2.ID, model.Planned, -1)
	if !wb.detailExpanded {
		t.Fatalf("expected StepTaskInColumn flush failure to expand detail panel")
	}
	planned := wb.TasksInColumn(model.Planned)
	if planned[0].ID != t1.ID || planned[1].ID != t2.ID {
		t.Fatalf("expected StepTaskInColumn not to reorder while draft is invalid")
	}
	tt.Frame()
	if !tt.HasText("任务标题不能为空") {
		t.Fatalf("expected warning toast with validation reason on StepTaskInColumn failure")
	}

	wb.detailExpanded = false
	wb.MoveOrReorderTaskOnto(t2.ID, t1.ID, false)
	if !wb.detailExpanded {
		t.Fatalf("expected MoveOrReorderTaskOnto flush failure to expand detail panel")
	}
	planned = wb.TasksInColumn(model.Planned)
	if planned[0].ID != t1.ID || planned[1].ID != t2.ID {
		t.Fatalf("expected MoveOrReorderTaskOnto not to reorder while draft is invalid")
	}

	// Verify FlushPending ErrConflict without saveErr surfaces a non-empty conflict message.
	wb.editTitle = "有效标题"
	wb.saveErr = ""
	wb.bumpDataEpoch()
	wb.MoveOrReorderTaskOnto(t2.ID, t1.ID, false)
	tt.Frame()
	if !tt.HasText("backup restore in progress") {
		t.Fatalf("expected FlushPending ErrConflict fallback description instead of empty saveErr")
	}
}

// Regression: failing to persist a theme setting previously left themeSegmentIdx out of sync with themeSetting.
func TestApplyThemeSettingErrorRevertsSegmentedControl(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.NavigateTo(NavSettings)
	tt.Frame()

	if err := app.Close(); err != nil {
		t.Fatalf("close backend: %v", err)
	}

	if err := tt.Click("浅色"); err != nil {
		t.Fatalf("click 浅色: %v", err)
	}
	if wb.themeSetting != ThemeSystem {
		t.Fatalf("expected themeSetting to remain %q on error, got %q", ThemeSystem, wb.themeSetting)
	}
	if wb.themeSegmentIdx != 0 {
		t.Fatalf("expected themeSegmentIdx to revert to 0 on error, got %d", wb.themeSegmentIdx)
	}
	if !tt.HasText("保存主题设置失败") {
		t.Fatalf("expected error toast when saving theme fails")
	}
}

// Real async attachment import, picker errors, and manual backup export run via Host.Dispatch while rejecting concurrent busy triggers.
func TestAsyncAttachmentImportAndBackupExportDispatchFlow(t *testing.T) {
	srcDir := t.TempDir()
	docPath := filepath.Join(srcDir, "spec.txt")
	if err := os.WriteFile(docPath, []byte("async spec content"), 0o600); err != nil {
		t.Fatal(err)
	}
	exportZip := filepath.Join(t.TempDir(), "async-export.zip")

	dispatched := make(chan func(), 16)
	pickerGate := make(chan struct{})
	pickerErr := errors.New("picker unavailable")
	returnPickerErr := false

	host := Host{
		Dispatch: func(fn func()) {
			dispatched <- fn
		},
		PickOpenFiles: func(_ string, _ bool, _ ...string) ([]string, error) {
			<-pickerGate
			if returnPickerErr {
				return nil, pickerErr
			}
			return []string{docPath}, nil
		},
		PickSaveFile: func(_ string, _ string, _ ...string) (string, error) {
			return exportZip, nil
		},
	}

	_, wb, tt := openTestWorkbench(t, t.TempDir(), host)
	wb.OpenCreateProjectModal()
	wb.projectModalName = "异步分发项目"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "异步附件任务")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)

	// Start real async attachment pick & import.
	wb.PickAndImportAttachmentsToSelectedTask()
	if !wb.IsBusy() {
		t.Fatal("expected workbench busy during async file picker/import")
	}

	// Concurrent async export must be rejected while busy without blocking UI.
	wb.PickAndExportManualBackup()
	if len(wb.pendingToasts) == 0 || wb.pendingToasts[len(wb.pendingToasts)-1].Title != "后台任务进行中" {
		t.Fatalf("expected busy rejection toast, got %+v", wb.pendingToasts)
	}
	wb.pendingToasts = nil

	close(pickerGate)
	wb.WaitForTasks()
	if len(wb.detailAttachments) != 0 {
		t.Fatal("background worker updated UI attachments before Dispatch callback ran")
	}
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	if wb.IsBusy() {
		t.Fatal("expected busy cleared after dispatched import apply")
	}
	if len(wb.detailAttachments) != 1 || wb.detailAttachments[0].Name != "spec.txt" {
		t.Fatalf("expected 1 imported attachment after dispatch, got %+v", wb.detailAttachments)
	}

	// Picker error must surface a toast instead of silently returning nil.
	returnPickerErr = true
	wb.PickAndImportAttachmentsToSelectedTask()
	wb.WaitForTasks()
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	tt.Frame()
	if !tt.HasText("选择附件失败") || !tt.HasText(pickerErr.Error()) {
		t.Fatalf("expected picker error toast to surface")
	}

	// Async manual backup export writes archive off UI thread and updates state on Dispatch.
	wb.PickAndExportManualBackup()
	wb.WaitForTasks()
	if wb.lastExportedZipPath != "" {
		t.Fatal("lastExportedZipPath updated before Dispatch callback ran")
	}
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	if wb.lastExportedZipPath != exportZip {
		t.Fatalf("expected lastExportedZipPath=%q, got %q", exportZip, wb.lastExportedZipPath)
	}
	if _, err := os.Stat(exportZip); err != nil {
		t.Fatalf("expected exported zip on disk: %v", err)
	}
	// Verify AttachmentManagedPath missing cache returns empty string without error when file is absent.
	missingAtt := model.Attachment{ID: "missing-att", StorageName: "does-not-exist"}
	if got := wb.AttachmentManagedPath(missingAtt); got != "" || !wb.attachmentMissing["missing-att"] {
		t.Fatalf("expected missing attachment path cached as missing, got %q missing=%v", got, wb.attachmentMissing["missing-att"])
	}
	if !strings.HasSuffix(wb.attachmentPaths[wb.detailAttachments[0].ID], wb.detailAttachments[0].StorageName) {
		t.Fatalf("expected imported attachment path cached")
	}
}

// Dragging a card onto its own column background ignores same-status drops while dropping onto another card reorders cleanly.
func TestBoardColumnIgnoresSameStatusDropWhileCardReorders(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "同列拖拽测试项目"
	wb.SubmitProjectModal()

	t1, err := wb.QuickCreateTask(model.Planned, "卡片一")
	if err != nil {
		t.Fatalf("create t1: %v", err)
	}
	t2, err := wb.QuickCreateTask(model.Planned, "卡片二")
	if err != nil {
		t.Fatalf("create t2: %v", err)
	}
	t3, err := wb.QuickCreateTask(model.Planned, "卡片三")
	if err != nil {
		t.Fatalf("create t3: %v", err)
	}
	wb.ClearSelectedTask()
	tt.Frame()

	drag := func(fromX, fromY, toX, toY float32) {
		tt.Press(fromX, fromY)
		for step := 1; step <= 4; step++ {
			f := float32(step) / 4
			tt.Move(fromX+(toX-fromX)*f, fromY+(toY-fromY)*f)
		}
		tt.Release(toX, toY)
	}

	card1Rect, ok := tt.Find("任务卡片 卡片一")
	if !ok {
		t.Fatal("could not find 卡片一")
	}
	plannedColRect, ok := tt.Find("看板列 计划")
	if !ok {
		t.Fatal("could not find 看板列 计划")
	}

	// Dropping onto the source column header/whitespace must not jump t1 to the end.
	drag(card1Rect.X+card1Rect.W/2, card1Rect.Y+16, plannedColRect.X+plannedColRect.W/2, plannedColRect.Y+14)
	planned := wb.TasksInColumn(model.Planned)
	if len(planned) != 3 || planned[0].ID != t1.ID || planned[1].ID != t2.ID || planned[2].ID != t3.ID {
		t.Fatalf("expected same-column drop on column box to keep order [%s %s %s], got %+v", t1.ID, t2.ID, t3.ID, planned)
	}

	// Dropping onto the bottom half of t2 still reorders t1 after t2 within the same column.
	// Hover card1 first (as a user's pointer does before pressing) so card2's target rect reflects the hovered layout.
	tt.Move(card1Rect.X+card1Rect.W/2, card1Rect.Y+16)
	card2Rect, ok := tt.Find("任务卡片 卡片二")
	if !ok {
		t.Fatal("could not find 卡片二")
	}
	drag(card1Rect.X+card1Rect.W/2, card1Rect.Y+16, card2Rect.X+card2Rect.W/2, card2Rect.Y+card2Rect.H*0.75)
	planned = wb.TasksInColumn(model.Planned)
	if len(planned) != 3 || planned[0].ID != t2.ID || planned[1].ID != t1.ID || planned[2].ID != t3.ID {
		t.Fatalf("expected card drop onto bottom half of t2 to reorder [%s %s %s], got %+v", t2.ID, t1.ID, t3.ID, planned)
	}
}

// Switching tasks, trashing via consistent labels, filtering trash, and triggering restore from Settings maintain consistent state and errors.
func TestCrossViewConsistencyFixes(t *testing.T) {
	pickErr := errors.New("dialog unavailable")
	host := Host{
		PickOpenFiles: func(_ string, _ bool, _ ...string) ([]string, error) {
			return nil, pickErr
		},
	}
	_, wb, tt := openTestWorkbench(t, t.TempDir(), host)

	// Empty active project count must not render a standalone zero badge on 所有项目.
	tt.Frame()
	if slices.Contains(tt.Texts(), "0") {
		t.Fatalf("expected empty 所有项目 not to display standalone zero badge, got texts %v", tt.Texts())
	}

	wb.OpenCreateProjectModal()
	wb.projectModalName = "一致性测试项目"
	wb.SubmitProjectModal()

	t1, err := wb.QuickCreateTask(model.Planned, "任务甲")
	if err != nil {
		t.Fatalf("create t1: %v", err)
	}
	t2, err := wb.QuickCreateTask(model.Planned, "任务乙")
	if err != nil {
		t.Fatalf("create t2: %v", err)
	}

	// Saving t1 sets lastSavedAt, which must reset when switching to t2 or closing detail panel.
	wb.SelectTask(t1.ID)
	wb.editTitle = "任务甲-已编辑"
	if err := wb.SaveSelectedTask(); err != nil {
		t.Fatalf("save t1: %v", err)
	}
	if wb.lastSavedAt.IsZero() {
		t.Fatal("expected lastSavedAt populated after SaveSelectedTask")
	}
	wb.SelectTask(t2.ID)
	if !wb.lastSavedAt.IsZero() {
		t.Fatalf("expected lastSavedAt cleared when switching to another task, got %v", wb.lastSavedAt)
	}
	wb.editTitle = "任务乙-已编辑"
	if err := wb.SaveSelectedTask(); err != nil {
		t.Fatalf("save t2: %v", err)
	}
	wb.ClearSelectedTask()
	if !wb.lastSavedAt.IsZero() {
		t.Fatalf("expected lastSavedAt cleared when closing detail panel, got %v", wb.lastSavedAt)
	}

	// Settings restore button surfaces picker errors and FlushPending dirty draft warnings consistently.
	wb.SelectTask(t1.ID)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	wb.editTitle = "   "
	_ = wb.SaveSelectedTask()
	wb.detailExpanded = false
	if err := tt.Click("从备份包恢复 (.zip)..."); err != nil {
		t.Fatalf("click restore button while dirty: %v", err)
	}
	if !tt.HasText("当前任务存在未保存修改") || !wb.detailExpanded {
		t.Fatalf("expected FlushPending dirty warning toast and detailExpanded=true")
	}
	wb.DiscardSelectedTaskDraft()
	if err := tt.Click("从备份包恢复 (.zip)..."); err != nil {
		t.Fatalf("click restore button with picker error: %v", err)
	}
	if !tt.HasText("选择备份包失败") || !tt.HasText(pickErr.Error()) {
		t.Fatalf("expected restore picker error toast to surface")
	}

	// Soft-deleting task and project uses consistent 移入回收站 buttons, and Trash filter bar clears cleanly.
	if err := tt.Click("一致性测试项目"); err != nil {
		t.Fatalf("click project in sidebar: %v", err)
	}
	wb.SelectTask(t1.ID)
	tt.Frame()
	if err := tt.Click("将任务移入回收站"); err != nil {
		t.Fatalf("click 将任务移入回收站: %v", err)
	}
	if err := tt.Click("将项目移入回收站"); err != nil {
		t.Fatalf("click 将项目移入回收站: %v", err)
	}
	for tt.Click("Close") == nil {
	}
	if err := tt.Click("回收站"); err != nil {
		t.Fatalf("click 回收站: %v", err)
	}
	if err := tt.Click("搜索回收站"); err != nil {
		t.Fatalf("click 搜索回收站: %v", err)
	}
	tt.Type("任务甲")
	if !tt.HasText("匹配 1 / 2 项") {
		t.Fatalf("expected trash filter summary 匹配 1 / 2 项, got texts %v", tt.Texts())
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("click trash 清除筛选: %v", err)
	}
	if wb.trashSearch != "" || !tt.HasText("共 2 项") {
		t.Fatalf("expected trash filter cleared and 共 2 项 displayed, got search=%q texts=%v", wb.trashSearch, tt.Texts())
	}
}

// Settings displays app version and update status from Host and triggers update checks and release links through Host stubs.
func TestUpdaterSettingsAndHostIntegration(t *testing.T) {
	checkCalls := 0
	var openedURLs []string
	var openErr error

	host := Host{
		UpdateInfo: UpdateCapability{
			Version:       "0.1.0",
			IsDev:         false,
			CanSelfUpdate: true,
			ReleasesURL:   "https://github.com/Games55k/Aion/releases",
		},
		CheckForUpdates: func() {
			checkCalls++
		},
		OpenURL: func(rawURL string) error {
			openedURLs = append(openedURLs, rawURL)
			return openErr
		},
	}
	_, wb, tt := openTestWorkbench(t, t.TempDir(), host)

	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	tt.Scroll(500, 300, 0, 500)
	tt.Frame()

	if !tt.HasText("v0.1.0") {
		t.Fatalf("expected current version v0.1.0 in Settings, got texts %v", tt.Texts())
	}
	if err := tt.Click("检查更新"); err != nil {
		t.Fatalf("click 检查更新: %v", err)
	}
	if checkCalls != 1 {
		t.Fatalf("expected Host.CheckForUpdates called once, got %d", checkCalls)
	}

	if err := tt.Click("前往 Releases 下载"); err != nil {
		t.Fatalf("click 前往 Releases 下载: %v", err)
	}
	if !slices.Equal(openedURLs, []string{"https://github.com/Games55k/Aion/releases"}) {
		t.Fatalf("unexpected opened URLs: %v", openedURLs)
	}

	// Discovered version and last-check timestamp reflect on the next frame.
	wb.SetUpdateCapability(UpdateCapability{
		Version:          "0.1.0",
		AvailableVersion: "0.2.0",
		LastCheck:        time.Date(2026, 11, 11, 15, 30, 0, 0, time.Local),
		IsDev:            false,
		CanSelfUpdate:    true,
		ReleasesURL:      "https://github.com/Games55k/Aion/releases",
	})
	tt.Frame()
	if !tt.HasText("发现新版本 v0.2.0") || !tt.HasText("2026-11-11 15:30:00") {
		t.Fatalf("expected available version and last check time in Settings, got texts %v", tt.Texts())
	}

	// Non-writable install location explains why self-update is unavailable.
	wb.SetUpdateCapability(UpdateCapability{
		Version:       "0.1.0",
		IsDev:         false,
		CanSelfUpdate: false,
		ReleasesURL:   "https://github.com/Games55k/Aion/releases",
	})
	tt.Frame()
	if !tt.HasText("当前安装位置不可写，无法应用内自动更新") {
		t.Fatalf("expected non-writable install notice in Settings, got texts %v", tt.Texts())
	}

	// Development builds explain why in-app updates are disabled.
	wb.SetUpdateCapability(UpdateCapability{
		Version:       "0.1.0",
		IsDev:         true,
		CanSelfUpdate: false,
		ReleasesURL:   "https://github.com/Games55k/Aion/releases",
	})
	tt.Frame()
	if !tt.HasText("当前为开发构建，不支持应用内自动更新") {
		t.Fatalf("expected development build notice in Settings, got texts %v", tt.Texts())
	}

	// OpenURL error surfaces a toast instead of dropping silently.
	openErr = errors.New("browser unavailable")
	if err := tt.Click("前往 Releases 下载"); err != nil {
		t.Fatalf("click 前往 Releases 下载 on error: %v", err)
	}
	if !tt.HasText("打开发布页失败") || !tt.HasText(openErr.Error()) {
		t.Fatalf("expected OpenReleasesPage error toast, got texts %v", tt.Texts())
	}
}

// Update capability changes and Settings rendering must complete immediately while the Store exclusive lock is held.
func TestUpdateStateChangeDoesNotWaitForStoreLock(t *testing.T) {
	checkCalls := 0
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		UpdateInfo: UpdateCapability{
			Version:       "0.1.0",
			CanSelfUpdate: true,
			ReleasesURL:   defaultReleasesURL,
		},
		CheckForUpdates: func() { checkCalls++ },
		OpenURL:         func(string) error { return nil },
	})

	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	app.Backups = backup.New(s)
	app.Attachments = attachments.New(s)

	locked := make(chan struct{})
	release := make(chan struct{})
	lockDone := make(chan error, 1)
	go func() {
		lockDone <- s.Exclusive(context.Background(), func(_ *sql.DB) error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	wb.navPage = NavSettings
	tt.Frame()
	tt.Scroll(500, 300, 0, 500)
	wb.SetUpdateCapability(UpdateCapability{
		Version:          "0.1.0",
		AvailableVersion: "0.2.0",
		LastCheck:        time.Date(2026, 11, 11, 16, 0, 0, 0, time.Local),
		CanSelfUpdate:    true,
		ReleasesURL:      defaultReleasesURL,
	})
	tt.Frame()
	if !tt.HasText("发现新版本 v0.2.0") {
		t.Fatalf("expected updated version state rendered while Store lock is held")
	}
	if err := tt.Click("检查更新"); err != nil {
		t.Fatalf("click 检查更新 while Store lock is held: %v", err)
	}
	if checkCalls != 1 {
		t.Fatalf("expected CheckForUpdates invoked while Store lock is held, got %d", checkCalls)
	}

	close(release)
	if err := <-lockDone; err != nil {
		t.Fatal(err)
	}
}

// PrepareClose blocks relaunch or quit when unsaved task edits fail validation and permits closing once fixed and saved.
func TestPrepareCloseBlocksInvalidDraftAndAllowsSaveThenClose(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "退出保存校验"
	wb.SubmitProjectModal()

	task, err := wb.QuickCreateTask(model.Planned, "初始任务标题")
	if err != nil {
		t.Fatalf("QuickCreateTask: %v", err)
	}

	wb.SelectTask(task.ID)
	wb.editTitle = "   "
	wb.editDescription = "修复标题后必须持久化的任务内容"
	wb.NavigateTo(NavSettings)
	if wb.navPage != NavProjectBoard {
		t.Fatalf("expected navigation blocked on invalid draft, got %s", wb.navPage)
	}
	wb.detailExpanded = false

	if wb.PrepareClose() {
		t.Fatalf("expected PrepareClose to block closing/relaunch when unsaved draft fails validation")
	}
	if wb.navPage != NavProjectBoard || !wb.detailExpanded {
		t.Fatalf("expected PrepareClose to reveal task detail panel, got page=%s detailExpanded=%v", wb.navPage, wb.detailExpanded)
	}
	tt.Frame()
	if !tt.HasText("存在未保存的任务修改") {
		t.Fatalf("expected warning toast when PrepareClose rejects invalid draft")
	}

	// Fix the invalid title and retry PrepareClose; pending edits must flush cleanly and allow quit/relaunch.
	wb.editTitle = "修复后的任务标题"
	if !wb.PrepareClose() {
		t.Fatalf("expected PrepareClose to succeed once invalid draft is fixed")
	}

	persisted, err := app.Projects.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask after PrepareClose: %v", err)
	}
	if persisted.Title != "修复后的任务标题" || persisted.Description != "修复标题后必须持久化的任务内容" {
		t.Fatalf("expected pending edits persisted on PrepareClose, got %+v", persisted)
	}
}

// PrepareClose rejects closing while an async backup or attachment task is active or awaiting UI dispatch and allows closing once drained.
func TestPrepareCloseWithActiveBackupOrAttachmentTask(t *testing.T) {
	backupDir := t.TempDir()
	backupDest := filepath.Join(backupDir, "close-guard-backup.zip")
	srcAtt := filepath.Join(t.TempDir(), "close-guard.txt")
	if err := os.WriteFile(srcAtt, []byte("attachment payload"), 0o600); err != nil {
		t.Fatalf("write attachment file: %v", err)
	}

	dispatched := make(chan func(), 16)
	saveStarted := make(chan struct{})
	releaseSave := make(chan struct{})
	openStarted := make(chan struct{})
	releaseOpen := make(chan struct{})

	host := Host{
		Dispatch: func(fn func()) {
			dispatched <- fn
		},
		PickSaveFile: func(_, _ string, _ ...string) (string, error) {
			close(saveStarted)
			<-releaseSave
			return backupDest, nil
		},
		PickOpenFiles: func(_ string, _ bool, _ ...string) ([]string, error) {
			close(openStarted)
			<-releaseOpen
			return []string{srcAtt}, nil
		},
	}

	_, wb, _ := openTestWorkbench(t, t.TempDir(), host)
	wb.OpenCreateProjectModal()
	wb.projectModalName = "关闭任务保护"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "附件与备份任务")
	if err != nil {
		t.Fatalf("QuickCreateTask: %v", err)
	}
	wb.SelectTask(task.ID)

	// 1. Active async backup task blocks PrepareClose until worker finishes and Dispatch callback drains.
	wb.PickAndExportManualBackup()
	<-saveStarted
	if wb.PrepareClose() {
		t.Fatal("PrepareClose must return false while backup worker is running")
	}
	close(releaseSave)
	wb.WaitForTasks()
	if wb.PrepareClose() {
		t.Fatal("PrepareClose must return false before queued backup UI Dispatch callback runs")
	}
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	if !wb.PrepareClose() {
		t.Fatal("PrepareClose must return true after backup task and UI callback complete")
	}
	if _, err := os.Stat(backupDest); err != nil {
		t.Fatalf("expected exported backup zip at %s: %v", backupDest, err)
	}

	// 2. Active async attachment import task also blocks PrepareClose until worker and Dispatch drain.
	wb.PickAndImportAttachmentsToSelectedTask()
	<-openStarted
	if wb.PrepareClose() {
		t.Fatal("PrepareClose must return false while attachment import worker is running")
	}
	close(releaseOpen)
	wb.WaitForTasks()
	if wb.PrepareClose() {
		t.Fatal("PrepareClose must return false before queued attachment UI Dispatch callback runs")
	}
	for len(dispatched) > 0 {
		(<-dispatched)()
	}
	if !wb.PrepareClose() {
		t.Fatal("PrepareClose must return true after attachment import and UI callback complete")
	}
	if len(wb.detailAttachments) != 1 || wb.detailAttachments[0].Name != "close-guard.txt" {
		t.Fatalf("expected imported attachment recorded after dispatch, got %+v", wb.detailAttachments)
	}
}

// Regression: a failing backup directory picker was silently ignored while every other native picker reported its error.
func TestBackupDirectoryPickerErrorIsReported(t *testing.T) {
	pickErr := errors.New("directory picker unavailable")
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		PickDirectory: func(string, string) (string, error) { return "", pickErr },
	})
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	before := wb.backupDirInput
	if err := tt.Click("选择目录..."); err != nil {
		t.Fatalf("click 选择目录...: %v", err)
	}
	if !tt.HasText("选择备份目录失败") || !tt.HasText(pickErr.Error()) {
		t.Fatalf("expected directory picker error toast")
	}
	if wb.backupDirInput != before {
		t.Fatalf("backup directory changed after picker error: %q", wb.backupDirInput)
	}
}

// Typing in the board search bar and toggling filters refilters the loaded board without waiting for the Store lock.
func TestBoardFiltersDoNotWaitForStoreLock(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "filter lock check"
	wb.SubmitProjectModal()
	ctx := context.Background()
	for _, in := range []projects.TaskInput{
		{Title: "alpha", Tags: []string{"go"}, Priority: model.High},
		{Title: "beta", Tags: []string{"go", "ui"}, Priority: model.Low},
	} {
		if _, err := app.Projects.CreateTask(ctx, wb.activeProjectID, in); err != nil {
			t.Fatal(err)
		}
	}
	wb.ReloadBoardTasks()
	tt.Frame()

	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	app.Projects = projects.New(s)
	app.Attachments = attachments.New(s)
	locked := make(chan struct{})
	release := make(chan struct{})
	lockDone := make(chan error, 1)
	go func() {
		lockDone <- s.Exclusive(ctx, func(_ *sql.DB) error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	if err := tt.Click("搜索任务"); err != nil {
		t.Fatalf("click 搜索任务: %v", err)
	}
	tt.Type("BET")
	if len(wb.filteredTasks) != 1 || wb.filteredTasks[0].Title != "beta" {
		t.Fatalf("search under lock matched %+v", wb.filteredTasks)
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("click 清除筛选: %v", err)
	}
	if err := tt.Click("筛选标签 ui"); err != nil {
		t.Fatalf("click 筛选标签 ui: %v", err)
	}
	if len(wb.filteredTasks) != 1 || wb.filteredTasks[0].Title != "beta" {
		t.Fatalf("tag filter under lock matched %+v", wb.filteredTasks)
	}
	wb.priorityFilter = string(model.High)
	wb.applyBoardFilters()
	if len(wb.filteredTasks) != 0 || len(wb.boardTasks) != 2 {
		t.Fatalf("combined filters under lock: %d visible of %d", len(wb.filteredTasks), len(wb.boardTasks))
	}

	close(release)
	if err := <-lockDone; err != nil {
		t.Fatal(err)
	}
}

// Detail-panel edits patch the loaded board in place, so the cards, tag list, filters, and summary must equal what a full reload from SQLite produces.
func TestDetailEditsKeepBoardInSyncWithoutReload(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "in-place sync"
	wb.SubmitProjectModal()
	ctx := context.Background()
	keep, err := app.Projects.CreateTask(ctx, wb.activeProjectID, projects.TaskInput{Title: "keep", Tags: []string{"shared"}})
	if err != nil {
		t.Fatal(err)
	}
	edited, err := app.Projects.CreateTask(ctx, wb.activeProjectID, projects.TaskInput{Title: "edit me", Tags: []string{"shared", "solo"}})
	if err != nil {
		t.Fatal(err)
	}
	wb.ReloadBoardTasks()
	wb.SelectTask(edited.ID)
	wb.tagFilters = []string{"solo"}
	wb.applyBoardFilters()
	tt.Frame()

	// Typing a title persists every keystroke and updates the card.
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatalf("click 任务详情标题: %v", err)
	}
	tt.Type(" now")
	stored, err := app.Projects.GetTask(ctx, edited.ID)
	if err != nil || stored.Title != "edit me now" {
		t.Fatalf("stored title = %q, %v", stored.Title, err)
	}
	if !tt.HasText("edit me now") || wb.dirty {
		t.Fatalf("board card or save state not updated after typing")
	}

	// Removing the last use of a tag drops it from the tag list and from the active filters.
	if err := tt.Click("移除标签 solo"); err != nil {
		t.Fatalf("click 移除标签 solo: %v", err)
	}
	if len(wb.tagFilters) != 0 || !slices.Equal(wb.allBoardTags, []string{"shared"}) || len(wb.filteredTasks) != 2 {
		t.Fatalf("after tag removal: filters=%v tags=%v visible=%d", wb.tagFilters, wb.allBoardTags, len(wb.filteredTasks))
	}

	// A status change still renumbers columns through a real reload.
	wb.editStatus = string(model.Done)
	if err := wb.SaveSelectedTask(); err != nil {
		t.Fatal(err)
	}
	if sum := wb.projectSummaries[wb.activeProjectID]; sum.Done != 1 || sum.Planned != 1 || sum.Total != 2 {
		t.Fatalf("summary after status change = %+v", sum)
	}

	inPlace := struct {
		board, visible []model.Task
		tags           []string
		sum            ProjectSummary
	}{slices.Clone(wb.boardTasks), slices.Clone(wb.filteredTasks), slices.Clone(wb.allBoardTags), wb.projectSummaries[wb.activeProjectID]}
	wb.ReloadBoardTasks()
	if !reflect.DeepEqual(inPlace.board, wb.boardTasks) || !reflect.DeepEqual(inPlace.visible, wb.filteredTasks) ||
		!slices.Equal(inPlace.tags, wb.allBoardTags) || inPlace.sum != wb.projectSummaries[wb.activeProjectID] {
		t.Fatalf("in-place board state diverged from a full reload")
	}
	if got, err := app.Projects.GetTask(ctx, keep.ID); err != nil || !got.UpdatedAt.Equal(keep.UpdatedAt) {
		t.Fatalf("untouched task was rewritten: %v %v", got.UpdatedAt, err)
	}
}

// Save timestamps come from Host.Now so tests can pin the clock instead of reading wall time.
func TestSaveTimestampUsesHostClock(t *testing.T) {
	fixed := time.Date(2026, 11, 11, 9, 30, 0, 0, time.Local)
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{Now: func() time.Time { return fixed }})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "clock"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "task")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	tt.Frame()
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatalf("click 任务详情标题: %v", err)
	}
	tt.Type("!")
	if !wb.lastSavedAt.Equal(fixed) {
		t.Fatalf("lastSavedAt = %v, want %v", wb.lastSavedAt, fixed)
	}
}

// Regression: a backup interval or retention outside the Settings presets was shown as a preset and rewritten to it whenever any other policy control was saved.
func TestSavingPolicyPreservesNonPresetIntervalAndKeep(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	ctx := context.Background()
	if err := app.Backups.SetPolicy(ctx, backup.Policy{Enabled: true, Interval: 48 * time.Hour, Keep: 9}); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	if err := tt.Click("启用自动备份开关"); err != nil {
		t.Fatalf("click 启用自动备份开关: %v", err)
	}
	got, err := app.Backups.GetPolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled || got.Interval != 48*time.Hour || got.Keep != 9 {
		t.Fatalf("policy after toggling the switch = %+v, want disabled with 48h and keep 9", got)
	}
	// Choosing a preset still replaces the custom value.
	wb.backupIntervalSelect = "6h"
	wb.SaveBackupPolicy()
	if got, err = app.Backups.GetPolicy(ctx); err != nil || got.Interval != 6*time.Hour || got.Keep != 9 {
		t.Fatalf("policy after choosing a preset = %+v, %v", got, err)
	}
}

// Regression: a failed file cleanup after removing an attachment was discarded and reported as a successful cleanup.
func TestRemoveAttachmentReportsFailedCleanup(t *testing.T) {
	dataDir := t.TempDir()
	app, wb, tt := openTestWorkbench(t, dataDir, Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "cleanup"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "with file")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	src := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(src, []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	wb.ImportAttachmentsToSelectedTask([]string{src})
	tt.Frame()

	// A read-only managed directory lets the row be deleted but not the file.
	managed := filepath.Join(dataDir, "attachments")
	if err := os.Chmod(managed, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(managed, 0o700) })
	if err := tt.Click("移除附件 notes.txt"); err != nil {
		t.Fatalf("click 移除附件 notes.txt: %v", err)
	}
	if !tt.HasText("已移除附件，但清理文件失败") {
		t.Fatalf("expected a failed-cleanup warning toast")
	}
	if atts, err := app.Attachments.List(context.Background(), task.ID); err != nil || len(atts) != 0 {
		t.Fatalf("attachment row not removed: %v %v", atts, err)
	}
	if files, err := os.ReadDir(managed); err != nil || len(files) != 1 {
		t.Fatalf("expected the uncollected file to remain: %v %v", files, err)
	}
}

// Pressing Cmd+, opens Settings from workspace views and focused inputs while respecting open modals, active invalid drafts, and already-open Settings edits.
func TestSettingsShortcutOpensSettingsWithModalAndDraftGuards(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})

	// 1. Open Settings via Cmd+, from the initial All Projects view.
	tt.Key(ui.Cmd, ui.KeyComma)
	if wb.navPage != NavSettings || !tt.HasText("全局设置与数据管理") {
		t.Fatalf("expected Cmd+, to open Settings, navPage=%s", wb.navPage)
	}

	// 2. Pressing Cmd+, while already on Settings must not overwrite unsaved policy edits.
	wb.backupDirInput = "/tmp/unsaved-custom-backup"
	tt.Key(ui.Cmd, ui.KeyComma)
	if wb.navPage != NavSettings || wb.backupDirInput != "/tmp/unsaved-custom-backup" {
		t.Fatalf("expected Cmd+, on Settings to preserve unsaved backupDirInput, got %q", wb.backupDirInput)
	}

	// 3. Create a project and enter its board; Cmd+, must ignore while a modal is open.
	wb.NavigateTo(NavAllProjects)
	tt.Frame()
	wb.OpenCreateProjectModal()
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyComma)
	if wb.navPage != NavAllProjects || !wb.projectModalOpen {
		t.Fatalf("expected Cmd+, behind modal to keep modal open and stay on %s, got %s (modal=%v)", NavAllProjects, wb.navPage, wb.projectModalOpen)
	}
	wb.projectModalName = "快捷键测试项目"
	wb.SubmitProjectModal()
	tt.Frame()

	task, err := wb.QuickCreateTask(model.Planned, "待检查任务")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	tt.Frame()

	// 4. Cmd+, is ignored while the task detail modal is open, and an invalid unsaved task draft blocks navigation when closed.
	tt.Key(ui.Cmd, ui.KeyComma)
	if wb.navPage != NavProjectBoard || !wb.detailModalOpen() {
		t.Fatalf("expected Cmd+, behind task detail modal to stay on board, got %s", wb.navPage)
	}
	wb.editTitle = "   "
	wb.dirty = true
	wb.detailExpanded = false
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyComma)
	if wb.navPage != NavProjectBoard || !wb.detailModalOpen() {
		t.Fatalf("expected Cmd+, with invalid draft to remain on board and reopen detail modal, got %s", wb.navPage)
	}
	tt.Frame()
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatalf("click 放弃修改: %v", err)
	}
	wb.ClearSelectedTask()
	tt.Frame()

	// 5. Cmd+, still navigates to Settings when a text input on the board has focus.
	if err := tt.Click("搜索任务"); err != nil {
		t.Fatalf("click 搜索任务: %v", err)
	}
	tt.Key(ui.Cmd, ui.KeyComma)
	if wb.navPage != NavSettings || !tt.HasText("全局设置与数据管理") {
		t.Fatalf("expected Cmd+, from focused search input to open Settings, got %s", wb.navPage)
	}
}

// Regression: quick-move buttons on task cards previously always rendered right arrows (→) even when the target column was to the left.
func TestQuickMoveButtonArrowsFollowColumnDirection(t *testing.T) {
	if quickMoveIsLeft(model.Planned, model.InProgress) || quickMoveIsLeft(model.Planned, model.Done) {
		t.Fatal("expected Planned -> InProgress/Done to point right")
	}
	if !quickMoveIsLeft(model.InProgress, model.Planned) || quickMoveIsLeft(model.InProgress, model.Done) {
		t.Fatal("expected InProgress -> Planned left and InProgress -> Done right")
	}
	if !quickMoveIsLeft(model.Done, model.Planned) || !quickMoveIsLeft(model.Done, model.InProgress) {
		t.Fatal("expected Done -> Planned/InProgress to point left")
	}

	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "方向箭头测试"
	wb.SubmitProjectModal()

	task, err := wb.QuickCreateTask(model.Planned, "方向任务")
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	hoverTaskCard := func() {
		r, ok := tt.Find("任务卡片 方向任务")
		if !ok {
			t.Fatal("could not find 任务卡片 方向任务")
		}
		tt.Move(r.X+r.W/2, r.Y+16)
	}

	// Planned (leftmost): hover reveals 进行中 and 已完成 actions.
	hoverTaskCard()
	if _, ok := tt.Find("移至进行中 方向任务"); !ok {
		t.Fatal("expected 移至进行中 方向任务 on hover in Planned")
	}
	if _, ok := tt.Find("移至已完成 方向任务"); !ok {
		t.Fatal("expected 移至已完成 方向任务 on hover in Planned")
	}

	// Move right into InProgress: hover reveals 计划 (left) and 已完成 (right).
	if err := tt.Click("移至进行中 方向任务"); err != nil {
		t.Fatalf("click 移至进行中 方向任务: %v", err)
	}
	hoverTaskCard()
	if _, ok := tt.Find("移至计划 方向任务"); !ok {
		t.Fatal("expected 移至计划 方向任务 on hover in InProgress")
	}
	if _, ok := tt.Find("移至已完成 方向任务"); !ok {
		t.Fatal("expected 移至已完成 方向任务 on hover in InProgress")
	}

	// Move right into Done: hover reveals 计划 and 进行中 (both left).
	if err := tt.Click("移至已完成 方向任务"); err != nil {
		t.Fatalf("click 移至已完成 方向任务: %v", err)
	}
	hoverTaskCard()
	if _, ok := tt.Find("移至计划 方向任务"); !ok {
		t.Fatal("expected 移至计划 方向任务 on hover in Done")
	}
	if _, ok := tt.Find("移至进行中 方向任务"); !ok {
		t.Fatal("expected 移至进行中 方向任务 on hover in Done")
	}

	// Move left back to Planned and verify persistence.
	if err := tt.Click("移至计划 方向任务"); err != nil {
		t.Fatalf("click 移至计划 方向任务: %v", err)
	}
	persisted, err := app.Projects.GetTask(context.Background(), task.ID)
	if err != nil || persisted.Status != model.Planned {
		t.Fatalf("expected task moved back to Planned in DB, got %+v err=%v", persisted, err)
	}
	hoverTaskCard()
	if _, ok := tt.Find("移至进行中 方向任务"); !ok {
		t.Fatal("expected 移至进行中 方向任务 after returning to Planned")
	}
}

// Creating tasks via the top-right modal or column quick-add leaves the right detail sidebar collapsed until a task is explicitly selected.
func TestCreateTaskDoesNotExpandDetailSidebar(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "默认不展开详情"
	wb.SubmitProjectModal()
	tt.Frame()

	// 1. Top-right "+ 新建任务" modal creates a task without opening the detail sidebar.
	if err := tt.Click("+ 新建任务"); err != nil {
		t.Fatalf("click + 新建任务: %v", err)
	}
	if err := tt.Click("新建任务标题输入框"); err != nil {
		t.Fatalf("click 新建任务标题输入框: %v", err)
	}
	tt.Type("弹窗新建任务")
	if err := tt.Click("创建任务"); err != nil {
		t.Fatalf("click 创建任务: %v", err)
	}
	if wb.detailExpanded || wb.selectedTaskID != "" {
		t.Fatalf("expected detail sidebar collapsed after modal create, got expanded=%v selected=%q", wb.detailExpanded, wb.selectedTaskID)
	}
	if _, found := tt.Find("任务详情标题"); found {
		t.Fatal("expected detail panel not rendered after modal create")
	}

	// 2. Column bottom quick-add creates a task without opening the detail sidebar.
	if err := tt.Click("快速添加计划任务"); err != nil {
		t.Fatalf("click 快速添加计划任务: %v", err)
	}
	tt.Type("快速添加任务")
	tt.Key(0, ui.KeyEnter)
	if wb.detailExpanded || wb.selectedTaskID != "" {
		t.Fatalf("expected detail sidebar collapsed after quick-add, got expanded=%v selected=%q", wb.detailExpanded, wb.selectedTaskID)
	}
	if _, found := tt.Find("任务详情标题"); found {
		t.Fatal("expected detail panel not rendered after quick-add")
	}
	if got := len(wb.TasksInColumn(model.Planned)); got != 2 {
		t.Fatalf("expected 2 created tasks in Planned column, got %d", got)
	}

	// 3. Clicking a created task card explicitly opens its detail sidebar.
	if err := tt.Click("任务卡片 快速添加任务"); err != nil {
		t.Fatalf("click 任务卡片 快速添加任务: %v", err)
	}
	if !wb.detailExpanded || wb.selectedTaskID == "" {
		t.Fatalf("expected detail sidebar expanded after clicking task card, got expanded=%v selected=%q", wb.detailExpanded, wb.selectedTaskID)
	}
	if _, found := tt.Find("任务详情标题"); !found {
		t.Fatal("expected detail panel rendered after clicking task card")
	}
}

// Clicking anywhere on a project card opens its board while nested action buttons handle their own actions without navigating to the board.
func TestProjectCardClickOpensBoardWithoutTriggeringFromActionButtons(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "整卡跳转项目"
	wb.SubmitProjectModal()
	projID := wb.activeProjectID

	// Return to All Projects overview.
	wb.NavigateTo(NavAllProjects)
	wb.activeProjectID = ""
	tt.Frame()

	// 1. Clicking Edit opens the modal without opening the project board.
	if err := tt.Click("编辑项目 整卡跳转项目"); err != nil {
		t.Fatalf("click 编辑项目 整卡跳转项目: %v", err)
	}
	if !wb.projectModalOpen || wb.navPage != NavAllProjects {
		t.Fatalf("expected edit modal open on NavAllProjects, got open=%v page=%s", wb.projectModalOpen, wb.navPage)
	}
	wb.projectModalOpen = false
	tt.Frame()

	// 2. Clicking the project card opens its board.
	if err := tt.Click("项目卡片 整卡跳转项目"); err != nil {
		t.Fatalf("click 项目卡片 整卡跳转项目: %v", err)
	}
	if wb.navPage != NavProjectBoard || wb.activeProjectID != projID {
		t.Fatalf("expected board open for %s, got page=%s active=%s", projID, wb.navPage, wb.activeProjectID)
	}

	// 3. Clicking Archive from All Projects archives without opening the board.
	wb.NavigateTo(NavAllProjects)
	wb.activeProjectID = ""
	tt.Frame()
	if err := tt.Click("归档项目 整卡跳转项目"); err != nil {
		t.Fatalf("click 归档项目 整卡跳转项目: %v", err)
	}
	if wb.navPage != NavAllProjects || len(wb.activeProjects) != 0 || len(wb.archivedProjects) != 1 {
		t.Fatalf("expected project archived on NavAllProjects, got page=%s active=%d archived=%d", wb.navPage, len(wb.activeProjects), len(wb.archivedProjects))
	}

	// 4. Clicking an archived project card opens its board; Unarchive stays in overview.
	wb.NavigateTo(NavArchivedProjects)
	tt.Frame()
	if err := tt.Click("项目卡片 整卡跳转项目"); err != nil {
		t.Fatalf("click archived 项目卡片 整卡跳转项目: %v", err)
	}
	if wb.navPage != NavProjectBoard || wb.activeProjectID != projID {
		t.Fatalf("expected archived project board open, got page=%s active=%s", wb.navPage, wb.activeProjectID)
	}

	wb.NavigateTo(NavArchivedProjects)
	wb.activeProjectID = ""
	tt.Frame()
	if err := tt.Click("恢复归档 整卡跳转项目"); err != nil {
		t.Fatalf("click 恢复归档 整卡跳转项目: %v", err)
	}
	if wb.navPage != NavArchivedProjects || len(wb.activeProjects) != 1 {
		t.Fatalf("expected project unarchived without opening board, got page=%s active=%d", wb.navPage, len(wb.activeProjects))
	}

	// 5. Clicking Move to Trash moves project without opening the board.
	wb.NavigateTo(NavAllProjects)
	tt.Frame()
	if err := tt.Click("移入回收站 整卡跳转项目"); err != nil {
		t.Fatalf("click 移入回收站 整卡跳转项目: %v", err)
	}
	if wb.navPage != NavAllProjects || len(wb.activeProjects) != 0 || len(wb.trashItems) != 1 {
		t.Fatalf("expected project trashed without opening board, got page=%s active=%d trash=%d", wb.navPage, len(wb.activeProjects), len(wb.trashItems))
	}
}

// Regression: column quick-add inputs previously shrank to placeholder width instead of stretching across wide columns.
func TestQuickAddInputFillsColumnWidth(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "全宽快速添加"
	wb.SubmitProjectModal()
	tt.Frame()

	for _, pair := range []struct {
		colLabel   string
		inputLabel string
	}{
		{"看板列 计划", "快速添加计划任务"},
		{"看板列 进行中", "快速添加进行中任务"},
		{"看板列 已完成", "快速添加已完成任务"},
	} {
		col, ok := tt.Access(pair.colLabel)
		if !ok {
			t.Fatalf("missing %s", pair.colLabel)
		}
		inp, ok := tt.Access(pair.inputLabel)
		if !ok {
			t.Fatalf("missing %s", pair.inputLabel)
		}
		// Inside a column with 10px horizontal row padding and 8px input horizontal padding, the inner text input should span nearly the full column width.
		if inp.Bounds.W < col.Bounds.W-48 {
			t.Fatalf("%s width %.1f did not fill column %s width %.1f", pair.inputLabel, inp.Bounds.W, pair.colLabel, col.Bounds.W)
		}
	}
}

// Clicking a task card's quick-move buttons (or context menu move items) inserts the task at the top of the target column instead of appending it to the bottom.
func TestQuickMoveMovesTaskToTopOfTargetColumn(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "快捷移动置顶"
	wb.SubmitProjectModal()

	p1, err := wb.QuickCreateTask(model.Planned, "计划已有任务一")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := wb.QuickCreateTask(model.Planned, "计划已有任务二")
	if err != nil {
		t.Fatal(err)
	}
	ip1, err := wb.QuickCreateTask(model.InProgress, "进行中已有任务")
	if err != nil {
		t.Fatal(err)
	}
	target, err := wb.QuickCreateTask(model.InProgress, "待转移任务")
	if err != nil {
		t.Fatal(err)
	}
	d1, err := wb.QuickCreateTask(model.Done, "已完成已有任务")
	if err != nil {
		t.Fatal(err)
	}
	wb.ClearSelectedTask()
	tt.Frame()

	hoverTarget := func() {
		r, ok := tt.Find("任务卡片 待转移任务")
		if !ok {
			t.Fatal("could not find 任务卡片 待转移任务")
		}
		tt.Move(r.X+r.W/2, r.Y+16)
	}

	// 1. Quick-move from InProgress to Planned places target at the top of Planned.
	hoverTarget()
	if err := tt.Click("移至计划 待转移任务"); err != nil {
		t.Fatalf("click 移至计划 待转移任务: %v", err)
	}
	planned := wb.TasksInColumn(model.Planned)
	if len(planned) != 3 || planned[0].ID != target.ID || planned[1].ID != p1.ID || planned[2].ID != p2.ID {
		t.Fatalf("expected [%s %s %s] in Planned, got %+v", target.ID, p1.ID, p2.ID, planned)
	}

	// 2. Quick-move from Planned to Done places target at the top of Done.
	hoverTarget()
	if err := tt.Click("移至已完成 待转移任务"); err != nil {
		t.Fatalf("click 移至已完成 待转移任务: %v", err)
	}
	done := wb.TasksInColumn(model.Done)
	if len(done) != 2 || done[0].ID != target.ID || done[1].ID != d1.ID {
		t.Fatalf("expected [%s %s] in Done, got %+v", target.ID, d1.ID, done)
	}

	// 3. Even with an active filter hiding existing InProgress tasks, quick-move to InProgress inserts at absolute top (position 0).
	if err := tt.Click("搜索任务"); err != nil {
		t.Fatalf("click 搜索任务: %v", err)
	}
	tt.Type("待转移")
	hoverTarget()
	if err := tt.Click("移至进行中 待转移任务"); err != nil {
		t.Fatalf("click 移至进行中 待转移任务: %v", err)
	}
	wb.ClearBoardFilters()
	tt.Frame()
	inProgress := wb.TasksInColumn(model.InProgress)
	if len(inProgress) != 2 || inProgress[0].ID != target.ID || inProgress[1].ID != ip1.ID {
		t.Fatalf("expected [%s %s] in InProgress after filtered quick-move, got %+v", target.ID, ip1.ID, inProgress)
	}
	persisted, err := app.Projects.GetTask(context.Background(), target.ID)
	if err != nil || persisted.Status != model.InProgress || persisted.Position != 0 {
		t.Fatalf("expected persisted task at InProgress pos 0, got %+v err=%v", persisted, err)
	}

	// 4. Context menu move item also places target at the top of the destination column.
	if err := tt.RightClick("任务卡片 待转移任务"); err != nil {
		t.Fatalf("right click 任务卡片 待转移任务: %v", err)
	}
	if err := tt.Click("移至 已完成"); err != nil {
		t.Fatalf("click context menu 移至 已完成: %v", err)
	}
	done = wb.TasksInColumn(model.Done)
	if len(done) != 2 || done[0].ID != target.ID || done[1].ID != d1.ID {
		t.Fatalf("expected target at top of Done via context menu, got %+v", done)
	}
}

// Dragging a card inserts cleanly into the top, middle, or bottom blank gap of a column without shifting card or column bounds during hover.
func TestBoardDragInsertsIntoTopMiddleBottomGapsWithoutLayoutShift(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "间隙插入拖拽"
	wb.SubmitProjectModal()

	ip1, err := wb.QuickCreateTask(model.InProgress, "进行中卡片一")
	if err != nil {
		t.Fatal(err)
	}
	ip2, err := wb.QuickCreateTask(model.InProgress, "进行中卡片二")
	if err != nil {
		t.Fatal(err)
	}
	d1, err := wb.QuickCreateTask(model.Done, "完成卡片一")
	if err != nil {
		t.Fatal(err)
	}
	d2, err := wb.QuickCreateTask(model.Done, "完成卡片二")
	if err != nil {
		t.Fatal(err)
	}
	wb.ClearSelectedTask()
	tt.Frame()

	mustFind := func(label string) ui.Rect {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing %q", label)
		}
		return r
	}
	dragTo := func(fromX, fromY, toX, toY float32, whileOver func()) {
		tt.Press(fromX, fromY)
		for step := 1; step <= 4; step++ {
			f := float32(step) / 4
			tt.Move(fromX+(toX-fromX)*f, fromY+(toY-fromY)*f)
		}
		if whileOver != nil {
			whileOver()
		}
		tt.Release(toX, toY)
	}

	// 1. Drag ip1 into the exact gap between d1 and d2; bounds must not shift during drag-over.
	ip1Rect := mustFind("任务卡片 进行中卡片一")
	d1RectBefore := mustFind("任务卡片 完成卡片一")
	d2RectBefore := mustFind("任务卡片 完成卡片二")
	doneColBefore := mustFind("看板列 已完成")
	gapX := d1RectBefore.X + d1RectBefore.W/2
	gapMidY := (d1RectBefore.Y + d1RectBefore.H + d2RectBefore.Y) / 2

	dragTo(ip1Rect.X+ip1Rect.W/2, ip1Rect.Y+16, gapX, gapMidY, func() {
		d1During := mustFind("任务卡片 完成卡片一")
		d2During := mustFind("任务卡片 完成卡片二")
		doneColDuring := mustFind("看板列 已完成")
		if d1During != d1RectBefore || d2During != d2RectBefore || doneColDuring != doneColBefore {
			t.Fatalf("expected zero layout shift while dragging over gap, before=(%v,%v,%v) during=(%v,%v,%v)",
				d1RectBefore, d2RectBefore, doneColBefore, d1During, d2During, doneColDuring)
		}
	})
	done := wb.TasksInColumn(model.Done)
	if len(done) != 3 || done[0].ID != d1.ID || done[1].ID != ip1.ID || done[2].ID != d2.ID {
		t.Fatalf("expected middle gap insertion [%s %s %s], got %+v", d1.ID, ip1.ID, d2.ID, done)
	}

	// 2. Drag ip2 into the top blank space above d1.
	ip2Rect := mustFind("任务卡片 进行中卡片二")
	d1Rect := mustFind("任务卡片 完成卡片一")
	dragTo(ip2Rect.X+ip2Rect.W/2, ip2Rect.Y+16, d1Rect.X+d1Rect.W/2, d1Rect.Y-5, nil)
	done = wb.TasksInColumn(model.Done)
	if len(done) != 4 || done[0].ID != ip2.ID || done[1].ID != d1.ID || done[2].ID != ip1.ID || done[3].ID != d2.ID {
		t.Fatalf("expected top gap insertion [%s %s %s %s], got %+v", ip2.ID, d1.ID, ip1.ID, d2.ID, done)
	}

	// 3. Same-column drag: drag top card (ip2) into the empty space below the last card (d2).
	topRect := mustFind("任务卡片 进行中卡片二")
	lastRect := mustFind("任务卡片 完成卡片二")
	dragTo(topRect.X+topRect.W/2, topRect.Y+16, lastRect.X+lastRect.W/2, lastRect.Y+lastRect.H+40, nil)
	done = wb.TasksInColumn(model.Done)
	if len(done) != 4 || done[0].ID != d1.ID || done[1].ID != ip1.ID || done[2].ID != d2.ID || done[3].ID != ip2.ID {
		t.Fatalf("expected bottom space insertion [%s %s %s %s], got %+v", d1.ID, ip1.ID, d2.ID, ip2.ID, done)
	}

	// 4. Same-column drag: drag bottom card (ip2) back to the top gap above d1.
	bottomRect := mustFind("任务卡片 进行中卡片二")
	firstRect := mustFind("任务卡片 完成卡片一")
	dragTo(bottomRect.X+bottomRect.W/2, bottomRect.Y+16, firstRect.X+firstRect.W/2, firstRect.Y-5, nil)
	done = wb.TasksInColumn(model.Done)
	if len(done) != 4 || done[0].ID != ip2.ID || done[1].ID != d1.ID || done[2].ID != ip1.ID || done[3].ID != d2.ID {
		t.Fatalf("expected same-column top insertion [%s %s %s %s], got %+v", ip2.ID, d1.ID, ip1.ID, d2.ID, done)
	}
}

// Task card quick-move and reorder buttons stay hidden on idle cards and reveal on pointer hover or card selection.
func TestTaskCardQuickActionsRevealOnHoverOrSelection(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "悬停显示操作栏"
	wb.SubmitProjectModal()

	t1, err := wb.QuickCreateTask(model.Planned, "卡片甲")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wb.QuickCreateTask(model.Planned, "卡片乙"); err != nil {
		t.Fatal(err)
	}
	wb.ClearSelectedTask()
	tt.Frame()

	// 1. Idle unselected cards keep quick-action bars hidden.
	if _, ok := tt.Find("移至进行中 卡片甲"); ok {
		t.Fatal("expected quick-move buttons hidden on unhovered, unselected card")
	}
	if _, ok := tt.Find("上移 卡片乙"); ok {
		t.Fatal("expected reorder buttons hidden on unhovered, unselected card")
	}

	// 2. Hovering 卡片甲 reveals its quick-move and down-reorder buttons without opening detail panel.
	card1Rect, ok := tt.Find("任务卡片 卡片甲")
	if !ok {
		t.Fatal("could not find 任务卡片 卡片甲")
	}
	tt.Move(card1Rect.X+card1Rect.W/2, card1Rect.Y+16)
	if _, ok := tt.Find("移至进行中 卡片甲"); !ok {
		t.Fatal("expected 移至进行中 卡片甲 visible while hovered")
	}
	if _, ok := tt.Find("下移 卡片甲"); !ok {
		t.Fatal("expected 下移 卡片甲 visible while hovered")
	}
	if wb.selectedTaskID != "" {
		t.Fatalf("expected hovering not to select task, got %q", wb.selectedTaskID)
	}

	// 3. Moving pointer out onto empty column header hides the quick-action bar again.
	colRect, ok := tt.Find("看板列 进行中")
	if !ok {
		t.Fatal("could not find 看板列 进行中")
	}
	tt.Move(colRect.X+colRect.W/2, colRect.Y+14)
	if _, ok := tt.Find("移至进行中 卡片甲"); ok {
		t.Fatal("expected quick-move buttons hidden after pointer leaves card")
	}

	// 4. Selecting 卡片甲 keeps its quick-action bar visible even when pointer is elsewhere.
	wb.SelectTask(t1.ID)
	tt.Frame()
	if _, ok := tt.Find("移至进行中 卡片甲"); !ok {
		t.Fatal("expected quick-move buttons visible while card is selected")
	}
}

// Opening task detail displays a modal dialog over the board without shifting column layout, and blocks Escape/close while an invalid draft exists.
func TestTaskDetailModalPreservesBoardLayoutAndGuardsDismiss(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "弹窗详情项目"
	wb.SubmitProjectModal()

	task, err := wb.QuickCreateTask(model.Planned, "弹窗测试任务")
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	plannedBefore, ok := tt.Find("看板列 计划")
	if !ok {
		t.Fatal("could not find 看板列 计划 before opening modal")
	}
	doneBefore, ok := tt.Find("看板列 已完成")
	if !ok {
		t.Fatal("could not find 看板列 已完成 before opening modal")
	}

	// 1. Clicking the task card opens the centered modal dialog without changing board column widths or positions.
	if err := tt.Click("任务卡片 弹窗测试任务"); err != nil {
		t.Fatalf("click task card: %v", err)
	}
	if _, ok := tt.Find("任务详情弹窗"); !ok {
		t.Fatal("expected 任务详情弹窗 to be rendered")
	}
	plannedAfter, ok := tt.Find("看板列 计划")
	if !ok || plannedAfter != plannedBefore {
		t.Fatalf("expected 看板列 计划 layout unchanged: before=%+v after=%+v", plannedBefore, plannedAfter)
	}
	doneAfter, ok := tt.Find("看板列 已完成")
	if !ok || doneAfter != doneBefore {
		t.Fatalf("expected 看板列 已完成 layout unchanged: before=%+v after=%+v", doneBefore, doneAfter)
	}

	// 2. Pressing Escape while title is invalid must keep the modal open and preserve the draft.
	wb.editTitle = "   "
	wb.editDescription = "不可丢失的草稿"
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if !wb.detailModalOpen() || wb.selectedTaskID != task.ID || wb.editDescription != "不可丢失的草稿" {
		t.Fatalf("expected Escape with invalid draft to keep modal open and preserve draft, open=%v selected=%q", wb.detailModalOpen(), wb.selectedTaskID)
	}

	// 3. After fixing or discarding the draft, Escape closes the modal cleanly.
	if err := tt.Click("放弃修改"); err != nil {
		t.Fatalf("click 放弃修改: %v", err)
	}
	tt.Key(0, ui.KeyEscape)
	if wb.detailModalOpen() || wb.selectedTaskID != "" {
		t.Fatalf("expected Escape to close valid task detail modal, open=%v selected=%q", wb.detailModalOpen(), wb.selectedTaskID)
	}
	if _, ok := tt.Find("任务详情弹窗"); ok {
		t.Fatal("expected 任务详情弹窗 removed after Escape")
	}

	// 4. Reopening and clicking the header close button closes the modal cleanly.
	wb.SelectTask(task.ID)
	tt.Frame()
	if err := tt.Click("关闭详情弹窗"); err != nil {
		t.Fatalf("click 关闭详情弹窗: %v", err)
	}
	if wb.detailModalOpen() || wb.selectedTaskID != "" {
		t.Fatalf("expected 关闭详情弹窗 to close modal, open=%v selected=%q", wb.detailModalOpen(), wb.selectedTaskID)
	}
}

// Clicking the priority pill on a board card opens a quick-switch menu, persists the new priority without opening the detail modal, and updates active board filters.
func TestCardPriorityBadgeQuickSwitch(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "快捷切换优先级项目"
	wb.SubmitProjectModal()

	ctx := context.Background()
	task, err := app.Projects.CreateTask(ctx, wb.activeProjectID, projects.TaskInput{
		Title:       "优先级切换卡片",
		Description: "保持正文不变",
		Status:      model.Done,
		Priority:    model.NoPriority,
		Tags:        []string{"UI"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wb.ReloadBoardTasks()
	tt.Frame()

	// 1. Clicking the priority badge opens the dropdown menu without opening the task detail modal.
	if err := tt.Click("切换优先级 优先级切换卡片"); err != nil {
		t.Fatalf("click 切换优先级 优先级切换卡片: %v", err)
	}
	tt.Advance(time.Second)
	if wb.detailModalOpen() || wb.selectedTaskID != "" {
		t.Fatalf("expected clicking priority badge not to open detail modal, open=%v selected=%q", wb.detailModalOpen(), wb.selectedTaskID)
	}
	if err := tt.Click("紧急"); err != nil {
		t.Fatalf("click 紧急 in priority menu: %v", err)
	}
	tt.Advance(time.Second)

	persisted, err := app.Projects.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask after priority switch: %v", err)
	}
	if persisted.Priority != model.Urgent {
		t.Fatalf("expected persisted priority %s, got %s", model.Urgent, persisted.Priority)
	}
	if persisted.Description != "保持正文不变" || persisted.Status != model.Done || !slices.Equal(persisted.Tags, []string{"UI"}) {
		t.Fatalf("priority switch mutated unrelated fields: %+v", persisted)
	}

	// 2. Switching priority while an active priority filter matches the old priority updates filteredTasks immediately.
	wb.priorityFilter = string(model.Urgent)
	wb.applyBoardFilters()
	tt.Frame()
	if len(wb.filteredTasks) != 1 {
		t.Fatalf("expected 1 visible task matching Urgent filter, got %d", len(wb.filteredTasks))
	}
	if err := tt.Click("切换优先级 优先级切换卡片"); err != nil {
		t.Fatalf("reopen priority menu: %v", err)
	}
	tt.Advance(time.Second)
	if err := tt.Click("低"); err != nil {
		t.Fatalf("click 低 in priority menu: %v", err)
	}
	tt.Advance(time.Second)

	persisted, err = app.Projects.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask after second priority switch: %v", err)
	}
	if persisted.Priority != model.Low {
		t.Fatalf("expected persisted priority %s, got %s", model.Low, persisted.Priority)
	}
	if len(wb.filteredTasks) != 0 {
		t.Fatalf("expected task filtered out of Urgent view after switching to Low, got %d", len(wb.filteredTasks))
	}
}
