package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"aion/backend"
	"aion/frontend"
)

func waitUpdateSnapshot(t *testing.T, ch <-chan frontend.UpdateCapability, desc string, match func(frontend.UpdateCapability) bool) frontend.UpdateCapability {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case snap := <-ch:
			if match(snap) {
				return snap
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for update snapshot: %s", desc)
		}
	}
}

// Silent background checks discover releases without auto-installing, persist skipped versions, and still show skipped versions on manual checks.
func TestUpdateControllerSilentCheckDiscoversReleaseAndRespectsSkip(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	var mu sync.Mutex
	currentRel := &updateRelease{Version: "0.2.0", Notes: "## Changes\n- Added feature"}
	installCalls := 0
	currentRel.Install = func(context.Context, func(int64, int64)) error {
		mu.Lock()
		installCalls++
		mu.Unlock()
		return nil
	}
	checkGate := make(chan struct{}, 4)
	checkGate <- struct{}{} // Allow the launch check.

	snaps := make(chan frontend.UpdateCapability, 32)
	ctrl := newUpdateController(updateControllerOptions{
		DataDir:         dir,
		Version:         "0.1.6",
		CanSelfUpdate:   true,
		Policy:          backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour},
		FirstCheckDelay: time.Millisecond,
		Now:             func() time.Time { return now },
		Check: func(ctx context.Context) (*updateRelease, error) {
			select {
			case <-checkGate:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			mu.Lock()
			defer mu.Unlock()
			return currentRel, nil
		},
		OnChange: func(info frontend.UpdateCapability) {
			snaps <- info
		},
	})
	defer ctrl.Close()

	ctrl.Start()

	got := waitUpdateSnapshot(t, snaps, "launch silent check finds 0.2.0", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})
	if got.AvailableNotes != "## Changes\n- Added feature" || !got.LastCheck.Equal(now) {
		t.Fatalf("unexpected discovered release snapshot: %+v", got)
	}
	mu.Lock()
	if installCalls != 0 {
		t.Fatalf("silent check must not auto-install, got %d installs", installCalls)
	}
	mu.Unlock()

	// Skipping 0.2.0 clears the indicator and persists skipped_version.
	ctrl.SkipVersion("0.2.0")
	got = waitUpdateSnapshot(t, snaps, "skip clears available version", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateIdle && s.AvailableVersion == ""
	})
	if got.AvailableVersion != "" {
		t.Fatalf("expected empty available version after skip, got %+v", got)
	}

	// A new controller reading disk state suppresses 0.2.0 on silent checks, but surfaces it on manual check.
	ctrl.Close()
	snaps2 := make(chan frontend.UpdateCapability, 32)
	checkGate2 := make(chan struct{}, 4)
	checkGate2 <- struct{}{}
	ctrl2 := newUpdateController(updateControllerOptions{
		DataDir:         dir,
		Version:         "0.1.6",
		CanSelfUpdate:   true,
		Policy:          backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour},
		FirstCheckDelay: time.Millisecond,
		Now:             func() time.Time { return now.Add(time.Hour) },
		Check: func(ctx context.Context) (*updateRelease, error) {
			select {
			case <-checkGate2:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			mu.Lock()
			defer mu.Unlock()
			return currentRel, nil
		},
		OnChange: func(info frontend.UpdateCapability) {
			snaps2 <- info
		},
	})
	defer ctrl2.Close()
	ctrl2.Start()

	waitUpdateSnapshot(t, snaps2, "silent check updates LastCheck while keeping skipped version hidden", func(s frontend.UpdateCapability) bool {
		return s.LastCheck.Equal(now.Add(time.Hour)) && s.AvailableVersion == "" && s.Status == frontend.UpdateIdle
	})

	// Manual check still surfaces skipped 0.2.0.
	checkGate2 <- struct{}{}
	ctrl2.CheckForUpdates()
	waitUpdateSnapshot(t, snaps2, "manual check shows skipped 0.2.0", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})

	// A newer version (0.3.0) is surfaced even on a silent check.
	mu.Lock()
	currentRel = &updateRelease{Version: "0.3.0", Notes: "Newer"}
	mu.Unlock()
	checkGate2 <- struct{}{}
	ctrl2.beginCheck(false, 0)
	waitUpdateSnapshot(t, snaps2, "silent check surfaces newer 0.3.0", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.3.0"
	})
}

