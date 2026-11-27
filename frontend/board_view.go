package frontend

import (
	"fmt"
	"slices"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

type boardColumnSpec struct {
	Status model.Status
	Title  string
	Dot    ui.Color
	Tag    tagColor
}

var (
	taskStatusOptions = []selectOption{
		{Value: string(model.Planned), Label: "计划"},
		{Value: string(model.InProgress), Label: "进行中"},
		{Value: string(model.Done), Label: "已完成"},
	}
	taskPriorityOptions = []selectOption{
		{Value: string(model.Urgent), Label: "紧急"},
		{Value: string(model.High), Label: "高"},
		{Value: string(model.Medium), Label: "中"},
		{Value: string(model.Low), Label: "低"},
		{Value: string(model.NoPriority), Label: "无优先级"},
	}
	priorityFilterOptions = append([]selectOption{{Value: "", Label: "全部优先级"}}, taskPriorityOptions...)
	dueDateFilterOptions  = []selectOption{
		{Value: "", Label: "全部日期"},
		{Value: "overdue", Label: "已逾期"},
		{Value: "due_soon", Label: "7 天内截止"},
		{Value: "none", Label: "无截止日期"},
	}
)

func (w *Workbench) renderBoardView(c *ui.Context, tk *uiTokens) {
	proj := w.ActiveProject()
	if proj == nil {
		ui.Column(c).Fill().Center().Padding(32).Children(func() {
			uiEmpty(c, emptyProps{
				Icon:        iconFolderKanban(),
				Title:       "未选择项目",
				Description: "请从顶部导航或菜单选择一个项目，或创建一个新项目。",
				Action: func() {
					if uiButton(c, "新建项目", buttonProps{Variant: buttonPrimary}).Clicked() {
						w.OpenCreateProjectModal()
					}
				},
			})
		})
		return
	}
	p := *proj

	ui.Column(c).Fill().Background(tk.BgLayout).Children(func() {
		// 1. Clean Two-Row Board Header:
		//    Row 1: Project switcher pills (zero-shift hover edit slot) + subtle project actions
		//    Row 2: Task search / priority / due-date / tag filters + "+ 新建任务"
		ui.Column(c).Padding(16, 48, 16, 48).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			// Row 1: Project Switcher & Project Actions
			ui.Row(c).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				// Active Project Pills + All Projects & New Project Shortcuts
				ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Grow(1).MinWidth(0).Children(func() {
					if uiActionIcon(c.Key("board-all-projects"), iconLayoutGrid(), actionIconProps{
						Title: "所有项目",
					}).Clicked() {
						w.NavigateTo(NavAllProjects)
					}
					for _, ap := range w.activeProjects {
						projItem := ap
						active := projItem.ID == p.ID
						sum := w.projectSummaries[projItem.ID]
						pillWrap := ui.Row(c.Key("board-proj-wrap-" + projItem.ID)).Height(36).Padding(0, 10, 0, 14).Gap(6).
							Radius(40).AlignItems(ui.Center)
						hovered := pillWrap.Hovered()
						if active || hovered {
							pillWrap.Background(tk.FillTertiary)
						}
						pillWrap.Children(func() {
							btn := ui.ButtonBase(c.Key("board-proj-" + projItem.ID)).
								AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(projItem.Name)
							if active {
								btn.TextColor(tk.Text).FontWeight(600)
							} else if hovered {
								btn.TextColor(tk.Text).FontWeight(500)
							} else {
								btn.TextColor(tk.TextSecondary).FontWeight(500)
							}
							btn.Children(func() {
								ui.Text(c, projItem.Name).FontSize(14).SingleLine()
							})
							if btn.Clicked() && !active {
								w.SelectProject(projItem.ID)
							}

							// Fixed-footprint trailing slot: swaps count with edit icon on hover with 0px width shift.
							ui.Box(c).MinWidth(28).Height(28).Center().Shrink(0).Children(func() {
								cnt := ui.Text(c, fmt.Sprintf("%d/%d", sum.Done, sum.Total)).
									Font("monospace").FontSize(13).TextColor(tk.TextSecondary).SingleLine()
								if hovered {
									cnt.Opacity(0)
									editBtn := uiActionIcon(c.Key("edit-proj-"+projItem.ID), iconPencil(), actionIconProps{
										Title: "编辑项目: " + projItem.Name,
									}).Absolute()
									if editBtn.Clicked() {
										w.OpenEditProjectModal(projItem)
									}
								}
							})
						})
					}
					if p.Archived {
						ui.Row(c).Height(36).PaddingX(14).Gap(8).Radius(40).Background(tk.FillTertiary).AlignItems(ui.Center).Children(func() {
							ui.Text(c, p.Name).FontSize(14).FontWeight(600).TextColor(tk.Text).SingleLine()
							ui.Text(c, "已归档").FontSize(14).TextColor(tk.Primary).SingleLine()
						})
					}
					if len(w.archivedProjects) > 0 {
						archBtn := ui.ButtonBase(c.Key("board-archived-link")).Height(36).PaddingX(12).Gap(8).
							Radius(40).AlignItems(ui.Center).Cursor(ui.CursorPointer).
							TextColor(tk.TextSecondary).Label("已归档项目")
						if archBtn.Hovered() {
							archBtn.Background(tk.FillTertiary).TextColor(tk.Text)
						}
						archBtn.Children(func() {
							ui.Text(c, fmt.Sprintf("已归档 (%d)", len(w.archivedProjects))).FontSize(14).SingleLine()
						})
						if archBtn.Clicked() {
							w.NavigateTo(NavArchivedProjects)
						}
					}
					if uiActionIcon(c.Key("board-new-project"), iconPlus(), actionIconProps{
						Title: "新建项目",
					}).Clicked() {
						w.OpenCreateProjectModal()
					}
				})

				// Right: Subtle Project Management Actions
				ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
					if uiButton(c, "编辑项目", buttonProps{Variant: buttonText, Size: sizeSmall}).Clicked() {
						w.OpenEditProjectModal(p)
					}
					if p.Archived {
						if uiButton(c, "恢复归档", buttonProps{Variant: buttonText, Size: sizeSmall}).Clicked() {
							w.UnarchiveProject(p.ID)
						}
					} else {
						if uiButton(c, "归档项目", buttonProps{Variant: buttonText, Size: sizeSmall}).Clicked() {
							w.ArchiveProject(p.ID)
						}
					}
					if uiButton(c, "移入回收站", buttonProps{Variant: buttonText, Size: sizeSmall, Danger: true}).
						Label("将项目移入回收站").Clicked() {
						w.SoftDeleteProject(p.ID)
					}
				})
			})

			// Row 2: Task Filter Controls + Inline Tag Pills + Create Task CTA
			ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
					ui.Box(c).Width(180).Children(func() {
						beforeQ := w.searchQuery
						uiSearchBar(c, &w.searchQuery, searchBarProps{
							Placeholder: "筛选",
							ShortKey:    "mod+f",
							Size:        sizeSmall,
						}).Label("搜索任务")
						if w.searchQuery != beforeQ {
							w.applyBoardFilters()
						}
					})

					ui.Box(c).Width(116).Children(func() {
						beforeP := w.priorityFilter
						uiSelect(c.Key("priority-filter"), &w.priorityFilter, selectProps{
							Placeholder: "优先级",
							Size:        sizeSmall,
							Options:     priorityFilterOptions,
						}).Label("按优先级筛选")
						if w.priorityFilter != beforeP {
							w.applyBoardFilters()
						}
					})

					ui.Box(c).Width(116).Children(func() {
						beforeD := w.dueDateFilter
						uiSelect(c.Key("due-filter"), &w.dueDateFilter, selectProps{
							Placeholder: "截止日期",
							Size:        sizeSmall,
							Options:     dueDateFilterOptions,
						}).Label("按截止日期筛选")
						if w.dueDateFilter != beforeD {
							w.applyBoardFilters()
						}
					})

					for _, tag := range w.allBoardTags {
						active := slices.Contains(w.tagFilters, tag)
						pill := ui.ButtonBase(c.Key("filter-tag-" + tag)).Height(28).PaddingX(12).Radius(40).
							FontSize(14).AlignItems(ui.Center).Cursor(ui.CursorPointer)
						if active {
							pill.Background(tk.Primary).TextColor(ui.RGB(255, 255, 255)).FontWeight(500)
						} else {
							pill.Background(tk.FillTertiary).TextColor(tk.TextSecondary)
						}
						pill.Label("筛选标签 " + tag)
						pill.Children(func() {
							ui.Text(c, "#"+tag).SingleLine()
						})
						if pill.Clicked() {
							w.ToggleTagFilter(tag)
						}
					}
					if w.HasBoardFilters() {
						if uiButton(c, "清除筛选", buttonProps{Variant: buttonText, Size: sizeSmall}).Clicked() {
							w.ClearBoardFilters()
						}
					}
				})

				ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
					if uiButton(c, "+ 新建任务", buttonProps{Variant: buttonDefault, Size: sizeSmall}).Clicked() {
						w.OpenCreateTaskModal(model.Planned)
					}
				})
			})
		})

		// 2. Three-Column Flat Neutral Kanban Lanes
		columns := []boardColumnSpec{
			{Status: model.Planned, Title: "计划", Dot: tk.TextSecondary, Tag: tagGray},
			{Status: model.InProgress, Title: "进行中", Dot: tk.Primary, Tag: tagProcessing},
			{Status: model.Done, Title: "已完成", Dot: tk.TextSecondary, Tag: tagGray},
		}

		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Padding(0, 48, 36, 48).Gap(24).AlignItems(ui.Stretch).
			AlignSelf(ui.Stretch).Children(func() {
			for _, col := range columns {
				w.renderBoardColumn(c.Key(string(col.Status)), tk, col)
			}
		})
	})
}

