package frontend

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

var (
	backupIntervalOptions = []selectOption{
		{Value: "6h", Label: "每 6 小时"},
		{Value: "12h", Label: "每 12 小时"},
		{Value: "24h", Label: "每天 (每 24 小时)"},
		{Value: "168h", Label: "每周 (每 7 天)"},
	}
	backupKeepOptions = []selectOption{
		{Value: "3", Label: "保留最近 3 份"},
		{Value: "5", Label: "保留最近 5 份"},
		{Value: "7", Label: "保留最近 7 份 (默认)"},
		{Value: "14", Label: "保留最近 14 份"},
		{Value: "30", Label: "保留最近 30 份"},
	}
	updateCheckIntervalOptions = []selectOption{
		{Value: "6h", Label: "每 6 小时"},
		{Value: "12h", Label: "每 12 小时"},
		{Value: "24h", Label: "每天 (每 24 小时)"},
	}
)

func (w *Workbench) renderSettingsView(c *ui.Context, tk *uiTokens) {
	ui.Column(c).Fill().Children(func() {
		// Top Header Bar
		ui.Row(c).MinHeight(66).Padding(16, 24).Gap(10).AlignItems(ui.Center).
			AlignSelf(ui.Stretch).Background(tk.BgContainer).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).DragWindow().Children(func() {
			ui.Icon(c, iconSettings()).FontSize(22).TextColor(tk.Primary)
			ui.Text(c, "全局设置与数据管理").FontSize(20).FontWeight(700).TextColor(tk.Text).SingleLine()
		})

		// Settings Body
		ui.Scroll(c).Grow(1).Basis(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Padding(24).Gap(20).MaxWidth(860).AlignSelf(ui.Stretch).Children(func() {
				if w.IsBusy() {
					ui.Row(c.Key("settings-busy-banner")).Padding(12, 16).Gap(12).Radius(tk.BorderRadius).
						Background(tk.PrimaryFillQuaternary).Border(1, tk.Primary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, w.BusyLabel()).FontSize(13).FontWeight(500).TextColor(tk.Text).Grow(1).SingleLine()
						if uiButton(c, "取消操作", buttonProps{
							Variant: buttonDefault,
							Size:    sizeSmall,
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
								ui.Icon(c, iconPalette()).FontSize(18).TextColor(tk.Primary)
								ui.Text(c, "外观与主题").FontSize(16).FontWeight(600).TextColor(tk.Text)
							})
							ui.Text(c, "支持跟随 macOS 系统外观，或固定为浅色、深色模式。").
								FontSize(13).TextColor(tk.TextSecondary)
						})

						beforeIdx := w.themeSegmentIdx
						uiSegmented(c.Key("theme-segmented"), &w.themeSegmentIdx, []segmentedOption{
							{Label: "跟随系统", Icon: iconMonitor()},
							{Label: "浅色", Icon: iconSun()},
							{Label: "深色", Icon: iconMoon()},
						}, segmentedProps{
							Size: sizeMiddle,
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
					ui.Column(c).Gap(4).Children(func() {
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconHardDrive()).FontSize(18).TextColor(tk.Primary)
							ui.Text(c, "本地数据存储位置").FontSize(16).FontWeight(600).TextColor(tk.Text)
						})
						ui.Text(c, "所有项目、任务、求职投递、回收站、附件与设置均保存在以下本地目录：").
							FontSize(13).TextColor(tk.TextSecondary)
					})

					dataDir := w.app.DataDir()
					ui.Row(c).Padding(10, 12).Gap(10).Radius(tk.BorderRadius).
						Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, dataDir).Font("monospace").FontSize(12).TextColor(tk.Text).Grow(1).SingleLine()

						ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
							if uiButton(c, "复制路径", buttonProps{
								Variant: buttonDefault,
								Size:    sizeSmall,
							}).Clicked() {
								c.Services().WriteClipboard(dataDir)
							}
							if uiButton(c, "在 Finder 中打开", buttonProps{
								Variant: buttonDefault,
								Size:    sizeSmall,
							}).Clicked() {
								if w.host.OpenPath != nil {
									_ = w.host.OpenPath(dataDir)
								} else if w.host.RevealPath != nil {
									_ = w.host.RevealPath(dataDir)
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
								ui.Icon(c, iconClock()).FontSize(18).TextColor(tk.Primary)
								ui.Text(c, "本地自动备份").FontSize(16).FontWeight(600).TextColor(tk.Text)
							})
							ui.Text(c, "按设定周期自动备份本地数据，超出保留份数的旧备份将自动清理。").
								FontSize(13).TextColor(tk.TextSecondary)
						})

						ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
							ui.Text(c, "启用自动备份").FontSize(13).TextColor(tk.Text)
							beforeEn := w.backupEnabled
							uiSwitch(c.Key("backup-enabled-switch"), &w.backupEnabled).
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
							ui.Icon(c, iconCircleCheck()).FontSize(15).TextColor(tk.Success)
							ui.Text(c, "最近一次自动备份成功时间: "+lastSuccessStr).FontSize(13).TextColor(tk.Text)
						})

						if uiButton(c, "立即检查到期备份", buttonProps{
							Variant: buttonDefault,
							Size:    sizeSmall,
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
						uiAlert(c.Key("settings-backup-err"), "最近一次自动备份失败", alertProps{
							Type:        alertError,
							Description: errMsg,
						})
					}

					// Frequency & Keep Count Row
					ui.Row(c.Key("settings-backup-freq-keep")).Gap(16).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
							ui.Text(c, "自动备份频率").FontSize(13).FontWeight(500).TextColor(tk.Text)
							beforeInt := w.backupIntervalSelect
							uiSelect(c.Key("backup-interval-select"), &w.backupIntervalSelect, selectProps{
								Options: withCurrentOption(backupIntervalOptions, w.backupIntervalSelect, intervalLabel(parseIntervalOption(w.backupIntervalSelect))),
							}).Label("自动备份频率选择")
							if w.backupIntervalSelect != beforeInt {
								w.SaveBackupPolicy()
							}
						})

						ui.Column(c).Gap(6).Grow(1).Basis(0).Children(func() {
							ui.Text(c, "自动备份保留份数").FontSize(13).FontWeight(500).TextColor(tk.Text)
							beforeKeep := w.backupKeepSelect
							uiSelect(c.Key("backup-keep-select"), &w.backupKeepSelect, selectProps{
								Options: withCurrentOption(backupKeepOptions, w.backupKeepSelect, "保留最近 "+w.backupKeepSelect+" 份"),
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
								inp := uiInput(c.Key("backup-dir-input"), &w.backupDirInput, inputProps{
									Placeholder: "默认目录: " + filepath.Join(w.app.DataDir(), "backups"),
								}).Label("自动备份目录输入框")
								if inp.Submitted() {
									w.SaveBackupPolicy()
								}
							})

							if uiButton(c, "选择目录...", buttonProps{
								Variant: buttonDefault,
							}).Disabled(w.IsBusy()).Clicked() {
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

							if uiButton(c, "保存策略", buttonProps{
								Variant: buttonPrimary,
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
							ui.Icon(c, iconArchiveRestore()).FontSize(18).TextColor(tk.Primary)
							ui.Text(c, "手动备份导出与完整恢复").FontSize(16).FontWeight(600).TextColor(tk.Text)
						})
						ui.Text(c, "将全部项目、任务、求职投递、附件与设置导出为 .zip 备份包，可用于跨设备迁移或恢复数据。").
							FontSize(13).TextColor(tk.TextSecondary)
					})

					if w.lastExportedZipPath != "" {
						uiAlert(c.Key("settings-export-alert"), "最近一次手动备份已导出", alertProps{
							Type:        alertSuccess,
							Description: "备份包路径: " + w.lastExportedZipPath,
						})
					}

					if w.lastRestoreError != "" {
						uiAlert(c.Key("settings-restore-error"), "备份恢复失败", alertProps{
							Type:        alertError,
							Description: w.lastRestoreError,
						})
					} else if w.lastRestoreResult != nil {
						uiAlert(c.Key("settings-restore-alert"), "数据已从备份包完整恢复", alertProps{
							Type:        alertSuccess,
							Description: fmt.Sprintf("恢复前自动生成的安全备份已保存在: %s", w.lastRestoreResult.SafetyBackupPath),
						})
					}

					ui.Row(c.Key("settings-manual-actions")).Gap(12).AlignItems(ui.Center).Children(func() {
						if uiButton(c, "导出完整备份 (.zip)", buttonProps{
							Variant: buttonPrimary,
						}).Disabled(w.IsBusy()).Clicked() {
							w.PickAndExportManualBackup()
						}

						if uiButton(c, "从备份包恢复 (.zip)...", buttonProps{
							Variant: buttonDefault,
						}).Disabled(w.IsBusy()).Clicked() {
							if w.IsBusy() {
								return
							}
							if err := w.flushPendingOrRevealDraft(); err != nil {
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

				// 5. Workspace Modules Visibility Card
				w.renderSettingsModulesCard(c, tk)

				// 6. Local AI / MCP Integration Card
				w.renderSettingsMCPCard(c, tk)

				// 7. App Version & Updates Card
				w.renderSettingsUpdatesCard(c, tk)
			})
		})
	})
}

