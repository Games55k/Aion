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
	boxShadow:          {{12, 32, 0, 0.10, 0.44}},
	boxShadowSecondary: {{8, 24, 0, 0.08, 0.36}},
	boxShadowTertiary:  {{2, 8, 0, 0.06, 0.24}},
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
	Blue:                  ui.RGB(0, 113, 227),
	BlueActive:            ui.RGB(0, 85, 172),
	BlueFillQuaternary:    ui.RGBA(0, 113, 227, 0.06),
	BlueFillTertiary:      ui.RGBA(0, 113, 227, 0.10),
	BgContainer:           ui.RGB(255, 255, 255),
	BgElevated:            ui.RGB(255, 255, 255),
	BgLayout:              ui.RGB(255, 255, 255),
	Border:                ui.RGBA(24, 24, 24, 0.14),
	BorderSecondary:       ui.RGBA(24, 24, 24, 0.08),
	Error:                 ui.RGB(201, 59, 66),
	ErrorActive:           ui.RGB(172, 44, 50),
	ErrorBg:               ui.RGBA(201, 59, 66, 0.08),
	ErrorBgActive:         ui.RGBA(201, 59, 66, 0.16),
	ErrorBgHover:          ui.RGBA(201, 59, 66, 0.12),
	ErrorBorder:           ui.RGBA(201, 59, 66, 0.24),
	ErrorBorderHover:      ui.RGBA(201, 59, 66, 0.36),
	ErrorFillQuaternary:   ui.RGBA(201, 59, 66, 0.06),
	ErrorFillTertiary:     ui.RGBA(201, 59, 66, 0.10),
	ErrorHover:            ui.RGB(218, 72, 79),
	ErrorText:             ui.RGB(185, 48, 55),
	ErrorTextActive:       ui.RGB(160, 38, 44),
	ErrorTextHover:        ui.RGB(201, 59, 66),
	Fill:                  ui.RGB(232, 232, 237),
	FillContent:           ui.RGB(239, 239, 242),
	FillQuaternary:        ui.RGB(245, 245, 247),
	FillSecondary:         ui.RGB(239, 239, 242),
	FillTertiary:          ui.RGB(245, 245, 247),
	Info:                  ui.RGB(0, 113, 227),
	InfoFillQuaternary:    ui.RGBA(0, 113, 227, 0.06),
	InfoFillTertiary:      ui.RGBA(0, 113, 227, 0.10),
	InfoText:              ui.RGB(0, 113, 227),
	Primary:               ui.RGB(0, 113, 227),
	PrimaryActive:         ui.RGB(0, 85, 172),
	PrimaryBg:             ui.RGBA(0, 113, 227, 0.10),
	PrimaryBorder:         ui.RGBA(0, 113, 227, 0.28),
	PrimaryFillQuaternary: ui.RGBA(0, 113, 227, 0.08),
	PrimaryFillTertiary:   ui.RGBA(0, 113, 227, 0.14),
	PrimaryHover:          ui.RGB(0, 98, 196),
	Success:               ui.RGB(0, 113, 227),
	SuccessBg:             ui.RGBA(0, 113, 227, 0.08),
	SuccessFillQuaternary: ui.RGBA(0, 113, 227, 0.06),
	SuccessFillTertiary:   ui.RGBA(0, 113, 227, 0.10),
	Text:                  ui.RGB(24, 24, 24),
	TextDescription:       ui.RGB(106, 106, 106),
	TextDisabled:          ui.RGB(158, 158, 164),
	TextLabel:             ui.RGB(106, 106, 106),
	TextPlaceholder:       ui.RGB(158, 158, 164),
	TextQuaternary:        ui.RGB(158, 158, 164),
	TextSecondary:         ui.RGB(106, 106, 106),
	TextTertiary:          ui.RGB(106, 106, 106),
	Warning:               ui.RGB(196, 102, 16),
	WarningFillQuaternary: ui.RGBA(217, 119, 6, 0.08),
	WarningFillTertiary:   ui.RGBA(217, 119, 6, 0.12),
	Cyan:                  ui.RGB(15, 118, 110),
	CyanActive:            ui.RGB(15, 118, 110),
	CyanFillQuaternary:    ui.RGBA(15, 118, 110, 0.07),
	CyanFillTertiary:      ui.RGBA(15, 118, 110, 0.11),
	Geekblue:              ui.RGB(0, 113, 227),
	Gray:                  ui.RGB(106, 106, 106),
	GrayActive:            ui.RGB(24, 24, 24),
	GrayFillQuaternary:    ui.RGB(245, 245, 247),
	GrayFillTertiary:      ui.RGB(239, 239, 242),
	Green:                 ui.RGB(36, 138, 61),
	GreenActive:           ui.RGB(28, 116, 49),
	GreenFillQuaternary:   ui.RGBA(36, 138, 61, 0.08),
	GreenFillTertiary:     ui.RGBA(36, 138, 61, 0.12),
	Orange:                ui.RGB(196, 102, 16),
	OrangeActive:          ui.RGB(178, 86, 8),
	OrangeFillQuaternary:  ui.RGBA(217, 119, 6, 0.08),
	OrangeFillTertiary:    ui.RGBA(217, 119, 6, 0.12),
	Purple:                ui.RGB(106, 106, 106),
	PurpleActive:          ui.RGB(24, 24, 24),
	PurpleFillQuaternary:  ui.RGB(245, 245, 247),
	PurpleFillTertiary:    ui.RGB(239, 239, 242),
	RedActive:             ui.RGB(196, 48, 56),
	RedFillTertiary:       ui.RGBA(201, 59, 66, 0.10),
	Volcano:               ui.RGB(0, 113, 227),
	Yellow9:               ui.RGBA(0, 113, 227, 0.18),
	BorderRadius:          12,
	BorderRadiusLG:        20,
	BorderRadiusSM:        8,
	FontSize:              14,
	FontSizeSM:            14,
}