// Background check errors stay silent and preserve a previously discovered version, while manual checks promote an in-flight silent check without duplicate requests.
func TestUpdateControllerSilentFailureAndSingleFlightPromotion(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	started := make(chan int, 4)
	release := make(chan struct{})
	var mu sync.Mutex
	callCount := 0
	var nextRel *updateRelease
	var nextErr error

	snaps := make(chan frontend.UpdateCapability, 32)
	ctrl := newUpdateController(updateControllerOptions{
		DataDir:         dir,
		Version:         "0.1.6",
		CanSelfUpdate:   true,
		Policy:          backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour},
		FirstCheckDelay: time.Hour,
		Now:             func() time.Time { return now },
		Check: func(ctx context.Context) (*updateRelease, error) {
			mu.Lock()
			callCount++
			n := callCount
			rel, err := nextRel, nextErr
			mu.Unlock()
			started <- n
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return rel, err
		},
		OnChange: func(info frontend.UpdateCapability) {
			snaps <- info
		},
	})
	defer ctrl.Close()
	ctrl.Start()

	// Seed an available update via silent check.
	mu.Lock()
	nextRel, nextErr = &updateRelease{Version: "0.2.0"}, nil
	mu.Unlock()
	ctrl.beginCheck(false, 0)
	<-started
	release <- struct{}{}
	waitUpdateSnapshot(t, snaps, "seed 0.2.0", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})

	// A failing silent check must not clear 0.2.0 or surface UpdateError.
	mu.Lock()
	nextRel, nextErr = nil, errors.New("network offline")
	mu.Unlock()
	ctrl.beginCheck(false, 0)
	<-started
	release <- struct{}{}
	// Wait until the check goroutine finishes.
	deadline := time.Now().Add(2 * time.Second)
	for {
		ctrl.mu.Lock()
		busy := ctrl.checking
		ctrl.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("silent failing check did not finish")
		}
		time.Sleep(time.Millisecond)
	}
	if snap := ctrl.Snapshot(); snap.Status != frontend.UpdateAvailable || snap.AvailableVersion != "0.2.0" || snap.LastError != "" {
		t.Fatalf("silent error clobbered available update: %+v", snap)
	}

	// Start a silent check and promote it with CheckForUpdates; only one Check call should run.
	mu.Lock()
	nextRel, nextErr = nil, nil
	mu.Unlock()
	ctrl.beginCheck(false, 0)
	if n := <-started; n != 3 {
		t.Fatalf("expected 3rd check call, got %d", n)
	}
	ctrl.CheckForUpdates()
	waitUpdateSnapshot(t, snaps, "promoted to checking", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateChecking
	})
	release <- struct{}{}
	waitUpdateSnapshot(t, snaps, "promoted check finishes up-to-date", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateUpToDate && s.AvailableVersion == ""
	})
	mu.Lock()
	if callCount != 3 {
		t.Fatalf("expected single-flight reuse with 3 total checks, got %d", callCount)
	}
	mu.Unlock()
}

