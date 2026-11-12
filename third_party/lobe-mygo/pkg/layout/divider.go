package layout

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type DividerProps struct {
	// Vertical draws a line one font size tall between inline items, in place of a line across
	// its column.
	Vertical bool
	// Dashed draws the line in dashes.
	Dashed bool
	// Label shows in the middle of a horizontal line.
	Label string
}

// Divider draws a thin line between content: across its column, or upright between the items of a
// row with Vertical.
func Divider(c *ui.Context, p DividerProps) ui.Element {
	tk := theme.TokensOf(c)
	if p.Vertical {
		return line(c, tk, p.Dashed).Width(1).Height(tk.FontSize).AlignSelf(ui.Center)
	}
	if p.Label == "" {
		return line(c, tk, p.Dashed).Height(1).AlignSelf(ui.Stretch)
	}
	row := ui.Row(c).Gap(12).AlignSelf(ui.Stretch).AlignItems(ui.Center)
	row.Children(func() {
		line(c, tk, p.Dashed).Height(1).Grow(1)
		ui.Text(c, p.Label).FontSize(12).TextColor(tk.TextDescription).SingleLine()
		line(c, tk, p.Dashed).Height(1).Grow(1)
	})
	return row
}

func line(c *ui.Context, tk *theme.Tokens, dashed bool) ui.Element {
	color := tk.BorderSecondary
	if !dashed {
		return ui.Box(c).Shrink(0).Background(color)
	}
	return ui.Box(c).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		const dash, gap = 4, 4
		if r.W >= r.H {
			for x := r.X; x < r.X+r.W; x += dash + gap {
				p.Line(x, r.Y+r.H/2, min(x+dash, r.X+r.W), r.Y+r.H/2, r.H, color)
			}
			return
		}
		for y := r.Y; y < r.Y+r.H; y += dash + gap {
			p.Line(r.X+r.W/2, y, r.X+r.W/2, min(y+dash, r.Y+r.H), r.W, color)
		}
	})
}
