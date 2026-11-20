package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aion/backend"
	"aion/frontend"
)

const (
	updaterStateFilename       = ".updater-state.json"
	legacyUpdaterStateFilename = "updater.json"
	defaultFirstCheckDelay     = 10 * time.Second
	defaultRetryBackoff        = time.Hour
	progressNotifyMinBytes     = 64 * 1024
)

type updateRelease struct {
	Version string
	Notes   string
	Date    time.Time
	Install func(ctx context.Context, progress func(downloaded, total int64)) error
}

type updaterDiskState struct {
	LastCheck      time.Time `json:"last_check,omitempty"`
	SkippedVersion string    `json:"skipped_version,omitempty"`
}

type legacyUpdaterDiskState struct {
	LastCheck      time.Time `json:"lastCheck,omitempty"`
	SkippedVersion string    `json:"skippedVersion,omitempty"`
}

type updateControllerOptions struct {
	DataDir         string
	Version         string
	IsDev           bool
	CanSelfUpdate   bool
	ReleasesURL     string
	Policy          backend.UpdateCheckPolicy
	FirstCheckDelay time.Duration
	RetryBackoff    time.Duration
	Now             func() time.Time
	Check           func(ctx context.Context) (*updateRelease, error)
	Relaunch        func()
	OnChange        func(frontend.UpdateCapability)
}

// updateController schedules silent background update checks and drives
// user-initiated downloads, installs, skips, and relaunches without ever
// holding backend SQLite Store locks.
type updateController struct {
	dataDir         string
	version         string
	isDev           bool
	canSelfUpdate   bool
	releasesURL     string
	firstCheckDelay time.Duration
	retryBackoff    time.Duration
	now             func() time.Time
	check           func(ctx context.Context) (*updateRelease, error)
	relaunch        func()
	onChange        func(frontend.UpdateCapability)

	emitMu sync.Mutex
	diskMu sync.Mutex

	// saveDiskHook is used by tests to deterministically interleave concurrent disk writes.
	saveDiskHook func(updaterDiskState)

	mu               sync.Mutex
	policy           backend.UpdateCheckPolicy
	disk             updaterDiskState
	status           frontend.UpdateStatus
	available        *updateRelease
	installedVersion string
	downloadedBytes  int64
	totalBytes       int64
	lastError        string
	checking         bool
	checkingUser     bool
	checkGen         uint64
	checkCancel      context.CancelFunc
	installing       bool
	installCancel    context.CancelFunc
	failedAt         time.Time
	launchChecked    bool
	started          bool
	closed           bool
	timer            *time.Timer
	timerGen         uint64

	wg sync.WaitGroup
}

func newUpdateController(opts updateControllerOptions) *updateController {
	pol := opts.Policy
	if pol.Interval <= 0 {
		pol = backend.DefaultUpdateCheckPolicy()
	}
	firstDelay := opts.FirstCheckDelay
	if firstDelay <= 0 {
		firstDelay = defaultFirstCheckDelay
	}
	retryBackoff := opts.RetryBackoff
	if retryBackoff <= 0 {
		retryBackoff = defaultRetryBackoff
	}
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	relURL := strings.TrimSpace(opts.ReleasesURL)
	if relURL == "" {
		relURL = releasesURL
	}
	c := &updateController{
		dataDir:         opts.DataDir,
		version:         opts.Version,
		isDev:           opts.IsDev,
		canSelfUpdate:   opts.CanSelfUpdate,
		releasesURL:     relURL,
		firstCheckDelay: firstDelay,
		retryBackoff:    retryBackoff,
		now:             nowFn,
		check:           opts.Check,
		relaunch:        opts.Relaunch,
		onChange:        opts.OnChange,
		policy:          pol,
	}
	c.loadDiskStateLocked()
	return c
}

func (c *updateController) statePath() string {
	if c.dataDir == "" {
		return ""
	}
	return filepath.Join(c.dataDir, updaterStateFilename)
}

func (c *updateController) loadDiskStateLocked() {
	path := c.statePath()
	if path == "" {
		return
	}
	if data, err := os.ReadFile(path); err == nil {
		var st updaterDiskState
		if json.Unmarshal(data, &st) == nil {
			c.disk = st
			return
		}
	}
	// Migrate last-check timestamp and skipped version from older builds that used plugins/updater.
	legacyPath := filepath.Join(c.dataDir, legacyUpdaterStateFilename)
	if data, err := os.ReadFile(legacyPath); err == nil {
		var legacy legacyUpdaterDiskState
		if json.Unmarshal(data, &legacy) == nil {
			c.disk = updaterDiskState{
				LastCheck:      legacy.LastCheck,
				SkippedVersion: legacy.SkippedVersion,
			}
		}
	}
}

