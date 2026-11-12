package overlay

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func tooltipTester(text string, p TooltipProps) *ui.Tester {
	return ui.NewTester(testkit.Centered(func(c *ui.Context) {
		anchor := ui.Box(c).Size(80, 32).Background(ui.RGB(200, 200, 200))
		Tooltip(c, anchor, text, p)
	}), 400, 300)
}

func hoverAnchor(tt *ui.Tester) {
	tt.Move(200, 150)
	tt.Advance(time.Second)
}

func TestTooltipShowsOnTheSideAsked(t *testing.T) {
	anchorTop, anchorBottom := float32(150-16), float32(150+16)
	for _, c := range []struct {
		placement Placement
		check     func(r ui.Rect) bool
	}{
		{Top, func(r ui.Rect) bool { return r.Y+r.H < anchorTop-6 && r.Y+r.H > anchorTop-20 }},
		{Bottom, func(r ui.Rect) bool { return r.Y > anchorBottom+6 && r.Y < anchorBottom+20 }},
		{Left, func(r ui.Rect) bool { return r.X+r.W < 200-40-6 && r.Y < 150 && r.Y+r.H > 150 }},
		{Right, func(r ui.Rect) bool { return r.X > 200+40+6 && r.Y < 150 && r.Y+r.H > 150 }},
	} {
		tt := tooltipTester("Copy link", TooltipProps{Placement: c.placement})
		if _, shown := tt.Find("Copy link"); shown {
			t.Fatalf("placement %v: tip shows before the pointer rests on the anchor", c.placement)
		}
		hoverAnchor(tt)
		r, ok := tt.Find("Copy link")
		if !ok {
			t.Fatalf("placement %v: no tip after hovering", c.placement)
		}
		if !c.check(r) {
			t.Errorf("placement %v: tip text at %+v", c.placement, r)
		}
	}
}

func TestTooltipUsesElevatedSurfaceAndHotkey(t *testing.T) {
	tk := theme.Light()
	tt := tooltipTester("Search", TooltipProps{Hotkey: "mod+k"})
	hoverAnchor(tt)
	text, ok := tt.Find("Search")
	if !ok {
		t.Fatal("no tip text")
	}
	key, ok := tt.Find("K")
	if !ok {
		t.Fatalf("no hotkey in the tip: %q", tt.Texts())
	}
	if key.X <= text.X+text.W || key.Y > text.Y+text.H || key.Y+key.H < text.Y {
		t.Errorf("hotkey at %+v, want after the text %+v on its line", key, text)
	}
	if got := testkit.PixelAt(tt, 400, key.X+key.W+3, key.Y+key.H/2); !testkit.Near(got, testkit.Over(tk.BgElevated, tk.FillTertiary)) {
		t.Errorf("hotkey cap %+v, want FillTertiary on the tip", got)
	}
	if got := testkit.PixelAt(tt, 400, text.X-4, text.Y+text.H/2); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("tip padding = %+v, want BgElevated %+v", got, tk.BgElevated)
	}
	if text.H > 18 {
		t.Errorf("tip text is %v tall, want the small font size", text.H)
	}
}

func TestTooltipFadesOutAfterThePointerLeaves(t *testing.T) {
	tt := tooltipTester("Copy link", TooltipProps{})
	hoverAnchor(tt)
	r, _ := tt.Find("Copy link")
	x, y := r.X-4, r.Y+r.H/2
	bg := testkit.PixelAt(tt, 400, 5, 5)
	tt.Move(5, 5)
	tt.Advance(30 * time.Millisecond)
	if got := testkit.PixelAt(tt, 400, x, y); testkit.Near(got, bg) {
		t.Errorf("tip gone at once, want it fading out")
	}
	tt.Advance(time.Second)
	if got := testkit.PixelAt(tt, 400, x, y); !testkit.Near(got, bg) {
		t.Errorf("tip still painted %+v after its exit, want background %+v", got, bg)
	}
}

func TestTooltipMovingToAnotherAnchorKeepsTheTextOnItsLines(t *testing.T) {
	const long = "Anyone with the link can view this page"
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		ui.Row(c).Gap(12).Children(func() {
			save := ui.Box(c).Size(60, 32).Background(ui.RGB(200, 200, 200))
			Tooltip(c, save, "Save", TooltipProps{})
			share := ui.Box(c).Size(60, 32).Background(ui.RGB(200, 200, 200))
			Tooltip(c, share, long, TooltipProps{})
		})
	}), 600, 300)
	tt.Move(300-36, 150)
	tt.Advance(time.Second)
	if !tt.HasText("Save") {
		t.Fatal("no first tip")
	}
	tt.Move(300+36, 150)
	for i := 0; i < 6; i++ {
		tt.Advance(16 * time.Millisecond)
		r, ok := tt.Find(long)
		if !ok {
			t.Fatalf("frame %d: second tip not shown at once on a warm move", i)
		}
		if r.H > 18 {
			t.Fatalf("frame %d: second tip text wraps to %v DIPs tall while moving, want one line", i, r.H)
		}
	}
}
