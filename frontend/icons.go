package frontend

import (
	"strings"
	"sync"

	brand "aion/assets/icon"
	"aion/backend"

	"github.com/egoist/mygo/ui"
)

const (
	svgOpen  = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">`
	svgClose = `</svg>`
)

// uiIconBox renders an 18px icon glyph (1.5px stroke) centered inside a 24x24 transparent box.
func uiIconBox(c *ui.Context, svg *ui.SVG, color ui.Color) ui.Element {
	return ui.Box(c).Size(24, 24).Center().Shrink(0).Children(func() {
		icon := ui.Icon(c, svg).FontSize(18)
		if color.A > 0 {
			icon.TextColor(color)
		}
	})
}

func iconMessageSquare() *ui.SVG {
	return loadIcon("message-square", `<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>`)
}

var (
	aionSymbolOnce   sync.Once
	aionSymbolParsed *ui.SVG

	iconMu    sync.Mutex
	iconCache = map[string]*ui.SVG{}

	projectIconsOnce  sync.Once
	projectIconLookup map[string]*ui.SVG
	defaultIconSVG    *ui.SVG
	projectsModSVG    *ui.SVG
	jobsModSVG        *ui.SVG
)

func loadIcon(name, shapes string) *ui.SVG {
	iconMu.Lock()
	defer iconMu.Unlock()
	if svg, ok := iconCache[name]; ok {
		return svg
	}
	svg := ui.MustParseSVG([]byte(svgOpen + shapes + svgClose))
	iconCache[name] = svg
	return svg
}

// AionSymbolSVG returns the Aion symbol SVG.
func AionSymbolSVG() *ui.SVG {
	aionSymbolOnce.Do(func() {
		aionSymbolParsed = ui.MustParseSVG([]byte(brand.SVG))
	})
	return aionSymbolParsed
}

func ensureIconCache() {
	projectIconsOnce.Do(func() {
		projectIconLookup = make(map[string]*ui.SVG, len(ProjectIconOptions))
		for _, opt := range ProjectIconOptions {
			projectIconLookup[opt.Key] = opt.SVG()
		}
		defaultIconSVG = iconFolder()
		projectsModSVG = iconFolderKanban()
		jobsModSVG = iconBriefcase()
	})
}

// ProjectIconOption defines a selectable project icon preset.
type ProjectIconOption struct {
	Key   string
	Label string
	SVG   func() *ui.SVG
}

// ProjectIconOptions lists the built-in project icon presets.
var ProjectIconOptions = []ProjectIconOption{
	{Key: "folder", Label: "文件夹", SVG: iconFolder},
	{Key: "code", Label: "代码", SVG: iconCode},
	{Key: "rocket", Label: "发布", SVG: iconRocket},
	{Key: "book", Label: "文档", SVG: iconBookOpen},
	{Key: "target", Label: "目标", SVG: iconTarget},
	{Key: "sparkles", Label: "灵感", SVG: iconSparkles},
	{Key: "layers", Label: "架构", SVG: iconLayers},
	{Key: "terminal", Label: "终端", SVG: iconTerminal},
	{Key: "briefcase", Label: "工作", SVG: iconBriefcase},
	{Key: "heart", Label: "生活", SVG: iconHeart},
}

// ProjectIconSVG resolves a stored project icon key to an SVG icon.
func ProjectIconSVG(icon string) *ui.SVG {
	ensureIconCache()
	key := strings.TrimSpace(icon)
	if svg, ok := projectIconLookup[key]; ok {
		return svg
	}
	if svg, ok := projectIconLookup[strings.ToLower(key)]; ok {
		return svg
	}
	return defaultIconSVG
}

// ModuleIconSVG resolves a registered module descriptor to an SVG icon.
func ModuleIconSVG(mod backend.ModuleDescriptor) *ui.SVG {
	ensureIconCache()
	switch mod.ID {
	case "projects":
		return projectsModSVG
	case "job_applications":
		return jobsModSVG
	default:
		return defaultIconSVG
	}
}

func iconArchive() *ui.SVG {
	return loadIcon("archive", `<rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/>`)
}

