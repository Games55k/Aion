package jobapplications

import (
	"context"
	"database/sql"

	"aion/backend/model"
)

const TrashKind = "job_application"

// Delete moves a job application to the recycle bin and retains its timeline.
func (s *Service) Delete(ctx context.Context, id string) error {
	nowTS := s.timestamp()
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE job_applications SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			nowTS, nowTS, id,
		)
		return trashMutationResult(ctx, tx, id, res, err)
	})
}

// Restore recovers a soft-deleted job application along with its complete timeline history.
func (s *Service) Restore(ctx context.Context, id string) error {
	nowTS := s.timestamp()
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE job_applications SET deleted_at = NULL, updated_at = ? WHERE id = ? AND deleted_at IS NOT NULL`,
			nowTS, id,
		)
		return trashMutationResult(ctx, tx, id, res, err)
	})
}

// TrashQuery selects recycled applications for the shared trash listing.
const TrashQuery = `SELECT id,'` + TrashKind + `',company_name || ' / ' || position_name,'',deleted_at
	FROM job_applications WHERE deleted_at IS NOT NULL`

// ListTrash returns this module's trash, newest deletion first.
func (s *Service) ListTrash(ctx context.Context) ([]model.TrashItem, error) {
	return s.s.ListTrash(ctx, TrashQuery)
}

// PermanentlyDelete removes one soft-deleted job application and cascades deletion to its timeline events.
func (s *Service) PermanentlyDelete(ctx context.Context, id string) error {
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM job_applications WHERE id = ? AND deleted_at IS NOT NULL`, id)
		return trashMutationResult(ctx, tx, id, res, err)
	})
}

// trashMutationResult distinguishes a missing ID from a state conflict in the same write transaction.
func trashMutationResult(ctx context.Context, tx *sql.Tx, id string, result sql.Result, execErr error) error {
	if execErr != nil {
		return execErr
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM job_applications WHERE id = ?)`, id).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return model.ErrConflict
	}
	return model.ErrNotFound
}

// ClearTrash permanently deletes all soft-deleted job applications and cascades deletion to their timeline events.
func (s *Service) ClearTrash(ctx context.Context) error {
	return s.s.Write(ctx, func(tx *sql.Tx) error { return ClearTrashTx(ctx, tx) })
}

// ClearTrashTx purges this module's recycled records in the caller's transaction.
func ClearTrashTx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM job_applications WHERE deleted_at IS NOT NULL`)
	return err
}
