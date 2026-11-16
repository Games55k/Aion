package frontend

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

type uiSize int

const (
	sizeMiddle uiSize = iota
	sizeSmall
	sizeLarge
)

type uiElevation int

const (
	boxShadow uiElevation = iota
	boxShadowSecondary
	boxShadowTertiary
)

type shadowLayer struct{ y, blur, spread, light, dark float32 }

var shadowLayers = map[uiElevation][]shadowLayer{
	boxShadow: {
		{40, 80, 0, 0.06, 0.09}, {20, 40, 0, 0.05, 0.08}, {10, 20, 0, 0.04, 0.06},
		{5, 10, 0, 0.03, 0.05}, {2, 4, 0, 0.02, 0.03},
	},
	boxShadowSecondary: {
		{17.5, 23.4, 0, 0.04, 0.06}, {9.4, 12.5, 0, 0.03, 0.05}, {5.25, 7, 0, 0.02, 0.03},
		{2.8, 3.7, -2, 0.01, 0.02}, {1.2, 1.5, 0, 0.01, 0.02},
	},
	boxShadowTertiary: {{1, 2, 0, 0.04, 0.06}, {2, 4, 0, 0.02, 0.03}},
}

type uiTokens struct {
	Dark                  bool
	Blue                  ui.Color
	BlueActive            ui.Color
	BlueFillQuaternary    ui.Color
	BlueFillTertiary      ui.Color
	BgContainer           ui.Color
	BgElevated            ui.Color
	BgLayout              ui.Color
	Border                ui.Color
	BorderSecondary       ui.Color
	Error                 ui.Color
	ErrorActive           ui.Color
	ErrorBg               ui.Color
	ErrorBgActive         ui.Color
	ErrorBgHover          ui.Color
	ErrorBorder           ui.Color
	ErrorBorderHover      ui.Color
	ErrorFillQuaternary   ui.Color
	ErrorFillTertiary     ui.Color
	ErrorHover            ui.Color
	ErrorText             ui.Color
	ErrorTextActive       ui.Color
	ErrorTextHover        ui.Color
	Fill                  ui.Color
	FillContent           ui.Color
	FillQuaternary        ui.Color
	FillSecondary         ui.Color
	FillTertiary          ui.Color
	Info                  ui.Color
	InfoFillQuaternary    ui.Color
	InfoFillTertiary      ui.Color
	InfoText              ui.Color
	Primary               ui.Color
	PrimaryActive         ui.Color
	PrimaryBg             ui.Color
	PrimaryBorder         ui.Color
	PrimaryFillQuaternary ui.Color
	PrimaryFillTertiary   ui.Color
	PrimaryHover          ui.Color
	Success               ui.Color
	SuccessBg             ui.Color
	SuccessFillQuaternary ui.Color
	SuccessFillTertiary   ui.Color
	Text                  ui.Color
	TextDescription       ui.Color
	TextDisabled          ui.Color
	TextLabel             ui.Color
	TextPlaceholder       ui.Color
	TextQuaternary        ui.Color
	TextSecondary         ui.Color
	TextTertiary          ui.Color
	Warning               ui.Color
	WarningFillQuaternary ui.Color
	WarningFillTertiary   ui.Color
	Cyan                  ui.Color
	CyanActive            ui.Color
	CyanFillQuaternary    ui.Color
	CyanFillTertiary      ui.Color
	Geekblue              ui.Color
	Gray                  ui.Color
	GrayActive            ui.Color
	GrayFillQuaternary    ui.Color
	GrayFillTertiary      ui.Color
	Green                 ui.Color
	GreenActive           ui.Color
	GreenFillQuaternary   ui.Color
	GreenFillTertiary     ui.Color
	Orange                ui.Color
	OrangeActive          ui.Color
	OrangeFillQuaternary  ui.Color
	OrangeFillTertiary    ui.Color
	Purple                ui.Color
	PurpleActive          ui.Color
	PurpleFillQuaternary  ui.Color
	PurpleFillTertiary    ui.Color
	RedActive             ui.Color
	RedFillTertiary       ui.Color
	Volcano               ui.Color
	Yellow9               ui.Color
	BorderRadius          float32
	BorderRadiusLG        float32
	BorderRadiusSM        float32
	FontSize              float32
	FontSizeSM            float32
}