func (c *updateController) persistDiskState() {
	c.diskMu.Lock()
	defer c.diskMu.Unlock()
	c.mu.Lock()
	st := c.disk
	hook := c.saveDiskHook
	c.mu.Unlock()
	if hook != nil {
		hook(st)
	}
	c.saveDiskState(st)
}

func (c *updateController) saveDiskState(st updaterDiskState) {
	path := c.statePath()
	if path == "" {
		return
	}
	data, err := json.Marshal(st)
	if err != nil {
		return
	}
	f, err := os.CreateTemp(c.dataDir, ".updater-state-*.tmp")
	if err != nil {
		return
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return
	}
	if err := f.Sync(); err != nil {
		return
	}
	if err := f.Close(); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		return
	}
	ok = true
}

// Snapshot returns the current update capability snapshot for the frontend.
func (c *updateController) Snapshot() frontend.UpdateCapability {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked()
}

func (c *updateController) snapshotLocked() frontend.UpdateCapability {
	snap := frontend.UpdateCapability{
		Version:          c.version,
		InstalledVersion: c.installedVersion,
		Status:           c.status,
		DownloadedBytes:  c.downloadedBytes,
		TotalBytes:       c.totalBytes,
		LastError:        c.lastError,
		LastCheck:        c.disk.LastCheck,
		IsDev:            c.isDev,
		CanSelfUpdate:    c.canSelfUpdate,
		ReleasesURL:      c.releasesURL,
	}
	if c.available != nil {
		snap.AvailableVersion = c.available.Version
		snap.AvailableNotes = c.available.Notes
	}
	return snap
}

func (c *updateController) emit() {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	c.mu.Lock()
	snap := c.snapshotLocked()
	fn := c.onChange
	closed := c.closed
	c.mu.Unlock()
	if !closed && fn != nil {
		fn(snap)
	}
}

