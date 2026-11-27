package main

import (
	"math"

	"github.com/egoist/mygo/plugins/render"
	"github.com/egoist/mygo/ui"
)

// Brand & editorial palette derived from Aion's icon & native Apple-minimalist design tokens.
var (
	colObsidian     = ui.Hex("#0D0F0E")
	colObsidianDeep = ui.Hex("#080A0C")
	colCharcoalCard = ui.Hex("#15191C")
	colIvory        = ui.Hex("#F7F6F2")
	colSage         = ui.Hex("#ADDA70")
	colLime         = ui.Hex("#B4E451")
	colAionBlue     = ui.Hex("#0071E3")
	colSkyGlow      = ui.Hex("#38BDF8")
	colStudioLight  = ui.Hex("#F3F5F8")
	colStudioWarm   = ui.Hex("#F6F7FB")
	colInkDark      = ui.Hex("#14171A")
	colInkMuted     = ui.Hex("#5B6572")
	colDarkMuted    = ui.Hex("#9AA4B2")
)

func buildComposition(assets *demoAssets, sfx *soundBank) render.Composition {
	return render.Composition{
		Width:      videoWidth,
		Height:     videoHeight,
		FPS:        videoFPS,
		Frames:     totalFrames,
		Scale:      videoScale,
		Dark:       true,
		Background: colObsidian,
		Locale:     "zh-Hans",
		View: func(c *ui.Context, f render.Frame) {
			// Continuous stereo ambient soundtrack across the entire video.
			f.Audio(sfx.Score, render.AudioOptions{Volume: 0.92})

			// 8 overlapping acts with smooth cross-transitions (total: 2160 frames = 36s @ 60fps).
			f.Sequence(c, 0, 264, func(f render.Frame) {
				act1BrandAwakening(c, f, assets, sfx)
			})
			f.Sequence(c, 240, 282, func(f render.Frame) {
				act2TodayFocus(c, f, assets, sfx)
			})
			f.Sequence(c, 498, 342, func(f render.Frame) {
				act3KanbanAndDrawer(c, f, assets, sfx)
			})
			f.Sequence(c, 816, 336, func(f render.Frame) {
				act4JobsAndTimeline(c, f, assets, sfx)
			})
			f.Sequence(c, 1128, 324, func(f render.Frame) {
				act5DarkSweepAndPalette(c, f, assets, sfx)
			})
			f.Sequence(c, 1428, 312, func(f render.Frame) {
				act6LocalMCPAndCLI(c, f, assets, sfx)
			})
			f.Sequence(c, 1716, 252, func(f render.Frame) {
				act7BentoArchitecture(c, f, sfx)
			})
			f.Sequence(c, 1944, 216, func(f render.Frame) {
				act8FinaleOutro(c, f, assets, sfx)
			})
		},
	}
}

// -----------------------------------------------------------------------------
// Act 1: Brand Awakening (0.0s – 4.4s, 264 frames)
// -----------------------------------------------------------------------------
func act1BrandAwakening(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 12 {
		f.Audio(sfx.ChimeMid, render.AudioOptions{Volume: 0.65})
	}
	if f.Index == 54 {
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.45})
	}

	fadeOut := f.Interpolate([]float64{236, 263}, []float64{1, 0})
	logoSpring := f.Spring(ui.SpringOptions{Duration: 720_000_000, Bounce: 0.28})
	glowPulse := 0.86 + 0.14*math.Sin(float64(f.Index)*0.06)

	ui.Box(c).Fill().Background(colObsidian).Opacity(float32(fadeOut)).Children(func() {
		// Full-canvas seamless radial aura (explicit Radius eliminates any box seam).
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.42},
				Radius: 340,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(180, 228, 81, float32(0.18*glowPulse)), At: 0},
					{Color: ui.RGBA(0, 113, 227, float32(0.07*glowPulse)), At: 0.45},
					{Color: ui.Transparent, At: 1},
				},
			})

		// Subtle editorial corner frame markers
		cornerOp := float32(f.Interpolate([]float64{30, 66}, []float64{0, 0.55}))
		ui.Row(c).Absolute().Left(42).Top(28).Right(42).AlignItems(ui.Center).Opacity(cornerOp).Children(func() {
			ui.Text(c, "AION · DESKTOP WORKBENCH").FontSize(10).Bold().LetterSpacing(2.2).TextColor(colDarkMuted)
			ui.Spacer(c)
			ui.Text(c, "LOCAL-FIRST · NATIVE GO").FontSize(10).Bold().LetterSpacing(2.2).TextColor(colSage)
		})

		// Center composition
		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(16).Children(func() {
			// App icon with luminous halo ring
			iconScale := float32(0.58 + 0.42*logoSpring)
			iconOp := float32(min(logoSpring*1.8, 1))
			ui.Box(c).Size(108, 108).Center().Radius(28).
				Background(ui.RGBA(255, 255, 255, 0.04)).
				Border(1, ui.RGBA(180, 228, 81, 0.30)).
				Shadow(0, 20, 54, 0, ui.RGBA(180, 228, 81, 0.22)).
				Scale(iconScale, iconScale).Opacity(iconOp).
				Children(func() {
					ui.Image(c, assets.LogoSVG).Size(92, 92).Radius(22)
				})

			// Wordmark & etymology
			tTitle, _ := f.Shift(18, 0)
			titleSpring := tTitle.Spring(ui.Snappy.Spring)
			ui.Column(c).AlignItems(ui.Center).Gap(6).
				Opacity(float32(min(titleSpring*2, 1))).
				Translate(0, float32(22*(1-titleSpring))).
				Children(func() {
					ui.Row(c).AlignItems(ui.Baseline).Gap(10).Children(func() {
						ui.Text(c, "Aion").FontSize(68).Bold().LetterSpacing(-2.2).TextColor(colIvory)
						ui.Text(c, "αἰών").FontSize(22).TextColor(colSage).Opacity(0.92)
					})
					ui.Text(c, "生命 · 时代 · 秩序与永恒").FontSize(12).LetterSpacing(3.5).TextColor(colDarkMuted)
				})

			// Hero Chinese tagline
			tSub, _ := f.Shift(46, 0)
			subP := tSub.Interpolate([]float64{0, 26}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
			ui.Column(c).AlignItems(ui.Center).Gap(6).Margin(8, 0, 0, 0).
				Opacity(float32(subP)).Translate(0, float32(18*(1-subP))).
				Children(func() {
					ui.Text(c, "纯原生、本地优先的个人工作台").FontSize(24).Bold().TextColor(colIvory)
					ui.Text(c, "项目看板 · 求职投递 · 本地附件 · AI MCP 实时协同，一处收敛").FontSize(14).TextColor(colDarkMuted)
				})

			// Bottom capability chips popping in sequentially
			ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				pills := []struct {
					label string
					tint  ui.Color
					bg    ui.Color
				}{
					{"macOS & Windows 原生", colIvory, ui.RGBA(255, 255, 255, 0.08)},
					{"零 WebView / Electron", colSkyGlow, ui.RGBA(56, 189, 248, 0.14)},
					{"100% 本地 SQLite", colLime, ui.RGBA(180, 228, 81, 0.14)},
					{"断网毫秒级响应", colIvory, ui.RGBA(255, 255, 255, 0.08)},
				}
				for i, p := range pills {
					g, _ := f.Shift(82+i*10, 0)
					s := g.Spring(ui.Snappy.Spring)
					if f.Index == 82+i*10 {
						f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.35})
					}
					ui.Text(c, p.label).FontSize(11).Bold().TextColor(p.tint).
						Padding(6, 13).Radius(999).Background(p.bg).
						Border(1, ui.RGBA(255, 255, 255, 0.10)).
						Opacity(float32(min(s*2, 1))).Translate(0, float32(16*(1-s)))
				}
			})
		})
	})
}

