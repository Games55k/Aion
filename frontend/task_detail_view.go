package frontend

import (
	"fmt"
	"slices"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/markdown"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var markdownTools = []struct {
	Key     string
	Label   string
	Snippet string
}{
	{Key: "h1", Label: "H1", Snippet: "# 一级标题\n"},
	{Key: "h2", Label: "H2", Snippet: "## 二级标题\n"},
	{Key: "bold", Label: "粗体", Snippet: "**粗体文本**"},
	{Key: "italic", Label: "斜体", Snippet: "*斜体文本*"},
	{Key: "list", Label: "列表", Snippet: "- 列表项\n"},
	{Key: "todo", Label: "待办", Snippet: "- [ ] 待办事项\n"},
	{Key: "code", Label: "代码", Snippet: "`inline_code`"},
	{Key: "codeblock", Label: "代码块", Snippet: "```go\n// 代码示例\n```\n"},
	{Key: "quote", Label: "引用", Snippet: "> 引用说明\n"},
	{Key: "link", Label: "链接", Snippet: "[链接文字](https://example.com)"},
}

func (w *Workbench) renderTaskDetailPanel(c *ui.Context, tk *theme.Tokens) {
	if w.detailTask == nil {
		return
	}
	t := *w.detailTask

	panelRoot := ui.Column(c).Fill().Background(tk.BgContainer).BorderWidth(0, 0, 0, 1).BorderColor(tk.BorderSecondary).
		Label("任务详情面板")

	// Allow dropping files anywhere on the task detail panel or its attachment zone
	if dropped := panelRoot.DroppedFiles(); len(dropped) > 0 {
		w.ImportAttachmentsToSelectedTask(dropped)
	}

	panelRoot.Children(func() {
		// 1. Top Header Bar: Save status + Delete + Close
		ui.Row(c).Padding(12, 16).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			// Save status indicator
			ui.Row(c).Gap(6).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
				switch w.saveState {
				case "validation_error":
					ui.Icon(c, lucide.TriangleAlert()).FontSize(14).TextColor(tk.Warning)
					ui.Text(c, w.saveErr).FontSize(12).TextColor(tk.Warning).SingleLine()
					if action.Button(c, "放弃修改", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Clicked() {
						w.DiscardSelectedTaskDraft()
					}
				case "error":
					ui.Icon(c, lucide.CircleX()).FontSize(14).TextColor(tk.Error)
					ui.Text(c, "保存失败: "+w.saveErr).FontSize(12).TextColor(tk.Error).SingleLine()
					if action.Button(c, "重试保存", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Clicked() {
						_ = w.SaveSelectedTask()
					}
					if action.Button(c, "放弃修改", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Clicked() {
						w.DiscardSelectedTaskDraft()
					}
				default:
					ui.Icon(c, lucide.CircleCheck()).FontSize(14).TextColor(tk.Success)
					saveText := "已自动保存"
					if !w.lastSavedAt.IsZero() {
						saveText = "已自动保存 · " + w.lastSavedAt.Local().Format("15:04:05")
					}
					ui.Text(c, saveText).FontSize(12).TextColor(tk.TextSecondary).SingleLine()
				}
			})

			// Delete & Close Buttons
			ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				if action.Button(c, "移入回收站", action.ButtonProps{
					Variant: action.Default,
					Size:    theme.Small,
					Danger:  true,
				}).Label("将任务移入回收站").Clicked() {
					w.SoftDeleteTask(t.ID)
					return
				}
				if action.ActionIcon(c, lucide.X(), action.ActionIconProps{
					Size:  theme.Small,
					Title: "关闭详情面板",
				}).Clicked() {
					w.ClearSelectedTask()
				}
			})
		})
		if w.detailTask == nil {
			return
		}

		// 2. Scrollable Detail Body
		ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Padding(16, 18, 24, 18).Gap(18).AlignSelf(ui.Stretch).Children(func() {
				// Background Operation Banner
				if w.IsBusy() {
					ui.Row(c.Key("detail-busy-banner")).Padding(10, 12).Gap(10).Radius(tk.BorderRadius).
						Background(tk.PrimaryFillQuaternary).Border(1, tk.Primary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, w.BusyLabel()).FontSize(12).TextColor(tk.Text).Grow(1).SingleLine()
						if action.Button(c, "取消操作", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Clicked() {
							w.CancelBusyTask()
						}
					})
				}

				// Validation Alert Banner if title is empty
				if w.saveState == "validation_error" {
					feedback.Alert(c.Key("detail-validation-alert"), "任务标题不能为空，请输入有效标题以完成自动保存，或点击顶部“放弃修改”恢复原内容", feedback.AlertProps{
						Type:    feedback.Warning,
						Variant: feedback.Soft,
					})
				}

				// Section A: Task Title
				ui.Column(c.Key("detail-sec-title")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "任务标题").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
					beforeTitle := w.editTitle
					field.Input(c.Key("detail-title-"+t.ID), &w.editTitle, field.InputProps{
						Placeholder: "输入任务标题（必填）",
						Size:        theme.Large,
						Invalid:     strings.TrimSpace(w.editTitle) == "",
					}).Label("任务详情标题")
					if w.editTitle != beforeTitle {
						w.dirty = true
						_ = w.SaveSelectedTask()
					}
				})

				// Section B: Status, Priority, Due Date
				ui.Row(c.Key("detail-sec-meta")).Gap(10).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
					// Status
					ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
						ui.Text(c, "状态").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
						beforeStatus := w.editStatus
						field.Select(c.Key("detail-status-"+t.ID), &w.editStatus, field.SelectProps{
							Size:    theme.Small,
							Options: taskStatusOptions,
						}).Label("任务详情状态")
						if w.editStatus != beforeStatus {
							w.dirty = true
							_ = w.SaveSelectedTask()
						}
					})

					// Priority
					ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
						ui.Text(c, "优先级").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
						beforePriority := w.editPriority
						field.Select(c.Key("detail-priority-"+t.ID), &w.editPriority, field.SelectProps{
							Size:    theme.Small,
							Options: taskPriorityOptions,
						}).Label("任务详情优先级")
						if w.editPriority != beforePriority {
							w.dirty = true
							_ = w.SaveSelectedTask()
						}
					})

					// Due Date
					ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
						ui.Text(c, "截止日期").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
						beforeDue := w.editDueDate
						field.DatePicker(c.Key("detail-due-"+t.ID), &w.editDueDate, field.DatePickerProps{
							Placeholder: "未设置截止日期",
							AllowClear:  true,
							Size:        theme.Small,
						}).Label("任务详情截止日期")
						if !w.editDueDate.Equal(beforeDue) {
							w.dirty = true
							_ = w.SaveSelectedTask()
						}
					})
				})

				// Section C: Tags
				ui.Column(c.Key("detail-sec-tags")).Gap(8).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "分类标签").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
					ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
						for i, tag := range w.editTags {
							idx := i
							tagVal := tag
							ui.Row(c.Key("edit-tag-"+tagVal)).Gap(4).Padding(2, 8).Radius(tk.BorderRadiusSM).
								Background(tk.FillTertiary).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "#"+tagVal).FontSize(12).TextColor(tk.Text).SingleLine()
								rmBtn := ui.ButtonBase(c).Size(14, 14).Radius(7).Center().
									Cursor(ui.CursorPointer).TextColor(tk.TextTertiary).
									Label("移除标签 " + tagVal)
								rmBtn.Children(func() { ui.Icon(c, lucide.X()).FontSize(11) })
								if rmBtn.Clicked() {
									w.editTags = slices.Delete(w.editTags, idx, idx+1)
									_ = w.SaveSelectedTask()
								}
							})
						}

						// Add Tag Input + Button
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Box(c).Width(150).Children(func() {
								inp := field.Input(c.Key("new-tag-"+t.ID), &w.newTagInput, field.InputProps{
									Placeholder: "+ 添加新标签...",
									Size:        theme.Small,
								}).Label("添加标签输入框")
								if inp.Submitted() {
									w.addTagFromInput()
								}
							})
							if strings.TrimSpace(w.newTagInput) != "" {
								if action.Button(c, "添加标签", action.ButtonProps{
									Variant: action.Default,
									Size:    theme.Small,
								}).Clicked() {
									w.addTagFromInput()
								}
							}
						})
					})
				})

				// Section D: Markdown Description Editor & Preview
				w.renderMarkdownSection(c, tk, t)

				// Section E: Subtask List
				w.renderSubtasksSection(c, tk, t)

				// Section F: Local Attachments
				w.renderAttachmentsSection(c, tk, t)

				// Section G: Metadata Timestamps
				ui.Row(c.Key("detail-sec-timestamps")).Padding(12, 0, 0, 0).Justify(ui.SpaceBetween).AlignItems(ui.Center).
					AlignSelf(ui.Stretch).BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
					ui.Text(c, "创建于 "+t.CreatedAt.Local().Format("2006-01-02 15:04:05")).
						FontSize(11).TextColor(tk.TextTertiary).SingleLine()
					ui.Text(c, "更新于 "+t.UpdatedAt.Local().Format("2006-01-02 15:04:05")).
						FontSize(11).TextColor(tk.TextTertiary).SingleLine()
				})
			})
		})
	})
}

