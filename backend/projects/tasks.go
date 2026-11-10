package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"aion/backend/model"
)

func (s *Service) CreateTask(ctx context.Context, projectID string, in TaskInput) (model.Task, error) {
	in, err := normalizeTask(in, true)
	if err != nil {
		return model.Task{}, err
	}
	var t model.Task
	err = s.store.Write(ctx, func(tx *sql.Tx) error {
		if _, err := visibleProject(ctx, tx, projectID); err != nil {
			return err
		}
		ids, err := columnIDs(ctx, tx, projectID, in.Status)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		t = model.Task{ID: model.NewID(), ProjectID: projectID, Title: in.Title, Description: in.Description,
			Status: in.Status, Priority: in.Priority, Tags: in.Tags, Subtasks: in.Subtasks, DueDate: in.DueDate,
			Position: len(ids), CreatedAt: now, UpdatedAt: now}
		if t.Status == model.Done {
			t.CompletedAt = &now
		}
		tags, _ := json.Marshal(t.Tags)
		subtasks, _ := json.Marshal(t.Subtasks)
		_, err = tx.ExecContext(ctx, `INSERT INTO tasks (`+taskColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,NULL)`,
			t.ID, projectID, t.Title, t.Description, t.Status, t.Priority, string(tags), string(subtasks), t.DueDate,
			t.Position, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), optionalTime(t.CompletedAt))
		return err
	})
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func (s *Service) GetTask(ctx context.Context, id string) (model.Task, error) {
	var t model.Task
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		var err error
		t, err = visibleTask(ctx, tx, id)
		return err
	})
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func (s *Service) ListTasks(ctx context.Context, projectID string, filter TaskFilter) ([]model.Task, error) {
	if (filter.Status != "" && !filter.Status.Valid()) || (filter.Priority != "" && !filter.Priority.Valid()) {
		return nil, invalid("invalid filter status or priority")
	}
	for _, tag := range filter.Tags {
		if strings.TrimSpace(tag) == "" {
			return nil, invalid("filter tag cannot be blank")
		}
	}
	result := make([]model.Task, 0)
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		if _, err := visibleProject(ctx, tx, projectID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE project_id=? AND deleted_at IS NULL
			ORDER BY CASE status WHEN 'planned' THEN 0 WHEN 'in_progress' THEN 1 ELSE 2 END,position,created_at,id`, projectID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			t, err := scanTask(rows)
			if err != nil {
				return err
			}
			if matches(t, filter) {
				result = append(result, t)
			}
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func matches(t model.Task, f TaskFilter) bool {
	if (f.Status != "" && t.Status != f.Status) || (f.Priority != "" && t.Priority != f.Priority) {
		return false
	}
	for _, wanted := range f.Tags {
		found := false
		for _, tag := range t.Tags {
			if tag == strings.TrimSpace(wanted) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	q := strings.ToLower(strings.TrimSpace(f.Query))
	if q == "" || strings.Contains(strings.ToLower(t.Title), q) || strings.Contains(strings.ToLower(t.Description), q) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

// UpdateTask replaces all content fields. Changing Status appends the task to
// the destination column. Use MoveTask to choose an explicit destination index.
func (s *Service) UpdateTask(ctx context.Context, id string, in TaskInput) (model.Task, error) {
	in, err := normalizeTask(in, false)
	if err != nil {
		return model.Task{}, err
	}
	var t model.Task
	err = s.store.Write(ctx, func(tx *sql.Tx) error {
		var err error
		t, err = visibleTask(ctx, tx, id)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if in.Status != t.Status {
			ids, err := columnIDs(ctx, tx, t.ProjectID, in.Status)
			if err != nil {
				return err
			}
			t, err = moveTask(ctx, tx, t, in.Status, len(ids), now)
			if err != nil {
				return err
			}
		}
		t.Title, t.Description, t.Priority = in.Title, in.Description, in.Priority
		t.Tags, t.Subtasks, t.DueDate, t.UpdatedAt = in.Tags, in.Subtasks, in.DueDate, now
		tags, _ := json.Marshal(t.Tags)
		subtasks, _ := json.Marshal(t.Subtasks)
		_, err = tx.ExecContext(ctx, `UPDATE tasks SET title=?,description=?,priority=?,tags=?,subtasks=?,due_date=?,updated_at=? WHERE id=?`,
			t.Title, t.Description, t.Priority, string(tags), string(subtasks), t.DueDate, now.Format(time.RFC3339Nano), id)
		return err
	})
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func columnIDs(ctx context.Context, tx *sql.Tx, projectID string, status model.Status) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM tasks WHERE project_id=? AND status=? AND deleted_at IS NULL ORDER BY position,created_at,id`, projectID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func writeColumn(ctx context.Context, tx *sql.Tx, ids []string, now time.Time) error {
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET position=?,updated_at=? WHERE id=? AND position<>?`, i, now.Format(time.RFC3339Nano), id, i); err != nil {
			return err
		}
	}
	return nil
}

func without(ids []string, excluded string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != excluded {
			result = append(result, id)
		}
	}
	return result
}

func insertAt(ids []string, index int, id string) []string {
	ids = append(ids, "")
	copy(ids[index+1:], ids[index:])
	ids[index] = id
	return ids
}

// MoveTask inserts at a zero-based index in the destination column AFTER
// removing the task from its old column. 0..destination length is valid.
// The relative order of every other task is preserved, even when moving from
// a filtered board. A task cannot move to a different project.
func (s *Service) MoveTask(ctx context.Context, id string, status model.Status, index int) (model.Task, error) {
	if !status.Valid() || index < 0 {
		return model.Task{}, invalid("invalid move status or index")
	}
	var t model.Task
	err := s.store.Write(ctx, func(tx *sql.Tx) error {
		var err error
		t, err = visibleTask(ctx, tx, id)
		if err != nil {
			return err
		}
		t, err = moveTask(ctx, tx, t, status, index, time.Now().UTC())
		return err
	})
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func moveTask(ctx context.Context, tx *sql.Tx, t model.Task, status model.Status, index int, now time.Time) (model.Task, error) {
	oldStatus := t.Status
	source, err := columnIDs(ctx, tx, t.ProjectID, oldStatus)
	if err != nil {
		return t, err
	}
	source = without(source, t.ID)
	dest := source
	if oldStatus != status {
		dest, err = columnIDs(ctx, tx, t.ProjectID, status)
		if err != nil {
			return t, err
		}
	}
	if index < 0 || index > len(dest) {
		return t, invalid("destination index must be between 0 and %d", len(dest))
	}
	dest = insertAt(dest, index, t.ID)
	if oldStatus != status {
		if status == model.Done {
			t.CompletedAt = &now
		} else {
			t.CompletedAt = nil
		}
	}
	t.Status, t.Position, t.UpdatedAt = status, index, now
	if _, err = tx.ExecContext(ctx, `UPDATE tasks SET status=?,position=?,completed_at=?,updated_at=? WHERE id=?`,
		status, index, optionalTime(t.CompletedAt), now.Format(time.RFC3339Nano), t.ID); err != nil {
		return t, err
	}
	if oldStatus != status {
		if err = writeColumn(ctx, tx, source, now); err != nil {
			return t, err
		}
	}
	return t, writeColumn(ctx, tx, dest, now)
}

// ReorderTasks permutes the slots occupied by the supplied IDs, in the supplied
// order. Unlisted tasks keep their slots. This supports filtered subsets without
// erasing or moving hidden tasks; pass all column IDs for a complete reorder.
// Duplicate IDs and IDs outside this live project/column are rejected atomically.
func (s *Service) ReorderTasks(ctx context.Context, projectID string, status model.Status, ids []string) error {
	if !status.Valid() {
		return invalid("invalid reorder status")
	}
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || selected[id] {
			return invalid("reorder IDs must be nonblank and unique")
		}
		selected[id] = true
	}
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		if _, err := visibleProject(ctx, tx, projectID); err != nil {
			return err
		}
		column, err := columnIDs(ctx, tx, projectID, status)
		if err != nil {
			return err
		}
		slots := make([]int, 0, len(ids))
		for i, id := range column {
			if selected[id] {
				slots = append(slots, i)
			}
		}
		if len(slots) != len(ids) {
			return invalid("reorder IDs must belong to the live project and column")
		}
		for i, slot := range slots {
			column[slot] = ids[i]
		}
		return writeColumn(ctx, tx, column, time.Now().UTC())
	})
}

func (s *Service) DeleteTask(ctx context.Context, id string) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		t, err := visibleTask(ctx, tx, id)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if _, err = tx.ExecContext(ctx, `UPDATE tasks SET deleted_at=?,updated_at=? WHERE id=?`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id); err != nil {
			return err
		}
		ids, err := columnIDs(ctx, tx, t.ProjectID, t.Status)
		if err != nil {
			return err
		}
		return writeColumn(ctx, tx, ids, now)
	})
}

func restoreTask(ctx context.Context, tx *sql.Tx, id string) error {
	t, err := taskByID(ctx, tx, id)
	if err != nil {
		return err
	}
	if _, err = visibleProject(ctx, tx, t.ProjectID); err != nil {
		return err
	}
	if t.DeletedAt == nil {
		return model.ErrConflict
	}
	ids, err := columnIDs(ctx, tx, t.ProjectID, t.Status)
	if err != nil {
		return err
	}
	index := max(0, min(t.Position, len(ids)))
	now := time.Now().UTC()
	if _, err = tx.ExecContext(ctx, `UPDATE tasks SET deleted_at=NULL,updated_at=? WHERE id=?`, now.Format(time.RFC3339Nano), id); err != nil {
		return err
	}
	return writeColumn(ctx, tx, insertAt(ids, index, id), now)
}

func (s *Service) RestoreTask(ctx context.Context, id string) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error { return restoreTask(ctx, tx, id) })
}
