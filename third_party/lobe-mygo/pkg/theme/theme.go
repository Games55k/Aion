package theme

import (
	"math"

	"github.com/egoist/mygo/ui"
)

func Mix(c, o ui.Color, t float32) ui.Color {
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

func (t *Tokens) FillBorder() ui.Color {
	if t.Dark {
		return t.FillSecondary
	}
	return t.Fill
}

func (t *Tokens) FillBorderStrong() ui.Color {
	if t.Dark {
		return t.Fill
	}
	return Mix(t.Fill, t.Text, 0.15)
}

func (t *Tokens) Theme() *ui.Theme {
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
