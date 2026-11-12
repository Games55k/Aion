package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type floatApp struct {
	open    bool
	p       FloatingPanelProps
	body    float32
	under   int
	w, h    int
	content func(c *ui.Context)
}

func floatTester(a *floatApp) *ui.Tester {
	if a.w == 0 {
		a.w, a.h = 1000, 700
	}
	if a.body == 0 {
		a.body = 200
	}
	if a.p.Title == "" {
		a.p.Title = "Run"
	}
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() {
			if ui.ButtonBase(c).Label("Under").Absolute().Left(0).Top(0).Right(0).Bottom(0).Clicked() {
				a.under++
			}
		})
		FloatingPanel(c, &a.open, a.p, func() {
			if a.content != nil {
				a.content(c)
				return
			}
			ui.Box(c).Height(a.body).Label("body")
		})
	}, a.w, a.h)
}

func floatPanel(t *testing.T, tt *ui.Tester, title string) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleDialog, title)
	if !ok {
		t.Fatalf("no %q panel", title)
	}
	return info.Bounds
}

func TestFloatingPanelAnchorsBottomRight(t *testing.T) {
	a := &floatApp{open: true}
	tt := floatTester(a)
	settle(tt)
	r := floatPanel(t, tt, "Run")
	if r.W != 640 {
		t.Errorf("width %v, want 640", r.W)
	}
	if r.X+r.W != 992 || r.Y+r.H != 692 {
		t.Errorf("panel %+v, want its corner 8 DIPs in from the bottom right", r)
	}
	body, _ := tt.Access("body")
	if body.Bounds.Y < r.Y+48 || body.Bounds.X != r.X || body.Bounds.W != r.W {
		t.Errorf("body %+v in %+v: want under a 48 DIP header, edge to edge", body.Bounds, r)
	}
	if r.H != body.Bounds.Y-r.Y+200 {
		t.Errorf("panel %v tall, want fitting the body", r.H)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Close"); !ok {
		t.Error("no Close button")
	}
}

func TestFloatingPanelPlacementsAndOffset(t *testing.T) {
	cases := []struct {
		at   FloatingPanelPlacement
		x, y func(r ui.Rect) float32
		want [2]float32
	}{
		{FloatingTopLeft, func(r ui.Rect) float32 { return r.X }, func(r ui.Rect) float32 { return r.Y }, [2]float32{20, 20}},
		{FloatingTopRight, func(r ui.Rect) float32 { return r.X + r.W }, func(r ui.Rect) float32 { return r.Y }, [2]float32{980, 20}},
		{FloatingBottomLeft, func(r ui.Rect) float32 { return r.X }, func(r ui.Rect) float32 { return r.Y + r.H }, [2]float32{20, 680}},
	}
	for _, cs := range cases {
		a := &floatApp{open: true, p: FloatingPanelProps{Placement: cs.at, Offset: 20}}
		tt := floatTester(a)
		settle(tt)
		r := floatPanel(t, tt, "Run")
		if got := [2]float32{cs.x(r), cs.y(r)}; got != cs.want {
			t.Errorf("placement %v: corner %v, want %v", cs.at, got, cs.want)
		}
	}
}

func TestFloatingPanelSizes(t *testing.T) {
	a := &floatApp{open: true, w: 500, h: 400, body: 1000}
	tt := floatTester(a)
	settle(tt)
	r := floatPanel(t, tt, "Run")
	if r.W != 468 || r.H != 368 {
		t.Errorf("panel %+v in a 500 × 400 window, want 32 DIPs less each way", r)
	}

	a = &floatApp{open: true, body: 20, p: FloatingPanelProps{Width: 400}}
	tt = floatTester(a)
	settle(tt)
	r = floatPanel(t, tt, "Run")
	if r.W != 400 || r.H != 180 {
		t.Errorf("panel %v × %v, want 400 wide and at least 180 tall", r.W, r.H)
	}

	a = &floatApp{open: true, p: FloatingPanelProps{Height: 300}}
	tt = floatTester(a)
	settle(tt)
	if r = floatPanel(t, tt, "Run"); r.H != 300 {
		t.Errorf("panel %v tall, want Height 300", r.H)
	}
}