// -----------------------------------------------------------------------------
// Act 2: Today Focus Command Center (4.0s – 8.7s, 282 frames)
// Centered Hero Window + Crisp Non-Overlapping Bottom Feature Dock
// -----------------------------------------------------------------------------
func act2TodayFocus(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.65})
	}
	fadeIn := f.Interpolate([]float64{0, 22}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{258, 281}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	winSpring := f.Spring(ui.SpringOptions{Duration: 680_000_000, Bounce: 0.16})
	driftY := f.Interpolate([]float64{0, 280}, []float64{3, -4})
	tiltX := float32(8.0 * (1.0 - min(winSpring, 1.0)))

	ui.Box(c).Fill().Background(colStudioLight).Opacity(sceneOp).Children(func() {
		// Full-canvas seamless studio washes
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.82, Y: 0.16},
				Radius: 380,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(0, 113, 227, 0.08), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.14, Y: 0.86},
				Radius: 360,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(173, 218, 112, 0.14), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		// Header
		renderLightActHeader(c, f, "01 · TODAY FOCUS", "打开 Aion，今日全局一眼掌控",
			"跨项目汇总待推进任务、进行中求职动态与本地 MCP 状态，支持行内快捷推进")

		// Real Aion Focus Window centered cleanly above bottom feature strip (608×380 = 1.6:1)
		wScale := float32(0.93 + 0.07*winSpring)
		ui.Box(c).Absolute().Left(176).Top(98).Size(608, 380).
			Perspective(1100).RotateX(tiltX).Scale(wScale, wScale).
			Translate(0, float32(34*(1-winSpring)+driftY)).
			Opacity(float32(min(winSpring*2, 1))).
			Children(func() {
				renderWindowCard(c, assets.FocusLight, 608, 380, false)
			})

		// Clean centered feature pills below window — ZERO occlusion!
		ui.Row(c).Absolute().Left(0).Top(492).Right(0).Justify(ui.Center).Gap(12).Children(func() {
			renderCompactPill(c, f, 44, sfx, "⌘1 聚焦总览", "待推进 / 进行中 / 已完成")
			renderCompactPill(c, f, 76, sfx, "行内快捷流转", "点击即可推进状态")
			renderCompactPill(c, f, 108, sfx, "模块无损显隐", "按阶段聚焦所需视图")
		})
	})
}

