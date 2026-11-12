package layout

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func panelTester(size *float32, expanded *bool, p DraggablePanelProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			main := func() { ui.Text(c, "main").Grow(1) }
			if p.Placement == Left {
				DraggablePanel(c, size, expanded, p, func() { ui.Text(c, "panel") })
				main()
				return
			}
			main()
			DraggablePanel(c, size, expanded, p, func() { ui.Text(c, "panel") })
		})
	}, 800, 300)
}

func drag(tt *ui.Tester, x0, x1, y float32) {
	tt.Press(x0, y)
	for i := 1; i <= 10; i++ {
		tt.Move(x0+(x1-x0)*float32(i)/10, y)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Release(x1, y)
	tt.Advance(time.Second)
}

func TestDraggablePanelTakesItsSizeAtItsEdge(t *testing.T) {
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{})
	tt.Frame()
	r, ok := tt.Find("panel")
	if !ok || r.X != 800-280 {
		t.Errorf("panel content at %+v (shown %v), want x 520", r, ok)
	}
}

func TestDraggablePanelResizesFromItsInnerEdge(t *testing.T) {
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{})
	tt.Frame()
	drag(tt, 520, 420, 60)
	if size < 379 || size > 381 {
		t.Errorf("dragging the right panel's edge 100 DIPs left: size %v, want 380", size)
	}
	if r, _ := tt.Find("panel"); r.X < 419 || r.X > 421 {
		t.Errorf("panel content at x %v, want 420", r.X)
	}

	size = 280
	tt = panelTester(&size, &expanded, DraggablePanelProps{Placement: Left})
	tt.Frame()
	drag(tt, 280, 330, 60)
	if size < 329 || size > 331 {
		t.Errorf("dragging the left panel's edge 50 DIPs right: size %v, want 330", size)
	}
}

func TestDraggablePanelKeepsBetweenMinAndMax(t *testing.T) {
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{MinSize: 200, MaxSize: 360})
	tt.Frame()
	drag(tt, 520, 300, 60)
	if size != 360 {
		t.Errorf("past the max: size %v, want 360", size)
	}
	drag(tt, 440, 700, 60)
	if size != 200 {
		t.Errorf("past the min: size %v, want 200", size)
	}
}

func TestDraggablePanelCollapsesWhenDraggedBelowItsThreshold(t *testing.T) {
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{CollapseThreshold: 120})
	tt.Frame()
	drag(tt, 520, 760, 60)
	if expanded {
		t.Fatal("dragged to 40 DIPs under a 120 threshold: still expanded")
	}
	if size != 280 {
		t.Errorf("collapsed panel keeps size %v, want 280 to reopen at", size)
	}
	if tt.HasText("panel") {
		if r, _ := tt.Find("panel"); r.X < 800 {
			t.Errorf("collapsed panel content still shows at x %v", r.X)
		}
	}
}

func TestDraggablePanelToggleCollapsesAndExpands(t *testing.T) {
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{})
	tt.Frame()
	tt.Move(600, 150)
	tt.Advance(time.Second)
	collapse, ok := tt.AccessByRole(ui.RoleButton, "Collapse panel")
	if !ok {
		t.Fatal("no Collapse panel button")
	}
	if b := collapse.Bounds; b.X+b.W/2 < 519 || b.X+b.W/2 > 521 {
		t.Errorf("toggle centered at x %v, want on the seam at 520", b.X+b.W/2)
	}
	tt.ClickAt(collapse.Bounds.X+collapse.Bounds.W/2, collapse.Bounds.Y+collapse.Bounds.H/2)
	tt.Advance(time.Second)
	if expanded {
		t.Fatal("toggle did not collapse the panel")
	}
	expand, ok := tt.AccessByRole(ui.RoleButton, "Expand panel")
	if !ok {
		t.Fatal("collapsed panel offers no Expand panel button")
	}
	tt.ClickAt(expand.Bounds.X+expand.Bounds.W/2, expand.Bounds.Y+expand.Bounds.H/2)
	tt.Advance(time.Second)
	if !expanded {
		t.Fatal("toggle did not expand the panel")
	}
	if r, _ := tt.Find("panel"); r.X != 520 {
		t.Errorf("reopened panel content at x %v, want 520", r.X)
	}
}

