package jobapplications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"aion/backend/model"
	"aion/backend/store"
)

// Options configures optional dependencies for the job applications service.
type Options struct {
	Now func() time.Time
}

// Service manages job applications, stage transitions, timeline events, and trash lifecycle.
type Service struct {
	s   *store.Store
	now func() time.Time
}

// New creates a job applications service backed by s.
func New(s *store.Store, opts ...Options) *Service {
	nowFn := time.Now
	for _, opt := range opts {
		if opt.Now != nil {
			nowFn = opt.Now
		}
	}
	return &Service{s: s, now: nowFn}
}

func (s *Service) nowUTC() time.Time {
	return s.now().UTC()
}

func (s *Service) timestamp() string {
	return s.nowUTC().Format(time.RFC3339Nano)
}

func (s *Service) todayLocalDate() string {
	return s.now().In(time.Local).Format("2006-01-02")
}

func (s *Service) resolveEventTime(dateStr string) (time.Time, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return s.nowUTC(), nil
	}
	parsed, err := model.ParseCalendarDate(trimmed)
	if err != nil {
		return time.Time{}, err
	}
	if trimmed == s.todayLocalDate() {
		return s.nowUTC(), nil
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 12, 0, 0, 0, time.Local).UTC(), nil
}

func parseTime(raw string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", raw, err)
	}
	return t, nil
}

func parseOptionalTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanApplication(scanner interface {
	Scan(dest ...any) error
}) (model.JobApplication, error) {
	var app model.JobApplication
	var stageRaw, lifecycleRaw, outcomeRaw string
	var closedAtRaw, deletedAtRaw sql.NullString
	var createdAtRaw, updatedAtRaw string
	var reachedOfferInt int
	if err := scanner.Scan(
		&app.ID,
		&app.CompanyName,
		&app.PositionName,
		&app.AppliedDate,
		&stageRaw,
		&lifecycleRaw,
		&outcomeRaw,
		&app.ClosureReason,
		&closedAtRaw,
		&app.ApplicationURL,
		&app.Channel,
		&app.NextFollowUpDate,
		&app.Notes,
		&reachedOfferInt,
		&createdAtRaw,
		&updatedAtRaw,
		&deletedAtRaw,
	); err != nil {
		return model.JobApplication{}, err
	}
	app.Stage = model.JobStage(stageRaw)
	app.Lifecycle = model.JobLifecycle(lifecycleRaw)
	app.Outcome = model.JobOutcome(outcomeRaw)
	app.ReachedOffer = reachedOfferInt != 0
	var err error
	if app.CreatedAt, err = parseTime(createdAtRaw); err != nil {
		return model.JobApplication{}, err
	}
	if app.UpdatedAt, err = parseTime(updatedAtRaw); err != nil {
		return model.JobApplication{}, err
	}
	if app.ClosedAt, err = parseOptionalTime(closedAtRaw); err != nil {
		return model.JobApplication{}, err
	}
	if app.DeletedAt, err = parseOptionalTime(deletedAtRaw); err != nil {
		return model.JobApplication{}, err
	}
	return app, nil
}

func scanEvent(scanner interface {
	Scan(dest ...any) error
}) (model.JobApplicationEvent, error) {
	var ev model.JobApplicationEvent
	var typeRaw, fromRaw, toRaw, outcomeRaw string
	var occurredRaw, createdRaw string
	if err := scanner.Scan(
		&ev.ID,
		&ev.ApplicationID,
		&typeRaw,
		&fromRaw,
		&toRaw,
		&outcomeRaw,
		&ev.Reason,
		&ev.Note,
		&occurredRaw,
		&createdRaw,
	); err != nil {
		return model.JobApplicationEvent{}, err
	}
	ev.EventType = model.JobEventType(typeRaw)
	ev.FromStage = model.JobStage(fromRaw)
	ev.ToStage = model.JobStage(toRaw)
	ev.Outcome = model.JobOutcome(outcomeRaw)
	var err error
	if ev.OccurredAt, err = parseTime(occurredRaw); err != nil {
		return model.JobApplicationEvent{}, err
	}
	if ev.CreatedAt, err = parseTime(createdRaw); err != nil {
		return model.JobApplicationEvent{}, err
	}
	return ev, nil
}

const selectApplicationColumns = `SELECT
	id, company_name, position_name, applied_date,
	stage, lifecycle, outcome, closure_reason, closed_at,
	application_url, channel, next_follow_up_date, notes,
	CASE WHEN stage = 'offer' OR EXISTS(
		SELECT 1 FROM job_application_events e
		WHERE e.application_id = job_applications.id
		  AND (e.to_stage = 'offer' OR e.from_stage = 'offer')
	) THEN 1 ELSE 0 END AS reached_offer,
	created_at, updated_at, deleted_at
FROM job_applications`

func getApplicationTx(ctx context.Context, tx *sql.Tx, id string, includeDeleted bool) (model.JobApplication, error) {
	query := selectApplicationColumns + ` WHERE id = ?`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}
	app, err := scanApplication(tx.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.JobApplication{}, model.ErrNotFound
	}
	return app, err
}

func findEventByIDTx(ctx context.Context, tx *sql.Tx, appID, eventID string) (model.JobApplicationEvent, bool, error) {
	if eventID == "" {
		return model.JobApplicationEvent{}, false, nil
	}
	ev, err := scanEvent(tx.QueryRowContext(ctx, `SELECT
		id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		FROM job_application_events WHERE id = ? AND application_id = ?`, eventID, appID))
	if errors.Is(err, sql.ErrNoRows) {
		return model.JobApplicationEvent{}, false, nil
	}
	if err != nil {
		return model.JobApplicationEvent{}, false, err
	}
	return ev, true, nil
}
