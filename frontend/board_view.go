package frontend

import (
	"fmt"
	"slices"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type boardColumnSpec struct {
	Status model.Status
	Title  string
	Dot    ui.Color
	Tag    display.TagColor
}

var (
	taskStatusOptions = []field.SelectOption{
		{Value: string(model.Planned), Label: "计划"},
		{Value: string(model.InProgress), Label: "进行中"},
		{Value: string(model.Done), Label: "已完成"},
	}
	taskPriorityOptions = []field.SelectOption{
		{Value: string(model.Urgent), Label: "紧急"},
		{Value: string(model.High), Label: "高"},
		{Value: string(model.Medium), Label: "中"},
		{Value: string(model.Low), Label: "低"},
		{Value: string(model.NoPriority), Label: "无优先级"},
	}
	priorityFilterOptions = append([]field.SelectOption{{Value: "", Label: "全部优先级"}}, taskPriorityOptions...)
)

func (w *Workbench) renderBoardView(c *ui.Context, tk *theme.Tokens) {
	proj := w.ActiveProject()
	if proj == nil {
		ui.Column(c).Fill().Center().Padding(32).Children(func() {
			display.Empty(c, display.EmptyProps{
				Icon:        lucide.FolderKanban(),
				Title:       "未选择项目",
				Description: "请从左侧导航选择一个项目，或创建一个新项目开始管理看板任务。",
				Action: func() {
					if action.Button(c, "新建项目", action.ButtonProps{Variant: action.Primary}).Clicked() {
						w.OpenCreateProjectModal()
					}
				},
			})
		})
		return
	}
	p := *proj

	ui.Column(c).Fill().Children(func() {
		// 1. Project Header Bar
		ui.Row(c).Padding(16, 20, 12, 20).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
				ui.Box(c).Size(38, 38).Radius(tk.BorderRadius).Background(tk.FillTertiary).Center().Shrink(0).Children(func() {
					ui.Icon(c, ProjectIconSVG(p.Icon)).FontSize(20).TextColor(tk.Primary)
				})
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, p.Name).FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
						if p.Archived {
							display.Tag(c, "已归档项目", display.TagProps{
								Size:    theme.Small,
								Variant: display.TagFilled,
								Color:   display.TagOrange,
							})
						}
					})
					desc := strings.TrimSpace(p.Description)
					if desc == "" {
						desc = "暂无项目简介"
					}
					ui.Text(c, desc).FontSize(12).TextColor(tk.TextSecondary).SingleLine()
				})
			})

			// Header Action Buttons
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				if action.Button(c, "编辑项目", action.ButtonProps{
					Variant: action.Default,
					Size:    theme.Small,
				}).Clicked() {
					w.OpenEditProjectModal(p)
				}
				if p.Archived {
					if action.Button(c, "恢复归档", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Clicked() {
						w.UnarchiveProject(p.ID)
					}
				} else {
					if action.Button(c, "归档项目", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Clicked() {
						w.ArchiveProject(p.ID)
					}
				}
				if action.Button(c, "移入回收站", action.ButtonProps{
					Variant: action.Default,
					Size:    theme.Small,
					Danger:  true,
				}).Label("将项目移入回收站").Clicked() {
					w.SoftDeleteProject(p.ID)
				}
				if action.Button(c, "+ 新建任务", action.ButtonProps{
					Variant: action.Primary,
					Size:    theme.Middle,
				}).Clicked() {
					w.OpenCreateTaskModal(model.Planned)
				}
			})
		})

		// 2. Search & Filter Bar
		ui.Column(c).Padding(10, 20).Gap(8).AlignSelf(ui.Stretch).Background(tk.BgContainer).
			BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				// Search Input
				ui.Box(c).Width(240).Children(func() {
					beforeQ := w.searchQuery
					field.SearchBar(c, &w.searchQuery, field.SearchBarProps{
						Placeholder: "搜索任务标题、描述或标签...",
						ShortKey:    "mod+k",
						Size:        theme.Small,
					}).Label("搜索任务")
					if w.searchQuery != beforeQ {
						w.applyBoardFilters()
					}
				})

				// Priority Filter Select
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "优先级:").FontSize(12).TextColor(tk.TextSecondary).SingleLine()
					ui.Box(c).Width(120).Children(func() {
						beforeP := w.priorityFilter
						field.Select(c.Key("priority-filter"), &w.priorityFilter, field.SelectProps{
							Placeholder: "全部优先级",
							Size:        theme.Small,
							Options:     priorityFilterOptions,
						}).Label("按优先级筛选")
						if w.priorityFilter != beforeP {
							w.applyBoardFilters()
						}
					})
				})

				// Tag Filter Pills
				if len(w.allBoardTags) > 0 {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Wrap().Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, "标签:").FontSize(12).TextColor(tk.TextSecondary).SingleLine()
						for _, tag := range w.allBoardTags {
							active := slices.Contains(w.tagFilters, tag)
							pill := ui.ButtonBase(c.Key("filter-tag-" + tag)).Height(22).PaddingX(8).Radius(11).
								FontSize(12).AlignItems(ui.Center).Cursor(ui.CursorPointer)
							if active {
								pill.Background(tk.Primary).TextColor(tk.BgLayout).FontWeight(600)
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
					})
				} else {
					ui.Box(c).Grow(1)
				}

				// Clear Filters & Task Count Summary
				ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
					if w.HasBoardFilters() {
						ui.Text(c, fmt.Sprintf("匹配 %d / %d 项", len(w.filteredTasks), len(w.boardTasks))).
							FontSize(12).TextColor(tk.Primary).SingleLine()
						if action.Button(c, "清除筛选", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Clicked() {
							w.ClearBoardFilters()
						}
					} else {
						ui.Text(c, fmt.Sprintf("共 %d 项任务", len(w.boardTasks))).
							FontSize(12).TextColor(tk.TextTertiary).SingleLine()
					}
				})
			})
		})

		// 3. Three-Column Kanban Area
		columns := []boardColumnSpec{
			{Status: model.Planned, Title: "计划", Dot: tk.Blue, Tag: display.TagBlue},
			{Status: model.InProgress, Title: "进行中", Dot: tk.Orange, Tag: display.TagOrange},
			{Status: model.Done, Title: "已完成", Dot: tk.Green, Tag: display.TagGreen},
		}

		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Padding(16, 20).Gap(16).AlignItems(ui.Stretch).
			AlignSelf(ui.Stretch).Children(func() {
			for _, col := range columns {
				w.renderBoardColumn(c.Key(string(col.Status)), tk, col)
			}
		})
	})
}

