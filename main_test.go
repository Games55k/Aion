package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aion/backend"
	"aion/backend/projects"

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
			if got, err := reopened.Projects.GetProject(ctx, p.ID); err != nil || got.Name != p.Name {
				t.Fatalf("snapshot lost project data: %+v, %v", got, err)
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