func TestFloatingPanelCloses(t *testing.T) {
	for name, dismiss := range map[string]func(tt *ui.Tester){
		"escape":  func(tt *ui.Tester) { tt.Key(0, ui.KeyEscape) },
		"outside": func(tt *ui.Tester) { tt.ClickAt(50, 50) },
		"close": func(tt *ui.Tester) {
			b, _ := tt.AccessByRole(ui.RoleButton, "Close")
			tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
		},
	} {
		a := &floatApp{open: true}
		tt := floatTester(a)
		settle(tt)
		dismiss(tt)
		settle(tt)
		if a.open {
			t.Errorf("%s left it open", name)
		}
		if _, ok := tt.AccessByRole(ui.RoleDialog, "Run"); ok {
			t.Errorf("%s: the panel stays after closing", name)
		}
	}
}

func TestFloatingPanelLeavesThePageUsable(t *testing.T) {
	a := &floatApp{open: true, p: FloatingPanelProps{NoOutsideClose: true}}
	tt := floatTester(a)
	settle(tt)
	tt.ClickAt(50, 50)
	settle(tt)
	if !a.open || a.under != 1 {
		t.Errorf("click outside: open %v, page clicked %d times: want open and the page clicked", a.open, a.under)
	}
	a = &floatApp{open: true, p: FloatingPanelProps{NoKeyboard: true}}
	tt = floatTester(a)
	settle(tt)
	tt.Key(0, ui.KeyEscape)
	settle(tt)
	if !a.open {
		t.Error("NoKeyboard: Escape closed it")
	}
	a = &floatApp{open: true, p: FloatingPanelProps{NoClose: true}}
	tt = floatTester(a)
	settle(tt)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Close"); ok {
		t.Error("NoClose still has a Close button")
	}
	if body, _ := tt.Access("body"); body.Bounds.Y-floatPanel(t, tt, "Run").Y != 48 {
		t.Errorf("body %v under the top, want a 48 DIP header", body.Bounds.Y-floatPanel(t, tt, "Run").Y)
	}
}

func TestFloatingPanelMask(t *testing.T) {
	tk := theme.Light()
	a := &floatApp{open: true, p: FloatingPanelProps{Mask: true}}
	tt := floatTester(a)
	settle(tt)
	if got := testkit.PixelAt(tt, 1000, 50, 50); testkit.Near(got, tk.BgLayout) {
		t.Error("Mask does not dim the page")
	}
	tt.ClickAt(50, 50)
	settle(tt)
	if a.open || a.under != 0 {
		t.Errorf("click on the mask: open %v, page clicked %d: want closed, the page untouched", a.open, a.under)
	}
}

func TestFloatingPanelHeaderActionsAndFooter(t *testing.T) {
	a := &floatApp{open: true}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		FloatingPanel(c, &a.open, FloatingPanelProps{
			Title:   "Run",
			Actions: func() { ui.Box(c).Size(32, 32).Label("act") },
			Footer:  func() { ui.Box(c).Size(80, 32).Label("foot") },
		}, func() { ui.Box(c).Height(200).Label("body") })
	}, 1000, 700)
	settle(tt)
	r := floatPanel(t, tt, "Run")
	act, _ := tt.Access("act")
	cl, _ := tt.AccessByRole(ui.RoleButton, "Close")
	if gap := cl.Bounds.X - (act.Bounds.X + act.Bounds.W); gap != 4 {
		t.Errorf("actions %v before Close, want 4", gap)
	}
	foot, _ := tt.Access("foot")
	if end := foot.Bounds.X + foot.Bounds.W; end != r.X+r.W-16 {
		t.Errorf("footer ends at %v, want 16 in from %v", end, r.X+r.W)
	}
	if bottom := foot.Bounds.Y + foot.Bounds.H; bottom != r.Y+r.H-12 {
		t.Errorf("footer bottom %v, want 12 above %v", bottom, r.Y+r.H)
	}
}

