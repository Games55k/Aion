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
	discardRebase     bool
	loadVersions      map[string]uint64
	navigationVersion uint64
	boardNeedsReload  bool

	// Preview caches are owned by the UI thread; workers publish through Host.Dispatch.
	attachmentPaths   map[string]string
	attachmentMissing map[string]bool
	bitmapCache       map[string]*ui.Bitmap
	bitmapFailed      map[string]bool
	bitmapOrder       []string
	bitmapLoading     bool
	bitmapLoader      func(string) (*ui.Bitmap, error)

	// Create/Edit Project modal state
	projectModalOpen   bool
	projectModalEditID string
	projectModalName   string
	projectModalDesc   string
	projectModalIcon   string
	projectModalErr    string

	// Create Task modal state
	taskModalOpen     bool
	taskModalStatus   string
	taskModalPriority string
	taskModalTitle    string
	taskModalDesc     string
	taskModalTags     string
	taskModalDueDate  time.Time
	taskModalErr      string

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

	projectID := w.activeProjectID
	service, attachments := w.app.Projects, w.app.Attachments
	w.loadUI("board", async, func() func() {
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
	})
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
	if atts, ok := w.taskAttachments[t.ID]; ok {
		w.detailAttachments = slices.Clone(atts)
		w.cacheAttachments(atts)
	} else if atts, err := w.app.Attachments.List(context.Background(), t.ID); err == nil {
		w.detailAttachments = atts
		w.taskAttachments[t.ID] = atts
		w.cacheAttachments(atts)
	}
	w.resetDetailEditFields(t)
}

// resetDetailEditFields copies a task's content into the draft inputs without
// replacing detailTask, so the last committed task stays the failure baseline.
func (w *Workbench) resetDetailEditFields(t model.Task) {
	w.copyDetailEditFields(t)
	w.dirty = false
	w.saveState = "saved"
	w.saveErr = ""
}

// copyDetailEditFields copies a task's content into the draft inputs and leaves the
// save state untouched, for rebasing drafts that are still invalid or in flight.
func (w *Workbench) copyDetailEditFields(t model.Task) {
	w.editTitle = t.Title
	w.editStatus = string(t.Status)
	w.editPriority = string(t.Priority)
	if strings.TrimSpace(t.DueDate) != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(t.DueDate), time.Local); err == nil {
			w.editDueDate = parsed
		} else {
			w.editDueDate = time.Time{}
		}
	} else {
		w.editDueDate = time.Time{}
	}
	w.editTags = slices.Clone(t.Tags)
	w.editDescription = t.Description
	w.editSubtasks = slices.Clone(t.Subtasks)
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

// taskInputFromTask snapshots a committed task's content as a save input.
func taskInputFromTask(t model.Task) projects.TaskInput {
	return projects.TaskInput{
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    t.Priority,
		Tags:        slices.Clone(t.Tags),
		Subtasks:    slices.Clone(t.Subtasks),
		DueDate:     t.DueDate,
	}
}

// overlayTaskInput projects a save input onto a task's content fields.
func overlayTaskInput(t model.Task, in projects.TaskInput) model.Task {
	t.Title, t.Description, t.Status, t.Priority = in.Title, in.Description, in.Status, in.Priority
	t.Tags, t.Subtasks, t.DueDate = in.Tags, in.Subtasks, in.DueDate
	return t
}

