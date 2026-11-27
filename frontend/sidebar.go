package frontend

import (
	"fmt"
	"time"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

type topNavSlideMatch struct{}

var topNavSlideTransition = ui.ElementTransition{
	Duration: 260 * time.Millisecond,
	Ease:     ui.CubicBezier(0.32, 0.72, 0, 1),
	Match:    topNavSlideMatch{},
}

func (w *Workbench) renderTopbar(c *ui.Context, tk *uiTokens) {
	bar := c.TitleBar()
	leftPad := float32(16)
	if bar.Left > 0 {
		leftPad = max(bar.Left+16, 84)
	}
	rightPad := float32(16)
	if bar.Right > 0 {
		rightPad = max(bar.Right+16, 16)
	}
	topHeight := float32(64)
	if bar.Height > topHeight {
		topHeight = bar.Height + 16
	}

	ui.Row(c.Key("aion-topbar")).Height(topHeight).Padding(0, rightPad, 0, leftPad).Gap(8).
		AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).
		Background(tk.BgLayout).DragWindow().Children(func() {
		// Left: Clean macOS traffic-light breathing space (shrinks gracefully on narrow windows)
		ui.Box(c).Grow(1).Basis(0).MinWidth(0)

		// Center: Spring-Gliding Floating Navigation Capsule
		w.renderTopNavCapsule(c, tk)

		// Right: Update (when available) + Search + Trash + Feedback + Theme + Universal Create Menu
		ui.Row(c).Gap(4).AlignItems(ui.Center).Justify(ui.End).Grow(1).Basis(0).MinWidth(256).Children(func() {
			updateInfo := w.UpdateCapability()
			if hasSidebarUpdateIndicator(updateInfo) {
				w.renderSidebarUpdateButton(c, tk, updateInfo)
			}

			searchBtn := ui.ButtonBase(c.Key("topbar-cmd-palette")).Size(32, 32).Radius(12).
				Center().Cursor(ui.CursorPointer).Label("打开全局命令面板")
			if searchBtn.Hovered() {
				searchBtn.Background(tk.FillTertiary).TextColor(tk.Text)
			} else {
				searchBtn.TextColor(tk.TextSecondary)
			}
			searchBtn.Children(func() { uiIconBox(c, iconSearch(), ui.Color{}) })
			uiTooltip(c, searchBtn, "搜索 ⌘K")
			if searchBtn.Clicked() {
				w.OpenCommandPalette()
			}

			trashActive := w.navPage == NavTrash
			trashBtn := ui.ButtonBase(c.Key("topbar-trash")).Height(32).PaddingX(6).Gap(4).Radius(12).
				AlignItems(ui.Center).Justify(ui.Center).Cursor(ui.CursorPointer).Label("回收站")
			switch {
			case trashActive:
				trashBtn.Background(tk.FillTertiary).TextColor(tk.Primary)
			case trashBtn.Hovered():
				trashBtn.Background(tk.FillTertiary).TextColor(tk.Text)
			default:
				trashBtn.TextColor(tk.TextSecondary)
			}
			trashBtn.Children(func() {
				uiIconBox(c, iconTrash2(), ui.Color{})
				if len(w.trashItems) > 0 {
					ui.Text(c, fmt.Sprint(len(w.trashItems))).Font("monospace").FontSize(14).SingleLine()
				}
			})
			uiTooltip(c, trashBtn, "回收站")
			if trashBtn.Clicked() {
				w.NavigateTo(NavTrash)
			}

			feedbackBtn := ui.ButtonBase(c.Key("topbar-feedback")).Size(32, 32).Radius(12).
				Center().Cursor(ui.CursorPointer).Label("问题反馈")
			if feedbackBtn.Hovered() {
				feedbackBtn.Background(tk.FillTertiary).TextColor(tk.Text)
			} else {
				feedbackBtn.TextColor(tk.TextSecondary)
			}
			feedbackBtn.Children(func() { uiIconBox(c, iconMessageSquare(), ui.Color{}) })
			uiTooltip(c, feedbackBtn, "问题反馈 (GitHub Issues)")
			if feedbackBtn.Clicked() {
				w.OpenFeedbackPage()
			}

			themeIcon := iconMoon()
			if tk.Dark {
				themeIcon = iconSun()
			}
			themeBtn := ui.ButtonBase(c.Key("topbar-theme")).Size(32, 32).Radius(12).
				Center().Cursor(ui.CursorPointer).Label("切换深浅外观")
			if themeBtn.Hovered() {
				themeBtn.Background(tk.FillTertiary).TextColor(tk.Text)
			} else {
				themeBtn.TextColor(tk.TextSecondary)
			}
			themeBtn.Children(func() { uiIconBox(c, themeIcon, ui.Color{}) })
			uiTooltip(c, themeBtn, "切换外观")
			if themeBtn.Clicked() {
				if tk.Dark {
					w.ApplyThemeSetting(ThemeLight)
				} else {
					w.ApplyThemeSetting(ThemeDark)
				}
			}

			createItems := w.buildCreateMenuItems()
			chosen := uiDropdownMenu(c.Key("topbar-create-menu"), &w.topMenuOpen, dropdownMenuProps{
				Items: createItems,
			}, func(t ui.Element) {
				t.Height(32).Padding(0, 12, 0, 8).Gap(4).Radius(40).
					AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
					Cursor(ui.CursorPointer).Label("快速新建")
				bg := tk.Primary
				if t.Pressed() || w.topMenuOpen {
					bg = tk.PrimaryActive
				} else if t.Hovered() {
					bg = tk.PrimaryHover
				}
				t.Background(bg).TextColor(ui.RGB(255, 255, 255))
				t.Children(func() {
					uiIconBox(c, iconPlus(), ui.RGB(255, 255, 255))
					ui.Text(c, "新建").FontSize(14).FontWeight(500).SingleLine()
				})
			})
			w.handleCreateMenuChoice(chosen)
		})
	})
}