func (w *Workbench) renderBoardColumn(c *ui.Context, tk *uiTokens, col boardColumnSpec) {
	tasks := w.TasksInColumn(col.Status)

	colBox := ui.Column(c).Grow(1).Basis(0).MinWidth(200).AlignSelf(ui.Stretch).
		Radius(24).Background(tk.FillTertiary).Padding(16, 12, 16, 12).
		Label("看板列 " + col.Title)

	if payload, over := ui.DragOver[DragTaskPayload](colBox); over && payload.FromStatus != col.Status {
		colBox.Background(tk.Fill)
	}
	if payload, dropped := ui.Drop[DragTaskPayload](colBox); dropped && payload.FromStatus != col.Status {
		w.MoveOrReorderTaskToSlot(payload.TaskID, col.Status, 0)
	}

	colBox.Children(func() {
		// Column Header (18px Bold + Count + Plus)
		ui.Row(c).Padding(0, 8, 8, 8).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, col.Title).FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
				ui.Text(c, fmt.Sprint(len(tasks))).Font("monospace").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary).SingleLine()
			})

			if uiActionIcon(c, iconPlus(), actionIconProps{
				Title: "在" + col.Title + "中新建任务",
			}).Clicked() {
				w.OpenCreateTaskModal(col.Status)
			}
		})

		// Column Task Cards List
		scrollBox := ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch)
		cardEls := make([]ui.Element, len(tasks))
		scrollBox.Children(func() {
			ui.Column(c).Padding(8, 0, 8, 0).Gap(16).AlignSelf(ui.Stretch).Children(func() {
				for idx, task := range tasks {
					cardEls[idx] = w.renderTaskCard(c.Key(task.ID), tk, task, idx, len(tasks))
				}
			})
		})

		overPayload, isOver := ui.DragOver[DragTaskPayload](scrollBox)
		dropPayload, isDropped := ui.Drop[DragTaskPayload](scrollBox)
		if isOver || isDropped {
			_, py, _ := scrollBox.PointerPosition()
			sb := scrollBox.Bounds()
			insertSlot := boardInsertionSlot(cardEls, sb.Y+py)

			if isDropped {
				w.MoveOrReorderTaskToSlot(dropPayload.TaskID, col.Status, insertSlot)
			} else if isOver {
				showIndicator := true
				if overPayload.FromStatus == col.Status {
					draggedIdx := slices.IndexFunc(tasks, func(t model.Task) bool { return t.ID == overPayload.TaskID })
					if draggedIdx >= 0 && (insertSlot == draggedIdx || insertSlot == draggedIdx+1) {
						showIndicator = false
					}
				} else {
					colBox.Background(tk.Fill)
				}
				if showIndicator {
					accent := tk.Primary
					slotRadius := tk.BorderRadius
					scrollBox.DrawOver(func(p *ui.Painter, r ui.Rect) {
						paintBoardInsertionIndicator(p, accent, slotRadius, r, cardEls, insertSlot)
					})
				}
			}
		}

		// Column Bottom Quick Add Input
		ui.Row(c).Padding(16, 0, 0, 0).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			quickVal := w.quickTaskTitles[col.Status]
			var submitted bool
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				inp := uiInput(c.Key("quick-input-"+string(col.Status)), &quickVal, inputProps{
					Placeholder: "+ 添加" + col.Title + "任务",
					Size:        sizeSmall,
					Variant:     inputElevated,
				}).Label("快速添加" + col.Title + "任务")
				submitted = inp.Submitted()
			})
			w.quickTaskTitles[col.Status] = quickVal

			if submitted && strings.TrimSpace(quickVal) != "" {
				if _, err := w.QuickCreateTask(col.Status, quickVal); err == nil {
					quickVal = w.quickTaskTitles[col.Status]
				}
			}
			if strings.TrimSpace(quickVal) != "" {
				if uiButton(c.Key("quick-btn-"+string(col.Status)), "添加", buttonProps{
					Variant: buttonPrimary,
					Size:    sizeSmall,
				}).Disabled(w.recordWritePending).Label("确认添加" + col.Title + "任务").Clicked() {
					_, _ = w.QuickCreateTask(col.Status, quickVal)
				}
			}
		})
	})
}

