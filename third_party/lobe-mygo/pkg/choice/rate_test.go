package choice

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func rateTester(v *float64, p RateProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { Rate(c, v, p).Label("rating") })
	}, 300, 100)
}

func star(tt *ui.Tester, i int, at float32) (float32, float32) {
	info, _ := tt.AccessByRole(ui.RoleSlider, "rating")
	return info.Bounds.X + float32(i)*30 + 20*at, info.Bounds.Y + 10
}

func TestRateClicksSetValue(t *testing.T) {
	v := 0.0
	tt := rateTester(&v, RateProps{})
	if info, ok := tt.AccessByRole(ui.RoleSlider, "rating"); !ok || info.Bounds.W != 5*20+4*10 {
		t.Fatalf("rate %+v, want five 20 DIP stars 10 apart", info.Bounds)
	}
	tt.ClickAt(star(tt, 3, 0.3))
	tt.Frame()
	if v != 4 {
		t.Errorf("value %v after clicking the 4th star, want 4", v)
	}
	v = 0
	tt = rateTester(&v, RateProps{AllowHalf: true})
	tt.ClickAt(star(tt, 2, 0.3))
	tt.Frame()
	if v != 2.5 {
		t.Errorf("half: value %v after clicking the left half of the 3rd star, want 2.5", v)
	}
}

func TestRateFillsStarsAndPreviewsOnHover(t *testing.T) {
	tk := theme.Light()
	v := 2.0
	tt := rateTester(&v, RateProps{})
	if got := starPixel(tt, 1, 0.5); !testkit.Near(got, tk.Warning) {
		t.Errorf("2nd star %+v, want Warning", got)
	}
	if got := starPixel(tt, 3, 0.5); testkit.Near(got, tk.Warning) {
		t.Error("4th star filled at 2")
	}
	tt.Move(star(tt, 3, 0.5))
	tt.Frame()
	if got := starPixel(tt, 3, 0.5); !testkit.Near(got, tk.Warning) {
		t.Errorf("4th star under the pointer %+v, want the preview fill", got)
	}
	if v != 2 {
		t.Errorf("hovering changed the value to %v", v)
	}
}

func starPixel(tt *ui.Tester, i int, at float32) ui.Color {
	x, y := star(tt, i, at)
	return testkit.PixelAt(tt, 300, x, y)
}

func TestRateKeys(t *testing.T) {
	v := 3.0
	tt := rateTester(&v, RateProps{})
	tt.ClickAt(star(tt, 2, 0.5))
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if v != 4 {
		t.Errorf("value %v after Right, want 4", v)
	}
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	if v != 0 {
		t.Errorf("value %v after Home, want 0", v)
	}
}
