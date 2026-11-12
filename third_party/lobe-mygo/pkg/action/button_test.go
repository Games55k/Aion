package action

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

type buttonHarness struct {
	t       *testing.T
	tt      *ui.Tester
	clicked bool
}

func newButtonHarness(t *testing.T, props ButtonProps) *buttonHarness {
	h := &buttonHarness{t: t}
	h.tt = ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Center().Children(func() {
			if Button(c, "Save", props).Clicked() {
				h.clicked = true
			}
		})
	}, 320, 200)
	return h
}

func (h *buttonHarness) bounds() ui.Rect {
	h.t.Helper()
	info, ok := h.tt.AccessByRole(ui.RoleButton, "Save")
	if !ok {
		h.t.Fatalf("no button named Save; texts %q", h.tt.Texts())
	}
	return info.Bounds
}

func (h *buttonHarness) fill() ui.Color {
	h.t.Helper()
	b := h.bounds()
	img := h.tt.Image()
	scale := float32(img.Bounds().Dx()) / 320
	px := img.RGBAAt(int((b.X+5)*scale), int((b.Y+b.H/2)*scale))
	return ui.RGB(px.R, px.G, px.B)
}

func (h *buttonHarness) center() (float32, float32) {
	b := h.bounds()
	return b.X + b.W/2, b.Y + b.H/2
}

func TestPrimaryButtonPaintsPrimary(t *testing.T) {
	h := newButtonHarness(t, ButtonProps{Variant: Primary})
	if got, want := h.fill(), theme.Light().Primary; !testkit.Near(got, want) {
		t.Errorf("fill = %+v, want Primary %+v", got, want)
	}
}

func TestPrimaryButtonFollowsDarkAppearance(t *testing.T) {
	h := newButtonHarness(t, ButtonProps{Variant: Primary})
	h.tt.SetDark(true)
	h.tt.Advance(time.Second)
	if got, want := h.fill(), theme.Dark().Primary; !testkit.Near(got, want) {
		t.Errorf("dark fill = %+v, want Primary %+v", got, want)
	}
}

func TestFilledButtonWashesOnHoverAndPress(t *testing.T) {
	tk := theme.Light()
	h := newButtonHarness(t, ButtonProps{Variant: Filled})
	if got, want := h.fill(), testkit.Over(tk.BgLayout, tk.FillTertiary); !testkit.Near(got, want) {
		t.Errorf("rest = %+v, want FillTertiary over BgLayout %+v", got, want)
	}

	x, y := h.center()
	h.tt.Move(x, y)
	h.tt.Advance(time.Second)
	if got, want := h.fill(), testkit.Over(tk.BgLayout, tk.FillSecondary); !testkit.Near(got, want) {
		t.Errorf("hover = %+v, want FillSecondary over BgLayout %+v", got, want)
	}

	h.tt.Press(x, y)
	h.tt.Advance(time.Second)
	if got, want := h.fill(), testkit.Over(tk.BgLayout, tk.Fill); !testkit.Near(got, want) {
		t.Errorf("pressed = %+v, want Fill over BgLayout %+v", got, want)
	}
}

func TestButtonHeightsMatchLobeSizes(t *testing.T) {
	for _, c := range []struct {
		size   theme.Size
		height float32
	}{{theme.Small, 24}, {theme.Middle, 32}, {theme.Large, 40}} {
		h := newButtonHarness(t, ButtonProps{Size: c.size})
		if got := h.bounds().H; got != c.height {
			t.Errorf("size %v height = %v, want %v", c.size, got, c.height)
		}
	}
}

func TestButtonReportsClicked(t *testing.T) {
	h := newButtonHarness(t, ButtonProps{})
	if err := h.tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	if !h.clicked {
		t.Error("Clicked() did not report the click")
	}
}
