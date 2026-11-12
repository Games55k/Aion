package choice

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func sliderTester(v *float64, p SliderProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(40, 50).Width(300).Children(func() { Slider(c, v, p).Label("volume") })
	}, 300, 100)
}

func TestSliderClickSetsValue(t *testing.T) {
	v := 20.0
	tt := sliderTester(&v, SliderProps{})
	info, ok := tt.AccessByRole(ui.RoleSlider, "volume")
	if !ok {
		t.Fatal("no slider")
	}
	r := info.Bounds
	tt.ClickAt(r.X+4+(r.W-8)*0.75, r.Y+r.H/2)
	tt.Frame()
	if math.Abs(v-75) > 1.5 {
		t.Errorf("value %v after clicking at 75%%, want 75", v)
	}
}

func TestSliderStepSnaps(t *testing.T) {
	v := 0.0
	tt := sliderTester(&v, SliderProps{Max: 10, Step: 5})
	info, _ := tt.AccessByRole(ui.RoleSlider, "volume")
	r := info.Bounds
	tt.ClickAt(r.X+4+(r.W-8)*0.6, r.Y+r.H/2)
	tt.Frame()
	if v != 5 {
		t.Errorf("value %v after clicking at 60%% of 0–10 in steps of 5, want 5", v)
	}
}

func TestSliderDrawsTrackFillAndThumb(t *testing.T) {
	tk := theme.Light()
	v := 50.0
	tt := sliderTester(&v, SliderProps{})
	info, _ := tt.AccessByRole(ui.RoleSlider, "volume")
	r := info.Bounds
	cy := r.Y + r.H/2
	if got := testkit.PixelAt(tt, 300, r.X+r.W*0.25, cy); !testkit.Near(got, tk.Primary) {
		t.Errorf("track before the thumb %+v, want Primary", got)
	}
	if got := testkit.PixelAt(tt, 300, r.X+r.W*0.8, cy); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillSecondary)) {
		t.Errorf("track after the thumb %+v, want FillSecondary", got)
	}
	if got := testkit.PixelAt(tt, 300, r.X+r.W/2, cy-6); !testkit.Near(got, tk.Primary) {
		t.Errorf("thumb 6 DIPs above the track %+v, want the 16 DIP tall Primary thumb", got)
	}
}
