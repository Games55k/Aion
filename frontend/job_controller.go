package frontend

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
)

func (w *Workbench) runJobWorker(work func() error, onDone func(error)) {
	if w.IsRestoring() {
		return
	}
	epoch := w.currentDataEpoch()
	if dispatch := w.host.Dispatch; dispatch != nil {
		w.taskWG.Add(1)
		go func() {
			defer w.taskWG.Done()
			err := work()
			dispatch(func() {
				if epoch != w.currentDataEpoch() {
					return
				}
				onDone(err)
			})
		}()
		return
	}
	err := work()
	if epoch != w.currentDataEpoch() {
		return
	}
	onDone(err)
}

// ReloadJobApplications reloads all non-deleted job applications and aggregate stats.
func (w *Workbench) ReloadJobApplications() { w.reloadJobApplications(false) }

func (w *Workbench) reloadJobApplications(async bool) {
	service := w.app.JobApplications
	w.loadUI("jobs", async, func() func() {
		apps, err := service.List(context.Background())
		return func() {
			if err != nil {
				w.enqueueErrorToast("加载投递记录失败", err)
				return
			}
			w.jobsNeedReload = false
			w.jobApps = apps
			w.jobStats = jobapplications.ComputeStats(apps)
			w.rebuildJobDerived()
		}
	})
}

func (w *Workbench) activeJobView() jobapplications.JobView {
	switch w.navPage {
	case NavJobActive:
		return jobapplications.ViewActive
	case NavJobOffer:
		return jobapplications.ViewOffer
	case NavJobClosed:
		return jobapplications.ViewClosed
	default:
		return jobapplications.ViewAll
	}
}

func (w *Workbench) rebuildJobDerived() {
	filter := jobapplications.Filter{
		View:        w.activeJobView(),
		Query:       w.jobSearchQuery,
		Stage:       model.JobStage(w.jobStageFilter),
		Outcome:     model.JobOutcome(w.jobOutcomeFilter),
		AppliedDate: formatCalendarDate(w.jobAppliedDateFilter),
	}
	w.filteredJobApps = jobapplications.FilterApplications(w.jobApps, filter, w.jobSortBy)
	w.clampJobPage()
}

func (w *Workbench) pageSizeForJobs() int {
	if w.jobPageSize <= 0 {
		return 20
	}
	return w.jobPageSize
}

func (w *Workbench) clampJobPage() {
	size := w.pageSizeForJobs()
	total := len(w.filteredJobApps)
	if total == 0 {
		w.jobPage = 0
		return
	}
	maxPage := (total - 1) / size
	if w.jobPage < 0 {
		w.jobPage = 0
	}
	if w.jobPage > maxPage {
		w.jobPage = maxPage
	}
}

func (w *Workbench) pagedJobApps() ([]model.JobApplication, int, int) {
	w.clampJobPage()
	total := len(w.filteredJobApps)
	if total == 0 {
		return nil, 0, 1
	}
	size := w.pageSizeForJobs()
	totalPages := (total-1)/size + 1
	start := w.jobPage * size
	end := min(start+size, total)
	return w.filteredJobApps[start:end], w.jobPage, totalPages
}

func (w *Workbench) upsertJobInCache(app model.JobApplication) {
	idx := slices.IndexFunc(w.jobApps, func(a model.JobApplication) bool { return a.ID == app.ID })
	if idx >= 0 {
		w.jobApps[idx] = app
	} else {
		w.jobApps = append(w.jobApps, app)
	}
	w.jobStats = jobapplications.ComputeStats(w.jobApps)
	w.rebuildJobDerived()
}

func (w *Workbench) removeJobFromCache(appID string) {
	idx := slices.IndexFunc(w.jobApps, func(a model.JobApplication) bool { return a.ID == appID })
	if idx >= 0 {
		w.jobApps = slices.Delete(w.jobApps, idx, idx+1)
	}
	w.jobStats = jobapplications.ComputeStats(w.jobApps)
	w.rebuildJobDerived()
}

