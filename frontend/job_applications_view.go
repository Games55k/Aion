package frontend

import (
	"fmt"
	"strings"

	"aion/backend/jobapplications"
	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

func jobViewTitle(page NavPage) string {
	switch page {
	case NavJobActive:
		return "进行中"
	case NavJobOffer:
		return "已获 Offer"
	case NavJobClosed:
		return "已结束"
	default:
		return "全部投递"
	}
}

func jobViewSubtitle(page NavPage) string {
	switch page {
	case NavJobActive:
		return "展示当前处于进行中流程的所有投递记录"
	case NavJobOffer:
		return "展示当前处于 Offer 阶段或以 Offer 结果结束的投递记录"
	case NavJobClosed:
		return "展示已结束流程的投递记录，保留阶段与结束原因记录"
	default:
		return "集中记录公司、岗位、阶段进度、投递链接与跟进历史"
	}
}

func jobOutcomeTagColor(outcome model.JobOutcome) tagColor {
	switch outcome {
	case model.JobOutcomeOfferAccepted:
		return tagSuccess
	case model.JobOutcomeOfferDeclined:
		return tagWarning
	case model.JobOutcomeRejected:
		return tagError
	default:
		return tagGray
	}
}

func jobStageTagColor(stage model.JobStage) tagColor {
	if stage == model.JobStageOffer {
		return tagSuccess
	}
	return tagInfo
}

func briefSingleLineSummary(app model.JobApplication) string {
	if app.ClosureReason != "" {
		return "原因：" + strings.ReplaceAll(app.ClosureReason, "\n", " ")
	}
	if app.Notes != "" {
		return strings.ReplaceAll(app.Notes, "\n", " ")
	}
	return ""
}

func (w *Workbench) renderJobApplicationsView(c *ui.Context, tk *uiTokens) {
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Children(func() {
		// Top Page Header (drag window area + primary CTA)
		ui.Column(c).Padding(24, 28, 16, 28).Gap(14).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Background(tk.BgContainer).DragWindow().Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Wrap().Children(func() {
				ui.Column(c).Gap(4).MinWidth(0).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, jobViewTitle(w.navPage)).FontSize(22).FontWeight(700).TextColor(tk.Text).SingleLine()
						uiTag(c, fmt.Sprintf("当前匹配 %d 条", len(w.filteredJobApps)), tagProps{
							Variant: tagOutlined,
							Size:    sizeSmall,
							Round:   true,
						})
					})
					ui.Text(c, jobViewSubtitle(w.navPage)).FontSize(13).TextColor(tk.TextSecondary).SingleLine()
				})
				createBtn := uiButton(c, "新增投递", buttonProps{Variant: buttonPrimary})
				createBtn.Label("新增投递")
				if createBtn.Clicked() {
					w.OpenJobCreateModal()
				}
			})

			// Top Aggregate Stats Strip (counts all non-deleted records independently of active filters)
			ui.Row(c).Gap(10).Wrap().AlignSelf(ui.Stretch).Children(func() {
				w.renderJobStatPill(c, tk, "总投递", w.jobStats.Total, tagGray)
				w.renderJobStatPill(c, tk, "进行中", w.jobStats.Active, tagProcessing)
				w.renderJobStatPill(c, tk, "已获 Offer", w.jobStats.Offered, tagSuccess)
			})

			// Filter & Sort Bar
			w.renderJobFilterBar(c, tk)
		})

		// Table Area / Empty State
		ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Padding(16, 28, 16, 28).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			if len(w.filteredJobApps) == 0 {
				w.renderJobEmptyState(c, tk)
				return
			}
			w.renderJobTable(c, tk)
			w.renderJobPaginationBar(c, tk)
		})
	})
}

func (w *Workbench) renderJobStatPill(c *ui.Context, tk *uiTokens, label string, count int, color tagColor) {
	ui.Row(c).Padding(8, 14).Gap(10).Radius(tk.BorderRadius).Border(1, tk.BorderSecondary).Background(tk.BgElevated).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(tk.TextSecondary).SingleLine()
		ui.Text(c, fmt.Sprint(count)).FontSize(16).FontWeight(700).TextColor(tk.Text).SingleLine()
	})
}

