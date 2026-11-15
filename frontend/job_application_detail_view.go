package frontend

import (
	"fmt"
	"strings"
	"time"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (w *Workbench) renderJobDetailModal(c *ui.Context, tk *theme.Tokens) {
	if !w.jobDetailModalOpen() {
		return
	}
	app := *w.detailJob
	var panelDismissed bool
	open := true

	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Padding(28, 24).Background(tk.BgContainer.Alpha(0.6))
		panel.Width(760).MaxWidthPercent(100).MaxHeightPercent(100).Radius(12).Clip().
			Background(tk.BgElevated).Border(1, tk.BorderSecondary).TextColor(tk.Text).FontSize(tk.FontSize).
			Label("投递详情弹窗")
		tk.Elevate(panel, theme.BoxShadow)

		if panel.Dismissed() {
			panelDismissed = true
		}

		ui.Column(c).FillWidth().MaxHeightPercent(100).Children(func() {
			// Modal Header Bar
			ui.Row(c).Padding(16, 22).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).Grow(1).MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Box(c).MinWidth(0).Shrink(1).Children(func() {
						ui.Text(c, fmt.Sprintf("%s / %s", app.CompanyName, app.PositionName)).
							FontSize(16).FontWeight(700).TextColor(tk.Text).SingleLine()
					})
					ui.Box(c).Shrink(0).Children(func() {
						w.renderJobSaveStatusBadge(c, tk)
					})
				})
				ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
					delBtn := action.Button(c, "移入回收站", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
						Danger:  true,
					})
					if delBtn.Clicked() {
						w.DeleteJobApplication(app.ID)
					}
					if action.ActionIcon(c, lucide.X(), action.ActionIconProps{
						Size:  theme.Small,
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
						alertType := feedback.Error
						if w.jobSaveState == "validation_error" {
							alertType = feedback.Warning
						}
						feedback.Alert(c, w.jobSaveErr, feedback.AlertProps{
							Type:    alertType,
							Variant: feedback.Soft,
						})
						ui.Row(c).Gap(8).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
							if w.jobSaveState == "error" {
								if action.Button(c, "重试保存", action.ButtonProps{
									Variant: action.Primary,
									Size:    theme.Small,
								}).Clicked() {
									_ = w.SaveSelectedJob()
								}
							}
							if action.Button(c, "放弃修改", action.ButtonProps{
								Variant: action.Default,
								Size:    theme.Small,
							}).Clicked() {
								w.DiscardJobChanges()
							}
						})
					})
				})
			}

			// Scrollable Detail Content Body
			layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Vertical}, func() {
				ui.Column(c).Padding(18, 22, 22, 22).Gap(18).AlignSelf(ui.Stretch).Children(func() {
					w.renderJobWorkflowSection(c, tk, app)
					w.renderJobMetadataSection(c, tk, app)
					w.renderJobTimelineSection(c, tk, app)
				})
			}).Grow(1).MinHeight(0).MaxHeight(540).AlignSelf(ui.Stretch)
		})
	})

	if panelDismissed {
		w.CloseJobDetail()
	}
}

func (w *Workbench) renderJobSaveStatusBadge(c *ui.Context, tk *theme.Tokens) {
	switch w.jobSaveState {
	case "saving":
		display.Tag(c, "正在保存…", display.TagProps{Color: display.TagProcessing, Size: theme.Small})
	case "validation_error":
		display.Tag(c, "校验未通过", display.TagProps{Color: display.TagWarning, Size: theme.Small})
	case "error":
		display.Tag(c, "保存失败", display.TagProps{Color: display.TagError, Size: theme.Small})
	default:
		label := "已自动保存"
		if !w.jobLastSavedAt.IsZero() {
			label = "已保存 " + w.jobLastSavedAt.In(time.Local).Format("15:04:05")
		}
		display.Tag(c, label, display.TagProps{Color: display.TagSuccess, Variant: display.TagBorderless, Size: theme.Small})
	}
}

