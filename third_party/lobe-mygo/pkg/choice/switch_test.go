package choice

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func switchBounds(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleSwitch, "Wi-Fi")
	if !ok {
		t.Fatalf("no switch named Wi-Fi")
	}
	return info.Bounds
}

func TestSwitchSizes(t *testing.T) {
	for _, c := range []struct {
		size theme.Size
		w, h float32
	}{{theme.Middle, 36, 22}, {theme.Small, 28, 16}} {
		on := false
		tt := ui.NewTester(testkit.Centered(func(ctx *ui.Context) {
			Switch(ctx, &on, SwitchProps{Size: c.size}).Label("Wi-Fi")
		}), 200, 100)
		if b := switchBounds(t, tt); b.W != c.w || b.H != c.h {
			t.Errorf("size %v = %vx%v, want %vx%v", c.size, b.W, b.H, c.w, c.h)
		}
	}
}

func TestSwitchTogglesTrackAndThumb(t *testing.T) {
	tk := theme.Light()
	on := false
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Switch(c, &on, SwitchProps{}).Label("Wi-Fi")
	}), 200, 100)
	b := switchBounds(t, tt)
	right := func() ui.Color { return testkit.PixelAt(tt, 200, b.X+b.W-6, b.Y+b.H/2) }
	left := func() ui.Color { return testkit.PixelAt(tt, 200, b.X+6, b.Y+b.H/2) }
	if got := left(); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("off: thumb side = %+v, want BgContainer thumb", got)
	}
	if got, want := right(), testkit.Over(tk.BgLayout, tk.FillSecondary); !testkit.Near(got, want) {
		t.Errorf("off: track = %+v, want FillSecondary over BgLayout %+v", got, want)
	}
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Move(0, 0)
	tt.Advance(time.Second)
	if !on {
		t.Fatal("click did not turn the switch on")
	}
	if got := right(); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("on: thumb side = %+v, want BgContainer thumb", got)
	}
	if got := left(); !testkit.Near(got, tk.Primary) {
		t.Errorf("on: track = %+v, want Primary %+v", got, tk.Primary)
	}
}