func (w *Workbench) renderBoardColumn(c *ui.Context, tk *theme.Tokens, col boardColumnSpec) {
	tasks := w.TasksInColumn(col.Status)

	colBox := ui.Column(c).Grow(1).Basis(0).MinWidth(180).AlignSelf(ui.Stretch).
		Radius(tk.BorderRadiusLG).Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
		Label("看板列 " + col.Title)

	// Header/outer column drop fallback from another column; keep border width fixed at 1px to avoid layout reflow during drag.
	if payload, over := ui.DragOver[DragTaskPayload](colBox); over && payload.FromStatus != col.Status {
		colBox.Border(1, col.Dot.Alpha(0.65))
	}
	if payload, dropped := ui.Drop[DragTaskPayload](colBox); dropped && payload.FromStatus != col.Status {
		w.MoveOrReorderTaskToSlot(payload.TaskID, col.Status, 0)
	}

	colBox.Children(func() {
		// Column Header
		ui.Row(c).Padding(12, 14, 10, 14).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(8, 8).Radius(4).Background(col.Dot)
				ui.Text(c, col.Title).FontSize(14).FontWeight(600).TextColor(tk.Text).SingleLine()
				display.Tag(c, fmt.Sprint(len(tasks)), display.TagProps{
					Size:    theme.Small,
					Variant: display.TagFilled,
					Color:   col.Tag,
					Round:   true,
				})
			})

			if action.ActionIcon(c, lucide.Plus(), action.ActionIconProps{
				Size:  theme.Small,
				Title: "在" + col.Title + "中新建任务",
			}).Clicked() {
				w.OpenCreateTaskModal(col.Status)
			}
		})

		// Column Task Cards List (single drop target per column so gap/card transitions never flicker or reflow)
		scrollBox := ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch)
		cardEls := make([]ui.Element, len(tasks))
		scrollBox.Children(func() {
			ui.Column(c).Padding(12, 10).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				if len(tasks) == 0 {
					emptyHint := "暂无" + col.Title + "任务"
					if w.HasBoardFilters() {
						emptyHint = "无匹配的" + col.Title + "任务"
					}
					ui.Column(c).Padding(28, 12).Gap(6).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, emptyHint).FontSize(12).TextColor(tk.TextTertiary)
						ui.Text(c, "可拖拽任务至此列，或在下方快速添加").FontSize(11).TextColor(tk.TextQuaternary)
					})
				} else {
					for idx, task := range tasks {
						cardEls[idx] = w.renderTaskCard(c.Key(task.ID), tk, task, idx, len(tasks))
					}
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
					colBox.Border(1, col.Dot.Alpha(0.65))
				}
				if showIndicator {
					accent := col.Dot
					slotRadius := tk.BorderRadius
					scrollBox.DrawOver(func(p *ui.Painter, r ui.Rect) {
						paintBoardInsertionIndicator(p, accent, slotRadius, r, cardEls, insertSlot)
					})
				}
			}
		}

		// Column Bottom Quick Add Input
		ui.Row(c).Padding(8, 10).Gap(6).AlignItems(ui.Center).AlignSelf(ui.Stretch).
			BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
			quickVal := w.quickTaskTitles[col.Status]
			var submitted bool
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				inp := field.Input(c.Key("quick-input-"+string(col.Status)), &quickVal, field.InputProps{
					Placeholder: "+ 快速添加" + col.Title + "任务...",
					Size:        theme.Small,
					Variant:     field.InputFilled,
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
				if action.Button(c.Key("quick-btn-"+string(col.Status)), "添加", action.ButtonProps{
					Variant: action.Primary,
					Size:    theme.Small,
				}).Label("确认添加" + col.Title + "任务").Clicked() {
					_, _ = w.QuickCreateTask(col.Status, quickVal)
				}
			}
		})
	})
}

