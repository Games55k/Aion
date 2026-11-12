package display

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestSkeletonFillsAndPulses(t *testing.T) {
	tk := theme.Light()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(200).Children(func() { Skeleton(c, SkeletonProps{Height: 20}) })
	}, 200, 100)
	tt.Frame()
	rest := testkit.PixelAt(tt, 200, 100, 10)
	if !testkit.Near(rest, testkit.Over(tk.BgLayout, tk.FillContent)) {
		t.Errorf("skeleton %+v, want FillContent", rest)
	}
	tt.Advance(800 * time.Millisecond)
	if half := testkit.PixelAt(tt, 200, 100, 10); testkit.Near(half, rest) {
		t.Errorf("skeleton at the middle of its pulse %+v, want it faded from %+v", half, rest)
	}
	if !tt.FramePending() {
		t.Error("pulse stopped asking for frames")
	}
}

func TestSkeletonTextRows(t *testing.T) {
	tk := theme.Light()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(200).Children(func() { SkeletonText(c, SkeletonTextProps{Rows: 3, Animation: SkeletonStatic}) })
	}, 200, 100)
	fill := testkit.Over(tk.BgLayout, tk.FillContent)
	for i, y := range []float32{2 + 9, 22 + 2 + 9, 44 + 2 + 9} {
		if got := testkit.PixelAt(tt, 200, 50, y); !testkit.Near(got, fill) {
			t.Errorf("row %d at y %v: %+v, want FillContent", i, y, got)
		}
	}
	if got := testkit.PixelAt(tt, 200, 150, 44+2+9); testkit.Near(got, fill) {
		t.Errorf("last row reaches 150 of 200, want it 66%% wide")
	}
	if got := testkit.PixelAt(tt, 200, 150, 22+2+9); !testkit.Near(got, fill) {
		t.Errorf("middle row stops before 150, want it full width")
	}
}
