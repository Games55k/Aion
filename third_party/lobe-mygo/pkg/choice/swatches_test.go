package choice

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestColorSwatchesChoose(t *testing.T) {
	red, blue := ui.Hex("#f5222d"), ui.Hex("#1677ff")
	v := red
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			ColorSwatches(c, &v, ColorSwatchesProps{Colors: []ColorSwatch{{Color: red, Title: "Red"}, {Color: blue, Title: "Blue"}}})
		})
	}, 300, 100)
	info, ok := tt.AccessByRole(ui.RoleRadio, "Blue")
	if !ok {
		t.Fatal("no Blue swatch")
	}
	b := info.Bounds
	if b.W != 24 || b.H != 24 {
		t.Errorf("swatch %vx%v, want 24", b.W, b.H)
	}
	if got := testkit.PixelAt(tt, 300, b.X+5, b.Y+12); !testkit.Near(got, blue) {
		t.Errorf("blue swatch %+v", got)
	}
	tt.ClickAt(b.X+12, b.Y+12)
	tt.Advance(time.Second)
	tt.Frame()
	if v != blue {
		t.Errorf("value %+v after clicking Blue", v)
	}
	if info, _ := tt.AccessByRole(ui.RoleRadio, "Blue"); !info.Checked {
		t.Error("Blue not checked")
	}
}

func TestColorSwatchesCheckIsCentered(t *testing.T) {
	blue := ui.Hex("#1677ff")
	v := blue
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			ColorSwatches(c, &v, ColorSwatchesProps{Colors: []ColorSwatch{{Color: blue, Title: "Blue"}}, Size: 32})
		})
	}, 300, 100)
	info, _ := tt.AccessByRole(ui.RoleRadio, "Blue")
	b := info.Bounds
	ink := func(x0, y0, x1, y1 float32) int {
		n := 0
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				if !testkit.Near(testkit.PixelAt(tt, 300, b.X+x, b.Y+y), blue) {
					n++
				}
			}
		}
		return n
	}
	if n := ink(8, 8, 13, 13); n != 0 {
		t.Errorf("%d check pixels in the top-left corner of the swatch, want the check centered", n)
	}
	if n := ink(10, 10, 22, 22); n == 0 {
		t.Error("no check in the middle of the chosen swatch")
	}
}
