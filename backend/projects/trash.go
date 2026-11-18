package projects

import (
	"context"
	"database/sql"

	"aion/backend/model"
)

const (
	TrashProject = "project"
	TrashTask    = "task"
)

// TrashQuery selects independently deleted items, including deleted tasks whose parent is in trash.
const TrashQuery = `SELECT id,'project',name,'',deleted_at FROM projects WHERE deleted_at IS NOT NULL
			UNION ALL SELECT id,'task',title,project_id,deleted_at FROM tasks WHERE deleted_at IS NOT NULL`

// ListTrash returns this module's trash, newest deletion first.
func (s *Service) ListTrash(ctx context.Context) ([]model.TrashItem, error) {
	return s.store.ListTrash(ctx, TrashQuery)
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
	return s.store.Write(ctx, func(tx *sql.Tx) error { return ClearTrashTx(ctx, tx) })
}

// ClearTrashTx purges this module's recycled records in the caller's transaction.
func ClearTrashTx(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE deleted_at IS NOT NULL`); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE deleted_at IS NOT NULL`)
	return err
}
