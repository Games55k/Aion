package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMixPremultipliesAlphaAsCSSColorMix(t *testing.T) {
	cases := []struct {
		name string
		got  ui.Color
		want ui.Color
	}{
		{"red 10% into transparent keeps its hue", Mix(ui.RGB(255, 0, 0), ui.Transparent, 0.9), ui.Color{R: 255, A: 26}},
		{"translucent fill 85% with text 15%", Mix(ui.Color{A: 31}, ui.Hex("#1f1f1f"), 0.15), ui.Color{R: 18, G: 18, B: 18, A: 65}},
		{"opaque 88% with white", Mix(ui.Hex("#000000"), ui.RGB(255, 255, 255), 0.12), ui.RGB(31, 31, 31)},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %+v, want %+v", c.name, c.got, c.want)
		}
	}
}

func TestFillBorderFollowsAppearance(t *testing.T) {
	l, d := Light(), Dark()
	if got := l.FillBorder(); got != l.Fill {
		t.Errorf("light FillBorder = %+v, want Fill %+v", got, l.Fill)
	}
	if got := d.FillBorder(); got != d.FillSecondary {
		t.Errorf("dark FillBorder = %+v, want FillSecondary %+v", got, d.FillSecondary)
	}
	if got, want := l.FillBorderStrong(), Mix(l.Fill, l.Text, 0.15); got != want {
		t.Errorf("light FillBorderStrong = %+v, want %+v", got, want)
	}
	if got := d.FillBorderStrong(); got != d.Fill {
		t.Errorf("dark FillBorderStrong = %+v, want Fill %+v", got, d.Fill)
	}
}

func TestThemeMapsLobeTokensOntoMygoTheme(t *testing.T) {
	for _, tk := range []*Tokens{Light(), Dark()} {
		th := tk.Theme()
		checks := []struct {
			field     string
			got, want ui.Color
		}{
			{"Background", th.Background, tk.BgLayout},
			{"Text", th.Text, tk.Text},
			{"TextMuted", th.TextMuted, tk.TextSecondary},
			{"Accent", th.Accent, tk.Primary},
			{"AccentHover", th.AccentHover, tk.PrimaryHover},
			{"AccentPressed", th.AccentPressed, tk.PrimaryActive},
			{"AccentText", th.AccentText, tk.BgLayout},
			{"Border", th.Border, tk.FillBorder()},
			{"Danger", th.Danger, tk.Error},
			{"Focus", th.Focus, tk.Info.Alpha(0.9)},
			{"Selection", th.Selection, tk.Yellow9},
		}
		for _, c := range checks {
			if c.got != c.want {
				t.Errorf("dark=%v %s = %+v, want %+v", tk.Dark, c.field, c.got, c.want)
			}
		}
		if th.Dark != tk.Dark || th.Radius != tk.BorderRadius || th.FontSize != tk.FontSize {
			t.Errorf("dark=%v metrics: Dark %v Radius %v FontSize %v", tk.Dark, th.Dark, th.Radius, th.FontSize)
		}
		if th.FocusRingWidth != 2 || th.FocusRingOffset != 2 {
			t.Errorf("focus ring = %v/%v, want 2/2", th.FocusRingWidth, th.FocusRingOffset)
		}
	}
}

func TestThemeScrollbarUsesQuaternaryText(t *testing.T) {
	for _, tk := range []*Tokens{Light(), Dark()} {
		th := tk.Theme()
		if th.Scrollbar != tk.TextQuaternary || th.ScrollbarWidth != 6 {
			t.Errorf("dark %v: scrollbar %+v width %v, want TextQuaternary %+v at 6", tk.Dark, th.Scrollbar, th.ScrollbarWidth, tk.TextQuaternary)
		}
	}
}
