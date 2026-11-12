package frontend

import (
	"fmt"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (w *Workbench) renderProjectsView(c *ui.Context, tk *theme.Tokens, archived bool) {
	list := w.activeProjects
	title := "所有项目"
	subtitle := fmt.Sprintf("共 %d 个活跃项目", len(w.activeProjects))
	if archived {
		list = w.archivedProjects
		title = "已归档项目"
		subtitle = fmt.Sprintf("共 %d 个已归档项目", len(w.archivedProjects))
	}

	ui.Column(c).Fill().Children(func() {
		// Top Header Bar
		ui.Row(c).Padding(20, 24, 16, 24).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					if archived {
						ui.Icon(c, lucide.Archive()).FontSize(22).TextColor(tk.TextSecondary)
					} else {
						ui.Icon(c, lucide.LayoutGrid()).FontSize(22).TextColor(tk.Primary)
					}
					ui.Text(c, title).FontSize(20).FontWeight(700).TextColor(tk.Text).SingleLine()
				})
				ui.Text(c, subtitle).FontSize(13).TextColor(tk.TextSecondary).SingleLine()
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				if !archived {
					if len(w.archivedProjects) > 0 {
						if action.Button(c, fmt.Sprintf("已归档 (%d)", len(w.archivedProjects)), action.ButtonProps{
							Variant: action.Default,
						}).Clicked() {
							w.NavigateTo(NavArchivedProjects)
						}
					}
					if action.Button(c, "新建项目", action.ButtonProps{
						Variant: action.Primary,
					}).Clicked() {
						w.OpenCreateProjectModal()
					}
				} else {
					if action.Button(c, "返回所有项目", action.ButtonProps{
						Variant: action.Default,
					}).Clicked() {
						w.NavigateTo(NavAllProjects)
					}
				}
			})
		})

		// Body Content
		ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Padding(24).Gap(16).AlignSelf(ui.Stretch).Children(func() {
				if len(list) == 0 {
					ui.Box(c).Padding(48, 24).Radius(tk.BorderRadiusLG).Background(tk.BgContainer).
						Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
						if archived {
							display.Empty(c, display.EmptyProps{
								Icon:        lucide.Archive(),
								Title:       "暂无已归档项目",
								Description: "归档后的项目将保存在此处，不占用左侧活跃项目列表空间。",
								Action: func() {
									if action.Button(c, "查看所有活跃项目", action.ButtonProps{
										Variant: action.Default,
									}).Clicked() {
										w.NavigateTo(NavAllProjects)
									}
								},
							})
						} else {
							display.Empty(c, display.EmptyProps{
								Icon:        lucide.FolderKanban(),
								Title:       "还没有任何项目",
								Description: "创建一个项目即可开始使用计划、进行中、已完成三列看板管理您的任务与本地附件。",
								Action: func() {
									if action.Button(c, "创建第一个项目", action.ButtonProps{
										Variant: action.Primary,
									}).Clicked() {
										w.OpenCreateProjectModal()
									}
								},
							})
						}
					})
					return
				}

				// Project Cards List
				ui.Column(c).Gap(14).AlignSelf(ui.Stretch).Children(func() {
					for _, proj := range list {
						w.renderProjectCard(c.Key(proj.ID), tk, proj, archived)
					}
				})
			})
		})
	})
}

