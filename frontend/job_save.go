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
	w.editJobNotes = ""
	w.editJobClosureReason = ""
	w.editJobReasonDirty = false
	w.jobSaveState = "saved"
	w.jobSaveErr = ""
	w.jobDirty = false
	w.jobSaving = false
	w.pendingJobSave = nil
	w.inFlightJobSave = nil
	w.discardJobAfterSave = false
	w.discardJobRebase = false
	w.jobNoteInput = ""
	w.jobNoteDate = time.Time{}
	w.jobNoteErr = ""
	w.jobNoteRequest = jobapplications.AddNoteInput{}
	w.jobBackfillStage = string(model.JobStageInterview1)
	w.jobBackfillDate = time.Time{}
	w.jobBackfillNote = ""
	w.jobBackfillErr = ""
	w.jobBackfillRequest = jobapplications.BackfillStageInput{}
	w.jobBackfillExpanded = false
	w.editingJobEventID = ""
	w.editingJobEventNote = ""
}

func (w *Workbench) syncDetailFromJob(app model.JobApplication) {
	copyApp := app
	w.selectedJobID = app.ID
	w.detailJob = &copyApp
	w.detailJobEpoch = w.currentDataEpoch()
	w.resetJobEditFields(app)
	w.discardJobAfterSave = false
}

// resetJobEditFields copies an application's details into the draft inputs without
// replacing detailJob, so the last committed record stays the failure baseline.
func (w *Workbench) resetJobEditFields(app model.JobApplication) {
	w.editJobCompany = app.CompanyName
	w.editJobPosition = app.PositionName
	w.editJobAppliedDate = parseCalendarDateValue(app.AppliedDate)
	w.editJobURL = app.ApplicationURL
	w.editJobChannel = app.Channel
	w.editJobNotes = app.Notes
	w.editJobClosureReason = app.ClosureReason
	w.editJobReasonDirty = false
	w.jobDirty = false
	w.jobSaveState = "saved"
	w.jobSaveErr = ""
}

// rebaseJobDraftOnCommitted rolls back fields inherited from the failed write
// while preserving newer input and any validation error.
func (w *Workbench) rebaseJobDraftOnCommitted(snap jobapplications.UpdateDetailsInput) {
	if w.detailJob == nil {
		return
	}
	committed := *w.detailJob
	draft := w.jobInputFromDraft()
	if draft.CompanyName == snap.CompanyName && draft.CompanyName != committed.CompanyName {
		w.editJobCompany = committed.CompanyName
	}
	if draft.PositionName == snap.PositionName && draft.PositionName != committed.PositionName {
		w.editJobPosition = committed.PositionName
	}
	if draft.AppliedDate == snap.AppliedDate && draft.AppliedDate != committed.AppliedDate {
		w.editJobAppliedDate = parseCalendarDateValue(committed.AppliedDate)
	}
	if draft.ApplicationURL == snap.ApplicationURL && draft.ApplicationURL != committed.ApplicationURL {
		w.editJobURL = committed.ApplicationURL
	}
	if draft.Channel == snap.Channel && draft.Channel != committed.Channel {
		w.editJobChannel = committed.Channel
	}
	if draft.Notes == snap.Notes && draft.Notes != committed.Notes {
		// The request trimmed trailing line breaks typed after the discard.
		w.editJobNotes = committed.Notes + w.editJobNotes[len(draft.Notes):]
	}
	// A reason without a pending edit is untouched, and a reason equal to the failed
	// baseline counts as untouched by the same rule as every other field; both fall
	// back together with the dirty marker.
	if !w.editJobReasonDirty || jobClosureReasonEqual(draft.ClosureReason, snap.ClosureReason) {
		if strings.TrimSpace(w.editJobClosureReason) != committed.ClosureReason {
			w.editJobClosureReason = committed.ClosureReason
		}
		w.editJobReasonDirty = false
	}
}

// jobInputFromDraft snapshots normalized draft inputs without validating them.
func (w *Workbench) jobInputFromDraft() jobapplications.UpdateDetailsInput {
	in := jobapplications.UpdateDetailsInput{
		ID:             w.selectedJobID,
		CompanyName:    strings.TrimSpace(w.editJobCompany),
		PositionName:   strings.TrimSpace(w.editJobPosition),
		AppliedDate:    formatCalendarDate(w.editJobAppliedDate),
		ApplicationURL: strings.TrimSpace(w.editJobURL),
		Channel:        strings.TrimSpace(w.editJobChannel),
		Notes:          strings.TrimRight(w.editJobNotes, "\r\n"),
	}
	if w.editJobReasonDirty && w.detailJob != nil && w.detailJob.Lifecycle == model.JobLifecycleClosed {
		r := strings.TrimSpace(w.editJobClosureReason)
		in.ClosureReason = &r
	}
	return in
}

