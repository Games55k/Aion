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
	ui.Scroll(c).Fill().Background(tk.BgLayout).Children(func() {
		ui.Column(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Column(c).Padding(28, 48, 56, 48).Gap(32).WidthPercent(100).MaxWidth(960).Children(func() {
				if w.IsBusy() {
					ui.Row(c.Key("settings-busy-banner")).Padding(12, 16).Gap(12).Radius(16).
						Background(tk.PrimaryFillQuaternary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, w.BusyLabel()).FontSize(14).FontWeight(500).TextColor(tk.Text).Grow(1).SingleLine()
						if uiButton(c, "取消操作", buttonProps{
							Variant: buttonDefault,
							Size:    sizeSmall,
						}).Clicked() {
							w.CancelBusyTask()
						}
					})
				}

				// 1. Appearance (zero nested card/subtitle)
				ui.Row(c.Key("settings-card-theme")).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "外观").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()

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

				// 2. Local Data Directory
				ui.Column(c.Key("settings-card-datadir")).Gap(12).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "数据目录").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()

					dataDir := w.app.DataDir()
					ui.Row(c).Padding(12, 16).Gap(12).Radius(16).Background(tk.FillTertiary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, dataDir).Font("monospace").FontSize(14).TextColor(tk.TextSecondary).Grow(1).SingleLine()

						ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
							if uiButton(c, "复制路径", buttonProps{
								Variant: buttonText,
								Size:    sizeSmall,
							}).Clicked() {
								c.Services().WriteClipboard(dataDir)
							}
							if uiButton(c, "在 Finder 中打开", buttonProps{
								Variant: buttonText,
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

				// 3. Automatic Local Backup
				ui.Column(c.Key("settings-card-backup-auto")).Gap(16).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "自动备份").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()

						ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
							beforeEn := w.backupEnabled
							uiSwitch(c.Key("backup-enabled-switch"), &w.backupEnabled).
								Label("启用自动备份开关")
							if w.backupEnabled != beforeEn {
								w.SaveBackupPolicy()
							}
						})
					})

					lastSuccessStr := "尚无记录"
					if !w.backupStatus.LastSuccess.IsZero() {
						lastSuccessStr = w.backupStatus.LastSuccess.Local().Format("2006-01-02 15:04:05")
					}
					ui.Row(c).Padding(12, 16).Gap(12).Radius(16).Background(tk.FillTertiary).
						AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "最近备份 · "+lastSuccessStr).Font("monospace").FontSize(14).TextColor(tk.TextSecondary).SingleLine()

						if uiButton(c, "立即检查到期备份", buttonProps{
							Variant: buttonText,
							Size:    sizeSmall,
						}).Disabled(w.IsBusy()).Clicked() {
							w.RunDueBackupNow()
						}
					})

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

					ui.Row(c.Key("settings-backup-freq-keep")).Gap(20).AlignItems(ui.Start).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(8).Grow(1).Basis(0).Children(func() {
							ui.Text(c, "备份频率").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
							beforeInt := w.backupIntervalSelect
							uiSelect(c.Key("backup-interval-select"), &w.backupIntervalSelect, selectProps{
								Options: withCurrentOption(backupIntervalOptions, w.backupIntervalSelect, intervalLabel(parseIntervalOption(w.backupIntervalSelect))),
							}).Label("自动备份频率选择")
							if w.backupIntervalSelect != beforeInt {
								w.SaveBackupPolicy()
							}
						})

						ui.Column(c).Gap(8).Grow(1).Basis(0).Children(func() {
							ui.Text(c, "保留份数").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
							beforeKeep := w.backupKeepSelect
							uiSelect(c.Key("backup-keep-select"), &w.backupKeepSelect, selectProps{
								Options: withCurrentOption(backupKeepOptions, w.backupKeepSelect, "保留最近 "+w.backupKeepSelect+" 份"),
							}).Label("自动备份保留份数选择")
							if w.backupKeepSelect != beforeKeep {
								w.SaveBackupPolicy()
							}
						})
					})

					ui.Column(c.Key("settings-backup-dir")).Gap(8).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "备份目录").FontSize(14).FontWeight(500).TextColor(tk.TextSecondary)
						ui.Row(c).Gap(12).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Children(func() {
								inp := uiInput(c.Key("backup-dir-input"), &w.backupDirInput, inputProps{
									Placeholder: filepath.Join(w.app.DataDir(), "backups"),
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
								Variant: buttonDefault,
							}).Clicked() {
								w.SaveBackupPolicy()
							}
						})
					})

					// Collapsible Managed Backup History (Auto & Pre-Restore Safety Backups)
					if len(w.managedBackups) > 0 {
						ui.Column(c.Key("settings-backup-history")).Gap(8).AlignSelf(ui.Stretch).Children(func() {
							toggleLabel := fmt.Sprintf("历史备份记录 (%d)", len(w.managedBackups))
							toggleBtn := ui.ButtonBase(c.Key("toggle-backup-history")).Padding(4, 0).Gap(6).
								AlignItems(ui.Center).Cursor(ui.CursorPointer).TextColor(tk.TextSecondary).
								Label("切换历史备份记录")
							if toggleBtn.Hovered() {
								toggleBtn.TextColor(tk.Text)
							}
							toggleBtn.Children(func() {
								chev := iconChevronRight()
								if w.backupHistoryExpanded {
									chev = iconChevronDown()
								}
								ui.Icon(c, chev).FontSize(14)
								ui.Text(c, toggleLabel).FontSize(14).FontWeight(500).SingleLine()
							})
							if toggleBtn.Clicked() {
								w.backupHistoryExpanded = !w.backupHistoryExpanded
							}

							if w.backupHistoryExpanded {
								ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
									for i, item := range w.managedBackups {
										b := item
										kindLabel := "自动备份"
										kindColor := tk.TextSecondary
										if b.Kind == "safety" {
											kindLabel = "安全备份"
											kindColor = tk.Primary
										}
										ts := b.CreatedAt.Local().Format("2006-01-02 15:04:05")
										baseName := filepath.Base(b.Path)
										ui.Row(c.Key(fmt.Sprintf("backup-hist-%d", i))).Padding(10, 14).Gap(12).Radius(12).
											Background(tk.FillTertiary).AlignItems(ui.Center).Justify(ui.SpaceBetween).
											AlignSelf(ui.Stretch).Children(func() {
											ui.Row(c).Gap(12).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
												ui.Text(c, kindLabel).FontSize(13).FontWeight(600).TextColor(kindColor).SingleLine().Shrink(0)
												ui.Text(c, ts).Font("monospace").FontSize(13).TextColor(tk.TextSecondary).SingleLine().Shrink(0)
												ui.Text(c, baseName).Font("monospace").FontSize(13).TextColor(tk.Text).Grow(1).SingleLine()
											})
											ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
												if uiButton(c.Key(fmt.Sprintf("reveal-bk-%d", i)), "Finder", buttonProps{
													Variant: buttonText,
													Size:    sizeSmall,
												}).Label("在 Finder 中显示备份 " + baseName).Clicked() {
													if w.host.RevealPath != nil {
														_ = w.host.RevealPath(b.Path)
													} else if w.host.OpenPath != nil {
														_ = w.host.OpenPath(filepath.Dir(b.Path))
													}
												}
												if uiButton(c.Key(fmt.Sprintf("restore-bk-%d", i)), "恢复", buttonProps{
													Variant: buttonText,
													Size:    sizeSmall,
												}).Disabled(w.IsBusy()).Label("恢复备份 " + baseName).Clicked() {
													if w.IsBusy() {
														return
													}
													if err := w.flushPendingOrRevealDraft(); err != nil {
														return
													}
													w.RequestRestoreBackup(b.Path)
												}
											})
										})
									}
								})
							}
						})
					}
				})

				// 4. Manual Backup Export & Full Restore
				ui.Column(c.Key("settings-card-backup-manual")).Gap(16).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "手动备份与恢复").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()

						ui.Row(c.Key("settings-manual-actions")).Gap(12).AlignItems(ui.Center).Children(func() {
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

							if uiButton(c, "导出完整备份 (.zip)", buttonProps{
								Variant: buttonPrimary,
							}).Disabled(w.IsBusy()).Clicked() {
								w.PickAndExportManualBackup()
							}
						})
					})

					if w.lastExportedZipPath != "" {
						uiAlert(c.Key("settings-export-alert"), "已导出备份", alertProps{
							Type:        alertSuccess,
							Description: w.lastExportedZipPath,
						})
					}

					if w.lastRestoreError != "" {
						uiAlert(c.Key("settings-restore-error"), "备份恢复失败", alertProps{
							Type:        alertError,
							Description: w.lastRestoreError,
						})
					} else if w.lastRestoreResult != nil {
						uiAlert(c.Key("settings-restore-alert"), "数据已完整恢复", alertProps{
							Type:        alertSuccess,
							Description: fmt.Sprintf("安全备份：%s", w.lastRestoreResult.SafetyBackupPath),
						})
					}
				})

				// 5. Workspace Modules Visibility
				w.renderSettingsModulesCard(c, tk)

				// 6. Local AI / MCP Integration
				w.renderSettingsMCPCard(c, tk)

				// 7. App Version & Updates
				w.renderSettingsUpdatesCard(c, tk)
			})
		})
	})
}

