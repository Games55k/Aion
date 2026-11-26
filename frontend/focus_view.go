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

// OpenFocusJob jumps from the Focus overview directly into the job application's detail modal.
func (w *Workbench) OpenFocusJob(appID string) {
	if w.IsRestoring() || appID == "" {
		return
	}
	if !w.IsModuleVisible("job_applications") {
		return
	}
	w.NavigateTo(NavJobActive)
	w.SelectJobApplication(appID)
}

func (w *Workbench) renderFocusView(c *ui.Context, tk *uiTokens) {
	showProjects := w.IsModuleVisible("projects")
	showJobs := w.IsModuleVisible("job_applications")
	today := w.now().Local()
	todayStr := today.Format("2006-01-02")
	soonCutoff := today.AddDate(0, 0, 7).Format("2006-01-02")

	focusTasks := w.computeFocusTasks()
	activeJobs := w.focusActiveJobs()

	ui.Column(c).Fill().Children(func() {
		// Top Header Bar
		ui.Row(c).MinHeight(66).Padding(16, 24).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconTarget()).FontSize(22).TextColor(tk.Primary)
				ui.Text(c, "聚焦总览").FontSize(20).FontWeight(700).TextColor(tk.Text).SingleLine()
				uiTag(c, today.Format("2006 年 01 月 02 日"), tagProps{
					Size:    sizeSmall,
					Variant: tagFilled,
					Color:   tagGray,
				})
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				if showProjects {
					if uiButton(c, "所有项目", buttonProps{
						Variant: buttonDefault,
						Size:    sizeSmall,
					}).Clicked() {
						w.NavigateTo(NavAllProjects)
					}
				}
				if showJobs {
					if uiButton(c, "+ 新增投递", buttonProps{
						Variant: buttonPrimary,
						Size:    sizeSmall,
					}).Clicked() {
						w.OpenJobCreateModal()
					}
				}
			})
		})

		// Body
		ui.Scroll(c.Key("focus-scroll")).Grow(1).Basis(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Padding(24).Gap(20).MaxWidth(1040).AlignSelf(ui.Stretch).Children(func() {
				// KPI Summary Row (adapts automatically to visible modules)
				ui.Row(c.Key("focus-kpi-row")).Gap(14).AlignSelf(ui.Stretch).Children(func() {
					if showProjects {
						renderFocusKPI(c, tk, "进行中任务", fmt.Sprint(focusTasks.InProgressCnt), iconClock(), tk.Primary, func() {
							w.NavigateTo(NavAllProjects)
						})
						dueColor := tk.Text
						if focusTasks.OverdueCount > 0 {
							dueColor = tk.Error
						} else if focusTasks.DueSoonCount > 0 {
							dueColor = tk.Warning
						}
						renderFocusKPI(c, tk, "逾期 / 7天内临期", fmt.Sprint(focusTasks.OverdueCount+focusTasks.DueSoonCount), iconCalendar(), dueColor, func() {
							w.NavigateTo(NavAllProjects)
						})
					}
					if showJobs {
						renderFocusKPI(c, tk, "进行中投递", fmt.Sprint(w.jobStats.Active), iconBriefcase(), tk.Primary, func() {
							w.NavigateTo(NavJobActive)
						})
						renderFocusKPI(c, tk, "已获 Offer", fmt.Sprint(w.jobStats.Offered), iconAward(), tk.Success, func() {
							w.NavigateTo(NavJobOffer)
						})
					}
				})

				// Section 1: Priority & Active Tasks
				if showProjects {
					ui.Column(c.Key("focus-tasks-card")).Padding(20).Gap(12).Radius(tk.BorderRadiusLG).
						Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconFolderKanban()).FontSize(17).TextColor(tk.Primary)
								ui.Text(c, "待推进任务").FontSize(16).FontWeight(600).TextColor(tk.Text)
								if len(focusTasks.Items) > 0 {
									uiTag(c, fmt.Sprintf("%d 项", len(focusTasks.Items)), tagProps{
										Size:    sizeSmall,
										Variant: tagFilled,
										Color:   tagGray,
									})
								}
							})
							ui.Text(c, "汇总活跃项目中已逾期、7 天内到期、进行中及高优先级任务").
								FontSize(12).TextColor(tk.TextTertiary)
						})

						if len(focusTasks.Items) == 0 {
							ui.Column(c).Padding(24, 16).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, "当前没有逾期、进行中或高优先级任务").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
								ui.Text(c, "可以在项目看板中拆分新任务，或直接让本地 AI 通过 MCP 帮你创建。").
									FontSize(12).TextColor(tk.TextTertiary)
							})
						} else {
							limit := min(len(focusTasks.Items), 12)
							ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
								for i := 0; i < limit; i++ {
									task := focusTasks.Items[i]
									w.renderFocusTaskRow(c, tk, task, todayStr, soonCutoff)
								}
							})
						}
					})
				}

				// Section 2: Active Job Applications
				if showJobs {
					ui.Column(c.Key("focus-jobs-card")).Padding(20).Gap(12).Radius(tk.BorderRadiusLG).
						Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconBriefcase()).FontSize(17).TextColor(tk.Primary)
								ui.Text(c, "进行中的求职投递").FontSize(16).FontWeight(600).TextColor(tk.Text)
								if w.jobStats.Active > 0 {
									uiTag(c, fmt.Sprintf("%d 条活跃", w.jobStats.Active), tagProps{
										Size:    sizeSmall,
										Variant: tagFilled,
										Color:   tagProcessing,
									})
								}
							})
							if uiButton(c, "查看全部投递", buttonProps{
								Variant: buttonDefault,
								Size:    sizeSmall,
							}).Clicked() {
								w.NavigateTo(NavJobActive)
							}
						})

						if len(activeJobs) == 0 {
							ui.Column(c).Padding(24, 16).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, "暂无进行中的求职投递").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
								ui.Text(c, "如果当前阶段不求职，可在「设置 → 工作区模块」中暂时隐藏求职模块（数据保留不删）。").
									FontSize(12).TextColor(tk.TextTertiary)
							})
						} else {
							ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
								for _, app := range activeJobs {
									w.renderFocusJobRow(c, tk, app)
								}
							})
						}
					})
				}
			})
		})
	})
}