func (w *Workbench) addTagFromInput() {
	tag := strings.TrimSpace(w.newTagInput)
	if tag == "" {
		return
	}
	if !slices.Contains(w.editTags, tag) {
		w.editTags = append(w.editTags, tag)
		_ = w.SaveSelectedTask()
	}
	w.newTagInput = ""
}

func (w *Workbench) appendMarkdownSnippet(snippet string) {
	if w.editDescription != "" && !strings.HasSuffix(w.editDescription, "\n") {
		w.editDescription += "\n"
	}
	w.editDescription += snippet
	_ = w.SaveSelectedTask()
}

func (w *Workbench) renderMarkdownSection(c *ui.Context, tk *theme.Tokens, t model.Task) {
	ui.Column(c.Key("detail-sec-markdown")).Gap(8).AlignSelf(ui.Stretch).Children(func() {
		// Header + Edit/Preview/Split Segmented Switcher
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, lucide.FileText()).FontSize(14).TextColor(tk.TextSecondary)
				ui.Text(c, "任务描述 (Markdown)").FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
			})
			action.Segmented(c.Key("md-mode-"+t.ID), &w.mdMode, []string{"编辑", "预览", "分栏"}, action.SegmentedProps{
				Size: theme.Small,
			}).Label("Markdown 模式切换")
		})

		// Formatting Toolbar (shown when editing or split)
		if w.mdMode == 0 || w.mdMode == 2 {
			ui.Row(c.Key("md-toolbar")).Gap(4).Wrap().Padding(4, 6).Radius(tk.BorderRadiusSM).
				Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
				for _, tool := range markdownTools {
					btn := ui.ButtonBase(c.Key("md-tool-" + tool.Key)).Height(22).PaddingX(6).Radius(4).
						FontSize(11).TextColor(tk.TextSecondary).Cursor(ui.CursorPointer).
						Label("格式 " + tool.Label)
					if btn.Hovered() {
						btn.Background(tk.FillSecondary).TextColor(tk.Text)
					}
					btn.Children(func() { ui.Text(c, tool.Label).SingleLine() })
					if btn.Clicked() {
						w.appendMarkdownSnippet(tool.Snippet)
					}
				}
			})
		}

		// Editor / Preview Content
		switch w.mdMode {
		case 0: // 编辑
			ui.Box(c.Key("md-edit-wrap")).AlignSelf(ui.Stretch).Children(func() {
				beforeDesc := w.editDescription
				field.TextArea(c.Key("detail-desc-"+t.ID), &w.editDescription, field.TextAreaProps{
					Placeholder: "输入任务详细说明，支持 Markdown 标题、列表、代码块、引用、链接...",
					MinRows:     5,
					MaxRows:     12,
				}).Label("任务详情描述输入框")
				if w.editDescription != beforeDesc {
					_ = w.SaveSelectedTask()
				}
			})

		case 1: // 预览
			ui.Box(c.Key("md-preview-wrap")).Padding(12, 14).Radius(tk.BorderRadius).Background(tk.FillQuaternary).
				Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
				if strings.TrimSpace(w.editDescription) == "" {
					ui.Text(c, "暂无任务描述，切换到“编辑”模式可输入 Markdown 内容。").
						FontSize(13).TextColor(tk.TextTertiary)
				} else {
					markdown.Markdown(c.Key("md-preview-"+t.ID), w.editDescription, markdown.MarkdownProps{
						Variant:               markdown.Chat,
						FullFeaturedCodeBlock: true,
					})
				}
			})

		case 2: // 分栏
			ui.Row(c.Key("md-split-wrap")).Gap(10).AlignItems(ui.Stretch).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
					beforeDesc := w.editDescription
					field.TextArea(c.Key("detail-desc-split-"+t.ID), &w.editDescription, field.TextAreaProps{
						Placeholder: "输入 Markdown 内容...",
						MinRows:     6,
						MaxRows:     12,
					}).Label("任务详情描述输入框")
					if w.editDescription != beforeDesc {
						_ = w.SaveSelectedTask()
					}
				})
				ui.Box(c).Grow(1).Basis(0).MinWidth(0).Padding(10, 12).Radius(tk.BorderRadius).
					Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).Children(func() {
					if strings.TrimSpace(w.editDescription) == "" {
						ui.Text(c, "实时预览区域").FontSize(12).TextColor(tk.TextTertiary)
					} else {
						markdown.Markdown(c.Key("md-split-preview-"+t.ID), w.editDescription, markdown.MarkdownProps{
							Variant:               markdown.Chat,
							FullFeaturedCodeBlock: true,
						})
					}
				})
			})
		}
	})
}

