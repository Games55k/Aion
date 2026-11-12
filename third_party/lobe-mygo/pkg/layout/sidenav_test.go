package layout

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type sideNavState struct {
	width    float32
	expanded bool
	p        DraggableSideNavProps
	chrome   bool
}

func dsnTester(s *sideNavState) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		if s.chrome {
			s.p.Header = func(SideNavState) { ui.Text(c, "Head") }
			s.p.Footer = func(SideNavState) { ui.Text(c, "Foot") }
		}
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			nav := func() {
				DraggableSideNav(c, &s.width, &s.expanded, s.p, func(nav SideNavState) {
					expanded := nav.Expanded
					if expanded {
						ui.Text(c, "Wide body")
					} else {
						ui.Text(c, "Slim")
					}
				}).Label("sidenav")
			}
			main := func() { ui.Box(c).Grow(1) }
			if s.p.Right {
				main()
				nav()
				return
			}
			nav()
			main()
		})
	}, 800, 400)
}

func navWidth(t *testing.T, tt *ui.Tester) float32 {
	t.Helper()
	info, ok := tt.Access("sidenav")
	if !ok {
		t.Fatal("no side nav")
	}
	return info.Bounds.W
}

func TestDraggableSideNavSizes(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := dsnTester(s)
	if w := navWidth(t, tt); w != 280 {
		t.Errorf("expanded width %v, want 280 by default", w)
	}
	if !tt.HasText("Wide body") {
		t.Error("expanded body not built with expanded true")
	}
	s = &sideNavState{width: 300, expanded: false}
	tt = dsnTester(s)
	if w := navWidth(t, tt); w != 64 {
		t.Errorf("collapsed width %v, want the 64 DIP minimum", w)
	}
	if !tt.HasText("Slim") {
		t.Error("collapsed body not built with expanded false")
	}
	s = &sideNavState{width: 300, expanded: false, p: DraggableSideNavProps{MinWidth: 80}}
	tt = dsnTester(s)
	if w := navWidth(t, tt); w != 80 {
		t.Errorf("collapsed width %v, want MinWidth 80", w)
	}
}

func TestDraggableSideNavHeaderFooterAndChrome(t *testing.T) {
	tk := theme.Light()
	s := &sideNavState{expanded: true, chrome: true}
	tt := dsnTester(s)
	head, _ := tt.Find("Head")
	body, _ := tt.Find("Wide body")
	foot, ok := tt.Find("Foot")
	if head.Y != 0 || body.Y < head.Y+head.H {
		t.Errorf("head %+v, body %+v: want the header first", head, body)
	}
	if !ok || foot.Y+foot.H != 400 {
		t.Errorf("footer %+v, want at the bottom", foot)
	}
	if got := testkit.PixelAt(tt, 800, 279.5, 200); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.BorderSecondary)) {
		t.Errorf("inner edge %+v, want the border", got)
	}
	if got := testkit.PixelAt(tt, 800, 200, 300); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("side nav %+v, want the layout background", got)
	}

	tint := ui.RGB(30, 90, 200)
	s = &sideNavState{expanded: true, p: DraggableSideNavProps{NoBorder: true, Background: tint}}
	tt = dsnTester(s)
	if got := testkit.PixelAt(tt, 800, 279.5, 200); !testkit.Near(got, tint) {
		t.Errorf("inner edge with NoBorder %+v, want the background", got)
	}
}

func TestDraggableSideNavHandleCollapsesThenSwapsBodyLate(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := dsnTester(s)
	tt.Move(100, 200)
	tt.Advance(time.Second)
	h, ok := tt.AccessByRole(ui.RoleButton, "Collapse side nav")
	if !ok {
		t.Fatal("no Collapse side nav handle")
	}
	if h.Bounds.W != 16 || h.Bounds.H != 54 || h.Bounds.X < 279 || h.Bounds.X > 280 {
		t.Errorf("handle %+v, want 16 × 54 outside the inner edge", h.Bounds)
	}
	tt.ClickAt(h.Bounds.X+8, h.Bounds.Y+27)
	tt.Advance(100 * time.Millisecond)
	tt.Frame()
	if s.expanded {
		t.Fatal("the handle did not collapse it")
	}
	if !tt.HasText("Wide body") {
		t.Error("the body swapped at once: want the expanded body for the first 200 ms")
	}
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("Slim") || navWidth(t, tt) != 64 {
		t.Errorf("after the animation: texts %q, width %v", tt.Texts(), navWidth(t, tt))
	}
	if s.width != 280 {
		t.Errorf("width %v after collapsing, want 280 kept to expand to", s.width)
	}

	tt.Move(30, 200)
	tt.Advance(time.Second)
	h, ok = tt.AccessByRole(ui.RoleButton, "Expand side nav")
	if !ok {
		t.Fatal("no Expand side nav handle")
	}
	tt.ClickAt(h.Bounds.X+8, h.Bounds.Y+27)
	tt.Frame()
	if !s.expanded || !tt.HasText("Wide body") {
		t.Error("expanding: want the expanded body at once")
	}
	tt.Advance(time.Second)
	tt.Frame()
	if w := navWidth(t, tt); w != 280 {
		t.Errorf("expanded again to %v, want 280", w)
	}
}

