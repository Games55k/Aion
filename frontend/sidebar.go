package frontend

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (w *Workbench) renderSidebar(c *ui.Context, tk *uiTokens) {
	bar := c.TitleBar()
	topPad := float32(14)
	if bar.Height > 0 {
		topPad = max(bar.Height+8, 42)
	}

	uiDraggableSideNav(c, &w.sideWidth, &w.sideExpanded, draggableSideNavProps{
		MinWidth:                78,
		MaxWidth:                340,
		ShowHandleWhenCollapsed: true,
		HighlightHandle:         true,
		Background:              tk.BgContainer,
		Header: func(nav sideNavState) {
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

				// Registered Module Switcher (shown in both expanded and collapsed sidebar modes)
				if len(w.modules) > 1 {
					if nav.Expanded {
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
									w.SwitchModule(mod.ID)
								}
							}
						})
					} else {
						ui.Column(c).Gap(4).Padding(4).Radius(tk.BorderRadius).Background(tk.FillQuaternary).AlignSelf(ui.Stretch).Children(func() {
							for _, mod := range w.modules {
								active := w.activeModule == mod.ID
								btn := ui.ButtonBase(c.Key("collapsed-"+mod.ID)).Height(28).PaddingX(6).
									Radius(tk.BorderRadiusSM).AlignItems(ui.Center).Justify(ui.Center).AlignSelf(ui.Stretch).
									Cursor(ui.CursorPointer).Label("切换模块 " + mod.Name)
								if active {
									btn.Background(tk.BgElevated).TextColor(tk.Text)
								} else {
									btn.TextColor(tk.TextSecondary)
								}
								btn.Children(func() {
									ui.Icon(c, ModuleIconSVG(mod)).FontSize(15)
								})
								if btn.Clicked() {
									w.SwitchModule(mod.ID)
								}
							}
						})
					}
				}
			})
		},
		Footer: func(nav sideNavState) {
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
				chosen := uiList(c, listProps{
					Active:    footerActive,
					Collapse:  nav.Collapse,
					RailWidth: nav.RailWidth,
					Items: []listItem{
						{
							Key:   "nav:trash",
							Label: "回收站",
							Icon:  iconTrash2(),
							Extra: trashExtra,
						},
						{
							Key:   "nav:settings",
							Label: "设置",
							Icon:  iconSettings(),
							Extra: "⌘,",
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
	}, func(nav sideNavState) {
		ui.Column(c).Padding(4, 8, 12, 8).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			if w.activeModule == "job_applications" {
				w.renderJobSidebar(c, tk, nav)
				return
			}
			w.renderProjectSidebar(c, tk, nav)
		})
	})
}

func (w *Workbench) renderJobSidebar(c *ui.Context, tk *uiTokens, nav sideNavState) {
	if nav.Expanded {
		ui.Row(c).Padding(8, 8, 2, 8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "投递记录").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary).SingleLine()
			if uiActionIcon(c, iconPlus(), actionIconProps{
				Title: "快速新增投递",
			}).Clicked() {
				w.OpenJobCreateModal()
			}
		})
	}
	countStr := func(n int) string {
		if n <= 0 {
			return ""
		}
		return fmt.Sprint(n)
	}
	activeKey := ""
	switch w.navPage {
	case NavJobAll:
		activeKey = "nav:job_all"
	case NavJobActive:
		activeKey = "nav:job_active"
	case NavJobOffer:
		activeKey = "nav:job_offer"
	case NavJobClosed:
		activeKey = "nav:job_closed"
	}
	chosen := uiList(c.Key("job-nav"), listProps{
		Active:    activeKey,
		Collapse:  nav.Collapse,
		RailWidth: nav.RailWidth,
		Items: []listItem{
			{
				Key:   "nav:job_all",
				Label: "全部投递",
				Icon:  iconBriefcase(),
				Extra: countStr(w.jobStats.Total),
			},
			{
				Key:   "nav:job_active",
				Label: "进行中",
				Icon:  iconClock(),
				Extra: countStr(w.jobStats.Active),
			},
			{
				Key:   "nav:job_offer",
				Label: "已获 Offer",
				Icon:  iconAward(),
				Extra: countStr(w.jobStats.Offered),
			},
			{
				Key:   "nav:job_closed",
				Label: "已结束",
				Icon:  iconFolderCheck(),
				Extra: countStr(w.jobStats.Closed),
			},
		},
	})
	switch chosen {
	case "nav:job_all":
		w.NavigateTo(NavJobAll)
	case "nav:job_active":
		w.NavigateTo(NavJobActive)
	case "nav:job_offer":
		w.NavigateTo(NavJobOffer)
	case "nav:job_closed":
		w.NavigateTo(NavJobClosed)
	}
}

func (w *Workbench) renderProjectSidebar(c *ui.Context, tk *uiTokens, nav sideNavState) {
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
	topChosen := uiList(c.Key("top-nav"), listProps{
		Active:    topActive,
		Collapse:  nav.Collapse,
		RailWidth: nav.RailWidth,
		Items: []listItem{
			{
				Key:   "nav:all_projects",
				Label: "所有项目",
				Icon:  iconLayoutGrid(),
				Extra: activeExtra,
			},
			{
				Key:   "nav:archived_projects",
				Label: "已归档项目",
				Icon:  iconArchive(),
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
			if uiActionIcon(c, iconFolderPlus(), actionIconProps{
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
		projItems := make([]listItem, 0, len(w.activeProjects))
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
						if uiActionIcon(c.Key("edit-"+p.ID), iconPencil(), actionIconProps{
							Title: "编辑项目: " + p.Name,
						}).Clicked() {
							w.OpenEditProjectModal(p)
						}
						if uiActionIcon(c.Key("archive-"+p.ID), iconArchive(), actionIconProps{
							Title: "归档项目: " + p.Name,
						}).Clicked() {
							w.ArchiveProject(p.ID)
						}
					})
				}
			}
			projItems = append(projItems, listItem{
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
		chosenProj := uiList(c.Key("active-projects-list"), listProps{
			Active:    activeProjKey,
			Collapse:  nav.Collapse,
			RailWidth: nav.RailWidth,
			Items:     projItems,
		})
		if len(chosenProj) > 5 && chosenProj[:5] == "proj:" {
			w.SelectProject(chosenProj[5:])
		}
	}
}