func floatDrag(tt *ui.Tester, x0, y0, x1, y1 float32) {
	tt.Press(x0, y0)
	for i := 1; i <= 10; i++ {
		tt.Move(x0+(x1-x0)*float32(i)/10, y0+(y1-y0)*float32(i)/10)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Release(x1, y1)
	settle(tt)
}

func TestFloatingPanelResizesFromItsFreeEdges(t *testing.T) {
	a := &floatApp{open: true}
	tt := floatTester(a)
	settle(tt)
	r := floatPanel(t, tt, "Run")
	floatDrag(tt, r.X, r.Y+r.H/2, r.X-100, r.Y+r.H/2)
	if got := floatPanel(t, tt, "Run"); got.W != 740 || got.X+got.W != 992 {
		t.Errorf("dragging the left edge 100 left: %+v, want 740 wide, still at the right", got)
	}
	r = floatPanel(t, tt, "Run")
	floatDrag(tt, r.X+r.W/2, r.Y, r.X+r.W/2, r.Y-50)
	if got := floatPanel(t, tt, "Run"); got.H != r.H+50 || got.Y+got.H != 692 {
		t.Errorf("dragging the top edge 50 up: %+v, want %v tall", got, r.H+50)
	}
	r = floatPanel(t, tt, "Run")
	floatDrag(tt, r.X, r.Y, r.X+900, r.Y+900)
	if got := floatPanel(t, tt, "Run"); got.W != 320 || got.H != 180 {
		t.Errorf("dragging the corner past the minimum: %v × %v, want 320 × 180", got.W, got.H)
	}
	r = floatPanel(t, tt, "Run")
	floatDrag(tt, r.X+r.W-2, r.Y+r.H/2, r.X+r.W+98, r.Y+r.H/2)
	if got := floatPanel(t, tt, "Run"); got.W != r.W {
		t.Errorf("the anchored right edge resized it to %v", got.W)
	}
}

func TestFloatingPanelResizeBoundsAndNoResize(t *testing.T) {
	a := &floatApp{open: true, p: FloatingPanelProps{Placement: FloatingTopLeft, MaxWidth: 700, MinWidth: 400}}
	tt := floatTester(a)
	settle(tt)
	r := floatPanel(t, tt, "Run")
	floatDrag(tt, r.X+r.W, r.Y+r.H/2, r.X+r.W+200, r.Y+r.H/2)
	if got := floatPanel(t, tt, "Run"); got.W != 700 || got.X != 8 {
		t.Errorf("top left panel's right edge past MaxWidth: %+v, want 700 wide at x 8", got)
	}
	floatDrag(tt, 708, r.Y+r.H/2, 100, r.Y+r.H/2)
	if got := floatPanel(t, tt, "Run"); got.W != 400 {
		t.Errorf("past MinWidth: %v, want 400", got.W)
	}

	a = &floatApp{open: true, p: FloatingPanelProps{NoResize: true}}
	tt = floatTester(a)
	settle(tt)
	r = floatPanel(t, tt, "Run")
	floatDrag(tt, r.X, r.Y+r.H/2, r.X-100, r.Y+r.H/2)
	if got := floatPanel(t, tt, "Run"); got.W != 640 {
		t.Errorf("NoResize: width %v after a drag", got.W)
	}
}

func TestFloatingPanelEntersFromBelow(t *testing.T) {
	a := &floatApp{}
	tt := floatTester(a)
	tt.Frame()
	a.open = true
	tt.Frame()
	tt.Advance(40 * time.Millisecond)
	tt.Frame()
	early, _ := tt.AccessByRole(ui.RoleDialog, "Run")
	tt.Advance(time.Second)
	tt.Frame()
	late := floatPanel(t, tt, "Run")
	if eb, lb := early.Bounds.Y+early.Bounds.H, late.Y+late.H; eb <= lb {
		t.Errorf("opening with its bottom at %v, settled at %v: want it rising into place", eb, lb)
	}
}

func TestFloatingPanelLinesUnderTheHeaderAndOverTheFooter(t *testing.T) {
	for _, dark := range []bool{false, true} {
		a := &floatApp{open: true}
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			FloatingPanel(c, &a.open, FloatingPanelProps{Title: "Run", Footer: func() { ui.Box(c).Size(80, 32) }},
				func() { ui.Box(c).Height(200).Label("body") })
		}, 1000, 700)
		tt.SetDark(dark)
		settle(tt)
		tk := theme.Light()
		if dark {
			tk = theme.Dark()
		}
		r := floatPanel(t, tt, "Run")
		body, _ := tt.Access("body")
		line := testkit.Over(tk.BgElevated, tk.BorderSecondary)
		if got := testkit.PixelAt(tt, 1000, r.X+r.W/2, body.Bounds.Y-0.5); !testkit.Near(got, line) {
			t.Errorf("dark %v: under the header %+v, want the line %+v", dark, got, line)
		}
		if got := testkit.PixelAt(tt, 1000, r.X+r.W/2, body.Bounds.Y+body.Bounds.H+0.5); !testkit.Near(got, line) {
			t.Errorf("dark %v: over the footer %+v, want the line %+v", dark, got, line)
		}
	}
}
