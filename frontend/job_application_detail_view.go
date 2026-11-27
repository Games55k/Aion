package frontend

import (
	"fmt"
	"strings"
	"time"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

func (w *Workbench) renderJobDetailModal(c *ui.Context, tk *uiTokens) {
	if !w.jobDetailModalOpen() {
		return
	}
	app := *w.detailJob
	var panelDismissed bool
	open := true

	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Padding(16).Background(ui.RGBA(24, 24, 24, 0.24))
		panel.Width(960).MaxWidthPercent(100).MaxHeightPercent(100).Radius(24).Clip().
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Label("投递详情弹窗")
		tk.Elevate(panel, boxShadow)

		if panel.Dismissed() {
			panelDismissed = true
		}

		ui.Column(c).FillWidth().MaxHeightPercent(100).Children(func() {
			// Modal Header Bar (18px bold, zero border noise)
			ui.Row(c).Padding(16, 24, 12, 24).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).Grow(1).MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Box(c).MinWidth(0).Shrink(1).Children(func() {
						ui.Text(c, fmt.Sprintf("%s · %s", app.CompanyName, app.PositionName)).
							FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
					})
					ui.Box(c).Shrink(0).Children(func() {
						w.renderJobSaveStatusBadge(c)
					})
				})
				ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
					delBtn := uiButton(c, "移入回收站", buttonProps{
						Variant: buttonDefault,
						Size:    sizeSmall,
						Danger:  true,
					})
					if delBtn.Clicked() {
						w.DeleteJobApplication(app.ID)
					}
					if uiActionIcon(c, iconX(), actionIconProps{
						Title: "关闭投递详情",
					}).Clicked() {
						w.CloseJobDetail()
					}
				})
			})

			// Validation / Persistence Error Banner with Retry & Discard
			if w.jobSaveErr != "" && (w.jobSaveState == "validation_error" || w.jobSaveState == "error") {
				ui.Box(c).Padding(10, 22, 0, 22).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
						alertType := alertError
						if w.jobSaveState == "validation_error" {
							alertType = alertWarning
						}
						uiAlert(c, w.jobSaveErr, alertProps{
							Type: alertType,
						})
						ui.Row(c).Gap(8).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
							if w.jobSaveState == "error" {
								if uiButton(c, "重试保存", buttonProps{
									Variant: buttonPrimary,
									Size:    sizeSmall,
								}).Clicked() {
									_ = w.SaveSelectedJob()
								}
							}
							if uiButton(c, "放弃修改", buttonProps{
								Variant: buttonDefault,
								Size:    sizeSmall,
							}).Clicked() {
								w.DiscardJobChanges()
							}
						})
					})
				})
			}

			// Scrollable Detail Content Body: full-width workflow stepper + two-column details & timeline
			uiScrollArea(c, scrollAreaProps{Direction: scrollVertical}, func() {
				ui.Column(c).Padding(18, 22, 22, 22).Gap(16).AlignSelf(ui.Stretch).Children(func() {
					w.renderJobWorkflowSection(c, tk, app)
					ui.Row(c).Gap(20).AlignItems(ui.Start).AlignSelf(ui.Stretch).Wrap().Children(func() {
						ui.Column(c).Grow(1).Basis(360).MinWidth(300).Children(func() {
							w.renderJobMetadataSection(c, tk)
						})
						ui.Column(c).Grow(1).Basis(420).MinWidth(320).Children(func() {
							w.renderJobTimelineSection(c, tk, app)
						})
					})
				})
			}).Grow(1).MinHeight(0).MaxHeight(640).AlignSelf(ui.Stretch)
		})
	})

	if panelDismissed {
		w.CloseJobDetail()
	}
}

func (w *Workbench) renderJobSaveStatusBadge(c *ui.Context) {
	switch w.jobSaveState {
	case "saving":
		uiTag(c, "正在保存…", tagProps{Color: tagProcessing, Size: sizeSmall})
	case "validation_error":
		uiTag(c, "校验未通过", tagProps{Color: tagWarning, Size: sizeSmall})
	case "error":
		uiTag(c, "保存失败", tagProps{Color: tagError, Size: sizeSmall})
	default:
		label := "已自动保存"
		if !w.jobLastSavedAt.IsZero() {
			label = "已保存 " + w.jobLastSavedAt.In(time.Local).Format("15:04:05")
		}
		uiTag(c, label, tagProps{Color: tagSuccess, Variant: tagBorderless, Size: sizeSmall})
	}
}