func (w *Workbench) renderSettingsModulesCard(c *ui.Context, tk *uiTokens) {
	visibleCount := len(w.VisibleModules())
	ui.Column(c.Key("settings-card-modules")).Padding(20).Gap(14).Radius(tk.BorderRadiusLG).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconLayers()).FontSize(18).TextColor(tk.Primary)
				ui.Text(c, "工作区模块").FontSize(16).FontWeight(600).TextColor(tk.Text)
			})
			ui.Text(c, "按当前阶段按需显示或隐藏内置模块；隐藏仅收起侧边栏入口，不会删除任何本地数据。").
				FontSize(13).TextColor(tk.TextSecondary)
		})

		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			for _, mod := range w.modules {
				m := mod
				visible := w.IsModuleVisible(m.ID)
				lockedLast := visible && visibleCount <= 1
				ui.Row(c.Key("module-toggle-"+m.ID)).Padding(10, 12).Gap(12).Radius(tk.BorderRadius).
					Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
					AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
						ui.Icon(c, ModuleIconSVG(m)).FontSize(16).TextColor(tk.Primary)
						ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, m.Name).FontSize(14).FontWeight(500).TextColor(tk.Text).SingleLine()
							ui.Text(c, moduleDescription(m.ID)).FontSize(12).TextColor(tk.TextSecondary).SingleLine()
						})
					})

					ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
						if lockedLast {
							ui.Text(c, "至少保留一个模块").FontSize(12).TextColor(tk.TextTertiary).SingleLine()
						}
						checked := visible
						uiSwitch(c.Key("module-switch-"+m.ID), &checked).
							Label("显示模块 " + m.Name).
							Disabled(lockedLast)
						if checked != visible {
							w.SetModuleVisible(m.ID, checked)
						}
					})
				})
			}
		})
	})
}