func (w *Workbench) renderJobFilterBar(c *ui.Context, tk *uiTokens) {
	ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
		// Keyword search (company & position)
		searchWrap := ui.Box(c).Width(220).MinWidth(160)
		searchWrap.Children(func() {
			search := uiSearchBar(c, &w.jobSearchQuery, searchBarProps{
				Placeholder: "搜索公司或岗位...",
				ShortKey:    "mod+k",
				Size:        sizeSmall,
			})
			search.Label("搜索投递记录")
			if search.Changed() {
				w.jobPage = 0
				w.rebuildJobDerived()
			}
		})

		// Stage filter
		stageOptions := make([]selectOption, 0, len(model.AllJobStages())+1)
		stageOptions = append(stageOptions, selectOption{Value: "", Label: "全部阶段"})
		for _, st := range model.AllJobStages() {
			stageOptions = append(stageOptions, selectOption{Value: string(st), Label: st.Label()})
		}
		ui.Box(c).Width(140).Children(func() {
			prev := w.jobStageFilter
			sel := uiSelect(c.Key("job-filter-stage"), &w.jobStageFilter, selectProps{
				Options:     stageOptions,
				Placeholder: "全部阶段",
				Size:        sizeSmall,
			})
			sel.Label("筛选阶段")
			if w.jobStageFilter != prev {
				w.jobPage = 0
				w.rebuildJobDerived()
			}
		})

		// Outcome filter
		outcomeOptions := make([]selectOption, 0, len(model.AllJobOutcomes())+1)
		outcomeOptions = append(outcomeOptions, selectOption{Value: "", Label: "全部结果"})
		for _, out := range model.AllJobOutcomes() {
			outcomeOptions = append(outcomeOptions, selectOption{Value: string(out), Label: out.Label()})
		}
		ui.Box(c).Width(140).Children(func() {
			prev := w.jobOutcomeFilter
			sel := uiSelect(c.Key("job-filter-outcome"), &w.jobOutcomeFilter, selectProps{
				Options:     outcomeOptions,
				Placeholder: "全部结果",
				Size:        sizeSmall,
			})
			sel.Label("筛选结果")
			if w.jobOutcomeFilter != prev {
				w.jobPage = 0
				w.rebuildJobDerived()
			}
		})

		// Applied date filter
		ui.Box(c).Width(150).Children(func() {
			prevDate := formatCalendarDate(w.jobAppliedDateFilter)
			dp := uiDatePicker(c.Key("job-filter-applied"), &w.jobAppliedDateFilter, datePickerProps{
				Placeholder: "投递日期筛选",
				AllowClear:  true,
				Size:        sizeSmall,
			})
			dp.Label("筛选投递日期")
			if formatCalendarDate(w.jobAppliedDateFilter) != prevDate {
				w.jobPage = 0
				w.rebuildJobDerived()
			}
		})

		// Sort selector (visually separated from filters with explicit sort icon & prefix)
		ui.Box(c).Width(1).Height(16).Background(tk.BorderSecondary)
		sortVal := string(w.jobSortBy)
		if sortVal == "" {
			sortVal = string(jobapplications.SortAppliedDesc)
		}
		ui.Box(c).Width(188).Children(func() {
			prevSort := sortVal
			sel := uiSelect(c.Key("job-sort"), &sortVal, selectProps{
				Options: []selectOption{
					{Value: string(jobapplications.SortAppliedDesc), Label: "排序：投递日期 ↓", Icon: iconArrowDownWideNarrow()},
					{Value: string(jobapplications.SortAppliedAsc), Label: "排序：投递日期 ↑", Icon: iconArrowUpNarrowWide()},
					{Value: string(jobapplications.SortUpdatedDesc), Label: "排序：最近更新 ↓", Icon: iconArrowDownWideNarrow()},
					{Value: string(jobapplications.SortUpdatedAsc), Label: "排序：最近更新 ↑", Icon: iconArrowUpNarrowWide()},
				},
				Size: sizeSmall,
			})
			sel.Label("排序方式")
			if sortVal != prevSort {
				w.jobSortBy = jobapplications.JobSort(sortVal)
				w.jobPage = 0
				w.rebuildJobDerived()
			}
		})

		if w.jobSearchQuery != "" || w.jobStageFilter != "" || w.jobOutcomeFilter != "" || !w.jobAppliedDateFilter.IsZero() {
			clearBtn := uiButton(c, "清除筛选", buttonProps{Variant: buttonText, Size: sizeSmall})
			if clearBtn.Clicked() {
				w.ClearJobFilters()
			}
		}
	})
}

