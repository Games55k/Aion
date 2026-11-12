package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type sheetApp struct {
	open    bool
	p       FloatingSheetProps
	snapped []float32
}

func sheetTester(a *sheetApp) *ui.Tester {
	if a.p.Title == "" {
		a.p.Title = "Notes"
	}
	a.p.OnSnapPointChange = func(v float32) { a.snapped = append(a.snapped, v) }
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Size(600, 500).Children(func() {
			page := ui.Box(c).Label("page")
			if a.p.Inline {
				page.Grow(1)
			} else {
				page.Fill()
			}
			FloatingSheet(c, &a.open, a.p, func() { ui.Text(c, "Sheet body") }).Label("sheet")
		})
	}, 700, 600)
}

func sheetBounds(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.Access("sheet")
	if !ok {
		t.Fatal("no sheet")
	}
	return info.Bounds
}

func sheetDrag(tt *ui.Tester, x, y0, y1 float32, steps int, step time.Duration) {
	tt.Press(x, y0)
	for i := 1; i <= steps; i++ {
		tt.Move(x, y0+(y1-y0)*float32(i)/float32(steps))
		tt.Advance(step)
	}
	tt.Release(x, y1)
	settle(tt)
}

func TestFloatingSheetRestsAtTheBottom(t *testing.T) {
	a := &sheetApp{open: true}
	tt := sheetTester(a)
	settle(tt)
	r := sheetBounds(t, tt)
	if r.X != 0 || r.W != 600 || r.H != 200 || r.Y+r.H != 500 {
		t.Errorf("sheet %+v, want 600 × 200 along the container's bottom", r)
	}
	if !tt.HasText("Notes") || !tt.HasText("Sheet body") {
		t.Errorf("texts %q, want the title and the body", tt.Texts())
	}
	tk := theme.Light()
	if got := testkit.PixelAt(tt, 700, r.X+1, r.Y+1); testkit.Near(got, tk.BgContainer) {
		t.Error("square top corner: want rounded")
	}
	if got := testkit.PixelAt(tt, 700, r.X+1, r.Y+r.H-1); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("bottom corner %+v, want square", got)
	}
	if got := testkit.PixelAt(tt, 700, 300, r.Y+10); !testkit.Near(got, tk.BorderSecondary) {
		t.Errorf("handle %+v, want a bar at the top middle", got)
	}
}

func TestFloatingSheetHeights(t *testing.T) {
	for _, cs := range []struct {
		p    FloatingSheetProps
		want float32
	}{
		{FloatingSheetProps{RestingHeight: 0.5}, 250},
		{FloatingSheetProps{RestingHeight: 0.95}, 400},
		{FloatingSheetProps{RestingHeight: 100}, 200},
		{FloatingSheetProps{MinHeight: 120}, 120},
		{FloatingSheetProps{RestingHeight: 0.95, MaxHeight: 300}, 300},
	} {
		a := &sheetApp{open: true, p: cs.p}
		tt := sheetTester(a)
		settle(tt)
		if r := sheetBounds(t, tt); r.H != cs.want {
			t.Errorf("%+v: height %v, want %v", cs.p, r.H, cs.want)
		}
	}
}

func TestFloatingSheetOpensAndClosesByHeight(t *testing.T) {
	a := &sheetApp{}
	tt := sheetTester(a)
	settle(tt)
	if tt.HasText("Sheet body") {
		t.Error("closed sheet shows its body")
	}
	a.open = true
	tt.Frame()
	tt.Advance(60 * time.Millisecond)
	tt.Frame()
	if r := sheetBounds(t, tt); r.H <= 0 || r.H >= 200 {
		t.Errorf("opening: height %v, want growing toward 200", r.H)
	}
	settle(tt)
	a.open = false
	settle(tt)
	if tt.HasText("Sheet body") {
		t.Error("closed sheet still shows its body")
	}
}