// SetJobPage navigates the job table pagination safely.
func (w *Workbench) SetJobPage(page int) {
	w.jobPage = page
	w.clampJobPage()
}

// ClearJobFilters resets search, stage, outcome, and applied date filters while keeping current sort and view.
func (w *Workbench) ClearJobFilters() {
	w.jobSearchQuery = ""
	w.jobStageFilter = ""
	w.jobOutcomeFilter = ""
	w.jobAppliedDateFilter = time.Time{}
	w.jobPage = 0
	w.rebuildJobDerived()
}

func (w *Workbench) activeDuplicateMatches(company, position string) []model.JobApplication {
	c := strings.TrimSpace(company)
	p := strings.TrimSpace(position)
	if c == "" || p == "" {
		return nil
	}
	var out []model.JobApplication
	for _, app := range w.jobApps {
		if app.DeletedAt != nil || app.Lifecycle != model.JobLifecycleActive {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(app.CompanyName), c) &&
			strings.EqualFold(strings.TrimSpace(app.PositionName), p) {
			out = append(out, app)
		}
	}
	return out
}

// OpenJobCreateModal opens the creation dialog with today's local date and Applied stage defaulted.
func (w *Workbench) OpenJobCreateModal() {
	if w.IsRestoring() {
		return
	}
	w.jobCreateCompany = ""
	w.jobCreatePosition = ""
	w.jobCreateAppliedDate = w.now().In(time.Local)
	w.jobCreateStage = string(model.JobStageApplied)
	w.jobCreateURL = ""
	w.jobCreateChannel = ""
	w.jobCreateFollowUpDate = time.Time{}
	w.jobCreateNotes = ""
	w.jobCreateErr = ""
	w.jobCreateOpen = true
}

// SubmitJobCreateModal validates required fields and inserts a new job application.
func (w *Workbench) SubmitJobCreateModal() {
	if w.IsRestoring() {
		return
	}
	company := strings.TrimSpace(w.jobCreateCompany)
	if company == "" {
		w.jobCreateErr = "公司名称不能为空"
		return
	}
	position := strings.TrimSpace(w.jobCreatePosition)
	if position == "" {
		w.jobCreateErr = "岗位名称不能为空"
		return
	}
	appliedDate := formatCalendarDate(w.jobCreateAppliedDate)
	if appliedDate == "" {
		w.jobCreateErr = "请选择投递日期"
		return
	}
	if _, err := model.ParseCalendarDate(appliedDate); err != nil {
		w.jobCreateErr = "投递日期格式无效"
		return
	}
	appURL, err := model.ValidateJobURL(w.jobCreateURL)
	if err != nil {
		w.jobCreateErr = err.Error()
		return
	}
	stage := model.JobStage(w.jobCreateStage)
	if stage == "" {
		stage = model.JobStageApplied
	}
	if !stage.Valid() {
		w.jobCreateErr = "请选择有效的当前阶段"
		return
	}
	in := jobapplications.CreateInput{
		CompanyName:      company,
		PositionName:     position,
		AppliedDate:      appliedDate,
		Stage:            stage,
		ApplicationURL:   appURL,
		Channel:          strings.TrimSpace(w.jobCreateChannel),
		NextFollowUpDate: formatCalendarDate(w.jobCreateFollowUpDate),
		Notes:            strings.TrimRight(w.jobCreateNotes, "\r\n"),
	}
	created, err := w.app.JobApplications.Create(context.Background(), in)
	if err != nil {
		w.jobCreateErr = err.Error()
		return
	}
	w.jobCreateOpen = false
	w.jobCreateErr = ""
	w.jobPage = 0
	w.upsertJobInCache(created)
	w.EnqueueToast(feedback.ToastProps{
		Title:       "已新增投递记录",
		Description: fmt.Sprintf("%s / %s", created.CompanyName, created.PositionName),
		Type:        feedback.ToastSuccess,
	})
}

