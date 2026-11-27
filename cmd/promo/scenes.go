package main

import (
	"math"

	"github.com/egoist/mygo/plugins/render"
	"github.com/egoist/mygo/ui"
)

// Apple-inspired minimalist color system: pure monochrome surfaces, hairline borders,
// refined contrast hierarchy, and restrained brand accents.
var (
	colBlack         = ui.Hex("#000000")
	colDarkSurface   = ui.Hex("#161617")
	colDarkCard      = ui.Hex("#1D1D1F")
	colWhiteStudio   = ui.Hex("#F5F5F7")
	colAppleText     = ui.Hex("#1D1D1F")
	colAppleSub      = ui.Hex("#6E6E73")
	colAppleLightTxt = ui.Hex("#F5F5F7")
	colAppleDarkSub  = ui.Hex("#86868B")
	colAppleBlue     = ui.Hex("#0071E3")
	colBrandSage     = ui.Hex("#A8D96C")
)

func buildComposition(assets *demoAssets, sfx *soundBank) render.Composition {
	return render.Composition{
		Width:      videoWidth,
		Height:     videoHeight,
		FPS:        videoFPS,
		Frames:     totalFrames,
		Scale:      videoScale,
		Dark:       true,
		Background: colBlack,
		Locale:     "zh-Hans",
		View: func(c *ui.Context, f render.Frame) {
			// Studio-mastered stereo soundtrack at full presence across the 60s film.
			f.Audio(sfx.Score, render.AudioOptions{Volume: 1.0})

			// Calm, unhurried Apple Keynote pacing (60 seconds = 3600 frames @ 60fps).
			// Overlapping 36-frame (0.6s) crossfades between acts.
			f.Sequence(c, 0, 360, func(f render.Frame) { // 0.0s – 6.0s
				act1BrandOpening(c, f, assets, sfx)
			})
			f.Sequence(c, 324, 432, func(f render.Frame) { // 5.4s – 12.6s (7.2s)
				act2TodayFocus(c, f, assets, sfx)
			})
			f.Sequence(c, 720, 696, func(f render.Frame) { // 12.0s – 23.6s (11.6s: Kanban + Task Detail)
				act3KanbanAndDrawer(c, f, assets, sfx)
			})
			f.Sequence(c, 1380, 660, func(f render.Frame) { // 23.0s – 34.0s (11.0s: Jobs + Timeline)
				act4JobsAndTimeline(c, f, assets, sfx)
			})
			f.Sequence(c, 2004, 636, func(f render.Frame) { // 33.4s – 44.0s (10.6s: Dark Sweep + ⌘K)
				act5DarkSweepAndPalette(c, f, assets, sfx)
			})
			f.Sequence(c, 2604, 540, func(f render.Frame) { // 43.4s – 52.4s (9.0s: Local MCP & CLI)
				act6LocalMCPAndCLI(c, f, assets, sfx)
			})
			f.Sequence(c, 3108, 288, func(f render.Frame) { // 51.8s – 56.6s (4.8s: 4 Minimal Pillars)
				act7ApplePillars(c, f, sfx)
			})
			f.Sequence(c, 3360, 240, func(f render.Frame) { // 56.0s – 60.0s (4.0s: Finale)
				act8FinaleOutro(c, f, assets, sfx)
			})
		},
	}
}

