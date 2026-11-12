package layout

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func gridColumns(t *testing.T, width int, p GridProps) int {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		Grid(c, p).Children(func() {
			for i := range 6 {
				ui.Text(c.Key(i), fmt.Sprintf("item%d", i)).Height(40)
			}
		})
	}, width, 400)
	tt.Frame()
	first, _ := tt.Find("item0")
	n := 0
	for i := range 6 {
		r, ok := tt.Find(fmt.Sprintf("item%d", i))
		if !ok {
			t.Fatalf("no item%d", i)
		}
		if r.Y == first.Y {
			n++
		}
	}
	return n
}

func TestGridFitsColumnsOfAtLeastMaxItemWidthUpToMaxColumns(t *testing.T) {
	for _, c := range []struct {
		width int
		p     GridProps
		want  int
	}{
		{800, GridProps{}, 3},
		{520, GridProps{}, 2},
		{400, GridProps{}, 1},
		{1600, GridProps{}, 3},
		{1600, GridProps{MaxColumns: 6}, 6},
		{800, GridProps{MaxItemWidth: 120, MaxColumns: 6}, 6},
	} {
		if got := gridColumns(t, c.width, c.p); got != c.want {
			t.Errorf("width %d %+v: %d columns, want %d", c.width, c.p, got, c.want)
		}
	}
}

func TestGridGap(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		Grid(c, GridProps{Gap: 20}).Children(func() {
			ui.Text(c, "a").Height(40)
			ui.Text(c, "b").Height(40)
		})
	}, 800, 200)
	tt.Frame()
	a, _ := tt.Find("a")
	b, _ := tt.Find("b")
	if gap := b.X - (a.X + a.W); gap < 19.5 || gap > 20.5 {
		t.Errorf("columns %v apart, want 20", gap)
	}
}