var lightTokens = uiTokens{
	Dark:                  false,
	Blue:                  ui.RGB(87, 171, 249),
	BlueActive:            ui.RGB(13, 120, 206),
	BlueFillQuaternary:    ui.RGBA(105, 105, 255, 0.02),
	BlueFillTertiary:      ui.RGBA(38, 138, 255, 0.06),
	BgContainer:           ui.RGB(255, 255, 255),
	BgElevated:            ui.RGB(255, 255, 255),
	BgLayout:              ui.RGB(248, 248, 248),
	Border:                ui.RGB(227, 227, 227),
	BorderSecondary:       ui.RGB(238, 238, 238),
	Error:                 ui.RGB(243, 88, 55),
	ErrorActive:           ui.RGB(206, 70, 30),
	ErrorBg:               ui.RGB(254, 248, 247),
	ErrorBgActive:         ui.RGB(255, 204, 199),
	ErrorBgHover:          ui.RGB(255, 240, 238),
	ErrorBorder:           ui.RGB(255, 222, 216),
	ErrorBorderHover:      ui.RGB(255, 231, 228),
	ErrorFillQuaternary:   ui.RGBA(230, 80, 55, 0.04),
	ErrorFillTertiary:     ui.RGBA(255, 41, 12, 0.07),
	ErrorHover:            ui.RGB(255, 119, 92),
	ErrorText:             ui.RGB(206, 70, 30),
	ErrorTextActive:       ui.RGB(206, 70, 30),
	ErrorTextHover:        ui.RGB(255, 119, 92),
	Fill:                  ui.RGBA(0, 0, 0, 0.12),
	FillContent:           ui.RGBA(0, 0, 0, 0.06),
	FillQuaternary:        ui.RGBA(0, 0, 0, 0.015),
	FillSecondary:         ui.RGBA(0, 0, 0, 0.06),
	FillTertiary:          ui.RGBA(0, 0, 0, 0.03),
	Info:                  ui.RGB(0, 118, 227),
	InfoFillQuaternary:    ui.RGBA(22, 55, 255, 0.03),
	InfoFillTertiary:      ui.RGBA(30, 117, 255, 0.08),
	InfoText:              ui.RGB(0, 96, 204),
	Primary:               ui.RGB(34, 34, 34),
	PrimaryActive:         ui.RGB(17, 17, 17),
	PrimaryBg:             ui.RGB(245, 245, 245),
	PrimaryBorder:         ui.RGB(170, 170, 170),
	PrimaryFillQuaternary: ui.RGBA(0, 0, 0, 0.08),
	PrimaryFillTertiary:   ui.RGBA(0, 0, 0, 0.16),
	PrimaryHover:          ui.RGB(51, 51, 51),
	Success:               ui.RGB(0, 160, 63),
	SuccessBg:             ui.RGB(248, 251, 241),
	SuccessFillQuaternary: ui.RGBA(138, 188, 22, 0.06),
	SuccessFillTertiary:   ui.RGBA(85, 165, 5, 0.1),
	Text:                  ui.RGB(31, 31, 31),
	TextDescription:       ui.RGB(138, 138, 138),
	TextDisabled:          ui.RGB(176, 176, 176),
	TextLabel:             ui.RGB(102, 102, 102),
	TextPlaceholder:       ui.RGB(176, 176, 176),
	TextQuaternary:        ui.RGB(176, 176, 176),
	TextSecondary:         ui.RGB(102, 102, 102),
	TextTertiary:          ui.RGB(138, 138, 138),
	Warning:               ui.RGB(254, 180, 0),
	WarningFillQuaternary: ui.RGBA(255, 115, 15, 0.05),
	WarningFillTertiary:   ui.RGBA(255, 147, 13, 0.12),
	Cyan:                  ui.RGB(149, 243, 217),
	CyanActive:            ui.RGB(47, 162, 138),
	CyanFillQuaternary:    ui.RGBA(55, 255, 122, 0.03),
	CyanFillTertiary:      ui.RGBA(26, 255, 155, 0.07),
	Geekblue:              ui.RGB(0, 114, 245),
	Gray:                  ui.RGB(136, 136, 136),
	GrayActive:            ui.RGB(102, 102, 102),
	GrayFillQuaternary:    ui.RGBA(0, 0, 0, 0.03),
	GrayFillTertiary:      ui.RGBA(0, 0, 0, 0.06),
	Green:                 ui.RGB(55, 157, 74),
	GreenActive:           ui.RGB(0, 121, 68),
	GreenFillQuaternary:   ui.RGBA(117, 230, 5, 0.06),
	GreenFillTertiary:     ui.RGBA(84, 205, 12, 0.1),
	Orange:                ui.RGB(248, 140, 19),
	OrangeActive:          ui.RGB(209, 128, 0),
	OrangeFillQuaternary:  ui.RGBA(255, 105, 255, 0.03),
	OrangeFillTertiary:    ui.RGBA(255, 80, 30, 0.08),
	Purple:                ui.RGB(189, 84, 198),
	PurpleActive:          ui.RGB(137, 43, 138),
	PurpleFillQuaternary:  ui.RGBA(255, 30, 155, 0.04),
	PurpleFillTertiary:    ui.RGBA(255, 15, 235, 0.08),
	RedActive:             ui.RGB(214, 34, 93),
	RedFillTertiary:       ui.RGBA(255, 15, 15, 0.05),
	Volcano:               ui.RGB(236, 94, 65),
	Yellow9:               ui.RGB(255, 239, 92),
	BorderRadius:          8,
	BorderRadiusLG:        12,
	BorderRadiusSM:        6,
	FontSize:              14,
	FontSizeSM:            12,
}

