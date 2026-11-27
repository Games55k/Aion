# REVIEW.md

Code review contract for Aion (used for fresh-context pre-commit diff reviews before pushing directly to `main`, and for pull requests).

## Scope & Ignored Paths

Review only the changed code and its direct behavioral impact.

Ignore unless explicitly modified as functional logic:
- `assets/`
- `build/`
- `cmd/promo/`
- `go.sum`

## Risk Tiers (Set by Change, Not Author)

- **High (Deep review + security pass + human check)**
  - `backend/store/`
  - `backend/backup/`
  - `backend/attachments/`
  - `updater_controller.go`, `mygo.json`
  - `ipc/`, `mcp/`
  - `.github/workflows/`
- **Medium (Standard review)**
  - `backend/projects/`, `backend/jobapplications/`
  - `frontend/*_save.go`, `frontend/record_write.go`, `frontend/workbench_load.go`, `frontend/external_sync.go`, `frontend/job_favicon.go`
  - `cli/`, `main.go`
- **Low (Quick review)**
  - `frontend/ui_*.go`, `frontend/icons.go`, `docs/`, `README.md`, `CHANGELOG.md`

## What Counts as a Real Failure in Aion

Flag issues in these categories as **Important**:

1. **Data Loss, Corruption, or Migration Breakage**
   - Lock ordering violations: backup operations MUST acquire `Store.BackupOperation` before `Store.Exclusive` (never reversed).
   - SQLite transactions missing rollback on error, partial cross-module trash clears, or restore accepting an older schema instead of rejecting before creating a safety archive.
   - Reopening after a failed restore creating an empty database or losing visibility of the `safety-*.zip` archive.
   - Restoring/importing/exporting files without verifying target visibility at commit or without atomic replacement / integrity checks.

2. **Concurrency, Draft Safety, & UI Responsiveness**
   - Holding `Store` locks across UI rendering, compression, network I/O, or attachment copy/export.
   - Stale background loads or external IPC/MCP sync refreshes overwriting active unsaved task or job drafts.
   - Async save / submit races (duplicate clicks, out-of-order responses, missing close/restore guards, lost edits after a rejected save).
   - Unbounded in-memory or disk caches (thumbnails, favicons) or leaked goroutines / locks (`.aion.lock`).

3. **Local Security & Non-Destructive Boundaries**
   - Path traversal, symlink escape, hard-link truncation, or FIFO/non-regular-file hazards in attachments, backups, snapshot output, or IPC socket paths.
   - Exposing destructive operations (permanent delete, clear trash, backup restore, settings writes) via `mcp/` or unsafe IPC endpoints.
   - Unsafe external fetches/decoders in `frontend/job_favicon.go` (missing size limits, timeouts, or malformed image handling).
   - Release workflow leaking secrets/keychains or uploading without confirming an absent tag or existing draft release.

4. **Cross-Platform & Build Regressions**
   - Breaking macOS or Windows compilation (`*_unix.go` / `*_windows.go` parity, path handling, socket/lock behavior).
   - Introducing wall-clock flakiness or `time.Sleep` in tests instead of deterministic clocks (`Now`) and channels.

## Anti-Noise & Convergence Rules

- **Do not nitpick** formatting, naming taste, comment wording, or micro-refactors (`gofmt`, `go vet`, and `mygo vet` handle mechanical checks).
- **Do not report pre-existing issues** outside the current diff unless the change directly triggers or worsens them.
- **Name the bug class**, not only one instance, when the same flaw appears in multiple places in the diff.
- **Convergence cap**: Limit automated review loops to **3 rounds**. After round 1, review only the **new incremental diff** and do not re-flag resolved or acknowledged items. If blocking issues persist after 3 rounds, stop and ask the author to re-evaluate the design.

## Required Output Format

1. **Summary line**: State the inferred risk tier (`Low` / `Medium` / `High`) and either the exact count of `Important` issues or an explicit `"No blocking issues found."`
2. **Findings**: Group by severity (`Important`, `Nit`, `Pre-existing`). Every finding must cite an exact, verified `path:line` from the repo (never guess file names or line numbers), describe the concrete failure mode, and suggest the minimal fix.
