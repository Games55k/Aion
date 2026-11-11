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
		nowStr := now.Format(time.RFC3339Nano)
		t = model.Task{ID: model.NewID(), ProjectID: projectID, Title: in.Title, Description: in.Description,
			Status: in.Status, Priority: in.Priority, Tags: in.Tags, Subtasks: in.Subtasks, DueDate: in.DueDate,
			Position: len(ids), CreatedAt: now, UpdatedAt: now}
		if t.Status == model.Done {
			t.CompletedAt = &now
		}
		tags, err := json.Marshal(t.Tags)
		if err != nil {
			return err
		}
		subtasks, err := json.Marshal(t.Subtasks)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO tasks (`+taskColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,NULL)`,
			t.ID, projectID, t.Title, t.Description, t.Status, t.Priority, string(tags), string(subtasks), t.DueDate,
			t.Position, nowStr, nowStr, optionalTime(t.CompletedAt))
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
	wantedTags := make([]string, 0, len(filter.Tags))
	for _, tag := range filter.Tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			return nil, invalid("filter tag cannot be blank")
		}
		wantedTags = append(wantedTags, trimmed)
	}
	queryText := strings.ToLower(strings.TrimSpace(filter.Query))

	result := make([]model.Task, 0)
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		if _, err := visibleProject(ctx, tx, projectID); err != nil {
			return err
		}
		query := `SELECT ` + taskColumns + ` FROM tasks WHERE project_id=? AND deleted_at IS NULL`
		args := []any{projectID}
		if filter.Status != "" {
			query += ` AND status=?`
			args = append(args, filter.Status)
		}
		if filter.Priority != "" {
			query += ` AND priority=?`
			args = append(args, filter.Priority)
		}
		query += ` ORDER BY CASE status WHEN 'planned' THEN 0 WHEN 'in_progress' THEN 1 ELSE 2 END,position,created_at,id`
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			t, err := scanTask(rows)
			if err != nil {
				return err
			}
			if matchesTextAndTags(t, wantedTags, queryText) {
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

func matchesTextAndTags(t model.Task, wantedTags []string, queryText string) bool {
	for _, wanted := range wantedTags {
		found := false
		for _, tag := range t.Tags {
			if tag == wanted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if queryText == "" || strings.Contains(strings.ToLower(t.Title), queryText) || strings.Contains(strings.ToLower(t.Description), queryText) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), queryText) {
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
		statusChanged := in.Status != t.Status
		var mv columnMove
		if statusChanged {
			t, mv, err = prepareColumnMove(ctx, tx, t, in.Status, 0, true, now)
			if err != nil {
				return err
			}
		}
		t.Title, t.Description, t.Priority = in.Title, in.Description, in.Priority
		t.Tags, t.Subtasks, t.DueDate, t.UpdatedAt = in.Tags, in.Subtasks, in.DueDate, now
		tags, err := json.Marshal(t.Tags)
		if err != nil {
			return err
		}
		subtasks, err := json.Marshal(t.Subtasks)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE tasks SET title=?,description=?,status=?,priority=?,tags=?,subtasks=?,due_date=?,position=?,completed_at=?,updated_at=? WHERE id=?`,
			t.Title, t.Description, t.Status, t.Priority, string(tags), string(subtasks), t.DueDate, t.Position, optionalTime(t.CompletedAt), now.Format(time.RFC3339Nano), id); err != nil {
			return err
		}
		if statusChanged {
			return writeMovedColumns(ctx, tx, t.Status, mv, now)
		}
		return nil
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
	nowStr := now.Format(time.RFC3339Nano)
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET position=?,updated_at=? WHERE id=? AND position<>?`, i, nowStr, id, i); err != nil {
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

type columnMove struct {
	oldStatus model.Status
	source    []string
	dest      []string
}

func prepareColumnMove(ctx context.Context, tx *sql.Tx, t model.Task, status model.Status, index int, appendEnd bool, now time.Time) (model.Task, columnMove, error) {
	mv := columnMove{oldStatus: t.Status}
	var err error
	mv.source, err = columnIDs(ctx, tx, t.ProjectID, mv.oldStatus)
	if err != nil {
		return t, mv, err
	}
	mv.source = without(mv.source, t.ID)
	mv.dest = mv.source
	if mv.oldStatus != status {
		mv.dest, err = columnIDs(ctx, tx, t.ProjectID, status)
		if err != nil {
			return t, mv, err
		}
	}
	if appendEnd {
		index = len(mv.dest)
	}
	if index < 0 || index > len(mv.dest) {
		return t, mv, invalid("destination index must be between 0 and %d", len(mv.dest))
	}
	mv.dest = insertAt(mv.dest, index, t.ID)
	if mv.oldStatus != status {
		if status == model.Done {
			t.CompletedAt = &now
		} else {
			t.CompletedAt = nil
		}
	}
	t.Status, t.Position, t.UpdatedAt = status, index, now
	return t, mv, nil
}

func writeMovedColumns(ctx context.Context, tx *sql.Tx, status model.Status, mv columnMove, now time.Time) error {
	if mv.oldStatus != status {
		if err := writeColumn(ctx, tx, mv.source, now); err != nil {
			return err
		}
	}
	return writeColumn(ctx, tx, mv.dest, now)
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
		now := time.Now().UTC()
		var mv columnMove
		t, mv, err = prepareColumnMove(ctx, tx, t, status, index, false, now)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE tasks SET status=?,position=?,completed_at=?,updated_at=? WHERE id=?`,
			t.Status, t.Position, optionalTime(t.CompletedAt), now.Format(time.RFC3339Nano), t.ID); err != nil {
			return err
		}
		return writeMovedColumns(ctx, tx, t.Status, mv, now)
	})
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
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
		nowStr := now.Format(time.RFC3339Nano)
		if _, err = tx.ExecContext(ctx, `UPDATE tasks SET deleted_at=?,updated_at=? WHERE id=?`, nowStr, nowStr, id); err != nil {
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
