package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func progressTester(percent *float32, p ProgressProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(240).Children(func() { Progress(c, *percent, p) })
	}, 240, 140)
}

func TestProgressLineFillsAndAnimates(t *testing.T) {
	tk := theme.Light()
	pct := float32(0)
	tt := progressTester(&pct, ProgressProps{ShowInfo: true, Label: "Uploading"})
	pct = 40
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Frame()
	label, _ := tt.Find("Uploading")
	y := label.Y + label.H + 6 + 2
	mid := testkit.PixelAt(tt, 240, 20+200*0.3, y)
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("40%") {
		t.Errorf("texts %q, want 40%%", tt.Texts())
	}
	if got := testkit.PixelAt(tt, 240, 20+200*0.3, y); !testkit.Near(got, tk.Primary) {
		t.Errorf("track at 30%% %+v, want filled", got)
	}
	if got := testkit.PixelAt(tt, 240, 20+200*0.6, y); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("track at 60%% %+v, want the empty track", got)
	}
	if testkit.Near(mid, tk.Primary) {
		t.Error("the bar reached 30% within 100 ms: want it growing over 300 ms")
	}
}

func TestProgressStatusAndCircle(t *testing.T) {
	pct := float32(100)
	tt := progressTester(&pct, ProgressProps{ShowInfo: true, Status: ProgressSuccess})
	if tt.HasText("100%") {
		t.Error("success shows the percent instead of a check")
	}
	pct = 75
	tt = progressTester(&pct, ProgressProps{Type: ProgressCircle, ShowInfo: true})
	r, ok := tt.Find("75%")
	if !ok {
		t.Fatalf("circle: %q", tt.Texts())
	}
	if cx := r.X + r.W/2; cx < 39 || cx > 41 {
		t.Errorf("circle info at x %v, want centered in the 40 DIP circle", cx)
	}
}

func TestProgressSegments(t *testing.T) {
	tk := theme.Light()
	pct := float32(50)
	tt := progressTester(&pct, ProgressProps{Variant: ProgressSegments, Segments: 10})
	tt.Advance(time.Second)
	tt.Frame()
	step := float32(200+2) / 10
	if got := testkit.PixelAt(tt, 240, 20+step*4+step/2-1, 24); !testkit.Near(got, tk.Primary) {
		t.Errorf("5th segment %+v, want filled", got)
	}
	if got := testkit.PixelAt(tt, 240, 20+step*5+step/2-1, 24); testkit.Near(got, tk.Primary) {
		t.Error("6th segment filled at 50%")
	}
}

func TestProgressInsetFillsToThePercent(t *testing.T) {
	tk := theme.Light()
	pct := float32(65)
	tt := progressTester(&pct, ProgressProps{Variant: ProgressInset})
	if got := testkit.PixelAt(tt, 240, 20+200*0.5, 24); !testkit.Near(got, tk.Primary) {
		t.Errorf("inset at 50%% of 65%% %+v, want filled on the first frame", got)
	}
	if got := testkit.PixelAt(tt, 240, 20+200*0.8, 24); testkit.Near(got, tk.Primary) {
		t.Error("inset filled at 80%")
	}
}
