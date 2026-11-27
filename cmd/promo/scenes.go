package main

import (
	"math"

	"github.com/egoist/mygo/plugins/render"
	"github.com/egoist/mygo/ui"
)

// Apple-inspired minimalist color system with refined obsidian/botanical accents
// for the flowing local MCP showcase.
var (
	colBlack         = ui.Hex("#000000")
	colObsidianDeep  = ui.Hex("#080A0C")
	colDarkSurface   = ui.Hex("#14161A")
	colDarkCard      = ui.Hex("#1D1D1F")
	colWhiteStudio   = ui.Hex("#F5F5F7")
	colAppleText     = ui.Hex("#1D1D1F")
	colAppleSub      = ui.Hex("#6E6E73")
	colAppleLightTxt = ui.Hex("#F5F5F7")
	colAppleDarkSub  = ui.Hex("#86868B")
	colAppleBlue     = ui.Hex("#0071E3")
	colSkyGlow       = ui.Hex("#38BDF8")
	colBrandSage     = ui.Hex("#A8D96C")
	colBrandLime     = ui.Hex("#B4E451")
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
			// Studio-mastered stereo soundtrack (~ -14 dBFS RMS) across the 54s film.
			f.Audio(sfx.Score, render.AudioOptions{Volume: 1.0})

			// Dynamic editorial pacing (54 seconds = 3240 frames @ 60fps):
			// brisk on low-density beats (Opening, Wipe, Pillars, Finale),
			// lingering on dense real-UI scenes (Kanban+Detail, BAT+Mag-7 Jobs+Timeline, MCP).
			f.Sequence(c, 0, 216, func(f render.Frame) { // 0.0s – 3.6s (Brisk Opening)
				act1BrandOpening(c, f, assets, sfx)
			})
			f.Sequence(c, 192, 372, func(f render.Frame) { // 3.2s – 9.4s (6.2s Today Focus + 3D Reveal)
				act2TodayFocus(c, f, assets, sfx)
			})
			f.Sequence(c, 540, 684, func(f render.Frame) { // 9.0s – 20.4s (11.4s Kanban Pan + Drawer Zoom)
				act3KanbanAndDrawer(c, f, assets, sfx)
			})
			f.Sequence(c, 1200, 696, func(f render.Frame) { // 20.0s – 31.6s (11.6s BAT+Mag-7 Jobs + Timeline)
				act4JobsAndTimeline(c, f, assets, sfx)
			})
			f.Sequence(c, 1872, 492, func(f render.Frame) { // 31.2s – 39.4s (8.2s Crisp Dark Wipe + ⌘K Push-In)
				act5DarkSweepAndPalette(c, f, assets, sfx)
			})
			f.Sequence(c, 2340, 528, func(f render.Frame) { // 39.0s – 47.8s (8.8s Flowing MCP & CLI Studio)
				act6LocalMCPAndCLI(c, f, assets, sfx)
			})
			f.Sequence(c, 2844, 228, func(f render.Frame) { // 47.4s – 51.2s (3.8s Brisk 2×2 Pillars)
				act7ApplePillars(c, f, sfx)
			})
			f.Sequence(c, 3048, 192, func(f render.Frame) { // 50.8s – 54.0s (3.2s Decisive Finale)
				act8FinaleOutro(c, f, assets, sfx)
			})
		},
	}
}

// -----------------------------------------------------------------------------
// Act 1: Crisp Apple Brand Opening (0.0s – 3.6s, 216 frames)
// Low info density -> fast, confident dolly-in with delicate 3D pitch settle.
// -----------------------------------------------------------------------------
func act1BrandOpening(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 14 {
		f.Audio(sfx.ChimeMid, render.AudioOptions{Volume: 0.48})
	}

	fadeOut := f.Interpolate([]float64{186, 215}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	iconIn := f.Interpolate([]float64{6, 46}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	titleIn := f.Interpolate([]float64{26, 62}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	lineIn := f.Interpolate([]float64{50, 88}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	// Camera: subtle push-in dolly across the 3.6s opening
	camScale := float32(f.Interpolate([]float64{0, 216}, []float64{0.93, 1.04}, render.Interpolation{Easing: ui.EaseOut}))
	camPitch := float32(10.0 * (1.0 - iconIn))

	ui.Box(c).Fill().Background(colBlack).Opacity(float32(fadeOut)).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.44},
				Radius: 300,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(168, 217, 108, float32(0.11*iconIn)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(18).
			Perspective(1200).RotateX(camPitch).Scale(camScale, camScale).
			Children(func() {
				iScale := float32(0.84 + 0.16*iconIn)
				ui.Box(c).Size(104, 104).Center().Radius(26).
					Border(1, ui.RGBA(255, 255, 255, 0.12)).
					Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.75)).
					Scale(iScale, iScale).Opacity(float32(iconIn)).
					Children(func() {
						ui.Image(c, assets.LogoSVG).Size(102, 102).Radius(25)
					})

				ui.Text(c, "Aion").FontSize(68).Bold().LetterSpacing(-2.2).TextColor(colAppleLightTxt).
					Opacity(float32(titleIn)).Translate(0, float32(16*(1-titleIn)))

				ui.Text(c, "项目与求职，本地原生。").FontSize(22).TextColor(colAppleDarkSub).
					Opacity(float32(lineIn)).Translate(0, float32(12*(1-lineIn)))
			})
	})
}