// SelectJobApplication opens the Detail Modal for appID after flushing any pending task or job draft.
func (w *Workbench) SelectJobApplication(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	if w.selectedJobID == appID && w.detailJob != nil {
		w.jobDetailOpen = true
		return
	}
	if !w.afterPendingSave(func() { w.SelectJobApplication(appID) }, "当前投递存在未保存修改", "，请先修正保存或放弃修改后再打开其他记录") {
		return
	}
	service := w.app.JobApplications
	w.loadUI("job_selection", true, func() func() {
		app, err := service.Get(context.Background(), appID)
		var events []model.JobApplicationEvent
		if err == nil {
			events, err = service.ListEvents(context.Background(), appID)
		}
		return func() {
			if err != nil {
				w.enqueueErrorToast("无法打开投递详情", err)
				return
			}
			w.selectLoadedJob(app, events)
		}
	})
}

func (w *Workbench) selectLoadedJob(app model.JobApplication, events []model.JobApplicationEvent) {
	w.jobDetailOpen = true
	w.jobLastSavedAt = time.Time{}
	w.detailJobEvents = events
	w.jobNoteInput = ""
	w.jobNoteDate = w.now().In(time.Local)
	w.jobNoteErr = ""
	w.jobNoteOpID = ""
	w.jobBackfillStage = string(model.JobStageInterview1)
	w.jobBackfillDate = w.now().In(time.Local)
	w.jobBackfillNote = ""
	w.jobBackfillErr = ""
	w.jobBackfillOpID = ""
	w.syncDetailFromJob(app)
}

// CloseJobDetail flushes any pending job draft before closing the detail modal.
func (w *Workbench) CloseJobDetail() {
	if !w.afterPendingSave(func() { w.CloseJobDetail() }, "当前投递存在未保存修改", "，请先修正保存或放弃修改后再关闭详情") {
		return
	}
	w.navigationVersion++
	w.resetSelectedJobState()
}

// ChangeJobStage updates the application's stage atomically with retry deduplication and without clobbering unsaved draft text.
func (w *Workbench) ChangeJobStage(appID string, toStage model.JobStage) {
	if w.IsRestoring() || !toStage.Valid() {
		return
	}
	if w.pendingStageAppID != appID || w.pendingStageTarget != toStage || w.pendingStageOpID == "" {
		w.pendingStageAppID = appID
		w.pendingStageTarget = toStage
		w.pendingStageOpID = model.NewID()
	}
	opID := w.pendingStageOpID
	service := w.app.JobApplications
	var updated model.JobApplication
	var events []model.JobApplicationEvent
	w.runJobWorker(func() error {
		var err error
		updated, err = service.ChangeStage(context.Background(), jobapplications.ChangeStageInput{
			ID:      appID,
			ToStage: toStage,
			EventID: opID,
		})
		if err != nil {
			return err
		}
		events, err = service.ListEvents(context.Background(), appID)
		return err
	}, func(err error) {
		if err != nil {
			w.enqueueErrorToast("更新阶段失败", err)
			return
		}
		if w.pendingStageAppID == appID && w.pendingStageTarget == toStage && w.pendingStageOpID == opID {
			w.pendingStageAppID = ""
			w.pendingStageTarget = ""
			w.pendingStageOpID = ""
		}
		w.upsertJobInCache(updated)
		if w.selectedJobID == appID && w.detailJob != nil {
			w.detailJob.Stage = updated.Stage
			w.detailJob.ReachedOffer = updated.ReachedOffer
			w.detailJob.UpdatedAt = updated.UpdatedAt
			if events != nil {
				w.detailJobEvents = events
			}
		}
	})
}