func (w *Workbench) renderProjectCard(c *ui.Context, tk *theme.Tokens, p model.Project, archived bool) {
	sum := w.projectSummaries[p.ID]
	pct := float32(0)
	if sum.Total > 0 {
		pct = float32(sum.Done) * 100 / float32(sum.Total)
	}

	card := ui.Column(c).Padding(18, 20).Gap(12).Radius(tk.BorderRadiusLG).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch)

	card.Children(func() {
		// Top Row: Icon + Name + Description + Actions
		ui.Row(c).Gap(14).AlignItems(ui.Start).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Start).Grow(1).MinWidth(0).Children(func() {
				ui.Box(c).Size(42, 42).Radius(tk.BorderRadius).Background(tk.FillTertiary).
					Center().Shrink(0).Children(func() {
					ui.Icon(c, ProjectIconSVG(p.Icon)).FontSize(22).TextColor(tk.Primary)
				})

				ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, p.Name).FontSize(16).FontWeight(600).TextColor(tk.Text).SingleLine()
						if p.Archived {
							display.Tag(c, "已归档", display.TagProps{
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
					ui.Text(c, desc).FontSize(13).TextColor(tk.TextSecondary)
				})
			})

			// Card Action Buttons
			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				boardLabel := "打开看板"
				if archived {
					boardLabel = "查看看板"
				}
				if action.Button(c.Key("open-"+p.ID), boardLabel, action.ButtonProps{
					Variant: action.Primary,
					Size:    theme.Small,
				}).Label(boardLabel + " " + p.Name).Clicked() {
					w.SelectProject(p.ID)
				}

				if action.Button(c.Key("edit-"+p.ID), "编辑", action.ButtonProps{
					Variant: action.Default,
					Size:    theme.Small,
				}).Label("编辑项目 " + p.Name).Clicked() {
					w.OpenEditProjectModal(p)
				}

				if archived {
					if action.Button(c.Key("unarchive-"+p.ID), "恢复归档", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Label("恢复归档 " + p.Name).Clicked() {
						w.UnarchiveProject(p.ID)
					}
				} else {
					if action.Button(c.Key("archive-"+p.ID), "归档", action.ButtonProps{
						Variant: action.Default,
						Size:    theme.Small,
					}).Label("归档项目 " + p.Name).Clicked() {
						w.ArchiveProject(p.ID)
					}
				}

				if action.Button(c.Key("trash-"+p.ID), "移入回收站", action.ButtonProps{
					Variant: action.Text,
					Size:    theme.Small,
				}).Label("移入回收站 " + p.Name).Clicked() {
					w.SoftDeleteProject(p.ID)
				}
			})
		})

		// Bottom Row: Task statistics & progress
		ui.Row(c).Padding(10, 0, 0, 0).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				display.Tag(c, fmt.Sprintf("总任务 %d", sum.Total), display.TagProps{
					Size:    theme.Small,
					Variant: display.TagFilled,
				})
				display.Tag(c, fmt.Sprintf("计划 %d", sum.Planned), display.TagProps{
					Size:    theme.Small,
					Variant: display.TagFilled,
					Color:   display.TagBlue,
				})
				display.Tag(c, fmt.Sprintf("进行中 %d", sum.InProgress), display.TagProps{
					Size:    theme.Small,
					Variant: display.TagFilled,
					Color:   display.TagOrange,
				})
				display.Tag(c, fmt.Sprintf("已完成 %d", sum.Done), display.TagProps{
					Size:    theme.Small,
					Variant: display.TagFilled,
					Color:   display.TagGreen,
				})
			})

			ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
				if sum.Total > 0 {
					ui.Box(c).Width(140).Children(func() {
						feedback.Progress(c, pct, feedback.ProgressProps{
							Size:     theme.Small,
							ShowInfo: true,
							Format:   formatProgressPercent,
						})
					})
				}
				timeLabel := "更新于 " + p.UpdatedAt.Local().Format("2006-01-02 15:04")
				ui.Text(c, timeLabel).FontSize(12).TextColor(tk.TextTertiary).SingleLine()
			})
		})
	})
}

func (w *Workbench) renderProjectModal(c *ui.Context, tk *theme.Tokens) {
	title := "新建项目"
	okText := "创建项目"
	if w.projectModalEditID != "" {
		title = "编辑项目"
		okText = "保存修改"
	}

	if feedback.Modal(c, &w.projectModalOpen, feedback.ModalProps{
		Title:      title,
		Width:      480,
		OkText:     okText,
		CancelText: "取消",
	}, func() {
		ui.Column(c).Gap(14).AlignSelf(ui.Stretch).Children(func() {
			if w.projectModalErr != "" {
				feedback.Alert(c.Key("project-modal-err"), w.projectModalErr, feedback.AlertProps{
					Type:    feedback.Error,
					Variant: feedback.Soft,
				})
			}

			// Project Name
			ui.Column(c.Key("project-modal-name-sec")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "项目名称 *").FontSize(13).FontWeight(500).TextColor(tk.Text)
				nameBefore := w.projectModalName
				inp := field.Input(c, &w.projectModalName, field.InputProps{
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
						btn := ui.ButtonBase(c.Key("icon-"+opt.Key)).Padding(6, 10).Gap(6).Radius(tk.BorderRadiusSM).
							AlignItems(ui.Center).Cursor(ui.CursorPointer)
						if chosen {
							btn.Background(tk.PrimaryFillTertiary).Border(1, tk.Primary).TextColor(tk.Primary)
						} else {
							btn.Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).TextColor(tk.TextSecondary)
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
				field.TextArea(c, &w.projectModalDesc, field.TextAreaProps{
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
