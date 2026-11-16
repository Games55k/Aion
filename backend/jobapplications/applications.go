package jobapplications

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"aion/backend/model"
)

// JobView identifies a built-in application navigation slice.
type JobView string

const (
	ViewAll    JobView = "all"
	ViewActive JobView = "active"
	ViewOffer  JobView = "offer"
	ViewClosed JobView = "closed"
)

// JobSort identifies a supported ordering for job applications.
type JobSort string

const (
	SortAppliedDesc JobSort = "applied_desc"
	SortAppliedAsc  JobSort = "applied_asc"
	SortUpdatedDesc JobSort = "updated_desc"
	SortUpdatedAsc  JobSort = "updated_asc"
)

// Filter specifies search and filter criteria combined with logical AND.
type Filter struct {
	View        JobView
	Query       string
	Stage       model.JobStage
	Outcome     model.JobOutcome
	AppliedDate string
	AppliedFrom string
	AppliedTo   string
}

// CreateInput holds parameters for creating a new job application.
type CreateInput struct {
	CompanyName      string
	PositionName     string
	AppliedDate      string
	Stage            model.JobStage
	ApplicationURL   string
	Channel          string
	NextFollowUpDate string
	Notes            string
}

// UpdateDetailsInput holds editable metadata fields for auto-save without changing workflow state.
type UpdateDetailsInput struct {
	ID               string
	CompanyName      string
	PositionName     string
	AppliedDate      string
	ApplicationURL   string
	Channel          string
	NextFollowUpDate *string
	Notes            string
	ClosureReason    *string
}

// ChangeStageInput transitions an application's stage atomically and records an event.
type ChangeStageInput struct {
	ID      string
	ToStage model.JobStage
	Note    string
	EventID string
}

// CloseInput finishes an active application with an outcome, optional reason, and atomic event.
type CloseInput struct {
	ID            string
	Outcome       model.JobOutcome
	Reason        string
	StageOverride model.JobStage
	EventID       string
}

// ReopenInput reopens a closed application while preserving closure snapshots in historical events.
type ReopenInput struct {
	ID      string
	Note    string
	EventID string
}

// AddNoteInput appends a manual timeline note without altering current workflow fields.
type AddNoteInput struct {
	ApplicationID string
	Note          string
	OccurredDate  string
	EventID       string
}

// BackfillStageInput inserts a historical stage event without overwriting the current stage or lifecycle.
type BackfillStageInput struct {
	ApplicationID string
	Stage         model.JobStage
	OccurredDate  string
	Note          string
	EventID       string
}

// UpdateEventNoteInput updates the note text on an existing timeline event without modifying workflow state.
type UpdateEventNoteInput struct {
	ApplicationID string
	EventID       string
	Note          string
}

// Matches reports whether app satisfies all filter conditions in f.
func (f Filter) Matches(app model.JobApplication) bool {
	if app.DeletedAt != nil {
		return false
	}
	switch f.View {
	case ViewActive:
		if app.Lifecycle != model.JobLifecycleActive {
			return false
		}
	case ViewOffer:
		if app.Stage != model.JobStageOffer && !app.ReachedOffer {
			return false
		}
	case ViewClosed:
		if app.Lifecycle != model.JobLifecycleClosed {
			return false
		}
	}
	if q := strings.ToLower(strings.TrimSpace(f.Query)); q != "" {
		companyMatch := strings.Contains(strings.ToLower(app.CompanyName), q)
		positionMatch := strings.Contains(strings.ToLower(app.PositionName), q)
		if !companyMatch && !positionMatch {
			return false
		}
	}
	if f.Stage != "" && app.Stage != f.Stage {
		return false
	}
	if f.Outcome != "" && app.Outcome != f.Outcome {
		return false
	}
	if d := strings.TrimSpace(f.AppliedDate); d != "" && app.AppliedDate != d {
		return false
	}
	if from := strings.TrimSpace(f.AppliedFrom); from != "" && app.AppliedDate < from {
		return false
	}
	if to := strings.TrimSpace(f.AppliedTo); to != "" && app.AppliedDate > to {
		return false
	}
	return true
}

// FilterApplications filters apps in memory using f and returns a sorted slice.
func FilterApplications(apps []model.JobApplication, f Filter, sortBy JobSort) []model.JobApplication {
	out := make([]model.JobApplication, 0, len(apps))
	for _, app := range apps {
		if f.Matches(app) {
			out = append(out, app)
		}
	}
	SortApplications(out, sortBy)
	return out
}

