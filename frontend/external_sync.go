package frontend

import (
	"context"
	"errors"

	"aion/backend/model"
)

// ExternalChange describes a mutation committed outside the UI (for example via local MCP/IPC)
// so Workbench can refresh affected views without overwriting active user drafts.
type ExternalChange struct {
	Scope     string
	ProjectID string
	TaskID    string
	JobID     string
}

func (w *Workbench) hasActiveTaskDraft() bool {
	return w.dirty || w.saving || w.pendingSave != nil || w.hasUnsavedChangesFromDetailTask()
}

func (w *Workbench) hasActiveJobDraft() bool {
	return w.jobDirty || w.jobSaving || w.pendingJobSave != nil || len(w.jobWrites) > 0 || w.hasUnsavedChangesFromDetailJob()
}

// HandleExternalChange refreshes affected Workbench views after an external backend write.
// Must be invoked on the UI thread (e.g. via Host.Dispatch / win.Update).
func (w *Workbench) HandleExternalChange(change ExternalChange) {
	if w.IsRestoring() {
		return
	}
	switch change.Scope {
	case "projects":
		w.reloadProjects(true)
		w.reloadFocusTasks(true)
	case "tasks":
		w.reloadProjects(true)
		w.reloadFocusTasks(true)
	case "job_applications":
		w.reloadJobApplications(true)
		if w.selectedJobID != "" && (change.JobID == "" || change.JobID == w.selectedJobID) {
			w.reloadSelectedJobExternal(w.selectedJobID)
		}
	case "trash":
		w.reloadTrash(true)
		w.reloadProjects(true)
		w.reloadFocusTasks(true)
		w.reloadJobApplications(true)
		if w.selectedJobID != "" {
			w.reloadSelectedJobExternal(w.selectedJobID)
		}
	default:
		w.reloadProjects(true)
		w.reloadFocusTasks(true)
		w.reloadJobApplications(true)
		w.reloadTrash(true)
		if w.selectedJobID != "" {
			w.reloadSelectedJobExternal(w.selectedJobID)
		}
	}
}

func (w *Workbench) reloadSelectedJobExternal(appID string) {
	service := w.app.JobApplications
	w.loadUI("job_external_detail", true, func() func() {
		ctx := context.Background()
		app, err := service.Get(ctx, appID)
		var events []model.JobApplicationEvent
		if err == nil {
			events, err = service.ListEvents(ctx, appID)
		}
		return func() {
			if w.selectedJobID != appID {
				return
			}
			if errors.Is(err, model.ErrNotFound) {
				if !w.hasActiveJobDraft() {
					w.resetSelectedJobState()
				}
				return
			}
			if err != nil {
				return
			}
			w.detailJobEvents = events
			if !w.hasActiveJobDraft() {
				w.syncDetailFromJob(app)
			}
		}
	})
}