// -----------------------------------------------------------------------------
// Act 3: Three-Column Kanban & Markdown Drawer (8.3s – 14.0s, 342 frames)
// Left Editorial Callout Rail + Right-Staged Real Window (ZERO Overlap!)
// -----------------------------------------------------------------------------
func act3KanbanAndDrawer(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.65})
	}
	if f.Index == 158 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.72})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.40})
	}

	fadeIn := f.Interpolate([]float64{0, 22}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{318, 341}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	// Sub-phase crossfade: 0..155 = Kanban Board, 155..342 = Task Detail Drawer
	drawerBlend := f.Interpolate([]float64{154, 186}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	boardIn := f.Spring(ui.SpringOptions{Duration: 620_000_000, Bounce: 0.14})

	ui.Box(c).Fill().Background(colStudioLight).Opacity(sceneOp).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.68, Y: 0.18},
				Radius: 420,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(0, 113, 227, 0.08), At: 0},
					{Color: ui.RGBA(173, 218, 112, 0.09), At: 0.55},
					{Color: ui.Transparent, At: 1},
				},
			})

		// Dynamic header morphs from Kanban -> Task Detail
		if drawerBlend < 0.5 {
			headOp := float32(1.0 - drawerBlend*2)
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(headOp).Children(func() {
				renderLightActHeader(c, f, "02 · PROJECT KANBAN", "三列敏捷看板，思路流畅推进",
					"围绕「计划 / 进行中 / 已完成」拖拽跨列、列内排序，配合 ⌘F 与多标签交集筛选")
			})
		} else {
			headOp := float32((drawerBlend - 0.5) * 2)
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(headOp).Children(func() {
				renderLightActHeaderStatic(c, "02 · TASK DETAIL", "Markdown 编辑器、子任务与本地附件",
					"支持编辑 / 预览 / 分栏模式、子任务拆解与拖拽导入受管附件，大图即时预览")
			})
		}

		// Left callout rail (x = 26..244, width = 218) — completely clear of window!
		if drawerBlend < 0.55 {
			badgeOp := float32(max(0, 1-drawerBlend*2))
			ui.Column(c).Absolute().Left(26).Top(148).Width(218).Gap(14).Opacity(badgeOp).Children(func() {
				renderRailCard(c, f, 28, sfx, "⌘2 三列看板", "拖拽跨列与列内精细排序，右键菜单快速设置优先级", colAionBlue)
				renderRailCard(c, f, 58, sfx, "⌘F 多维筛选", "按关键词、优先级、截止日（已逾期/7天内）与多标签交集过滤", colAionBlue)
				renderRailCard(c, f, 88, sfx, "活跃与已归档", "双行分层头部零位移切换项目，历史成果随时回溯", ui.Hex("#16A34A"))
			})
		}
		if drawerBlend > 0.35 {
			tB, _ := f.Shift(174, 0)
			bOp := float32(min(1, (drawerBlend-0.35)/0.35))
			ui.Column(c).Absolute().Left(26).Top(148).Width(218).Gap(14).Opacity(bOp).Children(func() {
				renderRailCard(c, tB, 6, sfx, "Markdown 分栏", "内置编辑 / 预览 / 分栏模式与快捷格式工具栏", colAionBlue)
				renderRailCard(c, tB, 30, sfx, "子任务拆解", "支持勾选进度追踪、行内重命名与上下排序", colAionBlue)
				renderRailCard(c, tB, 54, sfx, "本地受管附件", "支持 Finder 拖入、缩略图大图预览与另存为导出", ui.Hex("#16A34A"))
			})
		}

		// Right-staged real window (x = 264..932, width = 668, height = 418)
		ui.Box(c).Absolute().Left(264).Top(102).Size(668, 418).
			Translate(float32(34*(1-boardIn)), float32(16*(1-boardIn))).
			Opacity(float32(min(boardIn*2, 1))).
			Children(func() {
				renderWindowCard(c, assets.BoardLight, 668, 418, false)
				if drawerBlend > 0.005 {
					ui.Box(c).Absolute().Left(0).Top(0).Size(668, 418).
						Opacity(float32(drawerBlend)).
						Translate(float32(20*(1-drawerBlend)), 0).
						Children(func() {
							renderWindowCardNoLights(c, assets.TaskDetail, 668, 418, false)
						})
				}
			})
	})
}

// -----------------------------------------------------------------------------
// Act 4: Job Applications & Full-Lifecycle Timeline (13.6s – 19.2s, 336 frames)
// Left-Staged Real Window + Right Editorial Callout Rail (ZERO Overlap!)
// -----------------------------------------------------------------------------
func act4JobsAndTimeline(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.65})
	}
	if f.Index == 156 {
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.52})
	}

	fadeIn := f.Interpolate([]float64{0, 22}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{312, 335}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	enterSpring := f.Spring(ui.SpringOptions{Duration: 620_000_000, Bounce: 0.14})
	tModal, _ := f.Shift(152, 0)
	modalSpring := tModal.Spring(ui.Snappy.Spring)
	modalOp := float32(f.Interpolate([]float64{150, 174}, []float64{0, 1}))

	ui.Box(c).Fill().Background(colStudioWarm).Opacity(sceneOp).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.78, Y: 0.72},
				Radius: 400,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(180, 228, 81, 0.16), At: 0},
					{Color: ui.RGBA(0, 113, 227, 0.06), At: 0.55},
					{Color: ui.Transparent, At: 1},
				},
			})

		if modalOp < 0.5 {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity(1 - modalOp*2).Children(func() {
				renderLightActHeader(c, f, "03 · JOB TRACKER", "求职投递全流程，从投递到 Offer",
					"自动抓取企业网站图标、实时统计 Offer 转化率，支持多条件筛选与同名进行中提醒")
			})
		} else {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Opacity((modalOp - 0.5) * 2).Children(func() {
				renderLightActHeaderStatic(c, "03 · STAGE TIMELINE", "完整复盘时间线，不错过每次跃迁",
					"记录每一轮面试备注、支持历史阶段补录、流程结束归因与随时重启跟进")
			})
		}

		// Left-staged Jobs Window (x = 28..696, width = 668, height = 418)
		ui.Box(c).Absolute().Left(28).Top(102).Size(668, 418).
			Translate(float32(-26*(1-enterSpring)), float32(16*(1-enterSpring))).
			Opacity(float32(min(enterSpring*2, 1))).
			Children(func() {
				renderWindowCard(c, assets.JobsLight, 668, 418, false)
				if modalOp > 0.005 {
					ms := float32(0.95 + 0.05*modalSpring)
					ui.Box(c).Absolute().Left(0).Top(0).Size(668, 418).
						Scale(ms, ms).Opacity(modalOp).
						Children(func() {
							renderWindowCard(c, assets.JobDetail, 668, 418, false)
						})
				}
			})

		// Right callout rail (x = 716..934, width = 218) — ZERO overlap!
		ui.Column(c).Absolute().Left(716).Top(148).Width(218).Gap(14).Children(func() {
			renderRailCard(c, f, 32, sfx, "自动网站图标", "自动发现并缓存企业 Favicon，无链接时优雅降级首字图标", colAionBlue)
			renderRailCard(c, f, 66, sfx, "Offer 转化指标", "按全部 / 进行中 / Offer / 已结束多视角统计与筛选", ui.Hex("#16A34A"))
			renderRailCard(c, f, 158, sfx, "历史阶段补录", "按真实发生日期补录节点，还原完整面经复盘时间线", colAionBlue)
		})
	})
}