// UpdateJobFollowUpDate updates next_follow_up_date inline without overwriting concurrent text drafts or appending timeline events.
func (w *Workbench) UpdateJobFollowUpDate(appID string, d time.Time) {
	if w.IsRestoring() {
		return
	}
	dateStr := formatCalendarDate(d)
	service := w.app.JobApplications
	var updated model.JobApplication
	w.runJobWorker(func() error {
		var err error
		updated, err = service.UpdateFollowUpDate(context.Background(), appID, dateStr)
		return err
	}, func(err error) {
		if err != nil {
			w.enqueueErrorToast("更新下次跟进日期失败", err)
			return
		}
		w.upsertJobInCache(updated)
		if w.selectedJobID == appID && w.detailJob != nil {
			w.detailJob.NextFollowUpDate = updated.NextFollowUpDate
			w.detailJob.UpdatedAt = updated.UpdatedAt
			if !w.editJobFollowUpDirty {
				w.editJobFollowUpDate = parseCalendarDateValue(updated.NextFollowUpDate)
			}
		}
	})
}

// OpenJobCloseModal prepares the closure workflow modal for appID.
func (w *Workbench) OpenJobCloseModal(appID string) {
	if w.IsRestoring() {
		return
	}
	if !w.afterPendingSave(func() { w.OpenJobCloseModal(appID) }, "当前投递存在未保存修改", "，请先修正保存或放弃修改后再结束流程") {
		return
	}
	w.jobCloseAppID = appID
	w.jobCloseOutcome = string(model.JobOutcomeRejected)
	w.jobCloseReason = ""
	w.jobCloseConfirmOffer = false
	w.jobCloseErr = ""
	w.jobCloseOpID = model.NewID()
	w.jobCloseOpen = true
}

// SubmitJobCloseModal validates outcome & Offer-stage rule and atomically closes the application.
func (w *Workbench) SubmitJobCloseModal() {
	if w.IsRestoring() || w.jobCloseAppID == "" {
		return
	}
	outcome := model.JobOutcome(w.jobCloseOutcome)
	if !outcome.Valid() {
		w.jobCloseErr = "请选择有效的结束结果"
		return
	}
	var currentStage model.JobStage
	if w.selectedJobID == w.jobCloseAppID && w.detailJob != nil {
		currentStage = w.detailJob.Stage
	} else {
		for _, a := range w.jobApps {
			if a.ID == w.jobCloseAppID {
				currentStage = a.Stage
				break
			}
		}
	}
	var stageOverride model.JobStage
	if outcome.RequiresOfferStage() && currentStage != model.JobStageOffer {
		if !w.jobCloseConfirmOffer {
			w.jobCloseErr = fmt.Sprintf("选择「%s」要求阶段为 Offer，请勾选确认同步调整阶段为 Offer", outcome.Label())
			return
		}
		stageOverride = model.JobStageOffer
	}
	if w.jobCloseOpID == "" {
		w.jobCloseOpID = model.NewID()
	}
	opID := w.jobCloseOpID
	reason := strings.TrimSpace(w.jobCloseReason)
	appID := w.jobCloseAppID
	service := w.app.JobApplications
	var updated model.JobApplication
	var events []model.JobApplicationEvent
	w.runJobWorker(func() error {
		var err error
		updated, err = service.Close(context.Background(), jobapplications.CloseInput{
			ID:            appID,
			Outcome:       outcome,
			Reason:        reason,
			StageOverride: stageOverride,
			EventID:       opID,
		})
		if err != nil {
			return err
		}
		events, err = service.ListEvents(context.Background(), appID)
		return err
	}, func(err error) {
		if err != nil {
			w.jobCloseErr = err.Error()
			return
		}
		w.jobCloseOpen = false
		w.jobCloseErr = ""
		w.jobCloseOpID = ""
		w.upsertJobInCache(updated)
		if w.selectedJobID == appID && w.detailJob != nil {
			w.detailJob.Stage = updated.Stage
			w.detailJob.Lifecycle = updated.Lifecycle
			w.detailJob.Outcome = updated.Outcome
			w.detailJob.ClosureReason = updated.ClosureReason
			w.detailJob.ClosedAt = updated.ClosedAt
			w.detailJob.ReachedOffer = updated.ReachedOffer
			w.detailJob.UpdatedAt = updated.UpdatedAt
			w.editJobClosureReason = updated.ClosureReason
			w.editJobReasonDirty = false
			if events != nil {
				w.detailJobEvents = events
			}
		}
		w.EnqueueToast(feedback.ToastProps{
			Title:       "流程已结束",
			Description: fmt.Sprintf("%s · %s", updated.Stage.Label(), updated.Outcome.Label()),
			Type:        feedback.ToastInfo,
		})
	})
}

