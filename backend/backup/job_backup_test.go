package backup

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/store"
)

func buildV1BackupArchive(t *testing.T, createdAt time.Time, mutate func(*sql.DB)) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "v1.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplySchemaUpTo(testCtx, db, store.MinSchemaVersion); err != nil {
		db.Close()
		t.Fatal(err)
	}
	ts := createdAt.UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO projects(id,name,icon,description,archived,created_at,updated_at) VALUES('v1-proj','旧版项目','folder','v1 backup',0,?,?)`, ts, ts); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,status,priority,tags,subtasks,due_date,position,created_at,updated_at) VALUES('v1-task','v1-proj','旧版任务','','planned','medium','[]','[]','2026-12-01',0,?,?)`, ts, ts); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(db)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	dbBytes, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(dbBytes)
	m := manifest{
		Version:       FormatVersion,
		SchemaVersion: store.MinSchemaVersion,
		CreatedAt:     createdAt.UTC(),
		Kind:          "manual",
		Files: []fileRecord{{
			Path:   "database.sqlite",
			Size:   int64(len(dbBytes)),
			SHA256: hex.EncodeToString(sum[:]),
		}},
	}
	manifestBytes, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return writeItems(t, []zipItem{
		{name: "manifest.json", mode: 0600, data: manifestBytes},
		{name: "database.sqlite", mode: 0600, data: dbBytes},
	})
}

func TestExportAndRestoreV2WithJobApplications(t *testing.T) {
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

	archive := filepath.Join(t.TempDir(), "v2-backup.zip")
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

func TestRestoreValidV1BackupUpgradesToV2AndPreservesSafetyBackup(t *testing.T) {
	// Restoring a valid v1 backup upgrades it in staging, replaces the live workspace cleanly, and retains a safety backup of prior v2 jobs.
	v1Archive := buildV1BackupArchive(t, time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC), nil)

	target := testStore(t)
	seed(t, target, "live-proj", "live-bytes")
	targetJobs := jobapplications.New(target)
	liveJob, err := targetJobs.Create(testCtx, jobapplications.CreateInput{
		CompanyName:  "现有公司",
		PositionName: "架构师",
		AppliedDate:  "2026-11-12",
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := New(target).Restore(testCtx, v1Archive)
	if err != nil {
		t.Fatalf("restore v1 archive failed: %v", err)
	}

	// Restored store is now schema v2 containing only v1 data and an empty job_applications table.
	var userVersion int
	var projName, taskTitle string
	if err := target.Read(testCtx, func(ro *sql.Tx) error {
		if err := ro.QueryRow("PRAGMA user_version").Scan(&userVersion); err != nil {
			return err
		}
		if err := ro.QueryRow("SELECT name FROM projects WHERE id='v1-proj'").Scan(&projName); err != nil {
			return err
		}
		return ro.QueryRow("SELECT title FROM tasks WHERE id='v1-task'").Scan(&taskTitle)
	}); err != nil {
		t.Fatal(err)
	}
	if userVersion != store.SchemaVersion || projName != "旧版项目" || taskTitle != "旧版任务" {
		t.Fatalf("unexpected upgraded restore state: version=%d proj=%q task=%q", userVersion, projName, taskTitle)
	}
	restoredApps, err := targetJobs.List(testCtx)
	if err != nil || len(restoredApps) != 0 {
		t.Fatalf("restoring v1 archive should clear v2 job applications, got %+v err=%v", restoredApps, err)
	}

	// Safety backup created during restore allows recovering the pre-restore job application.
	if _, err := New(target).Restore(testCtx, res.SafetyBackupPath); err != nil {
		t.Fatalf("restore safety backup failed: %v", err)
	}
	recovered, err := targetJobs.Get(testCtx, liveJob.ID)
	if err != nil || recovered.CompanyName != "现有公司" {
		t.Fatalf("safety backup did not preserve pre-restore job application: %+v err=%v", recovered, err)
	}
}

func TestRejectCorruptV1AndInvalidV2JobBackupsWithoutTouchingLive(t *testing.T) {
	// Corrupt v1 archives, manifest/db version mismatches, and invalid job_applications values must be rejected before modifying live data.
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

	// Case 1: v1 archive with invalid timestamp inside v1 table.
	corruptV1 := buildV1BackupArchive(t, time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC), func(db *sql.DB) {
		_, err := db.Exec(`UPDATE projects SET created_at='not-rfc3339'`)
		if err != nil {
			t.Fatal(err)
		}
	})
	if _, err := New(live).Restore(testCtx, corruptV1); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("corrupt v1 got err=%v, want ErrInvalid", err)
	}

	// Case 2: manifest claims v1 while database.sqlite is actually v2.
	v2SourceArchive, _ := exportFixture(t)
	spoofed := zipItems(t, v2SourceArchive)
	editManifest(t, spoofed, func(m *manifest) { m.SchemaVersion = 1 })
	if _, err := New(live).Restore(testCtx, writeItems(t, spoofed)); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("manifest/db mismatch got err=%v, want ErrInvalid", err)
	}

	// Case 3: SchemaVersion 0 (too old) and SchemaVersion+1 (future).
	tooOld := zipItems(t, v2SourceArchive)
	editManifest(t, tooOld, func(m *manifest) { m.SchemaVersion = 0 })
	if _, err := New(live).Restore(testCtx, writeItems(t, tooOld)); !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("schema version 0 got err=%v, want ErrInvalid", err)
	}

	// Case 4: v2 archive with invalid job_applications semantic values.
	seedV2Source := testStore(t)
	srcJobs := jobapplications.New(seedV2Source)
	srcApp, err := srcJobs.Create(testCtx, jobapplications.CreateInput{
		CompanyName:  "合法公司",
		PositionName: "工程师",
		AppliedDate:  "2026-11-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	v2Archive := filepath.Join(t.TempDir(), "base-v2.zip")
	if err := New(seedV2Source).Export(testCtx, v2Archive); err != nil {
		t.Fatal(err)
	}

	invalidMutations := []struct {
		name string
		sql  string
	}{
		{"blank company", `UPDATE job_applications SET company_name='   ' WHERE id='` + srcApp.ID + `'`},
		{"bad applied_date", `UPDATE job_applications SET applied_date='2026-02-30' WHERE id='` + srcApp.ID + `'`},
		{"bad application_url", `UPDATE job_applications SET application_url='javascript:alert(1)' WHERE id='` + srcApp.ID + `'`},
		{"bad follow-up date", `UPDATE job_applications SET next_follow_up_date='2026-02-30' WHERE id='` + srcApp.ID + `'`},
		{"bad event timestamp", `UPDATE job_application_events SET occurred_at='bad-ts' WHERE application_id='` + srcApp.ID + `'`},
	}
	for _, tc := range invalidMutations {
		items := zipItems(t, v2Archive)
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