// -----------------------------------------------------------------------------
// Act 2: Today Focus (3.2s – 9.4s, 372 frames)
// Camera choreography: 3D Hero Pitch+Yaw Reveal -> Smooth Push-In toward the
// live tasks & right-hand BAT/Mag-7 job status column!
// -----------------------------------------------------------------------------
func act2TodayFocus(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.36})
	}
	fadeIn := f.Interpolate([]float64{0, 26}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{344, 371}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	// 1) Hero 3D arrival (frames 0..64)
	enter := f.Interpolate([]float64{0, 64}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// 2) Purposeful dolly-in + slight pan toward right column (frames 120..310)
	push := f.Interpolate([]float64{120, 310}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

	rotX := float32(13.0 * (1.0 - enter))
	rotY := float32(-7.0*(1.0-enter) + 1.5*push)
	camScale := float32((0.90 + 0.10*enter) * (1.0 + 0.065*push))
	camX := float32(-22.0 * push) // subtly brings right-hand Jobs + MCP status closer to center
	camY := float32(30.0*(1.0-enter) + 16.0*push)

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		renderAppleHeadline(c, f, "今天。", "待办、求职与状态，一眼掌控。", false)

		ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1250).RotateX(rotX).RotateY(rotY).
			Scale(camScale, camScale).Translate(camX, camY).
			Opacity(float32(min(enter*1.8, 1))).
			Children(func() {
				renderAppleWindow(c, assets.FocusLight, 704, 440, false, true)
			})
	})
}

