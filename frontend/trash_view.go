package frontend

import (
	"fmt"
	"strings"

	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var trashKindOptions = []field.SelectOption{
	{Value: "", Label: "全部类型"},
	{Value: projects.TrashProject, Label: "仅看项目"},
	{Value: projects.TrashTask, Label: "仅看任务"},
}

func (w *Workbench) filterTrashItems() []model.TrashItem {
	if len(w.trashItems) == 0 {
		return nil
	}
	query := strings.ToLower(strings.TrimSpace(w.trashSearch))
	if query == "" && w.trashKindFilter == "" {
		return w.trashItems
	}
	filtered := make([]model.TrashItem, 0, len(w.trashItems))
	for _, it := range w.trashItems {
		if w.trashKindFilter != "" && it.Kind != w.trashKindFilter {
			continue
		}
		if query != "" {
			projName := w.projectNames[it.ProjectID]
			if !strings.Contains(strings.ToLower(it.Name), query) && !strings.Contains(strings.ToLower(projName), query) {
				continue
			}
		}
		filtered = append(filtered, it)
	}
	return filtered
}

func (w *Workbench) renderTrashView(c *ui.Context, tk *theme.Tokens) {
	// Map deleted project IDs only when the recycle bin is non-empty
	var deletedProjects map[string]string
	if len(w.trashItems) > 0 {
		deletedProjects = make(map[string]string)
		for _, it := range w.trashItems {
			if it.Kind == projects.TrashProject {
				deletedProjects[it.ID] = it.Name
			}
		}
	}
	filtered := w.filterTrashItems()

	ui.Column(c).Fill().Children(func() {
		// Top Header Bar
		ui.Row(c).Padding(20, 24, 16, 24).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, lucide.Trash2()).FontSize(22).TextColor(tk.Error)
					ui.Text(c, "回收站").FontSize(20).FontWeight(700).TextColor(tk.Text).SingleLine()
					display.Tag(c, fmt.Sprintf("%d 项", len(w.trashItems)), display.TagProps{
						Size:    theme.Small,
						Variant: display.TagFilled,
					})
				})
				ui.Text(c, "已删除的项目与任务可在此恢复或彻底清除。").
					FontSize(13).TextColor(tk.TextSecondary).SingleLine()
			})

			if len(w.trashItems) > 0 {
				if action.Button(c, "清空回收站", action.ButtonProps{
					Variant: action.Primary,
					Danger:  true,
				}).Disabled(w.IsBusy()).Clicked() {
					w.confirmClearTrashOpen = true
				}
			}
		})

		// Filter Bar
		ui.Row(c).Padding(10, 24).Gap(12).AlignItems(ui.Center).AlignSelf(ui.Stretch).
			Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
			beforeSearch := w.trashSearch
			beforeKind := w.trashKindFilter

			ui.Box(c).Width(280).Children(func() {
				field.SearchBar(c, &w.trashSearch, field.SearchBarProps{
					Placeholder: "搜索已删除的项目或任务...",
					Size:        theme.Small,
				}).Label("搜索回收站")
			})

			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "类型:").FontSize(12).TextColor(tk.TextSecondary)
				ui.Box(c).Width(130).Children(func() {
					field.Select(c.Key("trash-kind-filter"), &w.trashKindFilter, field.SelectProps{
						Size:    theme.Small,
						Options: trashKindOptions,
					}).Label("回收站类型筛选")
				})
			})

			if w.trashSearch != beforeSearch || w.trashKindFilter != beforeKind {
				filtered = w.filterTrashItems()
			}

			ui.Box(c).Grow(1)

			hasTrashFilter := strings.TrimSpace(w.trashSearch) != "" || w.trashKindFilter != ""
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				if hasTrashFilter {
					ui.Text(c, fmt.Sprintf("匹配 %d / %d 项", len(filtered), len(w.trashItems))).
						FontSize(12).TextColor(tk.Primary).SingleLine()
					if action.Button(c, "清除筛选", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Clicked() {
						w.trashSearch = ""
						w.trashKindFilter = ""
						filtered = w.trashItems
					}
				} else {
					ui.Text(c, fmt.Sprintf("共 %d 项", len(w.trashItems))).
						FontSize(12).TextColor(tk.TextTertiary).SingleLine()
				}
			})
		})

		// Body Content
		ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Padding(24).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				if len(filtered) == 0 {
					ui.Box(c).Padding(48, 24).Radius(tk.BorderRadiusLG).Background(tk.BgContainer).
						Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
						emptyTitle := "回收站为空"
						emptyDesc := "当前没有任何已删除的项目或任务。"
						if len(w.trashItems) > 0 {
							emptyTitle = "没有匹配的回收站条目"
							emptyDesc = "请尝试调整搜索词或切换类型筛选条件。"
						}
						display.Empty(c, display.EmptyProps{
							Icon:        lucide.Trash2(),
							Title:       emptyTitle,
							Description: emptyDesc,
						})
					})
					return
				}

				for _, item := range filtered {
					it := item
					_, parentInTrash := deletedProjects[it.ProjectID]
					w.renderTrashRow(c.Key(it.Kind+"-"+it.ID), tk, it, parentInTrash)
				}
			})
		})
	})
}