func (w *Workbench) renderSettingsModulesCard(c *ui.Context, tk *uiTokens) {
	visibleCount := len(w.VisibleModules())
	ui.Column(c.Key("settings-card-modules")).Gap(16).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, "工作区").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()

		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			for _, mod := range w.modules {
				m := mod
				visible := w.IsModuleVisible(m.ID)
				lockedLast := visible && visibleCount <= 1
				row := ui.Row(c.Key("module-toggle-"+m.ID)).Padding(12, 16).Gap(16).Radius(16).
					AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch)
				if row.Hovered() {
					row.Background(tk.FillTertiary)
				}
				row.Children(func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
						uiIconBox(c, ModuleIconSVG(m), tk.TextSecondary)
						ui.Text(c, m.Name).FontSize(16).FontWeight(500).TextColor(tk.Text).SingleLine()
					})

					ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
						if lockedLast {
							ui.Text(c, "至少保留一个模块").FontSize(14).TextColor(tk.TextSecondary).SingleLine()
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

	ui.Column(c.Key("settings-card-mcp")).Gap(16).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "本地接入").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
			if info.Ready {
				ui.Text(c, "aion.sock").Font("monospace").FontSize(14).FontWeight(500).TextColor(tk.Primary).SingleLine()
			} else if info.LastError != "" {
				ui.Text(c, "离线模式").FontSize(14).TextColor(tk.TextSecondary).SingleLine()
			}
		})

		ui.Row(c).Padding(12, 16).Gap(12).Radius(16).Background(tk.FillTertiary).
			AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, cmdPreview).Font("monospace").FontSize(14).TextColor(tk.TextSecondary).Grow(1).SingleLine()
		})

		ui.Row(c).Gap(12).AlignItems(ui.Center).Wrap().Children(func() {
			if uiButton(c, "复制 Antigravity 配置", buttonProps{
				Variant: buttonDefault,
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
	versionLine := "开发版本"
	if displayVersion != "" {
		versionLine = "v" + displayVersion
		if info.IsDev {
			versionLine += " (dev)"
		}
	}
	lastCheckStr := "尚未检查"
	if !info.LastCheck.IsZero() {
		lastCheckStr = info.LastCheck.Local().Format("2006-01-02 15:04:05")
	}

	ui.Column(c.Key("settings-card-updates")).Gap(16).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "更新").FontSize(18).FontWeight(700).TextColor(tk.Text).SingleLine()
				if installedVersion != "" {
					ui.Text(c, "v"+installedVersion+" 待重启").Font("monospace").FontSize(14).FontWeight(500).TextColor(tk.Primary).SingleLine()
				} else if availVersion != "" {
					ui.Text(c, "发现新版本 v"+availVersion).Font("monospace").FontSize(14).FontWeight(500).TextColor(tk.Primary).SingleLine()
				}
			})

			ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Text(c, "自动检查").FontSize(14).TextColor(tk.TextSecondary)
				beforeAuto := w.updateCheckEnabled
				uiSwitch(c.Key("update-auto-check-switch"), &w.updateCheckEnabled).
					Label("自动检查更新开关")
				if w.updateCheckEnabled != beforeAuto {
					w.SaveUpdateCheckPolicy()
				}
			})
		})

		ui.Row(c).Padding(12, 16).Gap(12).Radius(16).Background(tk.FillTertiary).
			AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, versionLine+" · "+lastCheckStr).Font("monospace").FontSize(14).TextColor(tk.TextSecondary).SingleLine()

			ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
				fbBtn := uiButton(c.Key("settings-feedback-btn"), "", buttonProps{
					Variant: buttonText,
					Size:    sizeSmall,
				}).Disabled(!canOpenURL).Label("在设置中提交问题反馈")
				fbBtn.Children(func() {
					ui.Icon(c, iconMessageSquare()).FontSize(14)
					ui.Text(c, "问题反馈").SingleLine()
				})
				if fbBtn.Clicked() {
					w.OpenFeedbackPage()
				}
				if uiButton(c, "前往 Releases 下载", buttonProps{
					Variant: buttonText,
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

		ui.Row(c.Key("settings-update-interval")).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "检查频率").FontSize(14).TextColor(tk.TextSecondary)
			ui.Box(c).Width(200).Children(func() {
				beforeInt := w.updateCheckIntervalSelect
				uiSelect(c.Key("update-check-interval-select"), &w.updateCheckIntervalSelect, selectProps{
					Options: withCurrentOption(updateCheckIntervalOptions, w.updateCheckIntervalSelect, intervalLabel(parseIntervalOption(w.updateCheckIntervalSelect))),
				}).Label("自动检查频率选择").Disabled(!w.updateCheckEnabled)
				if w.updateCheckIntervalSelect != beforeInt {
					w.SaveUpdateCheckPolicy()
				}
			})
		})

		switch {
		case info.IsDev || displayVersion == "":
			uiAlert(c.Key("settings-updates-dev"), "当前为开发构建，不支持应用内自动更新", alertProps{Type: alertInfo})
		case !info.CanSelfUpdate:
			uiAlert(c.Key("settings-updates-readonly"), "当前安装位置不可写，无法应用内自动更新", alertProps{Type: alertWarning})
		case info.Status == UpdateReady || installedVersion != "":
			ver := installedVersion
			if ver == "" {
				ver = availVersion
			}
			uiAlert(c.Key("settings-updates-ready"), "v"+ver+" 已安装，重启生效", alertProps{Type: alertSuccess})
		case info.Status == UpdateError && availVersion != "":
			uiAlert(c.Key("settings-updates-error"), "更新 v"+availVersion+" 失败", alertProps{Type: alertError, Description: info.LastError})
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
