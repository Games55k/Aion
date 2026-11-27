package frontend

import (
	"fmt"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

func (w *Workbench) renderProjectsView(c *ui.Context, tk *uiTokens, archived bool) {
	list := w.activeProjects
	title := "所有项目"
	if archived {
		list = w.archivedProjects
		title = "已归档项目"
	}

	ui.Scroll(c).Fill().Background(tk.BgLayout).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Column(c).Padding(40, 48, 56, 48).Gap(32).WidthPercent(100).MaxWidth(1120).Children(func() {
				ui.Row(c).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, title).FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()

					ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
						if !archived {
							if len(w.archivedProjects) > 0 {
								if uiButton(c, fmt.Sprintf("已归档 (%d)", len(w.archivedProjects)), buttonProps{
									Variant: buttonDefault,
								}).Label("已归档项目").Clicked() {
									w.NavigateTo(NavArchivedProjects)
								}
							}
							if uiButton(c, "新建项目", buttonProps{
								Variant: buttonPrimary,
							}).Clicked() {
								w.OpenCreateProjectModal()
							}
						} else {
							if uiButton(c, "返回所有项目", buttonProps{
								Variant: buttonDefault,
							}).Clicked() {
								w.NavigateTo(NavAllProjects)
							}
						}
					})
				})

				if len(list) == 0 {
					ui.Column(c).Padding(48, 0).Gap(16).AlignItems(ui.Start).Children(func() {
						if archived {
							ui.Text(c, "暂无已归档项目").FontSize(16).TextColor(tk.TextSecondary)
							if uiButton(c, "查看所有活跃项目", buttonProps{Variant: buttonDefault}).Clicked() {
								w.NavigateTo(NavAllProjects)
							}
						} else {
							ui.Text(c, "还没有任何项目").FontSize(16).TextColor(tk.TextSecondary)
							if uiButton(c, "创建第一个项目", buttonProps{Variant: buttonPrimary}).Clicked() {
								w.OpenCreateProjectModal()
							}
						}
					})
					return
				}

				ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
					for _, proj := range list {
						w.renderProjectCard(c.Key(proj.ID), tk, proj, archived)
					}
				})
			})
		})
	})
}

func (w *Workbench) renderProjectCard(c *ui.Context, tk *uiTokens, p model.Project, archived bool) {
	sum := w.projectSummaries[p.ID]
	pct := 0
	if sum.Total > 0 {
		pct = sum.Done * 100 / sum.Total
	}

	card := ui.Column(c).Padding(20, 24).Gap(16).Radius(20).
		Background(tk.FillTertiary).AlignSelf(ui.Stretch).Cursor(ui.CursorPointer).
		Label("项目卡片 " + p.Name)
	if card.Hovered() {
		card.Background(tk.FillSecondary)
	}

	actionClicked := false
	card.Children(func() {
		// Top Row: Icon + Name + Optional Description + Actions
		ui.Row(c).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(16).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
				ui.Box(c).Size(40, 40).Radius(12).Background(tk.BgElevated).
					Center().Shrink(0).Children(func() {
					uiIconBox(c, ProjectIconSVG(p.Icon), tk.Text)
				})

				ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, p.Name).FontSize(16).FontWeight(600).TextColor(tk.Text).SingleLine()
						if p.Archived {
							ui.Text(c, "已归档").FontSize(14).FontWeight(500).TextColor(tk.Primary).SingleLine()
						}
					})
					if desc := strings.TrimSpace(p.Description); desc != "" {
						ui.Text(c, desc).FontSize(14).TextColor(tk.TextSecondary).SingleLine()
					}
				})
			})

			// Card Action Buttons
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				if uiButton(c.Key("edit-"+p.ID), "编辑", buttonProps{
					Variant: buttonText,
					Size:    sizeSmall,
				}).Label("编辑项目 " + p.Name).Clicked() {
					actionClicked = true
					w.OpenEditProjectModal(p)
				}

				if archived {
					if uiButton(c.Key("unarchive-"+p.ID), "恢复归档", buttonProps{
						Variant: buttonText,
						Size:    sizeSmall,
					}).Label("恢复归档 " + p.Name).Clicked() {
						actionClicked = true
						w.UnarchiveProject(p.ID)
					}
				} else {
					if uiButton(c.Key("archive-"+p.ID), "归档", buttonProps{
						Variant: buttonText,
						Size:    sizeSmall,
					}).Label("归档项目 " + p.Name).Clicked() {
						actionClicked = true
						w.ArchiveProject(p.ID)
					}
				}

				if uiButton(c.Key("trash-"+p.ID), "移入回收站", buttonProps{
					Variant: buttonText,
					Size:    sizeSmall,
					Danger:  true,
				}).Label("移入回收站 " + p.Name).Clicked() {
					actionClicked = true
					w.SoftDeleteProject(p.ID)
				}
			})
		})

		// Bottom Row: Minimalist Monochrome Metrics & Progress (zero divider / multi-color badges)
		ui.Row(c).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, fmt.Sprintf("计划 %d · 进行中 %d · 已完成 %d", sum.Planned, sum.InProgress, sum.Done)).
				Font("monospace").FontSize(14).TextColor(tk.TextSecondary).SingleLine()

			ui.Row(c).Gap(16).AlignItems(ui.Center).Shrink(0).Children(func() {
				if sum.Total > 0 {
					ui.Text(c, fmt.Sprintf("%d%%", pct)).
						Font("monospace").FontSize(14).FontWeight(600).TextColor(tk.Primary).SingleLine()
				}
				ui.Text(c, p.UpdatedAt.Local().Format("2006-01-02 15:04")).
					Font("monospace").FontSize(14).TextColor(tk.TextSecondary).SingleLine()
			})
		})
	})

	if card.Clicked() && !actionClicked {
		w.SelectProject(p.ID)
	}
}