// Installing emits download progress, supports canceling during download, ignores cancel during final swap, and enables relaunch once ready.
func TestUpdateControllerInstallProgressCancelAndRelaunch(t *testing.T) {
	dir := t.TempDir()
	snaps := make(chan frontend.UpdateCapability, 64)

	installBlock := make(chan struct{})
	installStep := make(chan struct{})
	var mu sync.Mutex
	cancelMode := true
	relaunchCalls := 0

	rel := &updateRelease{
		Version: "0.2.0",
		Notes:   "Release notes",
		Install: func(ctx context.Context, progress func(downloaded, total int64)) error {
			progress(25*1024*1024, 100*1024*1024)
			installStep <- struct{}{}
			mu.Lock()
			waitCancel := cancelMode
			mu.Unlock()
			if waitCancel {
				<-ctx.Done()
				return ctx.Err()
			}
			progress(100*1024*1024, 100*1024*1024)
			installStep <- struct{}{}
			<-installBlock
			return nil
		},
	}

	ctrl := newUpdateController(updateControllerOptions{
		DataDir:       dir,
		Version:       "0.1.6",
		CanSelfUpdate: true,
		Policy:        backend.UpdateCheckPolicy{Enabled: false, Interval: 24 * time.Hour},
		Check:         func(context.Context) (*updateRelease, error) { return rel, nil },
		Relaunch: func() {
			mu.Lock()
			relaunchCalls++
			mu.Unlock()
		},
		OnChange: func(info frontend.UpdateCapability) { snaps <- info },
	})
	defer ctrl.Close()
	ctrl.Start()

	ctrl.CheckForUpdates()
	waitUpdateSnapshot(t, snaps, "available 0.2.0", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})

	// Relaunch before installation must be a no-op.
	ctrl.Relaunch()
	if relaunchCalls != 0 {
		t.Fatalf("expected no relaunch before install, got %d", relaunchCalls)
	}

	// Cancel during download restores UpdateAvailable.
	ctrl.InstallUpdate()
	<-installStep
	waitUpdateSnapshot(t, snaps, "downloading 25%", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateDownloading && s.DownloadedBytes == 25*1024*1024
	})
	ctrl.CancelUpdate()
	waitUpdateSnapshot(t, snaps, "canceled download returns to available", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})

	// Complete install: CancelUpdate during UpdateInstalling must be ignored, then UpdateReady enables Relaunch.
	mu.Lock()
	cancelMode = false
	mu.Unlock()
	ctrl.InstallUpdate()
	<-installStep // 25%
	<-installStep // 100% -> UpdateInstalling
	waitUpdateSnapshot(t, snaps, "installing stage", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateInstalling
	})
	ctrl.CancelUpdate()
	close(installBlock)
	waitUpdateSnapshot(t, snaps, "ready to relaunch", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateReady && s.InstalledVersion == "0.2.0"
	})

	ctrl.Relaunch()
	mu.Lock()
	if relaunchCalls != 1 {
		t.Fatalf("expected 1 relaunch call, got %d", relaunchCalls)
	}
	mu.Unlock()
}

// Legacy updater.json state migrates cleanly, disabled policy stays silent, and clock-backwards scheduling stays bounded.
func TestUpdateControllerLegacyMigrationAndScheduleCalculation(t *testing.T) {
	dir := t.TempDir()
	lastCheck := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	legacyBytes, err := json.Marshal(legacyUpdaterDiskState{
		LastCheck:      lastCheck,
		SkippedVersion: "0.1.7",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, legacyUpdaterStateFilename), legacyBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	ctrl := newUpdateController(updateControllerOptions{
		DataDir:       dir,
		Version:       "0.1.6",
		CanSelfUpdate: true,
		Policy:        backend.UpdateCheckPolicy{Enabled: false, Interval: 24 * time.Hour},
	})
	defer ctrl.Close()

	snap := ctrl.Snapshot()
	if !snap.LastCheck.Equal(lastCheck) {
		t.Fatalf("expected migrated LastCheck %v, got %v", lastCheck, snap.LastCheck)
	}
	if ctrl.disk.SkippedVersion != "0.1.7" {
		t.Fatalf("expected migrated SkippedVersion 0.1.7, got %q", ctrl.disk.SkippedVersion)
	}

	// Backwards clock adjustment clamps lastCheck to now instead of delaying checks far into the future.
	now := time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)
	due := nextUpdateCheck(now.Add(48*time.Hour), time.Time{}, 12*time.Hour, time.Hour, now)
	if !due.Equal(now.Add(12 * time.Hour)) {
		t.Fatalf("expected backwards clock due=%v, got %v", now.Add(12*time.Hour), due)
	}

	// Failure backoff delays retry by min(interval, retryBackoff).
	dueRetry := nextUpdateCheck(now.Add(-24*time.Hour), now.Add(-10*time.Minute), 24*time.Hour, time.Hour, now)
	if !dueRetry.Equal(now.Add(50 * time.Minute)) {
		t.Fatalf("expected retry backoff due=%v, got %v", now.Add(50*time.Minute), dueRetry)
	}
}

