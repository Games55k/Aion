package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aion/backend"
	"aion/backend/model"
	"aion/ipc"
)

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatalf("cli %v failed: %v (stderr=%s)", args, err, stderr.String())
	}
	return stdout.String()
}

// CLI subcommands manage projects, cross-project tasks, attachments, job applications, and trash both offline and against a live GUI Unix socket.
func TestCLIEndToEndOfflineAndLiveSocket(t *testing.T) {
	root := t.TempDir()

	// Offline creation releases .aion.lock cleanly between invocations.
	out := runCLI(t, "project", "add", "-data-dir", root, "-name", "CLI Project", "-json")
	var proj model.Project
	if err := json.Unmarshal([]byte(out), &proj); err != nil {
		t.Fatalf("decode project JSON: %v\n%s", err, out)
	}

	out = runCLI(t, "task", "add", "-data-dir", root,
		"-project", proj.ID,
		"-title", "Ship CLI",
		"-priority", "high",
		"-tags", "cli,v0.2",
		"-due", "2026-12-01",
		"-json",
	)
	var task model.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("decode task JSON: %v\n%s", err, out)
	}

	// Attach a local regular file via CLI.
	sampleFile := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(sampleFile, []byte("hello aion attachment"), 0o600); err != nil {
		t.Fatal(err)
	}
	attOut := runCLI(t, "task", "attach", "-data-dir", root, "-id", task.ID, "-file", sampleFile, "-json")
	var att model.Attachment
	if err := json.Unmarshal([]byte(attOut), &att); err != nil {
		t.Fatalf("decode attachment JSON: %v\n%s", err, attOut)
	}
	if att.Name != "notes.txt" || att.TaskID != task.ID {
		t.Fatalf("unexpected attachment result: %+v", att)
	}

	// Start a live GUI IPC server and verify CLI commands route through it seamlessly.
	app, err := backend.Open(context.Background(), backend.Options{DataDir: root, DisableScheduler: true})
	if err != nil {
		t.Fatalf("open backend after offline CLI: %v", err)
	}
	defer app.Close()

	changes := 0
	srv, err := ipc.StartServer(root, ipc.NewHandler(app, func(ipc.ChangeEvent) {
		changes++
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Cross-project search via CLI.
	searchOut := runCLI(t, "task", "list", "-data-dir", root, "-tags", "cli", "-due-before", "2026-12-05", "-json")
	var items []ipc.TaskSearchItem
	if err := json.Unmarshal([]byte(searchOut), &items); err != nil {
		t.Fatalf("decode task list JSON: %v\n%s", err, searchOut)
	}
	if len(items) != 1 || items[0].Task.ID != task.ID || items[0].ProjectName != "CLI Project" {
		t.Fatalf("unexpected CLI task search items: %+v", items)
	}

	// Mark task done, create job, advance stage, add note, check summary.
	runCLI(t, "task", "done", "-data-dir", root, "-id", task.ID)
	jobOut := runCLI(t, "job", "add", "-data-dir", root, "-company", "Vercel", "-position", "Systems Engineer", "-date", "2026-11-26", "-json")
	var job model.JobApplication
	if err := json.Unmarshal([]byte(jobOut), &job); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "job", "stage", "-data-dir", root, "-id", job.ID, "-to", "interview_1", "-note", "Scheduled")
	runCLI(t, "job", "note", "-data-dir", root, "-id", job.ID, "-note", "Prep distributed systems")

	sumText := runCLI(t, "summary", "-data-dir", root)
	if !strings.Contains(sumText, "CLI Project") || !strings.Contains(sumText, "Vercel") {
		t.Fatalf("unexpected summary output:\n%s", sumText)
	}

	// Soft-delete task and restore via trash CLI.
	runCLI(t, "task", "trash", "-data-dir", root, "-id", task.ID)
	trashText := runCLI(t, "trash", "list", "-data-dir", root)
	if !strings.Contains(trashText, task.ID) {
		t.Fatalf("expected trashed task in trash list:\n%s", trashText)
	}
	runCLI(t, "trash", "restore", "-data-dir", root, "-kind", "task", "-id", task.ID)

	if changes < 6 {
		t.Fatalf("expected live IPC change notifications during CLI mutations, got %d", changes)
	}
}