// -----------------------------------------------------------------------------
// Act 1: Pure Apple-Style Brand Opening (0.0s – 6.0s, 360 frames)
// Minimal, serene, zero clutter: Icon + "Aion" + one concise phrase.
// -----------------------------------------------------------------------------
func act1BrandOpening(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 24 {
		f.Audio(sfx.ChimeMid, render.AudioOptions{Volume: 0.48})
	}

	fadeOut := f.Interpolate([]float64{320, 359}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	iconIn := f.Interpolate([]float64{14, 72}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	titleIn := f.Interpolate([]float64{48, 102}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	lineIn := f.Interpolate([]float64{92, 148}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	driftScale := float32(f.Interpolate([]float64{0, 360}, []float64{0.97, 1.02}))

	ui.Box(c).Fill().Background(colBlack).Opacity(float32(fadeOut)).Children(func() {
		// Whisper-subtle organic light behind the icon (pure Apple restraint)
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.44},
				Radius: 300,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(168, 217, 108, float32(0.10*iconIn)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(18).
			Scale(driftScale, driftScale).
			Children(func() {
				// Clean icon tile
				iScale := float32(0.86 + 0.14*iconIn)
				ui.Box(c).Size(104, 104).Center().Radius(26).
					Border(1, ui.RGBA(255, 255, 255, 0.12)).
					Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.75)).
					Scale(iScale, iScale).Opacity(float32(iconIn)).
					Children(func() {
						ui.Image(c, assets.LogoSVG).Size(102, 102).Radius(25)
					})

				// Iconic single wordmark
				ui.Text(c, "Aion").FontSize(68).Bold().LetterSpacing(-2.2).TextColor(colAppleLightTxt).
					Opacity(float32(titleIn)).Translate(0, float32(16*(1-titleIn)))

				// Single razor-sharp tagline — nothing more
				ui.Text(c, "项目与求职，本地原生。").FontSize(22).TextColor(colAppleDarkSub).
					Opacity(float32(lineIn)).Translate(0, float32(12*(1-lineIn)))
			})
	})
}

// -----------------------------------------------------------------------------
// Act 2: Today Focus (5.4s – 12.6s, 432 frames)
// Large, unobstructed centered window on pure Apple #F5F5F7 studio.
// -----------------------------------------------------------------------------
func act2TodayFocus(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 8 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	fadeIn := f.Interpolate([]float64{0, 36}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{396, 431}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	enter := f.Interpolate([]float64{8, 74}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	breathScale := f.Interpolate([]float64{0, 432}, []float64{0.985, 1.012})
	tiltX := float32(6.0 * (1.0 - enter))

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		renderAppleHeadline(c, f, "今天。", "待办、求职与状态，一眼掌控。", false)

		// Commanding centered Retina window (704×440 = exact 1.6:1)
		s := float32((0.95 + 0.05*enter) * breathScale)
		ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1400).RotateX(tiltX).Scale(s, s).
			Translate(0, float32(28*(1-enter))).
			Opacity(float32(enter)).
			Children(func() {
				renderAppleWindow(c, assets.FocusLight, 704, 440, false, true)
			})
	})
}

// -----------------------------------------------------------------------------
// Act 3: Kanban Board -> Task Detail Drawer (12.0s – 23.6s, 696 frames)
// ~5.5s on Kanban Board, calm 0.8s morph, ~5.3s on Task Detail Drawer.
// -----------------------------------------------------------------------------
func act3KanbanAndDrawer(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 8 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	if f.Index == 336 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.38})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.25})
	}

	fadeIn := f.Interpolate([]float64{0, 36}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{660, 695}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	enter := f.Interpolate([]float64{6, 66}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	drawerBlend := f.Interpolate([]float64{330, 380}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	slowDrift := f.Interpolate([]float64{0, 696}, []float64{0.99, 1.015})

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		// Smooth crossfading Apple headline
		if drawerBlend < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32(1 - drawerBlend*2)).Children(func() {
				renderAppleHeadline(c, f, "三列看板。", "拖拽排序，多维筛选。", false)
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32((drawerBlend - 0.5) * 2)).Children(func() {
				renderAppleHeadlineStatic(c, "任务详情。", "Markdown、子任务与本地附件。", false)
			})
		}

		// Centered hero window (704×440)
		s := float32((0.97 + 0.03*enter) * slowDrift)
		ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Scale(s, s).Translate(0, float32(22*(1-enter))).
			Opacity(float32(enter)).
			Children(func() {
				renderAppleWindow(c, assets.BoardLight, 704, 440, false, true)
				if drawerBlend > 0.005 {
					ui.Box(c).Absolute().Left(0).Top(0).Size(704, 440).
						Opacity(float32(drawerBlend)).
						Translate(float32(18*(1-drawerBlend)), 0).
						Children(func() {
							renderAppleWindow(c, assets.TaskDetail, 704, 440, false, false)
						})
				}
			})
	})
}