// -----------------------------------------------------------------------------
// Act 3: Kanban Board -> Task Detail Drawer (9.0s – 20.4s, 684 frames)
// High info density -> generous 11.4s with two distinct camera moves:
//   Phase A (Kanban): smooth lateral tracking pan across Planned -> InProgress -> Done
//   Phase B (Task Detail): intimate push-in zoom toward the left Markdown/Subtask drawer
// -----------------------------------------------------------------------------
func act3KanbanAndDrawer(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.36})
	}
	if f.Index == 324 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.40})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.25})
	}

	fadeIn := f.Interpolate([]float64{0, 26}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{656, 683}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	enter := f.Interpolate([]float64{0, 56}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// Lateral tracking pan across the 3 Kanban columns (frames 45..285)
	pan := f.Interpolate([]float64{45, 285}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Morph into Task Detail Drawer (frames 318..366)
	drawerBlend := f.Interpolate([]float64{318, 366}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Intimate focus push-in onto the Task Detail Drawer (frames 345..580)
	detailZoom := f.Interpolate([]float64{345, 580}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

	// Combined seamless camera curve (Y-compensated so zoomed close-ups never overlap the top headline)
	panZoom := 0.06 * math.Sin(math.Pi*pan) * (1.0 - drawerBlend)
	camScale := float32((0.95 + 0.05*enter) * (1.0 + panZoom + 0.11*detailZoom))
	camX := float32(
		26.0*(1.0-enter) +
			(24.0-48.0*pan)*(1.0-drawerBlend) +
			66.0*detailZoom, // centers the left-docked Task Detail drawer as we zoom in!
	)
	camY := float32(18.0*(1.0-enter) + 14.0*panZoom/0.06 + 26.0*detailZoom)
	rotY := float32(5.5*(1.0-enter) - 2.2*math.Sin(math.Pi*pan)*(1.0-drawerBlend) + 2.0*detailZoom)

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		if drawerBlend < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32(1 - drawerBlend*2)).Children(func() {
				renderAppleHeadline(c, f, "三列看板。", "拖拽排序，多维筛选。", false)
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32((drawerBlend - 0.5) * 2)).Children(func() {
				renderAppleHeadlineStatic(c, "任务详情。", "Markdown、子任务与本地附件。", false)
			})
		}

		ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1400).RotateY(rotY).
			Scale(camScale, camScale).Translate(camX, camY).
			Opacity(float32(min(enter*1.8, 1))).
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
// Act 4: Job Tracker (BAT + Mag-7 Logos!) -> Stage Timeline (20.0s – 31.6s, 696 frames)
// High info density -> generous 11.6s:
//   Phase A: 3D angled arrival -> smooth close-up sweep highlighting Google, ByteDance,
//            Apple, Microsoft, Tencent, Alibaba logos & Offer conversion rate!
//   Phase B: Center spotlight push-in on the full ByteDance Stage Timeline Modal!
// -----------------------------------------------------------------------------
func act4JobsAndTimeline(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.36})
	}
	if f.Index == 342 {
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.34})
	}

	fadeIn := f.Interpolate([]float64{0, 26}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{668, 695}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	enter := f.Interpolate([]float64{0, 62}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// Gentle push-in toward company logos & stages (frames 70..295)
	logoFocus := f.Interpolate([]float64{70, 295}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Reveal Timeline modal (frames 336..384)
	modalBlend := f.Interpolate([]float64{336, 384}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Smooth spotlight push-in on the Timeline modal (frames 370..620)
	modalZoom := f.Interpolate([]float64{370, 620}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

	rotX := float32(7.0 * (1.0 - enter))
	rotY := float32(9.0*(1.0-enter) - 1.5*logoFocus*(1.0-modalBlend))
	camScale := float32(
		(0.93 + 0.07*enter) *
			(1.0 + 0.075*logoFocus*(1.0-modalBlend) + 0.09*modalZoom),
	)
	camX := float32(-20.0*(1.0-enter) + 22.0*logoFocus*(1.0-modalBlend))
	camY := float32(20.0*(1.0-enter) + 18.0*logoFocus*(1.0-modalBlend) + 22.0*modalZoom)

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		if modalBlend < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32(1 - modalBlend*2)).Children(func() {
				renderAppleHeadline(c, f, "求职追踪。", "自动识别企业图标，进度与 Offer 一目了然。", false)
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32((modalBlend - 0.5) * 2)).Children(func() {
				renderAppleHeadlineStatic(c, "复盘时间线。", "记录每轮细节，支持历史节点补录。", false)
			})
		}

		ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1350).RotateX(rotX).RotateY(rotY).
			Scale(camScale, camScale).Translate(camX, camY).
			Opacity(float32(min(enter*1.8, 1))).
			Children(func() {
				renderAppleWindow(c, assets.JobsLight, 704, 440, false, true)
				if modalBlend > 0.005 {
					ms := float32(0.96 + 0.04*modalBlend)
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
// Act 5: Crisp Light -> Dark Wipe & ⌘K Command Palette (31.2s – 39.4s, 492 frames)
// Brisk visual sweep (0.7s hold -> 1.4s sweep -> 0.8s beat), then snappy ⌘K press
// + spotlight camera push-in onto the Command Palette (4.5s readable hold!).
// -----------------------------------------------------------------------------
func act5DarkSweepAndPalette(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 46 {
		f.Audio(sfx.WhooshSweep, render.AudioOptions{Volume: 0.48})
	}
	if f.Index == 176 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.68})
	}
	if f.Index == 186 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.74})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.32})
	}

	fadeIn := f.Interpolate([]float64{0, 24}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{464, 491}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	// Brisk 1.4s sweep from Light -> Dark (frames 44..128)
	sweep := f.Interpolate([]float64{44, 128}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	bgCol := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colWhiteStudio, colBlack})
	primaryTxt := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colAppleText, colAppleLightTxt})
	subTxt := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colAppleSub, colAppleDarkSub})

	// Command Palette pops in at frames 192..226, followed by spotlight push-in camera!
	palBlend := f.Interpolate([]float64{192, 226}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	palCam := f.Interpolate([]float64{196, 440}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

	ui.Box(c).Fill().Background(bgCol).Opacity(sceneOp).Children(func() {
		primary := "深浅外观。"
		secondary := "昼夜切换，始终专注。"
		if f.Index >= 184 {
			primary = "⌘K 全局直达。"
			secondary = "瞬时检索活跃与归档记录。"
		}
		ui.Row(c).Absolute().Left(0).Top(28).Right(0).Justify(ui.Center).AlignItems(ui.Baseline).Gap(4).Children(func() {
			ui.Text(c, primary).FontSize(23).Bold().TextColor(primaryTxt)
			ui.Text(c, secondary).FontSize(23).TextColor(subTxt)
		})

		const winX, winY, winW, winH = 128, 78, 704, 440
		// Only apply spotlight push-in AFTER sweep completes (keeps ClipInsetPercent 100% stationary for Metal GPU speed!)
		camScale := float32(1.0 + 0.10*palCam)
		camY := float32(24.0 * palCam)

		ui.Box(c).Absolute().Left(winX).Top(winY).Size(winW, winH).
			Scale(camScale, camScale).Translate(0, camY).
			Children(func() {
				if sweep < 0.999 {
					renderAppleWindow(c, assets.BoardLight, winW, winH, false, true)
					if sweep > 0.002 {
						rightInset := float32((1.0 - sweep) * 100.0)
						ui.Box(c).Absolute().Left(0).Top(0).Size(winW, winH).
							ClipInsetPercent(0, rightInset, 0, 0, 14).
							Children(func() {
								renderAppleWindow(c, assets.BoardDark, winW, winH, true, true)
							})
					}
					if sweep > 0.01 && sweep < 0.99 {
						beamX := float32(sweep * winW)
						beamAlpha := float32(math.Sin(math.Pi * sweep))
						ui.Box(c).Absolute().Left(beamX - 12).Top(0).Size(24, winH).Opacity(beamAlpha).
							LinearGradient(ui.LinearGradient{
								Angle: 90,
								Stops: []ui.GradientStop{
									{Color: ui.Transparent, At: 0},
									{Color: ui.RGBA(255, 255, 255, 0.60), At: 0.5},
									{Color: ui.Transparent, At: 1},
								},
							})
					}
				} else {
					renderAppleWindow(c, assets.BoardDark, winW, winH, true, true)
				}

				if palBlend > 0.005 {
					ps := float32(0.96 + 0.04*palBlend)
					ui.Box(c).Absolute().Left(0).Top(0).Size(winW, winH).
						Scale(ps, ps).Opacity(float32(palBlend)).
						Children(func() {
							renderAppleWindow(c, assets.PaletteDark, winW, winH, true, true)
						})
				}
			})

		// Crisp ⌘ + K keycap indicator (frames 152..234)
		if f.Index >= 152 && f.Index <= 238 {
			kIn := f.Interpolate([]float64{152, 170}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
			kOut := f.Interpolate([]float64{212, 238}, []float64{1, 0})
			pressCmd := f.Interpolate([]float64{174, 179, 190}, []float64{1, 0.90, 1})
			pressK := f.Interpolate([]float64{184, 189, 200}, []float64{1, 0.90, 1})
			ui.Row(c).Absolute().Left(408).Top(452).Gap(10).AlignItems(ui.Center).
				Opacity(float32(kIn * kOut)).Translate(0, float32(10*(1-kIn))).
				Children(func() {
					renderAppleKeyCap(c, "⌘", float32(pressCmd), f.Index >= 175 && f.Index <= 194)
					ui.Text(c, "+").FontSize(14).TextColor(colAppleDarkSub)
					renderAppleKeyCap(c, "K", float32(pressK), f.Index >= 185 && f.Index <= 204)
				})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 6: Flowing Local AI / MCP & CLI Real-Time Sync (39.0s – 47.8s, 528 frames)
// Brings back the luminous, dynamic v1 flowing-conduit energy with refined dual
// ambient auras, multi-packet glowing Unix Socket stream, AND facing 3D rack-focus
// studio camera!
// -----------------------------------------------------------------------------
func act6LocalMCPAndCLI(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.40})
	}
	if f.Index >= 32 && f.Index <= 104 && (f.Index-32)%8 == 0 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.24})
	}
	if f.Index == 216 || f.Index == 336 {
		f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.38})
	}
	if f.Index == 344 {
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.26})
	}

	fadeIn := f.Interpolate([]float64{0, 26}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{498, 527}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	leftIn := f.Spring(ui.SpringOptions{Duration: 680_000_000, Bounce: 0.10})
	tRight, _ := f.Shift(14, 0)
	rightIn := tRight.Spring(ui.SpringOptions{Duration: 700_000_000, Bounce: 0.10})

	// Subtle studio camera push-in + gentle pan from Terminal -> Bridge -> Settings as sync triggers
	stageZoom := float32(f.Interpolate([]float64{0, 528}, []float64{0.975, 1.03}, render.Interpolation{Easing: ui.EaseInOut}))
	stagePanX := float32(f.Interpolate([]float64{160, 420}, []float64{10, -10}, render.Interpolation{Easing: ui.EaseInOut}))
	auraPulse := 0.85 + 0.15*math.Sin(float64(f.Index)*0.07)

	ui.Box(c).Fill().Background(colObsidianDeep).Opacity(sceneOp).Children(func() {
		// Full-canvas seamless dual ambient aura (botanical lime left, Apple blue right)
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.24, Y: 0.56},
				Radius: 360,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(180, 228, 81, float32(0.13*auraPulse)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.77, Y: 0.54},
				Radius: 360,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(0, 113, 227, float32(0.16*auraPulse)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		// Crisp Apple two-tone headline
		renderAppleHeadline(c, f, "本地 MCP 与 CLI。", "经由 Unix Socket，实时流式协同。", true)

		// Camera rig wrapping the two facing panes + glowing conduit
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			Scale(stageZoom, stageZoom).Translate(stagePanX, 0).
			Children(func() {
				// Left Pane: Live Terminal gently angled inward (+5° -> +2.5°)
				leftYaw := float32(5.5*(1-min(leftIn, 1)) + 2.5)
				ui.Box(c).Absolute().Left(30).Top(90).Size(404, 396).
					Perspective(1400).RotateY(leftYaw).
					Translate(float32(-26*(1-leftIn)), 0).Opacity(float32(min(leftIn*2, 1))).
					Children(func() {
						renderFlowingMCPTerminal(c, f, 404, 396)
					})

				// Center Flowing IPC Conduit (x = 434..526, 92 DIP clean glowing data stream!)
				bridgeOp := float32(f.Interpolate([]float64{52, 86}, []float64{0, 1}))
				ui.Box(c).Absolute().Left(434).Top(246).Size(92, 92).Opacity(bridgeOp).Children(func() {
					// Primary glowing forward stream wire (Terminal -> Aion)
					ui.Box(c).Absolute().Left(4).Top(36).Size(84, 2.5).Radius(2).
						LinearGradient(ui.LinearGradient{
							Angle: 90,
							Stops: []ui.GradientStop{
								{Color: ui.RGBA(180, 228, 81, 0.20), At: 0},
								{Color: ui.RGBA(180, 228, 81, 0.95), At: 0.5},
								{Color: ui.RGBA(56, 189, 248, 0.50), At: 1},
							},
						})
					// Secondary subtle return acknowledgement wire (Aion -> Terminal)
					ui.Box(c).Absolute().Left(4).Top(50).Size(84, 1.5).Radius(2).
						LinearGradient(ui.LinearGradient{
							Angle: 90,
							Stops: []ui.GradientStop{
								{Color: ui.RGBA(56, 189, 248, 0.18), At: 0},
								{Color: ui.RGBA(56, 189, 248, 0.65), At: 0.5},
								{Color: ui.RGBA(180, 228, 81, 0.18), At: 1},
							},
						})

					// Two staggered luminous data packets streaming Left -> Right
					for pIdx := range 2 {
						phase := math.Mod(float64(f.Index)*0.028+float64(pIdx)*0.5, 1.0)
						dotX := float32(4 + phase*76)
						dotAlpha := float32(math.Sin(math.Pi * phase))
						// Soft trail tail
						ui.Box(c).Absolute().Left(max(4, dotX-14)).Top(34.5).Size(16, 5).Radius(999).
							Opacity(dotAlpha * 0.45).
							LinearGradient(ui.LinearGradient{
								Angle: 90,
								Stops: []ui.GradientStop{
									{Color: ui.Transparent, At: 0},
									{Color: colBrandLime, At: 1},
								},
							})
						ui.Box(c).Absolute().Left(dotX).Top(33.5).Size(7.5, 7.5).Radius(999).
							Background(colBrandLime).Opacity(dotAlpha).
							Shadow(0, 0, 12, 0, colBrandLime)
					}

					// Return ack pulse Right -> Left (sky blue)
					ackPhase := math.Mod(float64(f.Index)*0.022+0.25, 1.0)
					ackX := float32(80 - ackPhase*76)
					ackAlpha := float32(math.Sin(math.Pi * ackPhase))
					ui.Box(c).Absolute().Left(ackX).Top(47.5).Size(6, 6).Radius(999).
						Background(colSkyGlow).Opacity(ackAlpha * 0.90).
						Shadow(0, 0, 10, 0, colSkyGlow)

					// Crisp floating labels above & below stream
					ui.Column(c).Absolute().Left(4).Top(8).Width(84).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "aion.sock").FontSize(10).Bold().TextColor(colBrandLime)
					})
					ui.Column(c).Absolute().Left(4).Top(64).Width(84).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "实时同步").FontSize(10).TextColor(colAppleDarkSub)
					})
				})

				// Right Pane: Real Aion Dark Settings gently angled inward (-5° -> -2.5°)
				rightYaw := float32(-5.5*(1-min(rightIn, 1)) - 2.5)
				ui.Box(c).Absolute().Left(526).Top(90).Size(404, 396).
					Perspective(1400).RotateY(rightYaw).
					Translate(float32(26*(1-rightIn)), 0).Opacity(float32(min(rightIn*2, 1))).
					Children(func() {
						renderAppleWindow(c, assets.SettingsDark, 404, 396, true, true)
					})
			})
	})
}