func (w *Workbench) renderJobWorkflowSection(c *ui.Context, tk *uiTokens, app model.JobApplication) {
	ui.Column(c).Padding(16).Gap(12).Radius(16).Background(tk.FillTertiary).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).Wrap().AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				ui.Text(c, "当前阶段").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				stageOpts := make([]selectOption, 0, len(model.AllJobStages()))
				for _, st := range model.AllJobStages() {
					stageOpts = append(stageOpts, selectOption{Value: string(st), Label: st.Label()})
				}
				ui.Box(c).Width(155).Children(func() {
					stageVal := string(app.Stage)
					uiSelect(c.Key("detail-stage"), &stageVal, selectProps{
						Options: stageOpts,
						Size:    sizeSmall,
					}).Label("详情阶段")
					if stageVal != string(app.Stage) {
						w.ChangeJobStage(app.ID, model.JobStage(stageVal))
					}
				})
				if app.Lifecycle == model.JobLifecycleActive {
					uiTag(c, app.Lifecycle.Label(), tagProps{Color: tagProcessing, Size: sizeSmall})
				} else {
					uiTag(c, app.Lifecycle.Label(), tagProps{Color: tagGray, Size: sizeSmall})
					uiTag(c, app.Outcome.Label(), tagProps{Color: jobOutcomeTagColor(app.Outcome), Size: sizeSmall})
				}
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if app.Lifecycle == model.JobLifecycleActive {
					rejBtn := uiButton(c, "标为未通过", buttonProps{
						Variant: buttonDefault,
						Size:    sizeSmall,
						Danger:  true,
					}).Label("快速标为未通过")
					if rejBtn.Clicked() {
						w.QuickCloseJobApplication(app.ID, model.JobOutcomeRejected)
					}
					closeBtn := uiButton(c, "结束流程", buttonProps{Variant: buttonDefault, Size: sizeSmall})
					if closeBtn.Clicked() {
						w.OpenJobCloseModal(app.ID)
					}
				} else {
					reopenBtn := uiButton(c, "重新开启", buttonProps{Variant: buttonPrimary, Size: sizeSmall})
					if reopenBtn.Clicked() {
						w.OpenJobReopenModal(app.ID)
					}
				}
			})
		})

		// Interactive Horizontal Stage Stepper
		w.renderJobStageStepper(c, tk, app)

		if app.Lifecycle == model.JobLifecycleClosed {
			closedText := "流程已结束：调整阶段仅作记录更正，不会自动重新开启；若继续推进请点击「重新开启」。"
			if app.ClosedAt != nil {
				closedText = fmt.Sprintf("结束时间：%s · %s", app.ClosedAt.In(time.Local).Format("2006-01-02 15:04"), closedText)
			}
			ui.Text(c, closedText).FontSize(12).TextColor(tk.TextSecondary)

			ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束原因").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				area := uiTextArea(c.Key("detail-closure-reason"), &w.editJobClosureReason, textAreaProps{
					Placeholder: "补充或编辑结束原因（重新开启后会保留在历史事件中）...",
					MinRows:     2,
					MaxRows:     4,
				})
				area.Label("详情结束原因")
				if area.Changed() {
					w.editJobReasonDirty = true
					w.TouchJobDetailField()
				}
			})
		}
	})
}

