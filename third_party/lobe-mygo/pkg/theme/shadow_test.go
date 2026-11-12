package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func elevatedPixels(t *testing.T, dark bool, level Elevation) (bg, aboveTop, belowBottom ui.Color) {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		if dark {
			c.SetTheme(darkTheme)
		}
		Apply(c)
		tk := TokensOf(c)
		ui.Column(c).Fill().Center().Background(tk.BgLayout).Children(func() {
			tk.Elevate(ui.Box(c).Size(100, 60).Background(tk.BgElevated), level)
		})
	}, 200, 200)
	img := tt.Image()
	s := img.Bounds().Dx() / 200
	at := func(x, y float32) ui.Color {
		p := img.RGBAAt(int(x*float32(s)), int(y*float32(s)))
		return ui.RGB(p.R, p.G, p.B)
	}
	return at(5, 5), at(100, 69.5), at(100, 150)
}

func luma(c ui.Color) int { return int(c.R) + int(c.G) + int(c.B) }

func TestElevateDrawsTheRingAroundTheBox(t *testing.T) {
	for _, level := range []Elevation{BoxShadow, BoxShadowSecondary, BoxShadowTertiary} {
		bg, edge, _ := elevatedPixels(t, false, level)
		if luma(edge) >= luma(bg) {
			t.Errorf("light level %v: edge %+v not darker than background %+v", level, edge, bg)
		}
		bg, edge, _ = elevatedPixels(t, true, level)
		if luma(edge) <= luma(bg) {
			t.Errorf("dark level %v: edge %+v not lighter than background %+v", level, edge, bg)
		}
	}
}

func TestBoxShadowReachesFurtherThanTertiary(t *testing.T) {
	bg, _, deep := elevatedPixels(t, false, BoxShadow)
	_, _, shallow := elevatedPixels(t, false, BoxShadowTertiary)
	if luma(deep) >= luma(bg) {
		t.Errorf("BoxShadow 20 DIPs below = %+v, want darker than %+v", deep, bg)
	}
	if shallow != bg {
		t.Errorf("BoxShadowTertiary 20 DIPs below = %+v, want background %+v", shallow, bg)
	}
}