func renderFlowingMCPTerminal(c *ui.Context, f render.Frame, w, h float32) {
	cmdText := "aion summary --json"
	typed := int(f.Interpolate([]float64{30, 108}, []float64{0, float64(len(cmdText))}))
	if typed < 0 {
		typed = 0
	}
	if typed > len(cmdText) {
		typed = len(cmdText)
	}
	cursor := ""
	if f.Index < 130 && (f.Index/18)%2 == 0 {
		cursor = "▍"
	}

	showJSON := f.Interpolate([]float64{124, 152}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	showMCP := f.Interpolate([]float64{208, 240}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	showSync := f.Interpolate([]float64{326, 360}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	ui.Column(c).Size(w, h).Radius(14).Background(ui.Hex("#11161C")).
		Border(1, ui.RGBA(255, 255, 255, 0.13)).
		Shadow(0, 24, 56, 0, ui.RGBA(0, 0, 0, 0.62)).
		Children(func() {
			// Window titlebar
			ui.Row(c).FillWidth().Height(34).Padding(0, 14).AlignItems(ui.Center).Gap(6.5).
				Background(ui.RGBA(255, 255, 255, 0.03)).
				Children(func() {
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FF5F57"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FEBC2E"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#28C840"))
					ui.Spacer(c)
					ui.Text(c, "aion mcp · stdio JSON-RPC 2.0").FontSize(10).TextColor(colAppleDarkSub)
				})

			// Terminal body
			ui.Column(c).FillWidth().Grow(1).Padding(18, 18).Gap(11).Children(func() {
				ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "$").FontSize(12.5).Bold().TextColor(colBrandLime)
					ui.Text(c, cmdText[:typed]+cursor).FontSize(12.5).Bold().TextColor(colAppleLightTxt)
				})

				if showJSON > 0.01 {
					ui.Column(c).FillWidth().Padding(11, 13).Radius(10).Gap(3).
						Background(ui.RGBA(255, 255, 255, 0.04)).
						Border(1, ui.RGBA(255, 255, 255, 0.07)).
						Opacity(float32(showJSON)).
						Children(func() {
							ui.Text(c, `{ "projects": 2, "in_progress": 3,`).FontSize(11).TextColor(colSkyGlow)
							ui.Text(c, `  "jobs_active": 5, "offers": 1,`).FontSize(11).TextColor(colSkyGlow)
							ui.Text(c, `  "transport": "unix_socket_live" }`).FontSize(11).TextColor(colBrandSage)
						})
				}

				if showMCP > 0.01 {
					ui.Column(c).FillWidth().Margin(3, 0, 0, 0).Gap(6).
						Opacity(float32(showMCP)).Translate(0, float32(10*(1-showMCP))).
						Children(func() {
							ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "MCP").FontSize(9.5).Bold().TextColor(colObsidianDeep).
									Padding(2, 7).Radius(5).Background(colBrandLime)
								ui.Text(c, "Antigravity / Codex 调用工具").FontSize(11).Bold().TextColor(colAppleLightTxt)
							})
							ui.Column(c).FillWidth().Padding(11, 13).Radius(10).Gap(3).
								Background(ui.RGBA(0, 113, 227, 0.13)).
								Border(1, ui.RGBA(56, 189, 248, 0.28)).
								Children(func() {
									ui.Text(c, `→ task_create({ project: "Aion 1.0",`).FontSize(10.8).TextColor(colAppleLightTxt)
									ui.Text(c, `    title: "发布前全量回归检查",`).FontSize(10.8).TextColor(colAppleLightTxt)
									ui.Text(c, `    priority: "high" })`).FontSize(10.8).TextColor(colSkyGlow)
								})
						})
				}

				if showSync > 0.01 {
					ui.Row(c).FillWidth().Margin(3, 0, 0, 0).Padding(10, 13).Radius(10).
						AlignItems(ui.Center).Gap(9).
						Background(ui.RGBA(180, 228, 81, 0.13)).
						Border(1, ui.RGBA(180, 228, 81, 0.34)).
						Opacity(float32(showSync)).Translate(0, float32(8*(1-showSync))).
						Children(func() {
							ui.Text(c, "✓").FontSize(13.5).Bold().TextColor(colBrandLime)
							ui.Column(c).Gap(2).Children(func() {
								ui.Text(c, "桌面端已通过 Socket 实时刷新").FontSize(11).Bold().TextColor(colAppleLightTxt)
								ui.Text(c, "草稿自动保护 · 绝不暴露破坏性操作").FontSize(9.8).TextColor(colBrandSage)
							})
						})
				}
			})
		})
}

