package frontend

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (w *Workbench) renderSidebar(c *ui.Context, tk *theme.Tokens) {
	bar := c.TitleBar()
	topPad := float32(14)
	if bar.Height > 0 {
		topPad = max(bar.Height+8, 42)
	}

	layout.DraggableSideNav(c, &w.sideWidth, &w.sideExpanded, layout.DraggableSideNavProps{
		MinWidth:                78,
		MaxWidth:                340,
		ShowHandleWhenCollapsed: true,
		HighlightHandle:         true,
		Background:              tk.BgContainer,
		Header: func(nav layout.SideNavState) {
			ui.Column(c).Padding(topPad, 12, 10, 12).Gap(12).AlignSelf(ui.Stretch).DragWindow().Children(func() {
				// Brand Row
				brandRow := ui.Row(c).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch)
				if !nav.Expanded {
					brandRow.Justify(ui.Center)
				}
				brandRow.Children(func() {
					ui.Image(c, AionSymbolSVG()).Size(26, 26).Shrink(0)
					if nav.Expanded {
						ui.Text(c, "Aion").FontSize(17).FontWeight(700).TextColor(tk.Text).SingleLine()
					}
				})

				// Registered Module Switcher (only shown when multiple modules are registered)
				if len(w.modules) > 1 && nav.Expanded {
					ui.Row(c).Gap(6).Padding(4).Radius(tk.BorderRadius).Background(tk.FillQuaternary).AlignSelf(ui.Stretch).Children(func() {
						for _, mod := range w.modules {
							active := w.activeModule == mod.ID
							btn := ui.ButtonBase(c.Key(mod.ID)).Grow(1).Height(28).PaddingX(10).Gap(6).
								Radius(tk.BorderRadiusSM).AlignItems(ui.Center).Justify(ui.Center).Cursor(ui.CursorPointer).
								Label("切换模块 " + mod.Name)
							if active {
								btn.Background(tk.BgElevated).TextColor(tk.Text).FontWeight(600)
							} else {
								btn.TextColor(tk.TextSecondary)
							}
							btn.Children(func() {
								ui.Icon(c, ModuleIconSVG(mod)).FontSize(14)
								ui.Text(c, mod.Name).FontSize(12).SingleLine()
							})
							if btn.Clicked() {
								w.activeModule = mod.ID
								if w.navPage == NavTrash || w.navPage == NavSettings {
									w.NavigateTo(NavAllProjects)
								}
							}
						}
					})
				}

				// Quick New Project Button
				if nav.Expanded {
					if action.Button(c, "+ 新建项目", action.ButtonProps{
						Variant: action.Filled,
						Size:    theme.Middle,
					}).AlignSelf(ui.Stretch).Clicked() {
						w.OpenCreateProjectModal()
					}
				} else {
					ui.Row(c).Justify(ui.Center).AlignSelf(ui.Stretch).Children(func() {
						if action.ActionIcon(c, lucide.Plus(), action.ActionIconProps{
							Variant: action.ActionIconFilled,
							Size:    theme.Middle,
							Title:   "新建项目",
						}).Clicked() {
							w.OpenCreateProjectModal()
						}
					})
				}
			})
		},
		Footer: func(nav layout.SideNavState) {
			ui.Column(c).Padding(8).Gap(4).AlignSelf(ui.Stretch).BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
				trashExtra := ""
				if len(w.trashItems) > 0 {
					trashExtra = fmt.Sprint(len(w.trashItems))
				}
				footerActive := ""
				switch w.navPage {
				case NavTrash:
					footerActive = "nav:trash"
				case NavSettings:
					footerActive = "nav:settings"
				}
				chosen := display.List(c, display.ListProps{
					Active:    footerActive,
					Collapse:  nav.Collapse,
					RailWidth: nav.RailWidth,
					Items: []display.ListItem{
						{
							Key:   "nav:trash",
							Label: "回收站",
							Icon:  lucide.Trash2(),
							Extra: trashExtra,
						},
						{
							Key:   "nav:settings",
							Label: "设置",
							Icon:  lucide.Settings(),
						},
					},
				})
				switch chosen {
				case "nav:trash":
					w.NavigateTo(NavTrash)
				case "nav:settings":
					w.NavigateTo(NavSettings)
				}
			})
		},
	}, func(nav layout.SideNavState) {
		ui.Column(c).Padding(4, 8, 12, 8).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			// Top Overview & Archived Navigation
			topActive := ""
			switch w.navPage {
			case NavAllProjects:
				topActive = "nav:all_projects"
			case NavArchivedProjects:
				topActive = "nav:archived_projects"
			}
			activeExtra := ""
			if len(w.activeProjects) > 0 {
				activeExtra = fmt.Sprint(len(w.activeProjects))
			}
			archivedExtra := ""
			if len(w.archivedProjects) > 0 {
				archivedExtra = fmt.Sprint(len(w.archivedProjects))
			}
			topChosen := display.List(c.Key("top-nav"), display.ListProps{
				Active:    topActive,
				Collapse:  nav.Collapse,
				RailWidth: nav.RailWidth,
				Items: []display.ListItem{
					{
						Key:   "nav:all_projects",
						Label: "所有项目",
						Icon:  lucide.LayoutGrid(),
						Extra: activeExtra,
					},
					{
						Key:   "nav:archived_projects",
						Label: "已归档项目",
						Icon:  lucide.Archive(),
						Extra: archivedExtra,
					},
				},
			})
			switch topChosen {
			case "nav:all_projects":
				w.NavigateTo(NavAllProjects)
			case "nav:archived_projects":
				w.NavigateTo(NavArchivedProjects)
			}

			// Subtle Divider between Global Nav and Project List
			if nav.Expanded {
				ui.Box(c).Height(1).Margin(6, 6).Background(tk.BorderSecondary)
			}

			// Active Projects Section Header
			if nav.Expanded {
				ui.Row(c).Padding(8, 8, 2, 8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "项目列表").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary).SingleLine()
					if action.ActionIcon(c, lucide.FolderPlus(), action.ActionIconProps{
						Size:  theme.Small,
						Title: "创建项目",
					}).Clicked() {
						w.OpenCreateProjectModal()
					}
				})
			}

			// Active Projects List
			if len(w.activeProjects) == 0 && nav.Expanded {
				ui.Column(c).Padding(12, 8).Gap(4).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "暂无进行中的项目").FontSize(12).TextColor(tk.TextTertiary)
				})
			} else {
				projItems := make([]display.ListItem, 0, len(w.activeProjects))
				for _, proj := range w.activeProjects {
					p := proj
					sum := w.projectSummaries[p.ID]
					extra := ""
					if sum.Total > 0 {
						extra = fmt.Sprintf("%d/%d", sum.Done, sum.Total)
					}
					var actions func()
					if nav.Expanded {
						isProjActive := w.navPage == NavProjectBoard && w.activeProjectID == p.ID
						actions = func() {
							bg := tk.FillTertiary.Over(tk.BgContainer)
							if isProjActive {
								bg = tk.FillSecondary.Over(tk.BgContainer)
							}
							ui.Row(c).Height(28).Background(bg).PaddingX(2).Radius(tk.BorderRadiusSM).Gap(2).AlignItems(ui.Center).Children(func() {
								if action.ActionIcon(c.Key("edit-"+p.ID), lucide.Pencil(), action.ActionIconProps{
									Size:  theme.Small,
									Title: "编辑项目: " + p.Name,
								}).Clicked() {
									w.OpenEditProjectModal(p)
								}
								if action.ActionIcon(c.Key("archive-"+p.ID), lucide.Archive(), action.ActionIconProps{
									Size:  theme.Small,
									Title: "归档项目: " + p.Name,
								}).Clicked() {
									w.ArchiveProject(p.ID)
								}
							})
						}
					}
					projItems = append(projItems, display.ListItem{
						Key:     "proj:" + p.ID,
						Label:   p.Name,
						Icon:    ProjectIconSVG(p.Icon),
						Extra:   extra,
						Actions: actions,
					})
				}
				activeProjKey := ""
				if w.navPage == NavProjectBoard && w.activeProjectID != "" {
					activeProjKey = "proj:" + w.activeProjectID
				}
				chosenProj := display.List(c.Key("active-projects-list"), display.ListProps{
					Active:    activeProjKey,
					Collapse:  nav.Collapse,
					RailWidth: nav.RailWidth,
					Items:     projItems,
				})
				if len(chosenProj) > 5 && chosenProj[:5] == "proj:" {
					w.SelectProject(chosenProj[5:])
				}
			}
		})
	})
}