func (w *Workbench) renderSubtasksSection(c *ui.Context, tk *theme.Tokens, t model.Task) {
	doneCount := 0
	for _, st := range w.editSubtasks {
		if st.Done {
			doneCount++
		}
	}
	totalCount := len(w.editSubtasks)
	pct := float32(0)
	if totalCount > 0 {
		pct = float32(doneCount) * 100 / float32(totalCount)
	}

	ui.Column(c.Key("detail-sec-subtasks")).Gap(8).AlignSelf(ui.Stretch).Children(func() {
		// Header Row
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, lucide.SquareCheckBig()).FontSize(14).TextColor(tk.TextSecondary)
				ui.Text(c, fmt.Sprintf("子任务 (%d/%d)", doneCount, totalCount)).
					FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
			})
			if totalCount > 0 && doneCount == totalCount {
				display.Tag(c, "子任务已全部完成", display.TagProps{
					Size:    theme.Small,
					Variant: display.TagFilled,
					Color:   display.TagGreen,
				})
			}
		})

		// Progress Bar
		if totalCount > 0 {
			feedback.Progress(c, pct, feedback.ProgressProps{
				Size:     theme.Small,
				ShowInfo: true,
				Format:   formatProgressPercent,
			})
		}

		// Subtask Rows
		if totalCount > 0 {
			ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				for i, st := range w.editSubtasks {
					idx := i
					sub := st
					keyID := sub.ID
					if keyID == "" {
						keyID = fmt.Sprintf("idx-%d", idx)
					}
					ui.Row(c.Key("subtask-"+keyID)).Padding(6, 10).Gap(8).Radius(tk.BorderRadiusSM).
						Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
						AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
						// Checkbox
						done := sub.Done
						choice.Checkbox(c.Key("cb-"+keyID), &done, "", choice.CheckboxProps{}).
							Label("勾选子任务 " + sub.Title)
						if done != sub.Done {
							w.editSubtasks[idx].Done = done
							// Strictly per PRD 6.5: checking all subtasks does NOT auto-complete parent task
							_ = w.SaveSelectedTask()
						}

						if w.editingSubtaskID != "" && w.editingSubtaskID == keyID {
							saveEdited := func() {
								trimmed := strings.TrimSpace(w.editingSubtaskTitle)
								if trimmed == "" {
									return
								}
								w.editSubtasks[idx].Title = trimmed
								w.editingSubtaskID = ""
								w.editingSubtaskTitle = ""
								_ = w.SaveSelectedTask()
							}
							ui.Box(c.Key("edit-sub-wrap-" + keyID)).Grow(1).MinWidth(0).Children(func() {
								inp := field.Input(c.Key("edit-sub-inp-"+keyID), &w.editingSubtaskTitle, field.InputProps{
									Size:    theme.Small,
									Invalid: strings.TrimSpace(w.editingSubtaskTitle) == "",
								}).Label("编辑子任务输入框")
								if inp.Submitted() {
									saveEdited()
								}
							})
							if action.Button(c.Key("save-sub-"+keyID), "保存", action.ButtonProps{
								Variant: action.Primary,
								Size:    theme.Small,
							}).Disabled(strings.TrimSpace(w.editingSubtaskTitle) == "").Label("保存子任务 " + sub.Title).Clicked() {
								saveEdited()
							}
							if action.Button(c.Key("cancel-sub-"+keyID), "取消", action.ButtonProps{
								Variant: action.Default,
								Size:    theme.Small,
							}).Clicked() {
								w.editingSubtaskID = ""
								w.editingSubtaskTitle = ""
							}
						} else {
							txt := ui.Text(c, sub.Title).FontSize(13).Grow(1)
							if sub.Done {
								txt.TextColor(tk.TextTertiary)
							} else {
								txt.TextColor(tk.Text)
							}

							// Subtask Actions: Edit, Up, Down, Delete
							ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
								if action.ActionIcon(c.Key("edit-btn-"+keyID), lucide.Pencil(), action.ActionIconProps{
									Size:  theme.Small,
									Title: "编辑子任务 " + sub.Title,
								}).Clicked() {
									w.editingSubtaskID = keyID
									w.editingSubtaskTitle = sub.Title
								}
								if idx > 0 {
									if action.ActionIcon(c.Key("up-btn-"+keyID), lucide.ArrowUp(), action.ActionIconProps{
										Size:  theme.Small,
										Title: "上移子任务 " + sub.Title,
									}).Clicked() {
										w.editSubtasks[idx-1], w.editSubtasks[idx] = w.editSubtasks[idx], w.editSubtasks[idx-1]
										_ = w.SaveSelectedTask()
									}
								}
								if idx < totalCount-1 {
									if action.ActionIcon(c.Key("down-btn-"+keyID), lucide.ArrowDown(), action.ActionIconProps{
										Size:  theme.Small,
										Title: "下移子任务 " + sub.Title,
									}).Clicked() {
										w.editSubtasks[idx+1], w.editSubtasks[idx] = w.editSubtasks[idx], w.editSubtasks[idx+1]
										_ = w.SaveSelectedTask()
									}
								}
								if action.ActionIcon(c.Key("del-btn-"+keyID), lucide.Trash2(), action.ActionIconProps{
									Size:   theme.Small,
									Danger: true,
									Title:  "删除子任务 " + sub.Title,
								}).Clicked() {
									w.editSubtasks = slices.Delete(w.editSubtasks, idx, idx+1)
									_ = w.SaveSelectedTask()
								}
							})
						}
					})
				}
			})
		}

		// Add New Subtask Row
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			var submitted bool
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				inp := field.Input(c.Key("new-subtask-"+t.ID), &w.newSubtaskTitle, field.InputProps{
					Placeholder: "+ 输入新子任务标题...",
					Size:        theme.Small,
				}).Label("新子任务输入框")
				submitted = inp.Submitted()
			})
			addSubtask := func() {
				title := strings.TrimSpace(w.newSubtaskTitle)
				if title == "" {
					return
				}
				w.editSubtasks = append(w.editSubtasks, model.Subtask{
					ID:    "sub-" + model.NewID(),
					Title: title,
					Done:  false,
				})
				w.newSubtaskTitle = ""
				_ = w.SaveSelectedTask()
			}
			if submitted {
				addSubtask()
			}
			if action.Button(c, "添加子任务", action.ButtonProps{
				Variant: action.Default,
				Size:    theme.Small,
			}).Disabled(strings.TrimSpace(w.newSubtaskTitle) == "").Clicked() {
				addSubtask()
			}
		})
	})
}

