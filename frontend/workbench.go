package frontend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"aion/backend"
	"aion/backend/backup"
	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
)

// Theme values persisted in app.Settings.
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// NavPage identifies the active main view in the workbench.
type NavPage string

const (
	NavAllProjects      NavPage = "all_projects"
	NavProjectBoard     NavPage = "project_board"
	NavArchivedProjects NavPage = "archived_projects"
	NavJobAll           NavPage = "job_all"
	NavJobActive        NavPage = "job_active"
	NavJobOffer         NavPage = "job_offer"
	NavJobClosed        NavPage = "job_closed"
	NavTrash            NavPage = "trash"
	NavSettings         NavPage = "settings"
)

func isProjectNavPage(p NavPage) bool {
	return p == NavAllProjects || p == NavProjectBoard || p == NavArchivedProjects
}

func isJobNavPage(p NavPage) bool {
	return p == NavJobAll || p == NavJobActive || p == NavJobOffer || p == NavJobClosed
}

const defaultReleasesURL = "https://github.com/Games55k/Aion/releases"

// UpdateCapability describes the application's version and self-update availability.
type UpdateCapability struct {
	Version          string
	AvailableVersion string
	LastCheck        time.Time
	IsDev            bool
	CanSelfUpdate    bool
	ReleasesURL      string
}

// Host provides desktop OS integrations (file dialogs, Finder, default app, theme sync, updates)
// so that the UI works natively in a desktop window and deterministically in headless tests.
type Host struct {
	PickOpenFiles   func(title string, multiple bool, extensions ...string) ([]string, error)
	PickDirectory   func(title, defaultPath string) (string, error)
	PickSaveFile    func(title, defaultName string, extensions ...string) (string, error)
	OpenPath        func(path string) error
	RevealPath      func(path string) error
	OpenURL         func(url string) error
	SetTheme        func(theme string)
	UpdateInfo      UpdateCapability
	CheckForUpdates func()
	// Now supplies the clock for save timestamps and default export names. Defaults to time.Now.
	Now func() time.Time
	// Dispatch, when set (e.g. win.Update in a desktop window), applies worker results
	// on the UI thread. File operations, task saves, and navigation reads run in workers.
	// When nil (e.g. in headless ui.Tester tests), these operations run synchronously.
	Dispatch func(fn func())
}

// ProjectSummary holds task statistics for a project in the overview and sidebar.
type ProjectSummary struct {
	Total      int
	Planned    int
	InProgress int
	Done       int
}

// DragTaskPayload is carried by task cards during pointer drag-and-drop on the Kanban board.
type DragTaskPayload struct {
	TaskID     string
	FromStatus model.Status
}

// IsImageAttachment reports whether att has an image media type.
func IsImageAttachment(att model.Attachment) bool {
	return strings.HasPrefix(strings.ToLower(att.MediaType), "image/")
}

// Workbench holds all state for the Aion frontend.
type Workbench struct {
	app  *backend.App
	host Host

	mu            sync.Mutex
	pendingToasts []ToastProps
	lastAsyncErr  string
	busy          bool
	busyLabel     string
	busyCancel    context.CancelFunc
	restoring     bool
	dataEpoch     uint64
	taskWG        sync.WaitGroup

	// Layout state
	sideWidth      float32
	sideExpanded   bool
	detailExpanded bool

	// Navigation & Module state
	modules         []backend.ModuleDescriptor
	activeModule    string
	navPage         NavPage
	lastProjectPage NavPage
	lastJobPage     NavPage
	activeProjectID string

	// Job Applications module state (filter/sort/page preserved across module switches)
	jobApps              []model.JobApplication
	filteredJobApps      []model.JobApplication
	jobStats             model.JobApplicationStats
	jobSearchQuery       string
	jobStageFilter       string
	jobOutcomeFilter     string
	jobAppliedDateFilter time.Time
	jobSortBy            jobapplications.JobSort
	jobPage              int
	jobPageSize          int
	jobTableScroll       ui.ScrollState

	// Job Create modal state
	jobCreateOpen        bool
	jobCreateCompany     string
	jobCreatePosition    string
	jobCreateAppliedDate time.Time
	jobCreateStage       string
	jobCreateURL         string
	jobCreateChannel     string
	jobCreateNotes       string
	jobCreateErr         string
	jobCreateVersion     uint64
	jobCreatePending     bool

	// Job Detail modal & auto-save state
	selectedJobID        string
	detailJob            *model.JobApplication
	detailJobEpoch       uint64
	detailJobEvents      []model.JobApplicationEvent
	jobDetailOpen        bool
	editJobCompany       string
	editJobPosition      string
	editJobAppliedDate   time.Time
	editJobURL           string
	editJobChannel       string
	editJobNotes         string
	editJobClosureReason string
	editJobReasonDirty   bool
	jobSaveState         string // "saved", "saving", "validation_error", "error"
	jobSaveErr           string
	jobLastSavedAt       time.Time
	jobDirty             bool
	jobSaveRevision      uint64
	jobSaving            bool
	inFlightJobSave      *jobSave
	pendingJobSave       *jobSave
	discardJobAfterSave  bool
	// discardJobRebase marks that the in-flight write is a discard baseline: if it
	// fails, the queued follow-up save must rebase on the committed application.
	discardJobRebase bool
	jobWrites        []func()

	// Job Timeline input & retry state
	jobNoteInput           string
	jobNoteDate            time.Time
	jobNoteErr             string
	jobNoteRequest         jobapplications.AddNoteInput
	jobBackfillStage       string
	jobBackfillDate        time.Time
	jobBackfillNote        string
	jobBackfillErr         string
	jobBackfillRequest     jobapplications.BackfillStageInput
	jobTimelineDesc        bool
	jobBackfillExpanded    bool
	editingJobEventID      string
	editingJobEventNote    string
	pendingStageAppID      string
	pendingStageTarget     model.JobStage
	pendingStageOpID       string
	pendingQuickCloseAppID string
	pendingQuickCloseOpID  string

	// Job Close workflow modal state
	jobCloseOpen         bool
	jobCloseAppID        string
	jobCloseOutcome      string
	jobCloseReason       string
	jobCloseConfirmOffer bool
	jobCloseErr          string
	jobCloseOpID         string

	// Projects & Tasks state
	activeProjects   []model.Project
	archivedProjects []model.Project
	projectSummaries map[string]ProjectSummary
	projectNames     map[string]string
	boardTasks       []model.Task // Unfiltered tasks of activeProjectID
	filteredTasks    []model.Task // Filtered tasks of activeProjectID
	taskAttachments  map[string][]model.Attachment

	// Board filter state
	searchQuery    string
	priorityFilter string // "" or "none", "low", "medium", "high", "urgent"
	tagFilters     []string
	allBoardTags   []string

	// Quick inline task creation per column
	quickTaskTitles map[model.Status]string

	// Selected task detail & auto-save state
	selectedTaskID      string
	detailTask          *model.Task
	detailTaskEpoch     uint64
	detailAttachments   []model.Attachment
	editTitle           string
	editStatus          string
	editPriority        string
	editDueDate         time.Time
	editTags            []string
	newTagInput         string
	editDescription     string
	mdMode              int // 0: 编辑, 1: 预览, 2: 分栏
	editSubtasks        []model.Subtask
	newSubtaskTitle     string
	editingSubtaskID    string
	editingSubtaskTitle string
	saveState           string // "saved", "saving", "validation_error", "error"
	saveErr             string
	lastSavedAt         time.Time
	dirty               bool
	// UI-thread owned generations and one coalesced autosave behind the active write.
	saveRevision     uint64
	saving           bool
	inFlightSave     *taskSave
	pendingSave      *taskSave
	afterSave        func()
	discardAfterSave bool
	// discardRebase marks that the in-flight write is a discard baseline: if it fails,
	// the queued follow-up save must rebase on the committed task instead.
	discardRebase      bool
	loadVersions       map[string]uint64
	navigationVersion  uint64
	boardNeedsReload   bool
	recordWritePending bool
	afterRecordWrite   func()
	deletingTaskID     string

	// Preview caches are owned by the UI thread; workers publish through Host.Dispatch.
	attachmentPaths   map[string]string
	attachmentMissing map[string]bool
	bitmapCache       map[string]*ui.Bitmap
	bitmapFailed      map[string]bool
	bitmapOrder       []string
	bitmapLoading     bool
	bitmapLoader      func(string) (*ui.Bitmap, error)

	// Create/Edit Project modal state
	projectModalOpen    bool
	projectModalEditID  string
	projectModalName    string
	projectModalDesc    string
	projectModalIcon    string
	projectModalErr     string
	projectModalVersion uint64

	// Create Task modal state
	taskModalOpen     bool
	taskModalStatus   string
	taskModalPriority string
	taskModalTitle    string
	taskModalDesc     string
	taskModalTags     string
	taskModalDueDate  time.Time
	taskModalErr      string
	taskModalVersion  uint64

	// Trash view state
	trashSearch             string
	trashKindFilter         string // "", "project", "task", "job_application"
	trashItems              []model.TrashItem
	trashParentErrModalOpen bool
	trashOrphanTask         *model.TrashItem
	trashOrphanParent       *model.TrashItem
	confirmPermDeleteOpen   bool
	confirmPermDeleteItem   *model.TrashItem
	confirmClearTrashOpen   bool

	// Global Settings & Backup view state
	themeSetting         string
	themeSegmentIdx      int
	backupEnabled        bool
	backupIntervalSelect string
	backupKeepSelect     string
	backupDirInput       string
	backupStatus         backup.AutoStatus
	lastExportedZipPath  string
	lastRestoreResult    *backup.RestoreResult
	lastRestoreError     string
	confirmRestoreOpen   bool
	pendingRestoreZip    string
	updateInfo           UpdateCapability
	settingWrites        map[string]func() func()
}

// New creates and initializes a Workbench backed by app.
func New(app *backend.App, host Host) *Workbench {
	w := &Workbench{
		app:               app,
		host:              host,
		sideWidth:         240,
		sideExpanded:      true,
		detailExpanded:    false,
		activeModule:      "projects",
		navPage:           NavAllProjects,
		lastProjectPage:   NavAllProjects,
		lastJobPage:       NavJobAll,
		projectSummaries:  make(map[string]ProjectSummary),
		projectNames:      make(map[string]string),
		taskAttachments:   make(map[string][]model.Attachment),
		quickTaskTitles:   make(map[model.Status]string),
		attachmentPaths:   make(map[string]string),
		attachmentMissing: make(map[string]bool),
		bitmapCache:       make(map[string]*ui.Bitmap),
		bitmapFailed:      make(map[string]bool),
		saveState:         "saved",
		jobSaveState:      "saved",
		jobSortBy:         jobapplications.SortAppliedDesc,
		jobPageSize:       20,
		jobBackfillStage:  string(model.JobStageInterview1),
		jobTimelineDesc:   true,
		themeSetting:      ThemeSystem,
		updateInfo:        host.UpdateInfo,
	}
	w.host.Dispatch = nil
	w.ReloadAll()
	if len(w.activeProjects) > 0 {
		w.SelectProject(w.activeProjects[0].ID)
	}
	w.host = host
	return w
}

// SetHost updates the OS integration callbacks.
func (w *Workbench) SetHost(host Host) {
	w.mu.Lock()
	w.host = host
	w.updateInfo = host.UpdateInfo
	w.mu.Unlock()
}

// UpdateCapability returns a snapshot of the current app version and update availability without acquiring Store locks.
func (w *Workbench) UpdateCapability() UpdateCapability {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.updateInfo
}

// SetUpdateCapability updates the cached app version and update availability without acquiring Store locks.
func (w *Workbench) SetUpdateCapability(info UpdateCapability) {
	w.mu.Lock()
	w.updateInfo = info
	w.mu.Unlock()
}

// ReleasesURL returns the GitHub Releases URL configured for Aion.
func (w *Workbench) ReleasesURL() string {
	info := w.UpdateCapability()
	if u := strings.TrimSpace(info.ReleasesURL); u != "" {
		return u
	}
	return defaultReleasesURL
}

// CheckForUpdates delegates an update check to the configured Host without acquiring Store locks.
func (w *Workbench) CheckForUpdates() {
	w.mu.Lock()
	check := w.host.CheckForUpdates
	w.mu.Unlock()
	if check == nil {
		return
	}
	check()
}

// OpenReleasesPage opens the GitHub Releases page via Host.OpenURL.
func (w *Workbench) OpenReleasesPage() {
	w.mu.Lock()
	openURL := w.host.OpenURL
	w.mu.Unlock()
	if openURL == nil {
		return
	}
	if err := openURL(w.ReleasesURL()); err != nil {
		w.enqueueErrorToast("打开发布页失败", err)
	}
}

