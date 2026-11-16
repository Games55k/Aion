package frontend

import (
	"fmt"
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

// Regression: uiSelect clamped its non-scrolling popup at MaxHeight(256), causing
// the 9th stage option ("Offer") and longer option lists to overflow outside the popup panel.
func TestSelectPopupContainsAllStageOptions(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.SwitchModule("job_applications")
	tt.Frame()

	if err := tt.Click("筛选阶段"); err != nil {
		t.Fatalf("open stage filter select: %v", err)
	}
	popup, ok := tt.AccessByRole(ui.RoleListBox, "")
	if !ok {
		t.Fatalf("expected open RoleListBox popup for stage filter")
	}
	offerRect, ok := tt.Find("Offer")
	if !ok {
		t.Fatalf("could not find Offer option text inside stage filter popup; texts=%q", tt.Texts())
	}
	if offerRect.Y < popup.Bounds.Y || offerRect.Y+offerRect.H > popup.Bounds.Y+popup.Bounds.H ||
		offerRect.X < popup.Bounds.X || offerRect.X+offerRect.W > popup.Bounds.X+popup.Bounds.W {
		t.Fatalf("Offer option overflowed popup bounds: offer=%v popup=%v", offerRect, popup.Bounds)
	}
	offerAcc, ok := tt.AccessByRole(ui.RoleOption, "Offer")
	if !ok {
		t.Fatalf("expected RoleOption for Offer")
	}
	if offerAcc.Bounds.W < popup.Bounds.W-10 {
		t.Fatalf("expected Offer option row to stretch across popup width: option=%v popup=%v", offerAcc.Bounds, popup.Bounds)
	}

	// Verify 12+ options stay within the scrollable popup bounds and scroll to the last item.
	manyOpts := make([]selectOption, 0, 15)
	for i := 1; i <= 15; i++ {
		manyOpts = append(manyOpts, selectOption{
			Value: fmt.Sprintf("opt-%02d", i),
			Label: fmt.Sprintf("阶段选项 %02d", i),
		})
	}
	selected := manyOpts[0].Value
	longTT := ui.NewTester(func(c *ui.Context) {
		applyUITheme(c)
		ui.Box(c).Padding(24).Width(240).Children(func() {
			uiSelect(c.Key("long-select"), &selected, selectProps{
				Options: manyOpts,
				Size:    sizeSmall,
				Variant: inputOutlined,
			}).Label("长选项选择器")
		})
	}, 400, 600)
	if err := longTT.Click("长选项选择器"); err != nil {
		t.Fatalf("open long select: %v", err)
	}
	longPopup, ok := longTT.AccessByRole(ui.RoleListBox, "")
	if !ok {
		t.Fatalf("expected open RoleListBox for long select")
	}
	if longPopup.Bounds.H > 385 {
		t.Fatalf("expected scrollable popup height capped near 384, got %v", longPopup.Bounds)
	}
	longTT.Key(0, ui.KeyEnd)
	lastRect, ok := longTT.Find("阶段选项 15")
	if !ok {
		t.Fatalf("expected final option visible after KeyEnd")
	}
	if lastRect.Y < longPopup.Bounds.Y || lastRect.Y+lastRect.H > longPopup.Bounds.Y+longPopup.Bounds.H {
		t.Fatalf("final option scrolled out of popup bounds: item=%v popup=%v", lastRect, longPopup.Bounds)
	}
}