// rebaseTaskInputOnCommitted replaces fields the queued draft still shares with the
// failed discard baseline (snap) with the committed task's values, so a write that
// rolled back cannot reach the database through the follow-up save. Fields edited
// after the discard differ from snap and survive.
func rebaseTaskInputOnCommitted(in, snap, committed projects.TaskInput) projects.TaskInput {
	if in.Title == snap.Title {
		in.Title = committed.Title
	}
	if in.Description == snap.Description {
		in.Description = committed.Description
	}
	if in.Status == snap.Status {
		in.Status = committed.Status
	}
	if in.Priority == snap.Priority {
		in.Priority = committed.Priority
	}
	if in.DueDate == snap.DueDate {
		in.DueDate = committed.DueDate
	}
	in.Tags = rebaseTagsOnCommitted(in.Tags, snap.Tags, committed.Tags)
	in.Subtasks = rebaseSubtasksOnCommitted(in.Subtasks, snap.Subtasks, committed.Subtasks)
	return in
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

// rebaseSubtasksOnCommitted merges post-discard subtask edits by stable ID: added or
// modified subtasks survive, post-discard removals drop the committed copies, and
// untouched ones fall back to the committed version.
func rebaseSubtasksOnCommitted(in, snap, committed []model.Subtask) []model.Subtask {
	inByID := make(map[string]model.Subtask, len(in))
	for _, st := range in {
		inByID[st.ID] = st
	}
	snapByID := make(map[string]model.Subtask, len(snap))
	for _, st := range snap {
		snapByID[st.ID] = st
	}
	out := make([]model.Subtask, 0, len(committed)+len(in))
	for _, st := range committed {
		s, wasInSnap := snapByID[st.ID]
		cur, inDraft := inByID[st.ID]
		switch {
		case wasInSnap && !inDraft:
			// Removed after the discard.
		case inDraft && cur != s:
			out = append(out, cur)
		default:
			out = append(out, st)
		}
	}
	for _, st := range in {
		if _, wasInSnap := snapByID[st.ID]; !wasInSnap && st.ID != "" {
			if !slices.ContainsFunc(out, func(o model.Subtask) bool { return o.ID == st.ID }) {
				out = append(out, st)
			}
		}
	}
	return out
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

// rebaseInvalidDraftOnCommitted resets the fields the current draft still shares with
// the failed discard baseline (snap) to the committed values. The draft itself stays
// untouched in its invalid state: the user's newer input and the validation error
// must survive until a valid save can be built from them.
func (w *Workbench) rebaseInvalidDraftOnCommitted(snap projects.TaskInput) {
	if w.detailTask == nil {
		return
	}
	rebased := rebaseTaskInputOnCommitted(w.taskInputFromDraft(), snap, taskInputFromTask(*w.detailTask))
	w.copyDetailEditFields(overlayTaskInput(*w.detailTask, rebased))
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
	updated, err := w.app.Projects.UpdateTask(context.Background(), taskID, projects.TaskInput{
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    priority,
		Tags:        slices.Clone(t.Tags),
		Subtasks:    slices.Clone(t.Subtasks),
		DueDate:     t.DueDate,
	})
	if err != nil {
		w.enqueueErrorToast("更新优先级失败", err)
		return
	}
	w.boardTasks[idx] = updated
	if w.selectedTaskID == updated.ID && !w.dirty && !w.hasUnsavedChangesFromDetailTask() {
		w.syncDetailFromTask(updated)
	}
	w.rebuildBoardDerived()
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
	ctx := context.Background()
	if _, err := w.app.Projects.MoveTask(ctx, taskID, targetStatus, unfilteredIndex); err != nil {
		w.enqueueErrorToast("移动任务失败", err)
		return
	}
	w.ReloadBoardTasks()
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
		ctx := context.Background()
		if err := w.app.Projects.ReorderTasks(ctx, w.activeProjectID, targetStatus, ids); err != nil {
			w.enqueueErrorToast("排序失败", err)
			return
		}
		w.ReloadBoardTasks()
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
	ctx := context.Background()
	if err := w.app.Projects.ReorderTasks(ctx, w.activeProjectID, status, ids); err != nil {
		w.enqueueErrorToast("调整顺序失败", err)
		return
	}
	w.ReloadBoardTasks()
}

// OpenCreateProjectModal opens the modal to create a new project.
func (w *Workbench) OpenCreateProjectModal() {
	if w.IsRestoring() {
		return
	}
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
	ctx := context.Background()
	input := projects.ProjectInput{
		Name:        name,
		Description: strings.TrimSpace(w.projectModalDesc),
		Icon:        icon,
	}
	if w.projectModalEditID == "" {
		if err := w.FlushPending(); err != nil {
			w.projectModalErr = "当前打开的任务存在未保存修改（" + w.flushErrorDetail(err) + "），请先修正或放弃修改"
			return false
		}
		created, err := w.app.Projects.CreateProject(ctx, input)
		if err != nil {
			w.projectModalErr = err.Error()
			return false
		}
		w.projectModalOpen = false
		w.ReloadProjects()
		w.SelectProject(created.ID)
		w.EnqueueToast(ToastProps{
			Title:       "已创建项目",
			Description: created.Name,
			Type:        ToastSuccess,
		})
		return true
	}

	updated, err := w.app.Projects.UpdateProject(ctx, w.projectModalEditID, input)
	if err != nil {
		w.projectModalErr = err.Error()
		return false
	}
	w.projectModalOpen = false
	w.ReloadProjects()
	w.EnqueueToast(ToastProps{
		Title:       "已更新项目",
		Description: updated.Name,
		Type:        ToastSuccess,
	})
	return true
}

// ArchiveProject archives an active project.
func (w *Workbench) ArchiveProject(projectID string) {
	if w.IsRestoring() {
		return
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再归档项目"); err != nil {
		return
	}
	ctx := context.Background()
	p, err := w.app.Projects.ArchiveProject(ctx, projectID, true)
	if err != nil {
		w.enqueueErrorToast("归档项目失败", err)
		return
	}
	w.ReloadProjects()
	w.EnqueueToast(ToastProps{
		Title:       "项目已归档",
		Description: p.Name,
		Type:        ToastInfo,
	})
}

// UnarchiveProject restores an archived project back to the active project list.
func (w *Workbench) UnarchiveProject(projectID string) {
	if w.IsRestoring() {
		return
	}
	ctx := context.Background()
	p, err := w.app.Projects.ArchiveProject(ctx, projectID, false)
	if err != nil {
		w.enqueueErrorToast("恢复归档失败", err)
		return
	}
	w.ReloadProjects()
	w.EnqueueToast(ToastProps{
		Title:       "已恢复归档项目",
		Description: p.Name,
		Type:        ToastSuccess,
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
	ctx := context.Background()
	if err := w.app.Projects.DeleteProject(ctx, projectID); err != nil {
		w.enqueueErrorToast("移入回收站失败", err)
		return
	}
	if w.activeProjectID == projectID {
		w.resetSelectedTaskState()
		w.activeProjectID = ""
	}
	w.ReloadProjects()
	w.ReloadTrash()
	if w.activeProjectID == "" {
		if len(w.activeProjects) > 0 {
			w.SelectProject(w.activeProjects[0].ID)
		} else {
			w.navPage = NavAllProjects
		}
	}
	w.EnqueueToast(ToastProps{
		Title:       "项目已移入回收站",
		Description: "可在回收站中随时恢复或彻底删除",
		Type:        ToastInfo,
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

	var tags []string
	for _, part := range strings.FieldsFunc(w.taskModalTags, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	}) {
		if t := strings.TrimSpace(part); t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}

	dueStr := ""
	if !w.taskModalDueDate.IsZero() {
		dueStr = w.taskModalDueDate.Format("2006-01-02")
	}

	ctx := context.Background()
	created, err := w.app.Projects.CreateTask(ctx, w.activeProjectID, projects.TaskInput{
		Title:       title,
		Description: w.taskModalDesc,
		Status:      model.Status(w.taskModalStatus),
		Priority:    model.Priority(w.taskModalPriority),
		Tags:        tags,
		DueDate:     dueStr,
	})
	if err != nil {
		w.taskModalErr = err.Error()
		return false
	}

	w.taskModalOpen = false
	w.ReloadBoardTasks()
	w.EnqueueToast(ToastProps{
		Title:       "已创建任务",
		Description: created.Title,
		Type:        ToastSuccess,
	})
	return true
}

// QuickCreateTask creates a task directly in the specified column from its quick input.
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
	ctx := context.Background()
	created, err := w.app.Projects.CreateTask(ctx, w.activeProjectID, projects.TaskInput{
		Title:    title,
		Status:   status,
		Priority: model.NoPriority,
	})
	if err != nil {
		w.enqueueErrorToast("创建任务失败", err)
		return nil, err
	}
	w.quickTaskTitles[status] = ""
	w.ReloadBoardTasks()
	return &created, nil
}

// SoftDeleteTask moves a task to the Recycle Bin.
func (w *Workbench) SoftDeleteTask(taskID string) {
	if w.IsRestoring() || taskID == "" {
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
	ctx := context.Background()
	if err := w.app.Projects.DeleteTask(ctx, taskID); err != nil {
		w.enqueueErrorToast("移入回收站失败", err)
		return
	}
	if w.selectedTaskID == taskID {
		w.resetSelectedTaskState()
	}
	w.ReloadBoardTasks()
	w.ReloadTrash()
	w.EnqueueToast(ToastProps{
		Title:       "任务已移入回收站",
		Description: "可在回收站中恢复或永久删除",
		Type:        ToastInfo,
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
		w.ReloadBoardTasks()
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
			w.ReloadBoardTasks()
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

func (w *Workbench) exportAttachmentWithContext(ctx context.Context, att model.Attachment, destPath string) error {
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return nil
	}
	return w.app.Attachments.ExportOverwrite(ctx, att.ID, destPath)
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

// ExportAttachment exports a managed attachment to destPath using backend-validated atomic staging
// so any existing user regular file is preserved on failure and protected store paths are rejected.
func (w *Workbench) ExportAttachment(att model.Attachment, destPath string) error {
	if w.IsRestoring() {
		return fmt.Errorf("%w: backup restore in progress", model.ErrConflict)
	}
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return nil
	}
	err := w.exportAttachmentWithContext(context.Background(), att, destPath)
	w.applyExportAttachmentResult(destPath, err)
	return err
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
		exportErr := w.exportAttachmentWithContext(ctx, att, dest)
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
	ctx := context.Background()
	err := w.app.RestoreTrash(ctx, item.Kind, item.ID)
	if err != nil {
		if errors.Is(err, model.ErrParentDeleted) {
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
		w.EnqueueToast(ToastProps{
			Title:       "恢复失败",
			Description: err.Error(),
			Type:        ToastError,
		})
		return
	}
	w.ReloadProjects()
	w.ReloadJobApplications()
	w.ReloadTrash()
	w.EnqueueToast(ToastProps{
		Title:       "已从回收站恢复",
		Description: item.Name,
		Type:        ToastSuccess,
	})
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
	ctx := context.Background()
	taskItem := *w.trashOrphanTask
	if taskItem.ProjectID != "" {
		if err := w.app.Projects.RestoreTrash(ctx, projects.TrashProject, taskItem.ProjectID); err != nil {
			w.enqueueErrorToast("恢复所属项目失败", err)
			return
		}
	}
	// Independently deleted tasks require explicit restoration after restoring the parent project.
	var taskErr error
	if taskItem.ID != "" {
		taskErr = w.app.Projects.RestoreTrash(ctx, projects.TrashTask, taskItem.ID)
	}
	w.trashParentErrModalOpen = false
	w.trashOrphanTask = nil
	w.trashOrphanParent = nil
	w.ReloadProjects()
	w.ReloadTrash()
	if taskErr != nil {
		w.enqueueErrorToast("恢复任务失败（所属项目已恢复）", taskErr)
		return
	}
	w.EnqueueToast(ToastProps{
		Title:       "已恢复项目与任务",
		Description: taskItem.Name,
		Type:        ToastSuccess,
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
			w.ReloadProjects()
			w.ReloadJobApplications()
			w.ReloadTrash()
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
			w.ReloadProjects()
			w.ReloadJobApplications()
			w.ReloadTrash()
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
			if themeErr == nil && w.themeSetting == initialTheme {
				w.themeSetting = th
			}
			w.themeSegmentIdx = themeSegmentForSetting(w.themeSetting)
			if policyErr == nil && w.backupPolicyDraft() == initialPolicy {
				w.backupEnabled = pol.Enabled
				w.backupIntervalSelect = formatIntervalOption(pol.Interval)
				w.backupKeepSelect = fmt.Sprint(pol.Keep)
				w.backupDirInput = pol.Directory
			}
			if statusErr == nil {
				w.backupStatus = st
			}
		}
	})
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
	ctx := context.Background()
	if err := w.app.Settings.SetTheme(ctx, th); err != nil {
		w.themeSegmentIdx = themeSegmentForSetting(w.themeSetting)
		w.enqueueErrorToast("保存主题设置失败", err)
		return
	}
	w.themeSetting = th
	w.themeSegmentIdx = themeSegmentForSetting(th)
	if w.host.SetTheme != nil {
		w.host.SetTheme(th)
	}
}

// SaveBackupPolicy persists the current backup policy controls to SQLite and reconfigures the scheduler.
func (w *Workbench) SaveBackupPolicy() {
	if w.IsRestoring() {
		return
	}
	ctx := context.Background()
	keep := 7
	if rawKeep := strings.TrimSpace(w.backupKeepSelect); rawKeep != "" {
		if parsed, err := strconv.Atoi(rawKeep); err == nil {
			keep = parsed
		}
	}
	pol := backup.Policy{
		Enabled:   w.backupEnabled,
		Interval:  parseIntervalOption(w.backupIntervalSelect),
		Keep:      keep,
		Directory: strings.TrimSpace(w.backupDirInput),
	}
	if err := w.app.Backups.SetPolicy(ctx, pol); err != nil {
		w.enqueueErrorToast("保存自动备份策略失败", err)
		return
	}
	w.ReloadSettings()
	w.EnqueueToast(ToastProps{
		Title:       "自动备份策略已保存",
		Description: "本地自动备份调度器已更新",
		Type:        ToastSuccess,
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

func (w *Workbench) exportManualBackupWithContext(ctx context.Context, destPath string) error {
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return nil
	}
	return w.app.Backups.ExportOverwrite(ctx, destPath)
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

// ExportManualBackup exports a full backup ZIP to destPath using backend-validated atomic staging
// so any existing user regular file at destPath is preserved on failure and protected store paths are rejected.
func (w *Workbench) ExportManualBackup(destPath string) error {
	if w.IsRestoring() {
		return fmt.Errorf("%w: backup restore in progress", model.ErrConflict)
	}
	if err := w.flushPendingOrToast("当前任务存在未保存修改", "，请先修正保存或放弃修改后再导出备份"); err != nil {
		return err
	}
	destPath = strings.TrimSpace(destPath)
	if destPath == "" {
		return nil
	}
	err := w.exportManualBackupWithContext(context.Background(), destPath)
	w.applyExportBackupResult(destPath, err)
	return err
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
		exportErr := w.exportManualBackupWithContext(ctx, dest)
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
	if w.saving || w.jobSaving || len(w.jobWrites) > 0 {
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