func (w *Workbench) renderTaskCard(c *ui.Context, tk *uiTokens, t model.Task, indexInCol, colCount int) ui.Element {
	selected := w.selectedTaskID == t.ID

	card := ui.Column(c).Padding(20).Gap(12).Radius(16).
		Background(tk.BgElevated).AlignSelf(ui.Stretch).Cursor(ui.CursorPointer).
		Label("任务卡片 " + t.Title).
		Drag(DragTaskPayload{TaskID: t.ID, FromStatus: t.Status})

	if selected {
		card.Border(2, tk.Primary)
	}
	if card.Dragging() {
		card.Opacity(0.35)
	}

	// Context menu for right-click / keyboard menu key
	menuChoice := uiContextMenu(c, card, dropdownMenuProps{
		Items: []menuItem{
			{Key: "open", Label: "打开任务详情", Icon: iconSquarePen()},
			{Divider: true},
			{Key: "move:planned", Label: "移至 计划", Icon: iconCircleDashed(), Disabled: t.Status == model.Planned},
			{Key: "move:in_progress", Label: "移至 进行中", Icon: iconTimer(), Disabled: t.Status == model.InProgress},
			{Key: "move:done", Label: "移至 已完成", Icon: iconCircleCheck(), Disabled: t.Status == model.Done},
			{Divider: true},
			{Key: "step:up", Label: "在列内上移", Icon: iconArrowUp(), Disabled: indexInCol <= 0},
			{Key: "step:down", Label: "在列内下移", Icon: iconArrowDown(), Disabled: indexInCol >= colCount-1},
			{Divider: true},
			{Key: "delete", Label: "移入回收站", Icon: iconTrash2(), Danger: true},
		},
	})
	switch menuChoice {
	case "open":
		w.SelectTask(t.ID)
	case "move:planned":
		w.MoveTaskToColumn(t.ID, model.Planned, 0)
	case "move:in_progress":
		w.MoveTaskToColumn(t.ID, model.InProgress, 0)
	case "move:done":
		w.MoveTaskToColumn(t.ID, model.Done, 0)
	case "step:up":
		w.StepTaskInColumn(t.ID, t.Status, -1)
	case "step:down":
		w.StepTaskInColumn(t.ID, t.Status, 1)
	case "delete":
		w.SoftDeleteTask(t.ID)
	}

	actionClicked := false
	card.Children(func() {
		// Title Row
		ui.Row(c).Gap(8).AlignItems(ui.Start).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			titleText := ui.Text(c, t.Title).FontSize(14).FontWeight(600).TextColor(tk.Text).Grow(1)
			if t.Status == model.Done {
				titleText.TextColor(tk.TextSecondary)
			}
		})

		// Priority & Category Tags Row
		ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			prioOpen := ui.Local(card, "prioOpen", func() bool { return false })
			pLabel, pFg, pBg, pHoverBorder := priorityPillSpec(tk, t.Priority)
			prioChoice := uiDropdownMenu(c.Key("priority-menu"), prioOpen, dropdownMenuProps{
				Items: priorityQuickMenuItems(t.Priority),
			}, func(trig ui.Element) {
				trig.Height(24).PaddingX(8).Gap(4).Radius(8).AlignItems(ui.Center).
					Background(pBg).TextColor(pFg).FontSize(13).FontWeight(500).
					Cursor(ui.CursorPointer).Label("切换优先级 " + t.Title)
				if trig.Hovered() || *prioOpen {
					trig.Border(1, pHoverBorder)
				} else {
					trig.Border(1, ui.Transparent)
				}
				trig.Children(func() {
					ui.Text(c, pLabel).SingleLine()
					ui.Icon(c, iconChevronDown()).FontSize(10).TextColor(pFg.Alpha(0.75))
				})
				if trig.Clicked() || trig.Pressed() || *prioOpen {
					actionClicked = true
				}
			})
			if prioChoice != "" {
				actionClicked = true
				w.SetTaskPriority(t.ID, model.Priority(prioChoice))
			}

			for _, tag := range t.Tags {
				uiTag(c.Key("tag-"+tag), "#"+tag, tagProps{
					Size:    sizeSmall,
					Variant: tagOutlined,
				})
			}
		})

		// Metadata Summary Row: DueDate, Subtask Progress, Attachment Count
		hasDue := strings.TrimSpace(t.DueDate) != ""
		hasSubtasks := len(t.Subtasks) > 0
		atts := w.taskAttachments[t.ID]
		hasAttachments := len(atts) > 0

		if hasDue || hasSubtasks || hasAttachments {
			ui.Row(c).Gap(12).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).
				Padding(4, 0, 0, 0).Children(func() {
				if hasDue {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, iconCalendar()).FontSize(12).TextColor(tk.TextTertiary)
						ui.Text(c, strings.TrimSpace(t.DueDate)).FontSize(11).FontFeatures("tnum").TextColor(tk.TextSecondary).SingleLine()
					})
				}
				if hasSubtasks {
					doneCount := 0
					for _, st := range t.Subtasks {
						if st.Done {
							doneCount++
						}
					}
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, iconSquareCheckBig()).FontSize(12).TextColor(tk.TextTertiary)
						ui.Text(c, fmt.Sprintf("%d/%d", doneCount, len(t.Subtasks))).
							FontSize(11).FontFeatures("tnum").TextColor(tk.TextSecondary).SingleLine()
					})
				}
				if hasAttachments {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, iconPaperclip()).FontSize(12).TextColor(tk.TextTertiary)
						ui.Text(c, fmt.Sprint(len(atts))).
							FontSize(11).FontFeatures("tnum").TextColor(tk.TextSecondary).SingleLine()
					})
				}
			})
		}

		// Quick Move & Reorder Bar (minimalist card by default; revealed on card hover or selection)
		if selected || card.Hovered() || card.Dragging() {
			ui.Row(c).Padding(7, 0, 0, 0).Gap(6).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				AlignSelf(ui.Stretch).BorderWidth(1, 0, 0, 0).BorderColor(tk.FillTertiary).Children(func() {
				// Move across columns
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					if t.Status != model.Planned {
						if renderQuickMoveButton(c.Key("to-planned"), tk, t, model.Planned, "计划", "移至计划 "+t.Title, tk.Blue) {
							actionClicked = true
							w.MoveTaskToColumn(t.ID, model.Planned, 0)
						}
					}
					if t.Status != model.InProgress {
						if renderQuickMoveButton(c.Key("to-progress"), tk, t, model.InProgress, "进行中", "移至进行中 "+t.Title, tk.Orange) {
							actionClicked = true
							w.MoveTaskToColumn(t.ID, model.InProgress, 0)
						}
					}
					if t.Status != model.Done {
						if renderQuickMoveButton(c.Key("to-done"), tk, t, model.Done, "已完成", "移至已完成 "+t.Title, tk.Green) {
							actionClicked = true
							w.MoveTaskToColumn(t.ID, model.Done, 0)
						}
					}
				})

				// Reorder up/down within column
				ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
					if indexInCol > 0 {
						if renderStepReorderButton(c.Key("up"), tk, iconChevronUp(), "上移 "+t.Title) {
							actionClicked = true
							w.StepTaskInColumn(t.ID, t.Status, -1)
						}
					}
					if indexInCol < colCount-1 {
						if renderStepReorderButton(c.Key("down"), tk, iconChevronDown(), "下移 "+t.Title) {
							actionClicked = true
							w.StepTaskInColumn(t.ID, t.Status, 1)
						}
					}
				})
			})
		}
	})

	if card.Clicked() && !actionClicked {
		w.SelectTask(t.ID)
	}
	return card
}

