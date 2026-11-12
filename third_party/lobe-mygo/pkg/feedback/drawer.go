package feedback

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Placement is the edge of the window a Drawer slides in from.
type Placement int

const (
	Right Placement = iota
	Left
	Top
	Bottom
)

type DrawerProps struct {
	// Placement is the edge the drawer slides in from.
	Placement Placement
	// Title shows in the header.
	Title string
	// Size is the drawer's width, or height at Top and Bottom: 378 DIPs when zero.
	Size float32
	// NoCloseButton hides the header's close button; Escape and the backdrop still close it.
	NoCloseButton bool
	// Footer, if set, builds a row of actions along the bottom, aligned to the end.
	Footer func()
}

// Drawer slides a panel in from an edge of the window over a dimmed backdrop while *open is true.
// The close button, Escape and a click on the backdrop close it.
func Drawer(c *ui.Context, open *bool, p DrawerProps, content func()) {
	tk := theme.TokensOf(c)
	size := p.Size
	if size == 0 {
		size = 378
	}
	present(c, open, motionOf(motion.Drawer(p.Placement.side())), func(backdrop, panel ui.Element, shown, dim float32, _ bool) {
		backdrop.Background(tk.BgMask.Alpha(dim))
		away := 100 * (1 - shown)
		panel.Absolute().Background(tk.BgContainer).TextColor(tk.Text).FontSize(tk.FontSize)
		switch p.Placement {
		case Left:
			panel.Left(0).Top(0).Bottom(0).Width(size).TranslatePercent(-away, 0)
		case Top:
			panel.Top(0).Left(0).Right(0).Height(size).TranslatePercent(0, -away)
		case Bottom:
			panel.Bottom(0).Left(0).Right(0).Height(size).TranslatePercent(0, away)
		default:
			panel.Right(0).Top(0).Bottom(0).Width(size).TranslatePercent(away, 0)
		}
		drawerShadow(panel, p.Placement)
		panel.Children(func() {
			drawerEdge(c, tk, p.Placement)
			if p.Title != "" || !p.NoCloseButton {
				ui.Row(c).MinHeight(56).Padding(12, 16).Gap(8).Children(func() {
					ui.Text(c, p.Title).FontSize(17).FontWeight(600).Grow(1)
					if !p.NoCloseButton && dialogClose(c, tk, 18).Margin(-4, -8, -4, 0).Clicked() {
						*open = false
					}
				})
				ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(tk.Split)
			}
			ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
				ui.Column(c).Padding(12, 16).Children(content)
			})
			if p.Footer != nil {
				ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(tk.Split)
				ui.Row(c).Gap(8).Padding(12, 16).Justify(ui.End).Children(p.Footer)
			}
		})
	})
}

func (p Placement) side() motion.Side {
	switch p {
	case Left:
		return motion.Left
	case Top:
		return motion.Top
	case Bottom:
		return motion.Bottom
	}
	return motion.Right
}

func drawerEdge(c *ui.Context, tk *theme.Tokens, p Placement) {
	line := ui.Box(c).Absolute().Background(tk.Border)
	switch p {
	case Left:
		line.Right(0).Top(0).Bottom(0).Width(1)
	case Top:
		line.Bottom(0).Left(0).Right(0).Height(1)
	case Bottom:
		line.Top(0).Left(0).Right(0).Height(1)
	default:
		line.Left(0).Top(0).Bottom(0).Width(1)
	}
}

func drawerShadow(panel ui.Element, p Placement) {
	layers := []struct{ offset, blur, spread, alpha float32 }{{6, 16, 0, 0.08}, {3, 6, -4, 0.12}, {9, 28, 8, 0.05}}
	for _, l := range layers {
		x, y := float32(0), float32(0)
		switch p {
		case Left:
			x = l.offset
		case Top:
			y = l.offset
		case Bottom:
			y = -l.offset
		default:
			x = -l.offset
		}
		panel.Shadow(x, y, l.blur, l.spread, ui.RGBA(0, 0, 0, l.alpha))
	}
}
