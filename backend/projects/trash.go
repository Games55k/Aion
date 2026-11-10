package projects

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"aion/backend/model"
)

const (
	TrashProject = "project"
	TrashTask    = "task"
)

// ListTrash includes deleted projects and independently deleted tasks (including
// those inside deleted projects). Tasks merely hidden by a deleted project do
// not acquire their own trash entry.
func (s *Service) ListTrash(ctx context.Context) ([]model.TrashItem, error) {
	result := make([]model.TrashItem, 0)
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT id,'project',name,'',deleted_at FROM projects WHERE deleted_at IS NOT NULL
			UNION ALL SELECT id,'task',title,project_id,deleted_at FROM tasks WHERE deleted_at IS NOT NULL
			ORDER BY 5 DESC,2,1`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item model.TrashItem
			var deleted string
			if err = rows.Scan(&item.ID, &item.Kind, &item.Name, &item.ProjectID, &deleted); err != nil {
				return err
			}
			if item.DeletedAt, err = time.Parse(time.RFC3339Nano, deleted); err != nil {
				return err
			}
			result = append(result, item)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].DeletedAt.Equal(result[j].DeletedAt) {
			return result[i].DeletedAt.After(result[j].DeletedAt)
		}
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *Service) RestoreTrash(ctx context.Context, kind, id string) error {
	switch kind {
	case TrashProject:
		return s.RestoreProject(ctx, id)
	case TrashTask:
		return s.RestoreTask(ctx, id)
	default:
		return invalid("unknown trash kind %q", kind)
	}
}

// PermanentlyDelete deletes only an explicitly soft-deleted item. Project
// deletion cascades to every child task and attachment row. Managed files are
// deliberately left for attachments.Service.GC under the shared store lock.
func (s *Service) PermanentlyDelete(ctx context.Context, kind, id string) error {
	if kind != TrashProject && kind != TrashTask {
		return invalid("unknown trash kind %q", kind)
	}
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		if kind == TrashProject {
			p, err := projectByID(ctx, tx, id)
			if err != nil {
				return err
			}
			if p.DeletedAt == nil {
				return model.ErrConflict
			}
			_, err = tx.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, id)
			return err
		}
		t, err := taskByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if t.DeletedAt == nil {
			return model.ErrConflict
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM tasks WHERE id=?`, id)
		return err
	})
}

// ClearTrash atomically purges deleted projects (with their children) and
// independently deleted tasks. Live and archived projects remain untouched.
func (s *Service) ClearTrash(ctx context.Context) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE deleted_at IS NOT NULL`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE deleted_at IS NOT NULL`)
		return err
	})
}
