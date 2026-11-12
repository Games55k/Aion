package feedback

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SpinVariant int

const (
	SpinDefault SpinVariant = iota
	SpinNetwork
)

type SpinProps struct {
	// Variant is a turning loader, or a network of pulsing nodes for work over a network, as an
	// agent's.
	Variant SpinVariant
	// Size is the indicator's size: 14, 20 or 32 DIPs.
	Size theme.Size
	// CustomSize, if set, is the indicator's size in DIPs instead.
	CustomSize float32
	// Percent, if set, shows a ring filled that far, from 0 to 100, instead of the loader.
	Percent *float32
	// Tip shows under the indicator over content.
	Tip string
	// Spinning, if set, shows the indicator only while true; content shows either way.
	Spinning *bool
}

// Spin shows that something is loading: an indicator alone, or over what content builds, dimmed.
func Spin(c *ui.Context, p SpinProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
	size := map[theme.Size]float32{theme.Small: 14, theme.Middle: 20, theme.Large: 32}[p.Size]
	if p.CustomSize > 0 {
		size = p.CustomSize
	}
	spinning := p.Spinning == nil || *p.Spinning
	if content == nil {
		root := ui.Row(c).AlignItems(ui.Center).TextColor(tk.TextSecondary).Role(ui.RoleStatus).AlignSelf(ui.Start).Shrink(0)
		if spinning {
			root.Children(func() { spinGlyph(c, tk, p, size) })
		}
		return root
	}
	root := ui.Box(c)
	root.Children(func() {
		content()
		if !spinning {
			return
		}
		ui.Column(c).Absolute().Fill().Gap(8).Center().Background(tk.BgContainer.Alpha(0.7)).TextColor(tk.TextSecondary).
			Role(ui.RoleStatus).Children(func() {
			spinGlyph(c, tk, p, size)
			if p.Tip != "" {
				ui.Text(c, p.Tip).FontSize(tk.FontSizeSM)
			}
		})
	})
	return root
}

func spinGlyph(c *ui.Context, tk *theme.Tokens, p SpinProps, size float32) {
	switch {
	case p.Variant == SpinNetwork:
		networkGlyph(c, tk, size)
	case p.Percent != nil:
		percentRing(c, tk, *p.Percent, size)
	default:
		icon := ui.Icon(c, lucide.LoaderCircle()).FontSize(size)
		icon.Rotate(icon.Loop("spin", time.Second, ui.Linear) * 360)
	}
}

func percentRing(c *ui.Context, tk *theme.Tokens, percent, size float32) {
	frac := min(max(percent, 0), 100) / 100
	track, fill := tk.FillTertiary, tk.Primary
	ui.Box(c).Size(size, size).Shrink(0).Draw(func(pt *ui.Painter, r ui.Rect) {
		k := r.W / 24
		cx, cy, rad := r.X+12*k, r.Y+12*k, 9*k
		var ring ui.Path
		pt.StrokePath(ring.Circle(cx, cy, rad), 2.5*k, track)
		if frac <= 0 {
			return
		}
		var arc ui.Path
		steps := max(2, int(96*frac))
		for i := 0; i <= steps; i++ {
			a := float64(frac) * 2 * math.Pi * float64(i) / float64(steps)
			x, y := cx+rad*float32(math.Sin(a)), cy-rad*float32(math.Cos(a))
			if i == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		pt.StrokePath(arc.Cap(ui.RoundCap), 2.5*k, fill)
	})
}

const networkPeriod = 20 * time.Second

func networkGlyph(c *ui.Context, tk *theme.Tokens, size float32) {
	box := ui.Box(c).Size(size, size).Shrink(0)
	t := float64(box.Loop("network", networkPeriod, ui.Linear)) * networkPeriod.Seconds()
	color := tk.TextSecondary
	box.Draw(func(pt *ui.Painter, r ui.Rect) {
		k := size / 100
		at := func(x, y float32) (float32, float32) { return r.X + x*k, r.Y + y*k }
		node := max(6, 2/k) * k / 2
		particle := max(3, 1.5/k) * k / 2
		for layer := range 2 {
			for from := range 3 {
				for to := range 3 {
					var line ui.Path
					x1, y1 := at(25+float32(layer)*25, 25+float32(from)*25)
					x2, y2 := at(50+float32(layer)*25, 25+float32(to)*25)
					pt.StrokePath(line.MoveTo(x1, y1).LineTo(x2, y2), 0.5*k, color.Alpha(0.3))
				}
			}
		}
		i := 0
		for layer := range 3 {
			for row := range 3 {
				a := keyframe(frac((t-0.2*float64(i))/2), 0.3, 1)
				x, y := at(25+float32(layer)*25, 25+float32(row)*25)
				var dot ui.Path
				pt.FillPath(dot.Circle(x, y, node), color.Alpha(a))
				i++
			}
		}
		for n := range 3 {
			ph := frac((t - 0.6*float64(n)) / 2)
			x, y := at(25+50*float32(motion.CSSEase(float32(ph))), 50)
			var dot ui.Path
			pt.FillPath(dot.Circle(x, y, particle), color.Alpha(keyframe(ph, 0.5, 1)))
		}
		ph := frac(t / 2)
		s := keyframe(ph, 0.8, 1) * 6 * k
		cx, cy := at(50, 50)
		pt.Fill(ui.Rect{X: cx - s/2, Y: cy - s/2, W: s, H: s}, color.Alpha(keyframe(ph, 0.5, 1)), 0)
		var ring ui.Path
		turn := float32(t/networkPeriod.Seconds()) * 2 * math.Pi * 40 * k
		ring.Circle(cx, cy, 39.5*k).Dash(-turn, 3*k, 3*k)
		pt.StrokePath(&ring, k, color.Alpha(0.4))
	})
}

func frac(x float64) float64 { return x - math.Floor(x) }

func keyframe(phase float64, rest, peak float32) float32 {
	if phase < 0.5 {
		return rest + (peak-rest)*motion.CSSEase(float32(phase*2))
	}
	return peak + (rest-peak)*motion.CSSEase(float32(phase*2-1))
}