func (w *Workbench) updateActionsReady() (canCheck, canOpenURL bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.host.CheckForUpdates != nil, w.host.OpenURL != nil
}

func (w *Workbench) now() time.Time {
	if w.host.Now != nil {
		return w.host.Now()
	}
	return time.Now()
}

func (w *Workbench) runDialog(pick func() func()) {
	w.runAsyncTask("正在打开文件选择器...", func(_ context.Context) func() {
		return pick()
	})
}

// IsBusy reports whether a background file or backup task is currently running.
func (w *Workbench) IsBusy() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.busy
}

// BusyLabel returns the human-readable description of the active background task.
func (w *Workbench) BusyLabel() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.busyLabel
}

// CancelBusyTask cancels the currently running background file or backup task, if any.
func (w *Workbench) CancelBusyTask() {
	w.mu.Lock()
	cancel := w.busyCancel
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// WaitForTasks waits for all in-flight background tasks to complete.
func (w *Workbench) WaitForTasks() {
	w.taskWG.Wait()
}

// IsRestoring reports whether a full backup restore is currently in progress or awaiting UI completion.
// While true, all business writes and navigation saves are blocked so pre-restore UI state cannot overwrite restored data.
func (w *Workbench) IsRestoring() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.restoring
}

func (w *Workbench) setRestoring(v bool) {
	w.mu.Lock()
	w.restoring = v
	w.mu.Unlock()
}

func (w *Workbench) currentDataEpoch() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dataEpoch
}

func (w *Workbench) bumpDataEpoch() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.dataEpoch++
	return w.dataEpoch
}

func busyToast() ToastProps {
	return ToastProps{
		Title:       "后台任务进行中",
		Description: "请等待当前操作完成后再试，或先取消当前操作",
		Type:        ToastWarning,
	}
}

func (w *Workbench) enqueueBusyToast() {
	w.EnqueueToast(busyToast())
}

func (w *Workbench) enqueueErrorToast(title string, err error) {
	if err == nil {
		return
	}
	w.EnqueueToast(ToastProps{
		Title:       title,
		Description: err.Error(),
		Type:        ToastError,
	})
}

func (w *Workbench) flushErrorDetail(err error) string {
	if w.jobSelected() && strings.TrimSpace(w.jobSaveErr) != "" {
		return w.jobSaveErr
	}
	if strings.TrimSpace(w.saveErr) != "" {
		return w.saveErr
	}
	if err != nil {
		return err.Error()
	}
	return "存在未保存修改"
}

func (w *Workbench) flushPendingOrToast(title, suffix string) error {
	err := w.FlushPending()
	if err == nil {
		return nil
	}
	if w.jobSelected() {
		w.jobDetailOpen = true
		if title == "当前任务存在未保存修改" {
			title = "当前投递存在未保存修改"
		}
	} else {
		w.detailExpanded = true
	}
	w.EnqueueToast(ToastProps{
		Title:       title,
		Description: w.flushErrorDetail(err) + suffix,
		Type:        ToastWarning,
	})
	return err
}

// runAsyncTask executes bg off the UI thread when Host.Dispatch is configured,
// and dispatches the returned UI-state update closure back onto the UI thread.
// When Host.Dispatch is nil (headless synchronous tests), it executes synchronously.
// Duplicate triggers while another task is busy are rejected.
func (w *Workbench) runAsyncTask(label string, bg func(ctx context.Context) func()) bool {
	if w.host.Dispatch == nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if apply := bg(ctx); apply != nil {
			apply()
		}
		return true
	}

	w.mu.Lock()
	if w.busy {
		w.pendingToasts = append(w.pendingToasts, busyToast())
		w.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.busy = true
	w.busyLabel = label
	w.busyCancel = cancel
	w.taskWG.Add(1)
	dispatch := w.host.Dispatch
	w.mu.Unlock()

	go func() {
		defer w.taskWG.Done()
		apply := bg(ctx)
		dispatch(func() {
			w.mu.Lock()
			w.busy = false
			w.busyLabel = ""
			w.busyCancel = nil
			w.mu.Unlock()
			cancel()
			if apply != nil {
				apply()
			}
		})
	}()
	return true
}

// EnqueueToast queues a toast notification to be displayed on the next frame.
func (w *Workbench) EnqueueToast(t ToastProps) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pendingToasts = append(w.pendingToasts, t)
}

// RecordAsyncError records an asynchronous background error (e.g., scheduled backup failure)
// and queues an error toast notification.
func (w *Workbench) RecordAsyncError(err error) {
	if err == nil {
		return
	}
	w.mu.Lock()
	w.lastAsyncErr = err.Error()
	w.pendingToasts = append(w.pendingToasts, ToastProps{
		Title:       "后台任务异常",
		Description: err.Error(),
		Type:        ToastError,
	})
	w.mu.Unlock()
}

// LastAsyncError returns the most recent asynchronous background error message.
func (w *Workbench) LastAsyncError() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastAsyncErr
}

// hasUnsavedChangesFromDetailTask reports whether any editable detail field differs from detailTask.
func (w *Workbench) hasUnsavedChangesFromDetailTask() bool {
	if w.selectedTaskID == "" || w.detailTask == nil {
		return false
	}
	t := w.detailTask
	dueStr := ""
	if !w.editDueDate.IsZero() {
		dueStr = w.editDueDate.Format("2006-01-02")
	}
	// Compare the persisted title while preserving whitespace in the active input.
	if strings.TrimSpace(w.editTitle) != t.Title ||
		w.editDescription != t.Description ||
		w.editStatus != string(t.Status) ||
		w.editPriority != string(t.Priority) ||
		dueStr != strings.TrimSpace(t.DueDate) {
		return true
	}
	if !slices.Equal(w.editTags, t.Tags) || !slices.Equal(w.editSubtasks, t.Subtasks) {
		return true
	}
	return false
}

// FlushPending requests persistence of any unsaved draft. With Host.Dispatch it
// returns errSavePending/errJobSavePending until the worker result has been applied on the UI thread.
func (w *Workbench) FlushPending() error {
	if w.IsRestoring() ||
		(w.detailTask != nil && w.detailTaskEpoch != w.currentDataEpoch()) ||
		(w.detailJob != nil && w.detailJobEpoch != w.currentDataEpoch()) {
		return fmt.Errorf("%w: backup restore in progress", model.ErrConflict)
	}
	if w.selectedTaskID != "" && w.detailTask != nil {
		if w.saving {
			return errSavePending
		}
		if w.dirty || w.hasUnsavedChangesFromDetailTask() {
			if err := w.SaveSelectedTask(); err != nil {
				return err
			}
			if w.saving {
				return errSavePending
			}
		}
	}
	if w.selectedJobID != "" && w.detailJob != nil {
		if w.jobSaving {
			return errJobSavePending
		}
		if w.jobDirty || w.hasUnsavedChangesFromDetailJob() {
			if err := w.SaveSelectedJob(); err != nil {
				return err
			}
			if w.jobSaving {
				return errJobSavePending
			}
		}
	}
	return nil
}

// PrepareClose verifies whether the window can safely close.
// If there are unsaved task or job edits that fail validation/persistence, or a background task is running,
// PrepareClose keeps the draft/modal visible, notifies the user, and returns false.
func (w *Workbench) PrepareClose() bool {
	if w.recordWritePending || len(w.settingWrites) > 0 {
		w.EnqueueToast(ToastProps{Title: "正在保存更改", Description: "保存完成后再关闭窗口", Type: ToastInfo})
		return false
	}
	if w.saving {
		w.EnqueueToast(ToastProps{Title: "正在保存任务", Description: "保存完成后再关闭窗口", Type: ToastInfo})
		return false
	}
	if w.jobSaving || len(w.jobWrites) > 0 {
		w.EnqueueToast(ToastProps{Title: "正在保存投递记录", Description: "保存完成后再关闭窗口", Type: ToastInfo})
		return false
	}
	if w.IsBusy() {
		w.EnqueueToast(ToastProps{
			Title:       "后台任务正在执行",
			Description: "请等待当前备份或附件操作完成后再关闭窗口，或先取消操作",
			Type:        ToastWarning,
		})
		return false
	}
	if err := w.FlushPending(); err != nil {
		if w.jobSelected() {
			w.activeModule = "job_applications"
			if !isJobNavPage(w.navPage) {
				if isJobNavPage(w.lastJobPage) {
					w.navPage = w.lastJobPage
				} else {
					w.navPage = NavJobAll
				}
			}
			w.jobDetailOpen = true
			w.EnqueueToast(ToastProps{
				Title:       "存在未保存的投递修改",
				Description: w.flushErrorDetail(err) + "，请先修正保存或点击“放弃修改”后再退出应用",
				Type:        ToastWarning,
			})
			return false
		}
		w.activeModule = "projects"
		w.navPage = NavProjectBoard
		w.lastProjectPage = NavProjectBoard
		w.detailExpanded = true
		w.EnqueueToast(ToastProps{
			Title:       "存在未保存的任务修改",
			Description: w.flushErrorDetail(err) + "，请先修正保存或点击“放弃修改”后再退出应用",
			Type:        ToastWarning,
		})
		return false
	}
	return true
}

// ReloadAll reloads modules, projects, job applications, trash, settings, and backup status from the backend.
func (w *Workbench) ReloadAll() {
	w.modules = w.app.Modules.List()
	if len(w.modules) > 0 && w.activeModule == "" {
		w.activeModule = w.modules[0].ID
	}

	w.ReloadProjects()
	w.ReloadJobApplications()
	w.ReloadTrash()
	w.ReloadSettings()
}

func (w *Workbench) rebuildProjectNames() {
	names := make(map[string]string, len(w.activeProjects)+len(w.archivedProjects)+len(w.trashItems))
	for _, p := range w.activeProjects {
		names[p.ID] = p.Name
	}
	for _, p := range w.archivedProjects {
		names[p.ID] = p.Name
	}
	for _, it := range w.trashItems {
		if it.Kind == projects.TrashProject {
			names[it.ID] = it.Name
		}
	}
	w.projectNames = names
}

// ReloadProjects reloads active and archived projects and their task summaries.
func (w *Workbench) ReloadProjects() { w.reloadProjects(false) }

func (w *Workbench) reloadProjects(async bool) {
	service := w.app.Projects
	w.loadUI("projects", async, func() func() {
		ctx := context.Background()
		all, err := service.ListProjects(ctx, projects.ProjectFilter{})
		var counts map[string]map[model.Status]int
		if err == nil {
			counts, err = service.CountTasks(ctx)
		}
		return func() {
			if err != nil {
				w.enqueueErrorToast("加载项目失败", err)
				return
			}
			var active, archived []model.Project
			summaries := make(map[string]ProjectSummary, len(all))
			for _, p := range all {
				if !p.Archived {
					active = append(active, p)
				} else {
					archived = append(archived, p)
				}
				c := counts[p.ID]
				sum := ProjectSummary{Planned: c[model.Planned], InProgress: c[model.InProgress], Done: c[model.Done]}
				sum.Total = sum.Planned + sum.InProgress + sum.Done
				summaries[p.ID] = sum
			}
			w.activeProjects = active
			w.archivedProjects = archived
			w.projectSummaries = summaries
			w.rebuildProjectNames()

			if w.activeProjectID != "" {
				found := false
				for _, p := range all {
					if p.ID == w.activeProjectID {
						found = true
						break
					}
				}
				if !found {
					w.activeProjectID = ""
					w.boardTasks = nil
					w.filteredTasks = nil
					w.allBoardTags = nil
					w.taskAttachments = make(map[string][]model.Attachment)
					w.resetSelectedTaskState()
					if w.navPage == NavProjectBoard {
						w.navPage = NavAllProjects
					}
				} else {
					w.reloadBoardTasks(async)
				}
			}
		}
	})
}

// ActiveProject returns the currently selected project (active or archived), or nil.
func (w *Workbench) ActiveProject() *model.Project {
	for i := range w.activeProjects {
		if w.activeProjects[i].ID == w.activeProjectID {
			return &w.activeProjects[i]
		}
	}
	for i := range w.archivedProjects {
		if w.archivedProjects[i].ID == w.activeProjectID {
			return &w.archivedProjects[i]
		}
	}
	return nil
}