func (w *Workbench) renderJobStageStepper(c *ui.Context, tk *uiTokens, app model.JobApplication) {
	stages := model.AllJobStages()
	currentIdx := -1
	for i, st := range stages {
		if st == app.Stage {
			currentIdx = i
			break
		}
	}
	ui.Row(c).Gap(4).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
		for i, st := range stages {
			isCurrent := i == currentIdx
			isPast := currentIdx >= 0 && i < currentIdx
			node := ui.ButtonBase(c.Key("stepper-"+string(st))).Padding(5, 10).Radius(tk.BorderRadiusSM).
				Gap(5).AlignItems(ui.Center).Cursor(ui.CursorPointer).
				Label("推进阶段至 " + st.Label())

			fg := tk.TextTertiary
			fontWeight := 400
			switch {
			case isCurrent && app.Lifecycle == model.JobLifecycleClosed && app.Outcome == model.JobOutcomeRejected:
				node.Background(tk.ErrorBg).Border(1, tk.Error)
				fg = tk.Error
				fontWeight = 600
			case isCurrent && (st == model.JobStageOffer || app.Outcome == model.JobOutcomeOfferAccepted):
				node.Background(tk.SuccessBg).Border(1, tk.Success)
				fg = tk.Success
				fontWeight = 600
			case isCurrent:
				node.Background(tk.Primary).Border(1, tk.Primary)
				fg = tk.BgContainer
				fontWeight = 600
			case isPast:
				if node.Hovered() {
					node.Background(tk.FillTertiary).Border(1, tk.Primary)
				} else {
					node.Background(tk.PrimaryFillQuaternary).Border(1, tk.Primary.Alpha(0.35))
				}
				fg = tk.Primary
				fontWeight = 500
			default:
				if node.Hovered() {
					node.Background(tk.FillTertiary).Border(1, tk.Border)
					fg = tk.TextSecondary
				} else {
					node.Background(tk.FillQuaternary).Border(1, tk.BorderSecondary)
				}
			}

			node.Children(func() {
				if isPast {
					ui.Icon(c, iconCheck()).FontSize(11).TextColor(fg)
				}
				ui.Text(c, st.Label()).FontSize(12).FontWeight(fontWeight).TextColor(fg).SingleLine()
			})
			if node.Clicked() && !isCurrent {
				w.ChangeJobStage(app.ID, st)
			}

			if i < len(stages)-1 {
				lineColor := tk.BorderSecondary
				if isPast {
					lineColor = tk.Primary.Alpha(0.45)
				}
				ui.Box(c.Key("step-conn-"+string(st))).Width(8).Height(2).Shrink(0).Background(lineColor)
			}
		}
	})
}

func (w *Workbench) renderJobMetadataSection(c *ui.Context, tk *uiTokens) {
	ui.Column(c).Padding(4, 0).Gap(12).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, "基本信息与备注").FontSize(15).FontWeight(600).TextColor(tk.Text)

		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "公司名称 *").FontSize(12).TextColor(tk.TextSecondary)
			in := uiInput(c.Key("detail-company"), &w.editJobCompany, inputProps{
				Placeholder: "公司名称",
				Invalid:     strings.TrimSpace(w.editJobCompany) == "",
			})
			in.Label("详情公司名称")
			if in.Changed() {
				w.TouchJobDetailField()
			}
		})

		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "岗位名称 *").FontSize(12).TextColor(tk.TextSecondary)
			in := uiInput(c.Key("detail-position"), &w.editJobPosition, inputProps{
				Placeholder: "岗位名称",
				Invalid:     strings.TrimSpace(w.editJobPosition) == "",
			})
			in.Label("详情岗位名称")
			if in.Changed() {
				w.TouchJobDetailField()
			}
		})

		ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "投递日期 *").FontSize(12).TextColor(tk.TextSecondary)
				prev := formatCalendarDate(w.editJobAppliedDate)
				dp := uiDatePicker(c.Key("detail-applied-date"), &w.editJobAppliedDate, datePickerProps{
					Placeholder: "选择投递日期",
				})
				dp.Label("详情投递日期")
				if formatCalendarDate(w.editJobAppliedDate) != prev {
					w.TouchJobDetailField()
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "投递渠道").FontSize(12).TextColor(tk.TextSecondary)
				in := uiInput(c.Key("detail-channel"), &w.editJobChannel, inputProps{
					Placeholder: "官网 / 内推 / 招聘平台",
				})
				in.Label("详情投递渠道")
				if in.Changed() {
					w.TouchJobDetailField()
				}
			})
		})

		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "投递链接").FontSize(12).TextColor(tk.TextSecondary)
			ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					_, urlErr := model.ValidateJobURL(w.editJobURL)
					in := uiInput(c.Key("detail-url"), &w.editJobURL, inputProps{
						Placeholder: "https://...",
						Invalid:     urlErr != nil,
					})
					in.Label("详情投递链接")
					if in.Changed() {
						w.TouchJobDetailField()
					}
				})
				copyBtn := uiButton(c, "复制链接", buttonProps{Variant: buttonDefault, Size: sizeSmall})
				if strings.TrimSpace(w.editJobURL) == "" {
					copyBtn.Disabled(true)
				} else if copyBtn.Clicked() {
					c.Services().WriteClipboard(strings.TrimSpace(w.editJobURL))
				}
				openBtn := uiButton(c, "打开链接", buttonProps{Variant: buttonDefault, Size: sizeSmall})
				if strings.TrimSpace(w.editJobURL) == "" {
					openBtn.Disabled(true)
				} else if openBtn.Clicked() {
					w.OpenJobURL(w.editJobURL)
				}
			})
		})

		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "通用备注").FontSize(12).TextColor(tk.TextSecondary)
			area := uiTextArea(c.Key("detail-notes"), &w.editJobNotes, textAreaProps{
				Placeholder: "记录岗位详情、面试题目或谈薪信息（自动保存，不产生冗余时间线事件）...",
				MinRows:     3,
				MaxRows:     8,
			})
			area.Label("详情通用备注")
			if area.Changed() {
				w.TouchJobDetailField()
			}
		})
	})
}