// SortApplications sorts apps deterministically according to sortBy.
func SortApplications(apps []model.JobApplication, sortBy JobSort) {
	if sortBy == "" {
		sortBy = SortAppliedDesc
	}
	slices.SortFunc(apps, func(a, b model.JobApplication) int {
		switch sortBy {
		case SortAppliedAsc:
			if c := cmp.Compare(a.AppliedDate, b.AppliedDate); c != 0 {
				return c
			}
			if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
				return c
			}
		case SortUpdatedDesc:
			if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
				return c
			}
			if c := cmp.Compare(b.AppliedDate, a.AppliedDate); c != 0 {
				return c
			}
		case SortUpdatedAsc:
			if c := a.UpdatedAt.Compare(b.UpdatedAt); c != 0 {
				return c
			}
			if c := cmp.Compare(a.AppliedDate, b.AppliedDate); c != 0 {
				return c
			}
		default: // SortAppliedDesc
			if c := cmp.Compare(b.AppliedDate, a.AppliedDate); c != 0 {
				return c
			}
			if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
				return c
			}
		}
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

// ComputeStats calculates aggregate counts across all non-deleted applications.
func ComputeStats(apps []model.JobApplication) model.JobApplicationStats {
	var stats model.JobApplicationStats
	for _, app := range apps {
		if app.DeletedAt != nil {
			continue
		}
		stats.Total++
		switch app.Lifecycle {
		case model.JobLifecycleActive:
			stats.Active++
		case model.JobLifecycleClosed:
			stats.Closed++
		}
		if app.Stage == model.JobStageOffer || app.ReachedOffer {
			stats.Offered++
		}
	}
	return stats
}