func iconArchiveRestore() *ui.SVG {
	return loadIcon("archive-restore", `<rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h2"/><path d="M20 8v11a2 2 0 0 1-2 2h-2"/><path d="m9 15 3-3 3 3"/><path d="M12 12v9"/>`)
}

func iconArrowBigUp() *ui.SVG {
	return loadIcon("arrow-big-up", `<path d="M9 19a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-6a1 1 0 0 1 1-1h3.293a.707.707 0 0 0 .5-1.207l-7.086-7.086a1 1 0 0 0-1.414 0l-7.086 7.086a.707.707 0 0 0 .5 1.207H8a1 1 0 0 1 1 1z"/>`)
}

func iconArrowDown() *ui.SVG {
	return loadIcon("arrow-down", `<path d="M12 5v14"/><path d="m19 12-7 7-7-7"/>`)
}

func iconArrowDownWideNarrow() *ui.SVG {
	return loadIcon("arrow-down-wide-narrow", `<path d="m3 16 4 4 4-4"/><path d="M7 20V4"/><path d="M11 4h10"/><path d="M11 8h7"/><path d="M11 12h4"/>`)
}

func iconArrowLeft() *ui.SVG {
	return loadIcon("arrow-left", `<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>`)
}

func iconArrowRight() *ui.SVG {
	return loadIcon("arrow-right", `<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>`)
}

func iconArrowRightToLine() *ui.SVG {
	return loadIcon("arrow-right-to-line", `<path d="M17 12H3"/><path d="m11 18 6-6-6-6"/><path d="M21 5v14"/>`)
}

func iconArrowUp() *ui.SVG {
	return loadIcon("arrow-up", `<path d="m5 12 7-7 7 7"/><path d="M12 19V5"/>`)
}

func iconArrowUpDown() *ui.SVG {
	return loadIcon("arrow-up-down", `<path d="m21 16-4 4-4-4"/><path d="M17 20V4"/><path d="m3 8 4-4 4 4"/><path d="M7 4v16"/>`)
}

func iconArrowUpNarrowWide() *ui.SVG {
	return loadIcon("arrow-up-narrow-wide", `<path d="m3 8 4-4 4 4"/><path d="M7 4v16"/><path d="M11 12h4"/><path d="M11 16h7"/><path d="M11 20h10"/>`)
}

func iconAward() *ui.SVG {
	return loadIcon("award", `<path d="m15.477 12.89 1.515 8.526a.5.5 0 0 1-.81.47l-3.58-2.687a1 1 0 0 0-1.197 0l-3.586 2.686a.5.5 0 0 1-.81-.469l1.514-8.526"/><circle cx="12" cy="8" r="6"/>`)
}

func iconBan() *ui.SVG {
	return loadIcon("ban", `<circle cx="12" cy="12" r="10"/><path d="M4.929 4.929 19.07 19.071"/>`)
}

func iconBookOpen() *ui.SVG {
	return loadIcon("book-open", `<path d="M12 5v16"/><path d="M20.001 19A2 2 0 0022 17V5a2 2 0 00-1.999-2L16 3.002A5 5 0 0012 5a5 5 0 00-4-2H4a2 2 0 00-2 2v12a2 2 0 001.999 2H8a5 5 0 014 2 5 5 0 014-2z"/>`)
}

func iconBriefcase() *ui.SVG {
	return loadIcon("briefcase", `<path d="M16 20V4a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/><rect width="20" height="14" x="2" y="6" rx="2"/>`)
}

func iconCalendar() *ui.SVG {
	return loadIcon("calendar", `<path d="M8 2v3"/><path d="M16 2v3"/><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/>`)
}

func iconCheck() *ui.SVG {
	return loadIcon("check", `<path d="M20 6 9 17l-5-5"/>`)
}

func iconCheckCheck() *ui.SVG {
	return loadIcon("check-check", `<path d="M18 6 7 17l-5-5"/><path d="m22 10-7.5 7.5L13 16"/>`)
}

func iconChevronDown() *ui.SVG {
	return loadIcon("chevron-down", `<path d="m6 9 6 6 6-6"/>`)
}