var darkTokens = uiTokens{
	Dark:                  true,
	Blue:                  ui.RGB(96, 177, 255),
	BlueActive:            ui.RGB(45, 138, 224),
	BlueFillQuaternary:    ui.RGBA(0, 44, 233, 0.09),
	BlueFillTertiary:      ui.RGBA(0, 88, 246, 0.26),
	BgContainer:           ui.RGB(13, 13, 13),
	BgElevated:            ui.RGB(26, 26, 26),
	BgLayout:              ui.RGB(0, 0, 0),
	Border:                ui.RGB(32, 32, 32),
	BorderSecondary:       ui.RGB(26, 26, 26),
	Error:                 ui.RGB(233, 83, 80),
	ErrorActive:           ui.RGB(190, 55, 56),
	ErrorBg:               ui.RGB(52, 11, 12),
	ErrorBgActive:         ui.RGB(91, 37, 38),
	ErrorBgHover:          ui.RGB(75, 20, 26),
	ErrorBorder:           ui.RGB(138, 51, 52),
	ErrorBorderHover:      ui.RGB(170, 63, 62),
	ErrorFillQuaternary:   ui.RGBA(250, 100, 90, 0.1),
	ErrorFillTertiary:     ui.RGBA(248, 52, 57, 0.21),
	ErrorHover:            ui.RGB(255, 168, 160),
	ErrorText:             ui.RGB(233, 83, 80),
	ErrorTextActive:       ui.RGB(190, 55, 56),
	ErrorTextHover:        ui.RGB(255, 168, 160),
	Fill:                  ui.RGBA(255, 255, 255, 0.16),
	FillContent:           ui.RGBA(255, 255, 255, 0.1),
	FillQuaternary:        ui.RGBA(255, 255, 255, 0.02),
	FillSecondary:         ui.RGBA(255, 255, 255, 0.1),
	FillTertiary:          ui.RGBA(255, 255, 255, 0.06),
	Info:                  ui.RGB(76, 179, 255),
	InfoFillQuaternary:    ui.RGBA(60, 150, 240, 0.1),
	InfoFillTertiary:      ui.RGBA(0, 124, 252, 0.21),
	InfoText:              ui.RGB(76, 179, 255),
	Primary:               ui.RGB(238, 238, 238),
	PrimaryActive:         ui.RGB(204, 204, 204),
	PrimaryBg:             ui.RGB(17, 17, 17),
	PrimaryBorder:         ui.RGB(102, 102, 102),
	PrimaryFillQuaternary: ui.RGBA(255, 255, 255, 0.02),
	PrimaryFillTertiary:   ui.RGBA(255, 255, 255, 0.08),
	PrimaryHover:          ui.RGB(255, 255, 255),
	Success:               ui.RGB(128, 201, 51),
	SuccessBg:             ui.RGB(6, 31, 0),
	SuccessFillQuaternary: ui.RGBA(143, 229, 57, 0.07),
	SuccessFillTertiary:   ui.RGBA(46, 238, 0, 0.13),
	Text:                  ui.RGB(230, 230, 230),
	TextDescription:       ui.RGB(143, 143, 143),
	TextDisabled:          ui.RGB(107, 107, 107),
	TextLabel:             ui.RGB(180, 180, 180),
	TextPlaceholder:       ui.RGB(107, 107, 107),
	TextQuaternary:        ui.RGB(107, 107, 107),
	TextSecondary:         ui.RGB(180, 180, 180),
	TextTertiary:          ui.RGB(143, 143, 143),
	Warning:               ui.RGB(255, 190, 69),
	WarningFillQuaternary: ui.RGBA(233, 144, 22, 0.09),
	WarningFillTertiary:   ui.RGBA(247, 153, 0, 0.15),
	Cyan:                  ui.RGB(149, 243, 217),
	CyanActive:            ui.RGB(85, 188, 164),
	CyanFillQuaternary:    ui.RGBA(0, 250, 150, 0.02),
	CyanFillTertiary:      ui.RGBA(0, 243, 200, 0.14),
	Geekblue:              ui.RGB(0, 114, 245),
	Gray:                  ui.RGB(119, 119, 119),
	GrayActive:            ui.RGB(102, 102, 102),
	GrayFillQuaternary:    ui.RGBA(255, 255, 255, 0.02),
	GrayFillTertiary:      ui.RGBA(255, 255, 255, 0.06),
	Green:                 ui.RGB(98, 196, 115),
	GreenActive:           ui.RGB(27, 141, 77),
	GreenFillQuaternary:   ui.RGBA(0, 250, 150, 0.02),
	GreenFillTertiary:     ui.RGBA(0, 242, 150, 0.12),
	Orange:                ui.RGB(255, 153, 39),
	OrangeActive:          ui.RGB(232, 139, 0),
	OrangeFillQuaternary:  ui.RGBA(233, 100, 0, 0.03),
	OrangeFillTertiary:    ui.RGBA(241, 147, 0, 0.17),
	Purple:                ui.RGB(189, 84, 198),
	PurpleActive:          ui.RGB(154, 57, 158),
	PurpleFillQuaternary:  ui.RGBA(217, 0, 183, 0.06),
	PurpleFillTertiary:    ui.RGBA(242, 0, 221, 0.19),
	RedActive:             ui.RGB(228, 49, 101),
	RedFillTertiary:       ui.RGBA(255, 0, 95, 0.22),
	Volcano:               ui.RGB(236, 94, 65),
	Yellow9:               ui.RGB(255, 239, 92),
	BorderRadius:          8,
	BorderRadiusLG:        12,
	BorderRadiusSM:        6,
	FontSize:              14,
	FontSizeSM:            12,
}