// -----------------------------------------------------------------------------
// Act 4: Job Tracker -> Stage Timeline (23.0s – 34.0s, 660 frames)
// ~5.2s on Jobs table, smooth reveal, ~5.2s on full Stage Timeline modal.
// -----------------------------------------------------------------------------
func act4JobsAndTimeline(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 8 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	if f.Index == 314 {
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.32})
	}

	fadeIn := f.Interpolate([]float64{0, 36}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{624, 659}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	enter := f.Interpolate([]float64{6, 64}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	modalBlend := f.Interpolate([]float64{308, 356}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	slowDrift := f.Interpolate([]float64{0, 660}, []float64{0.99, 1.015})

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		if modalBlend < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32(1 - modalBlend*2)).Children(func() {
				renderAppleHeadline(c, f, "求职追踪。", "投递进程与 Offer 转化，清晰有序。", false)
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32((modalBlend - 0.5) * 2)).Children(func() {
				renderAppleHeadlineStatic(c, "复盘时间线。", "记录每轮细节，支持历史节点补录。", false)
			})
		}

		// Centered hero window (704×440)
		s := float32((0.97 + 0.03*enter) * slowDrift)
		ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Scale(s, s).Translate(0, float32(22*(1-enter))).
			Opacity(float32(enter)).
			Children(func() {
				renderAppleWindow(c, assets.JobsLight, 704, 440, false, true)
				if modalBlend > 0.005 {
					ms := float32(0.97 + 0.03*modalBlend)
					ui.Box(c).Absolute().Left(0).Top(0).Size(704, 440).
						Scale(ms, ms).Opacity(float32(modalBlend)).
						Children(func() {
							renderAppleWindow(c, assets.JobDetail, 704, 440, false, true)
						})
				}
			})
	})
}

