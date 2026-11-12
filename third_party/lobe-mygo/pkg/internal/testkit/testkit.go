package testkit

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func Over(dst, src ui.Color) ui.Color {
	a := float32(src.A) / 255
	ch := func(d, s uint8) uint8 { return uint8(float32(s)*a + float32(d)*(1-a) + 0.5) }
	return ui.RGB(ch(dst.R, src.R), ch(dst.G, src.G), ch(dst.B, src.B))
}

func Near(a, b ui.Color) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) <= 2 && d(a.G, b.G) <= 2 && d(a.B, b.B) <= 2
}

func PixelAt(tt *ui.Tester, width int, x, y float32) ui.Color {
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / float32(width)
	px := img.RGBAAt(int(x*scale), int(y*scale))
	return ui.RGB(px.R, px.G, px.B)
}

func Centered(view func(c *ui.Context)) func(c *ui.Context) {
	return func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Center().Padding(20).Children(func() { view(c) })
	}
}

func Ink(tt *ui.Tester, width int, r ui.Rect, bg ui.Color) ui.Color {
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / float32(width)
	best, far := bg, -1
	for y := int(r.Y * scale); y < int((r.Y+r.H)*scale); y++ {
		for x := int(r.X * scale); x < int((r.X+r.W)*scale); x++ {
			px := img.RGBAAt(x, y)
			d := absDiff(px.R, bg.R) + absDiff(px.G, bg.G) + absDiff(px.B, bg.B)
			if d > far {
				best, far = ui.RGB(px.R, px.G, px.B), d
			}
		}
	}
	return best
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