func boardInsertionSlot(cardEls []ui.Element, pointerWinY float32) int {
	for i, cardEl := range cardEls {
		b := cardEl.Bounds()
		if b.H <= 0 {
			continue
		}
		if pointerWinY <= b.Y+b.H/2 {
			return i
		}
	}
	return len(cardEls)
}

func paintBoardInsertionIndicator(p *ui.Painter, accent ui.Color, radius float32, r ui.Rect, cardEls []ui.Element, insertSlot int) {
	p.Clip(r, 0, func() {
		x := r.X + 10
		w := max(r.W-20, 40)
		if len(cardEls) == 0 {
			slotRect := ui.Rect{X: x, Y: r.Y + 12, W: w, H: 48}
			p.Fill(slotRect, accent.Alpha(0.08), radius)
			p.StrokeDashed(slotRect, accent.Alpha(0.75), radius, 1.5)
			return
		}
		if b0 := cardEls[0].Bounds(); b0.W > 0 {
			x, w = b0.X, b0.W
		}
		var lineY float32
		switch {
		case insertSlot <= 0:
			b := cardEls[0].Bounds()
			lineY = b.Y - 6
		case insertSlot >= len(cardEls):
			b := cardEls[len(cardEls)-1].Bounds()
			lineY = b.Y + b.H + 6
		default:
			prev := cardEls[insertSlot-1].Bounds()
			next := cardEls[insertSlot].Bounds()
			lineY = (prev.Y + prev.H + next.Y) / 2
		}
		lineY = max(r.Y+5, min(r.Y+r.H-5, lineY))
		p.Fill(ui.Rect{X: x, Y: lineY - 5, W: w, H: 10}, accent.Alpha(0.16), 5)
		p.Fill(ui.Rect{X: x + 4, Y: lineY - 1.5, W: max(w-8, 0), H: 3}, accent, 1.5)
		p.Fill(ui.Rect{X: x, Y: lineY - 4, W: 8, H: 8}, accent, 4)
		p.Fill(ui.Rect{X: x + w - 8, Y: lineY - 4, W: 8, H: 8}, accent, 4)
	})
}

