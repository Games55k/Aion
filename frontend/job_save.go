package frontend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"
)

var errJobSavePending = errors.New("投递记录正在后台保存，请稍候")

type jobSave struct {
	jobID           string
	revision, epoch uint64
	input           jobapplications.UpdateDetailsInput
}

func (w *Workbench) jobSelected() bool {
	return w.selectedJobID != "" && w.detailJob != nil
}

func formatCalendarDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02")
}

func parseCalendarDateValue(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}
	}
	t, err := model.ParseCalendarDate(trimmed)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (w *Workbench) resetSelectedJobState() {
	w.selectedJobID = ""
	w.detailJob = nil
	w.detailJobEpoch = 0
	w.detailJobEvents = nil
	w.jobDetailOpen = false
	w.editJobCompany = ""
	w.editJobPosition = ""
	w.editJobAppliedDate = time.Time{}
	w.editJobURL = ""
	w.editJobChannel = ""
	w.editJobFollowUpDate = time.Time{}
	w.editJobFollowUpDirty = false
	w.editJobNotes = ""
	w.editJobClosureReason = ""
	w.editJobReasonDirty = false
	w.jobSaveState = "saved"
	w.jobSaveErr = ""
	w.jobDirty = false
	w.jobSaving = false
	w.pendingJobSave = nil
	w.discardJobAfterSave = false
	w.jobNoteInput = ""
	w.jobNoteDate = time.Time{}
	w.jobNoteErr = ""
	w.jobNoteOpID = ""
	w.jobBackfillStage = string(model.JobStageInterview1)
	w.jobBackfillDate = time.Time{}
	w.jobBackfillNote = ""
	w.jobBackfillErr = ""
	w.jobBackfillOpID = ""
}

func (w *Workbench) syncDetailFromJob(app model.JobApplication) {
	copyApp := app
	w.selectedJobID = app.ID
	w.detailJob = &copyApp
	w.detailJobEpoch = w.currentDataEpoch()
	w.editJobCompany = app.CompanyName
	w.editJobPosition = app.PositionName
	w.editJobAppliedDate = parseCalendarDateValue(app.AppliedDate)
	w.editJobURL = app.ApplicationURL
	w.editJobChannel = app.Channel
	w.editJobFollowUpDate = parseCalendarDateValue(app.NextFollowUpDate)
	w.editJobFollowUpDirty = false
	w.editJobNotes = app.Notes
	w.editJobClosureReason = app.ClosureReason
	w.editJobReasonDirty = false
	w.jobDirty = false
	w.jobSaveState = "saved"
	w.jobSaveErr = ""
	w.discardJobAfterSave = false
}

func (w *Workbench) markJobDirty() {
	w.jobDirty = true
	w.jobSaveRevision++
	w.discardJobAfterSave = false
}

// TouchJobDetailField marks the job detail draft modified, clears pending discard, and triggers auto-save.
func (w *Workbench) TouchJobDetailField() {
	if !w.jobSelected() {
		return
	}
	w.markJobDirty()
	_ = w.SaveSelectedJob()
}

func (w *Workbench) hasUnsavedChangesFromDetailJob() bool {
	if w.detailJob == nil {
		return false
	}
	if strings.TrimSpace(w.editJobCompany) != w.detailJob.CompanyName ||
		strings.TrimSpace(w.editJobPosition) != w.detailJob.PositionName ||
		formatCalendarDate(w.editJobAppliedDate) != w.detailJob.AppliedDate ||
		strings.TrimSpace(w.editJobURL) != w.detailJob.ApplicationURL ||
		strings.TrimSpace(w.editJobChannel) != w.detailJob.Channel ||
		strings.TrimRight(w.editJobNotes, "\r\n") != w.detailJob.Notes {
		return true
	}
	if w.editJobFollowUpDirty && formatCalendarDate(w.editJobFollowUpDate) != w.detailJob.NextFollowUpDate {
		return true
	}
	if w.editJobReasonDirty && w.detailJob.Lifecycle == model.JobLifecycleClosed && strings.TrimSpace(w.editJobClosureReason) != w.detailJob.ClosureReason {
		return true
	}
	return false
}