// List returns all non-deleted job applications sorted by applied date descending.
func (s *Service) List(ctx context.Context) ([]model.JobApplication, error) {
	var out []model.JobApplication
	err := s.s.Read(ctx, func(ro *sql.Tx) error {
		rows, err := ro.QueryContext(ctx, selectApplicationColumns+` WHERE deleted_at IS NULL`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			app, err := scanApplication(rows)
			if err != nil {
				return err
			}
			out = append(out, app)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	SortApplications(out, SortAppliedDesc)
	return out, nil
}

// Stats returns aggregate counts across all non-deleted applications.
func (s *Service) Stats(ctx context.Context) (model.JobApplicationStats, error) {
	apps, err := s.List(ctx)
	if err != nil {
		return model.JobApplicationStats{}, err
	}
	return ComputeStats(apps), nil
}

// Get fetches one non-deleted job application by ID.
func (s *Service) Get(ctx context.Context, id string) (model.JobApplication, error) {
	var app model.JobApplication
	err := s.s.Read(ctx, func(ro *sql.Tx) error {
		var err error
		app, err = getApplicationTx(ctx, ro, id, false)
		return err
	})
	return app, err
}

// FindActiveDuplicates returns active non-deleted applications matching the same company and position (case-insensitive, trimmed).
func (s *Service) FindActiveDuplicates(ctx context.Context, companyName, positionName string) ([]model.JobApplication, error) {
	company := strings.TrimSpace(companyName)
	position := strings.TrimSpace(positionName)
	if company == "" || position == "" {
		return nil, nil
	}
	apps, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var dupes []model.JobApplication
	for _, app := range apps {
		if app.Lifecycle != model.JobLifecycleActive {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(app.CompanyName), company) &&
			strings.EqualFold(strings.TrimSpace(app.PositionName), position) {
			dupes = append(dupes, app)
		}
	}
	return dupes, nil
}

// Create validates and inserts a new active application along with its creation timeline event.
func (s *Service) Create(ctx context.Context, in CreateInput) (model.JobApplication, error) {
	company := strings.TrimSpace(in.CompanyName)
	position := strings.TrimSpace(in.PositionName)
	if company == "" {
		return model.JobApplication{}, fmt.Errorf("%w: 公司名称不能为空", model.ErrInvalid)
	}
	if position == "" {
		return model.JobApplication{}, fmt.Errorf("%w: 岗位名称不能为空", model.ErrInvalid)
	}
	appliedDate := strings.TrimSpace(in.AppliedDate)
	if appliedDate == "" {
		appliedDate = s.todayLocalDate()
	}
	if _, err := model.ParseCalendarDate(appliedDate); err != nil {
		return model.JobApplication{}, fmt.Errorf("%w: 投递日期格式无效", model.ErrInvalid)
	}
	stage := in.Stage
	if stage == "" {
		stage = model.JobStageApplied
	}
	if !stage.Valid() {
		return model.JobApplication{}, fmt.Errorf("%w: 无效的当前阶段 %q", model.ErrInvalid, stage)
	}
	appURL, err := model.ValidateJobURL(in.ApplicationURL)
	if err != nil {
		return model.JobApplication{}, err
	}
	followUp := strings.TrimSpace(in.NextFollowUpDate)
	if followUp != "" {
		if _, err := model.ParseCalendarDate(followUp); err != nil {
			return model.JobApplication{}, fmt.Errorf("%w: 下次跟进日期格式无效", model.ErrInvalid)
		}
	}
	channel := strings.TrimSpace(in.Channel)
	notes := strings.TrimRight(in.Notes, "\r\n")
	nowTS := s.timestamp()
	occurredTime, err := s.resolveEventTime(appliedDate)
	if err != nil {
		return model.JobApplication{}, err
	}
	occurredTS := occurredTime.Format(time.RFC3339Nano)
	appID := model.NewID()
	eventID := model.NewID()

	var created model.JobApplication
	err = s.s.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_applications(
			id, company_name, position_name, applied_date,
			stage, lifecycle, outcome, closure_reason, closed_at,
			application_url, channel, next_follow_up_date, notes,
			created_at, updated_at, deleted_at
		) VALUES(?,?,?,?,?,?,?,?,NULL,?,?,?,?,?,?,NULL)`,
			appID, company, position, appliedDate,
			string(stage), string(model.JobLifecycleActive), "", "",
			appURL, channel, followUp, notes,
			nowTS, nowTS,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_application_events(
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			eventID, appID, string(model.JobEventCreated), "", string(stage), "", "", "", occurredTS, nowTS,
		); err != nil {
			return err
		}
		var err error
		created, err = getApplicationTx(ctx, tx, appID, false)
		return err
	})
	return created, err
}

// UpdateDetails updates non-workflow metadata fields without modifying stage, lifecycle, outcome, or timeline history.
func (s *Service) UpdateDetails(ctx context.Context, in UpdateDetailsInput) (model.JobApplication, error) {
	company := strings.TrimSpace(in.CompanyName)
	position := strings.TrimSpace(in.PositionName)
	if company == "" {
		return model.JobApplication{}, fmt.Errorf("%w: 公司名称不能为空", model.ErrInvalid)
	}
	if position == "" {
		return model.JobApplication{}, fmt.Errorf("%w: 岗位名称不能为空", model.ErrInvalid)
	}
	appliedDate := strings.TrimSpace(in.AppliedDate)
	if _, err := model.ParseCalendarDate(appliedDate); err != nil {
		return model.JobApplication{}, fmt.Errorf("%w: 投递日期不能为空且须为 YYYY-MM-DD 格式", model.ErrInvalid)
	}
	appURL, err := model.ValidateJobURL(in.ApplicationURL)
	if err != nil {
		return model.JobApplication{}, err
	}
	var followUp *string
	if in.NextFollowUpDate != nil {
		trimmed := strings.TrimSpace(*in.NextFollowUpDate)
		if trimmed != "" {
			if _, err := model.ParseCalendarDate(trimmed); err != nil {
				return model.JobApplication{}, fmt.Errorf("%w: 下次跟进日期格式无效", model.ErrInvalid)
			}
		}
		followUp = &trimmed
	}
	channel := strings.TrimSpace(in.Channel)
	notes := strings.TrimRight(in.Notes, "\r\n")
	nowTS := s.timestamp()

	var updated model.JobApplication
	err = s.s.Write(ctx, func(tx *sql.Tx) error {
		cur, err := getApplicationTx(ctx, tx, in.ID, false)
		if err != nil {
			return err
		}
		nextFollowUp := cur.NextFollowUpDate
		if followUp != nil {
			nextFollowUp = *followUp
		}
		closureReason := cur.ClosureReason
		if in.ClosureReason != nil {
			trimmedReason := strings.TrimSpace(*in.ClosureReason)
			if cur.Lifecycle == model.JobLifecycleActive && trimmedReason != "" {
				return fmt.Errorf("%w: 进行中的记录不能设置结束原因", model.ErrInvalid)
			}
			if cur.Lifecycle == model.JobLifecycleClosed {
				closureReason = trimmedReason
			} else {
				closureReason = ""
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications
			SET company_name = ?, position_name = ?, applied_date = ?,
			    application_url = ?, channel = ?, next_follow_up_date = ?,
			    notes = ?, closure_reason = ?, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL`,
			company, position, appliedDate,
			appURL, channel, nextFollowUp,
			notes, closureReason, nowTS,
			in.ID,
		); err != nil {
			return err
		}
		updated, err = getApplicationTx(ctx, tx, in.ID, false)
		return err
	})
	return updated, err
}