func columnOrder(s model.Status) int {
	switch s {
	case model.Planned:
		return 0
	case model.InProgress:
		return 1
	case model.Done:
		return 2
	default:
		return 0
	}
}

func quickMoveIsLeft(from, to model.Status) bool {
	return columnOrder(to) < columnOrder(from)
}

func renderQuickMoveButton(c *ui.Context, tk *uiTokens, t model.Task, target model.Status, textLabel, a11yLabel string, dot ui.Color) bool {
	btn := ui.ButtonBase(c).Height(22).PaddingX(7).Gap(5).Radius(6).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(a11yLabel)
	fg := tk.TextSecondary
	if btn.Hovered() {
		btn.Background(dot.Alpha(0.14)).Border(1, dot.Alpha(0.48))
		fg = dot
	} else {
		btn.Background(tk.FillQuaternary).Border(1, tk.BorderSecondary)
	}
	isLeft := quickMoveIsLeft(t.Status, target)
	btn.Children(func() {
		if isLeft {
			ui.Icon(c, iconArrowLeft()).FontSize(11).TextColor(fg)
		}
		ui.Box(c).Size(5, 5).Radius(2.5).Background(dot)
		ui.Text(c, textLabel).FontSize(11).TextColor(fg).SingleLine()
		if !isLeft {
			ui.Icon(c, iconArrowRight()).FontSize(11).TextColor(fg)
		}
	})
	return btn.Clicked()
}

