package frontend

import (
	"context"
	"errors"
	"slices"

	"aion/backend/model"
	"aion/backend/projects"
)

var errSavePending = errors.New("任务正在后台保存，请稍候")

type taskSave struct {
	taskID          string
	revision, epoch uint64
	input           projects.TaskInput
}

// afterPendingSave keeps the active task or job draft visible until persistence succeeds.
// Repeated navigation replaces the intent, and failures never silently discard a draft.
func (w *Workbench) afterPendingSave(next func(), title, suffix string) bool {
	err := w.FlushPending()
	if errors.Is(err, errSavePending) || errors.Is(err, errJobSavePending) {
		w.afterSave = next
		return false
	}
	if err != nil {
		w.afterSave = nil
		if w.jobSelected() {
			w.jobDetailOpen = true
			if title == "当前任务存在未保存修改" {
				title = "当前投递存在未保存修改"
			}
		} else {
			w.detailExpanded = true
		}
		w.EnqueueToast(ToastProps{Title: title, Description: w.flushErrorDetail(err) + suffix, Type: ToastWarning})
		return false
	}
	w.afterSave = nil
	return true
}

// startTaskSave permits only one database write at a time. New keystrokes retain
// one latest immutable request, so backlog and write order stay bounded. The active
// request stays linked as the baseline a discard must rebase edits on.
func (w *Workbench) startTaskSave(request taskSave) error {
	service := w.app.Projects
	w.saving = true
	w.inFlightSave = &request
	work := func() (model.Task, error) {
		return service.UpdateTask(context.Background(), request.taskID, request.input)
	}
	apply := func(updated model.Task, err error) {
		w.saving = false
		w.inFlightSave = nil
		if request.epoch != w.currentDataEpoch() || request.taskID != w.selectedTaskID {
			return
		}
		projectID := w.detailTask.ProjectID
		// A status save may precede coalesced content edits. Keep its reload
		// pending until the last acknowledgement and any deferred navigation.
		defer func() {
			if w.boardNeedsReload && !w.saving && w.navPage == NavProjectBoard && w.activeProjectID == projectID {
				w.reloadBoardTasks(true)
			}
		}()
		if err == nil {
			for _, key := range []string{"board", "projects", "selection"} {
				w.loadVersions[key]++
			}
			w.detailTask = &updated
			w.lastSavedAt = w.now()
			idx := slices.IndexFunc(w.boardTasks, func(t model.Task) bool { return t.ID == updated.ID })
			w.boardNeedsReload = w.boardNeedsReload || idx < 0 || w.boardTasks[idx].Status != updated.Status
			if idx >= 0 {
				w.boardTasks[idx] = updated
				w.rebuildBoardDerived()
			}

		}
		if w.discardAfterSave {
			w.discardAfterSave = false
			// Transient inputs were already cleared when 放弃修改 was clicked; skip
			// clearing again so text typed while this write was blocked survives.
			w.syncDetailFromTask(*w.detailTask)
			if next := w.afterSave; next != nil {
				w.afterSave = nil
				next()
			}
			return
		}
		if next := w.pendingSave; next != nil {
			w.pendingSave = nil
			_ = w.startTaskSave(*next)
			return
		}
		if request.revision != w.saveRevision {
			// A newer invalid draft must not be marked saved by an older completion.
			w.afterSave = nil
			return
		}
		if err != nil {
			w.dirty = true
			w.saveState, w.saveErr = "error", err.Error()
			w.afterSave = nil
			return
		}
		w.editTags = slices.Clone(updated.Tags)
		w.editSubtasks = slices.Clone(updated.Subtasks)
		w.dirty = false
		w.saveState, w.saveErr = "saved", ""
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