func (w *Workbench) renderAttachmentsSection(c *ui.Context, tk *theme.Tokens, _ model.Task) {
	atts := w.detailAttachments

	ui.Column(c.Key("detail-sec-attachments")).Gap(10).AlignSelf(ui.Stretch).Children(func() {
		// Header Row
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, lucide.Paperclip()).FontSize(14).TextColor(tk.TextSecondary)
				ui.Text(c, fmt.Sprintf("附件 (%d)", len(atts))).
					FontSize(12).FontWeight(600).TextColor(tk.TextTertiary)
			})

			if action.Button(c, "+ 导入附件", action.ButtonProps{
				Variant: action.Default,
				Size:    theme.Small,
			}).Disabled(w.IsBusy()).Clicked() {
				w.PickAndImportAttachmentsToSelectedTask()
			}
		})

		// File Drag-and-Drop Zone
		dropzone := ui.Column(c).Padding(12).Gap(4).Radius(tk.BorderRadius).
			Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).BorderStyle(ui.BorderDashed).
			AlignItems(ui.Center).AlignSelf(ui.Stretch).Label("附件拖拽上传区")
		if dropzone.FileDragOver() {
			dropzone.Border(2, tk.Primary).Background(tk.PrimaryFillQuaternary)
		}
		if dropped := dropzone.DroppedFiles(); len(dropped) > 0 {
			w.ImportAttachmentsToSelectedTask(dropped)
		}
		dropzone.Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, lucide.Upload()).FontSize(14).TextColor(tk.TextTertiary)
				ui.Text(c, "将文件或图片拖拽到此处添加附件（单文件 ≤ 256 MB）").
					FontSize(12).TextColor(tk.TextTertiary)
			})
		})

		if len(atts) == 0 {
			return
		}

		// Image Thumbnails Preview Grid (with built-in Zoom / Rotate / Lightbox via display.ImageGroup)
		hasImages := false
		for _, att := range atts {
			if IsImageAttachment(att) {
				hasImages = true
				break
			}
		}
		if hasImages {
			display.ImageGroup(c, func(gallery *display.ImageGallery) {
				for _, att := range atts {
					if !IsImageAttachment(att) {
						continue
					}
					thumb := ui.Column(c.Key("img-thumb-" + att.ID)).Gap(4).Width(116)
					thumb.Children(func() {
						bmp := w.bitmapCache[att.ID]
						inView := w.host.Dispatch == nil || thumb.InViewMargin(100)
						// Decode only images near the viewport, so scrolling through
						// a long attachment list does not churn the bounded cache.
						if inView {
							bmp = w.AttachmentBitmap(att)
						} else if bmp == nil && !w.bitmapFailed[att.ID] && thumb.Bounds().W == 0 {
							// Trigger one post-layout pass so newly mounted thumbnails inside the viewport are detected.
							c.Invalidate()
						}
						if bmp == nil && !w.bitmapFailed[att.ID] && !inView {
							display.Skeleton(c, display.SkeletonProps{
								Width:     116,
								Height:    84,
								Animation: display.SkeletonStatic,
							}).Label("图片预览 " + att.Name)
						} else {
							display.Image(c, bmp, display.ImageProps{
								Width:   116,
								Height:  84,
								Gallery: gallery,
								Alt:     att.Name,
								Loading: bmp == nil && !w.bitmapFailed[att.ID],
								Failed:  w.bitmapFailed[att.ID],
							}).Label("图片预览 " + att.Name)
						}
						ui.Text(c, att.Name).FontSize(11).TextColor(tk.TextSecondary).SingleLine()
					})
				}
			}).Row().Wrap().Gap(10).AlignSelf(ui.Stretch)
		}

		// Attachment List Rows with Open / Finder / Export / Remove actions
		ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			for _, att := range atts {
				ui.Row(c.Key("att-row-"+att.ID)).Padding(8, 10).Gap(10).Radius(tk.BorderRadiusSM).
					Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
					AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
						iconSVG := lucide.File()
						if IsImageAttachment(att) {
							iconSVG = lucide.Image()
						}
						ui.Icon(c, iconSVG).FontSize(16).TextColor(tk.Primary).Shrink(0)
						ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, att.Name).FontSize(13).FontWeight(500).TextColor(tk.Text).SingleLine()
							ui.Text(c, fmt.Sprintf("%s · %s", formatByteSize(att.Size), att.MediaType)).
								FontSize(11).TextColor(tk.TextTertiary).SingleLine()
						})
					})

					// Attachment Action Buttons
					ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
						if action.Button(c.Key("open-att-"+att.ID), "打开", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Label("打开附件 " + att.Name).Clicked() {
							w.openOrRevealAttachment("打开附件失败", w.host.OpenPath, att)
						}

						if action.Button(c.Key("reveal-att-"+att.ID), "Finder", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Label("在 Finder 中显示 " + att.Name).Clicked() {
							w.openOrRevealAttachment("在 Finder 中显示失败", w.host.RevealPath, att)
						}

						if action.Button(c.Key("export-att-"+att.ID), "另存为", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Disabled(w.IsBusy()).Label("导出附件 " + att.Name).Clicked() {
							w.PickAndExportAttachment(att)
						}

						if action.ActionIcon(c.Key("rm-att-"+att.ID), lucide.Trash2(), action.ActionIconProps{
							Size:   theme.Small,
							Danger: true,
							Title:  "移除附件 " + att.Name,
						}).Clicked() {
							w.RemoveAttachmentFromSelectedTask(att.ID)
						}
					})
				})
			}
		})
	})
}

func (w *Workbench) openOrRevealAttachment(failTitle string, actionFn func(string) error, att model.Attachment) {
	if actionFn == nil {
		return
	}
	managedPath := w.AttachmentManagedPath(att)
	if managedPath == "" {
		w.EnqueueToast(feedback.ToastProps{
			Title:       failTitle,
			Description: "附件文件不存在或路径无效",
			Type:        feedback.ToastError,
		})
		return
	}
	if err := actionFn(managedPath); err != nil {
		w.EnqueueToast(feedback.ToastProps{
			Title:       failTitle,
			Description: err.Error(),
			Type:        feedback.ToastError,
		})
	}
}

func formatByteSize(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
