package frontend

import (
	"testing"
	"time"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

func assertClearCentered(t *testing.T, tt *ui.Tester, fieldName string) {
	t.Helper()
	field, ok := tt.Access(fieldName)
	if !ok {
		t.Fatalf("could not find %s", fieldName)
	}
	clear, ok := tt.AccessByRole(ui.RoleButton, "Clear")
	if !ok {
		t.Fatalf("could not find the date field clear button next to %s", fieldName)
	}
	delta := (clear.Bounds.Y + clear.Bounds.H/2) - (field.Bounds.Y + field.Bounds.H/2)
	if delta > 1 || delta < -1 {
		t.Fatalf("clear button not vertically centered in %s: clear=%v field=%v delta=%.2f", fieldName, clear.Bounds, field.Bounds, delta)
	}
}

// Regression: the due-date clear button sat at the top of the field in the
// detail panel and the new-task modal (GitHub issue #1).
func TestDueDateClearButtonIsVerticallyCentered(t *testing.T) {
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	_ = app
	wb.OpenCreateProjectModal()
	wb.projectModalName = "截止日期对齐"
	wb.SubmitProjectModal()

	task, err := wb.QuickCreateTask(model.Planned, "检查清除按钮")
	if err != nil {
		t.Fatalf("QuickCreateTask: %v", err)
	}
	wb.SelectTask(task.ID)
	wb.editDueDate = time.Date(2026, 11, 12, 0, 0, 0, 0, time.Local)
	tt.Frame()
	assertClearCentered(t, tt, "任务详情截止日期")

	wb.ClearSelectedTask()
	wb.OpenCreateTaskModal(model.Planned)
	wb.taskModalDueDate = time.Date(2026, 11, 13, 0, 0, 0, 0, time.Local)
	tt.Frame()
	assertClearCentered(t, tt, "新建任务截止日期")
}