func TestFloatingSheetDragResizesAndDismisses(t *testing.T) {
	a := &sheetApp{open: true}
	tt := sheetTester(a)
	settle(tt)
	sheetDrag(tt, 300, 312, 212, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 300 {
		t.Errorf("dragged up 100: height %v, want 300", r.H)
	}
	sheetDrag(tt, 300, 212, 0, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 400 {
		t.Errorf("dragged past the max: height %v, want 400", r.H)
	}
	sheetDrag(tt, 300, 112, 330, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 200 || !a.open {
		t.Errorf("dragged under the min: height %v, open %v, want back at 200", r.H, a.open)
	}
	sheetDrag(tt, 300, 312, 480, 10, 30*time.Millisecond)
	if a.open {
		t.Error("dragged under a quarter of the min: still open")
	}

	a = &sheetApp{open: true, p: FloatingSheetProps{NoDismiss: true}}
	tt = sheetTester(a)
	settle(tt)
	sheetDrag(tt, 300, 312, 480, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); !a.open || r.H != 200 {
		t.Errorf("NoDismiss dragged down: open %v, height %v", a.open, r.H)
	}
}

func TestFloatingSheetSnapPoints(t *testing.T) {
	a := &sheetApp{open: true, p: FloatingSheetProps{SnapPoints: []float32{0.4, 0.6, 0.9}}}
	tt := sheetTester(a)
	settle(tt)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Fatalf("resting %v, want the lowest snap point 200", r.H)
	}
	sheetDrag(tt, 300, 312, 222, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 300 {
		t.Errorf("dragged up 90 slowly: height %v, want the closest snap point 300", r.H)
	}
	if len(a.snapped) != 1 || a.snapped[0] != 0.6 {
		t.Errorf("OnSnapPointChange got %v, want [0.6]", a.snapped)
	}
	sheetDrag(tt, 300, 212, 182, 2, 16*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 400 {
		t.Errorf("flung up 30: height %v, want the next snap point 400", r.H)
	}
	sheetDrag(tt, 300, 112, 142, 2, 16*time.Millisecond)
	sheetDrag(tt, 300, 212, 242, 2, 16*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Errorf("flung down twice: height %v, want 200", r.H)
	}
	sheetDrag(tt, 300, 312, 342, 2, 16*time.Millisecond)
	if a.open {
		t.Error("flung down from the lowest snap point: still open")
	}
}

func TestFloatingSheetActiveSnapPoint(t *testing.T) {
	a := &sheetApp{open: true, p: FloatingSheetProps{SnapPoints: []float32{0.4, 0.6}, ActiveSnapPoint: 0.6}}
	tt := sheetTester(a)
	settle(tt)
	if r := sheetBounds(t, tt); r.H != 300 {
		t.Errorf("ActiveSnapPoint 0.6: height %v, want 300", r.H)
	}
	a.p.ActiveSnapPoint = 0.4
	settle(tt)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Errorf("ActiveSnapPoint 0.4: height %v, want 200", r.H)
	}
}

func TestFloatingSheetHeaderActionsDoNotDrag(t *testing.T) {
	clicks := 0
	a := &sheetApp{open: true}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Size(600, 500).Children(func() {
			ui.Box(c).Fill()
			FloatingSheet(c, &a.open, FloatingSheetProps{Title: "Notes", HeaderActions: func() {
				ui.Box(c).Size(24, 24).Label("Tag")
				if ui.ButtonBase(c).Label("Pin").Size(24, 24).Clicked() {
					clicks++
				}
			}}, func() {}).Label("sheet")
		})
	}, 700, 600)
	settle(tt)
	pin, _ := tt.AccessByRole(ui.RoleButton, "Pin")
	x, y := pin.Bounds.X+12, pin.Bounds.Y+12
	if pin.Bounds.X+pin.Bounds.W != 584 {
		t.Errorf("Pin ends at %v, want 16 in from the right", pin.Bounds.X+pin.Bounds.W)
	}
	sheetDrag(tt, x, y, y-100, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Errorf("dragging from an action resized the sheet to %v", r.H)
	}
	tag, _ := tt.Access("Tag")
	sheetDrag(tt, tag.Bounds.X+12, tag.Bounds.Y+12, tag.Bounds.Y-88, 10, 30*time.Millisecond)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Errorf("dragging from a plain action resized the sheet to %v", r.H)
	}
}