// -----------------------------------------------------------------------------
// Act 5: Light -> Dark Apple Wipe & ⌘K Command Palette (33.4s – 44.0s, 636 frames)
// Unhurried: 1.2s Light -> 2.0s silky wipe -> 1.6s Dark -> ⌘K press -> 4.6s Palette.
// -----------------------------------------------------------------------------
func act5DarkSweepAndPalette(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 74 {
		f.Audio(sfx.WhooshSweep, render.AudioOptions{Volume: 0.45})
	}
	if f.Index == 284 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.65})
	}
	if f.Index == 296 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.72})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.30})
	}

	fadeIn := f.Interpolate([]float64{0, 34}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{600, 635}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	// Silky 2-second sweep from Light -> Dark (frames 70..190)
	sweep := f.Interpolate([]float64{70, 190}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	bgCol := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colWhiteStudio, colBlack})
	primaryTxt := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colAppleText, colAppleLightTxt})
	subTxt := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colAppleSub, colAppleDarkSub})

	// Command Palette fades & settles smoothly at frames 302..342 (holds ~4.5s!)
	palBlend := f.Interpolate([]float64{302, 342}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	ui.Box(c).Fill().Background(bgCol).Opacity(sceneOp).Children(func() {
		// Single clean Apple-style inline headline
		primary := "深浅外观。"
		secondary := "昼夜切换，始终专注。"
		if f.Index >= 294 {
			primary = "⌘K 全局直达。"
			secondary = "瞬时检索活跃与归档记录。"
		}
		ui.Row(c).Absolute().Left(0).Top(28).Right(0).Justify(ui.Center).AlignItems(ui.Baseline).Gap(4).Children(func() {
			ui.Text(c, primary).FontSize(23).Bold().TextColor(primaryTxt)
			ui.Text(c, secondary).FontSize(23).TextColor(subTxt)
		})

		// Stationary centered 704×440 window (zero 3D clip -> 250+ FPS on Metal!)
		const winX, winY, winW, winH = 128, 78, 704, 440
		ui.Box(c).Absolute().Left(winX).Top(winY).Size(winW, winH).Children(func() {
			renderAppleWindow(c, assets.BoardLight, winW, winH, false, true)

			if sweep > 0.002 {
				rightInset := float32((1.0 - sweep) * 100.0)
				ui.Box(c).Absolute().Left(0).Top(0).Size(winW, winH).
					ClipInsetPercent(0, rightInset, 0, 0, 14).
					Children(func() {
						renderAppleWindow(c, assets.BoardDark, winW, winH, true, true)
					})
			}

			// Minimal Apple silver-white hairline edge sheen (clean, no neon laser)
			if sweep > 0.01 && sweep < 0.99 {
				beamX := float32(sweep * winW)
				beamAlpha := float32(math.Sin(math.Pi * sweep))
				ui.Box(c).Absolute().Left(beamX - 10).Top(0).Size(20, winH).Opacity(beamAlpha).
					LinearGradient(ui.LinearGradient{
						Angle: 90,
						Stops: []ui.GradientStop{
							{Color: ui.Transparent, At: 0},
							{Color: ui.RGBA(255, 255, 255, 0.55), At: 0.5},
							{Color: ui.Transparent, At: 1},
						},
					})
			}

			if palBlend > 0.005 {
				ps := float32(0.97 + 0.03*palBlend)
				ui.Box(c).Absolute().Left(0).Top(0).Size(winW, winH).
					Scale(ps, ps).Opacity(float32(palBlend)).
					Children(func() {
						renderAppleWindow(c, assets.PaletteDark, winW, winH, true, true)
					})
			}
		})

		// Quiet Apple-style monochrome ⌘ + K keycap hint (frames 252..340)
		if f.Index >= 252 && f.Index <= 346 {
			kIn := f.Interpolate([]float64{252, 274}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
			kOut := f.Interpolate([]float64{320, 346}, []float64{1, 0})
			pressCmd := f.Interpolate([]float64{282, 287, 300}, []float64{1, 0.92, 1})
			pressK := f.Interpolate([]float64{294, 299, 312}, []float64{1, 0.92, 1})
			ui.Row(c).Absolute().Left(408).Top(452).Gap(10).AlignItems(ui.Center).
				Opacity(float32(kIn * kOut)).Translate(0, float32(10*(1-kIn))).
				Children(func() {
					renderAppleKeyCap(c, "⌘", float32(pressCmd), f.Index >= 283 && f.Index <= 304)
					ui.Text(c, "+").FontSize(14).TextColor(colAppleDarkSub)
					renderAppleKeyCap(c, "K", float32(pressK), f.Index >= 295 && f.Index <= 316)
				})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 6: Local AI / MCP & CLI (43.4s – 52.4s, 540 frames)
// Unhurried, crisp monochrome dark studio: Terminal on left, real Dark Settings on right.
// -----------------------------------------------------------------------------
func act6LocalMCPAndCLI(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 8 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	if f.Index >= 40 && f.Index <= 112 && (f.Index-40)%8 == 0 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.22})
	}
	if f.Index == 232 || f.Index == 356 {
		f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.32})
	}

	fadeIn := f.Interpolate([]float64{0, 34}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{504, 539}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	enter := f.Interpolate([]float64{10, 68}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	ui.Box(c).Fill().Background(colBlack).Opacity(sceneOp).Children(func() {
		renderAppleHeadline(c, f, "本地 MCP 与 CLI。", "AI 实时协同，数据不离设备。", true)

		// Left: Minimal dark Terminal card (x = 46..458, w = 412, h = 406)
		ui.Box(c).Absolute().Left(46).Top(86).Size(412, 406).
			Translate(0, float32(20*(1-enter))).Opacity(float32(enter)).
			Children(func() {
				renderMinimalTerminal(c, f, 412, 406)
			})

		// Subtle monochrome connection indicator in center gap (x = 458..502)
		connOp := float32(f.Interpolate([]float64{110, 150}, []float64{0, 1}))
		ui.Column(c).Absolute().Left(464).Top(270).Width(32).AlignItems(ui.Center).Gap(6).Opacity(connOp).Children(func() {
			dotPulse := float32(0.45 + 0.55*math.Sin(float64(f.Index)*0.09))
			ui.Box(c).Size(6, 6).Radius(999).Background(ui.RGBA(168, 217, 108, dotPulse))
			ui.Text(c, "⇄").FontSize(15).TextColor(colAppleDarkSub)
		})

		// Right: Real Aion Dark Settings (Backup + Local MCP, x = 502..914, w = 412, h = 406)
		ui.Box(c).Absolute().Left(502).Top(86).Size(412, 406).
			Translate(0, float32(20*(1-enter))).Opacity(float32(enter)).
			Children(func() {
				renderAppleWindow(c, assets.SettingsDark, 412, 406, true, true)
			})
	})
}

func renderMinimalTerminal(c *ui.Context, f render.Frame, w, h float32) {
	cmdText := "aion summary --json"
	typed := int(f.Interpolate([]float64{38, 118}, []float64{0, float64(len(cmdText))}))
	if typed < 0 {
		typed = 0
	}
	if typed > len(cmdText) {
		typed = len(cmdText)
	}
	cursor := ""
	if f.Index < 140 && (f.Index/20)%2 == 0 {
		cursor = "▍"
	}

	showJSON := f.Interpolate([]float64{134, 164}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	showMCP := f.Interpolate([]float64{220, 256}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	showSync := f.Interpolate([]float64{344, 380}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	ui.Column(c).Size(w, h).Radius(14).Background(colDarkSurface).
		Border(1, ui.RGBA(255, 255, 255, 0.11)).
		Shadow(0, 24, 56, 0, ui.RGBA(0, 0, 0, 0.65)).
		Children(func() {
			// Quiet macOS titlebar
			ui.Row(c).FillWidth().Height(34).Padding(0, 14).AlignItems(ui.Center).Gap(6.5).
				Children(func() {
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FF5F57"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FEBC2E"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#28C840"))
					ui.Spacer(c)
					ui.Text(c, "aion mcp").FontSize(10.5).TextColor(colAppleDarkSub)
				})

			ui.Box(c).FillWidth().Height(1).Background(ui.RGBA(255, 255, 255, 0.06))

			// Spacious readable body
			ui.Column(c).FillWidth().Grow(1).Padding(20, 20).Gap(14).Children(func() {
				// Live typed command
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "$").FontSize(12.5).Bold().TextColor(colBrandSage)
					ui.Text(c, cmdText[:typed]+cursor).FontSize(12.5).Bold().TextColor(colAppleLightTxt)
				})

				if showJSON > 0.01 {
					ui.Column(c).FillWidth().Padding(12, 14).Radius(10).Gap(4).
						Background(colDarkCard).
						Opacity(float32(showJSON)).
						Children(func() {
							ui.Text(c, `{ "projects": 2, "in_progress": 3,`).FontSize(11).TextColor(colAppleLightTxt)
							ui.Text(c, `  "jobs_active": 4, "offers": 1,`).FontSize(11).TextColor(colAppleLightTxt)
							ui.Text(c, `  "socket": "aion.sock" }`).FontSize(11).TextColor(colBrandSage)
						})
				}

				if showMCP > 0.01 {
					ui.Column(c).FillWidth().Gap(6).
						Opacity(float32(showMCP)).Translate(0, float32(10*(1-showMCP))).
						Children(func() {
							ui.Text(c, "Antigravity / Codex · MCP Tool").FontSize(10.5).TextColor(colAppleDarkSub)
							ui.Column(c).FillWidth().Padding(12, 14).Radius(10).Gap(4).
								Background(colDarkCard).
								Border(1, ui.RGBA(255, 255, 255, 0.08)).
								Children(func() {
									ui.Text(c, `task_create({ project: "Aion 1.0",`).FontSize(11).TextColor(colAppleLightTxt)
									ui.Text(c, `  title: "发布前全量回归检查",`).FontSize(11).TextColor(colAppleLightTxt)
									ui.Text(c, `  priority: "high" })`).FontSize(11).TextColor(ui.Hex("#64D2FF"))
								})
						})
				}

				if showSync > 0.01 {
					ui.Row(c).FillWidth().Padding(11, 14).Radius(10).AlignItems(ui.Center).Gap(10).
						Background(ui.RGBA(168, 217, 108, 0.10)).
						Border(1, ui.RGBA(168, 217, 108, 0.24)).
						Opacity(float32(showSync)).Translate(0, float32(8*(1-showSync))).
						Children(func() {
							ui.Text(c, "✓").FontSize(13).Bold().TextColor(colBrandSage)
							ui.Column(c).Gap(2).Children(func() {
								ui.Text(c, "已通过 Unix Socket 实时同步桌面端").FontSize(11).Bold().TextColor(colAppleLightTxt)
								ui.Text(c, "自动保护未保存草稿 · 隔离破坏性操作").FontSize(10).TextColor(colAppleDarkSub)
							})
						})
				}
			})
		})
}

// -----------------------------------------------------------------------------
// Act 7: Apple Minimal 2×2 Pillars (51.8s – 56.6s, 288 frames)
// Ultra-clean typographic cards — generous negative space, effortless to read.
// -----------------------------------------------------------------------------
func act7ApplePillars(c *ui.Context, f render.Frame, sfx *soundBank) {
	if f.Index == 8 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	fadeIn := f.Interpolate([]float64{0, 32}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{252, 287}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	type pillar struct {
		hero   string
		sub    string
		accent ui.Color
	}
	items := []pillar{
		{"纯原生构建", "基于 MyGo 原生引擎，无 Chromium 与 WebView，macOS 与 Windows 轻量流畅。", colAppleLightTxt},
		{"100% 本地优先", "SQLite 事务落盘，零账号、零云端依赖，断网依然毫秒响应。", colBrandSage},
		{"安心数据兜底", "统一回收站、自动轮转备份与全量 .zip 迁移，随时可恢复。", colAppleLightTxt},
		{"由 MyGo Render 渲染", "包括你正在看的这段 60fps 宣传短片，同样由纯 Go 在 GPU 上直接渲染。", ui.Hex("#64D2FF")},
	}

	ui.Box(c).Fill().Background(colBlack).Opacity(sceneOp).Children(func() {
		renderAppleHeadline(c, f, "为日常而生。", "轻量、可靠、完全属于你。", true)

		const cardW, cardH, gapX, gapY = 408, 178, 24, 24
		const startX, startY = 60, 108
		for i, item := range items {
			row := i / 2
			col := i % 2
			x := float32(startX + col*(cardW+gapX))
			y := float32(startY + row*(cardH+gapY))

			delay := 18 + i*18
			if f.Index == delay {
				f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.24})
			}
			p := f.Interpolate([]float64{float64(delay), float64(delay + 34)}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

			ui.Column(c).Absolute().Left(x).Top(y).Size(cardW, cardH).
				Justify(ui.Center).Padding(26, 30).Gap(10).Radius(20).
				Background(colDarkSurface).
				Border(1, ui.RGBA(255, 255, 255, 0.08)).
				Opacity(float32(p)).Translate(0, float32(16*(1-p))).
				Children(func() {
					ui.Text(c, item.hero).FontSize(24).Bold().TextColor(item.accent)
					ui.Text(c, item.sub).FontSize(13).LineHeight(1.55).TextColor(colAppleDarkSub)
				})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 8: Minimal Apple Finale (56.0s – 60.0s, 240 frames)
// -----------------------------------------------------------------------------
func act8FinaleOutro(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 14 {
		f.Audio(sfx.FinaleChord, render.AudioOptions{Volume: 0.65})
	}

	fadeIn := f.Interpolate([]float64{0, 34}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{198, 239}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	logoIn := f.Interpolate([]float64{8, 56}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	textIn := f.Interpolate([]float64{28, 76}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	footIn := f.Interpolate([]float64{64, 108}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	ui.Box(c).Fill().Background(colBlack).Opacity(sceneOp).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.44},
				Radius: 310,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(168, 217, 108, 0.11), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(16).Children(func() {
			s := float32(0.88 + 0.12*logoIn)
			ui.Box(c).Size(94, 94).Center().Radius(24).
				Border(1, ui.RGBA(255, 255, 255, 0.12)).
				Scale(s, s).Opacity(float32(logoIn)).
				Children(func() {
					ui.Image(c, assets.LogoSVG).Size(92, 92).Radius(23)
				})

			ui.Column(c).AlignItems(ui.Center).Gap(8).
				Opacity(float32(textIn)).Translate(0, float32(14*(1-textIn))).
				Children(func() {
					ui.Text(c, "Aion").FontSize(56).Bold().LetterSpacing(-2).TextColor(colAppleLightTxt)
					ui.Text(c, "项目与求职，本地原生。").FontSize(19).TextColor(colAppleDarkSub)
				})

			ui.Text(c, "macOS · Windows · github.com/Games55k/Aion").
				FontSize(12).LetterSpacing(0.6).TextColor(colAppleDarkSub).
				Margin(14, 0, 0, 0).Opacity(float32(footIn * 0.85))
		})
	})
}

// -----------------------------------------------------------------------------
// Apple-Minimal Shared Primitives
// -----------------------------------------------------------------------------

// renderAppleHeadline draws Apple's iconic two-tone single-line editorial headline:
// crisp bold primary phrase + quiet muted secondary phrase, centered at the top.
func renderAppleHeadline(c *ui.Context, f render.Frame, primary, secondary string, dark bool) {
	p := f.Interpolate([]float64{6, 40}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	ui.Row(c).Absolute().Left(0).Top(28).Right(0).Justify(ui.Center).AlignItems(ui.Baseline).Gap(4).
		Opacity(float32(p)).Translate(0, float32(10*(1-p))).
		Children(func() {
			renderAppleHeadlineText(c, primary, secondary, dark)
		})
}

func renderAppleHeadlineStatic(c *ui.Context, primary, secondary string, dark bool) {
	ui.Row(c).Absolute().Left(0).Top(28).Right(0).Justify(ui.Center).AlignItems(ui.Baseline).Gap(4).
		Children(func() {
			renderAppleHeadlineText(c, primary, secondary, dark)
		})
}

func renderAppleHeadlineText(c *ui.Context, primary, secondary string, dark bool) {
	pCol, sCol := colAppleText, colAppleSub
	if dark {
		pCol, sCol = colAppleLightTxt, colAppleDarkSub
	}
	ui.Text(c, primary).FontSize(23).Bold().TextColor(pCol)
	ui.Text(c, secondary).FontSize(23).TextColor(sCol)
}

func renderAppleWindow(c *ui.Context, bmp *ui.Bitmap, w, h float32, dark, showLights bool) {
	borderCol := ui.RGBA(0, 0, 0, 0.08)
	shadowCol := ui.RGBA(0, 0, 0, 0.14)
	bg := ui.Hex("#FFFFFF")
	if dark {
		borderCol = ui.RGBA(255, 255, 255, 0.13)
		shadowCol = ui.RGBA(0, 0, 0, 0.65)
		bg = colDarkSurface
	}

	ui.Box(c).Size(w, h).Radius(14).Background(bg).
		Border(1, borderCol).
		Shadow(0, 24, 56, 0, shadowCol).
		Children(func() {
			ui.Image(c, bmp).Size(w-2, h-2).Radius(13).Fit(ui.Cover)
			if showLights {
				ui.Row(c).Absolute().Left(13).Top(12).Gap(6).Children(func() {
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FF5F57"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FEBC2E"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#28C840"))
				})
			}
		})
}

func renderAppleKeyCap(c *ui.Context, label string, scale float32, active bool) {
	bg := colDarkCard
	border := ui.RGBA(255, 255, 255, 0.18)
	txt := colAppleLightTxt
	if active {
		bg = colAppleBlue
		border = ui.RGBA(255, 255, 255, 0.35)
		txt = ui.Hex("#FFFFFF")
	}
	ui.Box(c).Size(48, 42).Center().Radius(10).
		Background(bg).Border(1, border).
		Shadow(0, 8, 22, 0, ui.RGBA(0, 0, 0, 0.45)).
		Scale(scale, scale).
		Children(func() {
			ui.Text(c, label).FontSize(18).Bold().TextColor(txt)
		})
}