// Regression: a background check finishing while an install was already downloading previously overwrote the download state, replaced the version, and cleared the install cancel handle.
func TestUpdateControllerBackgroundCheckDoesNotOverwriteActiveDownload(t *testing.T) {
	dir := t.TempDir()
	snaps := make(chan frontend.UpdateCapability, 64)

	checkStarted := make(chan struct{}, 2)
	checkRelease := make(chan struct{})
	checkDone := make(chan struct{})
	downloadStarted := make(chan struct{})
	downloadCanceled := make(chan struct{})

	v2 := &updateRelease{
		Version: "0.2.0",
		Install: func(ctx context.Context, progress func(downloaded, total int64)) error {
			progress(10*1024*1024, 100*1024*1024)
			close(downloadStarted)
			<-ctx.Done()
			close(downloadCanceled)
			return ctx.Err()
		},
	}
	v3 := &updateRelease{Version: "0.3.0"}

	var mu sync.Mutex
	checkNum := 0
	ctrl := newUpdateController(updateControllerOptions{
		DataDir:         dir,
		Version:         "0.1.6",
		CanSelfUpdate:   true,
		Policy:          backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour},
		FirstCheckDelay: time.Hour,
		Check: func(ctx context.Context) (*updateRelease, error) {
			mu.Lock()
			checkNum++
			n := checkNum
			mu.Unlock()
			if n == 1 {
				return v2, nil
			}
			checkStarted <- struct{}{}
			// Ignore ctx cancellation and still return v0.3.0 to simulate a check finishing right after InstallUpdate starts.
			<-checkRelease
			defer close(checkDone)
			return v3, nil
		},
		OnChange: func(info frontend.UpdateCapability) { snaps <- info },
	})
	defer ctrl.Close()
	ctrl.Start()

	ctrl.CheckForUpdates()
	waitUpdateSnapshot(t, snaps, "initial 0.2.0 available", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})

	// Start a background check, then start installing v0.2.0 while the check is still in flight.
	ctrl.beginCheck(false, 0)
	<-checkStarted
	ctrl.InstallUpdate()
	<-downloadStarted
	waitUpdateSnapshot(t, snaps, "downloading 0.2.0", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateDownloading && s.AvailableVersion == "0.2.0"
	})

	// Let the background check return v0.3.0; it must not clobber the active v0.2.0 download or cancel handle.
	close(checkRelease)
	<-checkDone
	snap := ctrl.Snapshot()
	if snap.Status != frontend.UpdateDownloading || snap.AvailableVersion != "0.2.0" {
		t.Fatalf("background check clobbered active download: %+v", snap)
	}

	// CancelUpdate must still reach the active v0.2.0 download.
	ctrl.CancelUpdate()
	select {
	case <-downloadCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("CancelUpdate failed to cancel active download after background check finished")
	}
	waitUpdateSnapshot(t, snaps, "canceled download returns to 0.2.0 available", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})
}

// Regression: canceling right as the last chunk completed previously checked ctx.Err() first and hid the staged Ready state even when Install returned nil.
func TestUpdateControllerCancelRacingWithSuccessfulInstallStaysReady(t *testing.T) {
	dir := t.TempDir()
	snaps := make(chan frontend.UpdateCapability, 32)
	enteredInstall := make(chan struct{})
	finishInstall := make(chan struct{})

	rel := &updateRelease{
		Version: "0.2.0",
		Install: func(ctx context.Context, progress func(downloaded, total int64)) error {
			progress(90*1024*1024, 100*1024*1024)
			close(enteredInstall)
			<-ctx.Done()
			<-finishInstall
			// Simulate MyGo finishing staging the update and returning nil despite late cancellation.
			return nil
		},
	}

	ctrl := newUpdateController(updateControllerOptions{
		DataDir:       dir,
		Version:       "0.1.6",
		CanSelfUpdate: true,
		Policy:        backend.UpdateCheckPolicy{Enabled: false, Interval: 24 * time.Hour},
		Check:         func(context.Context) (*updateRelease, error) { return rel, nil },
		OnChange:      func(info frontend.UpdateCapability) { snaps <- info },
	})
	defer ctrl.Close()
	ctrl.Start()

	ctrl.CheckForUpdates()
	waitUpdateSnapshot(t, snaps, "0.2.0 available", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateAvailable && s.AvailableVersion == "0.2.0"
	})

	ctrl.InstallUpdate()
	<-enteredInstall
	ctrl.CancelUpdate()
	close(finishInstall)

	got := waitUpdateSnapshot(t, snaps, "staged install enters Ready despite late cancel", func(s frontend.UpdateCapability) bool {
		return s.Status == frontend.UpdateReady && s.InstalledVersion == "0.2.0"
	})
	if got.InstalledVersion != "0.2.0" {
		t.Fatalf("expected InstalledVersion 0.2.0, got %+v", got)
	}
}