func (w *Workbench) renderJobEmptyState(c *ui.Context, tk *uiTokens) {
	hasFilter := w.jobSearchQuery != "" || w.jobStageFilter != "" || w.jobOutcomeFilter != "" || !w.jobAppliedDateFilter.IsZero()
	title := "暂无投递记录"
	desc := "点击「新增投递」记录你的第一条求职投递，支持后续阶段追踪、结果结束与时间线回溯。"
	if hasFilter {
		title = "没有符合筛选条件的记录"
		desc = "请调整关键词、阶段、结束结果或投递日期筛选条件。"
	}
	ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		uiEmpty(c, emptyProps{
			Icon:        iconBriefcase(),
			Title:       title,
			Description: desc,
			Action: func() {
				if hasFilter {
					if uiButton(c, "清空筛选条件", buttonProps{Variant: buttonDefault}).Clicked() {
						w.ClearJobFilters()
					}
				} else {
					if uiButton(c, "立即新增投递", buttonProps{Variant: buttonPrimary}).Clicked() {
						w.OpenJobCreateModal()
					}
				}
			},
		})
	})
}

const (
	jobColCompanyWidth  float32 = 190
	jobColPositionWidth float32 = 260
	jobColStageWidth    float32 = 148
	jobColStatusWidth   float32 = 116
	jobColDateWidth     float32 = 110
	jobColLinkWidth     float32 = 110
)

func jobStatusMenuItems(app model.JobApplication) []menuItem {
	if app.Lifecycle == model.JobLifecycleActive {
		return []menuItem{
			{Key: "reject", Label: "标为未通过（当前阶段挂）", Icon: iconCircleX(), Danger: true},
			{Key: "withdrawn", Label: "标为主动放弃", Icon: iconBan()},
			{Divider: true},
			{Key: "close_modal", Label: "结束流程（填写原因 / Offer 结果）…", Icon: iconFileText()},
		}
	}
	return []menuItem{
		{Key: "reopen", Label: "重新开启流程", Icon: iconRotateCcw()},
		{Key: "detail", Label: "查看 / 编辑结束原因…", Icon: iconFileText()},
	}
}

func (w *Workbench) renderJobTable(c *ui.Context, tk *uiTokens) {
	rows, _, _ := w.pagedJobApps()

	// Priority columns (公司, 岗位, 阶段, 状态/结果) come first so narrow windows keep core progress immediately visible while secondary columns scroll horizontally.
	tableContainer := ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).AlignSelf(ui.Stretch).
		Radius(tk.BorderRadius).Border(1, tk.BorderSecondary).Background(tk.BgContainer).Clip()

	tableContainer.Children(func() {
		uiScrollArea(c, scrollAreaProps{
			Direction: scrollBoth,
			State:     &w.jobTableScroll,
		}, func() {
			ui.Column(c).MinWidth(900).FillWidth().Children(func() {
				// Table Header Row
				ui.Row(c).Padding(10, 14).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).
					Background(tk.FillQuaternary).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
					tableHeaderCell(c, tk, "公司", jobColCompanyWidth)
					tableHeaderCell(c, tk, "岗位", jobColPositionWidth)
					tableHeaderCell(c, tk, "当前阶段", jobColStageWidth)
					tableHeaderCell(c, tk, "状态 / 结果", jobColStatusWidth)
					tableHeaderCell(c, tk, "投递日期", jobColDateWidth)
					tableHeaderCell(c, tk, "链接与操作", jobColLinkWidth)
				})

				// Visible Paginated Rows
				for idx, app := range rows {
					w.renderJobTableRow(c, tk, app, idx)
				}
			})
		}).Fill()
	})
}

