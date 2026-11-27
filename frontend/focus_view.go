package frontend

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
)

// ReloadFocusTasks reloads active cross-project tasks used by the Focus overview.
func (w *Workbench) ReloadFocusTasks() { w.reloadFocusTasks(false) }

func (w *Workbench) reloadFocusTasks(async bool) {
	service := w.app.Projects
	w.loadUI("focus", async, func() func() {
		tasks, err := service.SearchTasks(context.Background(), projects.SearchTasksFilter{
			IncludeArchived: false,
		})
		return func() {
			if err != nil {
				return
			}
			w.focusTasks = tasks
		}
	})
}

type focusTaskSummary struct {
	Items         []model.Task
	OverdueCount  int
	DueSoonCount  int
	InProgressCnt int
}

func (w *Workbench) computeFocusTasks() focusTaskSummary {
	today := w.now().Local().Format("2006-01-02")
	soonCutoff := w.now().Local().AddDate(0, 0, 7).Format("2006-01-02")

	var out focusTaskSummary
	for _, t := range w.focusTasks {
		if t.Status == model.Done {
			continue
		}
		isOverdue := t.DueDate != "" && t.DueDate < today
		isDueSoon := t.DueDate != "" && t.DueDate >= today && t.DueDate <= soonCutoff
		isInProgress := t.Status == model.InProgress
		isHighPriority := t.Priority == model.Urgent || t.Priority == model.High

		if isOverdue {
			out.OverdueCount++
		} else if isDueSoon {
			out.DueSoonCount++
		}
		if isInProgress {
			out.InProgressCnt++
		}
		if isOverdue || isDueSoon || isInProgress || isHighPriority {
			out.Items = append(out.Items, t)
		}
	}

	taskRank := func(t model.Task) int {
		if t.DueDate != "" && t.DueDate < today {
			return 0
		}
		if t.DueDate != "" && t.DueDate <= soonCutoff {
			return 1
		}
		if t.Status == model.InProgress {
			return 2
		}
		if t.Priority == model.Urgent {
			return 3
		}
		return 4
	}
	priorityRank := func(p model.Priority) int {
		switch p {
		case model.Urgent:
			return 0
		case model.High:
			return 1
		case model.Medium:
			return 2
		case model.Low:
			return 3
		default:
			return 4
		}
	}
	slices.SortFunc(out.Items, func(a, b model.Task) int {
		if c := cmp.Compare(taskRank(a), taskRank(b)); c != 0 {
			return c
		}
		if a.DueDate != "" && b.DueDate != "" {
			if c := cmp.Compare(a.DueDate, b.DueDate); c != 0 {
				return c
			}
		} else if a.DueDate != "" {
			return -1
		} else if b.DueDate != "" {
			return 1
		}
		if c := cmp.Compare(priorityRank(a.Priority), priorityRank(b.Priority)); c != 0 {
			return c
		}
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return out
}

func (w *Workbench) focusActiveJobs() []model.JobApplication {
	active := jobapplications.FilterApplications(w.jobApps, jobapplications.Filter{
		View: jobapplications.ViewActive,
	}, jobapplications.SortUpdatedDesc)
	if len(active) > 10 {
		return active[:10]
	}
	return active
}

func (w *Workbench) focusBadgeCount() int {
	total := 0
	if w.IsModuleVisible("projects") {
		ft := w.computeFocusTasks()
		total += ft.OverdueCount + ft.InProgressCnt
	}
	if w.IsModuleVisible("job_applications") {
		total += w.jobStats.Active
	}
	return total
}

// OpenFocusTask jumps from the Focus overview directly into the task's project board and detail modal.
func (w *Workbench) OpenFocusTask(task model.Task) {
	if w.IsRestoring() || task.ID == "" || task.ProjectID == "" {
		return
	}
	if !w.IsModuleVisible("projects") {
		return
	}
	w.SelectProject(task.ProjectID)
	w.SelectTask(task.ID)
}

// OpenFocusJob jumps from the Focus overview or command palette into the job application's matching list view and detail modal.
func (w *Workbench) OpenFocusJob(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	if !w.IsModuleVisible("job_applications") {
		return
	}
	targetPage := NavJobActive
	if idx := slices.IndexFunc(w.jobApps, func(a model.JobApplication) bool { return a.ID == appID }); idx >= 0 {
		if w.jobApps[idx].Lifecycle == model.JobLifecycleClosed {
			targetPage = NavJobClosed
		}
	}
	w.NavigateTo(targetPage)
	w.SelectJobApplication(appID)
}

// QuickSetFocusTaskStatus moves a task directly from the Focus overview without blocking on Store locks.
func (w *Workbench) QuickSetFocusTaskStatus(task model.Task, targetStatus model.Status) {
	if w.IsRestoring() || task.ID == "" || !targetStatus.Valid() || task.Status == targetStatus {
		return
	}
	service := w.app.Projects
	_, _ = runRecordWrite(w, func() (model.Task, error) {
		return service.MoveTask(context.Background(), task.ID, targetStatus, 0)
	}, func(moved model.Task, err error) {
		if err != nil {
			return
		}
		w.reloadFocusTasks(true)
		w.reloadProjects(true)
		if w.activeProjectID == moved.ProjectID {
			w.reloadBoardTasks(true)
		}
	})
}

func nextJobStage(cur model.JobStage) (model.JobStage, bool) {
	stages := model.AllJobStages()
	for i, st := range stages {
		if st == cur && i+1 < len(stages) {
			return stages[i+1], true
		}
	}
	return "", false
}

func (w *Workbench) renderFocusView(c *ui.Context, tk *uiTokens) {
	showProjects := w.IsModuleVisible("projects")
	showJobs := w.IsModuleVisible("job_applications")
	today := w.now().Local()
	todayStr := today.Format("2006-01-02")
	soonCutoff := today.AddDate(0, 0, 7).Format("2006-01-02")

	focusTasks := w.computeFocusTasks()
	activeJobs := w.focusActiveJobs()

	// Compute completion metrics across active projects
	totalTasks := len(w.focusTasks)
	doneTasks := 0
	var unfinishedTasks []model.Task
	var doneList []model.Task
	for _, t := range w.focusTasks {
		if t.Status == model.Done {
			doneTasks++
			doneList = append(doneList, t)
		} else {
			unfinishedTasks = append(unfinishedTasks, t)
		}
	}
	pct := 0
	if totalTasks > 0 {
		pct = doneTasks * 100 / totalTasks
	}

	// Build filtered task list
	var displayTasks []model.Task
	switch w.focusFilter {
	case "in_progress":
		for _, t := range focusTasks.Items {
			if t.Status == model.InProgress {
				displayTasks = append(displayTasks, t)
			}
		}
	case "all":
		displayTasks = unfinishedTasks
	case "done":
		displayTasks = doneList
	default:
		displayTasks = focusTasks.Items
	}

	ui.Scroll(c.Key("focus-scroll")).Fill().Background(tk.BgLayout).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Row(c).Padding(40, 48, 56, 48).Gap(64).WidthPercent(100).MaxWidth(1200).AlignItems(ui.Start).Children(func() {
				// Left Main Column: Today Task Stream
				if showProjects {
					ui.Column(c.Key("focus-tasks-card")).Grow(1).Basis(0).MinWidth(0).Gap(20).Children(func() {
						// Section Header: Title + Completion Metric (aligned with row padding)
						ui.Row(c).PaddingX(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
							ui.Text(c, "今天").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
							ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
								ui.Text(c, fmt.Sprintf("%d / %d", doneTasks, totalTasks)).
									Font("monospace").FontSize(14).TextColor(tk.TextSecondary).SingleLine()
								ui.Text(c, fmt.Sprintf("%d%%", pct)).
									Font("monospace").FontSize(16).FontWeight(600).TextColor(tk.Primary).SingleLine()
							})
						})

						// Minimal Typographic Filter Row
						ui.Row(c.Key("focus-filters")).PaddingX(16).Gap(24).AlignItems(ui.Center).Children(func() {
							for _, f := range []struct {
								key, label string
								count      int
							}{
								{"", "待推进", len(focusTasks.Items)},
								{"in_progress", "进行中", focusTasks.InProgressCnt},
								{"all", "全部未完成", len(unfinishedTasks)},
								{"done", "已完成", doneTasks},
							} {
								active := w.focusFilter == f.key
								btn := ui.ButtonBase(c.Key("focus-f-"+f.label)).Padding(4, 0).Gap(6).
									AlignItems(ui.Center).Cursor(ui.CursorPointer).Label("聚焦筛选 " + f.label)
								if active {
									btn.TextColor(tk.Text).FontWeight(600)
								} else {
									btn.TextColor(tk.TextSecondary).FontWeight(500)
								}
								btn.Children(func() {
									ui.Text(c, f.label).FontSize(14).SingleLine()
									ui.Text(c, fmt.Sprint(f.count)).Font("monospace").FontSize(14).TextColor(tk.TextSecondary).SingleLine()
								})
								if btn.Clicked() {
									w.focusFilter = f.key
								}
							}
						})

						// Borderless Task Stream
						if len(displayTasks) == 0 {
							ui.Column(c).Padding(48, 0).AlignItems(ui.Start).Children(func() {
								ui.Text(c, "暂无待推进任务").FontSize(16).TextColor(tk.TextSecondary)
							})
						} else {
							limit := min(len(displayTasks), 16)
							ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
								for i := 0; i < limit; i++ {
									w.renderFocusTaskRow(c, tk, displayTasks[i], todayStr, soonCutoff)
								}
							})
						}
					})
				}

				// Right Open Column: Active Jobs + Local Sync (Zero Boxed Cards)
				if showJobs || !showProjects {
					rightCol := ui.Column(c).Gap(48)
					if showProjects {
						rightCol.Width(320).Shrink(0)
					} else {
						rightCol.Grow(1).Basis(0).MinWidth(0)
					}
					rightCol.Children(func() {
						if showJobs {
							jobSec := ui.Column(c.Key("focus-jobs-card")).Gap(20).AlignSelf(ui.Stretch)
							jobSec.Children(func() {
								ui.Row(c).PaddingX(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, "求职").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
									allBtn := ui.ButtonBase(c.Key("focus-jobs-all")).Gap(4).AlignItems(ui.Center).
										Cursor(ui.CursorPointer).TextColor(tk.Primary).Label("查看全部投递")
									allBtn.Children(func() {
										ui.Text(c, "全部").FontSize(14).FontWeight(500).SingleLine()
										uiIconBox(c, iconChevronRight(), tk.Primary)
									})
									if allBtn.Clicked() {
										w.NavigateTo(NavJobActive)
									}
								})

								if len(activeJobs) == 0 {
									ui.Text(c, "暂无进行中的投递").FontSize(14).TextColor(tk.TextSecondary)
								} else {
									limit := min(len(activeJobs), 6)
									ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
										for i := 0; i < limit; i++ {
											w.renderFocusJobRow(c, tk, activeJobs[i])
										}
									})
								}
							})
						}

						// Local Sync Status Section
						ui.Column(c.Key("focus-sync-sec")).PaddingX(16).Gap(16).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, "同步").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
								setBtn := ui.ButtonBase(c.Key("focus-sync-settings")).Gap(4).AlignItems(ui.Center).
									Cursor(ui.CursorPointer).TextColor(tk.Primary)
								setBtn.Children(func() {
									ui.Text(c, "设置").FontSize(14).FontWeight(500).SingleLine()
									uiIconBox(c, iconChevronRight(), tk.Primary)
								})
								if setBtn.Clicked() {
									w.NavigateTo(NavSettings)
								}
							})
							if w.MCPCapability().Ready {
								ui.Text(c, "本地 MCP 已就绪 · aion.sock").FontSize(14).TextColor(tk.TextSecondary)
							} else {
								ui.Text(c, "离线工作模式").FontSize(14).TextColor(tk.TextSecondary)
							}
						})
					})
				}
			})
		})
	})
}

