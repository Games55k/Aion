package backup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aion/backend/jobapplications"
	"aion/backend/model"
)

func TestExportAndRestoreWithJobApplications(t *testing.T) {
	// Restore must preserve the Offer outcome even after the user corrects the current stage.
	source := testStore(t)
	seed(t, source, "proj-1", "attachment-payload")
	jobs := jobapplications.New(source)
	app, err := jobs.Create(testCtx, jobapplications.CreateInput{
		CompanyName:    "未来科技",
		PositionName:   "Go 工程师",
		AppliedDate:    "2026-11-01",
		ApplicationURL: "https://jobs.example.com/1",
		Channel:        "内推",
		Notes:          "重点岗位",
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err = jobs.ChangeStage(testCtx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageOffer})
	if err != nil {
		t.Fatal(err)
	}
	app, err = jobs.Close(testCtx, jobapplications.CloseInput{
		ID:      app.ID,
		Outcome: model.JobOutcomeOfferAccepted,
		Reason:  "已签约",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err = jobs.ChangeStage(testCtx, jobapplications.ChangeStageInput{ID: app.ID, ToStage: model.JobStageHR})
	if err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "jobs-backup.zip")
	if err := New(source).Export(testCtx, archive); err != nil {
		t.Fatal(err)
	}

	target := testStore(t)
	if _, err := New(target).Restore(testCtx, archive); err != nil {
		t.Fatal(err)
	}
	assertSeed(t, target, "proj-1", "attachment-payload")
	restoredJobs := jobapplications.New(target)
	got, err := restoredJobs.Get(testCtx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CompanyName != "未来科技" || got.Stage != model.JobStageHR || got.Outcome != model.JobOutcomeOfferAccepted || got.ClosureReason != "已签约" || !got.ReachedOffer {
		t.Fatalf("restored job application mismatch: %+v", got)
	}
	evs, err := restoredJobs.ListEvents(testCtx, app.ID)
	if err != nil || len(evs) != 4 {
		t.Fatalf("restored events = %+v, err=%v", evs, err)
	}
}

// Legacy backups are rejected before creating a safety archive or changing live data.
func TestRestoreRejectsLegacyDatabaseSchemas(t *testing.T) {
	archive, _ := exportFixture(t)
	for _, tc := range []struct {
		name    string
		version int
		ddl     string
	}{
		{"v1", 1, "DROP TABLE job_application_events; DROP TABLE job_applications; PRAGMA user_version=1"},
		{"v2", 2, "ALTER TABLE job_applications ADD COLUMN next_follow_up_date TEXT NOT NULL DEFAULT ''; PRAGMA user_version=2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := zipItems(t, archive)
			updateDB(t, items, tc.ddl)
			editManifest(t, items, func(m *manifest) { m.SchemaVersion = tc.version })
			live := testStore(t)
			seed(t, live, "live", "keep current data")
			result, err := New(live).Restore(testCtx, writeItems(t, items))
			if !errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), "不支持的备份数据库版本") {
				t.Fatalf("legacy archive error = %v", err)
			}
			if result.SafetyBackupPath != "" {
				t.Fatalf("unsupported archive reached the replacement step: %+v", result)
			}
			if _, err := os.Stat(filepath.Join(live.Root(), "backups")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unsupported archive reached the safety backup step: %v", err)
			}
			assertSeed(t, live, "live", "keep current data")
		})
	}
}

// Invalid job data and a manifest that misstates its database version are rejected before restore.
func TestRejectInvalidJobBackupsWithoutTouchingLive(t *testing.T) {
	live := testStore(t)
	seed(t, live, "survive", "stay-safe")
	liveJobs := jobapplications.New(live)
	liveApp, err := liveJobs.Create(testCtx, jobapplications.CreateInput{
		CompanyName:  "守卫科技",
		PositionName: "后端工程师",
		AppliedDate:  "2026-11-15",
	})
	if err != nil {
		t.Fatal(err)
	}

	source := testStore(t)
	srcJobs := jobapplications.New(source)
	srcApp, err := srcJobs.Create(testCtx, jobapplications.CreateInput{
		CompanyName:  "合法公司",
		PositionName: "工程师",
		AppliedDate:  "2026-11-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "base-jobs.zip")
	if err := New(source).Export(testCtx, archive); err != nil {
		t.Fatal(err)
	}

	invalidMutations := []struct {
		name string
		sql  string
	}{
		{"blank company", `UPDATE job_applications SET company_name='   ' WHERE id='` + srcApp.ID + `'`},
		{"bad applied_date", `UPDATE job_applications SET applied_date='2026-02-30' WHERE id='` + srcApp.ID + `'`},
		{"bad application_url", `UPDATE job_applications SET application_url='javascript:alert(1)' WHERE id='` + srcApp.ID + `'`},
		{"misstated database version", "PRAGMA user_version=2"},
		{"bad event timestamp", `UPDATE job_application_events SET occurred_at='bad-ts' WHERE application_id='` + srcApp.ID + `'`},
	}
	for _, tc := range invalidMutations {
		items := zipItems(t, archive)
		updateDB(t, items, tc.sql)
		if _, err := New(live).Restore(testCtx, writeItems(t, items)); !errors.Is(err, model.ErrInvalid) {
			t.Fatalf("%s: got err=%v, want ErrInvalid", tc.name, err)
		}
	}

	// Live data must remain completely intact after all rejected restores.
	assertSeed(t, live, "survive", "stay-safe")
	stillHere, err := liveJobs.Get(testCtx, liveApp.ID)
	if err != nil || stillHere.CompanyName != "守卫科技" {
		t.Fatalf("live job damaged by rejected restores: %+v err=%v", stillHere, err)
	}
}
