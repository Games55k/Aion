package frontend

const (
	themePreference  = "theme"
	backupPreference = "backup"
)

func (w *Workbench) settingWritePending(key string) bool {
	_, pending := w.settingWrites[key]
	return pending
}

// Settings controls can change while a write waits for Store. Keep only the
// latest unsaved value per preference behind the active write.
func (w *Workbench) saveSetting(key string, work func() func()) {
	w.loadVersions["settings"]++
	if w.settingWritePending(key) {
		w.settingWrites[key] = work
		return
	}
	if w.settingWrites == nil {
		w.settingWrites = make(map[string]func() func())
	}
	w.startSettingWrite(key, work)
}

func (w *Workbench) startSettingWrite(key string, work func() func()) {
	w.settingWrites[key] = nil
	finish := func(apply func()) {
		next := w.settingWrites[key]
		if next == nil {
			delete(w.settingWrites, key)
		}
		w.loadVersions["settings"]++
		apply()
		if next != nil {
			w.startSettingWrite(key, next)
		}
	}
	if dispatch := w.host.Dispatch; dispatch != nil {
		w.taskWG.Go(func() {
			apply := work()
			dispatch(func() { finish(apply) })
		})
		return
	}
	finish(work())
}
