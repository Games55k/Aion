package frontend

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
)

func drainTestWorkbench(w *Workbench, queue chan func()) {
	for {
		w.WaitForTasks()
		select {
		case fn := <-queue:
			fn()
		default:
			return
		}
	}
}

// External task mutations refresh the active Kanban board and clean task detail live, while never overwriting an unsaved user task draft.
func TestExternalChangeRefreshesTasksAndProtectsDirtyDraft(t *testing.T) {
	ctx := context.Background()
	queue := make(chan func(), 64)
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		Dispatch: func(fn func()) { queue <- fn },
	})

	proj, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "MCP Board"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.Projects.CreateTask(ctx, proj.ID, projects.TaskInput{
		Title:    "Original Task",
		Status:   model.Planned,
		Priority: model.Medium,
	})
	if err != nil {
		t.Fatal(err)
	}

	wb.HandleExternalChange(ExternalChange{Scope: "projects", ProjectID: proj.ID})
	drainTestWorkbench(wb, queue)
	wb.SelectProject(proj.ID)
	drainTestWorkbench(wb, queue)
	wb.SelectTask(task.ID)
	drainTestWorkbench(wb, queue)
	tt.Frame()

	// Clean open detail updates live when an external client modifies the task.
	if _, err := app.Projects.UpdateTask(ctx, task.ID, projects.TaskInput{
		Title:       "Updated By AI",
		Description: "Added notes",
		Status:      model.InProgress,
		Priority:    model.High,
	}); err != nil {
		t.Fatal(err)
	}
	wb.HandleExternalChange(ExternalChange{Scope: "tasks", ProjectID: proj.ID, TaskID: task.ID})
	drainTestWorkbench(wb, queue)
	if wb.editTitle != "Updated By AI" || wb.editDescription != "Added notes" {
		t.Fatalf("clean task detail did not sync external update: title=%q desc=%q", wb.editTitle, wb.editDescription)
	}

	// Simulate user typing an invalid/unsaved draft (whitespace title stays unsaved and dirty).
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatalf("focus task title: %v", err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("   ")
	tt.Frame()
	if !wb.hasActiveTaskDraft() {
		t.Fatal("expected active dirty task draft")
	}

	// External update arrives while draft is dirty: board reloads, but user draft must remain intact.
	if _, err := app.Projects.CreateTask(ctx, proj.ID, projects.TaskInput{
		Title:    "Second AI Task",
		Status:   model.Planned,
		Priority: model.Low,
	}); err != nil {
		t.Fatal(err)
	}
	wb.HandleExternalChange(ExternalChange{Scope: "tasks", ProjectID: proj.ID})
	drainTestWorkbench(wb, queue)

	if len(wb.boardTasks) != 2 {
		t.Fatalf("expected board to refresh with 2 tasks, got %d", len(wb.boardTasks))
	}
	if wb.editTitle != "   " || !wb.dirty {
		t.Fatalf("external change overwrote dirty task draft: title=%q dirty=%v", wb.editTitle, wb.dirty)
	}

	// Even if the selected task is externally trashed while dirty, do not silently discard the user's draft.
	if err := app.Projects.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	wb.HandleExternalChange(ExternalChange{Scope: "trash", ProjectID: proj.ID, TaskID: task.ID})
	drainTestWorkbench(wb, queue)
	if wb.selectedTaskID != task.ID || wb.editTitle != "   " {
		t.Fatalf("external trash discarded dirty task draft: selected=%q title=%q", wb.selectedTaskID, wb.editTitle)
	}
}