// inFlightJobBaseline predicts the selected application as it will read once the active
// background write lands. Discard must rebase on that result, because the write is
// already committing and cannot be unsaved.
func (w *Workbench) inFlightJobBaseline() (model.JobApplication, bool) {
	if w.detailJob == nil || w.inFlightJobSave == nil || w.inFlightJobSave.jobID != w.selectedJobID {
		return model.JobApplication{}, false
	}
	return overlayJobInput(*w.detailJob, w.inFlightJobSave.input), true
}

// overlayJobInput projects a save input onto an application's detail fields.
func overlayJobInput(app model.JobApplication, in jobapplications.UpdateDetailsInput) model.JobApplication {
	app.CompanyName, app.PositionName, app.AppliedDate = in.CompanyName, in.PositionName, in.AppliedDate
	app.ApplicationURL, app.Channel, app.Notes = in.ApplicationURL, in.Channel, in.Notes
	if in.ClosureReason != nil && app.Lifecycle == model.JobLifecycleClosed {
		app.ClosureReason = *in.ClosureReason
	}
	return app
}

// jobClosureReasonEqual compares optional closure reasons.
func jobClosureReasonEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (w *Workbench) markJobDirty() {
	w.jobDirty = true
	w.jobSaveRevision++
	w.loadVersions["job_selection"]++
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
	in := w.jobInputFromDraft()
	if in.CompanyName == "" {
		return jobSave{}, fmt.Errorf("%w: 公司名称不能为空", model.ErrInvalid)
	}
	if in.PositionName == "" {
		return jobSave{}, fmt.Errorf("%w: 岗位名称不能为空", model.ErrInvalid)
	}
	if in.AppliedDate == "" {
		return jobSave{}, fmt.Errorf("%w: 投递日期不能为空", model.ErrInvalid)
	}
	if _, err := model.ParseCalendarDate(in.AppliedDate); err != nil {
		return jobSave{}, err
	}
	if _, err := model.ValidateJobURL(in.ApplicationURL); err != nil {
		return jobSave{}, err
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
// While a background write is in flight, the draft rebases on that write's result — it
// is already committing — and the form only reverts to the last committed record when
// no write matches the selection.
func (w *Workbench) DiscardJobChanges() {
	if w.IsRestoring() || !w.jobSelected() {
		return
	}
	w.afterSave = nil
	w.pendingJobSave = nil
	w.jobSaveRevision++
	if w.jobSaving {
		if baseline, ok := w.inFlightJobBaseline(); ok {
			w.resetJobEditFields(baseline)
		} else {
			w.resetJobEditFields(*w.detailJob)
		}
		// The previewed baseline is still being written; it must not read as saved
		// before the acknowledgement lands (and it may still roll back on failure).
		w.jobSaveState = "saving"
		w.discardJobAfterSave = true
		w.discardJobRebase = true
		return
	}
	w.syncDetailFromJob(*w.detailJob)
}

func (w *Workbench) startJobSave(request jobSave) error {
	service := w.app.JobApplications
	w.jobSaving = true
	w.inFlightJobSave = &request
	work := func() (model.JobApplication, error) {
		return service.UpdateDetails(context.Background(), request.input)
	}
	apply := func(updated model.JobApplication, err error) {
		w.jobSaving = false
		w.inFlightJobSave = nil
		rebase := w.discardJobRebase
		w.discardJobRebase = false
		if request.epoch != w.currentDataEpoch() || request.jobID != w.selectedJobID {
			return
		}
		if err == nil {
			w.detailJob = &updated
			w.jobLastSavedAt = w.now()
			w.upsertJobInCache(updated)
		}
		if w.discardJobAfterSave {
			w.discardJobAfterSave = false
			if w.detailJob != nil {
				w.syncDetailFromJob(*w.detailJob)
			}
			if next := w.afterSave; next != nil {
				w.afterSave = nil
				next()
			}
			return
		}
		if err != nil && rebase {
			w.rebaseJobDraftOnCommitted(request.input)
		}
		if next := w.pendingJobSave; next != nil {
			w.pendingJobSave = nil
			if err != nil && rebase {
				next.input = w.jobInputFromDraft()
			}
			_ = w.startJobSave(*next)
			return
		}
		if request.revision != w.jobSaveRevision {
			// A newer invalid draft must not be marked saved by an older completion.
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
		w.editJobReasonDirty = false
		w.jobSaveState = "saved"
		w.jobSaveErr = ""
		if next := w.afterSave; next != nil {
			w.afterSave = nil
			next()
		}
	}
	if w.host.Dispatch != nil {
		var updated model.JobApplication
		w.runJobWorker(func() error {
			var err error
			updated, err = work()
			return err
		}, func(err error) { apply(updated, err) })
		return nil
	}
	updated, err := work()
	apply(updated, err)
	return err
}
