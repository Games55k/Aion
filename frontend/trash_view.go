package frontend

import (
	"fmt"
	"strings"

	"aion/backend/jobapplications"
	"aion/backend/model"
	"aion/backend/projects"

	"github.com/egoist/mygo/ui"
)

var trashKindOptions = []selectOption{
	{Value: "", Label: "全部类型"},
	{Value: projects.TrashProject, Label: "仅看项目"},
	{Value: projects.TrashTask, Label: "仅看任务"},
	{Value: jobapplications.TrashKind, Label: "仅看求职投递"},
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

func (w *Workbench) renderTrashView(c *ui.Context, tk *uiTokens) {
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
					ui.Icon(c, iconTrash2()).FontSize(22).TextColor(tk.Error)
					ui.Text(c, "回收站").FontSize(20).FontWeight(700).TextColor(tk.Text).SingleLine()
					uiTag(c, fmt.Sprintf("%d 项", len(w.trashItems)), tagProps{
						Size:    sizeSmall,
						Variant: tagFilled,
					})
				})
				ui.Text(c, "已删除的项目、任务与投递记录可在此恢复或彻底清除。").
					FontSize(13).TextColor(tk.TextSecondary).SingleLine()
			})

			if len(w.trashItems) > 0 {
				if uiButton(c, "清空回收站", buttonProps{
					Variant: buttonPrimary,
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
				uiSearchBar(c, &w.trashSearch, searchBarProps{
					Placeholder: "搜索已删除的项目、任务或投递...",
					Size:        sizeSmall,
				}).Label("搜索回收站")
			})

			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "类型:").FontSize(12).TextColor(tk.TextSecondary)
				ui.Box(c).Width(130).Children(func() {
					uiSelect(c.Key("trash-kind-filter"), &w.trashKindFilter, selectProps{
						Size:    sizeSmall,
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
					if uiButton(c, "清除筛选", buttonProps{
						Variant: buttonDefault,
						Size:    sizeSmall,
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
						emptyDesc := "当前没有任何已删除的项目、任务或投递记录。"
						if len(w.trashItems) > 0 {
							emptyTitle = "没有匹配的回收站条目"
							emptyDesc = "请尝试调整搜索词或切换类型筛选条件。"
						}
						uiEmpty(c, emptyProps{
							Icon:        iconTrash2(),
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

func (w *Workbench) renderTrashRow(c *ui.Context, tk *uiTokens, item model.TrashItem, parentInTrash bool) {
	ui.Row(c).Padding(14, 18).Gap(16).Radius(tk.BorderRadius).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).
		AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
			iconSVG := iconFolder()
			tagText := "项目"
			tagColor := tagPurple
			switch item.Kind {
			case projects.TrashTask:
				iconSVG = iconCheckCheck()
				tagText = "任务"
				tagColor = tagBlue
			case jobapplications.TrashKind:
				iconSVG = iconBriefcase()
				tagText = "求职投递"
				tagColor = tagCyan
			}

			ui.Box(c).Size(36, 36).Radius(tk.BorderRadiusSM).Background(tk.FillTertiary).Center().Shrink(0).Children(func() {
				ui.Icon(c, iconSVG).FontSize(18).TextColor(tk.TextSecondary)
			})

			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
					uiTag(c, tagText, tagProps{
						Size:    sizeSmall,
						Variant: tagFilled,
						Color:   tagColor,
					})
					ui.Text(c, item.Name).FontSize(15).FontWeight(600).TextColor(tk.Text).SingleLine()
					if item.Kind == projects.TrashTask && parentInTrash {
						uiTag(c, "所属项目也在回收站中", tagProps{
							Size:    sizeSmall,
							Variant: tagFilled,
							Color:   tagWarning,
						})
					}
				})

				metaParts := []string{"删除于 " + item.DeletedAt.Local().Format("2006-01-02 15:04:05")}
				projName := w.projectNames[item.ProjectID]
				if item.Kind == projects.TrashTask && projName != "" {
					metaParts = append([]string{"所属项目: " + projName}, metaParts...)
				} else if item.Kind == projects.TrashProject {
					metaParts = append([]string{"恢复项目将同时恢复随该项目一起删除的任务"}, metaParts...)
				} else if item.Kind == jobapplications.TrashKind {
					metaParts = append([]string{"恢复后将保留该投递的完整时间线历史"}, metaParts...)
				}
				ui.Text(c, strings.Join(metaParts, " · ")).FontSize(12).TextColor(tk.TextTertiary).SingleLine()
			})
		})

		// Action Buttons
		ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			if uiButton(c.Key("restore-"+item.ID), "恢复", buttonProps{
				Variant: buttonDefault,
				Size:    sizeSmall,
			}).Disabled(w.IsBusy()).Label("恢复 " + item.Name).Clicked() {
				w.RestoreTrashItem(item)
			}

			if uiButton(c.Key("perm-del-"+item.ID), "彻底删除", buttonProps{
				Variant: buttonDefault,
				Size:    sizeSmall,
				Danger:  true,
			}).Disabled(w.IsBusy()).Label("彻底删除 " + item.Name).Clicked() {
				itemCopy := item
				w.confirmPermDeleteItem = &itemCopy
				w.confirmPermDeleteOpen = true
			}
		})
	})
}

