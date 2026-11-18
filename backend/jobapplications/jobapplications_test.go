package jobapplications_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/store"
)

func openTestService(t *testing.T, clock *time.Time) (*jobapplications.Service, *store.Store) {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	svc := jobapplications.New(s, jobapplications.Options{
		Now: func() time.Time {
			*clock = clock.Add(time.Second)
			return *clock
		},
	})
	return svc, s
}

func TestValidationAndDefaults(t *testing.T) {
	// Creating and updating job applications must validate required fields, dates, HTTP/HTTPS URLs, and apply local-date defaults.
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 9, 30, 0, 0, time.Local)
	svc, _ := openTestService(t, &now)

	cases := []struct {
		name string
		in   jobapplications.CreateInput
	}{
		{"blank company", jobapplications.CreateInput{CompanyName: "  ", PositionName: "Go", AppliedDate: "2026-11-15"}},
		{"blank position", jobapplications.CreateInput{CompanyName: "Acme", PositionName: "\t", AppliedDate: "2026-11-15"}},
		{"invalid applied date", jobapplications.CreateInput{CompanyName: "Acme", PositionName: "Go", AppliedDate: "2026-02-30"}},
		{"javascript URL", jobapplications.CreateInput{CompanyName: "Acme", PositionName: "Go", ApplicationURL: "javascript:alert(1)"}},
		{"file URL", jobapplications.CreateInput{CompanyName: "Acme", PositionName: "Go", ApplicationURL: "file:///etc/passwd"}},
		{"missing host URL", jobapplications.CreateInput{CompanyName: "Acme", PositionName: "Go", ApplicationURL: "https:///jobs/1"}},
		{"invalid stage", jobapplications.CreateInput{CompanyName: "Acme", PositionName: "Go", Stage: "unknown"}},
	}
	for _, tc := range cases {
		if _, err := svc.Create(ctx, tc.in); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("%s: got err=%v, want ErrInvalid", tc.name, err)
		}
	}

	created, err := svc.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "  示例科技  ",
		PositionName:   "  Go 后端工程师 ",
		ApplicationURL: " https://jobs.example.com/123 ",
		Channel:        " 官网 ",
	})
	if err != nil {
		t.Fatalf("create valid app: %v", err)
	}
	if created.CompanyName != "示例科技" || created.PositionName != "Go 后端工程师" {
		t.Fatalf("expected trimmed company/position, got %+v", created)
	}
	if created.AppliedDate != "2026-11-15" || created.Stage != model.JobStageApplied || created.Lifecycle != model.JobLifecycleActive {
		t.Fatalf("unexpected defaults: %+v", created)
	}
	if created.Outcome != "" || created.ClosureReason != "" || created.ClosedAt != nil {
		t.Fatalf("active application must not contain closure fields: %+v", created)
	}
}

func TestActiveDuplicateDetectionDoesNotBlockMultiRoleOrReapply(t *testing.T) {
	// Active duplicate queries must warn on matching active roles without blocking multi-role or closed re-applications.
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 9, 30, 0, 0, time.Local)
	svc, _ := openTestService(t, &now)

	first, err := svc.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "Example Corp",
		PositionName: "Backend Engineer",
		AppliedDate:  "2026-11-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	dupes, err := svc.FindActiveDuplicates(ctx, " example corp ", "BACKEND ENGINEER")
	if err != nil || len(dupes) != 1 || dupes[0].ID != first.ID {
		t.Fatalf("FindActiveDuplicates = %+v, err=%v", dupes, err)
	}

	// Different position at the same company has no duplicate warning and creates independently.
	otherRoleDupes, err := svc.FindActiveDuplicates(ctx, "Example Corp", "Platform Engineer")
	if err != nil || len(otherRoleDupes) != 0 {
		t.Fatalf("unexpected dupes for different role: %+v, err=%v", otherRoleDupes, err)
	}
	second, err := svc.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "Example Corp",
		PositionName: "Backend Engineer",
		AppliedDate:  "2026-11-10",
	})
	if err != nil {
		t.Fatalf("duplicate company+position must not be rejected by DB constraints: %v", err)
	}

	// Closing the original records removes them from active duplicate matches.
	if _, err := svc.Close(ctx, jobapplications.CloseInput{ID: first.ID, Outcome: model.JobOutcomeRejected}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Close(ctx, jobapplications.CloseInput{ID: second.ID, Outcome: model.JobOutcomeWithdrawn}); err != nil {
		t.Fatal(err)
	}
	dupesAfterClose, err := svc.FindActiveDuplicates(ctx, "Example Corp", "Backend Engineer")
	if err != nil || len(dupesAfterClose) != 0 {
		t.Fatalf("closed roles must not trigger active duplicate warning: %+v, err=%v", dupesAfterClose, err)
	}
}

