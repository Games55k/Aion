package main

import (
	"bytes"
	"context"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"aion/backend"
	"aion/backend/model"
	"aion/backend/projects"
	"aion/frontend"

	"github.com/egoist/mygo/ui"
)

// Regression: -snapshot could truncate the live SQLite database, including through an aliased data directory.
func TestSnapshotRejectsDataDirectoryTargets(t *testing.T) {
	for _, name := range []string{"direct", "symlink-parent", "case-alias"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			ctx := context.Background()
			app, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
			if err != nil {
				t.Fatal(err)
			}
			p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Keep me"})
			if err != nil {
				t.Fatal(err)
			}
			if err := app.Close(); err != nil {
				t.Fatal(err)
			}
			targetRoot := root
			if name == "symlink-parent" {
				targetRoot = filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(root, targetRoot); err != nil {
					t.Fatal(err)
				}
			}
			if name == "case-alias" {
				targetRoot = strings.ToUpper(root)
				original, err := os.Stat(root)
				if err != nil {
					t.Fatal(err)
				}
				aliased, err := os.Stat(targetRoot)
				if err != nil || !os.SameFile(original, aliased) {
					t.Skip("filesystem is case-sensitive")
				}
			}
			if err := run(root, filepath.Join(targetRoot, "aion.sqlite")); err == nil {
				t.Fatal("snapshot accepted a live database destination")
			}
			reopened, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
			if err != nil {
				t.Fatalf("snapshot damaged the database: %v", err)
			}
			defer reopened.Close()
			list, err := reopened.Projects.ListProjects(ctx, projects.ProjectFilter{})
			if err != nil || !slices.ContainsFunc(list, func(got model.Project) bool {
				return got.ID == p.ID && got.Name == p.Name
			}) {
				t.Fatalf("snapshot lost project data: %+v, %v", list, err)
			}
		})
	}
}

// Snapshot replacement writes a complete PNG without changing the inode of an existing hard-linked source.
func TestSnapshotPublishesWithoutTruncatingLinkedFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "managed-file")
	if err := os.WriteFile(source, []byte("keep original data"), 0o600); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	destination := filepath.Join(outputDir, "snapshot.png")
	if err := os.Link(source, destination); err != nil {
		t.Fatal(err)
	}
	if err := writeSnapshot(destination, root, func(c *ui.Context) { ui.Text(c, "snapshot") }); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(source); err != nil || string(b) != "keep original data" {
		t.Fatalf("snapshot truncated linked data: %q, %v", b, err)
	}
	f, err := os.Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil || img.Bounds().Dx() != 1360 || img.Bounds().Dy() != 860 {
		t.Fatalf("invalid snapshot output: %v", err)
	}
	if entries, err := os.ReadDir(outputDir); err != nil || len(entries) != 1 {
		t.Fatalf("snapshot left staging files: %v, %v", entries, err)
	}
}

// Regression: OnQuit closed the backend before run's deferred cleanup flushed pending edits a second time against the closed store.
func TestShutdownFlushesAndClosesOnce(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{DataDir: root, DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	p, err := app.Projects.CreateProject(ctx, projects.ProjectInput{Name: "Shutdown"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := app.Projects.CreateTask(ctx, p.ID, projects.TaskInput{
		Title:    "Initial",
		Status:   model.Planned,
		Priority: model.Medium,
	})
	if err != nil {
		t.Fatal(err)
	}

	queue := make(chan func(), 32)
	drain := func(w *frontend.Workbench) {
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
	wb := frontend.New(app, frontend.Host{
		Dispatch: func(fn func()) { queue <- fn },
	})
	wb.SelectProject(p.ID)
	drain(wb)
	wb.SelectTask(task.ID)
	drain(wb)
	tt := ui.NewTester(wb.View, 1320, 840)
	tt.Frame()
	if err := tt.Click("任务详情标题"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("   ")

	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOut) })

	shutdown := newShutdownFunc(wb, app)
	shutdown()
	shutdown()

	if got := strings.Count(buf.String(), "failed to flush pending edits"); got != 1 {
		t.Fatalf("expected exactly 1 flush log across OnQuit and defer, got %d (%q)", got, buf.String())
	}
	if strings.Contains(buf.String(), "closed") {
		t.Fatalf("unexpected closed-store error during shutdown: %q", buf.String())
	}
}

// The `mcp` subcommand serves JSON-RPC over stdio against -data-dir without printing diagnostics to stdout.
func TestRunMCPSubcommandStdioRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_workspace_summary","arguments":{}}}` + "\n",
	)
	var out bytes.Buffer
	if err := runMCP([]string{"-data-dir", root}, in, &out); err != nil {
		t.Fatalf("runMCP: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON-RPC response lines, got %d:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("unexpected initialize response: %s", lines[0])
	}
	if !strings.Contains(lines[1], `\"trash_count\": 0`) || !strings.Contains(lines[1], `"isError":false`) {
		t.Fatalf("unexpected workspace summary response: %s", lines[1])
	}
}