func (t *uiTokens) Color(name string) (ui.Color, bool) {
	switch name {
	case "Blue":
		return t.Blue, true
	case "BlueActive":
		return t.BlueActive, true
	case "BlueFillQuaternary":
		return t.BlueFillQuaternary, true
	case "BlueFillTertiary":
		return t.BlueFillTertiary, true
	case "Cyan":
		return t.Cyan, true
	case "CyanActive":
		return t.CyanActive, true
	case "CyanFillQuaternary":
		return t.CyanFillQuaternary, true
	case "CyanFillTertiary":
		return t.CyanFillTertiary, true
	case "Gray":
		return t.Gray, true
	case "GrayActive":
		return t.GrayActive, true
	case "GrayFillQuaternary":
		return t.GrayFillQuaternary, true
	case "GrayFillTertiary":
		return t.GrayFillTertiary, true
	case "Green":
		return t.Green, true
	case "GreenActive":
		return t.GreenActive, true
	case "GreenFillQuaternary":
		return t.GreenFillQuaternary, true
	case "GreenFillTertiary":
		return t.GreenFillTertiary, true
	case "Orange":
		return t.Orange, true
	case "OrangeActive":
		return t.OrangeActive, true
	case "OrangeFillQuaternary":
		return t.OrangeFillQuaternary, true
	case "OrangeFillTertiary":
		return t.OrangeFillTertiary, true
	case "Purple":
		return t.Purple, true
	case "PurpleActive":
		return t.PurpleActive, true
	case "PurpleFillQuaternary":
		return t.PurpleFillQuaternary, true
	case "PurpleFillTertiary":
		return t.PurpleFillTertiary, true
	}
	return ui.Color{}, false
}

