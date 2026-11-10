package projects

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	"aion/backend/model"
)

func (s *Service) CreateProject(ctx context.Context, in ProjectInput) (model.Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return model.Project{}, invalid("project name is required")
	}
	now := time.Now().UTC()
	p := model.Project{ID: model.NewID(), Name: in.Name, Icon: in.Icon, Description: in.Description, CreatedAt: now, UpdatedAt: now}
	err := s.store.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO projects (`+projectColumns+`) VALUES (?,?,?,?,?,?,?,NULL)`,
			p.ID, p.Name, p.Icon, p.Description, false, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
		return err
	})
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

func (s *Service) GetProject(ctx context.Context, id string) (model.Project, error) {
	var p model.Project
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		var err error
		p, err = projectByID(ctx, tx, id)
		if err == nil && p.DeletedAt != nil {
			return model.ErrNotFound
		}
		return err
	})
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

func (s *Service) ListProjects(ctx context.Context, filter ProjectFilter) ([]model.Project, error) {
	result := make([]model.Project, 0)
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		query := `SELECT ` + projectColumns + ` FROM projects WHERE deleted_at IS NULL`
		var args []any
		if filter.Archived != nil {
			query += ` AND archived=?`
			args = append(args, *filter.Archived)
		}
		rows, err := tx.QueryContext(ctx, query+` ORDER BY created_at,id`, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanProject(rows)
			if err != nil {
				return err
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	// RFC3339Nano omits trailing fractional zeros, so SQL text ordering alone
	// is not chronological for all timestamps within the same second.
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Service) UpdateProject(ctx context.Context, id string, in ProjectInput) (model.Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return model.Project{}, invalid("project name is required")
	}
	var p model.Project
	err := s.store.Write(ctx, func(tx *sql.Tx) error {
		var err error
		p, err = projectByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if p.DeletedAt != nil {
			return model.ErrNotFound
		}
		p.Name, p.Icon, p.Description = in.Name, in.Icon, in.Description
		p.UpdatedAt = time.Now().UTC()
		_, err = tx.ExecContext(ctx, `UPDATE projects SET name=?,icon=?,description=?,updated_at=? WHERE id=?`,
			p.Name, p.Icon, p.Description, p.UpdatedAt.Format(time.RFC3339Nano), id)
		return err
	})
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

func (s *Service) ArchiveProject(ctx context.Context, id string, archived bool) (model.Project, error) {
	var p model.Project
	err := s.store.Write(ctx, func(tx *sql.Tx) error {
		var err error
		p, err = projectByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if p.DeletedAt != nil {
			return model.ErrNotFound
		}
		p.Archived, p.UpdatedAt = archived, time.Now().UTC()
		_, err = tx.ExecContext(ctx, `UPDATE projects SET archived=?,updated_at=? WHERE id=?`, archived, p.UpdatedAt.Format(time.RFC3339Nano), id)
		return err
	})
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

// DeleteProject marks ONLY the project. Task deletion provenance and board
// positions are unchanged, including tasks independently deleted beforehand.
func (s *Service) DeleteProject(ctx context.Context, id string) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		p, err := projectByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if p.DeletedAt != nil {
			return model.ErrConflict
		}
		now := model.Timestamp()
		_, err = tx.ExecContext(ctx, `UPDATE projects SET deleted_at=?,updated_at=? WHERE id=?`, now, now, id)
		return err
	})
}

func restoreProject(ctx context.Context, tx *sql.Tx, id string) error {
	p, err := projectByID(ctx, tx, id)
	if err != nil {
		return err
	}
	if p.DeletedAt == nil {
		return model.ErrConflict
	}
	_, err = tx.ExecContext(ctx, `UPDATE projects SET deleted_at=NULL,updated_at=? WHERE id=?`, model.Timestamp(), id)
	return err
}

func (s *Service) RestoreProject(ctx context.Context, id string) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error { return restoreProject(ctx, tx, id) })
}
