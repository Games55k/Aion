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

			// 8 acts with dynamic fast-slow editorial pacing AND 8 distinct cinema camera moves:
			//   Act 1: Optical Rack Focus + Pull-Back + Crane Exit
			//   Act 2: Low-Angle 3D Crane Rise + 3D Turntable Orbit (with moving glass sheen)
			//   Act 3: Whip-Pan Entry + Macro-to-Wide Dolly-Out + 3D Z-Depth Parallax Rack Focus
			//   Act 4: Dutch-Angle Roll-to-Level + Vertical Tilt-Down Scan + 3D Multi-Plane Elevation
			//   Act 5: Architectural Curtain Wipe + Overhead Crane Drop-In
			//   Act 6: Motivated 3D Studio Arc Orbit (follows flowing aion.sock data packet!)
			//   Act 7: 3D Floor-Tilt Cascade + Jib Crane Up
			//   Act 8: Optical Pull-Out + Soft Defocus Bookend
			f.Sequence(c, 0, 216, func(f render.Frame) { // 0.0s – 3.6s
				act1BrandOpening(c, f, assets, sfx)
			})
			f.Sequence(c, 192, 372, func(f render.Frame) { // 3.2s – 9.4s
				act2TodayFocus(c, f, assets, sfx)
			})
			f.Sequence(c, 540, 684, func(f render.Frame) { // 9.0s – 20.4s
				act3KanbanAndDrawer(c, f, assets, sfx)
			})
			f.Sequence(c, 1200, 696, func(f render.Frame) { // 20.0s – 31.6s
				act4JobsAndTimeline(c, f, assets, sfx)
			})
			f.Sequence(c, 1872, 492, func(f render.Frame) { // 31.2s – 39.4s
				act5DarkSweepAndPalette(c, f, assets, sfx)
			})
			f.Sequence(c, 2340, 528, func(f render.Frame) { // 39.0s – 47.8s
				act6LocalMCPAndCLI(c, f, assets, sfx)
			})
			f.Sequence(c, 2844, 228, func(f render.Frame) { // 47.4s – 51.2s
				act7ApplePillars(c, f, sfx)
			})
			f.Sequence(c, 3048, 192, func(f render.Frame) { // 50.8s – 54.0s
				act8FinaleOutro(c, f, assets, sfx)
			})
		},
	}
}

// -----------------------------------------------------------------------------
// Act 1: Optical Rack Focus + Pull-Back Reveal + Crane Exit (0.0s – 3.6s)
// Starts close & softly defocused -> snaps razor-sharp as lens pulls back ->
// lifts upward on a jib crane into Act 2.
// -----------------------------------------------------------------------------
func act1BrandOpening(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 14 {
		f.Audio(sfx.ChimeMid, render.AudioOptions{Volume: 0.48})
	}

	// Rack focus pulls from soft blur (12px) to crisp optical sharpness (0px)
	focusPull := f.Interpolate([]float64{0, 48}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	blurSigma := float32(12.0 * (1.0 - focusPull))

	// Pull-back dolly (1.24 -> 1.0) then subtle crane lift exit
	pullBack := f.Interpolate([]float64{0, 110}, []float64{1.24, 1.00}, render.Interpolation{Easing: ui.EaseOut})
	craneExit := f.Interpolate([]float64{176, 215}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseIn})
	fadeOut := 1.0 - craneExit

	titleIn := f.Interpolate([]float64{26, 62}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	lineIn := f.Interpolate([]float64{50, 88}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

	ui.Box(c).Fill().Background(colBlack).Opacity(float32(fadeOut)).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.44},
				Radius: 310,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(168, 217, 108, float32(0.12*focusPull)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		stage := ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(18).
			Scale(float32(pullBack), float32(pullBack)).
			Translate(0, float32(-54.0*craneExit))
		if blurSigma > 0.15 {
			stage.Blur(blurSigma)
		}
		stage.Children(func() {
			ui.Box(c).Size(104, 104).Center().Radius(26).
				Border(1, ui.RGBA(255, 255, 255, 0.14)).
				Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.75)).
				Opacity(float32(min(focusPull*1.6, 1))).
				Children(func() {
					ui.Image(c, assets.LogoSVG).Size(102, 102).Radius(25)
				})

			ui.Text(c, "Aion").FontSize(68).Bold().LetterSpacing(-2.2).TextColor(colAppleLightTxt).
				Opacity(float32(titleIn)).Translate(0, float32(18*(1-titleIn)))

			ui.Text(c, "项目与求职，本地原生。").FontSize(22).TextColor(colAppleDarkSub).
				Opacity(float32(lineIn)).Translate(0, float32(12*(1-lineIn)))
		})
	})
}

