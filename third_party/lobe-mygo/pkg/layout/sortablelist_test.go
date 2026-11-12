package layout

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type sortApp struct {
	items []string
	p     SortableListProps
}

func sortTester(a *sortApp) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(300).Children(func() {
			SortableList(c, &a.items, func(s string) string { return s }, a.p, func(s string, handle func()) {
				handle()
				ui.Text(c, "Item "+s).Grow(1)
			}).Label("list")
		})
	}, 400, 400)
}

func handles(tt *ui.Tester) []ui.AccessInfo {
	var out []ui.AccessInfo
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleButton && n.Name == "Reorder" {
			out = append(out, n)
		}
	}
	return out
}

func rowOf(t *testing.T, tt *ui.Tester, s string) ui.Rect {
	t.Helper()
	r, ok := tt.Find("Item " + s)
	if !ok {
		t.Fatalf("no Item %s", s)
	}
	return r
}

func TestSortableListRendersRowsWithHandles(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	if len(hs) != 4 {
		t.Fatalf("%d handles, want one per row", len(hs))
	}
	if gap := hs[1].Bounds.Y - (hs[0].Bounds.Y + hs[0].Bounds.H); gap != 8 {
		t.Errorf("rows %v apart, want 8", gap)
	}
	a2, b := rowOf(t, tt, "a"), rowOf(t, tt, "b")
	if a2.Y >= b.Y || a2.X != hs[0].Bounds.X+hs[0].Bounds.W+4 {
		t.Errorf("Item a %+v after handle %+v: want 4 DIPs after it", a2, hs[0].Bounds)
	}
	a.p.Gap = 2
	tt.Frame()
	hs = handles(tt)
	if gap := hs[1].Bounds.Y - (hs[0].Bounds.Y + hs[0].Bounds.H); gap != 2 {
		t.Errorf("Gap 2: rows %v apart", gap)
	}
}

func dragHandle(tt *ui.Tester, h ui.AccessInfo, dy float32, release bool) {
	x, y := h.Bounds.X+h.Bounds.W/2, h.Bounds.Y+h.Bounds.H/2
	tt.Press(x, y)
	for i := 1; i <= 10; i++ {
		tt.Move(x, y+dy*float32(i)/10)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if release {
		tt.Release(x, y+dy)
		tt.Advance(time.Second)
		tt.Frame()
	}
}

func TestSortableListDragsARowDown(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	pitch := hs[1].Bounds.Y - hs[0].Bounds.Y
	dragHandle(tt, hs[0], pitch*2, true)
	if want := []string{"b", "c", "a", "d"}; !slices.Equal(a.items, want) {
		t.Errorf("items %v, want %v", a.items, want)
	}
	if r := rowOf(t, tt, "a"); r.Y < rowOf(t, tt, "c").Y {
		t.Error("Item a still shows above c")
	}
}

func TestSortableListDragsARowUp(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	pitch := hs[1].Bounds.Y - hs[0].Bounds.Y
	dragHandle(tt, hs[3], -pitch*3, true)
	if want := []string{"d", "a", "b", "c"}; !slices.Equal(a.items, want) {
		t.Errorf("items %v, want %v", a.items, want)
	}
}

func TestSortableListShortDragKeepsTheOrder(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	dragHandle(tt, hs[1], 10, true)
	if want := []string{"a", "b", "c", "d"}; !slices.Equal(a.items, want) {
		t.Errorf("items %v, want unchanged", a.items)
	}
}

func TestSortableListMakesRoomWhileDragging(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	pitch := hs[1].Bounds.Y - hs[0].Bounds.Y
	bBefore := rowOf(t, tt, "b")
	dragHandle(tt, hs[0], pitch*2, false)
	if !slices.Equal(a.items, []string{"a", "b", "c", "d"}) {
		t.Errorf("items %v changed before the drop", a.items)
	}
	list, _ := tt.Access("list")
	var bNow ui.Rect
	for _, n := range tt.AccessAll() {
		if n.Name == "Item b" && n.Bounds.Y >= list.Bounds.Y {
			bNow = n.Bounds
		}
	}
	if bNow == (ui.Rect{}) {
		bNow = rowOf(t, tt, "b")
	}
	if d := bBefore.Y - bNow.Y; d < pitch-1 || d > pitch+1 {
		t.Errorf("Item b moved up %v while a is dragged past it, want %v", d, pitch)
	}
	count := 0
	for _, n := range tt.AccessAll() {
		if n.Name == "Item a" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("Item a shows %d times while dragged, want the row and the copy under the pointer", count)
	}
}

func TestSortableListKeyboard(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	tt.ClickAt(hs[1].Bounds.X+hs[1].Bounds.W/2, hs[1].Bounds.Y+hs[1].Bounds.H/2)
	tt.Frame()
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeySpace)
	tt.Advance(time.Second)
	tt.Frame()
	if want := []string{"a", "c", "d", "b"}; !slices.Equal(a.items, want) {
		t.Errorf("Space, Down, Down, Space: items %v, want %v", a.items, want)
	}
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	tt.Frame()
	if want := []string{"a", "c", "d", "b"}; !slices.Equal(a.items, want) {
		t.Errorf("Escape: items %v, want %v", a.items, want)
	}
}