var darkTokens = uiTokens{
	Dark:                  true,
	Blue:                  ui.RGB(41, 151, 255),
	BlueActive:            ui.RGB(10, 132, 255),
	BlueFillQuaternary:    ui.RGBA(41, 151, 255, 0.10),
	BlueFillTertiary:      ui.RGBA(41, 151, 255, 0.16),
	BgContainer:           ui.RGB(18, 18, 18),
	BgElevated:            ui.RGB(28, 28, 30),
	BgLayout:              ui.RGB(18, 18, 18),
	Border:                ui.RGBA(255, 255, 255, 0.16),
	BorderSecondary:       ui.RGBA(255, 255, 255, 0.09),
	Error:                 ui.RGB(224, 93, 99),
	ErrorActive:           ui.RGB(201, 70, 76),
	ErrorBg:               ui.RGBA(224, 93, 99, 0.14),
	ErrorBgActive:         ui.RGBA(224, 93, 99, 0.24),
	ErrorBgHover:          ui.RGBA(224, 93, 99, 0.18),
	ErrorBorder:           ui.RGBA(224, 93, 99, 0.30),
	ErrorBorderHover:      ui.RGBA(224, 93, 99, 0.44),
	ErrorFillQuaternary:   ui.RGBA(224, 93, 99, 0.10),
	ErrorFillTertiary:     ui.RGBA(224, 93, 99, 0.16),
	ErrorHover:            ui.RGB(240, 114, 120),
	ErrorText:             ui.RGB(235, 108, 114),
	ErrorTextActive:       ui.RGB(224, 93, 99),
	ErrorTextHover:        ui.RGB(245, 130, 136),
	Fill:                  ui.RGB(44, 44, 46),
	FillContent:           ui.RGB(36, 36, 38),
	FillQuaternary:        ui.RGB(28, 28, 30),
	FillSecondary:         ui.RGB(36, 36, 38),
	FillTertiary:          ui.RGB(28, 28, 30),
	Info:                  ui.RGB(41, 151, 255),
	InfoFillQuaternary:    ui.RGBA(41, 151, 255, 0.10),
	InfoFillTertiary:      ui.RGBA(41, 151, 255, 0.16),
	InfoText:              ui.RGB(41, 151, 255),
	Primary:               ui.RGB(41, 151, 255),
	PrimaryActive:         ui.RGB(10, 132, 255),
	PrimaryBg:             ui.RGBA(41, 151, 255, 0.14),
	PrimaryBorder:         ui.RGBA(41, 151, 255, 0.34),
	PrimaryFillQuaternary: ui.RGBA(41, 151, 255, 0.12),
	PrimaryFillTertiary:   ui.RGBA(41, 151, 255, 0.18),
	PrimaryHover:          ui.RGB(71, 167, 255),
	Success:               ui.RGB(41, 151, 255),
	SuccessBg:             ui.RGBA(41, 151, 255, 0.12),
	SuccessFillQuaternary: ui.RGBA(41, 151, 255, 0.10),
	SuccessFillTertiary:   ui.RGBA(41, 151, 255, 0.16),
	Text:                  ui.RGB(242, 242, 242),
	TextDescription:       ui.RGB(152, 152, 157),
	TextDisabled:          ui.RGB(102, 102, 107),
	TextLabel:             ui.RGB(152, 152, 157),
	TextPlaceholder:       ui.RGB(102, 102, 107),
	TextQuaternary:        ui.RGB(102, 102, 107),
	TextSecondary:         ui.RGB(152, 152, 157),
	TextTertiary:          ui.RGB(152, 152, 157),
	Warning:               ui.RGB(255, 169, 56),
	WarningFillQuaternary: ui.RGBA(255, 159, 10, 0.12),
	WarningFillTertiary:   ui.RGBA(255, 159, 10, 0.18),
	Cyan:                  ui.RGB(90, 200, 190),
	CyanActive:            ui.RGB(100, 210, 200),
	CyanFillQuaternary:    ui.RGBA(64, 200, 190, 0.12),
	CyanFillTertiary:      ui.RGBA(64, 200, 190, 0.18),
	Geekblue:              ui.RGB(41, 151, 255),
	Gray:                  ui.RGB(152, 152, 157),
	GrayActive:            ui.RGB(242, 242, 242),
	GrayFillQuaternary:    ui.RGB(28, 28, 30),
	GrayFillTertiary:      ui.RGB(36, 36, 38),
	Green:                 ui.RGB(48, 209, 88),
	GreenActive:           ui.RGB(66, 216, 104),
	GreenFillQuaternary:   ui.RGBA(48, 209, 88, 0.12),
	GreenFillTertiary:     ui.RGBA(48, 209, 88, 0.18),
	Orange:                ui.RGB(255, 169, 56),
	OrangeActive:          ui.RGB(255, 179, 64),
	OrangeFillQuaternary:  ui.RGBA(255, 159, 10, 0.12),
	OrangeFillTertiary:    ui.RGBA(255, 159, 10, 0.18),
	Purple:                ui.RGB(152, 152, 157),
	PurpleActive:          ui.RGB(242, 242, 242),
	PurpleFillQuaternary:  ui.RGB(28, 28, 30),
	PurpleFillTertiary:    ui.RGB(36, 36, 38),
	RedActive:             ui.RGB(255, 105, 97),
	RedFillTertiary:       ui.RGBA(255, 69, 58, 0.16),
	Volcano:               ui.RGB(41, 151, 255),
	Yellow9:               ui.RGBA(41, 151, 255, 0.24),
	BorderRadius:          12,
	BorderRadiusLG:        20,
	BorderRadiusSM:        8,
	FontSize:              14,
	FontSizeSM:            14,
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

func lightPalette() *uiTokens { return &lightTokens }

func darkPalette() *uiTokens { return &darkTokens }

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
	switchThumbAnim = ui.Animation{Spring: ui.SpringOptions{Stiffness: 360, Damping: 24, Mass: 1}}
)