func renderStepReorderButton(c *ui.Context, tk *uiTokens, icon *ui.SVG, a11yLabel string) bool {
	btn := ui.ButtonBase(c).Size(22, 22).Radius(6).Center().
		Cursor(ui.CursorPointer).Label(a11yLabel)
	fg := tk.TextTertiary
	if btn.Hovered() {
		btn.Background(tk.FillTertiary).Border(1, tk.PrimaryBorder)
		fg = tk.Text
	} else {
		btn.Background(tk.FillQuaternary).Border(1, tk.BorderSecondary)
	}
	btn.Children(func() { ui.Icon(c, icon).FontSize(12).TextColor(fg) })
	return btn.Clicked()
}

func priorityPillSpec(tk *uiTokens, p model.Priority) (label string, fg, bg, hoverBorder ui.Color) {
	switch p {
	case model.Urgent:
		return "紧急", tk.RedActive, tk.RedFillTertiary, tk.RedActive.Alpha(0.5)
	case model.High:
		return "高", tk.OrangeActive, tk.OrangeFillTertiary, tk.OrangeActive.Alpha(0.5)
	case model.Medium:
		return "中", tk.BlueActive, tk.BlueFillTertiary, tk.BlueActive.Alpha(0.5)
	case model.Low:
		return "低", tk.CyanActive, tk.CyanFillTertiary, tk.CyanActive.Alpha(0.5)
	default:
		return "无优先级", tk.TextSecondary, tk.FillTertiary, tk.Border
	}
}

