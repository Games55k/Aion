package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"aion/backend/model"
)

// SQLite integrity_check verifies physical pages and SQL constraints. Check the
// structured strings consumed by Go readers as well, before installing the DB.
func validateValues(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT created_at,updated_at,deleted_at,NULL FROM projects
		UNION ALL SELECT created_at,updated_at,deleted_at,completed_at FROM tasks
		UNION ALL SELECT created_at,created_at,NULL,NULL FROM attachments`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var created, updated string
		var deleted, completed sql.NullString
		if err = rows.Scan(&created, &updated, &deleted, &completed); err != nil {
			rows.Close()
			return err
		}
		for _, value := range []sql.NullString{{String: created, Valid: true}, {String: updated, Valid: true}, deleted, completed} {
			if !value.Valid {
				continue
			}
			if _, err = time.Parse(time.RFC3339Nano, value.String); err != nil {
				rows.Close()
				return invalid("invalid timestamp")
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	rows, err = db.QueryContext(ctx, "SELECT tags,subtasks,due_date FROM tasks")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tags, subtasks, due string
		if err = rows.Scan(&tags, &subtasks, &due); err != nil {
			return err
		}
		var tagValues []string
		var subValues []model.Subtask
		if len(tags) == 0 || tags[0] != '[' || json.Unmarshal([]byte(tags), &tagValues) != nil {
			return invalid("invalid tags JSON")
		}
		if len(subtasks) == 0 || subtasks[0] != '[' || decodeJSON([]byte(subtasks), &subValues) != nil {
			return invalid("invalid subtasks JSON")
		}
		if due != "" {
			if parsed, e := time.Parse("2006-01-02", due); e != nil || parsed.Format("2006-01-02") != due {
				return invalid("invalid due date")
			}
		}
	}
	return rows.Err()
}
