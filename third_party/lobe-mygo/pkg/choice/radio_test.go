package choice

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestRadioGroupSelectsOneValue(t *testing.T) {
	tk := theme.Light()
	size := "m"
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		RadioGroup(c, RadioGroupProps{}, func() {
			Radio(c, &size, "s", "Small", RadioProps{})
			Radio(c, &size, "m", "Medium", RadioProps{})
			Radio(c, &size, "l", "Large", RadioProps{})
		})
	}), 400, 100)
	if err := tt.Click("Large"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if size != "l" {
		t.Fatalf("size %q after clicking Large", size)
	}
	l, _ := tt.Find("Large")
	m, _ := tt.Find("Medium")
	dot := func(r ui.Rect) (ui.Color, ui.Color) {
		cx, cy := r.X-8-8, r.Y+r.H/2
		return testkit.PixelAt(tt, 400, cx, cy), testkit.PixelAt(tt, 400, cx+5.5, cy)
	}
	if center, ring := dot(l); !testkit.Near(center, tk.BgLayout) || !testkit.Near(ring, tk.Primary) {
		t.Errorf("checked radio: center %+v ring %+v, want a BgLayout dot on Primary", center, ring)
	}
	if center, _ := dot(m); !testkit.Near(center, tk.BgContainer) {
		t.Errorf("unchecked radio center %+v, want BgContainer", center)
	}
}

func TestRadioDotScalesInAndOut(t *testing.T) {
	tk := theme.Light()
	size := "m"
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		RadioGroup(c, RadioGroupProps{}, func() {
			Radio(c, &size, "m", "Medium", RadioProps{})
			Radio(c, &size, "l", "Large", RadioProps{})
		})
	}), 400, 100)
	center := func(label string) (float32, float32) {
		r, _ := tt.Find(label)
		return r.X - 16, r.Y + r.H/2
	}
	tt.Click("Large")
	tt.Advance(30 * time.Millisecond)
	tt.Frame()
	lx, ly := center("Large")
	if got := testkit.PixelAt(tt, 400, lx+1.8, ly); testkit.Near(got, tk.BgLayout) {
		t.Errorf("Large's dot 30 ms after choosing it reaches 1.8 DIPs out (%+v): want it still growing", got)
	}
	mx, my := center("Medium")
	if got := testkit.PixelAt(tt, 400, mx, my); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("Medium's dot gone 30 ms after leaving it (%+v): want it shrinking away", got)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 400, lx+1.8, ly); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("Large's dot settled %+v, want 6 DIPs of BgLayout", got)
	}
	if got := testkit.PixelAt(tt, 400, mx, my); testkit.Near(got, tk.BgLayout) {
		t.Errorf("Medium's dot still shows once settled")
	}
}