// SelectProject switches the workbench to the given project's Kanban board.
func (w *Workbench) SelectProject(projectID string) {
	if w.IsRestoring() {
		return
	}
	if !w.afterPendingSave(func() { w.SelectProject(projectID) }, "当前任务存在未保存修改", "，请先修正保存或放弃修改后再切换项目") {
		return
	}
	if w.activeProjectID != projectID {
		w.boardNeedsReload = false
		w.boardTasks = nil
		w.filteredTasks = nil
		w.allBoardTags = nil
		w.taskAttachments = make(map[string][]model.Attachment)
		w.activeProjectID = projectID
		w.searchQuery = ""
		w.priorityFilter = ""
		w.tagFilters = nil
		w.resetSelectedTaskState()
	}
	w.activeModule = "projects"
	w.lastProjectPage = NavProjectBoard
	w.navigationVersion++
	w.navPage = NavProjectBoard
	w.reloadBoardTasks(true)
}

// SwitchModule switches between registered modules while protecting drafts and preserving per-module navigation/filter state.
func (w *Workbench) SwitchModule(modID string) {
	if w.IsRestoring() || modID == "" {
		return
	}
	if isProjectNavPage(w.navPage) {
		w.lastProjectPage = w.navPage
	} else if isJobNavPage(w.navPage) {
		w.lastJobPage = w.navPage
	}
	if w.activeModule == modID && !w.IsGlobalPage() {
		return
	}
	if !w.afterPendingSave(func() { w.SwitchModule(modID) }, "存在未保存修改", "，请先修正保存或放弃修改后再切换模块") {
		return
	}
	w.activeModule = modID
	w.navigationVersion++
	switch modID {
	case "job_applications":
		target := w.lastJobPage
		if !isJobNavPage(target) {
			target = NavJobAll
		}
		w.navPage = target
		w.rebuildJobDerived()
		if !w.IsBusy() {
			w.reloadJobApplications(true)
		}
	default:
		w.activeModule = "projects"
		target := w.lastProjectPage
		if !isProjectNavPage(target) || (target == NavProjectBoard && w.activeProjectID == "") {
			target = NavAllProjects
		}
		w.navPage = target
		if !w.IsBusy() {
			if target == NavProjectBoard {
				w.reloadBoardTasks(true)
			} else {
				w.reloadProjects(true)
			}
		}
	}
}

// IsGlobalPage reports whether the current navigation page is shared across modules (Trash or Settings).
func (w *Workbench) IsGlobalPage() bool {
	return w.navPage == NavTrash || w.navPage == NavSettings
}

// NavigateTo switches the top-level page after flushing any pending task or job edits.
func (w *Workbench) NavigateTo(page NavPage) {
	if w.IsRestoring() {
		return
	}
	if isProjectNavPage(w.navPage) {
		w.lastProjectPage = w.navPage
	} else if isJobNavPage(w.navPage) {
		w.lastJobPage = w.navPage
	}
	if !w.afterPendingSave(func() { w.NavigateTo(page) }, "当前任务存在未保存修改", "，请先修正保存或放弃修改后再离开当前页面") {
		return
	}
	w.navigationVersion++
	w.navPage = page
	if isProjectNavPage(page) {
		w.activeModule = "projects"
		w.lastProjectPage = page
	} else if isJobNavPage(page) {
		w.activeModule = "job_applications"
		w.lastJobPage = page
		w.rebuildJobDerived()
	}
	if w.IsBusy() {
		return
	}
	switch page {
	case NavAllProjects, NavArchivedProjects:
		w.reloadProjects(true)
	case NavJobAll, NavJobActive, NavJobOffer, NavJobClosed:
		w.reloadJobApplications(true)
	case NavTrash:
		w.reloadTrash(true)
	case NavSettings:
		w.reloadSettings(true)
	case NavProjectBoard:
		w.reloadBoardTasks(true)
	}
}

func (w *Workbench) detailModalOpen() bool {
	return w.navPage == NavProjectBoard && w.detailExpanded && w.selectedTaskID != "" && w.detailTask != nil
}

func (w *Workbench) jobDetailModalOpen() bool {
	return isJobNavPage(w.navPage) && w.jobDetailOpen && w.jobSelected()
}

func (w *Workbench) hasOpenModal() bool {
	return w.projectModalOpen ||
		w.taskModalOpen ||
		w.detailModalOpen() ||
		w.jobCreateOpen ||
		w.jobDetailModalOpen() ||
		w.jobCloseOpen ||
		w.trashParentErrModalOpen ||
		w.confirmPermDeleteOpen ||
		w.confirmClearTrashOpen ||
		w.confirmRestoreOpen
}

// OpenSettings navigates to the global Settings view unless a modal dialog is
// open or Settings is already active.
func (w *Workbench) OpenSettings() {
	if w.hasOpenModal() || w.navPage == NavSettings {
		return
	}
	w.NavigateTo(NavSettings)
}

// ReloadBoardTasks reloads the tasks for the active project and applies any active filters.
func (w *Workbench) ReloadBoardTasks() { w.reloadBoardTasks(false) }

func (w *Workbench) reloadBoardTasks(async bool) {
	if w.activeProjectID == "" {
		w.boardTasks = nil
		w.filteredTasks = nil
		w.allBoardTags = nil
		w.taskAttachments = make(map[string][]model.Attachment)
		w.pruneAttachmentCaches()
		return
	}

	w.loadUI("board", async, w.boardReader(w.activeProjectID))
}

func (w *Workbench) boardReader(projectID string) func() func() {
	service, attachments := w.app.Projects, w.app.Attachments
	return func() func() {
		ctx := context.Background()
		unfiltered, err := service.ListTasks(ctx, projectID, projects.TaskFilter{})
		var attMap map[string][]model.Attachment
		if err == nil {
			attMap, err = attachments.ListByProject(ctx, projectID)
		}
		return func() {
			if projectID != w.activeProjectID {
				return
			}
			if err != nil {
				w.enqueueErrorToast("加载任务失败", err)
				return
			}
			w.boardTasks = unfiltered
			w.boardNeedsReload = false
			for _, t := range unfiltered {
				if _, ok := attMap[t.ID]; !ok {
					attMap[t.ID] = nil
				}
			}
			w.taskAttachments = attMap
			w.rebuildBoardDerived()

			// Keep selected task in sync if still present
			if w.selectedTaskID != "" {
				idx := slices.IndexFunc(unfiltered, func(t model.Task) bool {
					return t.ID == w.selectedTaskID
				})
				if idx < 0 {
					w.resetSelectedTaskState()
				} else {
					t := unfiltered[idx]
					if !w.dirty && !w.hasUnsavedChangesFromDetailTask() {
						w.syncDetailFromTask(t)
					} else {
						w.detailTask = &t
						w.detailAttachments = slices.Clone(attMap[t.ID])
						w.cacheAttachments(w.detailAttachments)
					}
				}
			} else {
				w.pruneAttachmentCaches()
			}
		}
	}
}

// rebuildBoardDerived recomputes everything derived from boardTasks (project
// summary, tag list, pruned tag filters, visible tasks) without touching SQLite.
func (w *Workbench) rebuildBoardDerived() {
	var s ProjectSummary
	s.Total = len(w.boardTasks)
	tagSet := make(map[string]bool)
	for _, t := range w.boardTasks {
		switch t.Status {
		case model.Planned:
			s.Planned++
		case model.InProgress:
			s.InProgress++
		case model.Done:
			s.Done++
		}
		for _, tag := range t.Tags {
			tagSet[tag] = true
		}
	}
	w.projectSummaries[w.activeProjectID] = s

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	w.allBoardTags = tags

	// Prune any selected tag filters that no longer exist on the board
	w.tagFilters = slices.DeleteFunc(w.tagFilters, func(tf string) bool { return !tagSet[tf] })
	w.applyBoardFilters()
}

// applyBoardFilters recomputes the visible tasks from the loaded board in memory.
func (w *Workbench) applyBoardFilters() {
	if !w.HasBoardFilters() {
		w.filteredTasks = slices.Clone(w.boardTasks)
		return
	}
	filter := projects.TaskFilter{
		Query:    w.searchQuery,
		Tags:     w.tagFilters,
		Priority: model.Priority(w.priorityFilter),
	}
	filtered := make([]model.Task, 0, len(w.boardTasks))
	for _, t := range w.boardTasks {
		if filter.Matches(t) {
			filtered = append(filtered, t)
		}
	}
	w.filteredTasks = filtered
}

// HasBoardFilters reports whether any search, priority, or tag filter is active on the board.
func (w *Workbench) HasBoardFilters() bool {
	return strings.TrimSpace(w.searchQuery) != "" || w.priorityFilter != "" || len(w.tagFilters) > 0
}

// ClearBoardFilters resets all search, priority, and tag filters on the board.
func (w *Workbench) ClearBoardFilters() {
	if w.IsRestoring() {
		return
	}
	w.searchQuery = ""
	w.priorityFilter = ""
	w.tagFilters = nil
	w.rebuildBoardDerived()
}

// ToggleTagFilter toggles a tag in the multi-tag AND filter list.
func (w *Workbench) ToggleTagFilter(tag string) {
	if w.IsRestoring() {
		return
	}
	if idx := slices.Index(w.tagFilters, tag); idx >= 0 {
		w.tagFilters = slices.Delete(w.tagFilters, idx, idx+1)
	} else {
		w.tagFilters = append(w.tagFilters, tag)
	}
	w.rebuildBoardDerived()
}