func lightPalette() *uiTokens {
	t := lightTokens
	return &t
}

func darkPalette() *uiTokens {
	t := darkTokens
	return &t
}

func mixColor(c, o ui.Color, t float32) ui.Color {
	t = max(0, min(t, 1))
	ca, oa := float64(c.A)/255, float64(o.A)/255
	a := ca*(1-float64(t)) + oa*float64(t)
	if a == 0 {
		return ui.Transparent
	}
	ch := func(x, y uint8) uint8 {
		v := (float64(x)*ca*(1-float64(t)) + float64(y)*oa*float64(t)) / a
		return uint8(math.Round(v))
	}
	return ui.Color{R: ch(c.R, o.R), G: ch(c.G, o.G), B: ch(c.B, o.B), A: uint8(math.Round(a * 255))}
}

func (t *uiTokens) FillBorder() ui.Color {
	if t.Dark {
		return t.FillSecondary
	}
	return t.Fill
}

func (t *uiTokens) Theme() *ui.Theme {
	return &ui.Theme{
		Dark:            t.Dark,
		Background:      t.BgLayout,
		Surface:         t.FillTertiary,
		SurfaceHover:    t.FillSecondary,
		SurfacePressed:  t.Fill,
		Border:          t.FillBorder(),
		Text:            t.Text,
		TextMuted:       t.TextSecondary,
		Accent:          t.Primary,
		AccentHover:     t.PrimaryHover,
		AccentPressed:   t.PrimaryActive,
		AccentText:      t.BgLayout,
		Danger:          t.Error,
		Warning:         t.Warning,
		Success:         t.Success,
		Selection:       t.Yellow9,
		Focus:           t.Info.Alpha(0.9),
		FocusRingWidth:  2,
		FocusRingOffset: 2,
		Scrollbar:       t.TextQuaternary,
		ScrollbarWidth:  6,
		Radius:          t.BorderRadius,
		Spacing:         4,
		FontSize:        t.FontSize,
	}
}

func (t *uiTokens) Elevate(e ui.Element, level uiElevation) ui.Element {
	for _, l := range shadowLayers[level] {
		a := l.light
		if t.Dark {
			a = l.dark
		}
		e.Shadow(0, l.y, l.blur, l.spread, ui.RGBA(0, 0, 0, a))
	}
	if t.Dark {
		return e.Shadow(0, 0, 0, 1, ui.RGBA(255, 255, 255, 0.18))
	}
	return e.Shadow(0, 0, 0, 1, ui.RGBA(0, 0, 0, 0.05))
}

var (
	uiLightTheme = lightTokens.Theme()
	uiDarkTheme  = darkTokens.Theme()
)

