package frontend

import (
	"context"
	"slices"
	"testing"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"
)

// Toggling module visibility in Settings hides the module from the sidebar switcher without deleting data, prevents hiding the last module, and persists across reopen.
func TestModuleVisibilityTogglePersistsAndPreservesData(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	app, wb, tt := openTestWorkbench(t, dataDir, Host{})

	if _, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "KeepCo",
		PositionName: "Go Engineer",
		AppliedDate:  "2026-11-26",
	}); err != nil {
		t.Fatal(err)
	}
	wb.ReloadJobApplications()

	wb.NavigateTo(NavSettings)
	tt.Frame()
	tt.Scroll(500, 300, 0, 500)
	tt.Frame()

	// Hide 求职 module via Settings switch.
	if err := tt.Click("显示模块 求职"); err != nil {
		t.Fatalf("toggle hide 求职: %v", err)
	}
	tt.Frame()

	if wb.IsModuleVisible("job_applications") {
		t.Fatal("expected job_applications module to be hidden")
	}
	if len(wb.VisibleModules()) != 1 || wb.VisibleModules()[0].ID != "projects" {
		t.Fatalf("unexpected visible modules: %+v", wb.VisibleModules())
	}

	// Attempting to hide the last remaining visible module (项目) must be blocked.
	wb.SetModuleVisible("projects", false)
	if !wb.IsModuleVisible("projects") {
		t.Fatal("must not allow hiding the last remaining visible module")
	}

	// Verify persistence in SQLite and that job data was not deleted.
	hidden, err := app.Settings.HiddenModules(ctx)
	if err != nil || !slices.Equal(hidden, []string{"job_applications"}) {
		t.Fatalf("expected persisted hidden modules [job_applications], got %v err=%v", hidden, err)
	}
	jobs, err := app.JobApplications.List(ctx)
	if err != nil || len(jobs) != 1 || jobs[0].CompanyName != "KeepCo" {
		t.Fatalf("hiding module unexpectedly altered job records: %+v err=%v", jobs, err)
	}

	// Re-enabling restores visibility immediately.
	if err := tt.Click("显示模块 求职"); err != nil {
		t.Fatalf("toggle show 求职: %v", err)
	}
	tt.Frame()
	if !wb.IsModuleVisible("job_applications") {
		t.Fatal("expected job_applications module visible again")
	}
}

// Focus overview aggregates overdue/in-progress/high-priority tasks and active jobs, adapts to hidden modules, and navigates directly into clicked records.
func TestFocusOverviewAggregatesAndNavigates(t *testing.T) {
	ctx := context.Background()
	fixedNow := func() time.Time { return time.Date(2026, 11, 26, 9, 0, 0, 0, time.Local) }
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{Now: fixedNow})

	proj, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Core"})
	if err != nil {
		t.Fatal(err)
	}
	overdueTask, err := app.Projects.CreateTask(ctx, proj.ID, projects.TaskInput{
		Title:    "Fix overdue bug",
		Status:   model.Planned,
		Priority: model.Medium,
		DueDate:  "2026-11-24",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Projects.CreateTask(ctx, proj.ID, projects.TaskInput{
		Title:    "Low backlog item",
		Status:   model.Planned,
		Priority: model.Low,
	}); err != nil {
		t.Fatal(err)
	}
	job, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "FocusCorp",
		PositionName: "Systems Engineer",
		AppliedDate:  "2026-11-25",
		Stage:        model.JobStageInterview1,
	})
	if err != nil {
		t.Fatal(err)
	}

	wb.ReloadAll()
	if err := tt.Click("聚焦"); err != nil {
		t.Fatalf("click 聚焦 in sidebar: %v", err)
	}
	tt.Frame()

	if wb.navPage != NavFocus {
		t.Fatalf("expected NavFocus, got %q", wb.navPage)
	}
	if !tt.HasText("Fix overdue bug") {
		t.Fatal("expected overdue task visible in Focus overview")
	}
	if tt.HasText("Low backlog item") {
		t.Fatal("did not expect low-priority undated planned task in Focus overview")
	}
	if !tt.HasText("FocusCorp") {
		t.Fatal("expected active job visible in Focus overview")
	}

	// Clicking a focus task opens its project board and detail modal.
	if err := tt.Click("打开聚焦任务 Fix overdue bug"); err != nil {
		t.Fatalf("click focus task: %v", err)
	}
	tt.Frame()
	if wb.navPage != NavProjectBoard || wb.activeProjectID != proj.ID || wb.selectedTaskID != overdueTask.ID || !wb.detailExpanded {
		t.Fatalf("focus task navigation mismatch: page=%q proj=%q task=%q expanded=%v", wb.navPage, wb.activeProjectID, wb.selectedTaskID, wb.detailExpanded)
	}

	// Go back to Focus and open the active job directly.
	wb.NavigateTo(NavFocus)
	tt.Frame()
	if err := tt.Click("打开聚焦投递 FocusCorp · Systems Engineer"); err != nil {
		t.Fatalf("click focus job: %v", err)
	}
	tt.Frame()
	if wb.navPage != NavJobActive || wb.selectedJobID != job.ID || !wb.jobDetailOpen {
		t.Fatalf("focus job navigation mismatch: page=%q job=%q open=%v", wb.navPage, wb.selectedJobID, wb.jobDetailOpen)
	}

	// Hiding job_applications hides the job section in Focus overview.
	wb.CloseJobDetail()
	wb.SetModuleVisible("job_applications", false)
	wb.NavigateTo(NavFocus)
	tt.Frame()
	if tt.HasText("进行中的求职投递") {
		t.Fatal("expected hidden job_applications module section removed from Focus overview")
	}
}
