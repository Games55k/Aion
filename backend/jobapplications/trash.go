package jobapplications

import (
	"context"
	"database/sql"

	"aion/backend/model"
)

const TrashKind = "job_application"

// Delete soft-deletes an active job application while retaining all timeline events for potential restore.
func (s *Service) Delete(ctx context.Context, id string) error {
	nowTS := s.timestamp()
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE job_applications SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			nowTS, nowTS, id,
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return model.ErrNotFound
		}
		return nil
	})
}

// Restore recovers a soft-deleted job application along with its complete timeline history.
func (s *Service) Restore(ctx context.Context, id string) error {
	nowTS := s.timestamp()
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE job_applications SET deleted_at = NULL, updated_at = ? WHERE id = ? AND deleted_at IS NOT NULL`,
			nowTS, id,
		)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return model.ErrNotFound
		}
		return nil
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
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return model.ErrNotFound
		}
		return nil
	})
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