// -----------------------------------------------------------------------------
// Act 7: Brisk Apple 2×2 Pillars (47.4s – 51.2s, 228 frames)
// Low-medium info density -> crisp 3.8s with smooth crane glide.
// -----------------------------------------------------------------------------
func act7ApplePillars(c *ui.Context, f render.Frame, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	fadeIn := f.Interpolate([]float64{0, 22}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{200, 227}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	craneY := float32(f.Interpolate([]float64{0, 228}, []float64{10, -6}, render.Interpolation{Easing: ui.EaseOut}))

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
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).Translate(0, craneY).Children(func() {
			for i, item := range items {
				row := i / 2
				col := i % 2
				x := float32(startX + col*(cardW+gapX))
				y := float32(startY + row*(cardH+gapY))

				delay := 10 + i*12
				if f.Index == delay {
					f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.24})
				}
				p := f.Interpolate([]float64{float64(delay), float64(delay + 26)}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

				ui.Column(c).Absolute().Left(x).Top(y).Size(cardW, cardH).
					Justify(ui.Center).Padding(26, 30).Gap(10).Radius(20).
					Background(colDarkSurface).
					Border(1, ui.RGBA(255, 255, 255, 0.08)).
					Opacity(float32(p)).Translate(0, float32(14*(1-p))).
					Children(func() {
						ui.Text(c, item.hero).FontSize(24).Bold().TextColor(item.accent)
						ui.Text(c, item.sub).FontSize(13).LineHeight(1.55).TextColor(colAppleDarkSub)
					})
			}
		})
	})
}

// -----------------------------------------------------------------------------
// Act 8: Decisive Apple Finale (50.8s – 54.0s, 192 frames)
// Low info density -> clean 3.2s with gentle pull-back settle.
// -----------------------------------------------------------------------------
func act8FinaleOutro(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 10 {
		f.Audio(sfx.FinaleChord, render.AudioOptions{Volume: 0.65})
	}

	fadeIn := f.Interpolate([]float64{0, 24}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{156, 191}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	logoIn := f.Interpolate([]float64{4, 42}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	textIn := f.Interpolate([]float64{18, 56}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	footIn := f.Interpolate([]float64{42, 78}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	pullBack := float32(f.Interpolate([]float64{0, 192}, []float64{1.04, 0.98}, render.Interpolation{Easing: ui.EaseOut}))

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

		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(16).
			Scale(pullBack, pullBack).
			Children(func() {
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

func renderAppleHeadline(c *ui.Context, f render.Frame, primary, secondary string, dark bool) {
	p := f.Interpolate([]float64{4, 34}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
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