func formatTimelineSummary(ev model.JobApplicationEvent, app model.JobApplication) (string, string) {
	switch ev.EventType {
	case model.JobEventCreated:
		title := fmt.Sprintf("创建投递：%s / %s", app.CompanyName, app.PositionName)
		if ev.ToStage != "" {
			title += fmt.Sprintf("（初始阶段：%s）", ev.ToStage.Label())
		}
		return title, ev.Note
	case model.JobEventStageChanged:
		return fmt.Sprintf("%s → %s", ev.FromStage.Label(), ev.ToStage.Label()), ev.Note
	case model.JobEventStageBackfilled:
		return fmt.Sprintf("补录阶段：%s", ev.ToStage.Label()), ev.Note
	case model.JobEventClosed:
		title := fmt.Sprintf("流程结束：%s", ev.Outcome.Label())
		if ev.ToStage != "" {
			title += fmt.Sprintf("（结束阶段：%s）", ev.ToStage.Label())
		}
		var parts []string
		if ev.Reason != "" {
			parts = append(parts, "原因："+ev.Reason)
		}
		if ev.Note != "" {
			parts = append(parts, ev.Note)
		}
		return title, strings.Join(parts, " · ")
	case model.JobEventReopened:
		title := "重新开启流程"
		if ev.Outcome != "" {
			title += fmt.Sprintf("（原结果：%s）", ev.Outcome.Label())
		}
		var parts []string
		if ev.Reason != "" {
			parts = append(parts, "原原因："+ev.Reason)
		}
		if ev.Note != "" {
			parts = append(parts, ev.Note)
		}
		return title, strings.Join(parts, " · ")
	case model.JobEventNoteAdded:
		return "跟进备注", ev.Note
	default:
		return ev.EventType.Label(), ev.Note
	}
}

func timelineEventDot(c *ui.Context, tk *uiTokens, ev model.JobApplicationEvent) {
	dot := ui.Box(c).Size(10, 10).Radius(5).Margin(5, 0, 0, 0).Shrink(0)
	switch ev.EventType {
	case model.JobEventStageBackfilled:
		dot.Background(tk.BgElevated).Border(2, tk.Primary)
	case model.JobEventClosed:
		switch ev.Outcome {
		case model.JobOutcomeOfferAccepted:
			dot.Background(tk.Success)
		case model.JobOutcomeRejected:
			dot.Background(tk.Error)
		default:
			dot.Background(tk.Warning)
		}
	case model.JobEventReopened, model.JobEventCreated, model.JobEventStageChanged:
		if ev.EventType == model.JobEventReopened || ev.ToStage == model.JobStageOffer {
			dot.Background(tk.Success)
		} else {
			dot.Background(tk.Primary)
		}
	case model.JobEventNoteAdded:
		dot.Background(tk.Warning)
	default:
		dot.Background(tk.Border)
	}
}

func canDeleteTimelineEvent(ev model.JobApplicationEvent) bool {
	return ev.EventType != model.JobEventCreated
}