// -----------------------------------------------------------------------------
// Act 2: Low-Angle 3D Crane Rise + 3D Turntable Orbit + Whip-Pan Exit (3.2s – 9.4s)
// Dramatic low-angle jib arm rises to level, then smoothly orbits left -> right
// around the floating macOS window with a synchronized studio glass sheen,
// before whipping briskly left out of frame into Act 3!
// -----------------------------------------------------------------------------
func act2TodayFocus(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.36})
	}
	if f.Index == 338 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.44})
	}

	fadeIn := f.Interpolate([]float64{0, 24}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// Brisk horizontal whip-pan exit (frames 336..371)
	whipOut := f.Interpolate([]float64{336, 371}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseIn})
	sceneOp := float32(fadeIn * (1.0 - whipOut))

	// 1) Low-angle 3D Crane Rise (frames 0..76): pitches from +26° up to level 0°
	crane := f.Interpolate([]float64{0, 76}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// 2) Smooth 3D Turntable Orbit (frames 76..328): sweeps -13° -> +11° -> 0°
	orbit := f.Interpolate([]float64{76, 328}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	orbitYaw := -13.0 * math.Sin(2*math.Pi*orbit*0.85) * (1.0 - 0.35*orbit)

	rotX := float32(26.0*(1.0-crane) + 2.5*math.Sin(math.Pi*orbit))
	rotY := float32(-10.0*(1.0-crane) + orbitYaw + 14.0*whipOut)
	camScale := float32(0.86 + 0.14*crane - 0.04*whipOut)
	camX := float32(-320.0 * whipOut) // whip pan left out of frame
	camY := float32(95.0 * (1.0 - crane))

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		renderAppleHeadline(c, f, "今天。", "待办、求职与状态，一眼掌控。", false)

		win := ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1100).RotateX(rotX).RotateY(rotY).
			Scale(camScale, camScale).Translate(camX, camY).
			Opacity(float32(min(crane*1.8, 1)))
		if whipOut > 0.12 {
			win.Blur(float32(8.0 * whipOut))
		}
		win.Children(func() {
			renderAppleWindowWithSheen(c, assets.FocusLight, 704, 440, false, true, orbit)
		})
	})
}