func TestStageLifecycleSeparationCloseReopenAndOfferRules(t *testing.T) {
	// Stages skip/reverse independently of lifecycle, Offer outcomes enforce Offer stage, and reopen clears closure fields while keeping history.
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 9, 30, 0, 0, time.Local)
	svc, _ := openTestService(t, &now)

	app, err := svc.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "星云软件",
		PositionName: "资深后端",
		AppliedDate:  "2026-11-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Skip directly to interview_2 then move back to assessment.
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageInterview2})
	if err != nil {
		t.Fatal(err)
	}
	if app.Stage != model.JobStageInterview2 || app.Lifecycle != model.JobLifecycleActive {
		t.Fatalf("unexpected state after stage jump: %+v", app)
	}
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageAssessment})
	if err != nil {
		t.Fatal(err)
	}
	// Advance back to interview_2, then reject closing with offer_accepted until stage is Offer.
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageInterview2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Close(ctx, jobapplications.CloseInput{
		ID:      app.ID,
		Outcome: model.JobOutcomeOfferAccepted,
	}); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("closing as offer_accepted from interview_2 got err=%v, want ErrInvalid", err)
	}

	// Reaching Offer marks ReachedOffer, and changing stage back away from Offer on an active row clears ReachedOffer.
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageOffer})
	if err != nil {
		t.Fatal(err)
	}
	if app.Lifecycle != model.JobLifecycleActive || !app.ReachedOffer {
		t.Fatalf("reaching Offer must keep lifecycle active and mark ReachedOffer: %+v", app)
	}
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageScreening})
	if err != nil {
		t.Fatal(err)
	}
	if app.ReachedOffer {
		t.Fatalf("moving an active row away from Offer must clear ReachedOffer: %+v", app)
	}
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageOffer})
	if err != nil {
		t.Fatal(err)
	}

	// Close at Offer stage with offer_declined + reason.
	app, err = svc.Close(ctx, jobapplications.CloseInput{
		ID:      app.ID,
		Outcome: model.JobOutcomeOfferDeclined,
		Reason:  "薪资总包低于预期",
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.Stage != model.JobStageOffer || app.Lifecycle != model.JobLifecycleClosed || app.Outcome != model.JobOutcomeOfferDeclined || app.ClosedAt == nil {
		t.Fatalf("unexpected closed state: %+v", app)
	}

	// Adjusting stage on a closed offer_declined record acts as a correction without auto-reopening and keeps Offer outcome status.
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageHR})
	if err != nil {
		t.Fatal(err)
	}
	if app.Stage != model.JobStageHR || app.Lifecycle != model.JobLifecycleClosed || !app.ReachedOffer {
		t.Fatalf("closed stage correction must stay closed and retain Offer outcome status: %+v", app)
	}

	// Updating ClosureReason on closed record succeeds without adding events, but non-empty reason on active record is rejected.
	newReason := "已婉拒并保留联系"
	app, err = svc.UpdateDetails(ctx, jobapplications.UpdateDetailsInput{
		ID:            app.ID,
		CompanyName:   app.CompanyName,
		PositionName:  app.PositionName,
		AppliedDate:   app.AppliedDate,
		ClosureReason: &newReason,
	})
	if err != nil || app.ClosureReason != newReason {
		t.Fatalf("update closure reason on closed record failed: %+v, err=%v", app, err)
	}

	// Explicit Reopen clears current outcome, closure_reason, closed_at, and ReachedOffer when stage is not Offer, preserving historical events.
	app, err = svc.Reopen(ctx, jobapplications.ReopenInput{ID: app.ID, Note: "团队加预算重新联系"})
	if err != nil {
		t.Fatal(err)
	}
	if app.Lifecycle != model.JobLifecycleActive || app.Outcome != "" || app.ClosureReason != "" || app.ClosedAt != nil || app.Stage != model.JobStageHR || app.ReachedOffer {
		t.Fatalf("reopen did not clear current closure/Offer fields properly: %+v", app)
	}
	if _, err := svc.UpdateDetails(ctx, jobapplications.UpdateDetailsInput{
		ID:            app.ID,
		CompanyName:   app.CompanyName,
		PositionName:  app.PositionName,
		AppliedDate:   app.AppliedDate,
		ClosureReason: &newReason,
	}); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("active application with non-empty closure reason got err=%v, want ErrInvalid", err)
	}

	// StageOverride during Close atomically advances to Offer and closes in one event.
	app, err = svc.Close(ctx, jobapplications.CloseInput{
		ID:            app.ID,
		Outcome:       model.JobOutcomeOfferAccepted,
		Reason:        "最终签约",
		StageOverride: model.JobStageOffer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.Stage != model.JobStageOffer || app.Lifecycle != model.JobLifecycleClosed || app.Outcome != model.JobOutcomeOfferAccepted {
		t.Fatalf("close with StageOverride failed: %+v", app)
	}

	events, err := svc.ListEvents(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	var closedSnapshotPreserved bool
	for _, ev := range events {
		if ev.EventType == model.JobEventClosed && ev.Outcome == model.JobOutcomeOfferDeclined && ev.Reason == "薪资总包低于预期" {
			closedSnapshotPreserved = true
		}
		if ev.EventType == model.JobEventReopened && ev.Outcome == model.JobOutcomeOfferDeclined && ev.Reason == "已婉拒并保留联系" {
			// Reopened event carries prior closure snapshot as well.
		}
	}
	if !closedSnapshotPreserved {
		t.Fatalf("original closed event snapshot lost after reopen: %+v", events)
	}
}

func TestTimelineOrderingBackfillDeduplicationAndFieldIsolation(t *testing.T) {
	// Text auto-saves and historical backfills must not overwrite workflow state or duplicate events across retries.
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.Local)
	svc, _ := openTestService(t, &now)

	app, err := svc.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "示例科技",
		PositionName: "Go 后端",
		AppliedDate:  "2026-11-01",
		Stage:        model.JobStageInterview3,
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err = svc.UpdateDetails(ctx, jobapplications.UpdateDetailsInput{
		ID:           app.ID,
		CompanyName:  app.CompanyName,
		PositionName: app.PositionName,
		AppliedDate:  app.AppliedDate,
		Channel:      "内推",
		Notes:        "多行备注第一版\n第二版自动保存",
	})
	if err != nil || app.Stage != model.JobStageInterview3 || app.Channel != "内推" {
		t.Fatalf("UpdateDetails clobbered stage or failed to save text: %+v, err=%v", app, err)
	}

	// Retrying ChangeStage with identical EventID must not insert a duplicate timeline event.
	opID := model.NewID()
	app, err = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageHR, EventID: opID})
	if err != nil {
		t.Fatal(err)
	}
	appRetry, err := svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageHR, EventID: opID})
	if err != nil || appRetry.Stage != model.JobStageHR {
		t.Fatalf("idempotent stage retry failed: %+v, err=%v", appRetry, err)
	}

	// Backfill an older stage event without regressing current Stage (still HR) or lifecycle.
	bfID := model.NewID()
	if _, err := svc.BackfillStageEvent(ctx, jobapplications.BackfillStageInput{
		ApplicationID: app.ID,
		Stage:         model.JobStageInterview1,
		OccurredDate:  "2026-11-05",
		Note:          "一面补录",
		EventID:       bfID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.BackfillStageEvent(ctx, jobapplications.BackfillStageInput{
		ApplicationID: app.ID,
		Stage:         model.JobStageInterview1,
		OccurredDate:  "2026-11-05",
		Note:          "一面补录",
		EventID:       bfID,
	}); err != nil {
		t.Fatal(err)
	}

	// Add manual note event.
	noteID := model.NewID()
	if _, err := svc.AddNoteEvent(ctx, jobapplications.AddNoteInput{
		ApplicationID: app.ID,
		Note:          "联系了 HR 确认进度",
		OccurredDate:  "2026-11-12",
		EventID:       noteID,
	}); err != nil {
		t.Fatal(err)
	}

	afterApp, err := svc.Get(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterApp.Stage != model.JobStageHR || afterApp.Lifecycle != model.JobLifecycleActive {
		t.Fatalf("backfill mutated current workflow state: %+v", afterApp)
	}

	events, err := svc.ListEvents(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("expected 4 distinct events (created, backfill, note, stage_changed), got %d: %+v", len(events), events)
	}
	if events[0].EventType != model.JobEventCreated ||
		events[1].EventType != model.JobEventStageBackfilled ||
		events[2].EventType != model.JobEventNoteAdded ||
		events[3].EventType != model.JobEventStageChanged {
		t.Fatalf("timeline events out of chronological order: %+v", events)
	}

	// UpdateEventNote updates manual note & stage-change notes while rejecting blank note_added values.
	if _, err := svc.UpdateEventNote(ctx, jobapplications.UpdateEventNoteInput{
		ApplicationID: app.ID,
		EventID:       noteID,
		Note:          "   ",
	}); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("expected ErrInvalid clearing note_added text, got %v", err)
	}
	updatedNote, err := svc.UpdateEventNote(ctx, jobapplications.UpdateEventNoteInput{
		ApplicationID: app.ID,
		EventID:       noteID,
		Note:          "已更新：HR 约定下周二二面",
	})
	if err != nil || updatedNote.Note != "已更新：HR 约定下周二二面" {
		t.Fatalf("UpdateEventNote on note event failed: %+v err=%v", updatedNote, err)
	}
	updatedStageEv, err := svc.UpdateEventNote(ctx, jobapplications.UpdateEventNoteInput{
		ApplicationID: app.ID,
		EventID:       events[3].ID,
		Note:          "HR 面补充说明",
	})
	if err != nil || updatedStageEv.Note != "HR 面补充说明" {
		t.Fatalf("UpdateEventNote on stage event failed: %+v err=%v", updatedStageEv, err)
	}

	// DeleteEvent removes non-created timeline events idempotently while protecting the initial created event and keeping current workflow state intact.
	if err := svc.DeleteEvent(ctx, app.ID, events[0].ID); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("expected ErrInvalid deleting created event, got %v", err)
	}
	if err := svc.DeleteEvent(ctx, app.ID, events[3].ID); err != nil {
		t.Fatalf("DeleteEvent on stage_changed event failed: %v", err)
	}
	if err := svc.DeleteEvent(ctx, app.ID, noteID); err != nil {
		t.Fatalf("DeleteEvent on note event failed: %v", err)
	}
	if err := svc.DeleteEvent(ctx, app.ID, bfID); err != nil {
		t.Fatalf("DeleteEvent on backfill event failed: %v", err)
	}
	if err := svc.DeleteEvent(ctx, app.ID, bfID); err != nil {
		t.Fatalf("DeleteEvent idempotency failed: %v", err)
	}
	afterDelete, err := svc.ListEvents(ctx, app.ID)
	if err != nil || len(afterDelete) != 1 || afterDelete[0].ID != events[0].ID {
		t.Fatalf("expected only created event remaining after deletes, err=%v events=%+v", err, afterDelete)
	}
	afterDelApp, err := svc.Get(ctx, app.ID)
	if err != nil || afterDelApp.Stage != model.JobStageHR || afterDelApp.Lifecycle != model.JobLifecycleActive {
		t.Fatalf("event deletion mutated current workflow state: app=%+v err=%v", afterDelApp, err)
	}
}