// Regression: an already-fired timer callback previously initiated a network check after SetPolicy disabled automatic checks.
func TestUpdateControllerStaleTimerDoesNotRunAfterDisablingAutoCheck(t *testing.T) {
	dir := t.TempDir()
	var mu sync.Mutex
	checkCalls := 0

	ctrl := newUpdateController(updateControllerOptions{
		DataDir:         dir,
		Version:         "0.1.6",
		CanSelfUpdate:   true,
		Policy:          backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour},
		FirstCheckDelay: time.Hour,
		Check: func(context.Context) (*updateRelease, error) {
			mu.Lock()
			checkCalls++
			mu.Unlock()
			return nil, nil
		},
	})
	defer ctrl.Close()
	ctrl.Start()

	ctrl.mu.Lock()
	staleGen := ctrl.timerGen
	ctrl.mu.Unlock()

	// Disable automatic checks, then fire the already-triggered timer callback with the old generation.
	ctrl.SetPolicy(backend.UpdateCheckPolicy{Enabled: false, Interval: 24 * time.Hour})
	ctrl.beginCheck(false, staleGen)

	// Also verify re-enabling with a new timer invalidates the older timer generation.
	ctrl.SetPolicy(backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour})
	ctrl.beginCheck(false, staleGen)

	ctrl.Close()
	mu.Lock()
	if checkCalls != 0 {
		t.Fatalf("stale timer callback must not initiate network checks, got %d calls", checkCalls)
	}
	mu.Unlock()
}

// Regression: concurrent disk writes from check completion and SkipVersion previously raced on rename so an older check snapshot could wipe out SkippedVersion.
func TestUpdateControllerConcurrentDiskPersistencePreservesSkippedVersion(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	firstWriteEntered := make(chan struct{})
	releaseFirstWrite := make(chan struct{})
	var hookOnce sync.Once

	snaps := make(chan frontend.UpdateCapability, 32)
	ctrl := newUpdateController(updateControllerOptions{
		DataDir:         dir,
		Version:         "0.1.6",
		CanSelfUpdate:   true,
		Policy:          backend.UpdateCheckPolicy{Enabled: true, Interval: 24 * time.Hour},
		FirstCheckDelay: time.Hour,
		Now:             func() time.Time { return now },
		Check: func(context.Context) (*updateRelease, error) {
			return &updateRelease{Version: "0.2.0"}, nil
		},
		OnChange: func(info frontend.UpdateCapability) { snaps <- info },
	})
	ctrl.saveDiskHook = func(updaterDiskState) {
		hookOnce.Do(func() {
			close(firstWriteEntered)
			<-releaseFirstWrite
		})
	}
	defer ctrl.Close()
	ctrl.Start()

	// Start a check and pause its disk write right before saving.
	ctrl.CheckForUpdates()
	<-firstWriteEntered

	// Concurrently skip 0.2.0 while the check's disk persistence is still blocked.
	skipDone := make(chan struct{})
	go func() {
		defer close(skipDone)
		ctrl.SkipVersion("0.2.0")
	}()

	// Wait until SkipVersion has updated in-memory state and is waiting on diskMu, then release the first write.
	deadline := time.Now().Add(2 * time.Second)
	for {
		ctrl.mu.Lock()
		skipped := ctrl.disk.SkippedVersion
		ctrl.mu.Unlock()
		if skipped == "0.2.0" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("SkipVersion did not update disk state in time")
		}
		time.Sleep(time.Millisecond)
	}
	close(releaseFirstWrite)
	<-skipDone
	ctrl.Close()

	// Reload from disk in a fresh controller: both LastCheck and SkippedVersion must be preserved.
	reloaded := newUpdateController(updateControllerOptions{
		DataDir:       dir,
		Version:       "0.1.6",
		CanSelfUpdate: true,
		Policy:        backend.UpdateCheckPolicy{Enabled: false, Interval: 24 * time.Hour},
	})
	defer reloaded.Close()

	if reloaded.disk.SkippedVersion != "0.2.0" || !reloaded.disk.LastCheck.Equal(now) {
		t.Fatalf("persisted disk state lost SkippedVersion or LastCheck: %+v", reloaded.disk)
	}
}