// -----------------------------------------------------------------------------
// Act 5: Dramatic Light -> Dark Sweep & ⌘K Command Palette (18.8s – 24.2s, 324 frames)
// -----------------------------------------------------------------------------
func act5DarkSweepAndPalette(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 34 {
		f.Audio(sfx.WhooshSweep, render.AudioOptions{Volume: 0.85})
	}
	if f.Index == 142 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.90})
	}
	if f.Index == 150 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.95})
		f.Audio(sfx.ChimeHigh, render.AudioOptions{Volume: 0.42})
	}

	fadeIn := f.Interpolate([]float64{0, 20}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{300, 323}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	// Sweep progress from 0 to 1 across frames 32..112
	sweep := f.Interpolate([]float64{32, 112}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseInOut})
	bgCol := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colStudioLight, colObsidianDeep})
	titleCol := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colInkDark, colIvory})
	subCol := render.InterpolateColor(sweep, []float64{0, 1}, []ui.Color{colInkMuted, colDarkMuted})

	// Command Palette pops in at frame 154
	tPal, _ := f.Shift(154, 0)
	palSpring := tPal.Spring(ui.Snappy.Spring)
	palOp := float32(f.Interpolate([]float64{152, 174}, []float64{0, 1}))

	ui.Box(c).Fill().Background(bgCol).Opacity(sceneOp).Children(func() {
		// Full-canvas seamless ambient aura fading in with dark mode
		if sweep > 0.05 {
			ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).Opacity(float32(sweep)).
				RadialGradient(ui.RadialGradient{
					Center: ui.Point{X: 0.50, Y: 0.22},
					Radius: 420,
					Stops: []ui.GradientStop{
						{Color: ui.RGBA(0, 113, 227, 0.20), At: 0},
						{Color: ui.RGBA(180, 228, 81, 0.08), At: 0.50},
						{Color: ui.Transparent, At: 1},
					},
				})
		}

		// Header morphing seamlessly from Light -> Dark and from Theme -> ⌘K Palette
		ui.Column(c).Absolute().Left(0).Top(22).Right(0).AlignItems(ui.Center).Gap(4).Children(func() {
			eyebrow := "04 · APPEARANCE & SPEED"
			headline := "深浅外观无缝切换，昼夜皆悦目"
			sub := "严格遵循 macOS / Windows 系统外观语义，深色模式专注护眼"
			if f.Index >= 148 {
				eyebrow = "04 · GLOBAL COMMAND PALETTE"
				headline = "按下 ⌘K，跨模块瞬时直达一切"
				sub = "实时检索活跃与已归档项目、任务、求职投递与高频快捷动作"
			}
			ui.Text(c, eyebrow).FontSize(10).Bold().LetterSpacing(2.4).TextColor(colSage)
			ui.Text(c, headline).FontSize(26).Bold().TextColor(titleCol)
			ui.Text(c, sub).FontSize(12.5).TextColor(subCol)
		})

		// Stationary window frame (no 3D clip -> ultra-fast GPU rendering!)
		const winX, winY, winW, winH = 150, 106, 660, 412
		ui.Box(c).Absolute().Left(winX).Top(winY).Size(winW, winH).Children(func() {
			// Light Board base
			renderWindowCard(c, assets.BoardLight, winW, winH, false)

			// Dark Board revealed via stationary ClipInsetPercent sweep from left -> right!
			if sweep > 0.002 {
				rightInset := float32((1.0 - sweep) * 100.0)
				ui.Box(c).Absolute().Left(0).Top(0).Size(winW, winH).
					ClipInsetPercent(0, rightInset, 0, 0, 14).
					Children(func() {
						renderWindowCard(c, assets.BoardDark, winW, winH, true)
					})
			}

			// Glowing vertical scanline beam riding the Light -> Dark transition edge
			if sweep > 0.01 && sweep < 0.99 {
				beamX := float32(sweep * winW)
				beamAlpha := float32(math.Sin(math.Pi * sweep))
				ui.Box(c).Absolute().Left(beamX - 24).Top(0).Size(48, winH).Opacity(beamAlpha).
					LinearGradient(ui.LinearGradient{
						Angle: 90,
						Stops: []ui.GradientStop{
							{Color: ui.Transparent, At: 0},
							{Color: ui.RGBA(56, 189, 248, 0.35), At: 0.44},
							{Color: ui.RGBA(180, 228, 81, 0.92), At: 0.50},
							{Color: ui.RGBA(56, 189, 248, 0.35), At: 0.56},
							{Color: ui.Transparent, At: 1},
						},
					})
			}

			// Real Dark Mode ⌘K Command Palette overlay popping in
			if palOp > 0.005 {
				ps := float32(0.96 + 0.04*palSpring)
				ui.Box(c).Absolute().Left(0).Top(0).Size(winW, winH).
					Scale(ps, ps).Opacity(palOp).
					Children(func() {
						renderWindowCard(c, assets.PaletteDark, winW, winH, true)
					})
			}
		})

		// Floating tactile ⌘ + K keycap indicator right before palette appears (frames 122..192)
		if f.Index >= 122 && f.Index <= 192 {
			kIn := f.Interpolate([]float64{122, 136}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
			kOut := f.Interpolate([]float64{172, 192}, []float64{1, 0})
			pressCmd := f.Interpolate([]float64{140, 144, 154}, []float64{1, 0.90, 1})
			pressK := f.Interpolate([]float64{148, 152, 162}, []float64{1, 0.90, 1})
			ui.Row(c).Absolute().Left(395).Top(446).Gap(10).AlignItems(ui.Center).
				Opacity(float32(kIn * kOut)).Translate(0, float32(14*(1-kIn))).
				Children(func() {
					renderKeyCap(c, "⌘", float32(pressCmd), f.Index >= 141 && f.Index <= 156)
					ui.Text(c, "+").FontSize(16).Bold().TextColor(colDarkMuted)
					renderKeyCap(c, "K", float32(pressK), f.Index >= 149 && f.Index <= 164)
				})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 6: Local AI / MCP & CLI Real-Time Sync (23.8s – 29.0s, 312 frames)
// Split Stage with Dedicated Center IPC Conduit (ZERO overlap!)
// -----------------------------------------------------------------------------
func act6LocalMCPAndCLI(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.7})
	}
	if f.Index >= 28 && f.Index <= 76 && (f.Index-28)%6 == 0 {
		f.Audio(sfx.KeyClick, render.AudioOptions{Volume: 0.28})
	}
	if f.Index == 118 || f.Index == 196 {
		f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.55})
	}

	fadeIn := f.Interpolate([]float64{0, 20}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{288, 311}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	leftSpring := f.Spring(ui.SpringOptions{Duration: 620_000_000, Bounce: 0.14})
	tRight, _ := f.Shift(14, 0)
	rightSpring := tRight.Spring(ui.SpringOptions{Duration: 640_000_000, Bounce: 0.14})

	ui.Box(c).Fill().Background(colObsidianDeep).Opacity(sceneOp).Children(func() {
		// Full-canvas seamless dual glow
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.25, Y: 0.55},
				Radius: 360,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(180, 228, 81, 0.11), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.76, Y: 0.52},
				Radius: 360,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(0, 113, 227, 0.14), At: 0},
					{Color: ui.Transparent, At: 1},
				},
			})

		// Header
		ui.Column(c).Absolute().Left(0).Top(22).Right(0).AlignItems(ui.Center).Gap(4).Children(func() {
			ui.Text(c, "05 · LOCAL AI · MCP & CLI").FontSize(10).Bold().LetterSpacing(2.4).TextColor(colLime)
			ui.Text(c, "原生 MCP Server 与 CLI，AI 协同不离本地").FontSize(26).Bold().TextColor(colIvory)
			ui.Text(c, "一键接入 Antigravity / Codex · 通过本地 Unix Socket 实时刷新并保护未保存草稿").
				FontSize(12.5).TextColor(colDarkMuted)
		})

		// Left card: Live animated Terminal + MCP JSON-RPC log (x = 30..436, w = 406)
		ui.Box(c).Absolute().Left(30).Top(114).Size(406, 380).
			Translate(float32(-28*(1-leftSpring)), 0).Opacity(float32(min(leftSpring*2, 1))).
			Children(func() {
				renderMCPTerminalCard(c, f, 406, 380)
			})

		// Center IPC conduit (x = 436..524, 88 DIP dedicated clean corridor!)
		bridgeOp := float32(f.Interpolate([]float64{56, 82}, []float64{0, 1}))
		pulse := math.Mod(float64(f.Index)*0.038, 1.0)
		ui.Box(c).Absolute().Left(436).Top(268).Size(88, 70).Opacity(bridgeOp).Children(func() {
			// Horizontal glowing wire
			ui.Box(c).Absolute().Left(4).Top(34).Size(80, 2).
				LinearGradient(ui.LinearGradient{
					Angle: 90,
					Stops: []ui.GradientStop{
						{Color: ui.RGBA(180, 228, 81, 0.25), At: 0},
						{Color: ui.RGBA(180, 228, 81, 0.95), At: 0.5},
						{Color: ui.RGBA(56, 189, 248, 0.45), At: 1},
					},
				})
			// Traveling luminous packet dot
			dotX := float32(6 + pulse*70)
			ui.Box(c).Absolute().Left(dotX).Top(31).Size(8, 8).Radius(999).
				Background(colLime).Shadow(0, 0, 12, 0, colLime)
			// Clean centered caption pill
			ui.Column(c).Absolute().Left(8).Top(6).Width(72).AlignItems(ui.Center).Gap(2).Children(func() {
				ui.Text(c, "aion.sock").FontSize(9.5).Bold().TextColor(colLime)
			})
			ui.Column(c).Absolute().Left(8).Top(46).Width(72).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "实时同步").FontSize(9.5).TextColor(colDarkMuted)
			})
		})

		// Right card: Real Aion Settings (Dark Mode) showing Backup & MCP Config (x = 524..930, w = 406)
		ui.Box(c).Absolute().Left(524).Top(114).Size(406, 380).
			Translate(float32(28*(1-rightSpring)), 0).Opacity(float32(min(rightSpring*2, 1))).
			Children(func() {
				renderWindowCard(c, assets.SettingsDark, 406, 380, true)
			})
	})
}

