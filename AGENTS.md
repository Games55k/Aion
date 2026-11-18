# AGENTS.md

## UI
Read https://mygo.run/zh/docs/ui and follow it to build native UI with MyGo (`github.com/egoist/mygo/ui`).

## Tests

Test behaviour that can regress. Cover input effects, backend invariants, rendering under long-held `Store` locks, image cache bounds, and bugs you fix. Skip assertions on labels, form defaults or row order, and framework behaviour.

- Use `openTestWorkbench` in `frontend/workbench_test.go` for UI tests. Locate controls by accessibility label, simulate input with `ui.Tester`, and assert on workbench, database, or filesystem state.
- Stub system capabilities through `frontend.Host`. Keep tests isolated from native dialogs, external apps, and the real Aion data directory.
- Use channels, a custom `Host.Dispatch` queue, and deterministic clocks (`Now` in `backup.Options`, `backend.Options`, and `frontend.Host`) as needed. Avoid wall-clock assertions and fixed sleeps. `WaitForTasks` waits only for background workers. Drain queued UI callbacks separately.
- For asynchronous saves and navigation, hold `Store` with channels, exercise input before releasing it, then drain workers and UI callbacks until idle. Cover latest-draft ordering, stale loads, failures, discard, close/restore guards, and attachment GC during unlocked backup packing. Backup operations acquire `Store.BackupOperation` before `Store.Exclusive`; never reverse that order.
- Inspect visual changes with `TestVisualSnapshotRender`: `AION_SAVE_SNAPSHOT` renders the board, `AION_SAVE_SETTINGS_SNAPSHOT` renders Settings, and `AION_SAVE_JOBS_SNAPSHOT` / `AION_SAVE_JOBS_NARROW_SNAPSHOT` / `AION_SAVE_JOB_DETAIL_SNAPSHOT` render the Job Applications table, narrow/collapsed responsive view, and detail modal.
- Use `automation_e2e.go` (`go build -tags e2e` with `MYGO_AUTOMATION` and `-data-dir` in a temporary directory) and `mygo-automation` for real-window smoke checks, layout inspection, and screenshots. Do not replace deterministic `ui.Tester` tests with Automation, do not bypass `Host` stubs for dialogs/external apps, and do not add GUI-dependent Automation checks as unconditional CI gates. Prefer stable `Key`s and accessibility labels over transient `@eN` refs, and check explicit state with bounded deadlines instead of fixed sleeps.
- Open each test with a one-sentence comment describing the behaviour. For regressions, explain what broke.
- Run `go test ./...` before committing and run new UI tests with `-count=5`. Fix or remove flaky tests.
- Round-trip a closed Offer application after a stage correction. When restoring from an old safety archive repeatedly, verify rotation retains the source and the latest five safety archives.
- Cover cross-module rollback when clearing trash fails, partial-success deletion (metadata committed but attachment GC failed), project visibility in counts, and default backup-directory validation. Snapshot CLI tests must use temporary data and verify that output cannot overwrite managed data or truncate hard-linked files.

## Documentation

Implementation plans live in `docs/plans/`. Keep each plan's status explicit; planned commands and capabilities are not current workflows until implemented. The Automation and UI simplification plan is in `docs/plans/automation-and-ui-simplification.md`.

Keep documentation in sync with code changes. Whenever you add, modify, or remove user-facing behaviour, shortcuts, CLI flags, data or backup layouts, build/test workflows, or repository structure, update the affected sections of `README.md`, `AGENTS.md`, and relevant comments in the same change. Update `CHANGELOG.md` when preparing a release, and refresh the headless screenshot in `README.md` (`assets/screenshot.png`) before each release via `AION_SAVE_SNAPSHOT=assets/screenshot.png go test ./frontend -run TestVisualSnapshotRender`.

## Commits

Automatically commit your changes after completing each modification once documentation is synced and required tests and checks pass.

Follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). Write the subject in English using `<type>(<scope>): <description>`. The scope is optional.

Use `feat` for new features, `fix` for bug fixes, and a suitable type such as `docs`, `refactor`, `test`, or `chore` for other changes. Mark breaking changes with `!` before the colon or a `BREAKING CHANGE:` footer.