func renderFocusKPI(c *ui.Context, tk *uiTokens, title, value string, icon *ui.SVG, accent ui.Color, onClick func()) {
	btn := ui.ButtonBase(c.Key("kpi-"+title)).Grow(1).Basis(0).Padding(16).Gap(8).Radius(tk.BorderRadiusLG).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignItems(ui.Start).Justify(ui.Start).Cursor(ui.CursorPointer).
		Label(title + " " + value)
	btn.Children(func() {
		ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icon).FontSize(15).TextColor(tk.TextSecondary)
				ui.Text(c, title).FontSize(13).TextColor(tk.TextSecondary).SingleLine()
			})
			ui.Text(c, value).FontSize(26).FontWeight(700).TextColor(accent).SingleLine()
		})
	})
	if btn.Clicked() && onClick != nil {
		onClick()
	}
}

func (w *Workbench) renderFocusTaskRow(c *ui.Context, tk *uiTokens, task model.Task, todayStr, soonCutoff string) {
	projName := w.projectNames[task.ProjectID]
	if projName == "" {
		projName = "未命名项目"
	}
	row := ui.ButtonBase(c.Key("focus-task-"+task.ID)).Padding(10, 14).Gap(10).Radius(tk.BorderRadius).
		Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).AlignItems(ui.Center).
		Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Cursor(ui.CursorPointer).
		Label("打开聚焦任务 " + task.Title)
	row.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
			if task.Priority != model.NoPriority {
				uiTag(c, priorityLabel(task.Priority), tagProps{
					Size:    sizeSmall,
					Variant: tagFilled,
					Color:   priorityTagColor(task.Priority),
				})
			}
			ui.Text(c, task.Title).FontSize(14).FontWeight(500).TextColor(tk.Text).Grow(1).SingleLine()
		})

		ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			uiTag(c, projName, tagProps{
				Size:    sizeSmall,
				Variant: tagOutlined,
				Color:   tagGray,
			})
			if task.Status == model.InProgress {
				uiTag(c, "进行中", tagProps{
					Size:    sizeSmall,
					Variant: tagFilled,
					Color:   tagProcessing,
				})
			}
			if len(task.Subtasks) > 0 {
				doneCnt := 0
				for _, s := range task.Subtasks {
					if s.Done {
						doneCnt++
					}
				}
				uiTag(c, fmt.Sprintf("%d/%d", doneCnt, len(task.Subtasks)), tagProps{
					Size:    sizeSmall,
					Variant: tagFilled,
					Color:   tagGray,
				})
			}
			if task.DueDate != "" {
				dueColor := tagGray
				dueText := task.DueDate
				if task.DueDate < todayStr {
					dueColor = tagError
					dueText = "已逾期 " + task.DueDate
				} else if task.DueDate <= soonCutoff {
					dueColor = tagOrange
					dueText = "截止 " + task.DueDate
				}
				uiTag(c, dueText, tagProps{
					Size:    sizeSmall,
					Variant: tagFilled,
					Color:   dueColor,
				})
			}
		})
	})
	if row.Clicked() {
		w.OpenFocusTask(task)
	}
}

func (w *Workbench) renderFocusJobRow(c *ui.Context, tk *uiTokens, app model.JobApplication) {
	title := strings.TrimSpace(app.CompanyName + " · " + app.PositionName)
	row := ui.ButtonBase(c.Key("focus-job-"+app.ID)).Padding(10, 14).Gap(10).Radius(tk.BorderRadius).
		Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).AlignItems(ui.Center).
		Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Cursor(ui.CursorPointer).
		Label("打开聚焦投递 " + title)
	row.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, app.CompanyName).FontSize(14).FontWeight(600).TextColor(tk.Text).SingleLine()
			ui.Text(c, app.PositionName).FontSize(13).TextColor(tk.TextSecondary).Grow(1).SingleLine()
		})

		ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			stageColor := tagProcessing
			if app.Stage == model.JobStageOffer {
				stageColor = tagGreen
			}
			uiTag(c, app.Stage.Label(), tagProps{
				Size:    sizeSmall,
				Variant: tagFilled,
				Color:   stageColor,
			})
			if app.Channel != "" {
				uiTag(c, app.Channel, tagProps{
					Size:    sizeSmall,
					Variant: tagOutlined,
					Color:   tagGray,
				})
			}
			ui.Text(c, "投递于 "+app.AppliedDate).FontSize(12).TextColor(tk.TextTertiary).SingleLine()
		})
	})
	if row.Clicked() {
		w.OpenFocusJob(app.ID)
	}
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
