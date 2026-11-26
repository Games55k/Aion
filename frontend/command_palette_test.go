package frontend

import (
	"context"
	"testing"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
)

// Cmd+K toggles the global command palette, searches across projects/tasks/jobs/commands, respects hidden modules, and navigates directly on Enter.
func TestCommandPaletteSearchNavigateAndGuards(t *testing.T) {
	ctx := context.Background()
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})

	proj, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Apollo"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.Projects.CreateTask(ctx, proj.ID, projects.TaskInput{
		Title:    "Benchmark SQLite WAL",
		Status:   model.Planned,
		Priority: model.High,
		Tags:     []string{"perf"},
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "Anthropic",
		PositionName: "Systems Engineer",
		AppliedDate:  "2026-11-26",
		Stage:        model.JobStageInterview1,
	})
	if err != nil {
		t.Fatal(err)
	}
	wb.ReloadAll()
	tt.Frame()

	// Open with Cmd+K, type task query, press Enter -> opens project board & task detail.
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	if !wb.CommandPaletteOpen() {
		t.Fatal("expected Cmd+K to open command palette")
	}
	tt.Type("Benchmark")
	tt.Frame()
	if !tt.HasText("Benchmark SQLite WAL") {
		t.Fatal("expected matching cross-project task in command palette")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if wb.CommandPaletteOpen() {
		t.Fatal("expected Enter to close command palette")
	}
	if wb.navPage != NavProjectBoard || wb.selectedTaskID != task.ID || !wb.detailExpanded {
		t.Fatalf("expected palette Enter to open task detail: page=%q task=%q expanded=%v", wb.navPage, wb.selectedTaskID, wb.detailExpanded)
	}

	// While task detail modal is open, Cmd+K must not open the palette over it.
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	if wb.CommandPaletteOpen() {
		t.Fatal("Cmd+K must be ignored while another modal is open")
	}

	// Navigate away to close task detail, open palette, search job company, click row -> opens job detail.
	wb.NavigateTo(NavFocus)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	tt.Type("Anthropic")
	tt.Frame()
	if err := tt.Click("命令面板项 Anthropic · Systems Engineer"); err != nil {
		t.Fatalf("click job item in palette: %v", err)
	}
	tt.Frame()
	if wb.selectedJobID != job.ID || !wb.jobDetailOpen {
		t.Fatalf("expected clicking job item to open job detail: id=%q open=%v", wb.selectedJobID, wb.jobDetailOpen)
	}

	// Close job detail, hide job_applications, verify palette excludes job entries and commands.
	wb.CloseJobDetail()
	wb.SetModuleVisible("job_applications", false)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	tt.Type("Anthropic")
	tt.Frame()
	if tt.HasText("Anthropic · Systems Engineer") {
		t.Fatal("hidden module items must not appear in command palette")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if wb.CommandPaletteOpen() {
		t.Fatal("Esc should close command palette")
	}
}

// Cmd+1, Cmd+2, and Cmd+3 switch between Focus, Projects, and Jobs while respecting hidden modules and open modals.
func TestGlobalNavigationShortcuts(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	tt.Frame()

	tt.Key(ui.Cmd, ui.Key1)
	tt.Frame()
	if wb.navPage != NavFocus {
		t.Fatalf("expected Cmd+1 to open NavFocus, got %q", wb.navPage)
	}

	tt.Key(ui.Cmd, ui.Key3)
	tt.Frame()
	if wb.activeModule != "job_applications" || !isJobNavPage(wb.navPage) {
		t.Fatalf("expected Cmd+3 to switch to jobs, got module=%q page=%q", wb.activeModule, wb.navPage)
	}

	tt.Key(ui.Cmd, ui.Key2)
	tt.Frame()
	if wb.activeModule != "projects" || !isProjectNavPage(wb.navPage) {
		t.Fatalf("expected Cmd+2 to switch to projects, got module=%q page=%q", wb.activeModule, wb.navPage)
	}

	// Hiding job_applications disables Cmd+3.
	wb.SetModuleVisible("job_applications", false)
	tt.Key(ui.Cmd, ui.Key3)
	tt.Frame()
	if wb.activeModule == "job_applications" {
		t.Fatal("Cmd+3 must not switch to hidden job_applications module")
	}

	// Opening a modal blocks Cmd+1.
	wb.OpenCreateProjectModal()
	tt.Frame()
	tt.Key(ui.Cmd, ui.Key1)
	tt.Frame()
	if wb.navPage == NavFocus {
		t.Fatal("Cmd+1 must be ignored while a modal is open")
	}
}

// Focus view inline quick buttons move a task to in_progress/done and advance a job application stage directly.
func TestFocusInlineQuickActions(t *testing.T) {
	ctx := context.Background()
	fixedNow := func() time.Time { return time.Date(2026, 11, 26, 10, 0, 0, 0, time.Local) }
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{Now: fixedNow})

	proj, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Sprint"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.Projects.CreateTask(ctx, proj.ID, projects.TaskInput{
		Title:    "Polish Focus Actions",
		Status:   model.Planned,
		Priority: model.Urgent,
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "Raycast",
		PositionName: "Mac Engineer",
		AppliedDate:  "2026-11-25",
		Stage:        model.JobStageInterview1,
	})
	if err != nil {
		t.Fatal(err)
	}

	wb.ReloadAll()
	wb.NavigateTo(NavFocus)
	tt.Frame()

	// Start task inline (planned -> in_progress).
	if err := tt.Click("开始任务 Polish Focus Actions"); err != nil {
		t.Fatalf("click inline start task: %v", err)
	}
	tt.Frame()
	gotTask, err := app.Projects.GetTask(ctx, task.ID)
	if err != nil || gotTask.Status != model.InProgress {
		t.Fatalf("expected task moved to in_progress, got %+v err=%v", gotTask, err)
	}

	// Complete task inline (in_progress -> done).
	if err := tt.Click("完成任务 Polish Focus Actions"); err != nil {
		t.Fatalf("click inline done task: %v", err)
	}
	tt.Frame()
	gotTask, err = app.Projects.GetTask(ctx, task.ID)
	if err != nil || gotTask.Status != model.Done {
		t.Fatalf("expected task moved to done, got %+v err=%v", gotTask, err)
	}
	if tt.HasText("Polish Focus Actions") {
		t.Fatal("completed task should leave unfinished Focus task list")
	}

	// Advance job stage inline (interview_1 -> interview_2).
	if err := tt.Click("推进阶段至 二面 Raycast"); err != nil {
		t.Fatalf("click inline advance job stage: %v", err)
	}
	tt.Frame()
	gotJob, err := app.JobApplications.Get(ctx, job.ID)
	if err != nil || gotJob.Stage != model.JobStageInterview2 {
		t.Fatalf("expected job advanced to interview_2, got %+v err=%v", gotJob, err)
	}
}
