# AGENTS.md

## UI
Read https://mygo.lobehub.com/skills.md and follow it to build native UI with Lobe MyGo.

## Tests

Only write a test for behaviour that can actually break. If the code under test could change without the test noticing, or the test would fail on a harmless rewording, don't write it.

**Write tests for:**

- What input does: clicks, keys, typing, drags and scrolling, and the state or effect they lead to (creating or editing projects and tasks, dragging cards across or within board columns, filtering and sorting, toggling subtasks, importing or exporting attachments, restoring or purging from Trash, and running backup or restore).
- Logic that doesn't need a view: SQLite transactions and lock ordering (`store`), project/task positioning and Trash cascades (`projects`), managed attachment paths, copy/export safety and unreferenced file GC (`attachments`), backup archive verification, atomic database replacement and rotation (`backup`), and bounded image cache eviction.
- A regression you just fixed. Say in the test's comment what went wrong.
- Non-blocking rendering and cache bounds, so views don't block on long-held `Store` locks or retain deleted full-size bitmaps.

**Don't write tests that:**

- Check that a list of labels shows on screen. That repeats the view code and breaks whenever the wording changes.
- Check default settings or the order of rows in a form.
- Test MyGo or Lobe MyGo itself, for example that `AnimateTo`, a spring, or a built-in component renders its own props.
- Depend on the wall clock or fixed `time.Sleep`. `ui.NewTester` starts its clock at the current time, and async coordination should use channels or deterministic clocks (`Now` options) rather than real-time sleeps.

**How:**

- Build views with `openTestWorkbench` (in `frontend/workbench_test.go`), which opens an isolated `backend.App` in `t.TempDir()` with `DisableScheduler: true` and sets up `ui.Tester`. Find elements by their accessibility label, then `Click`, `Type`, `Press`/`Move`/`Release` or `Key`, and check the workbench, database, or filesystem state afterwards.
- Swap out system capabilities through `frontend.Host` (`PickOpenFile`, `PickSaveFile`, `PickDirectory`, `OpenPath`, `CopyText`, `ApplyTheme`, `Dispatch`). Never open native dialogs, launch external apps, or touch `~/Library/Application Support/Aion` in tests.
- Control background goroutines and UI callbacks with a custom `Host.Dispatch` queue and channels when testing async ordering (`WaitForTasks` waits for background workers, not queued UI callbacks).
- Look at visuals instead of asserting on them. `TestVisualSnapshotRender` saves a PNG when `AION_SAVE_SNAPSHOT` is set and skips when it isn't.
- Start each test with a one-sentence comment saying the behaviour it checks, in the style of the code.
- Run `go test ./...` before committing, and run a new UI test with `-count=5`. Fix or delete a flaky test; don't retry it.