func renderMCPTerminalCard(c *ui.Context, f render.Frame, w, h float32) {
	cmdText := "aion summary --json"
	typed := int(f.Interpolate([]float64{24, 78}, []float64{0, float64(len(cmdText))}))
	if typed < 0 {
		typed = 0
	}
	if typed > len(cmdText) {
		typed = len(cmdText)
	}
	cursorOn := (f.Index/18)%2 == 0
	cursor := " "
	if cursorOn && f.Index < 92 {
		cursor = "▍"
	}

	showJSON := f.Interpolate([]float64{84, 100}, []float64{0, 1})
	showMCPCall := f.Interpolate([]float64{122, 144}, []float64{0, 1})
	showSyncOK := f.Interpolate([]float64{186, 206}, []float64{0, 1})

	ui.Column(c).Size(w, h).Radius(14).Background(ui.Hex("#11161C")).
		Border(1, ui.RGBA(255, 255, 255, 0.12)).
		Shadow(0, 22, 54, 0, ui.RGBA(0, 0, 0, 0.55)).
		Children(func() {
			// Window titlebar
			ui.Row(c).FillWidth().Height(34).Padding(0, 14).AlignItems(ui.Center).Gap(7).
				Background(ui.RGBA(255, 255, 255, 0.03)).
				Children(func() {
					ui.Box(c).Size(9, 9).Radius(999).Background(ui.Hex("#FF5F57"))
					ui.Box(c).Size(9, 9).Radius(999).Background(ui.Hex("#FEBC2E"))
					ui.Box(c).Size(9, 9).Radius(999).Background(ui.Hex("#28C840"))
					ui.Spacer(c)
					ui.Text(c, "aion mcp · stdio JSON-RPC 2.0").FontSize(10).TextColor(colDarkMuted)
				})

			// Terminal body
			ui.Column(c).FillWidth().Grow(1).Padding(16, 16).Gap(9).Children(func() {
				// Live typed CLI command
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "$").FontSize(12).Bold().TextColor(colLime)
					ui.Text(c, cmdText[:typed]+cursor).FontSize(12).Bold().TextColor(colIvory)
				})

				// Structured JSON output
				if showJSON > 0.01 {
					ui.Column(c).FillWidth().Padding(10, 12).Radius(9).Gap(3).
						Background(ui.RGBA(255, 255, 255, 0.04)).
						Border(1, ui.RGBA(255, 255, 255, 0.06)).
						Opacity(float32(showJSON)).
						Children(func() {
							ui.Text(c, `{ "projects": 2, "in_progress": 3,`).FontSize(11).TextColor(colSkyGlow)
							ui.Text(c, `  "jobs_active": 4, "offers": 1,`).FontSize(11).TextColor(colSkyGlow)
							ui.Text(c, `  "transport": "unix_socket_live" }`).FontSize(11).TextColor(colSage)
						})
				}

				// AI MCP tool invocation
				if showMCPCall > 0.01 {
					ui.Column(c).FillWidth().Margin(4, 0, 0, 0).Gap(5).
						Opacity(float32(showMCPCall)).
						Translate(0, float32(10*(1-showMCPCall))).
						Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "MCP").FontSize(9.5).Bold().TextColor(colObsidian).
									Padding(2, 6).Radius(4).Background(colLime)
								ui.Text(c, "Antigravity / Codex 调用工具").FontSize(11).Bold().TextColor(colIvory)
							})
							ui.Column(c).FillWidth().Padding(10, 12).Radius(9).Gap(3).
								Background(ui.RGBA(0, 113, 227, 0.12)).
								Border(1, ui.RGBA(56, 189, 248, 0.25)).
								Children(func() {
									ui.Text(c, `→ task_create({ project: "Aion 1.0",`).FontSize(10.5).TextColor(colIvory)
									ui.Text(c, `    title: "发布前全量回归检查",`).FontSize(10.5).TextColor(colIvory)
									ui.Text(c, `    priority: "high" })`).FontSize(10.5).TextColor(colSkyGlow)
								})
						})
				}

				// Real-time sync confirmation badge
				if showSyncOK > 0.01 {
					ui.Row(c).FillWidth().Margin(4, 0, 0, 0).Padding(9, 12).Radius(9).
						AlignItems(ui.Center).Gap(8).
						Background(ui.RGBA(180, 228, 81, 0.13)).
						Border(1, ui.RGBA(180, 228, 81, 0.32)).
						Opacity(float32(showSyncOK)).
						Translate(0, float32(8*(1-showSyncOK))).
						Children(func() {
							ui.Text(c, "✓").FontSize(13).Bold().TextColor(colLime)
							ui.Column(c).Gap(1).Children(func() {
								ui.Text(c, "桌面端已通过 Socket 实时刷新").FontSize(11).Bold().TextColor(colIvory)
								ui.Text(c, "绝不暴露永久删除 / 清空回收站等高危操作").FontSize(9.5).TextColor(colSage)
							})
						})
				}
			})
		})
}