func (w *Workbench) renderSettingsMCPCard(c *ui.Context, tk *uiTokens) {
	info := w.MCPCapability()
	cmd, args := mcpCommandAndArgs(info)
	cmdPreview := shellQuote(cmd)
	for _, a := range args {
		cmdPreview += " " + shellQuote(a)
	}

	ui.Column(c.Key("settings-card-mcp")).Padding(20).Gap(14).Radius(tk.BorderRadiusLG).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconTerminal()).FontSize(18).TextColor(tk.Primary)
					ui.Text(c, "AI 与 MCP 接入").FontSize(16).FontWeight(600).TextColor(tk.Text)
					if info.Ready {
						uiTag(c, "实时联动已就绪", tagProps{
							Size:    sizeSmall,
							Variant: tagFilled,
							Color:   tagGreen,
						})
					} else if info.LastError != "" {
						uiTag(c, "仅离线可用", tagProps{
							Size:    sizeSmall,
							Variant: tagFilled,
							Color:   tagOrange,
						})
					}
				})
				ui.Text(c, "通过标准 stdio MCP 让 Antigravity、Codex 等本地 AI 助手读写项目、任务与求职投递（仅限软删除进回收站）。").
					FontSize(13).TextColor(tk.TextSecondary)
			})
		})

		ui.Row(c).Padding(10, 12).Gap(10).Radius(tk.BorderRadius).
			Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
			AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, cmdPreview).Font("monospace").FontSize(12).TextColor(tk.Text).Grow(1).SingleLine()
		})

		ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if uiButton(c, "复制 Antigravity 配置", buttonProps{
					Variant: buttonPrimary,
					Size:    sizeSmall,
				}).Clicked() {
					c.Services().WriteClipboard(FormatMCPJSONConfig(info))
				}
				if uiButton(c, "复制 Codex 命令", buttonProps{
					Variant: buttonDefault,
					Size:    sizeSmall,
				}).Clicked() {
					c.Services().WriteClipboard(FormatCodexMCPCommand(info))
				}
				if uiButton(c, "复制 CLI 软链命令", buttonProps{
					Variant: buttonDefault,
					Size:    sizeSmall,
				}).Clicked() {
					c.Services().WriteClipboard(FormatCLISymlinkCommand(info))
				}
			})
			ui.Text(c, "软链后可在终端直接使用 aion summary / task / job").
				FontSize(12).TextColor(tk.TextTertiary).SingleLine()
		})

		if info.LastError != "" {
			uiAlert(c.Key("settings-mcp-error"), "本地实时联动未启动", alertProps{
				Type:        alertWarning,
				Description: info.LastError,
			})
		}
	})
}

