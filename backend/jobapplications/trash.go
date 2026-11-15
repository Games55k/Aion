package jobapplications

import (
	"context"
	"database/sql"
	"fmt"

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

// ListTrash lists all soft-deleted job applications ordered by deletion timestamp descending.
func (s *Service) ListTrash(ctx context.Context) ([]model.TrashItem, error) {
	var out []model.TrashItem
	err := s.s.Read(ctx, func(ro *sql.Tx) error {
		rows, err := ro.QueryContext(ctx, `SELECT id, company_name, position_name, deleted_at FROM job_applications WHERE deleted_at IS NOT NULL`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, company, position, deletedRaw string
			if err := rows.Scan(&id, &company, &position, &deletedRaw); err != nil {
				return err
			}
			deletedAt, err := parseTime(deletedRaw)
			if err != nil {
				return err
			}
			out = append(out, model.TrashItem{
				ID:        id,
				Kind:      TrashKind,
				Name:      fmt.Sprintf("%s / %s", company, position),
				DeletedAt: deletedAt,
			})
		}
		return rows.Err()
	})
	return out, err
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
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM job_applications WHERE deleted_at IS NOT NULL`)
		return err
	})
}
