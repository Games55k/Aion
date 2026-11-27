# AGENTS.md

## UI
Read https://mygo.run/zh/docs/ui and follow it to build native UI with MyGo (`github.com/egoist/mygo/ui`).

## Tests

Test behaviour that can regress. Cover input effects, backend invariants, rendering under long-held `Store` locks, image cache bounds, and bugs you fix. Skip assertions on labels, form defaults or row order, and framework behaviour. Open each test with a one-sentence comment describing the behaviour (for regressions, explain what broke). Run `go test ./...` before committing and run new UI tests with `-count=5`; fix or remove flaky tests.

### UI Harness, Snapshots & E2E
- Use `openTestWorkbench` in `frontend/workbench_test.go` for UI tests. Locate controls by accessibility label, simulate input with `ui.Tester`, and assert on workbench, database, or filesystem state.
- Stub system capabilities through `frontend.Host`. Keep tests isolated from native dialogs, external apps, and the real Aion data directory. Drive attachment and manual backup export UI tests through the asynchronous picker with `Host.PickSaveFile`, and check file validation and integrity through backend export methods.
- Inspect visual changes with `TestVisualSnapshotRender` using the `AION_SAVE_*_SNAPSHOT` variables documented in `docs/development.md` (board, settings, focus overview, command palette, update modal, jobs table, narrow jobs table, and job detail). Snapshot CLI tests must use temporary data and verify that output cannot overwrite managed data or truncate hard-linked files.
- Use `automation_e2e.go` (`go build -tags e2e` with `MYGO_AUTOMATION` and `-data-dir` in a temporary directory) and `mygo-automation` for real-window smoke checks, layout inspection, and screenshots. Do not replace deterministic `ui.Tester` tests with Automation, do not bypass `Host` stubs for dialogs/external apps, and do not add GUI-dependent Automation checks as unconditional CI gates. Prefer stable `Key`s and accessibility labels over transient `@eN` refs, and check explicit state with bounded deadlines instead of fixed sleeps.

### Concurrency, Async Writes & Locks
- Use channels, a custom `Host.Dispatch` queue, and deterministic clocks (`Now` in `backup.Options`, `backend.Options`, and `frontend.Host`) as needed. Avoid wall-clock assertions and fixed sleeps. `WaitForTasks` waits only for background workers; drain queued UI callbacks separately.
- Backup operations acquire `Store.BackupOperation` before `Store.Exclusive`; never reverse that order.
- Pause attachment imports during the staged copy and exercise concurrent Store writes, GC, and backup restore. Verify target visibility again at publication and check staging, managed files, and metadata after cancellation or transaction failure.
- For asynchronous saves, navigation, board operations, creates, trash restores, and settings writes, hold `Store` with channels while driving the actual UI controls, then drain workers and UI callbacks until idle. Cover duplicate submissions, latest-draft ordering, newer input/navigation, stale loads, failures, discard, close/restore guards, and attachment GC during unlocked backup packing and individual attachment exports.
- Exercise timeline submissions while a previous write is pending (including duplicate clicks and subsequent typing). When testing discard after a rejected save, verify that the next save preserves later subtask moves (even when conflicting with committed or restored positions), edits to a newly added subtask, and raw multiline input.

### Domain, Backup, Trash & IPC/MCP Invariants
- Use `backend.Options.Now` for project and task timestamp tests. Verify UTC storage and a shared timestamp for all task rows affected by a board move or reorder.
- Verify that backfilled job stages keep their historical occurrence date and refresh the application's recently updated sort order once, covering duplicate submissions, newer edits, and current workflow/closure fields. Exercise job empty states (no applications, empty category alongside existing records, active filters) and verify first-use creation and clearing filters via UI controls.
- Restore accepts only the current database schema and rejects older archives before creating a safety backup; keep local database migrations independent of restore (verify upgrading a v2 local database removes the follow-up column while preserving job records and timelines). After a failed restore, verify reopening cannot create a missing database and Settings reports any completed safety archive. Round-trip a closed Offer application after a stage correction, and verify repeated restores from an old safety archive retain the source and latest five safety archives even after a backwards clock adjustment.
- Keep the global trash list in a single read snapshot and sort parsed deletion timestamps. Cover cross-module rollback when clearing trash fails, partial-success deletion (metadata committed but attachment GC failed), project visibility in counts, default backup-directory validation, missing IDs, and invalid deletion states across projects, tasks, and job applications (including a recycled parent blocking task restore while an independently recycled task can still be purged).
- For local IPC and MCP (`ipc/`, `mcp/`), verify both live IPC-socket routing and offline per-call fallback release `.aion.lock` cleanly, verify partial update tools preserve omitted fields, verify destructive operations (permanent delete, clear trash, backup restore, settings writes) are never exposed, and verify GUI external-change refreshes never overwrite active task or job drafts.

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
