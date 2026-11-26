package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"aion/backend"
	"aion/backend/model"
	"aion/backend/projects"
)

func strPtr(s string) *string { return &s }

// IPC over a live Unix domain socket executes safe project, task, job, and trash operations, preserves untouched fields on partial updates, and emits change events.
func TestLiveServerSocketOperationsAndPatchSemantics(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	fixedNow := func() time.Time { return time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC) }

	app, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true, Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	var (
		mu     sync.Mutex
		events []ChangeEvent
	)
	srv, err := StartServer(root, NewHandler(app, func(ev ChangeEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	router, err := NewRouter(RouterOptions{DataDir: root, Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}

	// Create project via socket.
	raw, err := router.Call(ctx, MethodProjectsCreate, CreateProjectParams{
		Name:        "Aion MCP",
		Icon:        "rocket",
		Description: "Original description",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	var proj model.Project
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatal(err)
	}

	// Partial project update must keep icon and description intact.
	raw, err = router.Call(ctx, MethodProjectsUpdate, UpdateProjectParams{
		ID:   proj.ID,
		Name: strPtr("Aion Workbench"),
	})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatal(err)
	}
	if proj.Name != "Aion Workbench" || proj.Icon != "rocket" || proj.Description != "Original description" {
		t.Fatalf("partial project update clobbered fields: %+v", proj)
	}

	// Create task and verify partial update preserves description, tags, subtasks, and due_date.
	raw, err = router.Call(ctx, MethodTasksCreate, CreateTaskParams{
		ProjectID:   proj.ID,
		Title:       "Draft IPC",
		Description: "## Notes\nKeep me",
		Status:      model.Planned,
		Priority:    model.High,
		Tags:        []string{"backend", "mcp"},
		Subtasks:    []model.Subtask{{Title: "Write server", Done: false}},
		DueDate:     "2026-12-01",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	var task model.Task
	if err := json.Unmarshal(raw, &task); err != nil {
		t.Fatal(err)
	}

	raw, err = router.Call(ctx, MethodTasksUpdate, UpdateTaskParams{
		ID:    task.ID,
		Title: strPtr("Ship IPC"),
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if err := json.Unmarshal(raw, &task); err != nil {
		t.Fatal(err)
	}
	if task.Title != "Ship IPC" || task.Description != "## Notes\nKeep me" || len(task.Tags) != 2 || len(task.Subtasks) != 1 || task.DueDate != "2026-12-01" {
		t.Fatalf("partial task update lost fields: %+v", task)
	}

	// Create job application, patch notes, advance stage, add timeline note, backfill, close, reopen.
	raw, err = router.Call(ctx, MethodJobsCreate, CreateJobParams{
		CompanyName:    "Acme",
		PositionName:   "Staff Engineer",
		AppliedDate:    "2026-11-20",
		ApplicationURL: "https://acme.example/jobs/1",
		Channel:        "Referral",
		Notes:          "Initial notes",
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	var job model.JobApplication
	if err := json.Unmarshal(raw, &job); err != nil {
		t.Fatal(err)
	}
	raw, err = router.Call(ctx, MethodJobsUpdate, UpdateJobParams{
		ID:    job.ID,
		Notes: strPtr("Updated notes"),
	})
	if err != nil {
		t.Fatalf("update job: %v", err)
	}
	if err := json.Unmarshal(raw, &job); err != nil {
		t.Fatal(err)
	}
	if job.CompanyName != "Acme" || job.ApplicationURL != "https://acme.example/jobs/1" || job.Channel != "Referral" || job.Notes != "Updated notes" {
		t.Fatalf("partial job update lost fields: %+v", job)
	}
	if _, err := router.Call(ctx, MethodJobsChangeStage, ChangeJobStageParams{ID: job.ID, ToStage: model.JobStageInterview1, Note: "Phone screen passed"}); err != nil {
		t.Fatalf("change stage: %v", err)
	}
	if _, err := router.Call(ctx, MethodJobsAddNote, AddJobNoteParams{ApplicationID: job.ID, Note: "Prepare system design"}); err != nil {
		t.Fatalf("add note: %v", err)
	}
	if _, err := router.Call(ctx, MethodJobsBackfillStage, BackfillJobStageParams{ApplicationID: job.ID, Stage: model.JobStageScreening, OccurredDate: "2026-11-21", Note: "HR review"}); err != nil {
		t.Fatalf("backfill stage: %v", err)
	}

	// Workspace summary returns projects, task counts, and active jobs.
	raw, err = router.Call(ctx, MethodWorkspaceSummary, nil)
	if err != nil {
		t.Fatalf("workspace summary: %v", err)
	}
	var summary WorkspaceSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Projects) != 1 || summary.Projects[0].Total != 1 || summary.JobStats.Active != 1 || len(summary.ActiveJobs) != 1 {
		t.Fatalf("unexpected workspace summary: %+v", summary)
	}

	// Soft-delete task and restore it from trash.
	if _, err := router.Call(ctx, MethodTasksTrash, IDParams{ID: task.ID}); err != nil {
		t.Fatalf("trash task: %v", err)
	}
	raw, err = router.Call(ctx, MethodTrashList, nil)
	if err != nil {
		t.Fatalf("list trash: %v", err)
	}
	var trash []model.TrashItem
	if err := json.Unmarshal(raw, &trash); err != nil {
		t.Fatal(err)
	}
	if len(trash) != 1 || trash[0].ID != task.ID || trash[0].Kind != projects.TrashTask {
		t.Fatalf("unexpected trash items: %+v", trash)
	}
	if _, err := router.Call(ctx, MethodTrashRestore, RestoreTrashParams{Kind: projects.TrashTask, ID: task.ID}); err != nil {
		t.Fatalf("restore task: %v", err)
	}

	// Unknown / destructive methods are rejected.
	if _, err := router.Call(ctx, "trash.clear", nil); err == nil {
		t.Fatal("expected unknown method error for trash.clear")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) < 8 {
		t.Fatalf("expected change events for mutations, got %d", len(events))
	}
}

// When the GUI is offline, Router opens backend.App per call and releases .aion.lock immediately so the GUI can start afterwards without conflict.
func TestRouterOfflineFallbackReleasesLockForGUI(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	router, err := NewRouter(RouterOptions{DataDir: root})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := router.Call(ctx, MethodProjectsCreate, CreateProjectParams{Name: "Offline Created"})
	if err != nil {
		t.Fatalf("offline create project: %v", err)
	}
	var created model.Project
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}

	// GUI opens the same directory immediately after the offline MCP call without hitting a lock error.
	app, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
	if err != nil {
		t.Fatalf("GUI could not open data dir after offline router call: %v", err)
	}
	defer app.Close()

	list, err := app.Projects.ListProjects(ctx, projects.ProjectFilter{})
	if err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("GUI did not see offline-created project: %+v, err=%v", list, err)
	}
}

// Concurrent offline calls contend cleanly on .aion.lock via bounded retry without failing.
func TestRouterConcurrentOfflineCallsRetryLock(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	router, err := NewRouter(RouterOptions{DataDir: root, RetryTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	const workers = 6
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := router.Call(ctx, MethodProjectsCreate, CreateProjectParams{
				Name: "Concurrent " + string(rune('A'+idx)),
			})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent offline call failed: %v", err)
	}

	raw, err := router.Call(ctx, MethodProjectsList, nil)
	if err != nil {
		t.Fatal(err)
	}
	var list []model.Project
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != workers {
		t.Fatalf("expected %d projects, got %d", workers, len(list))
	}
}

// StartServer replaces a stale Unix socket left by a crashed process and rejects symlinks at the socket path.
func TestStartServerCleansStaleSocketAndRejectsSymlink(t *testing.T) {
	ctx := context.Background()
	// Use a short directory under /tmp so the direct <dataDir>/aion.sock path is exercised.
	root, err := os.MkdirTemp("/tmp", "aion-ipc-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	app, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	sockPath, err := SocketPath(root)
	if err != nil {
		t.Fatal(err)
	}

	// Create a real Unix socket file and close its listener without deleting the file (simulating a crash).
	staleLn, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	if uln, ok := staleLn.(*net.UnixListener); ok {
		uln.SetUnlinkOnClose(false)
	}
	_ = staleLn.Close()

	srv, err := StartServer(root, NewHandler(app, nil))
	if err != nil {
		t.Fatalf("StartServer should replace stale socket: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Fatalf("Close server: %v", err)
	}
	if _, err := os.Lstat(sockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected socket removed on Close, got err=%v", err)
	}

	// Symlink at socket path must be rejected.
	decoy := filepath.Join(root, "decoy")
	if err := os.WriteFile(decoy, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(decoy, sockPath); err != nil {
		t.Fatal(err)
	}
	if _, err := StartServer(root, NewHandler(app, nil)); err == nil {
		t.Fatal("expected StartServer to reject symlink socket path")
	}
}