func (w *Workbench) buildJobSaveRequest() (jobSave, error) {
	if w.selectedJobID == "" || w.detailJob == nil {
		return jobSave{}, nil
	}
	if w.IsRestoring() || w.detailJobEpoch != w.currentDataEpoch() {
		return jobSave{}, fmt.Errorf("%w: backup restore in progress", model.ErrConflict)
	}
	company := strings.TrimSpace(w.editJobCompany)
	if company == "" {
		return jobSave{}, fmt.Errorf("%w: 公司名称不能为空", model.ErrInvalid)
	}
	position := strings.TrimSpace(w.editJobPosition)
	if position == "" {
		return jobSave{}, fmt.Errorf("%w: 岗位名称不能为空", model.ErrInvalid)
	}
	appliedDate := formatCalendarDate(w.editJobAppliedDate)
	if appliedDate == "" {
		return jobSave{}, fmt.Errorf("%w: 投递日期不能为空", model.ErrInvalid)
	}
	if _, err := model.ParseCalendarDate(appliedDate); err != nil {
		return jobSave{}, err
	}
	appURL, err := model.ValidateJobURL(w.editJobURL)
	if err != nil {
		return jobSave{}, err
	}
	in := jobapplications.UpdateDetailsInput{
		ID:             w.selectedJobID,
		CompanyName:    company,
		PositionName:   position,
		AppliedDate:    appliedDate,
		ApplicationURL: appURL,
		Channel:        strings.TrimSpace(w.editJobChannel),
		Notes:          strings.TrimRight(w.editJobNotes, "\r\n"),
	}
	if w.editJobFollowUpDirty {
		f := formatCalendarDate(w.editJobFollowUpDate)
		in.NextFollowUpDate = &f
	}
	if w.editJobReasonDirty && w.detailJob.Lifecycle == model.JobLifecycleClosed {
		r := strings.TrimSpace(w.editJobClosureReason)
		in.ClosureReason = &r
	}
	return jobSave{
		jobID:    w.selectedJobID,
		revision: w.jobSaveRevision,
		epoch:    w.detailJobEpoch,
		input:    in,
	}, nil
}

// SaveSelectedJob validates the job detail draft and schedules serialized auto-save in the background.
func (w *Workbench) SaveSelectedJob() error {
	if !w.jobSelected() {
		return nil
	}
	w.discardJobAfterSave = false
	req, err := w.buildJobSaveRequest()
	if err != nil {
		w.jobDirty = true
		if errors.Is(err, model.ErrInvalid) {
			w.jobSaveState = "validation_error"
		} else {
			w.jobSaveState = "error"
		}
		w.jobSaveErr = err.Error()
		w.pendingJobSave = nil
		return err
	}
	if w.jobSaving {
		w.jobDirty = true
		w.pendingJobSave = &req
		w.jobSaveState = "saving"
		w.jobSaveErr = ""
		return nil
	}
	w.jobSaveState = "saving"
	w.jobSaveErr = ""
	return w.startJobSave(req)
}

// DiscardJobChanges reverts unsaved job detail edits back to the last persisted state.
func (w *Workbench) DiscardJobChanges() {
	if !w.jobSelected() {
		return
	}
	w.afterSave = nil
	w.pendingJobSave = nil
	w.jobSaveRevision++
	wasSaving := w.jobSaving
	if w.detailJob != nil {
		w.syncDetailFromJob(*w.detailJob)
	}
	if wasSaving {
		w.discardJobAfterSave = true
	}
}

func (w *Workbench) startJobSave(request jobSave) error {
	service := w.app.JobApplications
	w.jobSaving = true
	work := func() (model.JobApplication, error) {
		return service.UpdateDetails(context.Background(), request.input)
	}
	apply := func(updated model.JobApplication, err error) {
		w.jobSaving = false
		if request.epoch != w.currentDataEpoch() || request.jobID != w.selectedJobID {
			return
		}
		defer func() {
			if w.jobsNeedReload && !w.jobSaving && isJobNavPage(w.navPage) {
				w.reloadJobApplications(true)
			}
		}()
		if err == nil {
			w.loadVersions["jobs"]++
			w.detailJob = &updated
			w.jobLastSavedAt = w.now()
			w.upsertJobInCache(updated)
		}
		if w.discardJobAfterSave {
			w.discardJobAfterSave = false
			if w.detailJob != nil {
				w.syncDetailFromJob(*w.detailJob)
			}
			return
		}
		if next := w.pendingJobSave; next != nil {
			w.pendingJobSave = nil
			_ = w.startJobSave(*next)
			return
		}
		if request.revision != w.jobSaveRevision {
			w.afterSave = nil
			return
		}
		if err != nil {
			w.jobDirty = true
			w.jobSaveState = "error"
			w.jobSaveErr = err.Error()
			w.afterSave = nil
			return
		}
		w.jobDirty = false
		w.editJobFollowUpDirty = false
		w.editJobReasonDirty = false
		w.jobSaveState = "saved"
		w.jobSaveErr = ""
		if next := w.afterSave; next != nil {
			w.afterSave = nil
			next()
		}
	}
	if dispatch := w.host.Dispatch; dispatch != nil {
		w.taskWG.Add(1)
		go func() {
			defer w.taskWG.Done()
			updated, err := work()
			dispatch(func() { apply(updated, err) })
		}()
		return nil
	}
	updated, err := work()
	apply(updated, err)
	return err
}