func (w *Workbench) renderProjectModal(c *ui.Context, tk *uiTokens) {
	title := "新建项目"
	okText := "创建项目"
	if w.projectModalEditID != "" {
		title = "编辑项目"
		okText = "保存修改"
	}

	if uiModal(c, &w.projectModalOpen, modalProps{
		Title:      title,
		Width:      480,
		OkText:     okText,
		CancelText: "取消",
		OkDisabled: w.recordWritePending,
	}, func() {
		ui.Column(c).Gap(14).AlignSelf(ui.Stretch).Children(func() {
			if w.projectModalErr != "" {
				uiAlert(c.Key("project-modal-err"), w.projectModalErr, alertProps{
					Type: alertError,
				})
			}

			// Project Name
			ui.Column(c.Key("project-modal-name-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "项目名称 *").FontSize(13).FontWeight(500).TextColor(tk.Text)
				nameBefore := w.projectModalName
				inp := uiInput(c, &w.projectModalName, inputProps{
					Placeholder: "输入项目名称（例如：Aion 桌面端研发）",
					Invalid:     w.projectModalErr != "" && strings.TrimSpace(w.projectModalName) == "",
				}).Label("项目名称输入框")
				if w.projectModalName != nameBefore && w.projectModalErr != "" {
					w.projectModalErr = ""
				}
				if inp.Submitted() {
					w.SubmitProjectModal()
				}
			})

			// Project Icon Picker
			ui.Column(c.Key("project-modal-icon-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "项目图标").FontSize(13).FontWeight(500).TextColor(tk.Text)
				ui.Row(c).Gap(6).Wrap().AlignSelf(ui.Stretch).Children(func() {
					for _, opt := range ProjectIconOptions {
						chosen := w.projectModalIcon == opt.Key
						btn := ui.ButtonBase(c.Key("icon-"+opt.Key)).Padding(8, 12).Gap(8).Radius(12).
							AlignItems(ui.Center).Cursor(ui.CursorPointer)
						if chosen {
							btn.Background(tk.PrimaryFillTertiary).TextColor(tk.Primary)
						} else {
							btn.Background(tk.FillTertiary).TextColor(tk.TextSecondary)
						}
						btn.Label("图标 " + opt.Label)
						btn.Children(func() {
							ui.Icon(c, opt.SVG()).FontSize(14)
							ui.Text(c, opt.Label).FontSize(12).SingleLine()
						})
						if btn.Clicked() {
							w.projectModalIcon = opt.Key
						}
					}
				})
			})

			// Project Description
			ui.Column(c.Key("project-modal-desc-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "项目简介").FontSize(13).FontWeight(500).TextColor(tk.Text)
				uiTextArea(c, &w.projectModalDesc, textAreaProps{
					Placeholder: "补充项目目标、交付物或备注说明（可选）",
					MinRows:     3,
					MaxRows:     5,
				}).Label("项目简介输入框")
			})
		})
	}) {
		w.SubmitProjectModal()
	}
}

func formatProgressPercent(pct float32) string {
	return fmt.Sprintf("%.0f%%", pct)
}