func TestDraggableSideNavHandleShowsOnHover(t *testing.T) {
	tk := theme.Light()
	s := &sideNavState{expanded: true}
	tt := dsnTester(s)
	tt.Move(600, 200)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 800, 294.5, 200); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("away from the nav, handle edge %+v: want hidden", got)
	}
	tt.Move(100, 200)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 800, 294.5, 200); testkit.Near(got, tk.BgLayout) {
		t.Error("hovering the nav: want the handle shown")
	}

	s = &sideNavState{expanded: false, p: DraggableSideNavProps{ShowHandleWhenCollapsed: true}}
	tt = dsnTester(s)
	tt.Move(600, 200)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 800, 78.5, 200); testkit.Near(got, tk.BgLayout) {
		t.Error("collapsed with ShowHandleWhenCollapsed: want the handle shown")
	}

	s = &sideNavState{expanded: true, p: DraggableSideNavProps{NoHandle: true}}
	tt = dsnTester(s)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Collapse side nav"); ok {
		t.Error("NoHandle still has a handle")
	}
	s = &sideNavState{expanded: true, p: DraggableSideNavProps{NoExpand: true}}
	tt = dsnTester(s)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Collapse side nav"); ok {
		t.Error("NoExpand still has a handle")
	}
}

func TestDraggableSideNavResizes(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := dsnTester(s)
	drag(tt, 280, 340, 100)
	if !s.expanded || s.width < 339 || s.width > 341 {
		t.Errorf("dragged to 340: width %v, expanded %v", s.width, s.expanded)
	}
	if w := navWidth(t, tt); w < 339 || w > 341 {
		t.Errorf("shown %v wide, want 340", w)
	}

	s = &sideNavState{expanded: true, p: DraggableSideNavProps{MaxWidth: 360}}
	tt = dsnTester(s)
	drag(tt, 280, 500, 100)
	if s.width != 360 {
		t.Errorf("past the max: width %v, want 360", s.width)
	}

	s = &sideNavState{expanded: true, p: DraggableSideNavProps{Right: true}}
	tt = dsnTester(s)
	drag(tt, 520, 480, 100)
	if s.width < 319 || s.width > 321 {
		t.Errorf("right side nav dragged 40 left: width %v, want 320", s.width)
	}
}

func TestDraggableSideNavDragCollapsesAndExpands(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := dsnTester(s)
	drag(tt, 280, 130, 100)
	if s.expanded {
		t.Fatal("dragged to 130 under the 136 threshold: still expanded")
	}
	if s.width != 280 || navWidth(t, tt) != 64 {
		t.Errorf("collapsed: width %v, shown %v, want 280 kept and 64 shown", s.width, navWidth(t, tt))
	}

	s = &sideNavState{expanded: true}
	tt = dsnTester(s)
	drag(tt, 280, 140, 100)
	if !s.expanded || s.width < 139 || s.width > 141 {
		t.Errorf("dragged to 140 over the threshold: width %v, expanded %v", s.width, s.expanded)
	}

	s = &sideNavState{width: 280, expanded: false}
	tt = dsnTester(s)
	drag(tt, 64, 220, 100)
	if !s.expanded || s.width < 219 || s.width > 221 {
		t.Errorf("collapsed dragged to 220: width %v, expanded %v", s.width, s.expanded)
	}

	s = &sideNavState{expanded: true, p: DraggableSideNavProps{NoExpand: true}}
	tt = dsnTester(s)
	drag(tt, 280, 100, 100)
	if !s.expanded || s.width < 99 || s.width > 101 {
		t.Errorf("NoExpand dragged to 100: width %v, expanded %v", s.width, s.expanded)
	}
	drag(tt, 100, 20, 100)
	if s.width != 64 {
		t.Errorf("NoExpand past the minimum: width %v, want 64", s.width)
	}
}