func (w *Workbench) renderFocusTaskRow(c *ui.Context, tk *uiTokens, task model.Task, todayStr, soonCutoff string) {
	projName := w.projectNames[task.ProjectID]
	if projName == "" {
		projName = "未命名项目"
	}
	isDone := task.Status == model.Done

	wrap := ui.Row(c.Key("focus-task-wrap-"+task.ID)).Padding(12, 16).Gap(16).Radius(16).
		AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch)
	if wrap.Hovered() {
		wrap.Background(tk.FillTertiary)
	}
	if isDone {
		wrap.Opacity(0.5)
	}

	wrap.Children(func() {
		// Circular Check Button
		chk := ui.ButtonBase(c.Key("focus-done-"+task.ID)).Size(24, 24).Radius(40).
			Center().Shrink(0).Cursor(ui.CursorPointer).Disabled(w.recordWritePending).
			Label("完成任务 " + task.Title)
		if isDone {
			chk.Background(tk.Primary).Border(1.5, tk.Primary)
		} else if chk.Hovered() {
			chk.Border(1.5, tk.Primary)
		} else {
			chk.Border(1.5, tk.Border)
		}
		chk.Children(func() {
			if isDone {
				ui.Icon(c, checkGlyph).FontSize(12).TextColor(ui.RGB(255, 255, 255))
			}
		})
		if chk.Clicked() {
			if isDone {
				w.QuickSetFocusTaskStatus(task, model.InProgress)
			} else {
				w.QuickSetFocusTaskStatus(task, model.Done)
			}
		}

		// Task Title + Project Name
		row := ui.ButtonBase(c.Key("focus-task-" + task.ID)).Grow(1).MinWidth(0).
			AlignItems(ui.Start).Justify(ui.Start).Cursor(ui.CursorPointer).
			Label("打开聚焦任务 " + task.Title)
		row.Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
				titleColor := tk.Text
				if isDone {
					titleColor = tk.TextSecondary
				}
				ui.Text(c, task.Title).FontSize(16).FontWeight(500).TextColor(titleColor).SingleLine()
				ui.Text(c, projName).FontSize(14).TextColor(tk.TextSecondary).SingleLine()
			})
		})
		if row.Clicked() {
			w.OpenFocusTask(task)
		}

		// Right: Due Date + Intent-Revealed Actions
		ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
			if task.DueDate != "" {
				dueColor := tk.TextSecondary
				if !isDone && (task.DueDate <= todayStr || task.DueDate <= soonCutoff) {
					dueColor = tk.Primary
				}
				ui.Text(c, task.DueDate).Font("monospace").FontSize(14).TextColor(dueColor).SingleLine()
			}
			if task.Status == model.Planned {
				startBtn := uiActionIcon(c.Key("focus-start-"+task.ID), iconArrowRight(), actionIconProps{
					Title: "开始任务 " + task.Title,
				}).Disabled(w.recordWritePending)
				if !wrap.Hovered() {
					startBtn.Opacity(0)
				}
				if startBtn.Clicked() {
					w.QuickSetFocusTaskStatus(task, model.InProgress)
				}
			}
		})
	})
}