func (w *Workbench) renderSettingsUpdatesCard(c *ui.Context, tk *uiTokens) {
	info := w.UpdateCapability()
	canCheck, canOpenURL := w.updateActionsReady()
	displayVersion := displayVersionNumber(info.Version)
	availVersion := displayVersionNumber(info.AvailableVersion)
	installedVersion := displayVersionNumber(info.InstalledVersion)
	versionLine := "当前版本: 开发版本（未打包）"
	if displayVersion != "" {
		versionLine = "当前版本: v" + displayVersion
		if info.IsDev {
			versionLine += "（开发构建）"
		}
	}
	lastCheckStr := "尚未检查"
	if !info.LastCheck.IsZero() {
		lastCheckStr = info.LastCheck.Local().Format("2006-01-02 15:04:05")
	}

	ui.Column(c.Key("settings-card-updates")).Padding(20).Gap(14).Radius(tk.BorderRadiusLG).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconRefreshCw()).FontSize(18).TextColor(tk.Primary)
					ui.Text(c, "应用版本与更新").FontSize(16).FontWeight(600).TextColor(tk.Text)
					if installedVersion != "" {
						uiTag(c, "v"+installedVersion+" 待重启", tagProps{
							Size:    sizeSmall,
							Variant: tagFilled,
							Color:   tagGreen,
						})
					} else if availVersion != "" {
						uiTag(c, "发现新版本 v"+availVersion, tagProps{
							Size:    sizeSmall,
							Variant: tagFilled,
							Color:   tagGreen,
						})
					}
				})
				ui.Text(c, "开启后仅在后台静默检查更新并在侧边栏提示，不会自动下载或安装。").
					FontSize(13).TextColor(tk.TextSecondary)
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Text(c, "自动检查更新").FontSize(13).TextColor(tk.Text)
				beforeAuto := w.updateCheckEnabled
				uiSwitch(c.Key("update-auto-check-switch"), &w.updateCheckEnabled).
					Label("自动检查更新开关")
				if w.updateCheckEnabled != beforeAuto {
					w.SaveUpdateCheckPolicy()
				}
			})
		})

		ui.Row(c).Padding(10, 14).Gap(12).Radius(tk.BorderRadius).
			Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).
			AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, versionLine).FontSize(13).FontWeight(500).TextColor(tk.Text).SingleLine()
				ui.Text(c, "最近一次检查更新: "+lastCheckStr).FontSize(12).TextColor(tk.TextSecondary).SingleLine()
			})

			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				if uiButton(c, "前往 Releases 下载", buttonProps{
					Variant: buttonDefault,
					Size:    sizeSmall,
				}).Disabled(!canOpenURL).Clicked() {
					w.OpenReleasesPage()
				}
				if hasSidebarUpdateIndicator(info) {
					actionLabel := "查看更新"
					switch {
					case info.Status == UpdateReady || installedVersion != "":
						actionLabel = "立即重启更新"
					case info.Status == UpdateError && availVersion != "":
						actionLabel = "重试更新"
					}
					if uiButton(c, actionLabel, buttonProps{
						Variant: buttonPrimary,
						Size:    sizeSmall,
					}).Clicked() {
						w.OpenUpdateModal()
					}
				}
				checkVariant := buttonPrimary
				if hasSidebarUpdateIndicator(info) {
					checkVariant = buttonDefault
				}
				if uiButton(c, "检查更新", buttonProps{
					Variant: checkVariant,
					Size:    sizeSmall,
				}).Disabled(!canCheck).Clicked() {
					w.CheckForUpdatesAndShow()
				}
			})
		})

		ui.Column(c.Key("settings-update-interval")).Gap(6).Width(260).AlignSelf(ui.Start).Children(func() {
			ui.Text(c, "自动检查频率").FontSize(13).FontWeight(500).TextColor(tk.Text)
			beforeInt := w.updateCheckIntervalSelect
			uiSelect(c.Key("update-check-interval-select"), &w.updateCheckIntervalSelect, selectProps{
				Options: withCurrentOption(updateCheckIntervalOptions, w.updateCheckIntervalSelect, intervalLabel(parseIntervalOption(w.updateCheckIntervalSelect))),
			}).Label("自动检查频率选择").Disabled(!w.updateCheckEnabled)
			if w.updateCheckIntervalSelect != beforeInt {
				w.SaveUpdateCheckPolicy()
			}
		})

		switch {
		case info.IsDev || displayVersion == "":
			uiAlert(c.Key("settings-updates-dev"), "当前为开发构建，不支持应用内自动更新", alertProps{
				Type:        alertInfo,
				Description: "请使用正式打包的 .dmg 安装包运行 Aion，或点击“前往 Releases 下载”手动获取最新安装包。",
			})
		case !info.CanSelfUpdate:
			uiAlert(c.Key("settings-updates-readonly"), "当前安装位置不可写，无法应用内自动更新", alertProps{
				Type:        alertWarning,
				Description: "请将 Aion.app 移动到当前用户有写入权限的“应用程序”目录后重试，或点击“前往 Releases 下载”手动下载。",
			})
		case info.Status == UpdateReady || installedVersion != "":
			ver := installedVersion
			if ver == "" {
				ver = availVersion
			}
			uiAlert(c.Key("settings-updates-ready"), "新版本 v"+ver+" 已安装，重启即可生效", alertProps{
				Type:        alertSuccess,
				Description: "点击“立即重启更新”或侧边栏更新图标重启应用。",
			})
		case info.Status == UpdateError && availVersion != "":
			desc := info.LastError
			if desc == "" {
				desc = "点击“重试更新”或侧边栏更新图标重试安装。"
			}
			uiAlert(c.Key("settings-updates-error"), "更新 v"+availVersion+" 失败", alertProps{
				Type:        alertError,
				Description: desc,
			})
		case availVersion != "":
			uiAlert(c.Key("settings-updates-available"), "发现新版本 v"+availVersion, alertProps{
				Type:        alertSuccess,
				Description: "点击“查看更新”或侧边栏更新图标查看更新说明并决定是否安装。",
			})
		}
	})
}

