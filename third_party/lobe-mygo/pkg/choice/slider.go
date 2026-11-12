package choice

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SliderProps struct {
	// Min and Max bound the value: 0 to 100 when both are zero.
	Min, Max float64
	// Step snaps the value to Min and its multiples from it; 0 moves freely by a hundredth.
	Step float64
	// Disabled grays the slider out and keeps it from moving.
	Disabled bool
}

const (
	thumbW = 8
	thumbH = 16
)

var thumbRing = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.EaseOut}

// Slider sets *value between p.Min and p.Max by dragging or clicking along its track, or with the
// arrows, Home and End while it has the focus.
func Slider(c *ui.Context, value *float64, p SliderProps) ui.Element {
	tk := theme.TokensOf(c)
	lo, hi := p.Min, p.Max
	if lo == 0 && hi == 0 {
		hi = 100
	}
	s := ui.SliderBase(c, value, lo, hi).Height(20).PaddingX(thumbW / 2).FillWidth().Cursor(ui.CursorPointer)
	if p.Step > 0 {
		s.Step(p.Step)
	}
	if p.Disabled {
		s.Disabled(true).Cursor(ui.CursorNotAllowed)
	}
	fill := tk.Primary
	if p.Disabled {
		fill = tk.TextQuaternary
	}
	ring := float32(0)
	if !p.Disabled && (s.Hovered() || s.Pressed()) {
		ring = 1
	}
	ring = s.AnimateTo("ring", ring, thumbRing)
	frac := float32(0)
	if hi > lo {
		frac = float32(math.Min(math.Max((*value-lo)/(hi-lo), 0), 1))
	}
	track, ringColor, gap := tk.FillSecondary, tk.PrimaryBorder, tk.BgContainer
	s.Draw(func(pt *ui.Painter, r ui.Rect) {
		x0, w := r.X+thumbW/2, r.W-thumbW
		cy := r.Y + r.H/2
		x := x0 + w*frac
		pt.Fill(ui.Rect{X: x0, Y: cy - 2, W: w, H: 4}, track, 2)
		pt.Fill(ui.Rect{X: x0, Y: cy - 2, W: x - x0, H: 4}, fill, 2)
		thumb := ui.Rect{X: x - thumbW/2, Y: cy - thumbH/2, W: thumbW, H: thumbH}
		if ring > 0 {
			o := 2 * ring
			pt.Fill(ui.Rect{X: thumb.X - o, Y: thumb.Y - o, W: thumb.W + 2*o, H: thumb.H + 2*o}, ringColor, thumbW/2+o)
		} else {
			pt.Fill(ui.Rect{X: thumb.X - 1, Y: thumb.Y - 1, W: thumb.W + 2, H: thumb.H + 2}, gap, thumbW/2+1)
		}
		pt.Fill(thumb, fill, thumbW/2)
	})
	return s
}
