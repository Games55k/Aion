package frontend

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var (
	backupIntervalOptions = []field.SelectOption{
		{Value: "6h", Label: "每 6 小时"},
		{Value: "12h", Label: "每 12 小时"},
		{Value: "24h", Label: "每天 (每 24 小时)"},
		{Value: "168h", Label: "每周 (每 7 天)"},
	}
	backupKeepOptions = []field.SelectOption{
		{Value: "3", Label: "保留最近 3 份"},
		{Value: "5", Label: "保留最近 5 份"},
		{Value: "7", Label: "保留最近 7 份 (默认)"},
		{Value: "14", Label: "保留最近 14 份"},
		{Value: "30", Label: "保留最近 30 份"},
	}
)

func (w *Workbench) renderSettingsView(c *ui.Context, tk *theme.Tokens) {
	ui.Column(c).Fill().Children(func() {
		// Top Header Bar
		ui.Row(c).Padding(20, 24, 16, 24).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, lucide.Settings()).FontSize(22).TextColor(tk.Primary)
					ui.Text(c, "全局设置与数据管理").FontSize(20).FontWeight(700).TextColor(tk.Text).SingleLine()
				})
				ui.Text(c, "配置界面外观主题、查看本地 SQLite 与受管附件目录，以及管理本地自动备份与完整恢复。").
					FontSize(13).TextColor(tk.TextSecondary).SingleLine()
			})
		})

		// Settings Body
		ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Padding(24).Gap(20).MaxWidth(860).AlignSelf(ui.Stretch).Children(func() {
				if w.IsBusy() {
					ui.Row(c.Key("settings-busy-banner")).Padding(12, 16).Gap(12).Radius(tk.BorderRadius).
						Background(tk.PrimaryFillQuaternary).Border(1, tk.Primary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, w.BusyLabel()).FontSize(13).FontWeight(500).TextColor(tk.Text).Grow(1).SingleLine()
						if action.Button(c, "取消操作", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Clicked() {
							w.CancelBusyTask()
						}
					})
				}

				// 1. Appearance & Theme Card
				ui.Column(c.Key("settings-card-theme")).Padding(20).Gap(14).Radius(tk.BorderRadiusLG).
					Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(4).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, lucide.Palette()).FontSize(18).TextColor(tk.Primary)
								ui.Text(c, "外观与主题").FontSize(16).FontWeight(600).TextColor(tk.Text)
							})
							ui.Text(c, "支持跟随 macOS 系统外观，或固定为浅色、深色模式。设置将自动持久化到本地数据库。").
								FontSize(13).TextColor(tk.TextSecondary)
						})

						beforeIdx := w.themeSegmentIdx
						action.Segmented(c.Key("theme-segmented"), &w.themeSegmentIdx, []string{"跟随系统", "浅色", "深色"}, action.SegmentedProps{
							Size: theme.Middle,
						}).Label("主题模式切换")
						if w.themeSegmentIdx != beforeIdx {
							switch w.themeSegmentIdx {
							case 1:
								w.ApplyThemeSetting(ThemeLight)
							case 2:
								w.ApplyThemeSetting(ThemeDark)
							default:
								w.ApplyThemeSetting(ThemeSystem)
							}
						}
					})
				})

				// 2. Local Data Directory Card
				ui.Column(c.Key("settings-card-datadir")).Padding(20).Gap(14).Radius(tk.BorderRadiusLG).
					Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, lucide.HardDrive()).FontSize(18).TextColor(tk.Primary)
						ui.Text(c, "本地数据存储位置").FontSize(16).FontWeight(600).TextColor(tk.Text)
						display.Tag(c, "Offline-First", display.TagProps{
							Size:    theme.Small,
							Variant: display.TagFilled,
							Color:   display.TagGreen,
						})
					})
					ui.Text(c, "所有项目、任务、回收站、设置（aion.sqlite）与受管附件（attachments/）均保存在以下本地目录，无需任何云端账号：").
						FontSize(13).TextColor(tk.TextSecondary)

					dataDir := w.app.DataDir()
					ui.Row(c).Padding(10, 12).Gap(10).Radius(tk.BorderRadius).
						Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, dataDir).Font("monospace").FontSize(12).TextColor(tk.Text).Grow(1).SingleLine()

						ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
							if action.Button(c, "复制路径", action.ButtonProps{
								Variant: action.Default,
								Size:    theme.Small,
							}).Clicked() {
								c.Services().WriteClipboard(dataDir)
								if w.host.CopyText != nil {
									w.host.CopyText(dataDir)
								}
								w.EnqueueToast(feedback.ToastProps{
									Title:       "已复制数据目录路径",
									Description: dataDir,
									Type:        feedback.ToastSuccess,
								})
							}
							if action.Button(c, "在 Finder 中打开", action.ButtonProps{
								Variant: action.Default,
								Size:    theme.Small,
							}).Clicked() {
								var openErr error
								if w.host.OpenPath != nil {
									openErr = w.host.OpenPath(dataDir)
								} else if w.host.RevealPath != nil {
									openErr = w.host.RevealPath(dataDir)
								}
								if openErr != nil {
									w.EnqueueToast(feedback.ToastProps{
										Title:       "在 Finder 中打开失败",
										Description: openErr.Error(),
										Type:        feedback.ToastError,
									})
								}
							}
						})
					})
				})

				// 3. Automatic Local Backup Card
				ui.Column(c.Key("settings-card-backup-auto")).Padding(20).Gap(16).Radius(tk.BorderRadiusLG).
					Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(4).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, lucide.Clock()).FontSize(18).TextColor(tk.Primary)
								ui.Text(c, "本地自动备份").FontSize(16).FontWeight(600).TextColor(tk.Text)
							})
							ui.Text(c, "应用启动及运行期间按周期自动生成一致性快照压缩包（auto-*.zip），并自动轮转清理超出保留份数的旧自动备份。").
								FontSize(13).TextColor(tk.TextSecondary)
						})

						ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
							ui.Text(c, "启用自动备份").FontSize(13).TextColor(tk.Text)
							beforeEn := w.backupEnabled
							choice.Switch(c.Key("backup-enabled-switch"), &w.backupEnabled, choice.SwitchProps{}).
								Label("启用自动备份开关")
							if w.backupEnabled != beforeEn {
								w.SaveBackupPolicy()
							}
						})
					})

					// Status Banner
					lastSuccessStr := "尚无自动备份记录"
					if !w.backupStatus.LastSuccess.IsZero() {
						lastSuccessStr = w.backupStatus.LastSuccess.Local().Format("2006-01-02 15:04:05")
					}
					ui.Row(c).Padding(10, 14).Gap(12).Radius(tk.BorderRadius).
						Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, lucide.CircleCheck()).FontSize(15).TextColor(tk.Success)
							ui.Text(c, "最近一次自动备份成功时间: "+lastSuccessStr).FontSize(13).TextColor(tk.Text)
						})

						if action.Button(c, "立即检查到期备份", action.ButtonProps{
							Variant: action.Default,
							Size:    theme.Small,
						}).Disabled(w.IsBusy()).Clicked() {
							w.RunDueBackupNow()
						}
					})

					// Error Banner if scheduler reported an error
					errMsg := w.backupStatus.LastError
					if errMsg == "" {
						errMsg = w.LastAsyncError()
					}
					if errMsg != "" {
						feedback.Alert(c.Key("settings-backup-err"), "最近一次自动备份失败", feedback.AlertProps{
							Type:        feedback.Error,
							Variant:     feedback.Soft,
							Description: errMsg,
						})
					}

					// Frequency & Keep Count Row
					ui.Row(c.Key("settings-backup-freq-keep")).Gap(16).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
							ui.Text(c, "自动备份频率").FontSize(13).FontWeight(500).TextColor(tk.Text)
							beforeInt := w.backupIntervalSelect
							field.Select(c.Key("backup-interval-select"), &w.backupIntervalSelect, field.SelectProps{
								Options: backupIntervalOptions,
							}).Label("自动备份频率选择")
							if w.backupIntervalSelect != beforeInt {
								w.SaveBackupPolicy()
							}
						})

						ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
							ui.Text(c, "自动备份保留份数").FontSize(13).FontWeight(500).TextColor(tk.Text)
							beforeKeep := w.backupKeepSelect
							field.Select(c.Key("backup-keep-select"), &w.backupKeepSelect, field.SelectProps{
								Options: backupKeepOptions,
							}).Label("自动备份保留份数选择")
							if w.backupKeepSelect != beforeKeep {
								w.SaveBackupPolicy()
							}
						})
					})

					// Backup Directory Row
					ui.Column(c.Key("settings-backup-dir")).Gap(6).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "自动备份存储目录").FontSize(13).FontWeight(500).TextColor(tk.Text)
						ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Children(func() {
								inp := field.Input(c.Key("backup-dir-input"), &w.backupDirInput, field.InputProps{
									Placeholder: "默认目录: " + filepath.Join(w.app.DataDir(), "backups"),
								}).Label("自动备份目录输入框")
								if inp.Submitted() {
									w.SaveBackupPolicy()
								}
							})

							if action.Button(c, "选择目录...", action.ButtonProps{
								Variant: action.Default,
							}).Clicked() {
								if pick := w.host.PickDirectory; pick != nil {
									curDir := w.backupDirInput
									w.runDialog(func() func() {
										dir, err := pick("选择自动备份存储目录", curDir)
										if err != nil || dir == "" {
											return nil
										}
										return func() {
											w.backupDirInput = dir
											w.SaveBackupPolicy()
										}
									})
								}
							}

							if action.Button(c, "保存策略", action.ButtonProps{
								Variant: action.Primary,
							}).Clicked() {
								w.SaveBackupPolicy()
							}
						})
					})
				})

				// 4. Manual Backup Export & Full Restore Card
				ui.Column(c.Key("settings-card-backup-manual")).Padding(20).Gap(16).Radius(tk.BorderRadiusLG).
					Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Gap(4).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, lucide.ArchiveRestore()).FontSize(18).TextColor(tk.Primary)
							ui.Text(c, "手动备份导出与完整恢复").FontSize(16).FontWeight(600).TextColor(tk.Text)
						})
						ui.Text(c, "手动导出的完整备份包（.zip）包含 SQLite 快照（database.sqlite）、全部受管附件（attachments/）及 SHA-256 校验清单（manifest.json），可用于跨设备迁移或灾难恢复。").
							FontSize(13).TextColor(tk.TextSecondary)
					})

					if w.lastExportedZipPath != "" {
						feedback.Alert(c.Key("settings-export-alert"), "最近一次手动备份已导出", feedback.AlertProps{
							Type:        feedback.Success,
							Variant:     feedback.Soft,
							Description: "备份包路径: " + w.lastExportedZipPath,
						})
					}

					if w.lastRestoreResult != nil {
						feedback.Alert(c.Key("settings-restore-alert"), "数据已从备份包完整恢复", feedback.AlertProps{
							Type:        feedback.Success,
							Variant:     feedback.Soft,
							Description: fmt.Sprintf("恢复前自动生成的安全备份已保存在: %s", w.lastRestoreResult.SafetyBackupPath),
						})
					}

					ui.Row(c.Key("settings-manual-actions")).Gap(12).AlignItems(ui.Center).Children(func() {
						if action.Button(c, "导出完整备份 (.zip)", action.ButtonProps{
							Variant: action.Primary,
						}).Disabled(w.IsBusy()).Clicked() {
							w.PickAndExportManualBackup()
						}

						if action.Button(c, "从备份包恢复 (.zip)...", action.ButtonProps{
							Variant: action.Default,
						}).Disabled(w.IsBusy()).Clicked() {
							if w.IsBusy() {
								return
							}
							if err := w.FlushPending(); err != nil {
								desc := w.saveErr
								if desc == "" {
									desc = err.Error()
								}
								w.EnqueueToast(feedback.ToastProps{
									Title:       "当前任务存在未保存修改",
									Description: desc + "，请先修正保存或放弃修改后再恢复备份",
									Type:        feedback.ToastWarning,
								})
								return
							}
							if pick := w.host.PickOpenFiles; pick != nil {
								w.runDialog(func() func() {
									paths, err := pick("选择要恢复的 Aion 备份包 (.zip)", false, "zip")
									if err != nil || len(paths) == 0 {
										return nil
									}
									return func() {
										w.RequestRestoreBackup(paths[0])
									}
								})
							}
						}
					})
				})
			})
		})
	})
}