func (w *Workbench) renderJobTimelineSection(c *ui.Context, tk *uiTokens, app model.JobApplication) {
	showBackfill := w.jobBackfillExpanded || w.jobBackfillErr != ""
	ui.Column(c).Padding(12).Gap(8).Radius(16).Background(tk.FillTertiary).AlignSelf(ui.Stretch).Children(func() {
		// Timeline Header & Toolbar (Sort Order + Collapsible Backfill)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Wrap().AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "流程时间线").FontSize(13).FontWeight(600).TextColor(tk.Text)
				uiTag(c, fmt.Sprintf("共 %d 条", len(w.detailJobEvents)), tagProps{
					Variant: tagOutlined,
					Size:    sizeSmall,
				})
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				orderLabel := "排序：最新优先 ↓"
				if !w.jobTimelineDesc {
					orderLabel = "排序：时间正序 ↑"
				}
				orderBtn := uiButton(c.Key("toggle-timeline-order"), "", buttonProps{
					Variant: buttonText,
					Size:    sizeSmall,
				}).AlignItems(ui.Center).Label("切换时间线排序")
				orderBtn.Children(func() {
					ui.Icon(c, iconArrowUpDown()).FontSize(12)
					ui.Text(c, orderLabel).SingleLine()
				})
				if orderBtn.Clicked() {
					w.jobTimelineDesc = !w.jobTimelineDesc
				}

				backfillLabel := "+ 补录历史节点"
				if showBackfill {
					backfillLabel = "收起补录"
				}
				backfillBtn := uiButton(c.Key("toggle-backfill-panel"), backfillLabel, buttonProps{
					Variant: buttonDefault,
					Size:    sizeSmall,
				}).Label("切换补录历史节点")
				if backfillBtn.Clicked() {
					w.jobBackfillExpanded = !w.jobBackfillExpanded
				}
			})
		})

		// Compact Top Quick-Note Bar
		ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			if w.jobNoteErr != "" {
				uiAlert(c, w.jobNoteErr, alertProps{Type: alertError})
			}
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(150).Children(func() {
					uiInput(c.Key("timeline-note-input"), &w.jobNoteInput, inputProps{
						Placeholder: "记录沟通结果或面试反馈...",
						Size:        sizeSmall,
						Variant:     inputElevated,
					}).Label("时间线备注输入")
				})
				ui.Box(c).Width(128).Shrink(0).Children(func() {
					uiDatePicker(c.Key("timeline-note-date"), &w.jobNoteDate, datePickerProps{
						Size: sizeSmall,
					}).Label("时间线备注日期")
				})
				if uiButton(c.Key("add-timeline-note"), "添加备注", buttonProps{
					Variant: buttonDefault,
					Size:    sizeSmall,
				}).Clicked() {
					w.AddJobTimelineNote(app.ID)
				}
			})
		})

		// Collapsible Backfill Historical Stage Form (never overwrites current stage or lifecycle)
		if showBackfill {
			ui.Column(c).Padding(10).Gap(8).Radius(tk.BorderRadiusSM).Background(tk.FillQuaternary.Alpha(0.55)).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Wrap().AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "补录历史阶段事件").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
					ui.Text(c, "仅记录过去经历，不会更改当前阶段与流程状态").FontSize(11).TextColor(tk.TextTertiary)
				})
				if w.jobBackfillErr != "" {
					uiAlert(c, w.jobBackfillErr, alertProps{Type: alertError})
				}
				stageOpts := make([]selectOption, 0, len(model.AllJobStages()))
				for _, st := range model.AllJobStages() {
					stageOpts = append(stageOpts, selectOption{Value: string(st), Label: st.Label()})
				}
				ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
					ui.Box(c).Width(130).Children(func() {
						uiSelect(c.Key("backfill-stage"), &w.jobBackfillStage, selectProps{
							Options: stageOpts,
							Size:    sizeSmall,
						}).Label("补录阶段")
					})
					ui.Box(c).Width(132).Children(func() {
						uiDatePicker(c.Key("backfill-date"), &w.jobBackfillDate, datePickerProps{
							Size: sizeSmall,
						}).Label("补录发生日期")
					})
					ui.Box(c).Grow(1).MinWidth(130).Children(func() {
						uiInput(c.Key("backfill-note"), &w.jobBackfillNote, inputProps{
							Placeholder: "可选说明...",
							Size:        sizeSmall,
						}).Label("补录说明")
					})
					if uiButton(c.Key("submit-backfill"), "补录历史事件", buttonProps{
						Variant: buttonDefault,
						Size:    sizeSmall,
					}).Clicked() {
						w.BackfillJobStageEvent(app.ID)
					}
				})
			})
		}

		// Keep this frame on one slice when a row action replaces the timeline.
		events := w.detailJobEvents
		ui.Column(c).Gap(0).Padding(4, 0, 0, 0).AlignSelf(ui.Stretch).Children(func() {
			for i := range events {
				index := i
				if w.jobTimelineDesc {
					index = len(events) - 1 - i
				}
				ev := events[index]
				isLast := i == len(events)-1
				headline, sub := formatTimelineSummary(ev, app)
				ui.Row(c.Key("ev-"+ev.ID)).Gap(10).AlignItems(ui.Stretch).AlignSelf(ui.Stretch).Children(func() {
					// Left Rail: semantic dot + vertical connector line
					ui.Column(c).Width(16).Shrink(0).AlignItems(ui.Center).Children(func() {
						timelineEventDot(c, tk, ev)
						if !isLast {
							ui.Box(c).Width(2).Grow(1).MinHeight(16).Margin(3, 0, 0, 0).Background(tk.BorderSecondary)
						}
					})

					// Right Rail: compact node header + editable note/detail box
					ui.Column(c).Grow(1).MinWidth(0).Padding(0, 0, 12, 0).Gap(4).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
								ui.Text(c, ev.OccurredAt.In(time.Local).Format("2006-01-02")).
									FontSize(12).TextColor(tk.TextTertiary).Shrink(0)
								ui.Text(c, headline).FontSize(13).FontWeight(500).TextColor(tk.Text)
								if ev.EventType == model.JobEventStageBackfilled {
									uiTag(c, "补录", tagProps{Size: sizeSmall, Variant: tagOutlined})
								}
							})

							ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
								if uiActionIcon(c.Key("edit-ev-"+ev.ID), iconPencil(), actionIconProps{
									Title: "编辑时间线备注 " + ev.ID,
								}).Clicked() {
									w.StartEditJobTimelineEvent(ev)
								}
								if canDeleteTimelineEvent(ev) {
									if uiActionIcon(c.Key("del-ev-"+ev.ID), iconTrash2(), actionIconProps{
										Title: "删除时间线节点 " + ev.ID,
									}).Clicked() {
										w.DeleteJobTimelineEvent(app.ID, ev.ID)
									}
								}
							})
						})

						if w.editingJobEventID == ev.ID {
							ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
								ui.Box(c).Grow(1).MinWidth(160).Children(func() {
									uiInput(c.Key("edit-ev-input-"+ev.ID), &w.editingJobEventNote, inputProps{
										Placeholder: "输入节点备注...",
										Size:        sizeSmall,
									}).Label("编辑事件备注输入")
								})
								if uiButton(c.Key("save-edit-ev-"+ev.ID), "保存备注", buttonProps{
									Variant: buttonPrimary,
									Size:    sizeSmall,
								}).Clicked() {
									w.SaveJobTimelineEventNote(app.ID, ev.ID)
								}
								if uiButton(c.Key("cancel-edit-ev-"+ev.ID), "取消", buttonProps{
									Variant: buttonDefault,
									Size:    sizeSmall,
								}).Clicked() {
									w.CancelEditJobTimelineEvent()
								}
							})
						} else if sub != "" {
							ui.Box(c).Padding(6, 10).Radius(tk.BorderRadiusSM).Background(tk.FillQuaternary.Alpha(0.45)).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, sub).FontSize(12).TextColor(tk.TextSecondary)
							})
						}
					})
				})
			}
		})
	})
}