func iconChevronLeft() *ui.SVG {
	return loadIcon("chevron-left", `<path d="m15 18-6-6 6-6"/>`)
}

func iconChevronRight() *ui.SVG {
	return loadIcon("chevron-right", `<path d="m9 18 6-6-6-6"/>`)
}

func iconChevronUp() *ui.SVG {
	return loadIcon("chevron-up", `<path d="m18 15-6-6-6 6"/>`)
}

func iconChevronsLeft() *ui.SVG {
	return loadIcon("chevrons-left", `<path d="m11 17-5-5 5-5"/><path d="m18 17-5-5 5-5"/>`)
}

func iconChevronsRight() *ui.SVG {
	return loadIcon("chevrons-right", `<path d="m6 17 5-5-5-5"/><path d="m13 17 5-5-5-5"/>`)
}

func iconCircleArrowUp() *ui.SVG {
	return loadIcon("circle-arrow-up", `<circle cx="12" cy="12" r="10"/><path d="m16 12-4-4-4 4"/><path d="M12 16V8"/>`)
}

func iconCircleCheck() *ui.SVG {
	return loadIcon("circle-check", `<circle cx="12" cy="12" r="10"/><path d="m16 9-5.5 5.5L8 12"/>`)
}

func iconCircleDashed() *ui.SVG {
	return loadIcon("circle-dashed", `<path d="M10.1 2.182a10 10 0 0 1 3.8 0"/><path d="M13.9 21.818a10 10 0 0 1-3.8 0"/><path d="M17.609 3.721a10 10 0 0 1 2.69 2.7"/><path d="M2.182 13.9a10 10 0 0 1 0-3.8"/><path d="M20.279 17.609a10 10 0 0 1-2.7 2.69"/><path d="M21.818 10.1a10 10 0 0 1 0 3.8"/><path d="M3.721 6.391a10 10 0 0 1 2.7-2.69"/><path d="M6.391 20.279a10 10 0 0 1-2.69-2.7"/>`)
}

func iconCircleX() *ui.SVG {
	return loadIcon("circle-x", `<circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/>`)
}

func iconClock() *ui.SVG {
	return loadIcon("clock", `<circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>`)
}

func iconCode() *ui.SVG {
	return loadIcon("code", `<path d="m16 18 6-6-6-6"/><path d="m8 6-6 6 6 6"/>`)
}

func iconCommand() *ui.SVG {
	return loadIcon("command", `<path d="M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3"/>`)
}

func iconCopy() *ui.SVG {
	return loadIcon("copy", `<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>`)
}

func iconCornerDownLeft() *ui.SVG {
	return loadIcon("corner-down-left", `<path d="M20 4v7a4 4 0 0 1-4 4H4"/><path d="m9 10-5 5 5 5"/>`)
}

func iconDelete() *ui.SVG {
	return loadIcon("delete", `<path d="M10 5a2 2 0 0 0-1.344.519l-6.328 5.74a1 1 0 0 0 0 1.481l6.328 5.741A2 2 0 0 0 10 19h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2z"/><path d="m12 9 6 6"/><path d="m18 9-6 6"/>`)
}

func iconExpand() *ui.SVG {
	return loadIcon("expand", `<path d="m15 15 6 6"/><path d="m15 9 6-6"/><path d="M21 16v5h-5"/><path d="M21 8V3h-5"/><path d="M3 16v5h5"/><path d="m3 21 6-6"/><path d="M3 8V3h5"/><path d="M9 9 3 3"/>`)
}

func iconExternalLink() *ui.SVG {
	return loadIcon("external-link", `<path d="M15 3h6v6"/><path d="M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>`)
}

func iconFile() *ui.SVG {
	return loadIcon("file", `<path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z"/><path d="M14 2v5a1 1 0 0 0 1 1h5"/>`)
}

func iconFileText() *ui.SVG {
	return loadIcon("file-text", `<path d="M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z"/><path d="M14 2v5a1 1 0 0 0 1 1h5"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/>`)
}

func iconFolder() *ui.SVG {
	return loadIcon("folder", `<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`)
}

func iconFolderCheck() *ui.SVG {
	return loadIcon("folder-check", `<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/><path d="m9 13 2 2 4-4"/>`)
}