func (w *Workbench) renderTopNavCapsule(c *ui.Context, tk *uiTokens) {
	type navTabSpec struct {
		Key    string
		Text   string
		Label  string
		Icon   *ui.SVG
		Count  string
		Active bool
		OnTap  func()
	}

	focusCount := ""
	if n := w.focusBadgeCount(); n > 0 {
		focusCount = fmt.Sprint(n)
	}
	tabs := []navTabSpec{
		{
			Key:    "focus",
			Text:   "今天",
			Label:  "聚焦",
			Icon:   iconTarget(),
			Count:  focusCount,
			Active: w.navPage == NavFocus,
			OnTap:  func() { w.NavigateTo(NavFocus) },
		},
	}

	if w.IsModuleVisible("projects") {
		tabs = append(tabs, navTabSpec{
			Key:    "projects",
			Text:   "项目",
			Label:  "切换模块 项目",
			Icon:   iconFolderKanban(),
			Active: isProjectNavPage(w.navPage),
			OnTap:  func() { w.SwitchModule("projects") },
		})
	}

	if w.IsModuleVisible("job_applications") {
		jobCount := ""
		if w.jobStats.Active > 0 {
			jobCount = fmt.Sprint(w.jobStats.Active)
		}
		tabs = append(tabs, navTabSpec{
			Key:    "jobs",
			Text:   "求职",
			Label:  "切换模块 求职",
			Icon:   iconBriefcase(),
			Count:  jobCount,
			Active: isJobNavPage(w.navPage),
			OnTap:  func() { w.SwitchModule("job_applications") },
		})
	}

	tabs = append(tabs, navTabSpec{
		Key:    "settings",
		Text:   "设置",
		Label:  "设置",
		Icon:   iconSettings(),
		Active: w.navPage == NavSettings,
		OnTap:  func() { w.NavigateTo(NavSettings) },
	})

	ui.Row(c.Key("top-nav-capsule")).Padding(4).Gap(2).Radius(40).
		Background(tk.FillTertiary).AlignItems(ui.Center).Shrink(0).Children(func() {
		for _, tab := range tabs {
			btn := ui.ButtonBase(c.Key("top-tab-" + tab.Key)).Height(32).PaddingX(10).Gap(4).
				Radius(40).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
				Cursor(ui.CursorPointer).Label(tab.Label)
			fg := tk.TextSecondary
			if tab.Active || btn.Hovered() {
				fg = tk.Text
			}
			btn.TextColor(fg)
			btn.Children(func() {
				if tab.Active {
					ind := ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
						Radius(40).Background(tk.BgElevated).ZIndex(-1).
						Transition(topNavSlideTransition)
					if !tk.Dark {
						ind.Shadow(0, 2, 8, 0, ui.RGBA(24, 24, 24, 0.08))
					}
				}
				uiIconBox(c, tab.Icon, fg)
				ui.Text(c, tab.Text).FontSize(14).FontWeight(500).SingleLine()
				if tab.Count != "" {
					cntColor := tk.TextSecondary
					if tab.Active {
						cntColor = tk.Primary
					}
					ui.Text(c, tab.Count).Font("monospace").FontSize(14).TextColor(cntColor).SingleLine()
				}
			})
			if btn.Clicked() {
				tab.OnTap()
			}
		}
	})
}

func (w *Workbench) buildCreateMenuItems() []menuItem {
	items := make([]menuItem, 0, 3)
	if w.IsModuleVisible("projects") {
		if w.ActiveProject() != nil || len(w.activeProjects) > 0 {
			items = append(items, menuItem{Key: "new-task", Label: "新建任务", Icon: iconSquareCheckBig()})
		}
		items = append(items, menuItem{Key: "new-project", Label: "新建项目", Icon: iconFolderPlus()})
	}
	if w.IsModuleVisible("job_applications") {
		items = append(items, menuItem{Key: "new-job", Label: "新增投递", Icon: iconBriefcase()})
	}
	return items
}

func (w *Workbench) handleCreateMenuChoice(chosen string) {
	if chosen == "" {
		return
	}
	w.topMenuOpen = false
	switch chosen {
	case "new-task":
		if w.ActiveProject() == nil && len(w.activeProjects) > 0 {
			w.activeProjectID = w.activeProjects[0].ID
		}
		w.OpenCreateTaskModal(model.Planned)
	case "new-project":
		w.OpenCreateProjectModal()
	case "new-job":
		w.OpenJobCreateModal()
	}
}