// -----------------------------------------------------------------------------
// Act 7: Architecture & Local-First Bento Grid (28.6s – 32.8s, 252 frames)
// -----------------------------------------------------------------------------
func act7BentoArchitecture(c *ui.Context, f render.Frame, sfx *soundBank) {
	if f.Index == 6 {
		f.Audio(sfx.WhooshSoft, render.AudioOptions{Volume: 0.65})
	}

	fadeIn := f.Interpolate([]float64{0, 18}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{228, 251}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	type bentoItem struct {
		tag    string
		title  string
		desc   string
		accent ui.Color
		bg     ui.Color
	}
	cards := []bentoItem{
		{
			tag:    "NATIVE ENGINE",
			title:  "MyGo 纯原生界面",
			desc:   "不打包 Chromium 与 WebView，macOS Universal 与 Windows 原生丝滑运行。",
			accent: colSkyGlow,
			bg:     ui.RGBA(56, 189, 248, 0.14),
		},
		{
			tag:    "LOCAL-FIRST",
			title:  "100% 本地 SQLite v3",
			desc:   "零账号绑定、零云端依赖，断网环境下依然保持毫秒级读写与事务一致性。",
			accent: colLime,
			bg:     ui.RGBA(180, 228, 81, 0.14),
		},
		{
			tag:    "DATA SAFETY",
			title:  "统一回收站与自动备份",
			desc:   "软删除安全兜底，定时轮转备份、恢复前安全快照与全量 .zip 一键迁移。",
			accent: ui.Hex("#FBBF24"),
			bg:     ui.RGBA(251, 191, 36, 0.14),
		},
		{
			tag:    "AI READY",
			title:  "MCP Server + CLI",
			desc:   "标准 stdio MCP 协议与轻量 CLI，安全赋能 AI 助手且隔离所有高危操作。",
			accent: ui.Hex("#A78BFA"),
			bg:     ui.RGBA(167, 139, 250, 0.14),
		},
		{
			tag:    "KEYBOARD FIRST",
			title:  "快捷键贯穿全局",
			desc:   "⌘1/2/3 模块直达、⌘K 跨归档搜索、⌘F 实时筛选，双手不离键盘。",
			accent: colSkyGlow,
			bg:     ui.RGBA(56, 189, 248, 0.14),
		},
		{
			tag:    "META POWERED",
			title:  "由 MyGo Render 实时渲染",
			desc:   "连你此刻看到的这段 60fps 宣传短片，也是用纯 Go 与 Metal GPU 直接渲染而成！",
			accent: colLime,
			bg:     ui.RGBA(180, 228, 81, 0.18),
		},
	}

	ui.Box(c).Fill().Background(colObsidian).Opacity(sceneOp).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.56},
				Radius: 390,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(0, 113, 227, 0.13), At: 0},
					{Color: ui.RGBA(180, 228, 81, 0.07), At: 0.55},
					{Color: ui.Transparent, At: 1},
				},
			})

		ui.Column(c).Absolute().Left(0).Top(24).Right(0).AlignItems(ui.Center).Gap(4).Children(func() {
			ui.Text(c, "06 · CRAFTED IN GO").FontSize(10).Bold().LetterSpacing(2.4).TextColor(colSage)
			ui.Text(c, "轻量架构，经得起每天重度使用").FontSize(26).Bold().TextColor(colIvory)
		})

		// 3 × 2 Bento Grid
		const cardW, cardH, gapX, gapY = 278, 178, 18, 16
		const startX, startY = 45, 112
		for idx, item := range cards {
			row := idx / 3
			col := idx % 3
			x := float32(startX + col*(cardW+gapX))
			y := float32(startY + row*(cardH+gapY))

			delay := 14 + idx*12
			if f.Index == delay {
				f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.36})
			}
			g, _ := f.Shift(delay, 0)
			sp := g.Spring(ui.Snappy.Spring)

			borderCol := ui.RGBA(255, 255, 255, 0.10)
			if idx == 5 {
				borderCol = ui.RGBA(180, 228, 81, 0.38)
			}

			ui.Column(c).Absolute().Left(x).Top(y).Size(cardW, cardH).
				AlignItems(ui.Start).
				Padding(20, 20).Gap(10).Radius(16).
				Background(colCharcoalCard).
				Border(1, borderCol).
				Shadow(0, 16, 36, 0, ui.RGBA(0, 0, 0, 0.36)).
				Opacity(float32(min(sp*2, 1))).
				Translate(0, float32(24*(1-sp))).
				Scale(float32(0.94+0.06*sp), float32(0.94+0.06*sp)).
				Children(func() {
					// Content-hugging pill badge
					ui.Text(c, item.tag).FontSize(9.5).Bold().LetterSpacing(1.5).TextColor(item.accent).
						Padding(4, 10).Radius(999).Background(item.bg)
					ui.Text(c, item.title).FontSize(17).Bold().TextColor(colIvory)
					ui.Text(c, item.desc).FontSize(12).LineHeight(1.55).TextColor(colDarkMuted)
				})
		}
	})
}

