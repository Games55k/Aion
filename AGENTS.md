# AGENTS.md

## UI
Read https://mygo.lobehub.com/skills.md and follow it to build native UI with Lobe MyGo.

## Tests

Test behaviour that can regress. Cover input effects, backend invariants, rendering under long-held `Store` locks, image cache bounds, and bugs you fix. Skip assertions on labels, form defaults or row order, and framework behaviour.

- Use `openTestWorkbench` in `frontend/workbench_test.go` for UI tests. Locate controls by accessibility label, simulate input with `ui.Tester`, and assert on workbench, database, or filesystem state.
- Stub system capabilities through `frontend.Host`. Keep tests isolated from native dialogs, external apps, and the real Aion data directory.
- Use channels, a custom `Host.Dispatch` queue, and deterministic clocks (`Now` in `backup.Options`, `backend.Options`, and `frontend.Host`) as needed. Avoid wall-clock assertions and fixed sleeps. `WaitForTasks` waits only for background workers. Drain queued UI callbacks separately.
- Inspect visual changes with `TestVisualSnapshotRender`: `AION_SAVE_SNAPSHOT` renders the board, `AION_SAVE_SETTINGS_SNAPSHOT` renders Settings.
- Open each test with a one-sentence comment describing the behaviour. For regressions, explain what broke.
- Run `go test ./...` before committing and run new UI tests with `-count=5`. Fix or remove flaky tests.

## Documentation

Keep documentation in sync with code changes. Whenever you add, modify, or remove user-facing behaviour, shortcuts, CLI flags, data or backup layouts, build/test workflows, or repository structure, update the affected sections of `README.md`, `AGENTS.md`, and relevant comments in the same change. Update `CHANGELOG.md` when preparing a release.

## Commit Messages

Follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). Write the subject in English using `<type>(<scope>): <description>`. The scope is optional.

Use `feat` for new features, `fix` for bug fixes, and a suitable type such as `docs`, `refactor`, `test`, or `chore` for other changes. Mark breaking changes with `!` before the colon or a `BREAKING CHANGE:` footer.
