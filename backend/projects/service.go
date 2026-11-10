// Package projects implements local project boards and their recycle bin.
// All reads and mutations share the Store transaction lock with other modules.
package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"aion/backend/model"
	"aion/backend/store"
)

type Service struct{ store *store.Store }

func New(s *store.Store) *Service { return &Service{store: s} }

// ProjectInput replaces editable project content. Identity and lifecycle fields
// are changed only by their dedicated operations.
type ProjectInput struct {
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

// ProjectFilter includes both archived and unarchived projects if Archived is nil.
// Deleted projects are available only through ListTrash.
type ProjectFilter struct {
	Archived *bool `json:"archived,omitempty"`
}

// TaskInput is a full content replacement, never a persisted model: callers
// cannot change task identity, project, position, or server-owned timestamps.
// CreateTask defaults empty Status/Priority to planned/none. UpdateTask requires
// explicit valid values. Blank subtask IDs receive a new ID.
type TaskInput struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      model.Status    `json:"status"`
	Priority    model.Priority  `json:"priority"`
	Tags        []string        `json:"tags"`
	Subtasks    []model.Subtask `json:"subtasks"`
	DueDate     string          `json:"due_date"`
}

// TaskFilter conditions are combined with AND. Query is a case-insensitive
// literal substring of title, description, or any tag. Tags requires all exact
// tags; empty Status/Priority means any. Filtering never changes board positions.
type TaskFilter struct {
	Query    string         `json:"query"`
	Tags     []string       `json:"tags"`
	Status   model.Status   `json:"status"`
	Priority model.Priority `json:"priority"`
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", model.ErrInvalid, fmt.Sprintf(format, args...))
}

func normalizeTask(in TaskInput, defaults bool) (TaskInput, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return in, invalid("task title is required")
	}
	if defaults && in.Status == "" {
		in.Status = model.Planned
	}
	if defaults && in.Priority == "" {
		in.Priority = model.NoPriority
	}
	if !in.Status.Valid() || !in.Priority.Valid() {
		return in, invalid("invalid status or priority")
	}
	if in.DueDate != "" {
		d, err := time.Parse("2006-01-02", in.DueDate)
		if err != nil || d.Format("2006-01-02") != in.DueDate {
			return in, invalid("due_date must be a real YYYY-MM-DD calendar date")
		}
	}
	tags := make([]string, 0, len(in.Tags))
	seen := make(map[string]bool)
	for _, tag := range in.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return in, invalid("tag cannot be blank")
		}
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	in.Tags = tags
	subtasks := make([]model.Subtask, 0, len(in.Subtasks))
	seen = make(map[string]bool)
	for _, sub := range in.Subtasks {
		sub.Title = strings.TrimSpace(sub.Title)
		if sub.Title == "" {
			return in, invalid("subtask title is required")
		}
		if sub.ID == "" {
			sub.ID = model.NewID()
		}
		if sub.ID != strings.TrimSpace(sub.ID) || seen[sub.ID] {
			return in, invalid("subtask IDs must be nonblank and unique")
		}
		seen[sub.ID] = true
		subtasks = append(subtasks, sub)
	}
	in.Subtasks = subtasks
	return in, nil
}

const projectColumns = `id,name,icon,description,archived,created_at,updated_at,deleted_at`
const taskColumns = `id,project_id,title,description,status,priority,tags,subtasks,due_date,position,created_at,updated_at,completed_at,deleted_at`

type scanner interface{ Scan(...any) error }

func parseOptional(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanProject(row scanner) (model.Project, error) {
	var p model.Project
	var created, updated string
	var deleted sql.NullString
	err := row.Scan(&p.ID, &p.Name, &p.Icon, &p.Description, &p.Archived, &created, &updated, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return p, model.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if p.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return p, err
	}
	if p.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return p, err
	}
	p.DeletedAt, err = parseOptional(deleted)
	return p, err
}

func scanTask(row scanner) (model.Task, error) {
	var t model.Task
	var tags, subtasks, created, updated string
	var completed, deleted sql.NullString
	err := row.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&tags, &subtasks, &t.DueDate, &t.Position, &created, &updated, &completed, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return t, model.ErrNotFound
	}
	if err != nil {
		return t, err
	}
	if err = json.Unmarshal([]byte(tags), &t.Tags); err != nil {
		return t, err
	}
	if err = json.Unmarshal([]byte(subtasks), &t.Subtasks); err != nil {
		return t, err
	}
	if t.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return t, err
	}
	if t.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return t, err
	}
	if t.CompletedAt, err = parseOptional(completed); err != nil {
		return t, err
	}
	t.DeletedAt, err = parseOptional(deleted)
	return t, err
}

func projectByID(ctx context.Context, tx *sql.Tx, id string) (model.Project, error) {
	return scanProject(tx.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE id=?`, id))
}

func taskByID(ctx context.Context, tx *sql.Tx, id string) (model.Task, error) {
	return scanTask(tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id=?`, id))
}

func visibleProject(ctx context.Context, tx *sql.Tx, id string) (model.Project, error) {
	p, err := projectByID(ctx, tx, id)
	if err == nil && p.DeletedAt != nil {
		err = model.ErrParentDeleted
	}
	return p, err
}

func visibleTask(ctx context.Context, tx *sql.Tx, id string) (model.Task, error) {
	t, err := taskByID(ctx, tx, id)
	if err != nil {
		return t, err
	}
	if t.DeletedAt != nil {
		return t, model.ErrNotFound
	}
	_, err = visibleProject(ctx, tx, t.ProjectID)
	return t, err
}

func optionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