func TestFloatingSheetInlineOverlapsUpward(t *testing.T) {
	a := &sheetApp{open: true, p: FloatingSheetProps{Inline: true}}
	tt := sheetTester(a)
	settle(tt)
	page, _ := tt.Access("page")
	if page.Bounds.H != 300 {
		t.Errorf("inline: page %v tall, want the room the sheet leaves", page.Bounds.H)
	}
	sheetDrag(tt, 300, 312, 212, 10, 30*time.Millisecond)
	r := sheetBounds(t, tt)
	page, _ = tt.Access("page")
	if r.H != 300 || r.Y+r.H != 500 || page.Bounds.H != 300 {
		t.Errorf("inline dragged up: sheet %+v, page %v tall, want the sheet over the page", r, page.Bounds.H)
	}
}

func TestFloatingSheetVariants(t *testing.T) {
	tk := theme.Light()
	a := &sheetApp{open: true, p: FloatingSheetProps{Embedded: true}}
	tt := sheetTester(a)
	settle(tt)
	r := sheetBounds(t, tt)
	if got := testkit.PixelAt(tt, 700, 300, r.Y+0.5); !testkit.Near(got, testkit.Over(tk.BgContainer, tk.BorderSecondary)) {
		t.Errorf("embedded top edge %+v, want the border", got)
	}
	if got := testkit.PixelAt(tt, 700, 300, r.Y-0.5); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("above the embedded sheet %+v, want no shadow", got)
	}
	a = &sheetApp{open: true}
	tt = sheetTester(a)
	settle(tt)
	if got := testkit.PixelAt(tt, 700, 300, r.Y-0.5); testkit.Near(got, tk.BgLayout) {
		t.Error("above the elevated sheet: want its shadow")
	}
}

func TestFloatingSheetRestsAtTheLowestSnapPoint(t *testing.T) {
	a := &sheetApp{open: true, p: FloatingSheetProps{SnapPoints: []float32{0.7, 0.5}}}
	tt := sheetTester(a)
	settle(tt)
	if r := sheetBounds(t, tt); r.H != 250 {
		t.Errorf("resting %v, want the lowest snap point 250", r.H)
	}
}

func TestFloatingSheetDragStopsAtTheMax(t *testing.T) {
	a := &sheetApp{open: true}
	tt := sheetTester(a)
	settle(tt)
	tt.Press(300, 312)
	for i := 1; i <= 10; i++ {
		tt.Move(300, 312-30*float32(i))
		tt.Advance(16 * time.Millisecond)
	}
	tt.Frame()
	if r := sheetBounds(t, tt); r.H != 400 {
		t.Errorf("held 300 up: height %v, want stopped at 400", r.H)
	}
	tt.Release(300, 12)
}

func TestFloatingSheetShowsWhenOpenFromTheStart(t *testing.T) {
	a := &sheetApp{open: true}
	tt := sheetTester(a)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Errorf("height %v without any input, want 200", r.H)
	}
}

func TestFloatingSheetSettlesAfterReleaseWithoutMoreInput(t *testing.T) {
	a := &sheetApp{open: true}
	tt := sheetTester(a)
	tt.Press(300, 312)
	for _, y := range []float32{332, 352, 372, 392} {
		tt.Move(300, y)
		tt.Advance(50 * time.Millisecond)
	}
	tt.Release(300, 392)
	tt.Advance(time.Second)
	if r := sheetBounds(t, tt); r.H != 200 {
		t.Errorf("released under the min: height %v a second later, want back at 200", r.H)
	}
}