func priorityQuickMenuItems(current model.Priority) []menuItem {
	extraFor := func(p model.Priority) string {
		if current == p {
			return "✓"
		}
		return ""
	}
	return []menuItem{
		{Key: string(model.Urgent), Label: "紧急", Extra: extraFor(model.Urgent)},
		{Key: string(model.High), Label: "高", Extra: extraFor(model.High)},
		{Key: string(model.Medium), Label: "中", Extra: extraFor(model.Medium)},
		{Key: string(model.Low), Label: "低", Extra: extraFor(model.Low)},
		{Divider: true},
		{Key: string(model.NoPriority), Label: "无优先级", Extra: extraFor(model.NoPriority)},
	}
}

func (w *Workbench) renderTaskCreateModal(c *ui.Context, tk *uiTokens) {
	if uiModal(c, &w.taskModalOpen, modalProps{
		Title:      "新建任务",
		Width:      520,
		OkText:     "创建任务",
		CancelText: "取消",
		OkDisabled: w.recordWritePending,
	}, func() {
		ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			if w.taskModalErr != "" {
				uiAlert(c.Key("task-modal-err"), w.taskModalErr, alertProps{
					Type: alertError,
				})
			}

			// Title
			ui.Column(c.Key("task-modal-title-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "任务标题 *").FontSize(13).FontWeight(500).TextColor(tk.Text)
				before := w.taskModalTitle
				inp := uiInput(c, &w.taskModalTitle, inputProps{
					Placeholder: "输入任务标题（必填）",
					Invalid:     w.taskModalErr != "" && strings.TrimSpace(w.taskModalTitle) == "",
				}).Label("新建任务标题输入框")
				if w.taskModalTitle != before && w.taskModalErr != "" {
					w.taskModalErr = ""
				}
				if inp.Submitted() {
					w.SubmitTaskModal()
				}
			})

			// Status & Priority Row
			ui.Row(c.Key("task-modal-meta-row")).Gap(12).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "所属列 / 状态").FontSize(13).FontWeight(500).TextColor(tk.Text)
					uiSelect(c.Key("modal-task-status"), &w.taskModalStatus, selectProps{
						Options: taskStatusOptions,
					}).Label("新建任务状态")
				})

				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "优先级").FontSize(13).FontWeight(500).TextColor(tk.Text)
					uiSelect(c.Key("modal-task-priority"), &w.taskModalPriority, selectProps{
						Options: taskPriorityOptions,
					}).Label("新建任务优先级")
				})
			})

			// Due Date & Tags Row
			ui.Row(c.Key("task-modal-due-tags-row")).Gap(12).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "截止日期").FontSize(13).FontWeight(500).TextColor(tk.Text)
					uiDatePicker(c.Key("modal-task-duedate"), &w.taskModalDueDate, datePickerProps{
						Placeholder: "选择日期（可选）",
						AllowClear:  true,
					}).Label("新建任务截止日期")
				})

				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "分类标签（逗号分隔）").FontSize(13).FontWeight(500).TextColor(tk.Text)
					uiInput(c, &w.taskModalTags, inputProps{
						Placeholder: "例如：前端, UI, MVP",
					}).Label("新建任务标签输入框")
				})
			})

			// Description
			ui.Column(c.Key("task-modal-desc-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "Markdown 描述").FontSize(13).FontWeight(500).TextColor(tk.Text)
				uiTextArea(c, &w.taskModalDesc, textAreaProps{
					Placeholder: "支持标题、列表、代码块、引用等 Markdown 语法（可选）",
					MinRows:     3,
					MaxRows:     6,
				}).Label("新建任务描述输入框")
			})
		})
	}) {
		w.SubmitTaskModal()
	}
}