func TestFilterSortStatsAndCascadeDelete(t *testing.T) {
	// Filters intersect with AND, stats exclude deleted rows and count same-company rows separately, and permanent delete cascades events.
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 8, 0, 0, 0, time.Local)
	svc, s := openTestService(t, &now)

	a1, _ := svc.Create(ctx, jobapplications.CreateInput{CompanyName: "Alpha", PositionName: "Go", AppliedDate: "2026-11-03", Stage: model.JobStageOffer})
	a2, _ := svc.Create(ctx, jobapplications.CreateInput{CompanyName: "Alpha", PositionName: "Rust", AppliedDate: "2026-11-05"})
	a3, _ := svc.Create(ctx, jobapplications.CreateInput{CompanyName: "Beta", PositionName: "Go Infra", AppliedDate: "2026-11-03"})
	a4, _ := svc.Create(ctx, jobapplications.CreateInput{CompanyName: "Gamma", PositionName: "SRE", AppliedDate: "2026-11-01"})

	a3, _ = svc.ChangeStage(ctx, jobapplications.ChangeStageInput{ID: a3.ID, ToStage: model.JobStageInterview2})
	a3, _ = svc.Close(ctx, jobapplications.CloseInput{ID: a3.ID, Outcome: model.JobOutcomeRejected, Reason: "Headcount frozen"})
	_ = svc.Delete(ctx, a4.ID)

	all, err := svc.List(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("List = %d items, err=%v", len(all), err)
	}
	stats := jobapplications.ComputeStats(all)
	if stats.Total != 3 || stats.Active != 2 || stats.Offered != 1 || stats.Closed != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	// Intersect ViewAll + query "Go" + AppliedDate "2026-11-03" + Outcome rejected -> only a3.
	filtered := jobapplications.FilterApplications(all, jobapplications.Filter{
		View:        jobapplications.ViewAll,
		Query:       "go",
		AppliedDate: "2026-11-03",
		Outcome:     model.JobOutcomeRejected,
	}, jobapplications.SortAppliedDesc)
	if len(filtered) != 1 || filtered[0].ID != a3.ID {
		t.Fatalf("AND intersection failed: %+v", filtered)
	}

	// Default AppliedDesc puts a2 (11-05) first, then tie-broken 11-03 rows with most recently updated (a3) before a1.
	if len(all) != 3 || all[0].ID != a2.ID || all[1].ID != a3.ID || all[2].ID != a1.ID {
		t.Fatalf("unexpected SortAppliedDesc order: [%s %s %s]", all[0].ID, all[1].ID, all[2].ID)
	}

	// Soft-deleted record stays absent from List & Stats, restores with full timeline, and permanent delete cascades events.
	if err := svc.Restore(ctx, a4.ID); err != nil {
		t.Fatal(err)
	}
	evs, err := svc.ListEvents(ctx, a4.ID)
	if err != nil || len(evs) != 1 {
		t.Fatalf("restored events missing: %+v, err=%v", evs, err)
	}
	if err := svc.Delete(ctx, a4.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.PermanentlyDelete(ctx, a4.ID); err != nil {
		t.Fatal(err)
	}
	var leftoverEvents int
	if err := s.Read(ctx, func(ro *sql.Tx) error {
		return ro.QueryRowContext(ctx, `SELECT COUNT(*) FROM job_application_events WHERE application_id = ?`, a4.ID).Scan(&leftoverEvents)
	}); err != nil {
		t.Fatal(err)
	}
	if leftoverEvents != 0 {
		t.Fatalf("permanent delete left %d orphaned events", leftoverEvents)
	}
}