// -----------------------------------------------------------------------------
// Act 8: Grand Finale & Outro (32.4s – 36.0s, 216 frames)
// -----------------------------------------------------------------------------
func act8FinaleOutro(c *ui.Context, f render.Frame, assets *demoAssets, sfx *soundBank) {
	if f.Index == 10 {
		f.Audio(sfx.FinaleChord, render.AudioOptions{Volume: 0.85})
	}

	fadeIn := f.Interpolate([]float64{0, 24}, []float64{0, 1})
	fadeOut := f.Interpolate([]float64{184, 215}, []float64{1, 0})
	sceneOp := float32(fadeIn * fadeOut)

	sLogo := f.Spring(ui.SpringOptions{Duration: 760_000_000, Bounce: 0.22})
	tText, _ := f.Shift(20, 0)
	sText := tText.Spring(ui.Snappy.Spring)
	tMeta, _ := f.Shift(46, 0)
	metaOp := float32(tMeta.Interpolate([]float64{0, 26}, []float64{0, 1}))

	ui.Box(c).Fill().Background(colObsidian).Opacity(sceneOp).Children(func() {
		// Full-canvas seamless botanical bloom
		ui.Box(c).Absolute().Left(0).Top(0).Size(960, 540).
			RadialGradient(ui.RadialGradient{
				Center: ui.Point{X: 0.50, Y: 0.44},
				Radius: 360,
				Stops: []ui.GradientStop{
					{Color: ui.RGBA(180, 228, 81, 0.20), At: 0},
					{Color: ui.RGBA(0, 113, 227, 0.09), At: 0.48},
					{Color: ui.Transparent, At: 1},
				},
			})

		ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(16).Children(func() {
			scale := float32(0.72 + 0.28*sLogo)
			ui.Box(c).Size(98, 98).Center().Radius(26).
				Background(ui.RGBA(255, 255, 255, 0.04)).
				Border(1, ui.RGBA(180, 228, 81, 0.35)).
				Shadow(0, 22, 60, 0, ui.RGBA(180, 228, 81, 0.24)).
				Scale(scale, scale).Opacity(float32(min(sLogo*2, 1))).
				Children(func() {
					ui.Image(c, assets.LogoSVG).Size(82, 82).Radius(20)
				})

			ui.Column(c).AlignItems(ui.Center).Gap(8).
				Opacity(float32(min(sText*2, 1))).
				Translate(0, float32(18*(1-sText))).
				Children(func() {
					ui.Text(c, "Aion").FontSize(58).Bold().LetterSpacing(-2).TextColor(colIvory)
					ui.Text(c, "把项目、任务与每一次机会，安放于本地秩序之中").FontSize(20).Bold().TextColor(colIvory)
				})

			ui.Row(c).Gap(12).Margin(10, 0, 0, 0).Opacity(metaOp).Children(func() {
				ui.Text(c, "github.com/Games55k/Aion").FontSize(12).Bold().TextColor(colObsidian).
					Padding(8, 18).Radius(999).Background(colLime)
				ui.Text(c, "macOS · Windows · Built with Go & MyGo").FontSize(12).Bold().TextColor(colIvory).
					Padding(8, 18).Radius(999).Background(ui.RGBA(255, 255, 255, 0.08)).
					Border(1, ui.RGBA(255, 255, 255, 0.14))
			})
		})
	})
}

// -----------------------------------------------------------------------------
// Shared Visual Primitives
// -----------------------------------------------------------------------------