func TestDraggableSideNavNoResize(t *testing.T) {
	s := &sideNavState{expanded: true, p: DraggableSideNavProps{NoResize: true}}
	tt := dsnTester(s)
	if _, ok := tt.AccessByRole(ui.RoleSeparator, "Resize side nav"); ok {
		t.Error("NoResize still has a resize seam")
	}
	drag(tt, 280, 340, 100)
	if s.width != 280 {
		t.Errorf("NoResize dragged: width %v", s.width)
	}
}

func TestDraggableSideNavHandleHidesAfterAClickOnceThePointerLeaves(t *testing.T) {
	tk := theme.Light()
	s := &sideNavState{expanded: true}
	tt := dsnTester(s)
	tt.Move(100, 200)
	tt.Advance(time.Second)
	h, _ := tt.AccessByRole(ui.RoleButton, "Collapse side nav")
	tt.ClickAt(h.Bounds.X+8, h.Bounds.Y+27)
	tt.Advance(time.Second)
	tt.Move(600, 200)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 800, 78.5, 200); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("handle edge %+v after the click and the pointer left: want hidden", got)
	}
}

func stableTester(s *sideNavState) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			DraggableSideNav(c, &s.width, &s.expanded, s.p, func(nav SideNavState) {
				expanded := nav.Expanded
				label := "slim"
				if expanded {
					label = "wide"
				}
				ui.Box(c).Height(20).AlignSelf(ui.Stretch).Label(label)
			}).Label("sidenav")
			ui.Box(c).Grow(1)
		})
	}, 800, 400)
}

func bodyWidth(t *testing.T, tt *ui.Tester, label string) float32 {
	t.Helper()
	info, ok := tt.Access(label)
	if !ok {
		t.Fatalf("no %s body", label)
	}
	return info.Bounds.W
}

func TestDraggableSideNavKeepsItsContentWidthWhileAnimating(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := stableTester(s)
	if w := bodyWidth(t, tt, "wide"); w != 279 {
		t.Fatalf("expanded body %v wide, want 279 inside the border", w)
	}
	s.expanded = false
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Frame()
	if nav := navWidth(t, tt); nav >= 279 || nav <= 64 {
		t.Fatalf("nav %v wide 100 ms into collapsing, want on its way", nav)
	}
	if w := bodyWidth(t, tt, "wide"); w != 279 {
		t.Errorf("collapsing: body %v wide, want 279 kept and clipped", w)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if w := bodyWidth(t, tt, "slim"); w != 63 {
		t.Errorf("collapsed body %v wide, want 63", w)
	}

	s.expanded = true
	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	tt.Frame()
	if w := bodyWidth(t, tt, "wide"); w != 279 {
		t.Errorf("expanding: body %v wide, want 279 at once", w)
	}
}

func TestDraggableSideNavContentFollowsADrag(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := stableTester(s)
	tt.Press(280, 100)
	for i := 1; i <= 5; i++ {
		tt.Move(280+float32(i)*10, 100)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Frame()
	if nav, w := navWidth(t, tt), bodyWidth(t, tt, "wide"); w != nav-1 {
		t.Errorf("dragging: body %v wide in a %v nav, want it filling", w, nav)
	}
	tt.Release(330, 100)
}

func TestDraggableSideNavRightKeepsContentAtTheOuterEdge(t *testing.T) {
	s := &sideNavState{expanded: true, p: DraggableSideNavProps{Right: true}}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1)
			DraggableSideNav(c, &s.width, &s.expanded, s.p, func(SideNavState) {
				ui.Box(c).Height(20).AlignSelf(ui.Stretch).Label("body")
			})
		})
	}, 800, 400)
	s.expanded = false
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Frame()
	b, _ := tt.Access("body")
	if end := b.Bounds.X + b.Bounds.W; end != 800 {
		t.Errorf("right nav collapsing: body ends at %v, want held at the window's edge", end)
	}
}