func iconFolderKanban() *ui.SVG {
	return loadIcon("folder-kanban", `<path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"/><path d="M8 10v4"/><path d="M12 10v2"/><path d="M16 10v6"/>`)
}

func iconFolderPlus() *ui.SVG {
	return loadIcon("folder-plus", `<path d="M12 10v6"/><path d="M9 13h6"/><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`)
}

func iconGrid2x2() *ui.SVG {
	return loadIcon("grid-2x2", `<path d="M12 3v18"/><path d="M3 12h18"/><rect x="3" y="3" width="18" height="18" rx="2"/>`)
}

func iconHardDrive() *ui.SVG {
	return loadIcon("hard-drive", `<path d="M10 16h.01"/><path d="M2.212 11.577a2 2 0 0 0-.212.896V18a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-5.527a2 2 0 0 0-.212-.896L18.55 5.11A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/><path d="M21.946 12.013H2.054"/><path d="M6 16h.01"/>`)
}

func iconHeart() *ui.SVG {
	return loadIcon("heart", `<path d="M2 9.5a5.5 5.5 0 0 1 9.591-3.676.56.56 0 0 0 .818 0A5.49 5.49 0 0 1 22 9.5c0 2.29-1.5 4-3 5.5l-5.492 5.313a2 2 0 0 1-3 .019L5 15c-1.5-1.5-3-3.2-3-5.5"/>`)
}

func iconImage() *ui.SVG {
	return loadIcon("image", `<rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/>`)
}

func iconInfo() *ui.SVG {
	return loadIcon("info", `<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>`)
}

func iconLayers() *ui.SVG {
	return loadIcon("layers", `<path d="M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83z"/><path d="M2 12a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 12"/><path d="M2 17a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 17"/>`)
}

func iconLayoutGrid() *ui.SVG {
	return loadIcon("layout-grid", `<rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/><rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/>`)
}

func iconMonitor() *ui.SVG {
	return loadIcon("monitor", `<rect width="20" height="14" x="2" y="3" rx="2"/><line x1="8" x2="16" y1="21" y2="21"/><line x1="12" x2="12" y1="17" y2="21"/>`)
}

func iconMoon() *ui.SVG {
	return loadIcon("moon", `<path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"/>`)
}

func iconOption() *ui.SVG {
	return loadIcon("option", `<path d="M14 3h7"/><path d="M3 3h5.28a1 1 0 0 1 .948.684l5.544 16.632a1 1 0 0 0 .949.684H21"/>`)
}

func iconPalette() *ui.SVG {
	return loadIcon("palette", `<path d="M12 22a1 1 0 0 1 0-20 10 9 0 0 1 10 9 5 5 0 0 1-5 5h-2.25a1.75 1.75 0 0 0-1.4 2.8l.3.4a1.75 1.75 0 0 1-1.4 2.8z"/><circle cx="13.5" cy="6.5" r=".5" fill="currentColor"/><circle cx="17.5" cy="10.5" r=".5" fill="currentColor"/><circle cx="6.5" cy="12.5" r=".5" fill="currentColor"/><circle cx="8.5" cy="7.5" r=".5" fill="currentColor"/>`)
}

func iconPaperclip() *ui.SVG {
	return loadIcon("paperclip", `<path d="m16 6-8.414 8.586a2 2 0 0 0 2.829 2.829l8.414-8.586a4 4 0 1 0-5.657-5.657l-8.379 8.551a6 6 0 1 0 8.485 8.485l8.379-8.551"/>`)
}

func iconPencil() *ui.SVG {
	return loadIcon("pencil", `<path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/>`)
}

func iconPlus() *ui.SVG {
	return loadIcon("plus", `<path d="M5 12h14"/><path d="M12 5v14"/>`)
}

func iconRefreshCw() *ui.SVG {
	return loadIcon("refresh-cw", `<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>`)
}

func iconRocket() *ui.SVG {
	return loadIcon("rocket", `<path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"/><path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09"/><path d="M9 12a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.4 22.4 0 0 1-4 2z"/><path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 .05 5 .05"/>`)
}