func (w *Workbench) renderSettingsModals(c *ui.Context, tk *uiTokens) {
	if uiModal(c, &w.confirmRestoreOpen, modalProps{
		Title:      "确认从备份包完整恢复数据",
		Width:      520,
		OkText:     "确认覆盖并恢复",
		CancelText: "取消",
	}, func() {
		ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			uiAlert(c, "整包覆盖恢复说明", alertProps{
				Type:        alertWarning,
				Description: "从备份包恢复将完整替换当前工作区的全部项目、任务、求职投递、附件、回收站及全局设置。恢复仅支持当前数据库版本的备份包。",
			})
			ui.Column(c).Gap(6).Padding(10, 12).Radius(tk.BorderRadius).
				Background(tk.FillQuaternary).Border(1, tk.BorderSecondary).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "待恢复备份包:").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
				ui.Text(c, w.pendingRestoreZip).Font("monospace").FontSize(12).TextColor(tk.Text)
			})
			ui.Text(c, "• 恢复前会先校验备份包完整性；若文件损坏将自动中止，不会改动现有数据。").
				FontSize(12).TextColor(tk.TextSecondary)
			ui.Text(c, "• 替换前会自动将当前数据另存为安全备份并保存在备份目录中，以便需要时回滚。").
				FontSize(12).TextColor(tk.TextSecondary)
		})
	}) {
		w.TriggerRestoreBackup()
	}
}