// UpdateFollowUpDate updates only next_follow_up_date and updated_at without clobbering other fields or creating timeline events.
func (s *Service) UpdateFollowUpDate(ctx context.Context, id, followUpDate string) (model.JobApplication, error) {
	trimmed := strings.TrimSpace(followUpDate)
	if trimmed != "" {
		if _, err := model.ParseCalendarDate(trimmed); err != nil {
			return model.JobApplication{}, fmt.Errorf("%w: 下次跟进日期格式无效", model.ErrInvalid)
		}
	}
	nowTS := s.timestamp()
	var updated model.JobApplication
	err := s.s.Write(ctx, func(tx *sql.Tx) error {
		if _, err := getApplicationTx(ctx, tx, id, false); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications SET next_follow_up_date = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			trimmed, nowTS, id,
		); err != nil {
			return err
		}
		var err error
		updated, err = getApplicationTx(ctx, tx, id, false)
		return err
	})
	return updated, err
}

// ChangeStage atomically changes stage and records a stage_changed event.
// Changing stage on a closed application acts as a stage correction without automatically reopening lifecycle.
func (s *Service) ChangeStage(ctx context.Context, in ChangeStageInput) (model.JobApplication, error) {
	if !in.ToStage.Valid() {
		return model.JobApplication{}, fmt.Errorf("%w: 无效的阶段 %q", model.ErrInvalid, in.ToStage)
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = model.NewID()
	}
	note := strings.TrimSpace(in.Note)
	nowTS := s.timestamp()

	var updated model.JobApplication
	err := s.s.Write(ctx, func(tx *sql.Tx) error {
		cur, err := getApplicationTx(ctx, tx, in.ID, false)
		if err != nil {
			return err
		}
		if _, exists, err := findEventByIDTx(ctx, tx, in.ID, eventID); err != nil {
			return err
		} else if exists {
			updated = cur
			return nil
		}
		if cur.Stage == in.ToStage {
			updated = cur
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications SET stage = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			string(in.ToStage), nowTS, in.ID,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_application_events(
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			eventID, in.ID, string(model.JobEventStageChanged), string(cur.Stage), string(in.ToStage), "", "", note, nowTS, nowTS,
		); err != nil {
			return err
		}
		updated, err = getApplicationTx(ctx, tx, in.ID, false)
		return err
	})
	return updated, err
}

// Close closes an active application with outcome and optional reason in one atomic transaction.
func (s *Service) Close(ctx context.Context, in CloseInput) (model.JobApplication, error) {
	if !in.Outcome.Valid() {
		return model.JobApplication{}, fmt.Errorf("%w: 结束流程必须选择有效的结束结果", model.ErrInvalid)
	}
	if in.StageOverride != "" && !in.StageOverride.Valid() {
		return model.JobApplication{}, fmt.Errorf("%w: 无效的阶段 %q", model.ErrInvalid, in.StageOverride)
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = model.NewID()
	}
	reason := strings.TrimSpace(in.Reason)
	nowTS := s.timestamp()

	var updated model.JobApplication
	err := s.s.Write(ctx, func(tx *sql.Tx) error {
		cur, err := getApplicationTx(ctx, tx, in.ID, false)
		if err != nil {
			return err
		}
		if _, exists, err := findEventByIDTx(ctx, tx, in.ID, eventID); err != nil {
			return err
		} else if exists {
			updated = cur
			return nil
		}
		if cur.Lifecycle == model.JobLifecycleClosed {
			return fmt.Errorf("%w: 记录已处于结束状态", model.ErrConflict)
		}
		targetStage := cur.Stage
		if in.StageOverride != "" {
			targetStage = in.StageOverride
		}
		if in.Outcome.RequiresOfferStage() && targetStage != model.JobStageOffer {
			return fmt.Errorf("%w: 选择「%s」要求当前阶段为 Offer", model.ErrInvalid, in.Outcome.Label())
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications
			SET stage = ?, lifecycle = ?, outcome = ?, closure_reason = ?, closed_at = ?, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL`,
			string(targetStage), string(model.JobLifecycleClosed), string(in.Outcome), reason, nowTS, nowTS, in.ID,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_application_events(
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			eventID, in.ID, string(model.JobEventClosed), string(cur.Stage), string(targetStage), string(in.Outcome), reason, "", nowTS, nowTS,
		); err != nil {
			return err
		}
		updated, err = getApplicationTx(ctx, tx, in.ID, false)
		return err
	})
	return updated, err
}

// Reopen transitions a closed application back to active and clears current closure fields while keeping historical events.
func (s *Service) Reopen(ctx context.Context, in ReopenInput) (model.JobApplication, error) {
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = model.NewID()
	}
	note := strings.TrimSpace(in.Note)
	nowTS := s.timestamp()

	var updated model.JobApplication
	err := s.s.Write(ctx, func(tx *sql.Tx) error {
		cur, err := getApplicationTx(ctx, tx, in.ID, false)
		if err != nil {
			return err
		}
		if _, exists, err := findEventByIDTx(ctx, tx, in.ID, eventID); err != nil {
			return err
		} else if exists {
			updated = cur
			return nil
		}
		if cur.Lifecycle == model.JobLifecycleActive {
			updated = cur
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications
			SET lifecycle = ?, outcome = '', closure_reason = '', closed_at = NULL, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL`,
			string(model.JobLifecycleActive), nowTS, in.ID,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_application_events(
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			eventID, in.ID, string(model.JobEventReopened), string(cur.Stage), string(cur.Stage), string(cur.Outcome), cur.ClosureReason, note, nowTS, nowTS,
		); err != nil {
			return err
		}
		updated, err = getApplicationTx(ctx, tx, in.ID, false)
		return err
	})
	return updated, err
}

// ListEvents returns all timeline events for an active (non-deleted) application in chronological order.
func (s *Service) ListEvents(ctx context.Context, applicationID string) ([]model.JobApplicationEvent, error) {
	var events []model.JobApplicationEvent
	err := s.s.Read(ctx, func(ro *sql.Tx) error {
		if _, err := getApplicationTx(ctx, ro, applicationID, false); err != nil {
			return err
		}
		rows, err := ro.QueryContext(ctx, `SELECT
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
			FROM job_application_events
			WHERE application_id = ?`, applicationID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			ev, err := scanEvent(rows)
			if err != nil {
				return err
			}
			events = append(events, ev)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(events, func(a, b model.JobApplicationEvent) int {
		if c := a.OccurredAt.Compare(b.OccurredAt); c != 0 {
			return c
		}
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return events, nil
}

// AddNoteEvent appends a manual note event to an application's timeline.
func (s *Service) AddNoteEvent(ctx context.Context, in AddNoteInput) (model.JobApplicationEvent, error) {
	note := strings.TrimSpace(in.Note)
	if note == "" {
		return model.JobApplicationEvent{}, fmt.Errorf("%w: 时间线备注不能为空", model.ErrInvalid)
	}
	occurredTime, err := s.resolveEventTime(in.OccurredDate)
	if err != nil {
		return model.JobApplicationEvent{}, err
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = model.NewID()
	}
	nowTS := s.timestamp()
	occurredTS := occurredTime.Format(time.RFC3339Nano)

	var created model.JobApplicationEvent
	err = s.s.Write(ctx, func(tx *sql.Tx) error {
		if _, err := getApplicationTx(ctx, tx, in.ApplicationID, false); err != nil {
			return err
		}
		if existing, exists, err := findEventByIDTx(ctx, tx, in.ApplicationID, eventID); err != nil {
			return err
		} else if exists {
			created = existing
			return nil
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_application_events(
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			eventID, in.ApplicationID, string(model.JobEventNoteAdded), "", "", "", "", note, occurredTS, nowTS,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications SET updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			nowTS, in.ApplicationID,
		); err != nil {
			return err
		}
		var err error
		created, _, err = findEventByIDTx(ctx, tx, in.ApplicationID, eventID)
		return err
	})
	return created, err
}

// BackfillStageEvent appends a historical stage event with a calendar date without mutating current stage or lifecycle.
func (s *Service) BackfillStageEvent(ctx context.Context, in BackfillStageInput) (model.JobApplicationEvent, error) {
	if !in.Stage.Valid() {
		return model.JobApplicationEvent{}, fmt.Errorf("%w: 无效的补录阶段 %q", model.ErrInvalid, in.Stage)
	}
	dateStr := strings.TrimSpace(in.OccurredDate)
	if _, err := model.ParseCalendarDate(dateStr); err != nil {
		return model.JobApplicationEvent{}, fmt.Errorf("%w: 补录日期不能为空且须为 YYYY-MM-DD 格式", model.ErrInvalid)
	}
	occurredTime, err := s.resolveEventTime(dateStr)
	if err != nil {
		return model.JobApplicationEvent{}, err
	}
	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = model.NewID()
	}
	note := strings.TrimSpace(in.Note)
	nowTS := s.timestamp()
	occurredTS := occurredTime.Format(time.RFC3339Nano)

	var created model.JobApplicationEvent
	err = s.s.Write(ctx, func(tx *sql.Tx) error {
		if _, err := getApplicationTx(ctx, tx, in.ApplicationID, false); err != nil {
			return err
		}
		if existing, exists, err := findEventByIDTx(ctx, tx, in.ApplicationID, eventID); err != nil {
			return err
		} else if exists {
			created = existing
			return nil
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_application_events(
			id, application_id, event_type, from_stage, to_stage, outcome, reason, note, occurred_at, created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			eventID, in.ApplicationID, string(model.JobEventStageBackfilled), "", string(in.Stage), "", "", note, occurredTS, nowTS,
		); err != nil {
			return err
		}
		var err error
		created, _, err = findEventByIDTx(ctx, tx, in.ApplicationID, eventID)
		return err
	})
	return created, err
}

// UpdateEventNote updates the note attached to an existing timeline event for an active application.
func (s *Service) UpdateEventNote(ctx context.Context, in UpdateEventNoteInput) (model.JobApplicationEvent, error) {
	appID := strings.TrimSpace(in.ApplicationID)
	eventID := strings.TrimSpace(in.EventID)
	if appID == "" || eventID == "" {
		return model.JobApplicationEvent{}, fmt.Errorf("%w: 投递记录与事件标识不能为空", model.ErrInvalid)
	}
	note := strings.TrimSpace(in.Note)
	nowTS := s.timestamp()

	var updated model.JobApplicationEvent
	err := s.s.Write(ctx, func(tx *sql.Tx) error {
		if _, err := getApplicationTx(ctx, tx, appID, false); err != nil {
			return err
		}
		existing, exists, err := findEventByIDTx(ctx, tx, appID, eventID)
		if err != nil {
			return err
		}
		if !exists {
			return model.ErrNotFound
		}
		if existing.EventType == model.JobEventNoteAdded && note == "" {
			return fmt.Errorf("%w: 时间线备注不能为空", model.ErrInvalid)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_application_events SET note = ? WHERE id = ? AND application_id = ?`,
			note, eventID, appID,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications SET updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			nowTS, appID,
		); err != nil {
			return err
		}
		updated, _, err = findEventByIDTx(ctx, tx, appID, eventID)
		return err
	})
	return updated, err
}

// DeleteEvent deletes a timeline event (notes, backfilled stages, or accidental transitions/closures/reopenings) while preserving the initial created event.
func (s *Service) DeleteEvent(ctx context.Context, applicationID, eventID string) error {
	appID := strings.TrimSpace(applicationID)
	evID := strings.TrimSpace(eventID)
	if appID == "" || evID == "" {
		return fmt.Errorf("%w: 投递记录与事件标识不能为空", model.ErrInvalid)
	}
	nowTS := s.timestamp()

	return s.s.Write(ctx, func(tx *sql.Tx) error {
		if _, err := getApplicationTx(ctx, tx, appID, false); err != nil {
			return err
		}
		existing, exists, err := findEventByIDTx(ctx, tx, appID, evID)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		if existing.EventType == model.JobEventCreated {
			return fmt.Errorf("%w: 初始创建节点不可删除", model.ErrInvalid)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM job_application_events WHERE id = ? AND application_id = ?`,
			evID, appID,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job_applications SET updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
			nowTS, appID,
		); err != nil {
			return err
		}
		return nil
	})
}
