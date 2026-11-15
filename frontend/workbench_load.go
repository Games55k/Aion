package frontend

// loadUI reads into worker-local values and applies them on the UI thread only
// if the page, draft, restore epoch, and request are still current. Synchronous
// reloads also invalidate earlier requests. work must never read mutable UI state.
func (w *Workbench) loadUI(key string, async bool, work func() func()) {
	if w.loadVersions == nil {
		w.loadVersions = make(map[string]uint64)
	}
	w.loadVersions[key]++
	version, navigation, revision, epoch := w.loadVersions[key], w.navigationVersion, w.saveRevision, w.currentDataEpoch()
	if !async || w.host.Dispatch == nil {
		work()()
		return
	}
	dispatch := w.host.Dispatch
	w.taskWG.Add(1)
	go func() {
		defer w.taskWG.Done()
		apply := work()
		dispatch(func() {
			if w.IsRestoring() || epoch != w.currentDataEpoch() || navigation != w.navigationVersion || revision != w.saveRevision || version != w.loadVersions[key] {
				return
			}
			apply()
		})
	}()
}