func applyUITheme(c *ui.Context) {
	if c.Theme().Dark {
		c.SetTheme(uiDarkTheme)
	} else {
		c.SetTheme(uiLightTheme)
	}
}

func tokensOf(c *ui.Context) *uiTokens {
	if c.Theme().Dark {
		return darkPalette()
	}
	return lightPalette()
}

// Motion curves, springs, and element transitions shared across native UI primitives.
var (
	softEase        = ui.CubicBezier(0.32, 0.72, 0, 1)
	exitEase        = ui.CubicBezier(0.4, 0, 1, 1)
	easeOut         = ui.CubicBezier(0.215, 0.61, 0.355, 1)
	cssEase         = ui.CubicBezier(0.25, 0.1, 0.25, 1)
	cssEaseIn       = ui.CubicBezier(0.42, 0, 1, 1)
	toastEase       = ui.CubicBezier(0.22, 1, 0.36, 1)
	switchThumbAnim = ui.Animation{Spring: ui.SpringOptions{Stiffness: 360, Damping: 24, Mass: 1}}
)

type motionSide int

const (
	sideBottom motionSide = iota
	sideTop
)

func popupTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     150 * time.Millisecond,
		Ease:         easeOut,
		ExitDuration: 75 * time.Millisecond,
		ExitEase:     cssEaseIn,
		Enter:        &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
		Exit:         &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
	}
}

func tooltipTransition() ui.ElementTransition {
	enter := bouncySpring(280 * time.Millisecond)
	return ui.ElementTransition{
		Animation:    &enter,
		ExitDuration: 100 * time.Millisecond,
		ExitEase:     exitEase,
		Enter:        &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
		Exit:         &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
	}
}

func modalTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     220 * time.Millisecond,
		Ease:         softEase,
		ExitDuration: 120 * time.Millisecond,
		ExitEase:     exitEase,
		Enter:        &ui.Motion{Scale: 0.97},
		Exit:         &ui.Motion{Scale: 0.98},
	}
}

func backdropTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Ease:     softEase,
		Enter:    &ui.Motion{},
		Exit:     &ui.Motion{},
	}
}

func toastTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 400 * time.Millisecond,
		Ease:     toastEase,
		Enter:    &ui.Motion{YPercent: 150},
		Exit:     &ui.Motion{YPercent: 150},
	}
}

func collapseTransition() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Ease:     easeOut,
		Enter:    &ui.Motion{Collapse: true, Y: 6},
		Exit:     &ui.Motion{Collapse: true, Y: -6},
	}
}

func towardSide(tr ui.ElementTransition, side motionSide) ui.ElementTransition {
	turn := func(m *ui.Motion) *ui.Motion {
		if m == nil {
			return nil
		}
		t := *m
		if side == sideTop {
			t.Y = -m.Y
		}
		return &t
	}
	tr.Enter, tr.Exit = turn(tr.Enter), turn(tr.Exit)
	return tr
}

func bouncySpring(settle time.Duration) ui.Animation { return springAnim(0.7, 0.003, settle) }

func springAnim(zeta, threshold float64, settle time.Duration) ui.Animation {
	omega := unitSettle(zeta, threshold) / settle.Seconds()
	return ui.Animation{Spring: ui.SpringOptions{
		Stiffness: omega * omega,
		Damping:   2 * zeta * omega,
		Mass:      1,
	}}
}

func unitSettle(zeta, threshold float64) float64 {
	const step, horizon = 0.0005, 60.0
	last := 0.0
	for t := 0.0; t < horizon; t += step {
		if math.Abs(stepResponse(zeta, t)-1) > threshold {
			last = t
		}
	}
	return last
}

func stepResponse(zeta, t float64) float64 {
	if zeta >= 1 {
		return 1 - math.Exp(-t)*(1+t)
	}
	wd := math.Sqrt(1 - zeta*zeta)
	return 1 - math.Exp(-zeta*t)*(math.Cos(wd*t)+zeta/wd*math.Sin(wd*t))
}

func boolToFloat(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