func (w *Workbench) renderJobWorkflowSection(c *ui.Context, tk *theme.Tokens, app model.JobApplication) {
	ui.Column(c).Padding(14).Gap(12).Radius(tk.BorderRadius).Border(1, tk.BorderSecondary).Background(tk.BgContainer).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).Wrap().AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				ui.Text(c, "当前阶段").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				stageOpts := make([]field.SelectOption, 0, len(model.AllJobStages()))
				for _, st := range model.AllJobStages() {
					stageOpts = append(stageOpts, field.SelectOption{Value: string(st), Label: st.Label()})
				}
				ui.Box(c).Width(155).Children(func() {
					stageVal := string(app.Stage)
					field.Select(c.Key("detail-stage"), &stageVal, field.SelectProps{
						Options: stageOpts,
						Size:    theme.Small,
					}).Label("详情阶段")
					if stageVal != string(app.Stage) {
						w.ChangeJobStage(app.ID, model.JobStage(stageVal))
					}
				})
				if app.Lifecycle == model.JobLifecycleActive {
					display.Tag(c, app.Lifecycle.Label(), display.TagProps{Color: display.TagProcessing, Size: theme.Small})
				} else {
					display.Tag(c, app.Lifecycle.Label(), display.TagProps{Color: display.TagGray, Size: theme.Small})
					display.Tag(c, app.Outcome.Label(), display.TagProps{Color: jobOutcomeTagColor(app.Outcome), Size: theme.Small})
				}
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if app.Lifecycle == model.JobLifecycleActive {
					closeBtn := action.Button(c, "结束流程", action.ButtonProps{Variant: action.Default, Size: theme.Small})
					if closeBtn.Clicked() {
						w.OpenJobCloseModal(app.ID)
					}
				} else {
					reopenBtn := action.Button(c, "重新开启", action.ButtonProps{Variant: action.Primary, Size: theme.Small})
					if reopenBtn.Clicked() {
						w.ReopenJobApplication(app.ID)
					}
				}
			})
		})

		if app.Lifecycle == model.JobLifecycleClosed {
			closedText := "流程已结束：调整阶段仅作记录更正，不会自动重新开启；若继续推进请点击「重新开启」。"
			if app.ClosedAt != nil {
				closedText = fmt.Sprintf("结束时间：%s · %s", app.ClosedAt.In(time.Local).Format("2006-01-02 15:04"), closedText)
			}
			ui.Text(c, closedText).FontSize(12).TextColor(tk.TextSecondary)

			ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束原因").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				area := field.TextArea(c.Key("detail-closure-reason"), &w.editJobClosureReason, field.TextAreaProps{
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

func (w *Workbench) renderJobMetadataSection(c *ui.Context, tk *theme.Tokens, app model.JobApplication) {
	ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, "基本信息与备注").FontSize(13).FontWeight(600).TextColor(tk.Text)

		ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "公司名称 *").FontSize(12).TextColor(tk.TextSecondary)
				in := field.Input(c.Key("detail-company"), &w.editJobCompany, field.InputProps{
					Placeholder: "公司名称",
					Invalid:     strings.TrimSpace(w.editJobCompany) == "",
				})
				in.Label("详情公司名称")
				if in.Changed() {
					w.TouchJobDetailField()
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "岗位名称 *").FontSize(12).TextColor(tk.TextSecondary)
				in := field.Input(c.Key("detail-position"), &w.editJobPosition, field.InputProps{
					Placeholder: "岗位名称",
					Invalid:     strings.TrimSpace(w.editJobPosition) == "",
				})
				in.Label("详情岗位名称")
				if in.Changed() {
					w.TouchJobDetailField()
				}
			})
		})

		ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "投递日期 *").FontSize(12).TextColor(tk.TextSecondary)
				prev := formatCalendarDate(w.editJobAppliedDate)
				dp := field.DatePicker(c.Key("detail-applied-date"), &w.editJobAppliedDate, field.DatePickerProps{
					Placeholder: "选择投递日期",
				})
				dp.Label("详情投递日期")
				if formatCalendarDate(w.editJobAppliedDate) != prev {
					w.TouchJobDetailField()
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "投递渠道").FontSize(12).TextColor(tk.TextSecondary)
				in := field.Input(c.Key("detail-channel"), &w.editJobChannel, field.InputProps{
					Placeholder: "官网 / 内推 / 招聘平台",
				})
				in.Label("详情投递渠道")
				if in.Changed() {
					w.TouchJobDetailField()
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "下次跟进日期").FontSize(12).TextColor(tk.TextSecondary)
				prev := formatCalendarDate(w.editJobFollowUpDate)
				dp := field.DatePicker(c.Key("detail-followup-date"), &w.editJobFollowUpDate, field.DatePickerProps{
					Placeholder: "未设置",
					AllowClear:  true,
				})
				dp.Label("详情下次跟进日期")
				if formatCalendarDate(w.editJobFollowUpDate) != prev {
					w.editJobFollowUpDirty = true
					w.TouchJobDetailField()
				}
			})
		})

		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "投递链接").FontSize(12).TextColor(tk.TextSecondary)
			ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					_, urlErr := model.ValidateJobURL(w.editJobURL)
					in := field.Input(c.Key("detail-url"), &w.editJobURL, field.InputProps{
						Placeholder: "https://...",
						Invalid:     urlErr != nil,
					})
					in.Label("详情投递链接")
					if in.Changed() {
						w.TouchJobDetailField()
					}
				})
				copyBtn := action.Button(c, "复制链接", action.ButtonProps{Variant: action.Default, Size: theme.Small})
				if strings.TrimSpace(w.editJobURL) == "" {
					copyBtn.Disabled(true)
				} else if copyBtn.Clicked() {
					c.Services().WriteClipboard(strings.TrimSpace(w.editJobURL))
					w.EnqueueToast(feedback.ToastProps{Title: "已复制投递链接", Type: feedback.ToastSuccess})
				}
				openBtn := action.Button(c, "打开链接", action.ButtonProps{Variant: action.Default, Size: theme.Small})
				if strings.TrimSpace(w.editJobURL) == "" {
					openBtn.Disabled(true)
				} else if openBtn.Clicked() {
					w.OpenJobURL(w.editJobURL)
				}
			})
		})

		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "通用备注").FontSize(12).TextColor(tk.TextSecondary)
			area := field.TextArea(c.Key("detail-notes"), &w.editJobNotes, field.TextAreaProps{
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
		detail := ev.Reason
		if detail != "" {
			detail = "原因：" + detail
		}
		return title, detail
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

func (w *Workbench) renderJobTimelineSection(c *ui.Context, tk *theme.Tokens, app model.JobApplication) {
	ui.Column(c).Gap(12).AlignSelf(ui.Stretch).BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Padding(16, 0, 0, 0).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "流程时间线").FontSize(13).FontWeight(600).TextColor(tk.Text)
			ui.Text(c, fmt.Sprintf("共 %d 条事件", len(w.detailJobEvents))).FontSize(12).TextColor(tk.TextTertiary)
		})

		// Event List
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			for _, ev := range w.detailJobEvents {
				headline, sub := formatTimelineSummary(ev, app)
				ui.Row(c.Key("ev-"+ev.ID)).Padding(8, 12).Gap(12).Radius(tk.BorderRadiusSM).
					Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, ev.OccurredAt.In(time.Local).Format("2006-01-02")).
						FontSize(12).TextColor(tk.TextTertiary).Shrink(0)
					ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
							ui.Box(c).MinWidth(0).Grow(1).Children(func() {
								ui.Text(c, headline).FontSize(13).FontWeight(500).TextColor(tk.Text)
							})
							if ev.EventType == model.JobEventStageBackfilled {
								ui.Box(c).Shrink(0).Children(func() {
									display.Tag(c, "补录", display.TagProps{Size: theme.Small, Variant: display.TagOutlined})
								})
							}
						})
						if sub != "" {
							ui.Text(c, sub).FontSize(12).TextColor(tk.TextSecondary).AlignSelf(ui.Stretch)
						}
					})
				})
			}
		})

		// Add Manual Note Form
		ui.Column(c).Padding(12).Gap(8).Radius(tk.BorderRadius).Background(tk.FillQuaternary.Alpha(0.5)).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "追加时间线备注").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
			if w.jobNoteErr != "" {
				feedback.Alert(c, w.jobNoteErr, feedback.AlertProps{Type: feedback.Error, Variant: feedback.Soft})
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					field.Input(c.Key("timeline-note-input"), &w.jobNoteInput, field.InputProps{
						Placeholder: "记录沟通结果或面试反馈...",
						Size:        theme.Small,
					}).Label("时间线备注输入")
				})
				ui.Box(c).Width(138).Shrink(0).Children(func() {
					field.DatePicker(c.Key("timeline-note-date"), &w.jobNoteDate, field.DatePickerProps{
						Size: theme.Small,
					}).Label("时间线备注日期")
				})
				if action.Button(c, "添加备注", action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
					w.AddJobTimelineNote(app.ID)
				}
			})
		})

		// Backfill Historical Stage Form (never overwrites current stage or lifecycle)
		ui.Column(c).Padding(12).Gap(8).Radius(tk.BorderRadius).Background(tk.FillQuaternary.Alpha(0.5)).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "补录历史阶段事件").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				ui.Text(c, "仅记录过去经历，不会更改当前阶段与流程状态").FontSize(11).TextColor(tk.TextTertiary)
			})
			if w.jobBackfillErr != "" {
				feedback.Alert(c, w.jobBackfillErr, feedback.AlertProps{Type: feedback.Error, Variant: feedback.Soft})
			}
			stageOpts := make([]field.SelectOption, 0, len(model.AllJobStages()))
			for _, st := range model.AllJobStages() {
				stageOpts = append(stageOpts, field.SelectOption{Value: string(st), Label: st.Label()})
			}
			ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Width(135).Children(func() {
					field.Select(c.Key("backfill-stage"), &w.jobBackfillStage, field.SelectProps{
						Options: stageOpts,
						Size:    theme.Small,
					}).Label("补录阶段")
				})
				ui.Box(c).Width(140).Children(func() {
					field.DatePicker(c.Key("backfill-date"), &w.jobBackfillDate, field.DatePickerProps{
						Size: theme.Small,
					}).Label("补录发生日期")
				})
				ui.Box(c).Grow(1).MinWidth(140).Children(func() {
					field.Input(c.Key("backfill-note"), &w.jobBackfillNote, field.InputProps{
						Placeholder: "可选说明...",
						Size:        theme.Small,
					}).Label("补录说明")
				})
				if action.Button(c, "补录历史事件", action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
					w.BackfillJobStageEvent(app.ID)
				}
			})
		})
	})
}

