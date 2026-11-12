package layout

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var rowColor = ui.RGB(60, 60, 60)

func scrollTester(p ScrollAreaProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Children(func() {
			ScrollArea(c, p, func() {
				ui.Column(c).Children(func() {
					for i := range 30 {
						ui.Box(c.Key(i)).Height(30).AlignSelf(ui.Stretch).Background(rowColor)
					}
				})
			}).Width(300).Height(200)
		})
	}, 300, 300)
}

func faded(tt *ui.Tester, y float32) bool {
	return !testkit.Near(testkit.PixelAt(tt, 300, 150, y), rowColor)
}

func TestScrollAreaFadesOnlyTheEdgesItCanScrollToward(t *testing.T) {
	tt := scrollTester(ScrollAreaProps{Fade: true})
	tt.Frame()
	if faded(tt, 1) {
		t.Error("at the top: top edge faded with nothing above")
	}
	if !faded(tt, 199) {
		t.Error("at the top: bottom edge not faded with more below")
	}
	if faded(tt, 100) {
		t.Error("middle of the viewport faded")
	}
	tt.Scroll(150, 100, 0, 300)
	tt.Advance(time.Second)
	if !faded(tt, 1) || !faded(tt, 199) {
		t.Errorf("scrolled to the middle: top faded %v, bottom faded %v, want both", faded(tt, 1), faded(tt, 199))
	}
	tt.Scroll(150, 100, 0, 5000)
	tt.Advance(time.Second)
	if !faded(tt, 1) || faded(tt, 199) {
		t.Errorf("at the end: top faded %v, bottom faded %v, want only the top", faded(tt, 1), faded(tt, 199))
	}
}

func TestScrollAreaWithoutFadeShowsItsEdges(t *testing.T) {
	tt := scrollTester(ScrollAreaProps{})
	tt.Frame()
	if faded(tt, 199) {
		t.Error("bottom edge faded without Fade")
	}
}

func thumbTester(state *ui.ScrollState) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Children(func() {
			ScrollArea(c, ScrollAreaProps{State: state}, func() {
				ui.Box(c).Height(900).AlignSelf(ui.Stretch)
			}).Width(300).Height(200)
		})
	}, 400, 300)
}

func thumbAt(tt *ui.Tester, x, y float32) bool {
	return testkit.Near(testkit.PixelAt(tt, 400, x, y), theme.Light().TextQuaternary)
}

func TestScrollAreaThumbShowsWhileScrollingThenFades(t *testing.T) {
	var st ui.ScrollState
	tt := thumbTester(&st)
	tt.Frame()
	if thumbAt(tt, 295.5, 10) {
		t.Fatal("thumb shows at rest with the pointer away")
	}
	tt.Scroll(150, 100, 0, 100)
	tt.Frame()
	if !thumbAt(tt, 295.5, 40) {
		t.Fatalf("thumb hidden right after a scroll (scrolled to %v)", st.Y)
	}
	tt.Move(380, 280)
	tt.Advance(300 * time.Millisecond)
	if !thumbAt(tt, 295.5, 40) {
		t.Error("thumb gone 300 ms after the scroll, want it held for 500 ms")
	}
	tt.Advance(time.Second)
	if thumbAt(tt, 295.5, 40) {
		t.Error("thumb still shows a second after the scroll stopped")
	}
}

func TestScrollAreaThumbSpansTheTrackByTheVisibleShare(t *testing.T) {
	var st ui.ScrollState
	tt := thumbTester(&st)
	tt.Frame()
	tt.Move(150, 100)
	tt.Advance(time.Second)
	if !thumbAt(tt, 295.5, 4) || !thumbAt(tt, 295.5, 45) || thumbAt(tt, 295.5, 48) {
		t.Errorf("at the top: thumb rows 4 %v, 45 %v, 48 %v; want it from 3 to 3+194*200/900", thumbAt(tt, 295.5, 4), thumbAt(tt, 295.5, 45), thumbAt(tt, 295.5, 48))
	}
	if thumbAt(tt, 297.5, 20) || thumbAt(tt, 292.5, 20) {
		t.Error("resting thumb wider than 3 DIPs 3 DIPs in from the edge")
	}
	st.Y = 1e9
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	if !thumbAt(tt, 295.5, 196) || thumbAt(tt, 295.5, 150) {
		t.Errorf("at the end: thumb at 196 %v, at 150 %v; want it ending 3 DIPs above the bottom", thumbAt(tt, 295.5, 196), thumbAt(tt, 295.5, 150))
	}
}

func TestScrollAreaThumbWidensUnderThePointer(t *testing.T) {
	var st ui.ScrollState
	tt := thumbTester(&st)
	tt.Frame()
	tt.Move(295, 20)
	tt.Advance(time.Second)
	if !thumbAt(tt, 292.5, 20) || thumbAt(tt, 290.5, 20) {
		t.Errorf("hovered thumb: x 292.5 %v, 290.5 %v; want it 6 DIPs wide", thumbAt(tt, 292.5, 20), thumbAt(tt, 290.5, 20))
	}
}

func TestScrollAreaDraggingTheThumbScrolls(t *testing.T) {
	var st ui.ScrollState
	tt := thumbTester(&st)
	tt.Frame()
	tt.Move(295, 20)
	tt.Advance(time.Second)
	tt.Press(295, 20)
	for i := 1; i <= 5; i++ {
		tt.Move(295, 20+float32(i)*10)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Release(295, 70)
	tt.Advance(100 * time.Millisecond)
	want := float32(50) * 700 / (194 - 194*200/900.0)
	if st.Y < want-2 || st.Y > want+2 {
		t.Errorf("dragging the thumb 50 DIPs scrolled to %v, want %v", st.Y, want)
	}
}
