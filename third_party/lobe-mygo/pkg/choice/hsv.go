package choice

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

type hsva struct{ h, s, v, a float32 }

func toHSV(c ui.Color) hsva {
	r, g, b := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255
	hi, lo := max(r, g, b), min(r, g, b)
	d := hi - lo
	var h float32
	switch {
	case d == 0:
	case hi == r:
		h = 60 * float32(math.Mod(float64((g-b)/d), 6))
	case hi == g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	s := float32(0)
	if hi > 0 {
		s = d / hi
	}
	return hsva{h, s, hi, float32(c.A) / 255}
}

func (x hsva) color() ui.Color {
	c := x.v * x.s
	hp := float64(x.h) / 60
	k := c * float32(1-math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float32
	switch int(hp) % 6 {
	case 0:
		r, g, b = c, k, 0
	case 1:
		r, g, b = k, c, 0
	case 2:
		r, g, b = 0, c, k
	case 3:
		r, g, b = 0, k, c
	case 4:
		r, g, b = k, 0, c
	default:
		r, g, b = c, 0, k
	}
	m := x.v - c
	ch := func(f float32) uint8 { return uint8(math.Round(float64((f + m) * 255))) }
	return ui.Color{R: ch(r), G: ch(g), B: ch(b), A: uint8(math.Round(float64(x.a * 255)))}
}

func parseHex(s string) (ui.Color, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 && len(s) != 8 {
		return ui.Color{}, false
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return ui.Color{}, false
	}
	if len(s) == 6 {
		n = n<<8 | 0xff
	}
	return ui.Color{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, true
}

func formatHex(c ui.Color) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