func (w *Workbench) renderFocusJobRow(c *ui.Context, tk *uiTokens, app model.JobApplication) {
	title := strings.TrimSpace(app.CompanyName + " · " + app.PositionName)
	wrap := ui.Row(c.Key("focus-job-wrap-"+app.ID)).Padding(12, 16).Gap(12).Radius(16).
		AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch)
	if wrap.Hovered() {
		wrap.Background(tk.FillTertiary)
	}

	wrap.Children(func() {
		row := ui.ButtonBase(c.Key("focus-job-" + app.ID)).Grow(1).MinWidth(0).
			AlignItems(ui.Center).Justify(ui.SpaceBetween).Cursor(ui.CursorPointer).
			Label("打开聚焦投递 " + title)
		row.Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).Gap(12).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
					w.renderJobAvatar(c, tk, app, 28)
					ui.Text(c, app.CompanyName).FontSize(16).FontWeight(500).TextColor(tk.Text).SingleLine()
				})
				stColor := tk.TextSecondary
				if app.Stage == model.JobStageOffer {
					stColor = tk.Primary
				}
				ui.Text(c, app.Stage.Label()).FontSize(14).FontWeight(500).TextColor(stColor).SingleLine().Shrink(0)
			})
		})
		if row.Clicked() {
			w.OpenFocusJob(app.ID)
		}

		if nextStage, ok := nextJobStage(app.Stage); ok {
			nextBtn := uiActionIcon(c.Key("focus-job-next-"+app.ID), iconArrowRight(), actionIconProps{
				Title: "推进阶段至 " + nextStage.Label() + " " + app.CompanyName,
			})
			if !wrap.Hovered() {
				nextBtn.Opacity(0)
			}
			if nextBtn.Clicked() {
				w.ChangeJobStage(app.ID, nextStage)
			}
		} else {
			ui.Box(c).Size(28, 28).Shrink(0)
		}
	})
}

func priorityLabel(p model.Priority) string {
	switch p {
	case model.Urgent:
		return "紧急"
	case model.High:
		return "高"
	case model.Medium:
		return "中"
	case model.Low:
		return "低"
	default:
		return "无"
	}
}

func priorityTagColor(p model.Priority) tagColor {
	switch p {
	case model.Urgent:
		return tagError
	case model.High:
		return tagOrange
	case model.Medium:
		return tagProcessing
	default:
		return tagGray
	}
}