// TasksInColumn returns the visible tasks for the given status column.
func (w *Workbench) TasksInColumn(status model.Status) []model.Task {
	var out []model.Task
	for _, t := range w.filteredTasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

// UnfilteredTasksInColumn returns all tasks in the active project for the given status column.
func (w *Workbench) UnfilteredTasksInColumn(status model.Status) []model.Task {
	var out []model.Task
	for _, t := range w.boardTasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

// SelectTask opens the task detail modal after flushing any pending edits.
func (w *Workbench) SelectTask(taskID string) {
	if w.IsRestoring() {
		return
	}
	if w.recordWritePending {
		navigation := w.navigationVersion
		w.loadVersions["selection"]++
		w.afterRecordWrite = func() {
			if w.navigationVersion == navigation && slices.ContainsFunc(w.boardTasks, func(t model.Task) bool { return t.ID == taskID }) {
				w.SelectTask(taskID)
			}
		}
		return
	}
	if w.selectedTaskID == taskID && w.detailTask != nil {
		w.loadVersions["selection"]++
		w.afterSave = nil
		w.detailExpanded = true
		return
	}
	if !w.afterPendingSave(func() { w.SelectTask(taskID) }, "当前任务存在未保存修改", "，请先修正保存或放弃修改后再切换任务") {
		return
	}
	projectsService, attachmentsService := w.app.Projects, w.app.Attachments
	w.loadUI("selection", true, func() func() {
		t, err := projectsService.GetTask(context.Background(), taskID)
		var atts []model.Attachment
		if err == nil {
			atts, err = attachmentsService.List(context.Background(), taskID)
		}
		return func() {
			if err != nil {
				w.enqueueErrorToast("无法打开任务详情", err)
				return
			}
			if w.recordWritePending {
				w.SelectTask(taskID)
				return
			}
			w.taskAttachments[t.ID] = atts
			w.selectLoadedTask(t)
		}
	})
}

func (w *Workbench) selectLoadedTask(t model.Task) {
	w.selectedTaskID = t.ID
	w.detailExpanded = true
	w.lastSavedAt = time.Time{}
	w.clearDetailTransientInputs()
	w.syncDetailFromTask(t)
}

func (w *Workbench) clearDetailTransientInputs() {
	w.editingSubtaskID = ""
	w.editingSubtaskTitle = ""
	w.newSubtaskTitle = ""
	w.newTagInput = ""
}

func (w *Workbench) resetSelectedTaskState() {
	w.navigationVersion++
	w.selectedTaskID = ""
	w.detailTask = nil
	w.detailTaskEpoch = w.currentDataEpoch()
	w.detailAttachments = nil
	w.detailExpanded = false
	w.editTitle = ""
	w.editStatus = ""
	w.editPriority = ""
	w.editDueDate = time.Time{}
	w.editTags = nil
	w.editDescription = ""
	w.editSubtasks = nil
	w.dirty = false
	w.saveState = "saved"
	w.saveErr = ""
	w.pendingSave = nil
	w.inFlightSave = nil
	w.discardAfterSave = false
	w.discardRebase = false
	w.lastSavedAt = time.Time{}
	w.clearDetailTransientInputs()
	w.attachmentPaths = make(map[string]string)
	w.attachmentMissing = make(map[string]bool)
	w.bitmapCache = make(map[string]*ui.Bitmap)
	w.bitmapFailed = make(map[string]bool)
	w.bitmapOrder = nil
}

// DiscardSelectedTaskDraft explicitly discards unsaved edits on the currently selected task
// and restores its fields from the last saved version. While a background write is in
// flight, the draft rebases on that write's result — it is already committing — and the
// form only previews the last committed task when no write matches the selection.
func (w *Workbench) DiscardSelectedTaskDraft() {
	if w.IsRestoring() {
		return
	}
	w.afterSave = nil
	w.pendingSave = nil
	w.saveRevision++
	if w.saving {
		w.clearDetailTransientInputs()
		if baseline, ok := w.inFlightSaveBaseline(); ok {
			w.resetDetailEditFields(baseline)
		} else if w.detailTask != nil {
			w.resetDetailEditFields(*w.detailTask)
		}
		// The previewed baseline is still being written; it must not read as saved
		// before the acknowledgement lands (and it may still roll back on failure).
		w.saveState = "saving"
		w.discardAfterSave = true
		w.discardRebase = true
		return
	}
	if w.detailTask != nil {
		w.clearDetailTransientInputs()
		w.syncDetailFromTask(*w.detailTask)
		w.EnqueueToast(ToastProps{
			Title:       "已放弃未保存修改",
			Description: "任务内容已恢复为上次保存的状态",
			Type:        ToastInfo,
		})
	} else {
		w.resetSelectedTaskState()
	}
}

// ClearSelectedTask closes the task detail modal after flushing any pending edits.
// If flushing fails, the modal remains open and the unsaved draft is preserved.
func (w *Workbench) ClearSelectedTask() {
	if w.IsRestoring() {
		return
	}
	if !w.afterPendingSave(func() { w.ClearSelectedTask() }, "无法关闭详情：存在未保存修改", "（请修正后重试保存，或点击“放弃修改”）") {
		return
	}
	w.resetSelectedTaskState()
}

func (w *Workbench) syncDetailFromTask(t model.Task) {
	w.detailTask = &t
	w.detailTaskEpoch = w.currentDataEpoch()
	w.detailAttachments = slices.Clone(w.taskAttachments[t.ID])
	w.cacheAttachments(w.detailAttachments)
	w.resetDetailEditFields(t)
}

// resetDetailEditFields copies a task's content into the draft inputs without
// replacing detailTask, so the last committed task stays the failure baseline.
func (w *Workbench) resetDetailEditFields(t model.Task) {
	w.editTitle = t.Title
	w.editStatus = string(t.Status)
	w.editPriority = string(t.Priority)
	w.editDueDate = parseTaskDueDate(t.DueDate)
	w.editTags = slices.Clone(t.Tags)
	w.editDescription = t.Description
	w.editSubtasks = slices.Clone(t.Subtasks)
	w.dirty = false
	w.saveState = "saved"
	w.saveErr = ""
}

// parseTaskDueDate parses a persisted YYYY-MM-DD due date for the editor.
func parseTaskDueDate(due string) time.Time {
	if strings.TrimSpace(due) == "" {
		return time.Time{}
	}
	if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(due), time.Local); err == nil {
		return parsed
	}
	return time.Time{}
}

// taskInputFromDraft snapshots the current draft inputs as a save input, without
// validating them.
func (w *Workbench) taskInputFromDraft() projects.TaskInput {
	dueStr := ""
	if !w.editDueDate.IsZero() {
		dueStr = w.editDueDate.Format("2006-01-02")
	}
	return projects.TaskInput{
		Title:       strings.TrimSpace(w.editTitle),
		Description: w.editDescription,
		Status:      model.Status(w.editStatus),
		Priority:    model.Priority(w.editPriority),
		Tags:        slices.Clone(w.editTags),
		Subtasks:    slices.Clone(w.editSubtasks),
		DueDate:     dueStr,
	}
}

// inFlightSaveBaseline predicts the selected task as it will read once the active
// background write lands. Discard must rebase on that result, because the write is
// already committing and cannot be unsaved.
func (w *Workbench) inFlightSaveBaseline() (model.Task, bool) {
	if w.detailTask == nil || w.inFlightSave == nil || w.inFlightSave.taskID != w.selectedTaskID {
		return model.Task{}, false
	}
	return overlayTaskInput(*w.detailTask, w.inFlightSave.input), true
}

// overlayTaskInput projects a save input onto a task's content fields.
func overlayTaskInput(t model.Task, in projects.TaskInput) model.Task {
	t.Title, t.Description, t.Status, t.Priority = in.Title, in.Description, in.Status, in.Priority
	t.Tags, t.Subtasks, t.DueDate = in.Tags, in.Subtasks, in.DueDate
	return t
}

// rebaseTagsOnCommitted merges post-discard tag additions and removals into the
// committed set. Comparing whole slices would resurrect failed tags whenever any
// later edit changed the slice.
func rebaseTagsOnCommitted(in, snap, committed []string) []string {
	out := slices.Clone(committed)
	for _, tag := range in {
		if !slices.Contains(snap, tag) && !slices.Contains(out, tag) {
			out = append(out, tag)
		}
	}
	for _, tag := range snap {
		if !slices.Contains(in, tag) {
			if i := slices.Index(out, tag); i >= 0 {
				out = slices.Delete(out, i, i+1)
			}
		}
	}
	return out
}

// rebaseSubtasksOnCommitted merges post-discard subtask edits into the committed list.
// Titles and done flags fall back to the committed values field by field unless the
// user changed them after the discard. Additions and edits to unsaved additions stay,
// failed deletions roll back, and post-discard order edits take precedence.
func rebaseSubtasksOnCommitted(in, snap, committed []model.Subtask) []model.Subtask {
	snapByID := make(map[string]model.Subtask, len(snap))
	for _, st := range snap {
		snapByID[st.ID] = st
	}
	committedByID := make(map[string]model.Subtask, len(committed))
	for _, st := range committed {
		committedByID[st.ID] = st
	}
	mergedByID := make(map[string]model.Subtask, len(in)+len(committed))
	var draftOrder []string
	for _, st := range in {
		s, wasInSnap := snapByID[st.ID]
		c, inCommitted := committedByID[st.ID]
		if wasInSnap && !inCommitted && st == s {
			// The failed write added this row and the user left it untouched.
			continue
		}
		merged := st
		if wasInSnap && inCommitted {
			if st.Title == s.Title {
				merged.Title = c.Title
			}
			if st.Done == s.Done {
				merged.Done = c.Done
			}
		}
		mergedByID[merged.ID] = merged
		draftOrder = append(draftOrder, merged.ID)
	}
	for _, st := range committed {
		// Deleted by the failed write (absent from snap) and not re-added: the
		// rollback restored it. Its committed position is a preference in the merge.
		if _, wasInSnap := snapByID[st.ID]; !wasInSnap {
			if _, ok := mergedByID[st.ID]; !ok {
				mergedByID[st.ID] = st
			}
		}
	}
	var snapIDs, committedIDs []string
	for _, st := range snap {
		// An edited row from a failed addition keeps its position as a new row.
		_, inCommitted := committedByID[st.ID]
		if _, ok := mergedByID[st.ID]; ok && inCommitted {
			snapIDs = append(snapIDs, st.ID)
		}
	}
	for _, st := range committed {
		if _, ok := mergedByID[st.ID]; ok {
			committedIDs = append(committedIDs, st.ID)
		}
	}
	out := make([]model.Subtask, 0, len(mergedByID))
	for _, id := range rebaseSubtaskOrder(draftOrder, snapIDs, committedIDs) {
		out = append(out, mergedByID[id])
	}
	return out
}

// rebaseSubtaskOrder preserves post-discard swaps and the order of additions relative to
// visible subtasks. It then keeps compatible pairs in committed order. Placement of
// restored subtasks relative to additions comes last.
func rebaseSubtaskOrder(inIDs, snapIDs, committedIDs []string) []string {
	if slices.Equal(inIDs, snapIDs) {
		return slices.Clone(committedIDs)
	}
	if slices.Equal(snapIDs, committedIDs) {
		// The failed write did not change the order or hide a subtask.
		return slices.Clone(inIDs)
	}

	items := slices.Clone(committedIDs)
	index := make(map[string]int, len(committedIDs)+len(inIDs))
	for i, id := range committedIDs {
		index[id] = i
	}
	for _, id := range inIDs {
		if _, ok := index[id]; !ok {
			index[id] = len(items)
			items = append(items, id)
		}
	}
	if len(inIDs) >= len(snapIDs) && slices.Equal(inIDs[:len(snapIDs)], snapIDs) &&
		slices.Equal(inIDs[len(snapIDs):], items[len(committedIDs):]) {
		// Only new trailing items changed, so committed order followed by those
		// additions already satisfies every constraint, including restored positions.
		return items
	}

	positions := make([]subtaskOrderPosition, len(items))
	for i, id := range snapIDs {
		positions[index[id]].snapshot = i + 1
	}
	draft := make([]int, len(inIDs))
	for i, id := range inIDs {
		item := index[id]
		draft[i] = item
		positions[item].draft = i + 1
	}
	if order := compatibleSubtaskOrder(items, positions); order != nil {
		return order
	}

	// Required edges all follow draft order, so they cannot form a cycle.
	predecessors := make([][]int, len(items))
	for j, to := range draft {
		for _, from := range draft[:j] {
			if positions[from].usesDraftOrder(positions[to]) {
				predecessors[to] = append(predecessors[to], from)
			}
		}
	}

	seen := make([]bool, len(items))
	var markPredecessors func(int)
	markPredecessors = func(i int) {
		if seen[i] {
			return
		}
		seen[i] = true
		for _, prev := range predecessors[i] {
			markPredecessors(prev)
		}
	}
	preferBefore := func(from, start, end int) {
		clear(seen)
		markPredecessors(from)
		// Edges leaving from cannot change which items already precede it.
		for to := start; to < end; to++ {
			if !seen[to] && !slices.Contains(predecessors[to], from) {
				predecessors[to] = append(predecessors[to], from)
			}
		}
	}
	for i := range committedIDs {
		preferBefore(i, i+1, len(committedIDs))
	}
	if len(items) > len(committedIDs) {
		for i := range committedIDs {
			if positions[i].draft == 0 {
				preferBefore(i, len(committedIDs), len(items))
			}
		}
	}

	clear(seen)
	out := make([]string, 0, len(items))
	var emit func(int)
	emit = func(i int) {
		if seen[i] {
			return
		}
		seen[i] = true
		for _, prev := range predecessors[i] {
			emit(prev)
		}
		out = append(out, items[i])
	}
	for i := range items {
		emit(i)
	}
	return out
}

type subtaskOrderPosition struct {
	draft, snapshot int // one-based, zero if absent
}

// usesDraftOrder reports whether a pair includes a visible addition or was swapped
// after the discard. These are the pairs that take precedence in a conflict.
func (p subtaskOrderPosition) usesDraftOrder(other subtaskOrderPosition) bool {
	return p.draft != 0 && other.draft != 0 &&
		(p.snapshot == 0 || other.snapshot == 0 || (p.draft < other.draft) != (p.snapshot < other.snapshot))
}

// compatibleSubtaskOrder returns the order satisfying every pair, or nil if they
// conflict. One item precedes the other in every pair. Distinct predecessor counts
// therefore give the unique ranks: remove rank zero and repeat on the remaining items.
func compatibleSubtaskOrder(items []string, positions []subtaskOrderPosition) []string {
	counts := make([]int, len(items))
	for left, pos := range positions {
		for right := left + 1; right < len(positions); right++ {
			other := positions[right]
			// By default, committed items stay in order and restored items precede
			// additions. items already lists committed items first.
			if other.draft < pos.draft && pos.usesDraftOrder(other) {
				counts[left]++
			} else {
				counts[right]++
			}
		}
	}
	order := make([]string, len(items))
	occupied := make([]bool, len(items))
	for i, rank := range counts {
		if occupied[rank] {
			return nil
		}
		order[rank] = items[i]
		occupied[rank] = true
	}
	return order
}

// SaveSelectedTask validates a draft and queues an ordered background save when
// Host.Dispatch is configured. A nil error means accepted, not yet persisted;
// FlushPending and PrepareClose prevent losing work before the UI acknowledgement.
func (w *Workbench) SaveSelectedTask() error {
	if w.IsRestoring() || w.detailTaskEpoch != w.currentDataEpoch() {
		return fmt.Errorf("%w: backup restore in progress", model.ErrConflict)
	}
	if w.selectedTaskID == "" || w.detailTask == nil {
		return nil
	}
	w.saveRevision++
	w.pendingSave = nil
	w.discardAfterSave = false
	input := w.taskInputFromDraft()
	if input.Title == "" {
		w.dirty = true
		w.saveState = "validation_error"
		w.saveErr = "任务标题不能为空"
		return model.ErrInvalid
	}

	request := taskSave{taskID: w.selectedTaskID, revision: w.saveRevision, epoch: w.currentDataEpoch(), input: input}
	w.dirty = true
	w.saveState = "saving"
	w.saveErr = ""
	if w.saving {
		w.pendingSave = &request
		return nil
	}
	return w.startTaskSave(request)
}

// rebaseTaskDraftOnCommitted rolls back fields inherited from the failed write
// while preserving newer input and any validation error.
func (w *Workbench) rebaseTaskDraftOnCommitted(snap projects.TaskInput) {
	if w.detailTask == nil {
		return
	}
	committed := *w.detailTask
	draft := w.taskInputFromDraft()
	if draft.Title == snap.Title && draft.Title != committed.Title {
		w.editTitle = committed.Title
	}
	if draft.Description == snap.Description {
		w.editDescription = committed.Description
	}
	if draft.Status == snap.Status {
		w.editStatus = string(committed.Status)
	}
	if draft.Priority == snap.Priority {
		w.editPriority = string(committed.Priority)
	}
	if draft.DueDate == snap.DueDate {
		w.editDueDate = parseTaskDueDate(committed.DueDate)
	}
	w.editTags = rebaseTagsOnCommitted(draft.Tags, snap.Tags, committed.Tags)
	w.editSubtasks = rebaseSubtasksOnCommitted(draft.Subtasks, snap.Subtasks, committed.Subtasks)
}

// SetTaskPriority updates a task's priority directly from a board card and refreshes board filters in place.
func (w *Workbench) SetTaskPriority(taskID string, priority model.Priority) {
	if w.IsRestoring() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再修改优先级"); err != nil {
		return
	}
	idx := slices.IndexFunc(w.boardTasks, func(t model.Task) bool { return t.ID == taskID })
	if idx < 0 {
		return
	}
	t := w.boardTasks[idx]
	if t.Priority == priority {
		return
	}
	service := w.app.Projects
	runBoardWrite(w, t.ProjectID, func() (struct{}, error) {
		return struct{}{}, service.SetTaskPriority(context.Background(), taskID, priority)
	}, func(_ struct{}, err error) {
		w.enqueueErrorToast("更新优先级失败", err)
	})
}

