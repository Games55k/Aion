package choice

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestHSVRoundTrips(t *testing.T) {
	for _, c := range []ui.Color{ui.RGB(255, 0, 0), ui.RGB(22, 119, 255), ui.RGB(250, 219, 20), ui.RGB(0, 0, 0), ui.RGB(255, 255, 255), ui.RGBA(114, 46, 209, 0.5)} {
		if got := toHSV(c).color(); got != c {
			t.Errorf("%+v → %+v → %+v", c, toHSV(c), got)
		}
	}
	h := toHSV(ui.RGB(0, 255, 0))
	if math.Abs(float64(h.h-120)) > 0.5 || h.s != 1 || h.v != 1 {
		t.Errorf("green %+v, want h 120 s 1 v 1", h)
	}
}

func TestHexParseAndFormat(t *testing.T) {
	for _, c := range []struct {
		in   string
		want ui.Color
	}{
		{"#1677ff", ui.RGB(22, 119, 255)}, {"1677FF", ui.RGB(22, 119, 255)}, {"#f00", ui.RGB(255, 0, 0)},
		{"#1677ff80", ui.Color{R: 22, G: 119, B: 255, A: 128}},
	} {
		if got, ok := parseHex(c.in); !ok || got != c.want {
			t.Errorf("parseHex(%q) = %+v %v, want %+v", c.in, got, ok, c.want)
		}
	}
	for _, bad := range []string{"", "#12", "#gggggg", "#12345"} {
		if _, ok := parseHex(bad); ok {
			t.Errorf("parseHex(%q) accepted", bad)
		}
	}
	if got := formatHex(ui.RGB(22, 119, 255)); got != "#1677FF" {
		t.Errorf("formatHex %q", got)
	}
}