// Start arms periodic silent checks when automatic checks are enabled and supported.
func (c *updateController) Start() {
	c.mu.Lock()
	if c.started || c.closed {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.scheduleLocked()
	c.mu.Unlock()
	c.emit()
}

// OnResume re-evaluates the schedule after macOS wakes from sleep.
func (c *updateController) OnResume() {
	c.mu.Lock()
	if c.started && !c.closed {
		c.scheduleLocked()
	}
	c.mu.Unlock()
}

// SetPolicy updates the silent check preference and reschedules background checks.
func (c *updateController) SetPolicy(pol backend.UpdateCheckPolicy) {
	if pol.Interval <= 0 {
		pol.Interval = backend.DefaultUpdateCheckPolicy().Interval
	}
	var cancelSilent context.CancelFunc
	c.mu.Lock()
	c.policy = pol
	if !pol.Enabled && c.checking && !c.checkingUser && c.checkCancel != nil {
		cancelSilent = c.checkCancel
		c.checking = false
		c.checkingUser = false
		c.checkCancel = nil
		c.checkGen++
	}
	if c.started && !c.closed {
		c.scheduleLocked()
	} else {
		c.stopTimerLocked()
	}
	c.mu.Unlock()
	if cancelSilent != nil {
		cancelSilent()
	}
}

func (c *updateController) stopTimerLocked() {
	c.timerGen++
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
}

func (c *updateController) scheduleLocked() {
	c.stopTimerLocked()
	if c.closed || !c.started || !c.policy.Enabled || !c.canSelfUpdate || c.isDev || c.check == nil {
		return
	}
	if c.checking || c.installing || c.installedVersion != "" {
		return
	}
	wait := c.firstCheckDelay
	if c.launchChecked {
		now := c.now()
		due := nextUpdateCheck(c.disk.LastCheck, c.failedAt, c.policy.Interval, c.retryBackoff, now)
		wait = max(due.Sub(now), c.firstCheckDelay)
	}
	gen := c.timerGen
	c.timer = time.AfterFunc(wait, func() {
		c.beginCheck(false, gen)
	})
}

func nextUpdateCheck(last, failed time.Time, interval, retryBackoff time.Duration, now time.Time) time.Time {
	if last.After(now) {
		last = now
	}
	due := last.Add(interval)
	if !failed.IsZero() {
		if failed.After(now) {
			failed = now
		}
		if retry := failed.Add(min(interval, retryBackoff)); retry.After(due) {
			due = retry
		}
	}
	return due
}

// CheckForUpdates starts a user-initiated check or promotes an in-flight silent check.
func (c *updateController) CheckForUpdates() {
	c.beginCheck(true, 0)
}

func (c *updateController) beginCheck(user bool, scheduledGen uint64) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	if !user {
		if !c.policy.Enabled {
			c.mu.Unlock()
			return
		}
		if scheduledGen != 0 && scheduledGen != c.timerGen {
			c.mu.Unlock()
			return
		}
	}
	if !c.canSelfUpdate || c.isDev || c.check == nil {
		c.mu.Unlock()
		c.emit()
		return
	}
	if c.installedVersion != "" {
		c.status = frontend.UpdateReady
		c.mu.Unlock()
		c.emit()
		return
	}
	if c.installing {
		c.mu.Unlock()
		if user {
			c.emit()
		}
		return
	}
	if c.checking {
		if user && !c.checkingUser {
			c.checkingUser = true
			c.status = frontend.UpdateChecking
			c.lastError = ""
			c.mu.Unlock()
			c.emit()
			return
		}
		c.mu.Unlock()
		return
	}
	c.stopTimerLocked()
	ctx, cancel := context.WithCancel(context.Background())
	c.checkGen++
	myGen := c.checkGen
	c.checking = true
	c.checkingUser = user
	c.launchChecked = true
	c.checkCancel = cancel
	if user {
		c.status = frontend.UpdateChecking
		c.lastError = ""
	}
	c.wg.Add(1)
	c.mu.Unlock()

	if user {
		c.emit()
	}

	go func() {
		defer c.wg.Done()
		defer cancel()
		rel, err := c.check(ctx)

		shouldPersist := false
		shouldEmit := false

		c.mu.Lock()
		// Ignore stale results if another operation superseded this check (e.g. InstallUpdate, SetPolicy(off), Close).
		if myGen != c.checkGen || c.installing || c.installedVersion != "" {
			if myGen == c.checkGen {
				c.checking = false
				c.checkingUser = false
				c.checkCancel = nil
			}
			c.mu.Unlock()
			return
		}

		wasUser := c.checkingUser
		c.checking = false
		c.checkingUser = false
		c.checkCancel = nil

		if ctx.Err() != nil || c.closed {
			if c.available != nil {
				c.status = frontend.UpdateAvailable
			} else if c.status == frontend.UpdateChecking {
				c.status = frontend.UpdateIdle
			}
			c.scheduleLocked()
			c.mu.Unlock()
			c.emit()
			return
		}

		if err != nil {
			c.failedAt = c.now()
			if wasUser {
				c.status = frontend.UpdateError
				c.lastError = err.Error()
				shouldEmit = true
			} else if c.available != nil {
				c.status = frontend.UpdateAvailable
			}
			c.scheduleLocked()
			c.mu.Unlock()
			if shouldEmit {
				c.emit()
			}
			return
		}

		c.failedAt = time.Time{}
		c.disk.LastCheck = c.now().UTC()
		shouldPersist = true
		shouldEmit = true

		switch {
		case rel == nil:
			c.available = nil
			c.lastError = ""
			if wasUser {
				c.status = frontend.UpdateUpToDate
			} else {
				c.status = frontend.UpdateIdle
			}
		case !wasUser && c.disk.SkippedVersion != "" && rel.Version == c.disk.SkippedVersion:
			c.available = nil
			c.lastError = ""
			c.status = frontend.UpdateIdle
		default:
			c.available = rel
			c.lastError = ""
			c.status = frontend.UpdateAvailable
		}
		c.scheduleLocked()
		c.mu.Unlock()

		if shouldPersist {
			c.persistDiskState()
		}
		if shouldEmit {
			c.emit()
		}
	}()
}