// External job application updates refresh the jobs list, stats, and open timeline live without overwriting an unsaved job draft.
func TestExternalChangeRefreshesJobsAndProtectsDirtyDraft(t *testing.T) {
	ctx := context.Background()
	queue := make(chan func(), 64)
	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		Dispatch: func(fn func()) { queue <- fn },
	})

	created, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "Linear",
		PositionName: "Product Engineer",
		AppliedDate:  "2026-11-25",
		Notes:        "Initial note",
	})
	if err != nil {
		t.Fatal(err)
	}

	wb.HandleExternalChange(ExternalChange{Scope: "job_applications", JobID: created.ID})
	drainTestWorkbench(wb, queue)
	if wb.jobStats.Total != 1 || len(wb.jobApps) != 1 {
		t.Fatalf("expected job list refreshed with 1 item, got total=%d len=%d", wb.jobStats.Total, len(wb.jobApps))
	}

	wb.NavigateTo(NavJobAll)
	drainTestWorkbench(wb, queue)
	wb.SelectJobApplication(created.ID)
	drainTestWorkbench(wb, queue)
	tt.Frame()

	// Clean open job detail refreshes stage and timeline live on external change.
	if _, err := app.JobApplications.ChangeStage(ctx, jobapplications.ChangeStageInput{
		ID:      created.ID,
		ToStage: model.JobStageInterview1,
		Note:    "AI recorded interview invite",
	}); err != nil {
		t.Fatal(err)
	}
	wb.HandleExternalChange(ExternalChange{Scope: "job_applications", JobID: created.ID})
	drainTestWorkbench(wb, queue)
	if wb.detailJob == nil || wb.detailJob.Stage != model.JobStageInterview1 {
		t.Fatalf("clean job detail did not update stage: %+v", wb.detailJob)
	}
	if len(wb.detailJobEvents) < 2 {
		t.Fatalf("expected updated timeline events, got %d", len(wb.detailJobEvents))
	}

	// Make job draft invalid/dirty (empty company name) so auto-save retains local draft.
	if err := tt.Click("详情公司名称"); err != nil {
		t.Fatalf("focus job company input: %v", err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("   ")
	tt.Frame()
	if !wb.hasActiveJobDraft() {
		t.Fatal("expected active dirty job draft")
	}

	// External timeline note added while user draft is dirty: timeline updates, draft input stays untouched.
	if _, err := app.JobApplications.AddNoteEvent(ctx, jobapplications.AddNoteInput{
		ApplicationID: created.ID,
		Note:          "Follow-up note from AI",
	}); err != nil {
		t.Fatal(err)
	}
	wb.HandleExternalChange(ExternalChange{Scope: "job_applications", JobID: created.ID})
	drainTestWorkbench(wb, queue)

	if wb.editJobCompany != "   " || !wb.jobDirty {
		t.Fatalf("external change overwrote dirty job draft: company=%q dirty=%v", wb.editJobCompany, wb.jobDirty)
	}
	if len(wb.detailJobEvents) < 3 {
		t.Fatalf("expected live timeline append while preserving draft, got %d events", len(wb.detailJobEvents))
	}
}

// Settings MCP card copies valid Antigravity JSON config and Codex CLI registration command (including custom -data-dir quoting).
func TestSettingsMCPCardCopiesAntigravityAndCodexConfig(t *testing.T) {
	info := MCPCapability{
		Ready:          true,
		SocketPath:     "/tmp/aion-custom/aion.sock",
		ExecutablePath: "/Applications/My Apps/Aion.app/Contents/MacOS/Aion",
		DataDirArg:     "/tmp/aion data",
	}
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		MCPInfo: info,
	})
	wb.NavigateTo(NavSettings)
	tt.Frame()
	tt.Scroll(500, 300, 0, 500)
	tt.Frame()

	if err := tt.Click("复制 Antigravity 配置"); err != nil {
		t.Fatalf("click 复制 Antigravity 配置: %v", err)
	}
	var parsed struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(tt.Clipboard()), &parsed); err != nil {
		t.Fatalf("copied Antigravity JSON is invalid: %v\n%s", err, tt.Clipboard())
	}
	aionCfg, ok := parsed.MCPServers["aion"]
	if !ok {
		t.Fatalf("missing mcpServers.aion entry: %+v", parsed)
	}
	if aionCfg.Command != info.ExecutablePath {
		t.Fatalf("unexpected command: got %q want %q", aionCfg.Command, info.ExecutablePath)
	}
	wantArgs := []string{"mcp", "-data-dir", "/tmp/aion data"}
	if strings.Join(aionCfg.Args, "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("unexpected args: got %v want %v", aionCfg.Args, wantArgs)
	}

	if err := tt.Click("复制 Codex 命令"); err != nil {
		t.Fatalf("click 复制 Codex 命令: %v", err)
	}
	wantCmd := "codex mcp add aion -- '/Applications/My Apps/Aion.app/Contents/MacOS/Aion' mcp -data-dir '/tmp/aion data'"
	if got := tt.Clipboard(); got != wantCmd {
		t.Fatalf("unexpected Codex command:\ngot:  %s\nwant: %s", got, wantCmd)
	}
}