// -----------------------------------------------------------------------------
// Act 3: Whip-Pan Entry + Macro-to-Wide Dolly-Out + 3D Parallax Rack Focus (9.0s – 20.4s)
//   - Whips in smoothly from the right (continuing Act 2's whip pan!)
//   - Phase A (Kanban): Macro-to-Wide DOLLY-OUT! Starts close on the active board
//     at an angled hero perspective (1.28x, -12° yaw) and smoothly pulls BACK to wide!
//   - Phase B (Task Detail): True 3D Z-Depth Parallax + Optical Rack Focus!
//     Kanban recedes into shallow depth-of-field (scales back to 0.91x, tilts -7°,
//     softly blurs 4.5px) while the Task Detail card glides forward into crisp focus!
//
// -----------------------------------------------------------------------------
func act3KanbanAndDrawer(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 318 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.42})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.28})
	}

	// Whip-pan arrival from right (frames 0..42)
	whipIn := f.Interpolate([]float64{0, 42}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{654, 683}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(min(whipIn*2.2, 1) * fadeOut)

	// Phase A: Macro-to-Wide DOLLY-OUT (frames 36..270): pulls BACK from 1.24x to 1.00x!
	dollyOut := f.Interpolate([]float64{36, 270}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Phase B: 3D Parallax Rack Focus peel (frames 314..382)
	rack := f.Interpolate([]float64{314, 382}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Gentle float orbit on the foreground Task Detail card (frames 382..650)
	detailOrbit := f.Interpolate([]float64{382, 650}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

	// Background Kanban plane transforms (Dolly-Out -> then Recedes into Background DoF!)
	bgExtraScale := 0.22 * (1.0 - dollyOut)
	bgScale := float32((1.0 + bgExtraScale) * (1.0 - 0.08*rack))
	bgRotY := float32(-11.0*(1.0-dollyOut) - 8.0*rack)
	bgRotX := float32(4.0*(1.0-dollyOut) + 3.0*rack)
	bgX := float32(280.0*(1.0-whipIn) - 24.0*(1.0-dollyOut) + 42.0*rack)
	// Y-compensate macro scale so top edge never clips the headline
	bgY := float32(float64(220.0)*bgExtraScale + 8.0*rack)
	bgBlur := float32(6.0*(1.0-whipIn) + 4.5*rack)

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		if rack < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32(1 - rack*2)).Children(func() {
				renderAppleHeadline(c, f, "三列看板。", "拖拽排序，多维筛选。", false)
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32((rack - 0.5) * 2)).Children(func() {
				renderAppleHeadlineStatic(c, "任务详情。", "Markdown、子任务与本地附件。", false)
			})
		}

		// Plane 1 (Midground -> Receding Background): Kanban Board
		bgPlane := ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1200).RotateX(bgRotX).RotateY(bgRotY).
			Scale(bgScale, bgScale).Translate(bgX, bgY).
			Opacity(float32(1.0 - 0.35*rack))
		if bgBlur > 0.15 {
			bgPlane.Blur(bgBlur)
		}
		bgPlane.Children(func() {
			renderAppleWindow(c, assets.BoardLight, 704, 440, false, true)
		})

		// Plane 2 (Foreground Rack-Focus Plane): Task Detail glides forward in 3D Z-space!
		if rack > 0.005 {
			fgScale := float32(0.90 + 0.11*rack)
			fgRotY := float32(12.0*(1.0-rack) + 4.0*math.Sin(math.Pi*detailOrbit))
			fgRotX := float32(-5.0 * (1.0 - rack))
			fgX := float32(-90.0*(1.0-rack) + 8.0*detailOrbit)
			fgY := float32(14.0 * (1.0 - rack))
			fgBlur := float32(5.0 * (1.0 - rack))

			fgPlane := ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
				Perspective(1150).RotateX(fgRotX).RotateY(fgRotY).
				Scale(fgScale, fgScale).Translate(fgX, fgY).
				Opacity(float32(min(rack*1.5, 1)))
			if fgBlur > 0.15 {
				fgPlane.Blur(fgBlur)
			}
			fgPlane.Children(func() {
				renderAppleWindow(c, assets.TaskDetail, 704, 440, false, false)
			})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 4: Dutch-Angle Roll-to-Level + Vertical Tilt-Down Scan + 3D Z-Layer Lift (20.0s – 31.6s)
//   - Enters on a stylish canted Dutch Angle (-4.5° roll, +16° pitch) and levels cleanly
//     to symbolize clarity & order across BAT + Magnificent Seven job applications.
//   - Performs a smooth Vertical Tilt-Down Scan across all 6 company logos (Tencent ->
//     Apple -> ByteDance -> Microsoft -> Google Offer -> Alibaba).
//   - Transitions via 3D Multi-Plane Depth Lift: table drops back & softly blurs while
//     ByteDance's Stage Timeline Modal elevates into sharp foreground!
//
// -----------------------------------------------------------------------------
func act4JobsAndTimeline(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.36})
	}
	if f.Index == 340 {
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.35})
	}

	fadeIn := f.Interpolate([]float64{0, 26}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{666, 695}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	// 1) Dutch-Angle Roll-to-Level arrival (frames 0..70)
	levelIn := f.Interpolate([]float64{0, 70}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// 2) Vertical Tilt-Down Scan across the 6 BAT / Mag-7 company rows (frames 72..310)
	tiltScan := f.Interpolate([]float64{72, 310}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// 3) 3D Multi-Plane Elevation into Stage Timeline (frames 334..396)
	elevate := f.Interpolate([]float64{334, 396}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	// Gentle 3D float on Timeline modal
	modalFloat := f.Interpolate([]float64{396, 660}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

	// Background Jobs Table plane
	bgRoll := float32(-4.5 * (1.0 - levelIn))
	bgRotX := float32(16.0*(1.0-levelIn) + (6.0-11.0*tiltScan)*(1.0-elevate) + 7.0*elevate)
	bgRotY := float32(10.0*(1.0-levelIn) - 4.0*math.Sin(math.Pi*tiltScan)*(1.0-elevate))
	bgScale := float32((0.92 + 0.08*levelIn) * (1.0 - 0.08*elevate))
	bgY := float32(28.0*(1.0-levelIn) + (14.0-22.0*tiltScan)*(1.0-elevate) + 18.0*elevate)
	bgBlur := float32(4.0 * elevate)

	ui.Box(c).Fill().Background(colWhiteStudio).Opacity(sceneOp).Children(func() {
		if elevate < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32(1 - elevate*2)).Children(func() {
				renderAppleHeadline(c, f, "求职追踪。", "自动识别企业图标，进度与 Offer 一目了然。", false)
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(float32((elevate - 0.5) * 2)).Children(func() {
				renderAppleHeadlineStatic(c, "复盘时间线。", "记录每轮细节，支持历史节点补录。", false)
			})
		}

		// Plane 1: Jobs Table (BAT + Mag-7 logos!)
		bgPlane := ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
			Perspective(1150).Rotate(bgRoll).RotateX(bgRotX).RotateY(bgRotY).
			Scale(bgScale, bgScale).Translate(0, bgY).
			Opacity(float32(min(levelIn*1.8, 1) * (1.0 - 0.38*elevate)))
		if bgBlur > 0.15 {
			bgPlane.Blur(bgBlur)
		}
		bgPlane.Children(func() {
			renderAppleWindow(c, assets.JobsLight, 704, 440, false, true)
		})

		// Plane 2: Elevated Foreground Stage Timeline Modal
		if elevate > 0.005 {
			fgScale := float32(0.88 + 0.13*elevate)
			fgRotX := float32(-12.0*(1.0-elevate) + 2.0*math.Sin(math.Pi*modalFloat))
			fgRotY := float32(-4.0 * math.Sin(math.Pi*modalFloat))
			fgY := float32(52.0 * (1.0 - elevate))
			fgBlur := float32(4.0 * (1.0 - elevate))

			fgPlane := ui.Box(c).Absolute().Left(128).Top(78).Size(704, 440).
				Perspective(1150).RotateX(fgRotX).RotateY(fgRotY).
				Scale(fgScale, fgScale).Translate(0, fgY).
				Opacity(float32(min(elevate*1.5, 1)))
			if fgBlur > 0.15 {
				fgPlane.Blur(fgBlur)
			}
			fgPlane.Children(func() {
				renderAppleWindow(c, assets.JobDetail, 704, 440, false, true)
			})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 5: Architectural Curtain Wipe + Overhead Crane Drop onto ⌘K (31.2s – 39.4s)
//   - Stationary razor-sharp Light -> Dark wipe (fast on Metal GPU!)
//   - On ⌘ + K press, executes an Overhead Crane Drop & Pitch Settle (RotateX +14° -> 0°,
//     vertical crane descent) rather than a plain scale zoom!
//
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

	sweep := f.Interpolate([]float64{44, 128}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	bgCol := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colWhiteStudio, colBlack})
	primaryTxt := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colAppleText, colAppleLightTxt})
	subTxt := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colAppleSub, colAppleDarkSub})

	// Overhead Crane Drop onto ⌘K Command Palette (frames 192..246)
	palIn := f.Interpolate([]float64{192, 246}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	palOrbit := f.Interpolate([]float64{246, 464}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})

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

		if sweep < 0.999 {
			// Stationary architectural wipe (zero 3D clip -> 250+ FPS!)
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
				if sweep > 0.01 && sweep < 0.99 {
					beamX := float32(sweep * winW)
					beamAlpha := float32(math.Sin(math.Pi * sweep))
					ui.Box(c).Absolute().Left(beamX-12).Top(0).Size(24, winH).Opacity(beamAlpha).
						LinearGradient(ui.LinearGradient{
							Angle: 90,
							Stops: []ui.GradientStop{
								{Color: ui.Transparent, At: 0},
								{Color: ui.RGBA(255, 255, 255, 0.60), At: 0.5},
								{Color: ui.Transparent, At: 1},
							},
						})
				}
			})
		} else {
			// Post-wipe: Background Dark Board + Overhead Crane Drop into ⌘K Palette
			bgScale := float32(1.0 - 0.05*palIn)
			bgPitch := float32(5.0 * palIn)
			ui.Box(c).Absolute().Left(winX).Top(winY).Size(winW, winH).
				Perspective(1200).RotateX(bgPitch).Scale(bgScale, bgScale).
				Opacity(float32(1.0 - 0.30*palIn)).
				Children(func() {
					renderAppleWindow(c, assets.BoardDark, winW, winH, true, true)
				})

			if palIn > 0.005 {
				// Overhead crane drop: descends from above (-26 -> 0) & pitches down (+14° -> 0°)
				dropPitch := float32(14.0 * (1.0 - palIn))
				dropYaw := float32(5.0 * math.Sin(math.Pi*palOrbit))
				dropY := float32(-22.0 * (1.0 - palIn))
				dropScale := float32(0.95 + 0.07*palIn)
				ui.Box(c).Absolute().Left(winX).Top(winY).Size(winW, winH).
					Perspective(1150).RotateX(dropPitch).RotateY(dropYaw).
					Scale(dropScale, dropScale).Translate(0, dropY).
					Opacity(float32(min(palIn*1.6, 1))).
					Children(func() {
						renderAppleWindow(c, assets.PaletteDark, winW, winH, true, true)
					})
			}
		}

		// Crisp ⌘ + K keycap indicator (frames 152..238)
		if f.Index >= 152 && f.Index <= 238 {
			kIn := f.Interpolate([]float64{152, 170}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
			kOut := f.Interpolate([]float64{212, 238}, []float64{1, 0})
			pressCmd := f.Interpolate([]float64{174, 179, 190}, []float64{1, 0.90, 1})
			pressK := f.Interpolate([]float64{184, 189, 200}, []float64{1, 0.90, 1})
			ui.Row(c).Absolute().Left(408).Top(452).Gap(10).AlignItems(ui.Center).
				Opacity(float32(kIn*kOut)).Translate(0, float32(10*(1-kIn))).
				Children(func() {
					renderAppleKeyCap(c, "⌘", float32(pressCmd), f.Index >= 175 && f.Index <= 194)
					ui.Text(c, "+").FontSize(14).TextColor(colAppleDarkSub)
					renderAppleKeyCap(c, "K", float32(pressK), f.Index >= 185 && f.Index <= 204)
				})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 6: Motivated 3D Studio Arc Orbit + Flowing aion.sock Data Stream (39.0s – 47.8s)
// True storytelling cinematography:
//   - Act starts orbited toward the LEFT Terminal (hero foreground, Settings deeper at -18° yaw)
//     as CLI & MCP tool calls stream live.
//   - As the glowing lime packet shoots across `aion.sock`, the virtual camera smoothly ARCS
//     across the bridge from Left -> Right, bringing the Dark Aion Settings pane forward to hero
//     prominence (+18° left yaw / -3° right yaw) as the live sync lands!
//
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

	enter := f.Interpolate([]float64{0, 54}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	// Motivated Studio Arc Orbit from Left Terminal (0) -> Centered Bridge (0.5) -> Right Settings (1.0)
	arc := f.Interpolate([]float64{180, 410}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	auraPulse := 0.85 + 0.15*math.Sin(float64(f.Index)*0.07)

	// Parallax depth shift: Left pane starts hero-close (1.04x, +3° yaw) -> arcs back (0.94x, +17° yaw)
	// Right pane starts deeper (0.94x, -17° yaw) -> arcs forward to hero-close (1.04x, -3° yaw)
	leftScale := float32(1.04 - 0.10*arc)
	rightScale := float32(0.94 + 0.10*arc)
	leftYaw := float32(3.0 + 14.0*arc)
	rightYaw := float32(-17.0 + 14.0*arc)
	stagePanX := float32(28.0 - 56.0*arc) // glides camera along the glowing conduit!

	ui.Box(c).Fill().Background(colObsidianDeep).Opacity(sceneOp).Children(func() {
		// Dual ambient auras shift subtly with the arc orbit (parallax background!)
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: float32(0.28 - 0.08*arc), Y: 0.56},
				Radius: 370,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(180, 228, 81, float32((0.15-0.04*arc)*auraPulse)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: float32(0.72 + 0.06*arc), Y: 0.54},
				Radius: 370,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(0, 113, 227, float32((0.12+0.07*arc)*auraPulse)), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		renderAppleHeadline(c, f, "本地 MCP 与 CLI。", "经由 Unix Socket，实时流式协同。", true)

		// Studio 3D Arc Rig
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			Translate(stagePanX, float32(22*(1-enter))).
			Opacity(float32(enter)).
			Children(func() {
				// Left Pane: Live Terminal
				ui.Box(c).Absolute().Left(30).Top(90).Size(404, 396).
					Perspective(1100).RotateY(leftYaw).Scale(leftScale, leftScale).
					Children(func() {
						renderFlowingMCPTerminal(c, f, 404, 396)
					})

				// Center Flowing IPC Conduit (x = 434..526)
				bridgeOp := float32(f.Interpolate([]float64{46, 80}, []float64{0, 1}))
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
					// Return ack wire (Aion -> Terminal)
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
						Background(colSkyGlow).Opacity(ackAlpha*0.90).
						Shadow(0, 0, 10, 0, colSkyGlow)

					ui.Column(c).Absolute().Left(4).Top(8).Width(84).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "aion.sock").FontSize(10).Bold().TextColor(colBrandLime)
					})
					ui.Column(c).Absolute().Left(4).Top(64).Width(84).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "实时同步").FontSize(10).TextColor(colAppleDarkSub)
					})
				})

				// Right Pane: Real Aion Dark Settings
				ui.Box(c).Absolute().Left(526).Top(90).Size(404, 396).
					Perspective(1100).RotateY(rightYaw).Scale(rightScale, rightScale).
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
			ui.Row(c).FillWidth().Height(34).Padding(0, 14).AlignItems(ui.Center).Gap(6.5).
				Background(ui.RGBA(255, 255, 255, 0.03)).
				Children(func() {
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FF5F57"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FEBC2E"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#28C840"))
					ui.Spacer(c)
					ui.Text(c, "aion mcp · stdio JSON-RPC 2.0").FontSize(10).TextColor(colAppleDarkSub)
				})

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
// Act 7: 3D Floor-Tilt Cascade + Jib Crane Up (47.4s – 51.2s, 228 frames)
// Each architectural pillar pitches up from a 3D perspective floor plane while
// the camera glides upward on a jib crane!
// -----------------------------------------------------------------------------
func act7ApplePillars(c *ui.Context, f render.Frame, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.35})
	}
	fadeIn := f.Interpolate([]float64{0, 22}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	fadeOut := f.Interpolate([]float64{200, 227}, []float64{1, 0}, render.Interpolation{Easing: ui.EaseInOut})
	sceneOp := float32(fadeIn * fadeOut)

	craneY := float32(f.Interpolate([]float64{0, 228}, []float64{18, -10}, render.Interpolation{Easing: ui.EaseOut}))

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
				p := f.Interpolate([]float64{float64(delay), float64(delay + 32)}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})

				ui.Column(c).Absolute().Left(x).Top(y).Size(cardW, cardH).
					Perspective(900).RotateX(float32(24.0*(1.0-p))).
					Justify(ui.Center).Padding(26, 30).Gap(10).Radius(20).
					Background(colDarkSurface).
					Border(1, ui.RGBA(255, 255, 255, 0.08)).
					Opacity(float32(p)).Translate(0, float32(36*(1-p))).
					Children(func() {
						ui.Text(c, item.hero).FontSize(24).Bold().TextColor(item.accent)
						ui.Text(c, item.sub).FontSize(13).LineHeight(1.55).TextColor(colAppleDarkSub)
					})
			}
		})
	})
}