type motionSide int

const (
	sideBottom motionSide = iota
	sideTop
)

var (
	tooltipEnterAnim = bouncySpring(280 * time.Millisecond)

	basePopupTransition = ui.ElementTransition{
		Duration:     150 * time.Millisecond,
		Ease:         easeOut,
		ExitDuration: 75 * time.Millisecond,
		ExitEase:     cssEaseIn,
		Enter:        &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
		Exit:         &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
	}
	baseTooltipTransition = ui.ElementTransition{
		Animation:    &tooltipEnterAnim,
		ExitDuration: 100 * time.Millisecond,
		ExitEase:     exitEase,
		Enter:        &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
		Exit:         &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
	}
	baseModalTransition = ui.ElementTransition{
		Duration:     220 * time.Millisecond,
		Ease:         softEase,
		ExitDuration: 120 * time.Millisecond,
		ExitEase:     exitEase,
		Enter:        &ui.Motion{Scale: 0.97},
		Exit:         &ui.Motion{Scale: 0.98},
	}
	baseBackdropTransition = ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Ease:     softEase,
		Enter:    &ui.Motion{},
		Exit:     &ui.Motion{},
	}
	baseCollapseTransition = ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Ease:     easeOut,
		Enter:    &ui.Motion{Collapse: true, Y: 6},
		Exit:     &ui.Motion{Collapse: true, Y: -6},
	}
)

func popupTransition() ui.ElementTransition    { return basePopupTransition }
func tooltipTransition() ui.ElementTransition  { return baseTooltipTransition }
func modalTransition() ui.ElementTransition    { return baseModalTransition }
func backdropTransition() ui.ElementTransition { return baseBackdropTransition }
func collapseTransition() ui.ElementTransition { return baseCollapseTransition }

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
