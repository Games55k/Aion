package action

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

var solidGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect width="24" height="24" fill="currentColor"/></svg>`))

type actionIconHarness struct {
	t       *testing.T
	tt      *ui.Tester
	clicked bool
}

func newActionIconHarness(t *testing.T, p ActionIconProps) *actionIconHarness {
	h := &actionIconHarness{t: t}
	if p.Title == "" {
		p.Title = "Settings"
	}
	h.tt = ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Center().Children(func() {
			if ActionIcon(c, solidGlyph, p).Clicked() {
				h.clicked = true
			}
		})
	}, 200, 200)
	return h
}

func (h *actionIconHarness) bounds() ui.Rect {
	h.t.Helper()
	info, ok := h.tt.AccessByRole(ui.RoleButton, "Settings")
	if !ok {
		h.t.Fatalf("no button named Settings")
	}
	return info.Bounds
}

func (h *actionIconHarness) pixel(x, y float32) ui.Color {
	img := h.tt.Image()
	scale := float32(img.Bounds().Dx()) / 200
	px := img.RGBAAt(int(x*scale), int(y*scale))
	return ui.RGB(px.R, px.G, px.B)
}

func (h *actionIconHarness) surface() ui.Color {
	b := h.bounds()
	return h.pixel(b.X+2, b.Y+b.H/2)
}

func (h *actionIconHarness) glyph() ui.Color {
	b := h.bounds()
	return h.pixel(b.X+b.W/2, b.Y+b.H/2)
}

func TestActionIconSizes(t *testing.T) {
	for _, c := range []struct {
		size theme.Size
		want float32
	}{{theme.Small, 24}, {theme.Middle, 36}, {theme.Large, 44}} {
		b := newActionIconHarness(t, ActionIconProps{Size: c.size}).bounds()
		if b.W != c.want || b.H != c.want {
			t.Errorf("size %v = %vx%v, want %v", c.size, b.W, b.H, c.want)
		}
	}
}

func TestActionIconGlyphColorFollowsState(t *testing.T) {
	tk := theme.Light()
	h := newActionIconHarness(t, ActionIconProps{})
	if got := h.glyph(); !testkit.Near(got, tk.TextTertiary) {
		t.Errorf("rest glyph = %+v, want TextTertiary %+v", got, tk.TextTertiary)
	}
	b := h.bounds()
	h.tt.Move(b.X+b.W/2, b.Y+b.H/2)
	h.tt.Advance(time.Second)
	if got := h.glyph(); !testkit.Near(got, tk.TextSecondary) {
		t.Errorf("hover glyph = %+v, want TextSecondary %+v", got, tk.TextSecondary)
	}
	if got, want := h.surface(), testkit.Over(tk.BgLayout, tk.FillSecondary); !testkit.Near(got, want) {
		t.Errorf("hover surface = %+v, want FillSecondary over BgLayout %+v", got, want)
	}
}

func TestActionIconDangerGlyph(t *testing.T) {
	tk := theme.Light()
	h := newActionIconHarness(t, ActionIconProps{Danger: true})
	if got, want := h.glyph(), testkit.Over(tk.BgLayout, tk.Error.Alpha(0.66)); !testkit.Near(got, want) {
		t.Errorf("danger glyph = %+v, want Error 66%% over BgLayout %+v", got, want)
	}
}

func TestActionIconFilledAndActiveSurfaces(t *testing.T) {
	tk := theme.Light()
	if got, want := newActionIconHarness(t, ActionIconProps{Variant: ActionIconFilled}).surface(), testkit.Over(tk.BgLayout, tk.FillTertiary); !testkit.Near(got, want) {
		t.Errorf("filled surface = %+v, want %+v", got, want)
	}
	if got, want := newActionIconHarness(t, ActionIconProps{Active: true}).surface(), testkit.Over(tk.BgLayout, tk.FillSecondary); !testkit.Near(got, want) {
		t.Errorf("active surface = %+v, want %+v", got, want)
	}
}

func TestActionIconClicks(t *testing.T) {
	h := newActionIconHarness(t, ActionIconProps{})
	b := h.bounds()
	h.tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	if !h.clicked {
		t.Error("Clicked() did not report the click")
	}
}

func TestActionIconTitleShowsTheLobeTooltip(t *testing.T) {
	tk := theme.Light()
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		ActionIcon(c, solidGlyph, ActionIconProps{Title: "Copy"})
	}), 300, 200)
	tt.Move(150, 100)
	tt.Advance(time.Second)
	if got := testkit.PixelAt(tt, 300, 150, 100-18-6-1.5); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("tooltip surface above the icon %+v, want BgElevated %+v", got, tk.BgElevated)
	}
}
