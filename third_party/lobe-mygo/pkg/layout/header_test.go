package layout

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var headerUnder = ui.RGB(200, 40, 40)

func headerApp(mobile bool) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Background(headerUnder).Children(func() {
			Header(c, HeaderProps{
				Mobile:  mobile,
				Logo:    func() { ui.Text(c, "Logo") },
				Nav:     func() { ui.Text(c, "Nav") },
				Actions: func() { ui.Text(c, "Act1"); ui.Text(c, "Act2") },
			}).Label("header")
		})
	}, 800, 300)
}

func TestHeaderDesktopLayout(t *testing.T) {
	tt := headerApp(false)
	h, _ := tt.Access("header")
	if h.Bounds.H != 64 || h.Bounds.W != 800 {
		t.Errorf("header %+v, want 800 × 64", h.Bounds)
	}
	logo, _ := tt.Find("Logo")
	nav, _ := tt.Find("Nav")
	a1, _ := tt.Find("Act1")
	a2, _ := tt.Find("Act2")
	if logo.X != 24 {
		t.Errorf("logo at x %v, want 24", logo.X)
	}
	if nav.X != logo.X+logo.W+48 {
		t.Errorf("nav at x %v, want 48 after the logo's end %v", nav.X, logo.X+logo.W)
	}
	if end := a2.X + a2.W; end < 775 || end > 776 {
		t.Errorf("actions end at %v, want 24 in from the right", end)
	}
	if gap := a2.X - (a1.X + a1.W); gap != 8 {
		t.Errorf("actions %v apart, want 8", gap)
	}
	if mid := logo.Y + logo.H/2; mid < 31 || mid > 33 {
		t.Errorf("logo centered at y %v, want the middle", mid)
	}
}

func TestHeaderMobileLayout(t *testing.T) {
	tt := headerApp(true)
	nav, _ := tt.Find("Nav")
	logo, _ := tt.Find("Logo")
	a2, _ := tt.Find("Act2")
	if nav.X != 12 {
		t.Errorf("nav at x %v, want first, 12 in", nav.X)
	}
	if logo.X < nav.X+nav.W || logo.X > nav.X+nav.W+1 {
		t.Errorf("logo at x %v, want right after the nav", logo.X)
	}
	if end := a2.X + a2.W; end < 787 || end > 788 {
		t.Errorf("actions end at %v, want 12 in from the right", end)
	}
}

func TestHeaderBottomBorderAndBackground(t *testing.T) {
	tk := theme.Light()
	tt := headerApp(false)
	if got := testkit.PixelAt(tt, 800, 400, 63.5); !testkit.Near(got, testkit.Over(testkit.Over(headerUnder, tk.BgLayout.Alpha(0.4)), tk.BorderSecondary)) {
		t.Errorf("bottom edge %+v, want the border", got)
	}
	want := testkit.Over(headerUnder, tk.BgLayout.Alpha(0.4))
	if got := testkit.PixelAt(tt, 800, 400, 20); !testkit.Near(got, want) {
		t.Errorf("header %+v, want the layout background", got)
	}
}

func sideNavApp(h int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().Children(func() {
			SideNav(c, SideNavProps{
				Avatar:        func() { ui.Box(c).Size(40, 40).Label("avatar") },
				TopActions:    func() { ui.Box(c).Size(36, 36).Label("top1"); ui.Box(c).Size(36, 36).Label("top2") },
				BottomActions: func() { ui.Box(c).Size(36, 36).Label("bottom1"); ui.Box(c).Size(36, 36).Label("bottom2") },
			}).Label("sidenav")
		})
	}, 400, h)
}

func TestSideNavLayout(t *testing.T) {
	tt := sideNavApp(700)
	nav, _ := tt.Access("sidenav")
	if nav.Bounds.W != 58 || nav.Bounds.H != 700 {
		t.Errorf("side nav %+v, want 58 wide and the full height", nav.Bounds)
	}
	av, _ := tt.Access("avatar")
	t1, _ := tt.Access("top1")
	t2, _ := tt.Access("top2")
	b1, _ := tt.Access("bottom1")
	b2, _ := tt.Access("bottom2")
	if av.Bounds.Y != 12 || av.Bounds.X != 8.5 {
		t.Errorf("avatar %+v, want centered 12 from the top", av.Bounds)
	}
	if gap := t1.Bounds.Y - (av.Bounds.Y + av.Bounds.H); gap != 12 {
		t.Errorf("top actions %v under the avatar, want 12", gap)
	}
	if gap := t2.Bounds.Y - (t1.Bounds.Y + t1.Bounds.H); gap != 8 {
		t.Errorf("top actions %v apart, want 8", gap)
	}
	if gap := b2.Bounds.Y - (b1.Bounds.Y + b1.Bounds.H); gap != 4 {
		t.Errorf("bottom actions %v apart, want 4", gap)
	}
	if end := b2.Bounds.Y + b2.Bounds.H; end != 688 {
		t.Errorf("bottom actions end at %v, want 12 from the bottom", end)
	}
}

func TestSideNavMinHeightAndBorder(t *testing.T) {
	tk := theme.Light()
	tt := sideNavApp(500)
	nav, _ := tt.Access("sidenav")
	if nav.Bounds.H != 640 {
		t.Errorf("height %v in a 500 window, want at least 640", nav.Bounds.H)
	}
	if got := testkit.PixelAt(tt, 400, 57.5, 300); !testkit.Near(got, testkit.Over(tk.BgContainer, tk.BorderSecondary)) {
		t.Errorf("right edge %+v, want the border", got)
	}
	if got := testkit.PixelAt(tt, 400, 30, 300); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("side nav %+v, want the container background", got)
	}
}