func TestDraggablePanelSeamLightsWhileResizing(t *testing.T) {
	tk := theme.Light()
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{})
	tt.Frame()
	if got, want := testkit.PixelAt(tt, 800, 520.5, 40), testkit.Over(tk.BgLayout, tk.BorderSecondary); !testkit.Near(got, want) {
		t.Errorf("idle seam = %+v, want BorderSecondary %+v", got, want)
	}
	tt.Press(520, 40)
	tt.Move(515, 40)
	tt.Advance(time.Second)
	if got := testkit.PixelAt(tt, 800, 515.5, 40); !testkit.Near(got, tk.Primary) {
		t.Errorf("seam while resizing = %+v, want Primary %+v", got, tk.Primary)
	}
	tt.Release(515, 40)
}

func TestDraggablePanelContentKeepsItsWidthWhileCollapsing(t *testing.T) {
	size, expanded := float32(280), true
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Text(c, "main").Grow(1)
			DraggablePanel(c, &size, &expanded, DraggablePanelProps{}, func() {
				ui.Box(c).Height(40).AlignSelf(ui.Stretch).Label("body").Role(ui.RoleGroup)
			})
		})
	}, 800, 300)
	tt.Frame()
	expanded = false
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Frame()
	info, ok := tt.AccessByRole(ui.RoleGroup, "body")
	if !ok {
		t.Fatal("no body")
	}
	if info.Bounds.W != 280 || info.Bounds.X <= 520 {
		t.Errorf("content mid-collapse at x %v, %v wide: want it sliding out 280 wide", info.Bounds.X, info.Bounds.W)
	}
}

func TestDraggablePanelContentStartsAtTheTop(t *testing.T) {
	for _, placement := range []Placement{Right, Left} {
		size, expanded := float32(280), true
		tt := panelTester(&size, &expanded, DraggablePanelProps{Placement: placement})
		tt.Frame()
		if r, _ := tt.Find("panel"); r.Y > 1 {
			t.Errorf("placement %v: content at y %v, want at the top", placement, r.Y)
		}
	}
}

func TestDraggablePanelToggleHidesAfterAClickOnceThePointerLeaves(t *testing.T) {
	tk := theme.Light()
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{})
	tt.Frame()
	tt.Move(600, 150)
	tt.Advance(time.Second)
	collapse, _ := tt.AccessByRole(ui.RoleButton, "Collapse panel")
	tt.ClickAt(collapse.Bounds.X+collapse.Bounds.W/2, collapse.Bounds.Y+collapse.Bounds.H/2)
	tt.Advance(time.Second)
	expand, _ := tt.AccessByRole(ui.RoleButton, "Expand panel")
	tt.ClickAt(expand.Bounds.X+expand.Bounds.W/2, expand.Bounds.Y+expand.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Move(200, 150)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 800, 515, 150); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("toggle %+v after the click and the pointer left: want hidden", got)
	}
}

func TestDraggablePanelToggleShowsOnKeyboardFocus(t *testing.T) {
	tk := theme.Light()
	size, expanded := float32(280), true
	tt := panelTester(&size, &expanded, DraggablePanelProps{})
	tt.Move(200, 150)
	tt.Advance(time.Second)
	for range 4 {
		if tt.Focused("Collapse panel") {
			break
		}
		tt.Key(0, ui.KeyTab)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.Focused("Collapse panel") {
		t.Fatal("Tab never reached the toggle")
	}
	if got := testkit.PixelAt(tt, 800, 515, 150); testkit.Near(got, tk.BgLayout) {
		t.Error("toggle hidden while focused from the keyboard")
	}
}