func (w *Workbench) renderJobCloseModal(c *ui.Context, tk *theme.Tokens) {
	if !w.jobCloseOpen {
		return
	}
	var currentStage model.JobStage
	if w.selectedJobID == w.jobCloseAppID && w.detailJob != nil {
		currentStage = w.detailJob.Stage
	} else {
		for _, a := range w.jobApps {
			if a.ID == w.jobCloseAppID {
				currentStage = a.Stage
				break
			}
		}
	}
	outcome := model.JobOutcome(w.jobCloseOutcome)
	needsOfferConfirm := outcome.RequiresOfferStage() && currentStage != model.JobStageOffer

	var dismissed bool
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Padding(32, 24).Background(tk.BgContainer.Alpha(0.6))
		panel.Width(480).MaxWidthPercent(100).Radius(12).Clip().
			Background(tk.BgElevated).Border(1, tk.BorderSecondary).TextColor(tk.Text).FontSize(tk.FontSize).
			Label("结束流程弹窗")
		tk.Elevate(panel, theme.BoxShadow)

		if panel.Dismissed() {
			dismissed = true
		}

		ui.Column(c).FillWidth().Padding(20, 24).Gap(14).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束求职流程").FontSize(17).FontWeight(700).TextColor(tk.Text)
				if action.ActionIcon(c, lucide.X(), action.ActionIconProps{Size: theme.Small, Title: "关闭结束流程弹窗"}).Clicked() {
					w.jobCloseOpen = false
				}
			})

			ui.Text(c, fmt.Sprintf("当前阶段为「%s」。结束流程会保留当前阶段并写入一条不可变历史事件。", currentStage.Label())).
				FontSize(12).TextColor(tk.TextSecondary)

			if w.jobCloseErr != "" {
				feedback.Alert(c, w.jobCloseErr, feedback.AlertProps{Type: feedback.Error, Variant: feedback.Soft})
			}

			ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束结果 *").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				opts := make([]field.SelectOption, 0, len(model.AllJobOutcomes()))
				for _, o := range model.AllJobOutcomes() {
					opts = append(opts, field.SelectOption{Value: string(o), Label: o.Label()})
				}
				field.Select(c.Key("close-outcome"), &w.jobCloseOutcome, field.SelectProps{
					Options: opts,
				}).Label("选择结束结果")
			})

			if needsOfferConfirm {
				feedback.Alert(c, "当前阶段尚未处于 Offer", feedback.AlertProps{
					Type:        feedback.Warning,
					Variant:     feedback.Soft,
					Description: fmt.Sprintf("选择「%s」要求阶段为 Offer，请勾选确认后随结果一并提交。", outcome.Label()),
				})
				choice.Checkbox(c, &w.jobCloseConfirmOffer, "确认同步将当前阶段调整为 Offer", choice.CheckboxProps{}).
					Label("确认同步将当前阶段调整为 Offer")
			}

			ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "结束原因（可选）").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				field.TextArea(c, &w.jobCloseReason, field.TextAreaProps{
					Placeholder: "记录失败复盘、放弃理由或薪资差异...",
					MinRows:     3,
					MaxRows:     6,
				}).Label("输入结束原因")
			})

			ui.Row(c).Gap(10).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
				if action.Button(c, "取消", action.ButtonProps{Variant: action.Default}).Clicked() {
					w.jobCloseOpen = false
				}
				confirmBtn := action.Button(c, "确认结束流程", action.ButtonProps{Variant: action.Primary})
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
