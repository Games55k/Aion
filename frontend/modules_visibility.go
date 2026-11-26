package frontend

import (
	"context"
	"slices"

	"aion/backend"
)

// VisibleModules returns built-in modules not currently hidden by user preferences.
// If every module were somehow marked hidden, the first registered module stays visible.
func (w *Workbench) VisibleModules() []backend.ModuleDescriptor {
	if len(w.modules) == 0 {
		return nil
	}
	out := make([]backend.ModuleDescriptor, 0, len(w.modules))
	for _, m := range w.modules {
		if !w.hiddenModules[m.ID] {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return []backend.ModuleDescriptor{w.modules[0]}
	}
	return out
}

// IsModuleVisible reports whether modID is currently shown in the workbench navigation.
func (w *Workbench) IsModuleVisible(modID string) bool {
	for _, m := range w.VisibleModules() {
		if m.ID == modID {
			return true
		}
	}
	return false
}

func (w *Workbench) hiddenModulesList() []string {
	out := make([]string, 0, len(w.hiddenModules))
	for _, m := range w.modules {
		if w.hiddenModules[m.ID] {
			out = append(out, m.ID)
		}
	}
	slices.Sort(out)
	return out
}

func (w *Workbench) applyHiddenModules(ids []string) {
	next := make(map[string]bool, len(ids))
	for _, id := range ids {
		next[id] = true
	}
	// Never allow every registered module to disappear.
	if len(w.modules) > 0 {
		visibleCount := 0
		for _, m := range w.modules {
			if !next[m.ID] {
				visibleCount++
			}
		}
		if visibleCount == 0 {
			delete(next, w.modules[0].ID)
		}
	}
	w.hiddenModules = next
	w.ensureVisibleModuleSelection()
}

func (w *Workbench) ensureVisibleModuleSelection() {
	vis := w.VisibleModules()
	if len(vis) == 0 {
		return
	}
	if !w.IsModuleVisible(w.activeModule) {
		w.activeModule = vis[0].ID
	}
	if isProjectNavPage(w.navPage) && !w.IsModuleVisible("projects") {
		w.navPage = w.defaultPageForModule(vis[0].ID)
	} else if isJobNavPage(w.navPage) && !w.IsModuleVisible("job_applications") {
		w.navPage = w.defaultPageForModule(vis[0].ID)
	}
}

func (w *Workbench) defaultPageForModule(modID string) NavPage {
	switch modID {
	case "job_applications":
		if isJobNavPage(w.lastJobPage) {
			return w.lastJobPage
		}
		return NavJobAll
	default:
		if isProjectNavPage(w.lastProjectPage) && (w.lastProjectPage != NavProjectBoard || w.activeProjectID != "") {
			return w.lastProjectPage
		}
		return NavAllProjects
	}
}

// SetModuleVisible toggles visibility of a built-in module and persists the preference asynchronously.
// Data in hidden modules stays intact in SQLite and backups. At least one module always remains visible.
func (w *Workbench) SetModuleVisible(modID string, visible bool) {
	if w.IsRestoring() || modID == "" {
		return
	}
	registered := false
	for _, m := range w.modules {
		if m.ID == modID {
			registered = true
			break
		}
	}
	if !registered {
		return
	}
	if w.hiddenModules == nil {
		w.hiddenModules = make(map[string]bool)
	}
	currentlyVisible := !w.hiddenModules[modID]
	if currentlyVisible == visible {
		return
	}
	if !visible && len(w.VisibleModules()) <= 1 {
		return
	}
	if visible {
		delete(w.hiddenModules, modID)
	} else {
		w.hiddenModules[modID] = true
	}
	w.ensureVisibleModuleSelection()

	desired := w.hiddenModulesList()
	settings := w.app.Settings
	w.saveSetting(modulesPreference, func() func() {
		ctx := context.Background()
		if err := settings.SetHiddenModules(ctx, desired); err != nil {
			saved, loadErr := settings.HiddenModules(ctx)
			return func() {
				if loadErr == nil && !w.settingWritePending(modulesPreference) {
					w.applyHiddenModules(saved)
				}
			}
		}
		return nil
	})
}

func moduleDescription(modID string) string {
	switch modID {
	case "projects":
		return "多项目三列看板、Markdown 任务详情、子任务与本地受管附件"
	case "job_applications":
		return "求职投递跟踪、阶段流转时间线、历史补录与结果统计"
	default:
		return "内置工作区模块"
	}
}
