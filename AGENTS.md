# AGENTS.md

## UI
Read https://mygo.run/zh/docs/ui and follow it to build native UI with MyGo (`github.com/egoist/mygo/ui`).

## Tests

Test behaviour that can regress. Cover input effects, backend invariants, rendering under long-held `Store` locks, image cache bounds, and bugs you fix. Skip assertions on labels, form defaults or row order, and framework behaviour.

- Use `openTestWorkbench` in `frontend/workbench_test.go` for UI tests. Locate controls by accessibility label, simulate input with `ui.Tester`, and assert on workbench, database, or filesystem state.
- Stub system capabilities through `frontend.Host`. Keep tests isolated from native dialogs, external apps, and the real Aion data directory.
- Drive attachment and manual backup export UI tests through the asynchronous picker with `Host.PickSaveFile`. Check file validation and integrity through the backend export methods.
- Pause attachment imports during the staged copy and exercise concurrent Store writes, GC, and backup restore. Verify target visibility again at publication and check staging, managed files, and metadata after cancellation or transaction failure.
- Use channels, a custom `Host.Dispatch` queue, and deterministic clocks (`Now` in `backup.Options`, `backend.Options`, and `frontend.Host`) as needed. Avoid wall-clock assertions and fixed sleeps. `WaitForTasks` waits only for background workers. Drain queued UI callbacks separately.
- Use `backend.Options.Now` for project and task timestamp tests. Verify UTC storage and a shared timestamp for all task rows affected by a board move or reorder.
- For asynchronous saves and navigation, hold `Store` with channels, exercise input before releasing it, then drain workers and UI callbacks until idle. Exercise timeline submissions while a previous write is pending, including duplicate clicks and subsequent typing. Cover latest-draft ordering, stale loads, failures, discard, close/restore guards, and attachment GC during unlocked backup packing and individual attachment exports. Backup operations acquire `Store.BackupOperation` before `Store.Exclusive`; never reverse that order.
- Verify that backfilled stages keep their historical occurrence date and refresh the application's recently updated sort order once. Cover duplicate submissions, newer edits, and the current workflow and closure fields.
- Exercise job empty states with no applications, an empty category alongside existing records, and active filters. Verify first-use creation and clearing filters through the UI controls.
- When testing subtask order after a discard, reject the first save and verify that the next save preserves the user's later moves even when they conflict with committed positions or the position of a restored subtask.
- Cover edits to a newly added subtask and raw multiline input submitted after discard when the preceding save fails.
- For board, create, trash restore, and settings writes, hold Store while driving the actual UI controls. Verify duplicate submission, newer input and navigation, failure, and close and backup restore guards before applying worker results.
- Inspect visual changes with `TestVisualSnapshotRender`: `AION_SAVE_SNAPSHOT` renders the board, `AION_SAVE_SETTINGS_SNAPSHOT` renders Settings, `AION_SAVE_UPDATE_MODAL_SNAPSHOT` renders the in-app Update modal, and `AION_SAVE_JOBS_SNAPSHOT` / `AION_SAVE_JOBS_NARROW_SNAPSHOT` / `AION_SAVE_JOB_DETAIL_SNAPSHOT` render the Job Applications table, narrow/collapsed responsive view, and detail modal.
- Use `automation_e2e.go` (`go build -tags e2e` with `MYGO_AUTOMATION` and `-data-dir` in a temporary directory) and `mygo-automation` for real-window smoke checks, layout inspection, and screenshots. Do not replace deterministic `ui.Tester` tests with Automation, do not bypass `Host` stubs for dialogs/external apps, and do not add GUI-dependent Automation checks as unconditional CI gates. Prefer stable `Key`s and accessibility labels over transient `@eN` refs, and check explicit state with bounded deadlines instead of fixed sleeps.
- Open each test with a one-sentence comment describing the behaviour. For regressions, explain what broke.
- Run `go test ./...` before committing and run new UI tests with `-count=5`. Fix or remove flaky tests.
- After a failed restore, verify that reopening cannot create a missing database and that Settings reports any completed safety archive. Round-trip a closed Offer application after a stage correction. When restoring from an old safety archive repeatedly, verify rotation retains the source and the latest five safety archives, including after a backwards clock adjustment.
- Restore accepts only the current database schema and rejects older archives before creating a safety backup. Keep local database migrations independent of restore. Verify that upgrading a v2 local database removes the follow-up column and preserves job records and their timeline.
- Keep the global trash list in a single read snapshot and sort parsed deletion timestamps. Cover cross-module rollback when clearing trash fails, partial-success deletion (metadata committed but attachment GC failed), project visibility in counts, and default backup-directory validation. Snapshot CLI tests must use temporary data and verify that output cannot overwrite managed data or truncate hard-linked files.
- Cover missing IDs and invalid deletion states across projects, tasks, and job applications for soft deletion, restore, and permanent deletion. Verify that a recycled parent blocks task restore and that an independently recycled task can still be purged.
- For local IPC and MCP (`ipc/`, `mcp/`), verify both live Unix-socket routing and offline per-call fallback release `.aion.lock` cleanly, verify partial update tools preserve omitted fields, verify destructive operations (permanent delete, clear trash, backup restore, settings writes) are never exposed, and verify GUI external-change refreshes never overwrite active task or job drafts.

## Documentation

Write implementation plans in `docs/plans/` and state each plan's status. Document commands and capabilities as current workflows once implemented.

Keep documentation in sync with code changes. Keep `README.md` concise (overview, core features, quick start, and links to `docs/`) and place detailed specifications and workflows in `docs/` (`docs/behavior.md`, `docs/data-and-backup.md`, `docs/development.md`). Whenever you add, modify, or remove user-facing behaviour, shortcuts, CLI flags, data or backup layouts, build/test workflows, or repository structure, update the affected files among `README.md`, `docs/`, `AGENTS.md`, and relevant comments in the same change. Update `CHANGELOG.md` when preparing a release, and refresh the headless screenshot in `README.md` (`assets/screenshot.png`) before each release via `AION_SAVE_SNAPSHOT="$PWD/assets/screenshot.png" go test -count=1 ./frontend -run TestVisualSnapshotRender`.

Release uploads require a successful lookup confirming an absent tag release or an existing draft. A failed lookup must stop the workflow.

## Review

Follow `REVIEW.md` as the review contract (risk tiers, Aion failure modes, anti-noise rules, and output format).

- For Medium- and High-risk changes committed directly to `main` (or opened as a PR), run a fresh-context diff review against `REVIEW.md` before committing/pushing, plus a focused security pass for High-risk paths (`backend/store/`, `backend/backup/`, `backend/attachments/`, `updater_controller.go`, `mygo.json`, `ipc/`, `mcp/`, `.github/workflows/`).
- Limit automated review loops to 3 rounds; after round 1, review only the new incremental diff and stop to re-evaluate the approach if blocking issues still do not converge.
- When a real bug escapes review, add a deterministic regression test and, if it represents a recurring failure mode, add a concise rule to `REVIEW.md`.

## Commits

Automatically commit your changes after completing each modification once documentation is synced and required tests and checks pass.

Follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). Write the subject in English using `<type>(<scope>): <description>`. The scope is optional.

Use `feat` for new features, `fix` for bug fixes, and a suitable type such as `docs`, `refactor`, `test`, or `chore` for other changes. Mark breaking changes with `!` before the colon or a `BREAKING CHANGE:` footer.