// MoveTaskToColumn moves a task to targetStatus at the given unfiltered index (or end of column when index < 0).
func (w *Workbench) MoveTaskToColumn(taskID string, targetStatus model.Status, unfilteredIndex int) {
	if w.IsRestoring() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再移动任务"); err != nil {
		return
	}
	if unfilteredIndex < 0 {
		dest := w.UnfilteredTasksInColumn(targetStatus)
		count := len(dest)
		for _, t := range dest {
			if t.ID == taskID {
				count--
				break
			}
		}
		unfilteredIndex = count
	}
	service := w.app.Projects
	runBoardWrite(w, w.activeProjectID, func() (model.Task, error) {
		return service.MoveTask(context.Background(), taskID, targetStatus, unfilteredIndex)
	}, func(_ model.Task, err error) {
		w.enqueueErrorToast("移动任务失败", err)
	})
}

// MoveOrReorderTaskToSlot moves draggedTaskID to the insertion gap visibleSlot
// (0..len(visible)) within targetStatus, preserving hidden tasks when filtered.
func (w *Workbench) MoveOrReorderTaskToSlot(draggedTaskID string, targetStatus model.Status, visibleSlot int) {
	if w.IsRestoring() || draggedTaskID == "" || !targetStatus.Valid() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再移动任务"); err != nil {
		return
	}
	var draggedTask *model.Task
	for i := range w.boardTasks {
		if w.boardTasks[i].ID == draggedTaskID {
			draggedTask = &w.boardTasks[i]
			break
		}
	}
	if draggedTask == nil {
		return
	}

	visible := w.TasksInColumn(targetStatus)
	if visibleSlot < 0 {
		visibleSlot = 0
	} else if visibleSlot > len(visible) {
		visibleSlot = len(visible)
	}

	if draggedTask.Status == targetStatus {
		draggedIdx := slices.IndexFunc(visible, func(t model.Task) bool { return t.ID == draggedTaskID })
		if draggedIdx < 0 || visibleSlot == draggedIdx || visibleSlot == draggedIdx+1 {
			return
		}
		ids := make([]string, 0, len(visible))
		for _, t := range visible {
			if t.ID != draggedTaskID {
				ids = append(ids, t.ID)
			}
		}
		insertIdx := visibleSlot
		if visibleSlot > draggedIdx {
			insertIdx--
		}
		ids = slices.Insert(ids, insertIdx, draggedTaskID)
		w.reorderBoard(targetStatus, ids, "排序失败")
		return
	}

	destTasks := w.UnfilteredTasksInColumn(targetStatus)
	destIdx := 0
	switch {
	case len(visible) == 0:
		destIdx = 0
	case visibleSlot == 0:
		if idx := slices.IndexFunc(destTasks, func(t model.Task) bool { return t.ID == visible[0].ID }); idx >= 0 {
			destIdx = idx
		}
	case visibleSlot >= len(visible):
		if idx := slices.IndexFunc(destTasks, func(t model.Task) bool { return t.ID == visible[len(visible)-1].ID }); idx >= 0 {
			destIdx = idx + 1
		} else {
			destIdx = len(destTasks)
		}
	default:
		if idx := slices.IndexFunc(destTasks, func(t model.Task) bool { return t.ID == visible[visibleSlot].ID }); idx >= 0 {
			destIdx = idx
		}
	}
	w.MoveTaskToColumn(draggedTaskID, targetStatus, destIdx)
}

// StepTaskInColumn moves a task up (delta = -1) or down (delta = +1) within its visible column.
func (w *Workbench) StepTaskInColumn(taskID string, status model.Status, delta int) {
	if w.IsRestoring() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再调整顺序"); err != nil {
		return
	}
	visible := w.TasksInColumn(status)
	idx := -1
	for i, t := range visible {
		if t.ID == taskID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	newIdx := idx + delta
	if newIdx < 0 || newIdx >= len(visible) {
		return
	}
	ids := make([]string, len(visible))
	for i, t := range visible {
		ids[i] = t.ID
	}
	ids[idx], ids[newIdx] = ids[newIdx], ids[idx]
	w.reorderBoard(status, ids, "调整顺序失败")
}

func (w *Workbench) reorderBoard(status model.Status, ids []string, failureTitle string) {
	projectID, service := w.activeProjectID, w.app.Projects
	runBoardWrite(w, projectID, func() (struct{}, error) {
		return struct{}{}, service.ReorderTasks(context.Background(), projectID, status, ids)
	}, func(_ struct{}, err error) {
		w.enqueueErrorToast(failureTitle, err)
	})
}

// OpenCreateProjectModal opens the modal to create a new project.
func (w *Workbench) OpenCreateProjectModal() {
	if w.IsRestoring() {
		return
	}
	w.projectModalVersion++
	w.projectModalOpen = true
	w.projectModalEditID = ""
	w.projectModalName = ""
	w.projectModalDesc = ""
	w.projectModalIcon = "folder"
	w.projectModalErr = ""
}

// OpenEditProjectModal opens the modal to edit an existing project.
func (w *Workbench) OpenEditProjectModal(p model.Project) {
	if w.IsRestoring() {
		return
	}
	w.projectModalVersion++
	w.projectModalOpen = true
	w.projectModalEditID = p.ID
	w.projectModalName = p.Name
	w.projectModalDesc = p.Description
	w.projectModalIcon = p.Icon
	if strings.TrimSpace(w.projectModalIcon) == "" {
		w.projectModalIcon = "folder"
	}
	w.projectModalErr = ""
}

// SubmitProjectModal creates or updates a project from the project modal fields.
func (w *Workbench) SubmitProjectModal() bool {
	if w.IsRestoring() {
		return false
	}
	name := strings.TrimSpace(w.projectModalName)
	if name == "" {
		w.projectModalErr = "项目名称不能为空"
		return false
	}
	icon := strings.TrimSpace(w.projectModalIcon)
	if icon == "" {
		icon = "folder"
	}
	draft := w.projectFormInput()
	input := projects.ProjectInput{
		Name:        name,
		Description: strings.TrimSpace(w.projectModalDesc),
		Icon:        icon,
	}
	editID, version, navigation := w.projectModalEditID, w.projectModalVersion, w.navigationVersion
	if editID == "" {
		if err := w.FlushPending(); err != nil {
			w.projectModalErr = "当前打开的任务存在未保存修改（" + w.flushErrorDetail(err) + "），请先修正或放弃修改"
			return false
		}
	}
	service := w.app.Projects
	_, err := runRecordWrite(w, func() (model.Project, error) {
		if editID == "" {
			return service.CreateProject(context.Background(), input)
		}
		return service.UpdateProject(context.Background(), editID, input)
	}, func(project model.Project, err error) {
		current := w.projectModalOpen && w.projectModalVersion == version && w.projectFormInput() == draft
		if err != nil {
			if current {
				w.projectModalErr = err.Error()
			} else {
				w.enqueueErrorToast("保存项目失败", err)
			}
			return
		}
		if current {
			w.projectModalOpen = false
		}
		w.upsertProjectInCache(project)
		title := "已更新项目"
		if editID == "" {
			title = "已创建项目"
			if current && w.navigationVersion == navigation {
				w.SelectProject(project.ID)
			}
		}
		w.EnqueueToast(ToastProps{Title: title, Description: project.Name, Type: ToastSuccess})
	})
	return err == nil
}

func (w *Workbench) projectFormInput() projects.ProjectInput {
	return projects.ProjectInput{Name: w.projectModalName, Icon: w.projectModalIcon, Description: w.projectModalDesc}
}

// ArchiveProject archives an active project.
func (w *Workbench) ArchiveProject(projectID string) {
	if w.IsRestoring() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再归档项目"); err != nil {
		return
	}
	w.setProjectArchived(projectID, true, "项目已归档", "归档项目失败", ToastInfo)
}

// UnarchiveProject restores an archived project back to the active project list.
func (w *Workbench) UnarchiveProject(projectID string) {
	if w.IsRestoring() {
		return
	}
	w.setProjectArchived(projectID, false, "已恢复归档项目", "恢复归档失败", ToastSuccess)
}

func (w *Workbench) setProjectArchived(projectID string, archived bool, success, failure string, severity ToastType) {
	service := w.app.Projects
	runRecordWrite(w, func() (model.Project, error) {
		return service.ArchiveProject(context.Background(), projectID, archived)
	}, func(p model.Project, err error) {
		if err != nil {
			w.enqueueErrorToast(failure, err)
			return
		}
		w.upsertProjectInCache(p)
		w.EnqueueToast(ToastProps{Title: success, Description: p.Name, Type: severity})
	})
}

// SoftDeleteProject moves a project and its tasks to the Recycle Bin.
func (w *Workbench) SoftDeleteProject(projectID string) {
	if w.IsRestoring() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再操作"); err != nil {
		return
	}
	service, navigation := w.app.Projects, w.navigationVersion
	runRecordWrite(w, func() (struct{}, error) {
		return struct{}{}, service.DeleteProject(context.Background(), projectID)
	}, func(_ struct{}, err error) {
		if err != nil {
			w.enqueueErrorToast("移入回收站失败", err)
			return
		}
		w.removeProjectFromCache(projectID)
		delete(w.projectSummaries, projectID)
		if w.activeProjectID == projectID {
			samePage := w.navigationVersion == navigation
			w.resetSelectedTaskState()
			w.activeProjectID = ""
			w.reloadBoardTasks(true)
			if w.navPage == NavProjectBoard {
				if samePage && len(w.activeProjects) > 0 {
					w.SelectProject(w.activeProjects[0].ID)
				} else {
					w.NavigateTo(NavAllProjects)
				}
			}
		}
		w.reloadProjects(true)
		w.reloadTrash(true)
		w.EnqueueToast(ToastProps{
			Title:       "项目已移入回收站",
			Description: "可在回收站中随时恢复或彻底删除",
			Type:        ToastInfo,
		})
	})
}