func (w *Workbench) renderTaskCard(c *ui.Context, tk *theme.Tokens, t model.Task, indexInCol, colCount int) ui.Element {
	selected := w.selectedTaskID == t.ID

	card := ui.Column(c).Padding(12).Gap(8).Radius(tk.BorderRadius).
		Background(tk.BgContainer).AlignSelf(ui.Stretch).Cursor(ui.CursorPointer).
		Label("任务卡片 " + t.Title).
		Drag(DragTaskPayload{TaskID: t.ID, FromStatus: t.Status})

	if selected {
		card.Border(2, tk.Primary)
	} else if card.Hovered() {
		card.Border(1, tk.PrimaryHover)
	} else {
		card.Border(1, tk.BorderSecondary)
	}
	if card.Dragging() {
		card.Opacity(0.35)
	}

	// Context menu for right-click / keyboard menu key
	menuChoice := overlay.ContextMenu(c, card, overlay.DropdownMenuProps{
		Items: []overlay.MenuItem{
			{Key: "open", Label: "打开任务详情", Icon: lucide.SquarePen()},
			{Divider: true},
			{Key: "move:planned", Label: "移至 计划", Icon: lucide.CircleDashed(), Disabled: t.Status == model.Planned},
			{Key: "move:in_progress", Label: "移至 进行中", Icon: lucide.Timer(), Disabled: t.Status == model.InProgress},
			{Key: "move:done", Label: "移至 已完成", Icon: lucide.CircleCheck(), Disabled: t.Status == model.Done},
			{Divider: true},
			{Key: "step:up", Label: "在列内上移", Icon: lucide.ArrowUp(), Disabled: indexInCol <= 0},
			{Key: "step:down", Label: "在列内下移", Icon: lucide.ArrowDown(), Disabled: indexInCol >= colCount-1},
			{Divider: true},
			{Key: "delete", Label: "移入回收站", Icon: lucide.Trash2(), Danger: true},
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
			pLabel, pColor := priorityBadgeSpec(t.Priority)
			display.Tag(c, pLabel, display.TagProps{
				Size:    theme.Small,
				Variant: display.TagFilled,
				Color:   pColor,
			})

			for _, tag := range t.Tags {
				display.Tag(c.Key("tag-"+tag), "#"+tag, display.TagProps{
					Size:    theme.Small,
					Variant: display.TagOutlined,
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
						ui.Icon(c, lucide.Calendar()).FontSize(12).TextColor(tk.TextTertiary)
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
						ui.Icon(c, lucide.SquareCheckBig()).FontSize(12).TextColor(tk.TextTertiary)
						ui.Text(c, fmt.Sprintf("%d/%d", doneCount, len(t.Subtasks))).
							FontSize(11).FontFeatures("tnum").TextColor(tk.TextSecondary).SingleLine()
					})
				}
				if hasAttachments {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, lucide.Paperclip()).FontSize(12).TextColor(tk.TextTertiary)
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
						if renderStepReorderButton(c.Key("up"), tk, lucide.ChevronUp(), "上移 "+t.Title) {
							actionClicked = true
							w.StepTaskInColumn(t.ID, t.Status, -1)
						}
					}
					if indexInCol < colCount-1 {
						if renderStepReorderButton(c.Key("down"), tk, lucide.ChevronDown(), "下移 "+t.Title) {
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

func renderQuickMoveButton(c *ui.Context, tk *theme.Tokens, t model.Task, target model.Status, textLabel, a11yLabel string, dot ui.Color) bool {
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
			ui.Icon(c, lucide.ArrowLeft()).FontSize(11).TextColor(fg)
		}
		ui.Box(c).Size(5, 5).Radius(2.5).Background(dot)
		ui.Text(c, textLabel).FontSize(11).TextColor(fg).SingleLine()
		if !isLeft {
			ui.Icon(c, lucide.ArrowRight()).FontSize(11).TextColor(fg)
		}
	})
	return btn.Clicked()
}

func renderStepReorderButton(c *ui.Context, tk *theme.Tokens, icon *ui.SVG, a11yLabel string) bool {
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

func priorityBadgeSpec(p model.Priority) (string, display.TagColor) {
	switch p {
	case model.Urgent:
		return "紧急", display.TagRed
	case model.High:
		return "高", display.TagOrange
	case model.Medium:
		return "中", display.TagBlue
	case model.Low:
		return "低", display.TagCyan
	default:
		return "无优先级", display.TagGray
	}
}

func (w *Workbench) renderTaskCreateModal(c *ui.Context, tk *theme.Tokens) {
	if feedback.Modal(c, &w.taskModalOpen, feedback.ModalProps{
		Title:      "新建任务",
		Width:      520,
		OkText:     "创建任务",
		CancelText: "取消",
	}, func() {
		ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			if w.taskModalErr != "" {
				feedback.Alert(c.Key("task-modal-err"), w.taskModalErr, feedback.AlertProps{
					Type:    feedback.Error,
					Variant: feedback.Soft,
				})
			}

			// Title
			ui.Column(c.Key("task-modal-title-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "任务标题 *").FontSize(13).FontWeight(500).TextColor(tk.Text)
				before := w.taskModalTitle
				inp := field.Input(c, &w.taskModalTitle, field.InputProps{
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
					field.Select(c.Key("modal-task-status"), &w.taskModalStatus, field.SelectProps{
						Options: taskStatusOptions,
					}).Label("新建任务状态")
				})

				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "优先级").FontSize(13).FontWeight(500).TextColor(tk.Text)
					field.Select(c.Key("modal-task-priority"), &w.taskModalPriority, field.SelectProps{
						Options: taskPriorityOptions,
					}).Label("新建任务优先级")
				})
			})

			// Due Date & Tags Row
			ui.Row(c.Key("task-modal-due-tags-row")).Gap(12).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "截止日期").FontSize(13).FontWeight(500).TextColor(tk.Text)
					field.DatePicker(c.Key("modal-task-duedate"), &w.taskModalDueDate, field.DatePickerProps{
						Placeholder: "选择日期（可选）",
						AllowClear:  true,
					}).Label("新建任务截止日期")
				})

				ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
					ui.Text(c, "分类标签（逗号分隔）").FontSize(13).FontWeight(500).TextColor(tk.Text)
					field.Input(c, &w.taskModalTags, field.InputProps{
						Placeholder: "例如：前端, UI, MVP",
					}).Label("新建任务标签输入框")
				})
			})

			// Description
			ui.Column(c.Key("task-modal-desc-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "Markdown 描述").FontSize(13).FontWeight(500).TextColor(tk.Text)
				field.TextArea(c, &w.taskModalDesc, field.TextAreaProps{
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