func (w *Workbench) renderJobCloseModal(c *ui.Context, tk *uiTokens) {
	if !w.jobCloseOpen {
		return
	}
	currentStage := w.currentStageForJob(w.jobCloseAppID)
	outcome := model.JobOutcome(w.jobCloseOutcome)
	needsOfferConfirm := outcome.RequiresOfferStage() && currentStage != model.JobStageOffer

	var dismissed bool
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Padding(32, 24).Background(ui.RGBA(24, 24, 24, 0.24))
		panel.Width(480).MaxWidthPercent(100).Radius(24).Clip().
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Label("结束流程弹窗")
		tk.Elevate(panel, boxShadow)

		if panel.Dismissed() {
			dismissed = true
		}

		ui.Column(c).FillWidth().Padding(24).Gap(16).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束求职流程").FontSize(18).FontWeight(700).TextColor(tk.Text)
				if uiActionIcon(c, iconX(), actionIconProps{Title: "关闭结束流程弹窗"}).Clicked() {
					w.jobCloseOpen = false
				}
			})

			ui.Text(c, fmt.Sprintf("当前阶段为「%s」。结束流程会保留当前阶段并写入一条不可变历史事件。", currentStage.Label())).
				FontSize(12).TextColor(tk.TextSecondary)

			if w.jobCloseErr != "" {
				uiAlert(c, w.jobCloseErr, alertProps{Type: alertError})
			}

			ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束结果 *").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				opts := make([]selectOption, 0, len(model.AllJobOutcomes()))
				for _, o := range model.AllJobOutcomes() {
					opts = append(opts, selectOption{Value: string(o), Label: o.Label()})
				}
				uiSelect(c.Key("close-outcome"), &w.jobCloseOutcome, selectProps{
					Options: opts,
				}).Label("选择结束结果")
			})

			if needsOfferConfirm {
				uiAlert(c, "当前阶段尚未处于 Offer", alertProps{
					Type:        alertWarning,
					Description: fmt.Sprintf("选择「%s」要求阶段为 Offer，请勾选确认后随结果一并提交。", outcome.Label()),
				})
				uiCheckbox(c, &w.jobCloseConfirmOffer, "确认同步将当前阶段调整为 Offer").
					Label("确认同步将当前阶段调整为 Offer")
			}

			ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束原因（可选）").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				uiTextArea(c, &w.jobCloseReason, textAreaProps{
					Placeholder: "记录失败复盘、放弃理由或薪资差异...",
					MinRows:     3,
					MaxRows:     6,
				}).Label("输入结束原因")
			})

			ui.Row(c).Gap(10).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
				if uiButton(c, "取消", buttonProps{Variant: buttonDefault}).Clicked() {
					w.jobCloseOpen = false
				}
				confirmBtn := uiButton(c, "确认结束流程", buttonProps{Variant: buttonPrimary})
				if confirmBtn.Clicked() {
					w.SubmitJobCloseModal()
				}
			})
		})
	})

	if dismissed {
		w.jobCloseOpen = false
	}
}

