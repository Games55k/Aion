package store

import (
	"cmp"
	"context"
	"database/sql"
	"slices"
	"time"

	"aion/backend/model"
)

// ListTrash reads one or more modules' trash in a single snapshot. RFC3339Nano
// strings need to be parsed before sorting to compare instants correctly.
func (s *Store) ListTrash(ctx context.Context, query string) ([]model.TrashItem, error) {
	items := make([]model.TrashItem, 0)
	err := s.Read(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item model.TrashItem
			var deleted string
			if err := rows.Scan(&item.ID, &item.Kind, &item.Name, &item.ProjectID, &deleted); err != nil {
				return err
			}
			item.DeletedAt, err = time.Parse(time.RFC3339Nano, deleted)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(x, y model.TrashItem) int {
		if c := y.DeletedAt.Compare(x.DeletedAt); c != 0 {
			return c
		}
		if c := cmp.Compare(x.Kind, y.Kind); c != 0 {
			return c
		}
		return cmp.Compare(x.ID, y.ID)
	})
	return items, nil
}