func renderLightActHeader(c *ui.Context, f render.Frame, eyebrow, headline, sub string) {
	p := f.Interpolate([]float64{4, 28}, []float64{0, 1}, render.Interpolation{Easing: ui.EaseOut})
	ui.Column(c).Absolute().Left(0).Top(20).Right(0).AlignItems(ui.Center).Gap(4).
		Opacity(float32(p)).Translate(0, float32(14*(1-p))).
		Children(func() {
			renderLightActHeaderInner(c, eyebrow, headline, sub)
		})
}

func renderLightActHeaderStatic(c *ui.Context, eyebrow, headline, sub string) {
	ui.Column(c).Absolute().Left(0).Top(20).Right(0).AlignItems(ui.Center).Gap(4).Children(func() {
		renderLightActHeaderInner(c, eyebrow, headline, sub)
	})
}

func renderLightActHeaderInner(c *ui.Context, eyebrow, headline, sub string) {
	ui.Text(c, eyebrow).FontSize(10).Bold().LetterSpacing(2.4).TextColor(colAionBlue)
	ui.Text(c, headline).FontSize(24).Bold().TextColor(colInkDark)
	ui.Text(c, sub).FontSize(12.5).TextColor(colInkMuted)
}

// renderWindowCard displays a real 2x Retina captured Aion screen inside an elevated
// macOS-style window frame without using a 3D-clipped mask (for max Metal GPU speed).
func renderWindowCard(c *ui.Context, bmp *ui.Bitmap, w, h float32, dark bool) {
	renderWindowCardOpt(c, bmp, w, h, dark, true)
}

func renderWindowCardNoLights(c *ui.Context, bmp *ui.Bitmap, w, h float32, dark bool) {
	renderWindowCardOpt(c, bmp, w, h, dark, false)
}

func renderWindowCardOpt(c *ui.Context, bmp *ui.Bitmap, w, h float32, dark, showLights bool) {
	borderCol := ui.RGBA(15, 23, 42, 0.12)
	shadowCol := ui.RGBA(15, 23, 42, 0.18)
	bg := ui.Hex("#FFFFFF")
	if dark {
		borderCol = ui.RGBA(255, 255, 255, 0.15)
		shadowCol = ui.RGBA(0, 0, 0, 0.56)
		bg = ui.Hex("#181A1F")
	}

	ui.Box(c).Size(w, h).Radius(14).Background(bg).
		Border(1, borderCol).
		Shadow(0, 20, 48, 0, shadowCol).
		Children(func() {
			// Rounded image draws its own corners natively on GPU without Clip()
			ui.Image(c, bmp).Size(w-2, h-2).Radius(13).Fit(ui.Cover)

			if showLights {
				// Authentic macOS traffic lights in top-left inset (matching Aion's TitleBarHiddenInset)
				ui.Row(c).Absolute().Left(13).Top(12).Gap(6).Children(func() {
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FF5F57"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#FEBC2E"))
					ui.Box(c).Size(8.5, 8.5).Radius(999).Background(ui.Hex("#28C840"))
				})
			}
		})
}

func renderCompactPill(c *ui.Context, f render.Frame, delay int, sfx *soundBank, title, sub string) {
	if f.Index == delay {
		f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.28})
	}
	g, ok := f.Shift(delay, 0)
	if !ok {
		return
	}
	sp := g.Spring(ui.Snappy.Spring)
	ui.Row(c).Padding(7, 14).Gap(8).AlignItems(ui.Center).Radius(999).
		Background(ui.RGBA(255, 255, 255, 0.94)).
		Border(1, ui.RGBA(15, 23, 42, 0.10)).
		Shadow(0, 8, 22, 0, ui.RGBA(15, 23, 42, 0.08)).
		Opacity(float32(min(sp*2, 1))).Translate(0, float32(12*(1-sp))).
		Children(func() {
			ui.Box(c).Size(6, 6).Radius(999).Background(colAionBlue)
			ui.Text(c, title).FontSize(11).Bold().TextColor(colInkDark)
			ui.Text(c, "· "+sub).FontSize(10.5).TextColor(colInkMuted)
		})
}

func renderRailCard(c *ui.Context, f render.Frame, delay int, sfx *soundBank, title, desc string, dotColor ui.Color) {
	if f.Index == delay {
		f.Audio(sfx.PopSoft, render.AudioOptions{Volume: 0.28})
	}
	g, ok := f.Shift(delay, 0)
	if !ok {
		return
	}
	sp := g.Spring(ui.Snappy.Spring)
	ui.Column(c).FillWidth().Padding(13, 14).Gap(5).Radius(13).
		Background(ui.RGBA(255, 255, 255, 0.92)).
		Border(1, ui.RGBA(15, 23, 42, 0.09)).
		Shadow(0, 10, 26, 0, ui.RGBA(15, 23, 42, 0.08)).
		Opacity(float32(min(sp*2, 1))).Translate(0, float32(16*(1-sp))).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
				ui.Box(c).Size(7, 7).Radius(999).Background(dotColor)
				ui.Text(c, title).FontSize(12.5).Bold().TextColor(colInkDark)
			})
			ui.Text(c, desc).FontSize(10.5).LineHeight(1.45).TextColor(colInkMuted)
		})
}

func renderKeyCap(c *ui.Context, label string, scale float32, active bool) {
	bg := ui.Hex("#1E242D")
	border := ui.RGBA(255, 255, 255, 0.22)
	txt := colIvory
	if active {
		bg = colAionBlue
		border = ui.RGBA(180, 228, 81, 0.75)
		txt = ui.Hex("#FFFFFF")
	}
	ui.Box(c).Size(52, 46).Center().Radius(11).
		Background(bg).Border(1.5, border).
		Shadow(0, 8, 22, 0, ui.RGBA(0, 0, 0, 0.45)).
		Scale(scale, scale).
		Children(func() {
			ui.Text(c, label).FontSize(20).Bold().TextColor(txt)
		})
}