// ReopenJobApplication explicitly reopens a closed application back to active while preserving historical closure events.
func (w *Workbench) ReopenJobApplication(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	if !w.afterPendingSave(func() { w.ReopenJobApplication(appID) }, "当前投递存在未保存修改", "，请先修正保存或放弃修改后再重新开启") {
		return
	}
	if w.jobReopenOpID == "" {
		w.jobReopenOpID = model.NewID()
	}
	opID := w.jobReopenOpID
	note := strings.TrimSpace(w.jobReopenNote)
	service := w.app.JobApplications
	var updated model.JobApplication
	var events []model.JobApplicationEvent
	w.runJobWorker(func() error {
		var err error
		updated, err = service.Reopen(context.Background(), jobapplications.ReopenInput{
			ID:      appID,
			Note:    note,
			EventID: opID,
		})
		if err != nil {
			return err
		}
		events, err = service.ListEvents(context.Background(), appID)
		return err
	}, func(err error) {
		if err != nil {
			w.jobReopenErr = err.Error()
			w.enqueueErrorToast("重新开启失败", err)
			return
		}
		w.jobReopenNote = ""
		w.jobReopenOpID = ""
		w.jobReopenErr = ""
		w.upsertJobInCache(updated)
		if w.selectedJobID == appID && w.detailJob != nil {
			w.detailJob.Lifecycle = updated.Lifecycle
			w.detailJob.Outcome = updated.Outcome
			w.detailJob.ClosureReason = updated.ClosureReason
			w.detailJob.ClosedAt = updated.ClosedAt
			w.detailJob.UpdatedAt = updated.UpdatedAt
			w.editJobClosureReason = ""
			w.editJobReasonDirty = false
			if events != nil {
				w.detailJobEvents = events
			}
		}
		w.EnqueueToast(feedback.ToastProps{
			Title:       "已重新开启流程",
			Description: updated.CompanyName + " / " + updated.PositionName,
			Type:        feedback.ToastSuccess,
		})
	})
}

// AddJobTimelineNote appends a manual note event onto the selected job's timeline.
func (w *Workbench) AddJobTimelineNote(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	note := strings.TrimSpace(w.jobNoteInput)
	if note == "" {
		w.jobNoteErr = "备注内容不能为空"
		return
	}
	if w.jobNoteOpID == "" {
		w.jobNoteOpID = model.NewID()
	}
	opID := w.jobNoteOpID
	dateStr := formatCalendarDate(w.jobNoteDate)
	service := w.app.JobApplications
	var events []model.JobApplicationEvent
	var updated model.JobApplication
	w.runJobWorker(func() error {
		if _, err := service.AddNoteEvent(context.Background(), jobapplications.AddNoteInput{
			ApplicationID: appID,
			Note:          note,
			OccurredDate:  dateStr,
			EventID:       opID,
		}); err != nil {
			return err
		}
		var err error
		updated, err = service.Get(context.Background(), appID)
		if err != nil {
			return err
		}
		events, err = service.ListEvents(context.Background(), appID)
		return err
	}, func(err error) {
		if err != nil {
			w.jobNoteErr = err.Error()
			return
		}
		w.jobNoteInput = ""
		w.jobNoteErr = ""
		w.jobNoteOpID = ""
		w.upsertJobInCache(updated)
		if w.selectedJobID == appID && w.detailJob != nil {
			w.detailJob.UpdatedAt = updated.UpdatedAt
			w.detailJobEvents = events
		}
	})
}