func (w *Workbench) renderTrashRow(c *ui.Context, tk *theme.Tokens, item model.TrashItem, parentInTrash bool) {
	ui.Row(c).Padding(14, 18).Gap(16).Radius(tk.BorderRadius).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).
		AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
			iconSVG := lucide.Folder()
			tagText := "项目"
			tagColor := display.TagPurple
			if item.Kind == projects.TrashTask {
				iconSVG = lucide.CheckCheck()
				tagText = "任务"
				tagColor = display.TagBlue
			}

			ui.Box(c).Size(36, 36).Radius(tk.BorderRadiusSM).Background(tk.FillTertiary).Center().Shrink(0).Children(func() {
				ui.Icon(c, iconSVG).FontSize(18).TextColor(tk.TextSecondary)
			})

			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
					display.Tag(c, tagText, display.TagProps{
						Size:    theme.Small,
						Variant: display.TagFilled,
						Color:   tagColor,
					})
					ui.Text(c, item.Name).FontSize(15).FontWeight(600).TextColor(tk.Text).SingleLine()
					if item.Kind == projects.TrashTask && parentInTrash {
						display.Tag(c, "所属项目也在回收站中", display.TagProps{
							Size:    theme.Small,
							Variant: display.TagFilled,
							Color:   display.TagWarning,
						})
					}
				})

				metaParts := []string{"删除于 " + item.DeletedAt.Local().Format("2006-01-02 15:04:05")}
				projName := w.projectNames[item.ProjectID]
				if item.Kind == projects.TrashTask && projName != "" {
					metaParts = append([]string{"所属项目: " + projName}, metaParts...)
				} else if item.Kind == projects.TrashProject {
					metaParts = append([]string{"恢复项目将同时恢复随该项目一起删除的任务"}, metaParts...)
				}
				ui.Text(c, strings.Join(metaParts, " · ")).FontSize(12).TextColor(tk.TextTertiary).SingleLine()
			})
		})

		// Action Buttons
		ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			if action.Button(c.Key("restore-"+item.ID), "恢复", action.ButtonProps{
				Variant: action.Default,
				Size:    theme.Small,
			}).Disabled(w.IsBusy()).Label("恢复 " + item.Name).Clicked() {
				w.RestoreTrashItem(item)
			}

			if action.Button(c.Key("perm-del-"+item.ID), "彻底删除", action.ButtonProps{
				Variant: action.Default,
				Size:    theme.Small,
				Danger:  true,
			}).Disabled(w.IsBusy()).Label("彻底删除 " + item.Name).Clicked() {
				itemCopy := item
				w.confirmPermDeleteItem = &itemCopy
				w.confirmPermDeleteOpen = true
			}
		})
	})
}

func (w *Workbench) renderTrashModals(c *ui.Context, tk *theme.Tokens) {
	// 1. Parent Project in Trash Warning Modal (ErrParentDeleted)
	if feedback.Modal(c, &w.trashParentErrModalOpen, feedback.ModalProps{
		Title:      "无法单独恢复任务：所属项目仍在回收站中",
		Width:      480,
		OkText:     "一并恢复项目与任务",
		CancelText: "暂不恢复",
	}, func() {
		taskTitle := ""
		projTitle := ""
		if w.trashOrphanTask != nil {
			taskTitle = w.trashOrphanTask.Name
			projTitle = w.projectNames[w.trashOrphanTask.ProjectID]
		}
		if projTitle == "" && w.trashOrphanParent != nil {
			projTitle = w.trashOrphanParent.Name
		}
		ui.Column(c).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			feedback.Alert(c, "所属项目处于已删除状态", feedback.AlertProps{
				Type:        feedback.Warning,
				Variant:     feedback.Soft,
				Description: fmt.Sprintf("任务「%s」所属的项目「%s」仍在回收站中，请先恢复所属项目，才能恢复该任务。", taskTitle, projTitle),
			})
			ui.Text(c, "点击“一并恢复项目与任务”将立即恢复该项目及其关联任务。").
				FontSize(13).TextColor(tk.TextSecondary)
		})
	}) {
		w.RestoreOrphanTaskWithParent()
	}

	// 2. Permanent Delete Confirmation Modal
	if feedback.Modal(c, &w.confirmPermDeleteOpen, feedback.ModalProps{
		Title:      "彻底删除条目",
		Width:      460,
		OkText:     "确认彻底删除",
		CancelText: "取消",
	}, func() {
		desc := "即将永久删除所选条目及其本地附件，此操作无法恢复。"
		if w.confirmPermDeleteItem != nil {
			kindLabel := "任务"
			if w.confirmPermDeleteItem.Kind == projects.TrashProject {
				kindLabel = "项目及其下属所有任务"
			}
			desc = fmt.Sprintf("即将永久删除%s「%s」及其本地附件，此操作无法恢复。", kindLabel, w.confirmPermDeleteItem.Name)
		}
		ui.Column(c).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			feedback.Alert(c, "此操作不可撤销", feedback.AlertProps{
				Type:        feedback.Error,
				Variant:     feedback.Soft,
				Description: desc,
			})
		})
	}) {
		w.ConfirmPermanentDelete()
	}

	// 3. Clear Trash Confirmation Modal
	if feedback.Modal(c, &w.confirmClearTrashOpen, feedback.ModalProps{
		Title:      "清空回收站确认",
		Width:      460,
		OkText:     "确认清空回收站",
		CancelText: "取消",
	}, func() {
		ui.Column(c).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			feedback.Alert(c, "清空后无法恢复", feedback.AlertProps{
				Type:        feedback.Error,
				Variant:     feedback.Soft,
				Description: fmt.Sprintf("即将永久删除回收站中的全部 %d 项内容（包括项目、任务及其关联附件）。", len(w.trashItems)),
			})
		})
	}) {
		w.ConfirmClearTrash()
	}
}