func TestSortableListVariants(t *testing.T) {
	tk := theme.Light()
	a := &sortApp{items: []string{"a", "b"}, p: SortableListProps{Variant: SortableOutlined}}
	tt := sortTester(a)
	hs := handles(tt)
	list, _ := tt.Access("list")
	if hs[0].Bounds.X != list.Bounds.X+5 {
		t.Errorf("outlined: handle at %v, want the border and 4 in from %v", hs[0].Bounds.X, list.Bounds.X)
	}
	if got := testkit.PixelAt(tt, 400, list.Bounds.X+0.5, hs[0].Bounds.Y+hs[0].Bounds.H/2); !testkit.Near(got, testkit.Over(tk.BgContainer, tk.BorderSecondary)) {
		t.Errorf("outlined edge %+v, want the border", got)
	}
	a = &sortApp{items: []string{"a", "b"}, p: SortableListProps{Variant: SortableFilled}}
	tt = sortTester(a)
	hs = handles(tt)
	list, _ = tt.Access("list")
	if got := testkit.PixelAt(tt, 400, list.Bounds.X+list.Bounds.W-5, hs[0].Bounds.Y+hs[0].Bounds.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("filled row %+v, want the fill", got)
	}
	a = &sortApp{items: []string{"a", "b"}}
	tt = sortTester(a)
	hs = handles(tt)
	list, _ = tt.Access("list")
	if hs[0].Bounds.X != list.Bounds.X {
		t.Errorf("borderless: handle at %v, want flush with %v", hs[0].Bounds.X, list.Bounds.X)
	}
}

func allNamed(tt *ui.Tester, name string) []ui.Rect {
	var out []ui.Rect
	for _, n := range tt.AccessAll() {
		if n.Name == name {
			out = append(out, n.Bounds)
		}
	}
	return out
}

func TestSortableListMakesRoomWhileDraggingUp(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	pitch := hs[1].Bounds.Y - hs[0].Bounds.Y
	cBefore := rowOf(t, tt, "c")
	dragHandle(tt, hs[3], -pitch*2, false)
	if c := rowOf(t, tt, "c"); c.Y-cBefore.Y < pitch-1 || c.Y-cBefore.Y > pitch+1 {
		t.Errorf("Item c moved down %v while d is dragged above it, want %v", c.Y-cBefore.Y, pitch)
	}
}

func TestSortableListCopyFollowsThePointerOverAFadedRow(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	pitch := hs[1].Bounds.Y - hs[0].Bounds.Y
	start := rowOf(t, tt, "a")
	dragHandle(tt, hs[0], pitch*1.45, false)
	var slot, copyAt ui.Rect
	for _, r := range allNamed(tt, "Item a") {
		switch {
		case abs(r.Y-(start.Y+pitch)) < 1:
			slot = r
		case abs(r.Y-(start.Y+pitch*1.45)) < 1:
			copyAt = r
		}
	}
	if slot == (ui.Rect{}) || copyAt == (ui.Rect{}) {
		t.Fatalf("Item a at %v, want a row in the next slot and a copy under the pointer", allNamed(tt, "Item a"))
	}
	bg := theme.Light().BgLayout
	strip := func(r ui.Rect) ui.Rect { return ui.Rect{X: r.X, Y: r.Y, W: r.W, H: pitch * 0.4} }
	faded, solid := testkit.Ink(tt, 400, strip(slot), bg), testkit.Ink(tt, 400, strip(rowOf(t, tt, "c")), bg)
	if faded.R <= solid.R {
		t.Errorf("dragged row's ink %+v, others' %+v: want it faded", faded, solid)
	}
}

func TestSortableListKeyboardMovesUp(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	tt.ClickAt(hs[2].Bounds.X+hs[2].Bounds.W/2, hs[2].Bounds.Y+hs[2].Bounds.H/2)
	tt.Frame()
	tt.Key(0, ui.KeySpace)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeySpace)
	tt.Advance(time.Second)
	tt.Frame()
	if want := []string{"c", "a", "b", "d"}; !slices.Equal(a.items, want) {
		t.Errorf("Space, Up ×3, Space: items %v, want %v", a.items, want)
	}
}

func TestSortableListDropsWithoutMovingRowsTwice(t *testing.T) {
	a := &sortApp{items: []string{"a", "b", "c", "d"}}
	tt := sortTester(a)
	hs := handles(tt)
	pitch := hs[1].Bounds.Y - hs[0].Bounds.Y
	top := rowOf(t, tt, "a").Y
	dragHandle(tt, hs[0], pitch*2, false)
	x, y := hs[0].Bounds.X+hs[0].Bounds.W/2, hs[0].Bounds.Y+hs[0].Bounds.H/2+pitch*2
	tt.Release(x, y)
	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	tt.Frame()
	for i, s := range []string{"b", "c"} {
		if r := rowOf(t, tt, s); abs(r.Y-(top+pitch*float32(i))) > 1 {
			t.Errorf("Item %s at %v just after the drop, want %v", s, r.Y, top+pitch*float32(i))
		}
	}
}