// OpenCreateTaskModal opens the modal to create a new task in the given column status.
func (w *Workbench) OpenCreateTaskModal(status model.Status) {
	if w.IsRestoring() {
		return
	}
	if status == "" {
		status = model.Planned
	}
	w.taskModalVersion++
	w.taskModalOpen = true
	w.taskModalStatus = string(status)
	w.taskModalPriority = string(model.Medium)
	w.taskModalTitle = ""
	w.taskModalDesc = ""
	w.taskModalTags = ""
	w.taskModalDueDate = time.Time{}
	w.taskModalErr = ""
}

// SubmitTaskModal creates a new task from the task modal fields.
func (w *Workbench) SubmitTaskModal() bool {
	if w.IsRestoring() {
		return false
	}
	if w.activeProjectID == "" {
		w.taskModalErr = "请先选择或创建一个项目"
		return false
	}
	title := strings.TrimSpace(w.taskModalTitle)
	if title == "" {
		w.taskModalErr = "任务标题不能为空"
		return false
	}
	if err := w.FlushPending(); err != nil {
		w.taskModalErr = "当前打开的任务存在未保存修改（" + w.flushErrorDetail(err) + "），请先修正或放弃修改"
		return false
	}

	type taskFormDraft struct {
		title, description, status, priority, tags string
		due                                        time.Time
	}
	readDraft := func() taskFormDraft {
		return taskFormDraft{w.taskModalTitle, w.taskModalDesc, w.taskModalStatus, w.taskModalPriority, w.taskModalTags, w.taskModalDueDate}
	}
	draft := readDraft()
	var tags []string
	for _, part := range strings.FieldsFunc(draft.tags, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	}) {
		if t := strings.TrimSpace(part); t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}

	dueStr := ""
	if !draft.due.IsZero() {
		dueStr = draft.due.Format("2006-01-02")
	}

	input := projects.TaskInput{
		Title:       title,
		Description: draft.description,
		Status:      model.Status(draft.status),
		Priority:    model.Priority(draft.priority),
		Tags:        tags,
		DueDate:     dueStr,
	}
	service, projectID, version := w.app.Projects, w.activeProjectID, w.taskModalVersion
	_, err := runBoardWrite(w, projectID, func() (model.Task, error) {
		return service.CreateTask(context.Background(), projectID, input)
	}, func(created model.Task, err error) {
		current := w.taskModalOpen && w.taskModalVersion == version && readDraft() == draft
		if err != nil {
			if current {
				w.taskModalErr = err.Error()
			} else {
				w.enqueueErrorToast("创建任务失败", err)
			}
			return
		}
		if current {
			w.taskModalOpen = false
		}
		w.EnqueueToast(ToastProps{Title: "已创建任务", Description: created.Title, Type: ToastSuccess})
	})
	return err == nil
}

// QuickCreateTask creates a task from the quick input. With Dispatch, a nil error
// means the write was accepted and the task will appear after acknowledgement.
func (w *Workbench) QuickCreateTask(status model.Status, title string) (*model.Task, error) {
	if w.IsRestoring() {
		return nil, fmt.Errorf("%w: backup restore in progress", model.ErrConflict)
	}
	if w.activeProjectID == "" {
		err := fmt.Errorf("%w: 请先选择或创建一个项目", model.ErrInvalid)
		w.enqueueErrorToast("创建任务失败", err)
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, model.ErrInvalid
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再新建任务"); err != nil {
		return nil, err
	}
	service, projectID, draft := w.app.Projects, w.activeProjectID, w.quickTaskTitles[status]
	created, err := runBoardWrite(w, projectID, func() (*model.Task, error) {
		task, err := service.CreateTask(context.Background(), projectID, projects.TaskInput{
			Title: title, Status: status, Priority: model.NoPriority,
		})
		if err != nil {
			return nil, err
		}
		return &task, nil
	}, func(_ *model.Task, err error) {
		if err != nil {
			w.enqueueErrorToast("创建任务失败", err)
			return
		}
		if w.quickTaskTitles[status] == draft {
			w.quickTaskTitles[status] = ""
		}
	})
	return created, err
}

// SoftDeleteTask moves a task to the Recycle Bin.
func (w *Workbench) SoftDeleteTask(taskID string) {
	if w.IsRestoring() || taskID == "" {
		return
	}
	if w.recordWritePending {
		w.enqueueErrorToast("正在保存更改", errRecordWritePending)
		return
	}
	if w.saving && w.selectedTaskID == taskID {
		w.EnqueueToast(ToastProps{Title: "正在保存任务", Description: "保存完成后再移入回收站", Type: ToastInfo})
		return
	}
	if w.selectedTaskID == taskID {
		w.pendingSave = nil
		w.inFlightSave = nil
		w.discardAfterSave = false
		w.discardRebase = false
		w.afterSave = nil
	} else {
		if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再删除其他任务"); err != nil {
			return
		}
	}
	service := w.app.Projects
	w.deletingTaskID = taskID
	runBoardWrite(w, w.activeProjectID, func() (struct{}, error) {
		return struct{}{}, service.DeleteTask(context.Background(), taskID)
	}, func(_ struct{}, err error) {
		w.deletingTaskID = ""
		if err != nil {
			w.enqueueErrorToast("移入回收站失败", err)
			return
		}
		if w.selectedTaskID == taskID {
			w.resetSelectedTaskState()
		}
		w.reloadTrash(true)
		w.EnqueueToast(ToastProps{
			Title:       "任务已移入回收站",
			Description: "可在回收站中恢复或永久删除",
			Type:        ToastInfo,
		})
	})
}

func (w *Workbench) importAttachmentsWork(ctx context.Context, taskID string, paths []string) (int, []ToastProps) {
	imported := 0
	var toasts []ToastProps
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, err := w.app.Attachments.Import(ctx, taskID, p)
		if err != nil {
			toasts = append(toasts, ToastProps{
				Title:       "导入附件失败: " + filepath.Base(p),
				Description: err.Error(),
				Type:        ToastError,
			})
			continue
		}
		imported++
	}
	if imported > 0 {
		toasts = append(toasts, ToastProps{
			Title:       "附件已导入",
			Description: fmt.Sprintf("成功导入 %d 个附件", imported),
			Type:        ToastSuccess,
		})
	}
	return imported, toasts
}

func (w *Workbench) applyImportedAttachments(imported int, toasts []ToastProps) {
	for _, t := range toasts {
		w.EnqueueToast(t)
	}
	if imported > 0 {
		w.reloadBoardTasks(true)
	}
}

// ImportAttachmentsToSelectedTask imports local file paths into the currently selected task.
func (w *Workbench) ImportAttachmentsToSelectedTask(paths []string) {
	if w.IsRestoring() || w.selectedTaskID == "" || len(paths) == 0 {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再导入附件"); err != nil {
		return
	}
	taskID := w.selectedTaskID
	pathsCopy := slices.Clone(paths)
	w.runAsyncTask("正在导入附件...", func(ctx context.Context) func() {
		imported, toasts := w.importAttachmentsWork(ctx, taskID, pathsCopy)
		return func() {
			w.applyImportedAttachments(imported, toasts)
		}
	})
}

// PickAndImportAttachmentsToSelectedTask opens the native file picker and imports chosen files off the UI thread.
func (w *Workbench) PickAndImportAttachmentsToSelectedTask() {
	if w.IsRestoring() {
		return
	}
	pick := w.host.PickOpenFiles
	if pick == nil || w.selectedTaskID == "" {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再导入附件"); err != nil {
		return
	}
	taskID := w.selectedTaskID
	w.runAsyncTask("正在选择并导入附件...", func(ctx context.Context) func() {
		paths, err := pick("选择要导入到任务的附件", true)
		if err != nil {
			return func() {
				w.enqueueErrorToast("选择附件失败", err)
			}
		}
		if len(paths) == 0 {
			return nil
		}
		imported, toasts := w.importAttachmentsWork(ctx, taskID, paths)
		return func() {
			w.applyImportedAttachments(imported, toasts)
		}
	})
}

// RemoveAttachmentFromSelectedTask removes an attachment from the selected task and runs GC.
func (w *Workbench) RemoveAttachmentFromSelectedTask(attachmentID string) {
	if w.IsRestoring() || w.selectedTaskID == "" {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再移除附件"); err != nil {
		return
	}
	w.runAsyncTask("正在移除附件并清理存储...", func(ctx context.Context) func() {
		if err := w.app.Attachments.Remove(ctx, attachmentID); err != nil {
			return func() {
				w.enqueueErrorToast("移除附件失败", err)
			}
		}
		_, gcErr := w.app.Attachments.GC(ctx)
		return func() {
			delete(w.attachmentPaths, attachmentID)
			delete(w.attachmentMissing, attachmentID)
			delete(w.bitmapCache, attachmentID)
			delete(w.bitmapFailed, attachmentID)
			w.reloadBoardTasks(true)
			if gcErr != nil {
				// The row is gone; the file stays until the next successful cleanup.
				w.EnqueueToast(ToastProps{
					Title:       "已移除附件，但清理文件失败",
					Description: gcErr.Error(),
					Type:        ToastWarning,
				})
				return
			}
			w.EnqueueToast(ToastProps{
				Title:       "已移除附件",
				Description: "任务附件关联已移除并清理未引用文件",
				Type:        ToastInfo,
			})
		}
	})
}

func (w *Workbench) applyExportAttachmentResult(destPath string, err error) {
	if err != nil {
		w.enqueueErrorToast("导出附件失败", err)
		return
	}
	w.EnqueueToast(ToastProps{
		Title:       "附件已另存为",
		Description: destPath,
		Type:        ToastSuccess,
	})
}

