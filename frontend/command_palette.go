package frontend

import (
	"fmt"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

type paletteItem struct {
	Key      string
	Group    string
	Title    string
	Subtitle string
	Badge    string
	BadgeCol tagColor
	Icon     *ui.SVG
	Shortcut string
	Run      func()
}

// OpenCommandPalette opens the global ⌘K command palette unless another modal is open
// or a backup restore is running.
func (w *Workbench) OpenCommandPalette() {
	if w.IsRestoring() || w.hasOpenModal() {
		return
	}
	w.cmdPaletteOpen = true
	w.cmdPaletteQuery = ""
	w.cmdPaletteIndex = 0
	w.cmdPaletteNeedFocus = true
	if !w.IsBusy() {
		w.reloadFocusTasks(true)
		w.reloadProjects(true)
		w.reloadJobApplications(true)
	}
}

// CloseCommandPalette closes the global command palette.
func (w *Workbench) CloseCommandPalette() {
	w.cmdPaletteOpen = false
	w.cmdPaletteNeedFocus = false
}

// ToggleCommandPalette toggles the global ⌘K command palette.
func (w *Workbench) ToggleCommandPalette() {
	if w.cmdPaletteOpen {
		w.CloseCommandPalette()
		return
	}
	w.OpenCommandPalette()
}

// CommandPaletteOpen reports whether the global command palette is open.
func (w *Workbench) CommandPaletteOpen() bool {
	return w.cmdPaletteOpen
}

func (w *Workbench) buildCommandPaletteItems(rawQuery string) []paletteItem {
	q := strings.ToLower(strings.TrimSpace(rawQuery))
	match := func(fields ...string) bool {
		if q == "" {
			return true
		}
		for _, f := range fields {
			if strings.Contains(strings.ToLower(f), q) {
				return true
			}
		}
		return false
	}

	showProjects := w.IsModuleVisible("projects")
	showJobs := w.IsModuleVisible("job_applications")

	items := make([]paletteItem, 0, 24)

	// 1. Quick Commands
	addCmd := func(key, title, subtitle, keywords, shortcut string, icon *ui.SVG, run func()) {
		if match(title, subtitle, keywords) {
			items = append(items, paletteItem{
				Key:      "cmd:" + key,
				Group:    "快捷操作",
				Title:    title,
				Subtitle: subtitle,
				Icon:     icon,
				Shortcut: shortcut,
				Run:      run,
			})
		}
	}

	addCmd("focus", "打开聚焦总览", "查看跨模块待推进任务与进行中投递", "focus today home 聚焦 总览 今日", "⌘1", iconTarget(), func() {
		w.NavigateTo(NavFocus)
	})
	if showProjects {
		addCmd("projects", "切换到项目工作区", "浏览活跃项目与三列看板", "projects board kanban 项目 看板", "⌘2", iconFolderKanban(), func() {
			w.SwitchModule("projects")
		})
		addCmd("new-project", "新建项目", "创建一个新的项目看板", "create new project 新建 创建 项目", "", iconFolderPlus(), func() {
			w.SwitchModule("projects")
			w.OpenCreateProjectModal()
		})
		if len(w.activeProjects) > 0 {
			addCmd("new-task", "新建任务", "在当前或首个活跃项目中创建任务", "create new task 新建 创建 任务", "", iconPlus(), func() {
				if w.activeProjectID == "" {
					w.SelectProject(w.activeProjects[0].ID)
				} else if w.navPage != NavProjectBoard {
					w.NavigateTo(NavProjectBoard)
				}
				w.OpenCreateTaskModal(model.Planned)
			})
		}
	}
	if showJobs {
		addCmd("jobs", "切换到求职工作区", "管理投递记录与阶段时间线", "jobs career applications 求职 投递 面试", "⌘3", iconBriefcase(), func() {
			w.SwitchModule("job_applications")
		})
		addCmd("new-job", "新增求职投递", "记录一家新的公司与岗位投递", "create new job application 新增 投递 求职", "", iconPlus(), func() {
			w.SwitchModule("job_applications")
			w.OpenJobCreateModal()
		})
	}
	addCmd("trash", "打开回收站", "查看并恢复已软删除的项目、任务或投递", "trash recycle restore 回收站 恢复", "", iconTrash2(), func() {
		w.NavigateTo(NavTrash)
	})
	addCmd("settings", "打开全局设置", "主题、模块显隐、备份、MCP 接入与更新", "settings preferences backup mcp 设置 备份", "⌘,", iconSettings(), func() {
		w.NavigateTo(NavSettings)
	})

	// 2. Projects
	if showProjects {
		maxProj := 4
		if q != "" {
			maxProj = 8
		}
		added := 0
		for _, proj := range w.activeProjects {
			p := proj
			if !match(p.Name, p.Description) {
				continue
			}
			sum := w.projectSummaries[p.ID]
			badge := ""
			if sum.Total > 0 {
				badge = fmt.Sprintf("%d/%d", sum.Done, sum.Total)
			}
			items = append(items, paletteItem{
				Key:      "proj:" + p.ID,
				Group:    "项目",
				Title:    p.Name,
				Subtitle: p.Description,
				Badge:    badge,
				BadgeCol: tagGray,
				Icon:     ProjectIconSVG(p.Icon),
				Run: func() {
					w.SelectProject(p.ID)
				},
			})
			added++
			if added >= maxProj {
				break
			}
		}
	}

	// 3. Tasks across active projects
	if showProjects {
		maxTasks := 4
		if q != "" {
			maxTasks = 10
		}
		added := 0
		for _, t := range w.focusTasks {
			task := t
			if q == "" && task.Status == model.Done {
				continue
			}
			projName := w.projectNames[task.ProjectID]
			tagsStr := strings.Join(task.Tags, " ")
			if !match(task.Title, task.Description, tagsStr, projName) {
				continue
			}
			sub := projName
			if task.DueDate != "" {
				if sub != "" {
					sub += " · "
				}
				sub += "截止 " + task.DueDate
			}
			badge, badgeCol := taskStatusBadge(task.Status)
			items = append(items, paletteItem{
				Key:      "task:" + task.ID,
				Group:    "任务",
				Title:    task.Title,
				Subtitle: sub,
				Badge:    badge,
				BadgeCol: badgeCol,
				Icon:     iconSquareCheckBig(),
				Run: func() {
					w.OpenFocusTask(task)
				},
			})
			added++
			if added >= maxTasks {
				break
			}
		}
	}

	// 4. Job Applications
	if showJobs {
		maxJobs := 4
		if q != "" {
			maxJobs = 10
		}
		added := 0
		for _, a := range w.jobApps {
			app := a
			if q == "" && app.Lifecycle != model.JobLifecycleActive {
				continue
			}
			if !match(app.CompanyName, app.PositionName, app.Channel, app.Notes, app.Stage.Label()) {
				continue
			}
			stageCol := tagProcessing
			if app.Lifecycle == model.JobLifecycleClosed {
				stageCol = tagGray
			} else if app.Stage == model.JobStageOffer {
				stageCol = tagGreen
			}
			items = append(items, paletteItem{
				Key:      "job:" + app.ID,
				Group:    "求职投递",
				Title:    app.CompanyName + " · " + app.PositionName,
				Subtitle: "投递于 " + app.AppliedDate,
				Badge:    app.Stage.Label(),
				BadgeCol: stageCol,
				Icon:     iconBriefcase(),
				Run: func() {
					w.OpenFocusJob(app.ID)
				},
			})
			added++
			if added >= maxJobs {
				break
			}
		}
	}

	return items
}

func taskStatusBadge(st model.Status) (string, tagColor) {
	switch st {
	case model.InProgress:
		return "进行中", tagProcessing
	case model.Done:
		return "已完成", tagGreen
	default:
		return "计划中", tagGray
	}
}

func (w *Workbench) executePaletteItem(it paletteItem) {
	w.CloseCommandPalette()
	if it.Run != nil {
		it.Run()
	}
}

func (w *Workbench) renderCommandPaletteModal(c *ui.Context, tk *uiTokens) {
	m := modalTransition()
	dialogPresent(c.Key("command-palette"), &w.cmdPaletteOpen, motionOfDialog(m), func(backdrop, panel ui.Element, shown, dim float32, closing bool) {
		items := w.buildCommandPaletteItems(w.cmdPaletteQuery)
		if len(items) == 0 {
			w.cmdPaletteIndex = 0
		} else {
			if w.cmdPaletteIndex < 0 {
				w.cmdPaletteIndex = 0
			}
			if w.cmdPaletteIndex >= len(items) {
				w.cmdPaletteIndex = len(items) - 1
			}
		}

		if !closing {
			switch {
			case panel.OverlayShortcut(0, ui.KeyEscape):
				w.CloseCommandPalette()
			case panel.OverlayShortcut(0, ui.KeyDown) && len(items) > 0:
				w.cmdPaletteIndex = (w.cmdPaletteIndex + 1) % len(items)
			case panel.OverlayShortcut(0, ui.KeyUp) && len(items) > 0:
				w.cmdPaletteIndex = (w.cmdPaletteIndex - 1 + len(items)) % len(items)
			case panel.OverlayShortcut(0, ui.KeyEnter) && len(items) > 0:
				w.executePaletteItem(items[w.cmdPaletteIndex])
			}
		}

		backdrop.Padding(72, 16, 32, 16).AlignItems(ui.Center).Justify(ui.Start).
			Background(tk.BgContainer.Alpha(0.45 * dim))
		from := m.Enter.Scale
		if closing {
			from = m.Exit.Scale
		}
		scale := from + (1-from)*shown
		panel.Width(620).MaxWidthPercent(100).MaxHeight(520).Radius(12).Clip().
			Background(tk.BgElevated).Border(1, tk.BorderSecondary).TextColor(tk.Text).FontSize(tk.FontSize).
			Scale(scale, scale).Opacity(shown)
		tk.Elevate(panel, boxShadow)

		panel.Children(func() {
			// Search Bar Header
			ui.Row(c).Padding(12, 14).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).
				BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
				ui.Icon(c, iconSearch()).FontSize(17).TextColor(tk.TextSecondary).Shrink(0)
				beforeQ := w.cmdPaletteQuery
				inp := ui.TextInputBase(c.Key("cmd-palette-input"), &w.cmdPaletteQuery).
					FocusRing(false).
					Placeholder("搜索项目、任务、求职投递或快捷命令…").
					Label("全局命令面板搜索").
					Grow(1).MinWidth(0).FontSize(15).TextColor(tk.Text)
				if w.cmdPaletteNeedFocus && !closing {
					inp.Focus()
					w.cmdPaletteNeedFocus = false
				}
				if w.cmdPaletteQuery != beforeQ {
					w.cmdPaletteIndex = 0
				}
				if inp.Submitted() && len(items) > 0 && !closing {
					w.executePaletteItem(items[w.cmdPaletteIndex])
				}
				uiTag(c, "Esc", tagProps{
					Size:    sizeSmall,
					Variant: tagOutlined,
					Color:   tagGray,
				})
			})

			// Results List
			ui.Scroll(c.Key("cmd-palette-scroll")).Shrink(1).MaxHeight(400).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Padding(8).Gap(4).AlignSelf(ui.Stretch).Children(func() {
					if len(items) == 0 {
						ui.Column(c).Padding(32, 16).Gap(6).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
							ui.Text(c, "没有匹配的项目、任务、投递或命令").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
							ui.Text(c, "试试搜索项目名、任务标题、公司名或输入“新建”“设置”。").FontSize(12).TextColor(tk.TextTertiary)
						})
						return
					}
					lastGroup := ""
					for i, it := range items {
						item := it
						idx := i
						if item.Group != lastGroup {
							lastGroup = item.Group
							topPad := float32(6)
							if idx == 0 {
								topPad = 2
							}
							ui.Row(c.Key("grp-"+item.Group)).Padding(topPad, 10, 2, 10).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, item.Group).FontSize(11).FontWeight(600).TextColor(tk.TextTertiary).SingleLine()
							})
						}
						selected := idx == w.cmdPaletteIndex
						btn := ui.ButtonBase(c.Key(item.Key)).Padding(8, 10).Gap(10).Radius(tk.BorderRadius).
							AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).
							Cursor(ui.CursorPointer).Label("命令面板项 " + item.Title)
						if selected {
							btn.Background(tk.PrimaryFillQuaternary)
						}
						btn.Children(func() {
							ui.Row(c).Gap(10).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
								iconCol := tk.TextSecondary
								if selected {
									iconCol = tk.Primary
								}
								ui.Icon(c, item.Icon).FontSize(15).TextColor(iconCol).Shrink(0)
								ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
									ui.Text(c, item.Title).FontSize(13).FontWeight(500).TextColor(tk.Text).SingleLine()
									if item.Subtitle != "" {
										ui.Text(c, item.Subtitle).FontSize(11).TextColor(tk.TextTertiary).SingleLine()
									}
								})
							})
							ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
								if item.Badge != "" {
									uiTag(c, item.Badge, tagProps{
										Size:    sizeSmall,
										Variant: tagFilled,
										Color:   item.BadgeCol,
									})
								}
								if item.Shortcut != "" {
									uiTag(c, item.Shortcut, tagProps{
										Size:    sizeSmall,
										Variant: tagOutlined,
										Color:   tagGray,
									})
								}
							})
						})
						if btn.Hovered() && w.cmdPaletteIndex != idx {
							w.cmdPaletteIndex = idx
						}
						if btn.Clicked() {
							w.executePaletteItem(item)
						}
					}
				})
			})

			// Footer Hints
			ui.Row(c).Padding(8, 14).Gap(14).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).
				Background(tk.FillQuaternary).BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
				ui.Text(c, "↑↓ 选择  ·  ↵ 执行  ·  Esc 关闭").FontSize(11).TextColor(tk.TextTertiary).SingleLine()
				ui.Text(c, fmt.Sprintf("%d 项结果", len(items))).FontSize(11).TextColor(tk.TextTertiary).SingleLine()
			})
		})
	})
}