func (w *Workbench) renderJobReopenModal(c *ui.Context, tk *uiTokens) {
	if !w.jobReopenOpen {
		return
	}
	var dismissed bool
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Padding(32, 24).Background(ui.RGBA(24, 24, 24, 0.24))
		panel.Width(440).MaxWidthPercent(100).Radius(24).Clip().
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Label("重新开启流程弹窗")
		tk.Elevate(panel, boxShadow)

		if panel.Dismissed() {
			dismissed = true
		}

		ui.Column(c).FillWidth().Padding(24).Gap(16).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "重新开启流程").FontSize(18).FontWeight(700).TextColor(tk.Text)
				if uiActionIcon(c, iconX(), actionIconProps{Title: "关闭重新开启弹窗"}).Clicked() {
					w.jobReopenOpen = false
				}
			})

			ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "重启备注（可选）").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
				uiTextArea(c.Key("reopen-note"), &w.jobReopenNote, textAreaProps{
					Placeholder: "例如：HR 捞起、转岗继续推进...",
					MinRows:     3,
					MaxRows:     5,
				}).Label("输入重新开启备注")
			})

			ui.Row(c).Gap(12).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
				if uiButton(c, "取消", buttonProps{Variant: buttonDefault}).Clicked() {
					w.jobReopenOpen = false
				}
				if uiButton(c, "确认重新开启", buttonProps{Variant: buttonPrimary}).Clicked() {
					w.SubmitJobReopenModal()
				}
			})
		})
	})

	if dismissed {
		w.jobReopenOpen = false
	}
}