// -----------------------------------------------------------------------------
// Act 8: Optical Pull-Out + Soft Defocus Bookend (50.8s – 54.0s, 192 frames)
// Mirrors Act 1's opening Rack Focus in reverse: pulls gently away and melts
// softly out of focus into black.
// -----------------------------------------------------------------------------
func act8FinaleOutro(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 10 {
		f.Audio(sfx.FinaleChord, render.AudioOptions{Volume: 0.65})
	}

	fadeIn := f.Interpolate([]float64{0, 24}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	outro := f.Interpolate([]float64{150, 191}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseIn})
	sceneOp := float32(fadeIn * (1.0 - outro))

	logoIn := f.Interpolate([]float64{4, 42}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	textIn := f.Interpolate([]float64{18, 56}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	footIn := f.Interpolate([]float64{42, 78}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	pullBack := float32(f.Interpolate([]float64{0, 192}, []float64{1.08, 0.94}, render.Interpolation{Easing: ui.EaseOut}))
	defocus := float32(9.0 * outro)

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

		stage := ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(16).
			Scale(pullBack, pullBack)
		if defocus > 0.15 {
			stage.Blur(defocus)
		}
		stage.Children(func() {
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
	renderAppleWindowWithSheen(c, bmp, w, h, dark, showLights, -1)
}

// renderAppleWindowWithSheen renders a native macOS window card and, when sheenPos
// is in [0, 1], sweeps a subtle studio specular reflection across the glass in sync
// with a 3D turntable orbit.
func renderAppleWindowWithSheen(c *ui.Context, bmp *ui.Bitmap, w, h float32, dark, showLights bool, sheenPos float64) {
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

			if sheenPos >= 0 && sheenPos <= 1 {
				sx := float32((sheenPos*1.4 - 0.2) * float64(w))
				sAlpha := float32(0.18 * math.Sin(math.Pi*sheenPos))
				if sx > -120 && sx < w+20 && sAlpha > 0.005 {
					ui.Box(c).Absolute().Left(max(0, sx)).Top(0).Size(min(130, w-max(0, sx)), h).Radius(13).
						Opacity(sAlpha).
						LinearGradient(ui.LinearGradient{
							Angle: 108,
							Stops: []ui.GradientStop{
								{Color: ui.Transparent, At: 0},
								{Color: ui.RGBA(255, 255, 255, 0.85), At: 0.5},
								{Color: ui.Transparent, At: 1},
							},
						})
				}
			}

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