// PickAndExportAttachment opens the save dialog and exports att off the UI thread.
func (w *Workbench) PickAndExportAttachment(att model.Attachment) {
	if w.IsRestoring() {
		return
	}
	pick := w.host.PickSaveFile
	if pick == nil {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	ext := strings.TrimPrefix(filepath.Ext(att.Name), ".")
	var exts []string
	if ext != "" {
		exts = []string{ext}
	}
	w.runAsyncTask("正在导出附件...", func(ctx context.Context) func() {
		dest, err := pick("导出附件", att.Name, exts...)
		if err != nil {
			return func() {
				w.enqueueErrorToast("选择导出路径失败", err)
			}
		}
		dest = strings.TrimSpace(dest)
		if dest == "" {
			return nil
		}
		exportErr := w.app.Attachments.ExportOverwrite(ctx, att.ID, dest)
		return func() {
			w.applyExportAttachmentResult(dest, exportErr)
		}
	})
}

func (w *Workbench) resolveManagedAttachmentPathLockFree(att model.Attachment) string {
	if w.app == nil || w.app.DataDir() == "" {
		return ""
	}
	name := strings.TrimSpace(att.StorageName)
	if name == "" || name == "." || name == ".." || name != filepath.Base(name) || strings.ContainsAny(name, "/\\:\x00") {
		return ""
	}
	candidate := filepath.Join(w.app.DataDir(), "attachments", name)
	st, err := os.Lstat(candidate)
	if err != nil || !st.Mode().IsRegular() {
		return ""
	}
	return candidate
}

func (w *Workbench) cacheAttachments(atts []model.Attachment) {
	for _, att := range atts {
		if p := w.resolveManagedAttachmentPathLockFree(att); p != "" {
			w.attachmentPaths[att.ID] = p
			delete(w.attachmentMissing, att.ID)
		} else {
			delete(w.attachmentPaths, att.ID)
			w.attachmentMissing[att.ID] = true
		}
	}
	w.pruneAttachmentCaches()
}

func (w *Workbench) pruneAttachmentCaches() {
	defer func() {
		w.bitmapOrder = slices.DeleteFunc(w.bitmapOrder, func(id string) bool { return w.bitmapCache[id] == nil })
	}()
	if w.selectedTaskID == "" || len(w.detailAttachments) == 0 {
		if len(w.attachmentPaths) > 0 {
			w.attachmentPaths = make(map[string]string)
		}
		if len(w.attachmentMissing) > 0 {
			w.attachmentMissing = make(map[string]bool)
		}
		if len(w.bitmapCache) > 0 {
			w.bitmapCache = make(map[string]*ui.Bitmap)
		}
		if len(w.bitmapFailed) > 0 {
			w.bitmapFailed = make(map[string]bool)
		}
		return
	}
	keep := make(map[string]bool, len(w.detailAttachments))
	for _, att := range w.detailAttachments {
		keep[att.ID] = true
	}
	for id := range w.attachmentPaths {
		if !keep[id] {
			delete(w.attachmentPaths, id)
		}
	}
	for id := range w.attachmentMissing {
		if !keep[id] {
			delete(w.attachmentMissing, id)
		}
	}
	for id := range w.bitmapCache {
		if !keep[id] {
			delete(w.bitmapCache, id)
		}
	}
	for id := range w.bitmapFailed {
		if !keep[id] {
			delete(w.bitmapFailed, id)
		}
	}
}

// AttachmentManagedPath returns the local managed file path for an attachment without blocking on the store lock.
func (w *Workbench) AttachmentManagedPath(att model.Attachment) string {
	if p, ok := w.attachmentPaths[att.ID]; ok && p != "" {
		return p
	}
	if w.attachmentMissing[att.ID] {
		return ""
	}
	if p := w.resolveManagedAttachmentPathLockFree(att); p != "" {
		w.attachmentPaths[att.ID] = p
		return p
	}
	w.attachmentMissing[att.ID] = true
	return ""
}

// ReloadTrash reloads deleted items across all modules from the backend.
func (w *Workbench) ReloadTrash() { w.reloadTrash(false) }

func (w *Workbench) reloadTrash(async bool) {
	app := w.app
	w.loadUI("trash", async, func() func() {
		items, err := app.ListTrash(context.Background())
		return func() {
			if err != nil {
				w.enqueueErrorToast("加载回收站失败", err)
				return
			}
			w.trashItems = items
			w.rebuildProjectNames()
		}
	})
}

// RestoreTrashItem restores a deleted item (project, task, or job application) from the Recycle Bin.
// If restoring a task fails with model.ErrParentDeleted, it opens a helper modal
// allowing the user to restore the parent project and the task together.
func (w *Workbench) RestoreTrashItem(item model.TrashItem) {
	if w.IsRestoring() {
		return
	}
	app, navigation := w.app, w.navigationVersion
	work := func() (model.JobApplication, error) {
		ctx := context.Background()
		if err := app.RestoreTrash(ctx, item.Kind, item.ID); err != nil {
			return model.JobApplication{}, err
		}
		if item.Kind == jobapplications.TrashKind {
			restored, _ := app.JobApplications.Get(ctx, item.ID)
			return restored, nil
		}
		return model.JobApplication{}, nil
	}
	apply := func(restored model.JobApplication, err error) {
		if errors.Is(err, model.ErrParentDeleted) && w.navigationVersion == navigation {
			itemCopy := item
			w.trashOrphanTask = &itemCopy
			w.trashOrphanParent = nil
			for i := range w.trashItems {
				if w.trashItems[i].Kind == projects.TrashProject && w.trashItems[i].ID == item.ProjectID {
					parentCopy := w.trashItems[i]
					w.trashOrphanParent = &parentCopy
					break
				}
			}
			w.trashParentErrModalOpen = true
			return
		}
		if err != nil {
			w.enqueueErrorToast("恢复失败", err)
			return
		}
		if item.Kind != jobapplications.TrashKind {
			w.reloadProjects(true)
		} else if restored.ID != "" {
			w.upsertJobInCache(restored)
		} else {
			w.reloadJobApplications(true)
		}
		w.removeRestoredTrashItem(item)
		w.EnqueueToast(ToastProps{Title: "已从回收站恢复", Description: item.Name, Type: ToastSuccess})
	}
	if item.Kind == projects.TrashTask && item.ProjectID != "" {
		runBoardWrite(w, item.ProjectID, work, apply)
	} else {
		runRecordWrite(w, work, apply)
	}
}

func (w *Workbench) removeRestoredTrashItem(item model.TrashItem) {
	w.trashItems = slices.DeleteFunc(w.trashItems, func(t model.TrashItem) bool { return t.Kind == item.Kind && t.ID == item.ID })
	w.rebuildProjectNames()
	w.reloadTrash(true)
}

// RestoreOrphanTaskWithParent restores the deleted parent project and then restores the task.
func (w *Workbench) RestoreOrphanTaskWithParent() {
	if w.IsRestoring() {
		return
	}
	if w.trashOrphanTask == nil {
		w.trashParentErrModalOpen = false
		return
	}
	selected, service := w.trashOrphanTask, w.app.Projects
	taskItem := *selected
	runBoardWrite(w, taskItem.ProjectID, func() (error, error) {
		ctx := context.Background()
		if taskItem.ProjectID != "" {
			if err := service.RestoreProject(ctx, taskItem.ProjectID); err != nil {
				return nil, err
			}
		}
		if taskItem.ID != "" {
			return service.RestoreTask(ctx, taskItem.ID), nil
		}
		return nil, nil
	}, func(taskErr, parentErr error) {
		if parentErr != nil {
			w.enqueueErrorToast("恢复所属项目失败", parentErr)
			return
		}
		if w.trashOrphanTask == selected {
			w.trashParentErrModalOpen = false
			w.trashOrphanTask = nil
			w.trashOrphanParent = nil
		}
		w.reloadProjects(true)
		if taskErr != nil {
			w.reloadTrash(true)
			w.enqueueErrorToast("恢复任务失败（所属项目已恢复）", taskErr)
			return
		}
		w.removeRestoredTrashItem(taskItem)
		w.EnqueueToast(ToastProps{Title: "已恢复项目与任务", Description: taskItem.Name, Type: ToastSuccess})
	})
}

// ConfirmPermanentDelete executes app.PermanentlyDelete for the pending trash item.
func (w *Workbench) ConfirmPermanentDelete() {
	if w.IsRestoring() {
		return
	}
	if w.confirmPermDeleteItem == nil {
		w.confirmPermDeleteOpen = false
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if w.recordWritePending {
		w.enqueueErrorToast("正在保存更改", errRecordWritePending)
		return
	}
	item := *w.confirmPermDeleteItem
	w.confirmPermDeleteOpen = false
	w.confirmPermDeleteItem = nil
	started := w.runAsyncTask("正在彻底删除并清理未引用附件...", func(ctx context.Context) func() {
		err := w.app.PermanentlyDelete(ctx, item.Kind, item.ID)
		return func() {
			var cleanupErr *backend.AttachmentCleanupError
			if err != nil && !errors.As(err, &cleanupErr) {
				w.enqueueErrorToast("彻底删除失败", err)
				return
			}
			if w.trashOrphanTask != nil && (w.trashOrphanTask.ID == item.ID || (item.Kind == projects.TrashProject && w.trashOrphanTask.ProjectID == item.ID)) {
				w.trashParentErrModalOpen = false
				w.trashOrphanTask = nil
				w.trashOrphanParent = nil
			}
			w.reloadProjects(true)
			w.reloadJobApplications(true)
			w.reloadTrash(true)
			w.pruneAttachmentCaches()
			if cleanupErr != nil {
				w.reportTrashCleanupError(cleanupErr)
				return
			}
			w.EnqueueToast(ToastProps{
				Title:       "已彻底删除",
				Description: item.Name,
				Type:        ToastInfo,
			})
		}
	})
	if !started {
		itemCopy := item
		w.confirmPermDeleteItem = &itemCopy
		w.confirmPermDeleteOpen = true
	}
}

// ConfirmClearTrash executes app.ClearTrash.
func (w *Workbench) ConfirmClearTrash() {
	if w.IsRestoring() {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if w.recordWritePending {
		w.enqueueErrorToast("正在保存更改", errRecordWritePending)
		return
	}
	w.confirmClearTrashOpen = false
	started := w.runAsyncTask("正在清空回收站并清理未引用附件...", func(ctx context.Context) func() {
		err := w.app.ClearTrash(ctx)
		return func() {
			var cleanupErr *backend.AttachmentCleanupError
			if err != nil && !errors.As(err, &cleanupErr) {
				w.enqueueErrorToast("清空回收站失败", err)
				return
			}
			w.trashParentErrModalOpen = false
			w.trashOrphanTask = nil
			w.trashOrphanParent = nil
			w.confirmPermDeleteOpen = false
			w.confirmPermDeleteItem = nil
			w.reloadProjects(true)
			w.reloadJobApplications(true)
			w.reloadTrash(true)
			w.pruneAttachmentCaches()
			if cleanupErr != nil {
				w.reportTrashCleanupError(cleanupErr)
				return
			}
			w.EnqueueToast(ToastProps{
				Title:       "回收站已清空",
				Description: "所有回收站项目、任务、投递记录与未引用附件均已彻底清理",
				Type:        ToastInfo,
			})
		}
	})
	if !started {
		w.confirmClearTrashOpen = true
	}
}

func (w *Workbench) reportTrashCleanupError(err error) {
	w.EnqueueToast(ToastProps{
		Title:       "条目已删除，但清理附件失败",
		Description: err.Error(),
		Type:        ToastWarning,
	})
}

func themeSegmentForSetting(th string) int {
	switch th {
	case ThemeLight:
		return 1
	case ThemeDark:
		return 2
	default:
		return 0
	}
}

// ReloadSettings reloads the theme preference and automatic backup policy/status.
func (w *Workbench) ReloadSettings() { w.reloadSettings(false) }

func (w *Workbench) reloadSettings(async bool) {
	settings, backups := w.app.Settings, w.app.Backups
	initialTheme := w.themeSetting
	initialPolicy := w.backupPolicyDraft()
	w.loadUI("settings", async, func() func() {
		ctx := context.Background()
		th, themeErr := settings.Theme(ctx)
		pol, policyErr := backups.GetPolicy(ctx)
		st, statusErr := backups.Status(ctx)
		return func() {
			if !w.settingWritePending(themePreference) {
				if themeErr == nil && w.themeSetting == initialTheme {
					w.themeSetting = th
				}
				w.themeSegmentIdx = themeSegmentForSetting(w.themeSetting)
			}
			if policyErr == nil && !w.settingWritePending(backupPreference) && w.backupPolicyDraft() == initialPolicy {
				w.applyBackupPolicy(pol)
			}
			if statusErr == nil {
				w.backupStatus = st
			}
		}
	})
}

func (w *Workbench) applyBackupPolicy(pol backup.Policy) {
	w.backupEnabled = pol.Enabled
	w.backupIntervalSelect = formatIntervalOption(pol.Interval)
	w.backupKeepSelect = fmt.Sprint(pol.Keep)
	w.backupDirInput = pol.Directory
}

// backupPolicyDraft compares the raw controls so an in-flight Settings load
// cannot replace edits made after navigation started.
func (w *Workbench) backupPolicyDraft() struct {
	enabled             bool
	interval, keep, dir string
} {
	return struct {
		enabled             bool
		interval, keep, dir string
	}{w.backupEnabled, w.backupIntervalSelect, w.backupKeepSelect, w.backupDirInput}
}

// formatIntervalOption returns the select value for d: a preset key when one
// matches exactly, otherwise the duration itself so saving never rewrites it.
func formatIntervalOption(d time.Duration) string {
	for _, opt := range backupIntervalOptions {
		if parseIntervalOption(opt.Value) == d {
			return opt.Value
		}
	}
	return d.String()
}

func parseIntervalOption(val string) time.Duration {
	if d, err := time.ParseDuration(val); err == nil && d > 0 {
		return d
	}
	return 24 * time.Hour
}

// intervalLabel describes a non-preset interval in the presets' wording.
func intervalLabel(d time.Duration) string {
	switch {
	case d%(24*time.Hour) == 0:
		return fmt.Sprintf("每 %d 天", d/(24*time.Hour))
	case d%time.Hour == 0:
		return fmt.Sprintf("每 %d 小时", d/time.Hour)
	default:
		return "每 " + d.String()
	}
}

// withCurrentOption appends an entry for current when it is not one of the
// presets, so a policy set outside the UI is shown and preserved as it is.
func withCurrentOption(presets []selectOption, current, label string) []selectOption {
	if current == "" || slices.ContainsFunc(presets, func(o selectOption) bool { return o.Value == current }) {
		return presets
	}
	return append(slices.Clone(presets), selectOption{Value: current, Label: label})
}

// ApplyThemeSetting persists and applies the given theme preference.
func (w *Workbench) ApplyThemeSetting(th string) {
	if w.IsRestoring() {
		w.themeSegmentIdx = themeSegmentForSetting(w.themeSetting)
		return
	}
	w.themeSegmentIdx = themeSegmentForSetting(th)
	settings := w.app.Settings
	w.saveSetting(themePreference, func() func() {
		err := settings.SetTheme(context.Background(), th)
		return func() {
			if err != nil {
				w.enqueueErrorToast("保存主题设置失败", err)
			} else {
				w.themeSetting = th
				if w.host.SetTheme != nil {
					w.host.SetTheme(th)
				}
			}
			if !w.settingWritePending(themePreference) {
				w.themeSegmentIdx = themeSegmentForSetting(w.themeSetting)
			}
		}
	})
}

// SaveBackupPolicy persists the current backup policy controls to SQLite and reconfigures the scheduler.
func (w *Workbench) SaveBackupPolicy() {
	if w.IsRestoring() {
		return
	}
	draft := w.backupPolicyDraft()
	keep := 7
	if rawKeep := strings.TrimSpace(draft.keep); rawKeep != "" {
		if parsed, err := strconv.Atoi(rawKeep); err == nil {
			keep = parsed
		}
	}
	pol := backup.Policy{
		Enabled:   draft.enabled,
		Interval:  parseIntervalOption(draft.interval),
		Keep:      keep,
		Directory: strings.TrimSpace(draft.dir),
	}
	backups := w.app.Backups
	w.saveSetting(backupPreference, func() func() {
		ctx := context.Background()
		if err := backups.SetPolicy(ctx, pol); err != nil {
			return func() { w.enqueueErrorToast("保存自动备份策略失败", err) }
		}
		saved, err := backups.GetPolicy(ctx)
		return func() {
			if err == nil && !w.settingWritePending(backupPreference) && w.backupPolicyDraft() == draft {
				w.applyBackupPolicy(saved)
			}
			if !w.settingWritePending(backupPreference) {
				w.EnqueueToast(ToastProps{Title: "自动备份策略已保存", Description: "本地自动备份调度器已更新", Type: ToastSuccess})
			}
		}
	})
}

// RunDueBackupNow triggers app.Backups.RunDue(ctx) off the UI thread.
func (w *Workbench) RunDueBackupNow() {
	if w.IsRestoring() {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if w.recordWritePending || len(w.settingWrites) > 0 || len(w.jobWrites) > 0 {
		w.EnqueueToast(ToastProps{Title: "正在保存更改", Description: "保存完成后再执行备份", Type: ToastInfo})
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再执行备份"); err != nil {
		return
	}
	w.runAsyncTask("正在检查并执行到期自动备份...", func(ctx context.Context) func() {
		path, err := w.app.Backups.RunDue(ctx)
		st, stErr := w.app.Backups.Status(ctx)
		return func() {
			if stErr == nil {
				w.backupStatus = st
			}
			if err != nil {
				w.enqueueErrorToast("自动备份执行失败", err)
				return
			}
			if path != "" || (stErr == nil && st.LastError == "") {
				w.mu.Lock()
				w.lastAsyncErr = ""
				w.mu.Unlock()
			}
			if path != "" {
				w.EnqueueToast(ToastProps{
					Title:       "已执行到期自动备份",
					Description: path,
					Type:        ToastSuccess,
				})
			} else {
				w.EnqueueToast(ToastProps{
					Title:       "当前无需执行自动备份",
					Description: "距离上次自动备份尚未到达设定周期，或自动备份未开启",
					Type:        ToastInfo,
				})
			}
		}
	})
}

func (w *Workbench) applyExportBackupResult(destPath string, err error) {
	if err != nil {
		w.enqueueErrorToast("导出完整备份失败", err)
		return
	}
	w.lastExportedZipPath = destPath
	w.EnqueueToast(ToastProps{
		Title:       "完整备份导出成功",
		Description: destPath,
		Type:        ToastSuccess,
	})
}

// PickAndExportManualBackup opens the save dialog and exports a full backup ZIP off the UI thread.
func (w *Workbench) PickAndExportManualBackup() {
	if w.IsRestoring() {
		return
	}
	pick := w.host.PickSaveFile
	if pick == nil {
		return
	}
	if w.IsBusy() {
		w.enqueueBusyToast()
		return
	}
	if w.recordWritePending || len(w.settingWrites) > 0 || len(w.jobWrites) > 0 {
		w.EnqueueToast(ToastProps{Title: "正在保存更改", Description: "保存完成后再导出备份", Type: ToastInfo})
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再导出备份"); err != nil {
		return
	}
	defaultName := "aion-backup-" + w.now().Local().Format("20060102-150405") + ".zip"
	w.runAsyncTask("正在导出完整备份 (.zip)...", func(ctx context.Context) func() {
		dest, err := pick("导出 Aion 完整备份", defaultName, "zip")
		if err != nil {
			return func() {
				w.enqueueErrorToast("选择备份保存路径失败", err)
			}
		}
		dest = strings.TrimSpace(dest)
		if dest == "" {
			return nil
		}
		exportErr := w.app.Backups.ExportOverwrite(ctx, dest)
		return func() {
			w.applyExportBackupResult(dest, exportErr)
		}
	})
}

// RequestRestoreBackup opens the confirmation modal for restoring from zipPath.
func (w *Workbench) RequestRestoreBackup(zipPath string) {
	if w.IsRestoring() {
		return
	}
	zipPath = strings.TrimSpace(zipPath)
	if zipPath == "" {
		return
	}
	w.pendingRestoreZip = zipPath
	w.confirmRestoreOpen = true
}

func (w *Workbench) applyRestoredState(res backup.RestoreResult) {
	w.lastRestoreResult = &res
	w.lastRestoreError = ""
	w.mu.Lock()
	w.lastAsyncErr = ""
	w.mu.Unlock()

	// Reset all pre-restore UI/filter/modal/draft state WITHOUT flushing old drafts into the restored DB.
	w.afterSave = nil
	w.boardNeedsReload = false
	w.activeProjectID = ""
	w.boardTasks = nil
	w.filteredTasks = nil
	w.allBoardTags = nil
	w.taskAttachments = make(map[string][]model.Attachment)
	w.projectNames = make(map[string]string)
	w.searchQuery = ""
	w.priorityFilter = ""
	w.tagFilters = nil
	w.quickTaskTitles = make(map[model.Status]string)
	w.trashSearch = ""
	w.trashKindFilter = ""
	w.trashItems = nil
	w.trashParentErrModalOpen = false
	w.trashOrphanTask = nil
	w.trashOrphanParent = nil
	w.confirmPermDeleteOpen = false
	w.confirmPermDeleteItem = nil
	w.confirmClearTrashOpen = false
	w.projectModalOpen = false
	w.projectModalEditID = ""
	w.projectModalName = ""
	w.projectModalDesc = ""
	w.projectModalIcon = ""
	w.projectModalErr = ""
	w.taskModalOpen = false
	w.taskModalTitle = ""
	w.taskModalDesc = ""
	w.taskModalTags = ""
	w.taskModalDueDate = time.Time{}
	w.taskModalErr = ""
	w.pendingRestoreZip = ""
	w.confirmRestoreOpen = false
	w.lastSavedAt = time.Time{}

	// Reset Job Applications UI/filter/modal/draft state cleanly without writing old drafts into restored DB.
	w.jobApps = nil
	w.filteredJobApps = nil
	w.jobStats = model.JobApplicationStats{}
	w.jobSearchQuery = ""
	w.jobStageFilter = ""
	w.jobOutcomeFilter = ""
	w.jobAppliedDateFilter = time.Time{}
	w.jobSortBy = jobapplications.SortAppliedDesc
	w.jobPage = 0
	w.jobTableScroll = ui.ScrollState{}
	w.jobCreateOpen = false
	w.jobCreateCompany = ""
	w.jobCreatePosition = ""
	w.jobCreateAppliedDate = time.Time{}
	w.jobCreateStage = ""
	w.jobCreateURL = ""
	w.jobCreateChannel = ""
	w.jobCreateNotes = ""
	w.jobCreateErr = ""
	w.jobCloseOpen = false
	w.jobCloseAppID = ""
	w.jobCloseOutcome = ""
	w.jobCloseReason = ""
	w.jobCloseConfirmOffer = false
	w.jobCloseErr = ""
	w.jobCloseOpID = ""
	w.pendingStageAppID = ""
	w.pendingStageTarget = ""
	w.pendingStageOpID = ""
	w.jobTimelineDesc = true
	w.jobBackfillExpanded = false
	w.editingJobEventID = ""
	w.editingJobEventNote = ""
	w.pendingQuickCloseAppID = ""
	w.pendingQuickCloseOpID = ""
	w.jobLastSavedAt = time.Time{}

	w.resetSelectedTaskState()
	w.resetSelectedJobState()
	w.ReloadAll()
	if len(w.activeProjects) > 0 {
		w.activeProjectID = w.activeProjects[0].ID
		w.ReloadBoardTasks()
	} else {
		w.activeProjectID = ""
		w.ReloadBoardTasks()
		if w.navPage == NavProjectBoard {
			w.navPage = NavAllProjects
			w.lastProjectPage = NavAllProjects
		}
	}
	if w.host.SetTheme != nil {
		w.host.SetTheme(w.themeSetting)
	}
	w.EnqueueToast(ToastProps{
		Title:       "已从备份完整恢复数据",
		Description: "恢复前安全备份已保存至: " + res.SafetyBackupPath,
		Type:        ToastSuccess,
	})
}

// TriggerRestoreBackup executes app.Backups.Restore off the UI thread via runAsyncTask.
func (w *Workbench) TriggerRestoreBackup() {
	if w.saving || w.jobSaving || len(w.jobWrites) > 0 || w.recordWritePending || len(w.settingWrites) > 0 {
		w.EnqueueToast(ToastProps{Title: "正在保存未完成修改", Description: "保存完成后再恢复备份", Type: ToastInfo})
		return
	}
	if w.IsBusy() || w.IsRestoring() {
		return
	}
	zipPath := strings.TrimSpace(w.pendingRestoreZip)
	w.confirmRestoreOpen = false
	if zipPath == "" {
		return
	}
	w.setRestoring(true)
	started := w.runAsyncTask("正在校验备份包并执行完整恢复...", func(ctx context.Context) func() {
		res, err := w.app.Backups.Restore(ctx, zipPath)
		if err == nil {
			w.bumpDataEpoch()
		}
		return func() {
			defer w.setRestoring(false)
			if err != nil {
				w.lastRestoreResult = nil
				w.lastRestoreError = err.Error()
				if res.SafetyBackupPath != "" {
					w.lastRestoreError += "\n恢复前安全备份：" + res.SafetyBackupPath
				}
				w.EnqueueToast(ToastProps{Title: "备份恢复失败", Description: w.lastRestoreError, Type: ToastError})
				return
			}
			w.applyRestoredState(res)
		}
	})
	if !started {
		w.setRestoring(false)
	}
}

func (w *Workbench) drainPendingToasts(c *ui.Context) {
	w.mu.Lock()
	toasts := w.pendingToasts
	w.pendingToasts = nil
	w.mu.Unlock()
	for _, t := range toasts {
		uiShowToast(c, t)
	}
}

// View renders the entire Aion native UI for one frame.
func (w *Workbench) View(c *ui.Context) {
	// Apply theme according to saved setting (system / light / dark)
	switch w.themeSetting {
	case ThemeLight:
		c.SetTheme(uiLightTheme)
	case ThemeDark:
		c.SetTheme(uiDarkTheme)
	default:
		applyUITheme(c)
	}
	tk := tokensOf(c)

	c.OnWindowShortcut(ui.Cmd, ui.KeyComma, func() {
		if !c.ShortcutRepeated() {
			w.OpenSettings()
		}
	})

	ui.Row(c).Fill().Background(tk.BgLayout).TextColor(tk.Text).FontSize(tk.FontSize).Children(func() {
		// Left Collapsible & Resizable Navigation Sidebar
		w.renderSidebar(c, tk)

		// Center Main Workspace Content
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).AlignSelf(ui.Stretch).Background(tk.BgLayout).Children(func() {
			switch w.navPage {
			case NavAllProjects:
				w.renderProjectsView(c, tk, false)
			case NavArchivedProjects:
				w.renderProjectsView(c, tk, true)
			case NavProjectBoard:
				w.renderBoardView(c, tk)
			case NavJobAll, NavJobActive, NavJobOffer, NavJobClosed:
				w.renderJobApplicationsView(c, tk)
			case NavTrash:
				w.renderTrashView(c, tk)
			case NavSettings:
				w.renderSettingsView(c, tk)
			default:
				w.renderProjectsView(c, tk, false)
			}
		})
	})

	// Global Modals
	w.renderProjectModal(c, tk)
	w.renderTaskCreateModal(c, tk)
	w.renderTaskDetailModal(c, tk)
	w.renderJobCreateModal(c, tk)
	w.renderJobDetailModal(c, tk)
	w.renderJobCloseModal(c, tk)
	w.renderTrashModals(c, tk)
	w.renderSettingsModals(c, tk)

	// Drain queued toasts right before ToastHost so same-frame action toasts display immediately.
	w.drainPendingToasts(c)

	// Toast Host
	uiToastHost(c)
}