func iconRotateCcw() *ui.SVG {
	return loadIcon("rotate-ccw", `<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/>`)
}

func iconRotateCw() *ui.SVG {
	return loadIcon("rotate-cw", `<path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/>`)
}

func iconSearch() *ui.SVG {
	return loadIcon("search", `<path d="m21 21-4.34-4.34"/><circle cx="11" cy="11" r="8"/>`)
}

func iconSettings() *ui.SVG {
	return loadIcon("settings", `<path d="M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915"/><circle cx="12" cy="12" r="3"/>`)
}

func iconShrink() *ui.SVG {
	return loadIcon("shrink", `<path d="m15 15 6 6m-6-6v4.8m0-4.8h4.8"/><path d="M9 19.8V15m0 0H4.2M9 15l-6 6"/><path d="M15 4.2V9m0 0h4.8M15 9l6-6"/><path d="M9 4.2V9m0 0H4.2M9 9 3 3"/>`)
}

func iconSpace() *ui.SVG {
	return loadIcon("space", `<path d="M22 17v1c0 .5-.5 1-1 1H3c-.5 0-1-.5-1-1v-1"/>`)
}

func iconSparkles() *ui.SVG {
	return loadIcon("sparkles", `<path d="M11.017 2.814a1 1 0 0 1 1.966 0l1.051 5.558a2 2 0 0 0 1.594 1.594l5.558 1.051a1 1 0 0 1 0 1.966l-5.558 1.051a2 2 0 0 0-1.594 1.594l-1.051 5.558a1 1 0 0 1-1.966 0l-1.051-5.558a2 2 0 0 0-1.594-1.594l-5.558-1.051a1 1 0 0 1 0-1.966l5.558-1.051a2 2 0 0 0 1.594-1.594z"/><path d="M20 2v4"/><path d="M22 4h-4"/><circle cx="4" cy="20" r="2"/>`)
}

func iconSquareCheckBig() *ui.SVG {
	return loadIcon("square-check-big", `<path d="M21 10.656V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h12.344"/><path d="m9 11 3 3L22 4"/>`)
}

func iconSquarePen() *ui.SVG {
	return loadIcon("square-pen", `<path d="M12 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.375 2.625a1 1 0 0 1 3 3l-9.013 9.014a2 2 0 0 1-.853.505l-2.873.84a.5.5 0 0 1-.62-.62l.84-2.873a2 2 0 0 1 .506-.852z"/>`)
}

func iconSun() *ui.SVG {
	return loadIcon("sun", `<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>`)
}

func iconTarget() *ui.SVG {
	return loadIcon("target", `<circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/>`)
}

func iconTerminal() *ui.SVG {
	return loadIcon("terminal", `<path d="M12 19h8"/><path d="m4 17 6-6-6-6"/>`)
}

func iconTimer() *ui.SVG {
	return loadIcon("timer", `<line x1="10" x2="14" y1="2" y2="2"/><line x1="12" x2="15" y1="14" y2="11"/><circle cx="12" cy="14" r="8"/>`)
}

func iconTrash2() *ui.SVG {
	return loadIcon("trash-2", `<path d="M10 11v6"/><path d="M14 11v6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>`)
}

func iconTriangleAlert() *ui.SVG {
	return loadIcon("triangle-alert", `<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>`)
}

func iconUpload() *ui.SVG {
	return loadIcon("upload", `<path d="M12 3v12"/><path d="m17 8-5-5-5 5"/><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>`)
}

func iconX() *ui.SVG {
	return loadIcon("x", `<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)
}

func iconZoomIn() *ui.SVG {
	return loadIcon("zoom-in", `<circle cx="11" cy="11" r="8"/><line x1="21" x2="16.65" y1="21" y2="16.65"/><line x1="11" x2="11" y1="8" y2="14"/><line x1="8" x2="14" y1="11" y2="11"/>`)
}

func iconZoomOut() *ui.SVG {
	return loadIcon("zoom-out", `<circle cx="11" cy="11" r="8"/><line x1="21" x2="16.65" y1="21" y2="16.65"/><line x1="8" x2="14" y1="11" y2="11"/>`)
}