// InstallUpdate downloads and installs the discovered update in the background.
func (c *updateController) InstallUpdate() {
	var cancelCheck context.CancelFunc
	c.mu.Lock()
	if c.closed || c.installing || c.installedVersion != "" || c.available == nil || c.available.Install == nil {
		c.mu.Unlock()
		return
	}
	c.stopTimerLocked()
	if c.checking || c.checkCancel != nil {
		cancelCheck = c.checkCancel
		c.checking = false
		c.checkingUser = false
		c.checkCancel = nil
		c.checkGen++
	}
	rel := c.available
	ctx, cancel := context.WithCancel(context.Background())
	c.installing = true
	c.installCancel = cancel
	c.status = frontend.UpdateDownloading
	c.downloadedBytes = 0
	c.totalBytes = 0
	c.lastError = ""
	c.wg.Add(1)
	c.mu.Unlock()

	if cancelCheck != nil {
		cancelCheck()
	}
	c.emit()

	go func() {
		defer c.wg.Done()
		defer cancel()

		var lastNotifiedBytes int64
		var lastNotifiedPct int = -1
		var lastStatus frontend.UpdateStatus = frontend.UpdateDownloading

		err := rel.Install(ctx, func(downloaded, total int64) {
			nextStatus := frontend.UpdateDownloading
			if total > 0 && downloaded >= total {
				nextStatus = frontend.UpdateInstalling
			}
			shouldEmit := nextStatus != lastStatus
			if !shouldEmit && total > 0 {
				pct := int(downloaded * 100 / total)
				if pct != lastNotifiedPct {
					shouldEmit = true
					lastNotifiedPct = pct
				}
			} else if !shouldEmit && downloaded-lastNotifiedBytes >= progressNotifyMinBytes {
				shouldEmit = true
			}
			if !shouldEmit {
				return
			}
			lastNotifiedBytes = downloaded
			lastStatus = nextStatus

			c.mu.Lock()
			if c.installing && !c.closed {
				c.downloadedBytes = downloaded
				c.totalBytes = total
				c.status = nextStatus
			}
			c.mu.Unlock()
			c.emit()
		})

		c.mu.Lock()
		c.installing = false
		c.installCancel = nil
		switch {
		case err == nil:
			// Trust the actual install result even if CancelUpdate raced with the final chunk:
			// once MyGo staged the update on disk, the app must present Ready to Relaunch.
			c.installedVersion = rel.Version
			c.status = frontend.UpdateReady
			c.lastError = ""
		case ctx.Err() != nil || c.closed:
			c.downloadedBytes = 0
			c.totalBytes = 0
			if c.available != nil {
				c.status = frontend.UpdateAvailable
			} else {
				c.status = frontend.UpdateIdle
			}
			c.scheduleLocked()
		default:
			c.downloadedBytes = 0
			c.totalBytes = 0
			c.status = frontend.UpdateError
			c.lastError = err.Error()
			c.scheduleLocked()
		}
		c.mu.Unlock()
		c.emit()
	}()
}

// CancelUpdate cancels an active update check or download (ignored once final installation swap begins).
func (c *updateController) CancelUpdate() {
	c.mu.Lock()
	var cancel context.CancelFunc
	switch {
	case c.installing:
		if c.status == frontend.UpdateInstalling {
			c.mu.Unlock()
			return
		}
		cancel = c.installCancel
	case c.checking:
		cancel = c.checkCancel
	}
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// SkipVersion hides notifications for version until a newer release is published or the user checks manually.
func (c *updateController) SkipVersion(version string) {
	version = strings.TrimSpace(version)
	if version == "" {
		return
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.disk.SkippedVersion = version
	if c.available != nil && c.available.Version == version && !c.installing && c.installedVersion == "" {
		c.available = nil
		c.status = frontend.UpdateIdle
		c.lastError = ""
	}
	c.scheduleLocked()
	c.mu.Unlock()

	c.persistDiskState()
	c.emit()
}

// Relaunch triggers an application relaunch when an update has been installed.
func (c *updateController) Relaunch() {
	c.mu.Lock()
	ready := c.installedVersion != "" && !c.closed
	fn := c.relaunch
	c.mu.Unlock()
	if ready && fn != nil {
		fn()
	}
}

// Close stops scheduled checks, cancels in-flight update work, and waits for workers to exit.
func (c *updateController) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.stopTimerLocked()
	cancelCheck := c.checkCancel
	cancelInstall := c.installCancel
	c.mu.Unlock()
	if cancelCheck != nil {
		cancelCheck()
	}
	if cancelInstall != nil {
		cancelInstall()
	}
	c.wg.Wait()
}