func tableHeaderCell(c *ui.Context, tk *uiTokens, title string, width float32) {
	ui.Row(c).Width(width).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(tk.TextSecondary).SingleLine()
	})
}

func (w *Workbench) renderJobTableRow(c *ui.Context, tk *uiTokens, app model.JobApplication, idx int) {
	row := ui.Row(c.Key("job-row-"+app.ID)).Padding(10, 14).Gap(10).MinHeight(56).AlignSelf(ui.Stretch).
		AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
	if idx%2 == 1 {
		row.Background(tk.FillQuaternary.Alpha(0.35))
	}
	row.Children(func() {
		// 1. Company (clickable to open detail modal)
		compBtn := ui.ButtonBase(c.Key("open-comp-"+app.ID)).Width(jobColCompanyWidth).Shrink(0).
			Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer).
			Label("查看详情 " + app.CompanyName + " " + app.PositionName)
		compBtn.Children(func() {
			ui.Column(c).MinWidth(0).FillWidth().AlignItems(ui.Center).Gap(2).Children(func() {
				ui.Text(c, app.CompanyName).FontSize(13).FontWeight(600).TextColor(tk.Text).SingleLine()
				if app.Channel != "" {
					ui.Text(c, app.Channel).FontSize(11).TextColor(tk.TextTertiary).SingleLine()
				}
			})
		})
		if compBtn.Clicked() {
			w.SelectJobApplication(app.ID)
		}

		// 2. Position + brief 1-line summary (closure reason / notes never expand row height)
		posBtn := ui.ButtonBase(c.Key("open-pos-"+app.ID)).Width(jobColPositionWidth).Shrink(0).
			Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer).
			Label("打开投递详情 " + app.PositionName)
		posBtn.Children(func() {
			ui.Column(c).MinWidth(0).FillWidth().AlignItems(ui.Center).Gap(2).Children(func() {
				ui.Text(c, app.PositionName).FontSize(13).TextColor(tk.Text).SingleLine()
				if summary := briefSingleLineSummary(app); summary != "" {
					ui.Text(c, summary).FontSize(11).TextColor(tk.TextTertiary).SingleLine()
				}
			})
		})
		if posBtn.Clicked() {
			w.SelectJobApplication(app.ID)
		}

		// 3. Inline Stage Selector (with quick close / reopen actions)
		stageOpts := make([]selectOption, 0, len(model.AllJobStages())+3)
		for _, st := range model.AllJobStages() {
			stageOpts = append(stageOpts, selectOption{Value: string(st), Label: st.Label()})
		}
		if app.Lifecycle == model.JobLifecycleActive {
			stageOpts = append(stageOpts,
				selectOption{Value: "__divider__", Label: "── 快捷结束流程 ──", Disabled: true},
				selectOption{Value: "__quick_reject__", Label: "标为未通过（当前阶段挂）"},
				selectOption{Value: "__open_close_modal__", Label: "更多结束结果 / 原因…"},
			)
		} else {
			stageOpts = append(stageOpts,
				selectOption{Value: "__divider__", Label: "── 流程状态操作 ──", Disabled: true},
				selectOption{Value: "__reopen__", Label: "重新开启流程"},
			)
		}
		ui.Box(c).Width(jobColStageWidth).Shrink(0).Children(func() {
			selectedStage := string(app.Stage)
			sel := uiSelect(c.Key("row-stage-"+app.ID), &selectedStage, selectProps{
				Options: stageOpts,
				Size:    sizeSmall,
			})
			sel.Label("阶段 " + app.CompanyName + " " + app.PositionName)
			if selectedStage != string(app.Stage) {
				switch selectedStage {
				case "__quick_reject__":
					w.QuickCloseJobApplication(app.ID, model.JobOutcomeRejected)
				case "__open_close_modal__":
					w.OpenJobCloseModal(app.ID)
				case "__reopen__":
					w.ReopenJobApplication(app.ID)
				default:
					w.ChangeJobStage(app.ID, model.JobStage(selectedStage))
				}
			}
		})

		// 4. Lifecycle / Outcome Tag with Quick Close / Reopen Menu
		ui.Row(c).Width(jobColStatusWidth).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			statusOpen := ui.Local(row, "statusMenuOpen", func() bool { return false })
			chosen := uiDropdownMenu(c.Key("status-menu-"+app.ID), statusOpen, dropdownMenuProps{
				Items: jobStatusMenuItems(app),
			}, func(trig ui.Element) {
				trig.Cursor(ui.CursorPointer).Radius(tk.BorderRadiusSM).Padding(3, 5).
					Justify(ui.Center).AlignItems(ui.Center).
					Label("状态操作 " + app.CompanyName + " " + app.PositionName)
				if trig.Hovered() || *statusOpen {
					trig.Background(tk.FillTertiary)
				}
				trig.Children(func() {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						if app.Lifecycle == model.JobLifecycleActive {
							uiTag(c, app.Lifecycle.Label(), tagProps{
								Color: tagProcessing,
								Size:  sizeSmall,
							})
						} else {
							uiTag(c, app.Outcome.Label(), tagProps{
								Color: jobOutcomeTagColor(app.Outcome),
								Size:  sizeSmall,
							})
						}
						ui.Icon(c, iconChevronDown()).FontSize(10).TextColor(tk.TextTertiary)
					})
				})
			})
			switch chosen {
			case "reject":
				w.QuickCloseJobApplication(app.ID, model.JobOutcomeRejected)
			case "withdrawn":
				w.QuickCloseJobApplication(app.ID, model.JobOutcomeWithdrawn)
			case "close_modal":
				w.OpenJobCloseModal(app.ID)
			case "reopen":
				w.ReopenJobApplication(app.ID)
			case "detail":
				w.SelectJobApplication(app.ID)
			}
		})

		// 5. Applied Date
		ui.Row(c).Width(jobColDateWidth).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Text(c, app.AppliedDate).FontSize(12).TextColor(tk.TextSecondary).SingleLine()
		})

		// 6. Application URL Copy / Open & Detail Trigger
		ui.Row(c).Width(jobColLinkWidth).Shrink(0).Gap(4).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			if app.ApplicationURL != "" {
				if uiActionIcon(c.Key("copy-url-"+app.ID), iconCopy(), actionIconProps{
					Title: "复制链接: " + app.CompanyName + " " + app.PositionName,
				}).Clicked() {
					c.Services().WriteClipboard(app.ApplicationURL)
					w.EnqueueToast(ToastProps{
						Title:       "已复制投递链接",
						Description: app.ApplicationURL,
						Type:        ToastSuccess,
					})
				}
				if uiActionIcon(c.Key("open-url-"+app.ID), iconExternalLink(), actionIconProps{
					Title: "打开链接: " + app.CompanyName + " " + app.PositionName,
				}).Clicked() {
					w.OpenJobURL(app.ApplicationURL)
				}
			} else {
				ui.Text(c, "无链接").FontSize(11).TextColor(tk.TextQuaternary).SingleLine()
			}
		})
	})
}

