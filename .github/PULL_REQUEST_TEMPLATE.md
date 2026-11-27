## Summary

<!-- Briefly describe what changed and why. -->

## Risk

- [ ] **Low** — UI styling/layout, icons, `cmd/promo/`, docs, `README.md`, `CHANGELOG.md`
- [ ] **Medium** — `backend/projects/`, `backend/jobapplications/`, `frontend/*_save.go`, `frontend/record_write.go`, `frontend/workbench_load.go`, `frontend/external_sync.go`, `frontend/job_favicon.go`, `cli/`, `main.go`
- [ ] **High** — `backend/store/`, `backend/backup/`, `backend/attachments/`, `updater_controller.go`, `mygo.json`, `ipc/`, `mcp/`, `.github/workflows/`

## Testing

- [ ] `go vet ./...` and `go tool mygo vet .`
- [ ] `go test ./...` and `go test -race . ./backend/... ./cli/... ./ipc/... ./mcp/...`
- [ ] New/modified UI tests run with `-count=5`
- [ ] Visual snapshots verified with `TestVisualSnapshotRender` (if UI changed)
- [ ] Reviewed diff against `REVIEW.md` (and security pass if High risk)

Additional verification notes:

## How to revert

<!-- Is a plain `git revert` safe? Note any SQLite schema changes, local data layout changes, or updater/release metadata impacts. -->
