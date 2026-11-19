package frontend

import (
	"errors"
	"slices"
	"strings"

	"aion/backend/model"
)

var errRecordWritePending = errors.New("上一项更改正在保存，请稍候")

// runRecordWrite serializes actions that depend on the current board or list.
// The slot remains occupied until the UI has applied the result.
func runRecordWrite[T any](w *Workbench, work func() (T, error), apply func(T, error)) (T, error) {
	var zero T
	if w.IsRestoring() {
		return zero, model.ErrConflict
	}
	if w.recordWritePending {
		w.enqueueErrorToast("正在保存更改", errRecordWritePending)
		return zero, errRecordWritePending
	}
	w.recordWritePending = true
	finish := func(result T, err error) {
		w.recordWritePending = false
		next := w.afterRecordWrite
		w.afterRecordWrite = nil
		apply(result, err)
		if next != nil {
			next()
		}
	}
	if dispatch := w.host.Dispatch; dispatch != nil {
		w.taskWG.Go(func() {
			result, err := work()
			dispatch(func() { finish(result, err) })
		})
		return zero, nil
	}
	result, err := work()
	finish(result, err)
	return result, err
}

// runBoardWrite reads the committed board before releasing the write slot.
// Reorder and move actions use the positions from that refreshed board.
func runBoardWrite[T any](w *Workbench, projectID string, work func() (T, error), apply func(T, error)) (T, error) {
	if projectID == "" {
		return runRecordWrite(w, work, apply)
	}
	boardVersion := w.loadVersions["board"]
	readBoard := w.boardReader(projectID)
	var refresh func()
	return runRecordWrite(w, func() (T, error) {
		result, err := work()
		if err == nil {
			refresh = readBoard()
		}
		return result, err
	}, func(result T, err error) {
		if refresh != nil {
			if w.activeProjectID == projectID {
				reloaded := w.loadVersions["board"] != boardVersion
				w.loadVersions["board"]++
				refresh()
				if reloaded {
					// An attachment refresh or a return to this board may have
					// requested a newer snapshot while the write was pending.
					w.reloadBoardTasks(true)
				}
			} else {
				w.reloadProjects(true)
			}
		}
		apply(result, err)
	})
}

func (w *Workbench) removeProjectFromCache(id string) {
	sameProject := func(p model.Project) bool { return p.ID == id }
	w.activeProjects = slices.DeleteFunc(slices.Clone(w.activeProjects), sameProject)
	w.archivedProjects = slices.DeleteFunc(slices.Clone(w.archivedProjects), sameProject)
	w.loadVersions["projects"]++
}

func (w *Workbench) upsertProjectInCache(p model.Project) {
	w.removeProjectFromCache(p.ID)
	list := &w.activeProjects
	if p.Archived {
		list = &w.archivedProjects
	}
	*list = append(*list, p)
	slices.SortFunc(*list, func(a, b model.Project) int {
		if order := a.CreatedAt.Compare(b.CreatedAt); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	w.rebuildProjectNames()
}
