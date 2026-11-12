package feedback

import (
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ProgressType int

const (
	ProgressLine ProgressType = iota
	ProgressCircle
)

type ProgressVariant int

const (
	ProgressBar ProgressVariant = iota
	ProgressSegments
	ProgressInset
)

type ProgressStatus int

const (
	ProgressNormal ProgressStatus = iota
	ProgressActive
	ProgressSuccess
	ProgressException
)

type ProgressProps struct {
	// Type draws a line or a circle.
	Type ProgressType
	// Variant draws the line as one bar, as Segments blocks, or as a filled pill beside the rest.
	Variant ProgressVariant
	// Status colors it: Active adds a moving shine, Success and Exception show a check and a cross
	// instead of the percent.
	Status ProgressStatus
	// Size sets the line's thickness (2, 4 or 6) and the circle's diameter (20, 40 or 72).
	Size theme.Size
	// ShowInfo shows the percent, or the status's icon.
	ShowInfo bool
	// Label shows above the line, on the start side.
	Label string
	// Format writes the percent: "40%" when nil.
	Format func(percent float32) string
	// Segments is how many blocks the Segments variant has: 20 when zero.
	Segments int
	// Color fills the progress instead of the status color.
	Color ui.Color
}

var (
	progressMove  = ui.Animation{Duration: 300 * time.Millisecond, Ease: motion.CSSEase}
	progressShine = 1200 * time.Millisecond
)

// Progress shows how far along a task is, from 0 to 100 percent; the fill grows to a new percent
// over 300 ms.
func Progress(c *ui.Context, percent float32, p ProgressProps) ui.Element {
	tk := theme.TokensOf(c)
	pct := min(max(percent, 0), 100)
	fill := p.Color
	if fill == (ui.Color{}) {
		fill = map[ProgressStatus]ui.Color{ProgressNormal: tk.Primary, ProgressActive: tk.Info,
			ProgressSuccess: tk.Success, ProgressException: tk.Error}[p.Status]
	}
	info := func(size float32) {
		switch p.Status {
		case ProgressSuccess:
			ui.Icon(c, lucide.Check()).FontSize(size).TextColor(fill)
		case ProgressException:
			ui.Icon(c, lucide.X()).FontSize(size).TextColor(fill)
		default:
			text := strconv.FormatFloat(float64(pct), 'f', -1, 32) + "%"
			if p.Format != nil {
				text = p.Format(pct)
			}
			ui.Text(c, text).SingleLine().FontFeatures("tnum")
		}
	}
	if p.Type == ProgressCircle {
		return progressCircle(c, tk, pct, fill, p, info)
	}
	thick := map[theme.Size]float32{theme.Small: 2, theme.Middle: 4, theme.Large: 6}[p.Size]
	block := map[theme.Size]float32{theme.Small: 4, theme.Middle: 8, theme.Large: 12}[p.Size]

	root := ui.Column(c).Gap(6).FillWidth().Role(ui.RoleProgress).Range(0, 100, float64(pct))
	shown := root.AnimateTo("pct", pct, progressMove) / 100
	root.Children(func() {
		switch p.Variant {
		case ProgressSegments, ProgressInset:
			ui.Row(c).Gap(10).AlignItems(ui.Center).FillWidth().Children(func() {
				if p.Variant == ProgressSegments {
					n := p.Segments
					if n <= 0 {
						n = 20
					}
					filled := int(math.Round(float64(n) * float64(pct) / 100))
					ui.Row(c).Grow(1).MinWidth(0).Gap(2).Height(block).Children(func() {
						for i := range n {
							bg := tk.FillTertiary
							if i < filled {
								bg = fill
							}
							ui.Box(c.Key(i)).Grow(1).Basis(0).Height(block).Radius(1).Background(bg)
						}
					})
				} else {
					ui.Row(c).Grow(1).MinWidth(0).Gap(3).Height(block).Children(func() {
						if shown > 0 {
							ui.Box(c).Grow(shown).Basis(0).MinWidth(block).Height(block).Radius(block / 2).Background(fill)
						}
						if pct < 100 {
							ui.Box(c).Grow(1 - shown).Basis(0).Height(block).Radius(block / 2).Background(tk.FillSecondary)
						}
					})
				}
				if p.ShowInfo {
					ui.Row(c).Justify(ui.End).MinWidth(32).FontSize(12).Font("monospace").TextColor(tk.TextSecondary).
						Children(func() { info(14) })
				}
			})
		default:
			if p.Label != "" || p.ShowInfo {
				ui.Row(c).Gap(12).Justify(ui.SpaceBetween).FontSize(12).TextColor(tk.TextSecondary).Children(func() {
					ui.Text(c, p.Label)
					if p.ShowInfo {
						ui.Row(c).Font("monospace").TextColor(tk.Text).Children(func() { info(14) })
					}
				})
			}
			track := ui.Box(c).FillWidth().Height(thick).Radius(thick / 2).Clip().Background(tk.FillTertiary)
			track.Children(func() {
				bar := ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).WidthPercent(100 * shown).Radius(thick / 2).Background(fill)
				if p.Status == ProgressActive {
					t := bar.Loop("shine", progressShine, ui.Linear)
					w := track.Bounds().W
					bar.Clip().Children(func() {
						ui.Box(c).Absolute().Top(0).Bottom(0).Width(80).Left(-80 + (w+80)*t).LinearGradient(ui.LinearGradient{
							Angle: 90, Stops: []ui.GradientStop{{Color: ui.RGBA(255, 255, 255, 0), At: 0},
								{Color: ui.RGBA(255, 255, 255, 0.35), At: 0.5}, {Color: ui.RGBA(255, 255, 255, 0), At: 1}},
						})
					})
				}
			})
		}
	})
	return root
}

func progressCircle(c *ui.Context, tk *theme.Tokens, pct float32, fill ui.Color, p ProgressProps, info func(float32)) ui.Element {
	d := map[theme.Size]float32{theme.Small: 20, theme.Middle: 40, theme.Large: 72}[p.Size]
	stroke := map[theme.Size]float32{theme.Small: 2, theme.Middle: 3, theme.Large: 4}[p.Size]
	root := ui.Row(c).Size(d, d).Center().Shrink(0).AlignSelf(ui.Start).Role(ui.RoleProgress).Range(0, 100, float64(pct))
	shown := root.AnimateTo("pct", pct, progressMove) / 100
	track := tk.FillTertiary
	root.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, d/2-stroke
		var ring ui.Path
		pt.StrokePath(ring.Circle(cx, cy, rad), stroke, track)
		if shown <= 0 {
			return
		}
		var arc ui.Path
		steps := max(2, int(96*shown))
		for i := 0; i <= steps; i++ {
			a := float64(shown) * 2 * math.Pi * float64(i) / float64(steps)
			x, y := cx+rad*float32(math.Sin(a)), cy-rad*float32(math.Cos(a))
			if i == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		pt.StrokePath(arc.Cap(ui.RoundCap), stroke, fill)
	})
	if p.ShowInfo && d >= 40 {
		font := float32(12)
		if d >= 72 {
			font = 17
		}
		root.FontSize(font).TextColor(tk.Text).Children(func() { info(font + 2) })
	}
	return root
}
