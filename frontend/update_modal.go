package frontend

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

func displayVersionNumber(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func formatMegabytes(bytes int64) string {
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

func hasSidebarUpdateIndicator(info UpdateCapability) bool {
	if info.IsDev || !info.CanSelfUpdate {
		return false
	}
	switch info.Status {
	case UpdateDownloading, UpdateInstalling, UpdateReady:
		return true
	}
	return displayVersionNumber(info.AvailableVersion) != "" || displayVersionNumber(info.InstalledVersion) != ""
}

func sidebarUpdateTooltip(info UpdateCapability) string {
	installedVer := displayVersionNumber(info.InstalledVersion)
	availVer := displayVersionNumber(info.AvailableVersion)
	switch {
	case info.Status == UpdateReady || installedVer != "":
		ver := installedVer
		if ver == "" {
			ver = availVer
		}
		if ver != "" {
			return fmt.Sprintf("v%s 已安装，点击重启更新", ver)
		}
		return "更新已就绪，点击重启更新"
	case info.Status == UpdateInstalling:
		if availVer != "" {
			return fmt.Sprintf("正在安装更新 v%s", availVer)
		}
		return "正在安装更新"
	case info.Status == UpdateDownloading:
		label := "正在下载更新"
		if availVer != "" {
			label = fmt.Sprintf("正在下载更新 v%s", availVer)
		}
		if info.TotalBytes > 0 {
			pct := min(max(int(info.DownloadedBytes*100/info.TotalBytes), 0), 100)
			return fmt.Sprintf("%s (%d%%)", label, pct)
		}
		return label
	case availVer != "":
		return fmt.Sprintf("发现新版本 v%s，点击查看更新", availVer)
	default:
		return "软件更新"
	}
}

func (w *Workbench) renderSidebarUpdateButton(c *ui.Context, tk *uiTokens, info UpdateCapability) {
	tip := sidebarUpdateTooltip(info)
	b := ui.ButtonBase(c.Key("sidebar-update-btn")).Size(26, 26).Radius(13).
		Center().Shrink(0).Cursor(ui.CursorPointer).Transition(buttonTransition).
		Label(tip)

	isProgress := info.Status == UpdateDownloading || info.Status == UpdateInstalling
	state := stateOfButton(b)
	fg, bg := tk.Success, tk.SuccessFillQuaternary
	if isProgress {
		fg, bg = tk.Primary, tk.PrimaryFillQuaternary
	}
	switch state {
	case buttonHover, buttonActive:
		if isProgress {
			bg = tk.PrimaryFillTertiary
		} else {
			bg = tk.SuccessFillTertiary
		}
	}
	b.Background(bg).TextColor(fg)
	b.Children(func() {
		ui.Icon(c, iconCircleArrowUp()).FontSize(16)
	})
	uiTooltip(c, b, tip)
	if b.Clicked() {
		w.OpenUpdateModal()
	}
}

func (w *Workbench) renderUpdateModal(c *ui.Context, tk *uiTokens) {
	m := modalTransition()
	dialogPresent(c.Key("update-modal"), &w.updateModalOpen, motionOfDialog(m), func(backdrop, panel ui.Element, shown, dim float32, closing bool) {
		info := w.UpdateCapability()
		canCheck, canOpenURL := w.updateActionsReady()
		curVer := displayVersionNumber(info.Version)
		availVer := displayVersionNumber(info.AvailableVersion)
		installedVer := displayVersionNumber(info.InstalledVersion)

		backdrop.Padding(32, 16).Background(tk.BgContainer.Alpha(0.6 * dim))
		from := m.Enter.Scale
		if closing {
			from = m.Exit.Scale
		}
		scale := from + (1-from)*shown
		panel.Width(520).MaxWidthPercent(100).MaxHeightPercent(100).Radius(12).Clip().
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Scale(scale, scale).Opacity(shown)
		tk.Elevate(panel, boxShadow)

		panel.Children(func() {
			// Header
			ui.Row(c).MinHeight(50).Padding(10, 56, 10, 16).Gap(8).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
				ui.Icon(c, iconCircleArrowUp()).FontSize(18).TextColor(tk.Primary)
				ui.Text(c, "软件更新").FontSize(15).FontWeight(600).TextColor(tk.Text).SingleLine()
				if installedVer != "" {
					uiTag(c, "v"+installedVer+" 待重启", tagProps{
						Size:    sizeSmall,
						Variant: tagFilled,
						Color:   tagGreen,
					})
				} else if availVer != "" && !info.IsDev && info.CanSelfUpdate {
					uiTag(c, "v"+availVer, tagProps{
						Size:    sizeSmall,
						Variant: tagFilled,
						Color:   tagGreen,
					})
				}
			})

			// Body
			ui.Scroll(c).Shrink(1).Children(func() {
				ui.Column(c).Padding(16).Gap(12).AlignSelf(ui.Stretch).Children(func() {
					switch {
					case info.IsDev || curVer == "":
						uiAlert(c.Key("update-modal-dev"), "当前为开发构建，不支持应用内更新", alertProps{
							Type:        alertInfo,
							Description: "请使用正式打包的 .dmg 安装包运行 Aion，或前往 Releases 页面手动获取最新版本。",
						})
					case !info.CanSelfUpdate:
						uiAlert(c.Key("update-modal-readonly"), "当前安装位置不可写，无法应用内更新", alertProps{
							Type:        alertWarning,
							Description: "请将 Aion.app 移动到当前用户有写入权限的“应用程序”目录后重试，或前往 Releases 页面手动下载。",
						})
					case info.Status == UpdateReady || installedVer != "":
						ver := installedVer
						if ver == "" {
							ver = availVer
						}
						uiAlert(c.Key("update-modal-ready"), "更新已准备就绪", alertProps{
							Type:        alertSuccess,
							Description: fmt.Sprintf("Aion v%s 已安装完成，立即重启即可使用新版本；若选择稍后，将在下次打开应用时生效。", ver),
						})
					case info.Status == UpdateDownloading:
						title := "正在下载更新…"
						if availVer != "" {
							title = fmt.Sprintf("正在下载 Aion v%s…", availVer)
						}
						ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(tk.Text)
						progress := -1.0
						if info.TotalBytes > 0 {
							progress = min(max(float64(info.DownloadedBytes)/float64(info.TotalBytes), 0), 1)
						}
						ui.Progress(c.Key("update-download-progress"), progress).AlignSelf(ui.Stretch)
						if info.TotalBytes > 0 {
							ui.Text(c, fmt.Sprintf("%s / %s", formatMegabytes(info.DownloadedBytes), formatMegabytes(info.TotalBytes))).
								FontSize(12).TextColor(tk.TextSecondary)
						}
						ui.Text(c, "关闭此窗口不会中断下载，完成后可从侧边栏或此处重启应用。").
							FontSize(12).TextColor(tk.TextTertiary)
					case info.Status == UpdateInstalling:
						title := "正在安装更新…"
						if availVer != "" {
							title = fmt.Sprintf("正在安装 Aion v%s…", availVer)
						}
						ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(tk.Text)
						ui.Progress(c.Key("update-install-progress"), -1).AlignSelf(ui.Stretch)
						ui.Text(c, "正在校验签名并准备新版本，请稍候…").
							FontSize(12).TextColor(tk.TextSecondary)
					case info.Status == UpdateChecking:
						ui.Text(c, "正在检查更新…").FontSize(14).FontWeight(600).TextColor(tk.Text)
						ui.Progress(c.Key("update-check-progress"), -1).AlignSelf(ui.Stretch)
						ui.Text(c, fmt.Sprintf("当前版本: v%s", curVer)).FontSize(12).TextColor(tk.TextSecondary)
					case info.Status == UpdateError:
						desc := strings.TrimSpace(info.LastError)
						if desc == "" {
							desc = "请检查网络连接后重试，或前往 Releases 页面手动下载。"
						}
						uiAlert(c.Key("update-modal-error"), "更新操作未能完成", alertProps{
							Type:        alertError,
							Description: desc,
						})
					case availVer != "":
						ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
							ui.Text(c, fmt.Sprintf("Aion v%s 现已推出", availVer)).
								FontSize(15).FontWeight(600).TextColor(tk.Text)
							ui.Text(c, fmt.Sprintf("当前版本 v%s → 新版本 v%s", curVer, availVer)).
								FontSize(12).TextColor(tk.TextSecondary)
						})
						notes := strings.TrimSpace(info.AvailableNotes)
						if notes != "" {
							ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, "更新说明").FontSize(12).FontWeight(600).TextColor(tk.TextSecondary)
								ui.Scroll(c.Key("update-notes-scroll")).MaxHeight(240).AlignSelf(ui.Stretch).
									Radius(tk.BorderRadius).Background(tk.FillQuaternary).
									Border(1, tk.BorderSecondary).Children(func() {
									ui.Column(c).Padding(12).AlignSelf(ui.Stretch).Children(func() {
										uiMarkdown(c, notes)
									})
								})
							})
						}
					default:
						uiAlert(c.Key("update-modal-uptodate"), "当前已是最新版本", alertProps{
							Type:        alertSuccess,
							Description: fmt.Sprintf("Aion v%s 是目前可用的最新版本。", curVer),
						})
					}
				})
			})

			// Footer
			ui.Row(c).Gap(8).Padding(12, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
				// Left auxiliary action
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if !info.IsDev && info.CanSelfUpdate && availVer != "" &&
						info.Status != UpdateDownloading && info.Status != UpdateInstalling &&
						info.Status != UpdateReady && installedVer == "" &&
						info.Status != UpdateChecking && info.Status != UpdateError {
						if uiButton(c.Key("update-skip-btn"), "跳过此版本", buttonProps{
							Variant: buttonText,
						}).Disabled(!w.canSkipUpdate()).Clicked() {
							w.SkipUpdateVersion(info.AvailableVersion)
						}
					} else if info.IsDev || curVer == "" || !info.CanSelfUpdate || info.Status == UpdateError {
						if uiButton(c.Key("update-modal-releases-btn"), "前往 Releases 下载", buttonProps{
							Variant: buttonDefault,
						}).Disabled(!canOpenURL).Clicked() {
							w.OpenReleasesPage()
						}
					}
				})

				// Right primary actions
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					switch {
					case info.IsDev || curVer == "" || !info.CanSelfUpdate:
						if uiButton(c.Key("update-close-btn"), "关闭", buttonProps{
							Variant: buttonPrimary,
						}).Clicked() {
							w.CloseUpdateModal()
						}
					case info.Status == UpdateReady || installedVer != "":
						if uiButton(c.Key("update-later-relaunch-btn"), "稍后", buttonProps{
							Variant: buttonDefault,
						}).Clicked() {
							w.CloseUpdateModal()
						}
						if uiButton(c.Key("update-relaunch-btn"), "立即重启", buttonProps{
							Variant: buttonPrimary,
						}).Disabled(!w.canRelaunchUpdate()).Clicked() {
							w.RelaunchToUpdate()
						}
					case info.Status == UpdateDownloading:
						if uiButton(c.Key("update-cancel-download-btn"), "取消下载", buttonProps{
							Variant: buttonDefault,
						}).Disabled(!w.canCancelUpdate()).Clicked() {
							w.CancelUpdate()
						}
						if uiButton(c.Key("update-bg-download-btn"), "后台下载", buttonProps{
							Variant: buttonPrimary,
						}).Clicked() {
							w.CloseUpdateModal()
						}
					case info.Status == UpdateInstalling:
						if uiButton(c.Key("update-bg-install-btn"), "后台继续", buttonProps{
							Variant: buttonPrimary,
						}).Clicked() {
							w.CloseUpdateModal()
						}
					case info.Status == UpdateChecking:
						if uiButton(c.Key("update-cancel-check-btn"), "取消", buttonProps{
							Variant: buttonDefault,
						}).Clicked() {
							w.CancelUpdate()
							w.CloseUpdateModal()
						}
					case info.Status == UpdateError:
						if uiButton(c.Key("update-retry-btn"), "重试", buttonProps{
							Variant: buttonDefault,
						}).Disabled(!canCheck).Clicked() {
							w.CheckForUpdates()
						}
						if uiButton(c.Key("update-error-close-btn"), "关闭", buttonProps{
							Variant: buttonPrimary,
						}).Clicked() {
							w.CloseUpdateModal()
						}
					case availVer != "":
						if uiButton(c.Key("update-remind-later-btn"), "稍后提醒", buttonProps{
							Variant: buttonDefault,
						}).Clicked() {
							w.CloseUpdateModal()
						}
						if uiButton(c.Key("update-install-btn"), "下载并安装", buttonProps{
							Variant: buttonPrimary,
						}).Disabled(!w.canInstallUpdate()).Clicked() {
							w.InstallUpdate()
						}
					default:
						if uiButton(c.Key("update-recheck-btn"), "重新检查", buttonProps{
							Variant: buttonDefault,
						}).Disabled(!canCheck).Clicked() {
							w.CheckForUpdates()
						}
						if uiButton(c.Key("update-ok-btn"), "好", buttonProps{
							Variant: buttonPrimary,
						}).Clicked() {
							w.CloseUpdateModal()
						}
					}
				})
			})

			if dialogCloseButton(c, tk, 18).Absolute().Top(9).Right(12).Radius(12).Clicked() {
				w.CloseUpdateModal()
			}
		})
	})
}