func TestDraggableSideNavClipsItsContentWhileAnimating(t *testing.T) {
	ink := ui.RGB(220, 30, 30)
	for _, right := range []bool{false, true} {
		s := &sideNavState{expanded: true, p: DraggableSideNavProps{Right: right}}
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
				nav := func() {
					DraggableSideNav(c, &s.width, &s.expanded, s.p, func(SideNavState) {
						ui.Box(c).Height(40).AlignSelf(ui.Stretch).Background(ink)
					}).Label("sidenav")
				}
				if right {
					ui.Box(c).Grow(1)
					nav()
				} else {
					nav()
					ui.Box(c).Grow(1)
				}
			})
		}, 800, 400)
		s.expanded = false
		tt.Frame()
		tt.Advance(100 * time.Millisecond)
		tt.Frame()
		nav, _ := tt.Access("sidenav")
		x := nav.Bounds.X + nav.Bounds.W + 20
		if right {
			x = nav.Bounds.X - 20
		}
		if got := testkit.PixelAt(tt, 800, x, 20); testkit.Near(got, ink) {
			t.Errorf("right %v: content shows at x %v outside the %+v nav", right, x, nav.Bounds)
		}
		if got := testkit.PixelAt(tt, 800, nav.Bounds.X+nav.Bounds.W/2, 20); !testkit.Near(got, ink) {
			t.Errorf("right %v: content %+v missing inside the nav", right, got)
		}
	}
}

func collapseTester(s *sideNavState, seen *SideNavState) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			DraggableSideNav(c, &s.width, &s.expanded, s.p, func(nav SideNavState) { *seen = nav }).Label("sidenav")
			ui.Box(c).Grow(1)
		})
	}, 800, 400)
}

func TestDraggableSideNavReportsCollapseWithItsWidth(t *testing.T) {
	var seen SideNavState
	s := &sideNavState{expanded: true}
	tt := collapseTester(s, &seen)
	if seen.Collapse != 0 || seen.RailWidth != 63 {
		t.Errorf("expanded: %+v, want Collapse 0 and a 63 DIP rail", seen)
	}
	s.expanded = false
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	tt.Frame()
	nav := navWidth(t, tt)
	if want := 1 - (nav-64)/(280-64); abs(seen.Collapse-want) > 0.05 {
		t.Errorf("collapsing at width %v: Collapse %v, want %v", nav, seen.Collapse, want)
	}
	tt.Advance(time.Second)
	if seen.Collapse != 1 || seen.Expanded {
		t.Errorf("collapsed: %+v, want Collapse 1", seen)
	}
}

func TestDraggableSideNavCollapseFollowsADrag(t *testing.T) {
	var seen SideNavState
	s := &sideNavState{expanded: true}
	tt := collapseTester(s, &seen)
	tt.Press(280, 100)
	for i := 1; i <= 10; i++ {
		tt.Move(280-float32(i)*10.8, 100)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Frame()
	if abs(seen.Collapse-0.5) > 0.05 {
		t.Errorf("dragged halfway to the rail: Collapse %v, want 0.5", seen.Collapse)
	}
	tt.Release(172, 100)
	tt.Advance(time.Second)
	if seen.Collapse != 0 {
		t.Errorf("released over the threshold: Collapse %v, want back at 0", seen.Collapse)
	}
}

func TestDraggableSideNavRailRowsShrinkWithoutAJump(t *testing.T) {
	s := &sideNavState{expanded: true}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			DraggableSideNav(c, &s.width, &s.expanded, s.p, func(nav SideNavState) {
				display.List(c, display.ListProps{Active: "home", Collapse: nav.Collapse, RailWidth: nav.RailWidth,
					Items: []display.ListItem{{Key: "home", Label: "Home", Icon: lucide.House()}}})
			}).Label("sidenav")
			ui.Box(c).Grow(1)
		})
	}, 800, 400)
	s.expanded = false
	tt.Frame()
	prevRow, prevNav := float32(0), float32(0)
	for i := range 30 {
		tt.Advance(16 * time.Millisecond)
		tt.Frame()
		row, _ := tt.AccessByRole(ui.RoleButton, "Home")
		nav := navWidth(t, tt)
		if row.Bounds.X+row.Bounds.W > nav {
			t.Errorf("frame %d: row ends at %v past the %v nav", i, row.Bounds.X+row.Bounds.W, nav)
		}
		if i > 0 && prevRow-row.Bounds.W > (prevNav-nav)*1.2+0.5 {
			t.Errorf("frame %d: row narrowed %v while the nav narrowed %v", i, prevRow-row.Bounds.W, prevNav-nav)
		}
		prevRow, prevNav = row.Bounds.W, nav
	}
	row, _ := tt.AccessByRole(ui.RoleButton, "Home")
	if row.Bounds.W != row.Bounds.H || abs(row.Bounds.X+row.Bounds.W/2-31.5) > 0.5 {
		t.Errorf("collapsed row %+v, want a square centered in the 63 DIP rail", row.Bounds)
	}
}
