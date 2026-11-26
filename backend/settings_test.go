package backend_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"aion/backend"
	"aion/backend/model"
)

// UpdateCheckPolicy persists enabled state and check interval while rejecting out-of-range durations.
func TestSettingsUpdateCheckPolicyRoundTripAndValidation(t *testing.T) {
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{
		DataDir:          t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	got, err := app.Settings.UpdateCheckPolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.Interval != 24*time.Hour {
		t.Fatalf("unexpected initial policy: %+v", got)
	}

	want := backend.UpdateCheckPolicy{Enabled: false, Interval: 12 * time.Hour}
	if err := app.Settings.SetUpdateCheckPolicy(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err = app.Settings.UpdateCheckPolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}

	for _, bad := range []time.Duration{0, -time.Hour, 30 * time.Second, 366 * 24 * time.Hour} {
		err := app.Settings.SetUpdateCheckPolicy(ctx, backend.UpdateCheckPolicy{Enabled: true, Interval: bad})
		if !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("expected ErrInvalid for interval %v, got %v", bad, err)
		}
	}
}

// HiddenModules defaults to empty, deduplicates and sorts persisted module IDs, and rejects invalid IDs.
func TestSettingsHiddenModulesRoundTripAndValidation(t *testing.T) {
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{
		DataDir:          t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	got, err := app.Settings.HiddenModules(ctx)
	if err != nil || len(got) != 0 {
		t.Fatalf("expected empty default hidden modules, got %v err=%v", got, err)
	}

	if err := app.Settings.SetHiddenModules(ctx, []string{"job_applications", "projects", "job_applications"}); err != nil {
		t.Fatal(err)
	}
	got, err = app.Settings.HiddenModules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "job_applications" || got[1] != "projects" {
		t.Fatalf("expected sorted deduplicated modules, got %v", got)
	}

	if err := app.Settings.SetHiddenModules(ctx, []string{"Bad Module!"}); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for bad module id, got %v", err)
	}
}
