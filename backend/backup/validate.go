package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"aion/backend/model"
)

// SQLite integrity_check verifies physical pages and SQL constraints. Check the
// structured strings consumed by Go readers as well, before installing the DB.
func validateValues(ctx context.Context, db *sql.DB) error {
	tsQuery := `SELECT created_at,updated_at,deleted_at,NULL FROM projects
		UNION ALL SELECT created_at,updated_at,deleted_at,completed_at FROM tasks
		UNION ALL SELECT created_at,NULL,NULL,NULL FROM attachments
		UNION ALL SELECT created_at,updated_at,deleted_at,closed_at FROM job_applications
		UNION ALL SELECT created_at,occurred_at,NULL,NULL FROM job_application_events`
	if err := validateTimestampColumns(ctx, db, tsQuery); err != nil {
		return err
	}
	if err := validateTaskRows(ctx, db); err != nil {
		return err
	}
	if err := validateJobApplicationRows(ctx, db); err != nil {
		return err
	}
	return validateJobApplicationEventRows(ctx, db)
}

func validateTimestampColumns(ctx context.Context, db *sql.DB, query string) error {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var required string
		var opt1, opt2, opt3 sql.NullString
		if err = rows.Scan(&required, &opt1, &opt2, &opt3); err != nil {
			return err
		}
		for _, value := range []sql.NullString{{String: required, Valid: true}, opt1, opt2, opt3} {
			if !value.Valid {
				continue
			}
			if _, err = time.Parse(time.RFC3339Nano, value.String); err != nil {
				return invalid("invalid timestamp")
			}
		}
	}
	return rows.Err()
}

func validateTaskRows(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT tags,subtasks,due_date FROM tasks")
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
			if _, err = model.ParseCalendarDate(due); err != nil {
				return invalid("invalid due date")
			}
		}
	}
	return rows.Err()
}

func validateJobApplicationRows(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT company_name,position_name,applied_date,stage,lifecycle,outcome,closure_reason,closed_at,application_url FROM job_applications`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var company, position, appliedDate, stageRaw, lifecycleRaw, outcomeRaw, reason, appURL string
		var closedAt sql.NullString
		if err = rows.Scan(&company, &position, &appliedDate, &stageRaw, &lifecycleRaw, &outcomeRaw, &reason, &closedAt, &appURL); err != nil {
			return err
		}
		if strings.TrimSpace(company) == "" || strings.TrimSpace(position) == "" {
			return invalid("blank job application company or position")
		}
		if _, err = model.ParseCalendarDate(appliedDate); err != nil {
			return invalid("invalid applied date")
		}
		if appURL != "" {
			if _, err = model.ValidateJobURL(appURL); err != nil {
				return invalid("invalid application URL")
			}
		}
		stage := model.JobStage(stageRaw)
		lifecycle := model.JobLifecycle(lifecycleRaw)
		outcome := model.JobOutcome(outcomeRaw)
		if !stage.Valid() || !lifecycle.Valid() {
			return invalid("invalid job application stage or lifecycle")
		}
		switch lifecycle {
		case model.JobLifecycleActive:
			if outcome != "" || reason != "" || closedAt.Valid {
				return invalid("active job application cannot hold closure fields")
			}
		case model.JobLifecycleClosed:
			if !outcome.Valid() || !closedAt.Valid {
				return invalid("closed job application requires outcome and closed_at")
			}
			// Closing enforces the Offer stage at that instant. The user may
			// subsequently correct the stage while retaining the outcome.
		}
	}
	return rows.Err()
}

func validateJobApplicationEventRows(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT event_type,from_stage,to_stage,outcome FROM job_application_events`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var typeRaw, fromRaw, toRaw, outcomeRaw string
		if err = rows.Scan(&typeRaw, &fromRaw, &toRaw, &outcomeRaw); err != nil {
			return err
		}
		if !model.JobEventType(typeRaw).Valid() {
			return invalid("invalid job application event type")
		}
		if fromRaw != "" && !model.JobStage(fromRaw).Valid() {
			return invalid("invalid job event from_stage")
		}
		if toRaw != "" && !model.JobStage(toRaw).Valid() {
			return invalid("invalid job event to_stage")
		}
		if outcomeRaw != "" && !model.JobOutcome(outcomeRaw).Valid() {
			return invalid("invalid job event outcome")
		}
	}
	return rows.Err()
}