func (w *Workbench) renderSettingsModals(c *ui.Context, tk *theme.Tokens) {
	if feedback.Modal(c, &w.confirmRestoreOpen, feedback.ModalProps{
		Title:      "确认从备份包完整恢复数据",
		Width:      520,
		OkText:     "确认覆盖并恢复",
		CancelText: "取消",
	}, func() {
		ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			feedback.Alert(c, "整包覆盖恢复说明", feedback.AlertProps{
				Type:        feedback.Warning,
				Variant:     feedback.Soft,
				Description: "从备份包恢复将完整替换当前本地的所有项目、任务、子任务、标签、受管附件、回收站以及全局设置。",
			})
			ui.Column(c).Gap(6).Padding(10, 12).Radius(tk.BorderRadius).
				Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "待恢复备份包:").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				ui.Text(c, w.pendingRestoreZip).Font("monospace").FontSize(12).TextColor(tk.Text)
			})
			ui.Text(c, "• 恢复前将先校验备份包清单（manifest.json）、所有文件 SHA-256 摘要及 SQLite 完整性；若备份包损坏，恢复将立即中止且不改动现有数据。").
				FontSize(12).TextColor(tk.TextSecondary)
			ui.Text(c, "• 替换前系统会自动将当前数据打包为一份安全备份（safety-*.zip）保存在备份目录中，以便随时回滚。").
				FontSize(12).TextColor(tk.TextSecondary)
		})
	}) {
		w.TriggerRestoreBackup()
	}
}