// BackfillJobStageEvent records a past-date stage event without altering current stage or lifecycle.
func (w *Workbench) BackfillJobStageEvent(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	stage := model.JobStage(w.jobBackfillStage)
	if !stage.Valid() {
		w.jobBackfillErr = "请选择有效的补录阶段"
		return
	}
	dateStr := formatCalendarDate(w.jobBackfillDate)
	if dateStr == "" {
		w.jobBackfillErr = "请选择补录日期"
		return
	}
	if _, err := model.ParseCalendarDate(dateStr); err != nil {
		w.jobBackfillErr = "补录日期格式无效"
		return
	}
	if w.jobBackfillOpID == "" {
		w.jobBackfillOpID = model.NewID()
	}
	opID := w.jobBackfillOpID
	note := strings.TrimSpace(w.jobBackfillNote)
	service := w.app.JobApplications
	var updated model.JobApplication
	var events []model.JobApplicationEvent
	w.runJobWorker(func() error {
		if _, err := service.BackfillStageEvent(context.Background(), jobapplications.BackfillStageInput{
			ApplicationID: appID,
			Stage:         stage,
			OccurredDate:  dateStr,
			Note:          note,
			EventID:       opID,
		}); err != nil {
			return err
		}
		var err error
		updated, err = service.Get(context.Background(), appID)
		if err != nil {
			return err
		}
		events, err = service.ListEvents(context.Background(), appID)
		return err
	}, func(err error) {
		if err != nil {
			w.jobBackfillErr = err.Error()
			return
		}
		w.jobBackfillNote = ""
		w.jobBackfillErr = ""
		w.jobBackfillOpID = ""
		w.upsertJobInCache(updated)
		if w.selectedJobID == appID && w.detailJob != nil {
			w.detailJob.ReachedOffer = updated.ReachedOffer
			w.detailJobEvents = events
		}
	})
}

// DeleteJobApplication soft-deletes an application (moving it and its preserved events to the global Recycle Bin).
func (w *Workbench) DeleteJobApplication(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	if w.jobSaving && w.selectedJobID == appID {
		w.EnqueueToast(feedback.ToastProps{Title: "正在保存投递记录", Description: "保存完成后再移入回收站", Type: feedback.ToastInfo})
		return
	}
	if w.selectedJobID == appID && (w.jobDirty || w.hasUnsavedChangesFromDetailJob()) {
		// Explicit delete supersedes unsaved text edits cleanly instead of trapping the user on validation errors.
		w.pendingJobSave = nil
		w.afterSave = nil
	}
	service := w.app.JobApplications
	w.runJobWorker(func() error {
		return service.Delete(context.Background(), appID)
	}, func(err error) {
		if err != nil {
			w.enqueueErrorToast("移入回收站失败", err)
			return
		}
		if w.selectedJobID == appID {
			w.resetSelectedJobState()
		}
		w.removeJobFromCache(appID)
		w.ReloadTrash()
		w.EnqueueToast(feedback.ToastProps{
			Title:       "已移入回收站",
			Description: "可在全局回收站中恢复或彻底删除",
			Type:        feedback.ToastInfo,
		})
	})
}

// OpenJobURL validates HTTP/HTTPS and delegates opening to Host.OpenURL with failure feedback.
func (w *Workbench) OpenJobURL(rawURL string) {
	cleaned, err := model.ValidateJobURL(rawURL)
	if err != nil || cleaned == "" {
		desc := "请填写有效的 http:// 或 https:// 投递链接"
		if err != nil {
			desc = err.Error()
		}
		w.EnqueueToast(feedback.ToastProps{
			Title:       "投递链接无效",
			Description: desc,
			Type:        feedback.ToastWarning,
		})
		return
	}
	if w.host.OpenURL == nil {
		w.EnqueueToast(feedback.ToastProps{
			Title:       "无法打开链接",
			Description: "当前窗口未提供浏览器打开能力",
			Type:        feedback.ToastError,
		})
		return
	}
	if err := w.host.OpenURL(cleaned); err != nil {
		w.EnqueueToast(feedback.ToastProps{
			Title:       "打开投递链接失败",
			Description: err.Error(),
			Type:        feedback.ToastError,
		})
	}
}