func (w *Workbench) renderJobPaginationBar(c *ui.Context, tk *uiTokens) {
	_, page, totalPages := w.pagedJobApps()
	if totalPages <= 1 {
		return
	}
	ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
		ui.Text(c, fmt.Sprintf("第 %d / %d 页 · 共 %d 条记录", page+1, totalPages, len(w.filteredJobApps))).
			FontSize(12).TextColor(tk.TextSecondary).SingleLine()
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			prevBtn := uiButton(c, "上一页", buttonProps{Variant: buttonDefault, Size: sizeSmall})
			if page <= 0 {
				prevBtn.Disabled(true)
			} else if prevBtn.Clicked() {
				w.SetJobPage(page - 1)
			}
			nextBtn := uiButton(c, "下一页", buttonProps{Variant: buttonDefault, Size: sizeSmall})
			if page >= totalPages-1 {
				nextBtn.Disabled(true)
			} else if nextBtn.Clicked() {
				w.SetJobPage(page + 1)
			}
		})
	})
}

func (w *Workbench) renderJobCreateModal(c *ui.Context, tk *uiTokens) {
	if !w.jobCreateOpen {
		return
	}
	var dismissed bool
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Padding(32, 24).Background(tk.BgContainer.Alpha(0.6))
		panel.Width(560).MaxWidthPercent(100).MaxHeightPercent(100).Radius(12).Clip().
			Background(tk.BgElevated).Border(1, tk.BorderSecondary).TextColor(tk.Text).FontSize(tk.FontSize).
			Label("新增投递弹窗")
		tk.Elevate(panel, boxShadow)

		if panel.Dismissed() {
			dismissed = true
		}

		ui.Column(c).FillWidth().Padding(20, 24).Gap(14).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "新增投递记录").FontSize(17).FontWeight(700).TextColor(tk.Text)
				if uiActionIcon(c, iconX(), actionIconProps{Title: "关闭新增投递弹窗"}).Clicked() {
					w.jobCreateOpen = false
				}
			})

			// Non-blocking warning if an active application already matches company + position
			if dupes := w.activeDuplicateMatches(w.jobCreateCompany, w.jobCreatePosition); len(dupes) > 0 {
				uiAlert(c, fmt.Sprintf("检测到 %d 条同公司同岗位的进行中记录", len(dupes)), alertProps{
					Type:        alertWarning,
					Variant:     alertSoft,
					Description: "仍可继续创建独立记录（例如同一岗位再次投递或区分不同渠道）。",
				})
			}

			if w.jobCreateErr != "" {
				uiAlert(c, w.jobCreateErr, alertProps{Type: alertError, Variant: alertSoft})
			}

			uiScrollArea(c, scrollAreaProps{Direction: scrollVertical}, func() {
				ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							ui.Text(c, "公司名称 *").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
							uiInput(c, &w.jobCreateCompany, inputProps{Placeholder: "如：示例科技"}).Label("创建公司名称")
						})
						ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							ui.Text(c, "岗位名称 *").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
							uiInput(c, &w.jobCreatePosition, inputProps{Placeholder: "如：Go 后端工程师"}).Label("创建岗位名称")
						})
					})

					ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							ui.Text(c, "投递日期 *").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
							uiDatePicker(c.Key("create-applied-date"), &w.jobCreateAppliedDate, datePickerProps{
								Placeholder: "选择投递日期",
							}).Label("创建投递日期")
						})
						ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							ui.Text(c, "当前阶段").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
							opts := make([]selectOption, 0, len(model.AllJobStages()))
							for _, st := range model.AllJobStages() {
								opts = append(opts, selectOption{Value: string(st), Label: st.Label()})
							}
							uiSelect(c.Key("create-stage"), &w.jobCreateStage, selectProps{
								Options: opts,
							}).Label("创建当前阶段")
						})
					})

					ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "投递渠道").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
						uiInput(c, &w.jobCreateChannel, inputProps{Placeholder: "官网 / 内推 / 招聘平台"}).Label("创建投递渠道")
					})

					ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "投递链接").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
						uiInput(c, &w.jobCreateURL, inputProps{Placeholder: "https://..."}).Label("创建投递链接")
					})

					ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "通用备注").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
						uiTextArea(c, &w.jobCreateNotes, textAreaProps{
							Placeholder: "记录岗位亮点、面试要求或个人备忘...",
							MinRows:     3,
							MaxRows:     6,
						}).Label("创建通用备注")
					})
				})
			}).MaxHeight(400)

			ui.Row(c).Gap(10).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
				if uiButton(c, "取消", buttonProps{Variant: buttonDefault}).Clicked() {
					w.jobCreateOpen = false
				}
				submit := uiButton(c, "确认创建", buttonProps{Variant: buttonPrimary})
				if submit.Clicked() {
					w.SubmitJobCreateModal()
				}
			})
		})
	})
	if dismissed {
		w.jobCreateOpen = false
	}
}