func (w *Workbench) renderTrashModals(c *ui.Context, tk *uiTokens) {
	// 1. Parent Project in Trash Warning Modal (ErrParentDeleted)
	if uiModal(c, &w.trashParentErrModalOpen, modalProps{
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
			uiAlert(c, "所属项目处于已删除状态", alertProps{
				Type:        alertWarning,
				Description: fmt.Sprintf("任务「%s」所属的项目「%s」仍在回收站中，请先恢复所属项目，才能恢复该任务。", taskTitle, projTitle),
			})
			ui.Text(c, "点击“一并恢复项目与任务”将立即恢复该项目及其关联任务。").
				FontSize(13).TextColor(tk.TextSecondary)
		})
	}) {
		w.RestoreOrphanTaskWithParent()
	}

	// 2. Permanent Delete Confirmation Modal
	if uiModal(c, &w.confirmPermDeleteOpen, modalProps{
		Title:      "彻底删除条目",
		Width:      460,
		OkText:     "确认彻底删除",
		CancelText: "取消",
	}, func() {
		desc := "即将永久删除所选条目及其本地附件，此操作无法恢复。"
		if w.confirmPermDeleteItem != nil {
			switch w.confirmPermDeleteItem.Kind {
			case projects.TrashProject:
				desc = fmt.Sprintf("即将永久删除项目及其下属所有任务「%s」与本地附件，此操作无法恢复。", w.confirmPermDeleteItem.Name)
			case jobapplications.TrashKind:
				desc = fmt.Sprintf("即将永久删除投递记录「%s」及其全部流程时间线，此操作无法恢复。", w.confirmPermDeleteItem.Name)
			default:
				desc = fmt.Sprintf("即将永久删除任务「%s」及其本地附件，此操作无法恢复。", w.confirmPermDeleteItem.Name)
			}
		}
		ui.Column(c).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			uiAlert(c, "此操作不可撤销", alertProps{
				Type:        alertError,
				Description: desc,
			})
		})
	}) {
		w.ConfirmPermanentDelete()
	}

	// 3. Clear Trash Confirmation Modal
	if uiModal(c, &w.confirmClearTrashOpen, modalProps{
		Title:      "清空回收站确认",
		Width:      460,
		OkText:     "确认清空回收站",
		CancelText: "取消",
	}, func() {
		ui.Column(c).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			uiAlert(c, "清空后无法恢复", alertProps{
				Type:        alertError,
				Description: fmt.Sprintf("即将永久删除回收站中的全部 %d 项内容（包括项目、任务、求职投递及关联数据）。", len(w.trashItems)),
			})
		})
	}) {
		w.ConfirmClearTrash()
	}
}
