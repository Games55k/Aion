package choice

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var (
	dotIn  = ui.Animation{Duration: 200 * time.Millisecond, Ease: motion.EaseOut}
	dotOut = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.ExitEase}
)

type RadioProps struct {
	// Size is the circle's diameter in DIPs; 0 is 16.
	Size float32
	// Disabled dims the radio and keeps it from being chosen.
	Disabled bool
}

type RadioGroupProps struct {
	// Vertical stacks the radios; they sit in a wrapping row otherwise.
	Vertical bool
	// Gap is the space between the radios: 12 DIPs when zero.
	Gap float32
}

// Radio shows a radio button choosing value into *selected, with a label. Build radios of one
// choice in a RadioGroup, where the arrows move between them.
func Radio[T comparable](c *ui.Context, selected *T, value T, label string, p RadioProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 16
	}
	row := ui.RadioBase(c, selected, value).Gap(float32(int(size / 2))).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Unselectable()
	if p.Disabled {
		row.Disabled(true).Cursor(ui.CursorNotAllowed)
	}
	checked := *selected == value
	border, bg := tk.BorderSecondary, tk.BgContainer
	switch {
	case p.Disabled:
		border, bg = tk.Fill, tk.Fill
	case checked:
		border, bg = tk.Primary, tk.Primary
	case row.Hovered():
		border = tk.Border
	}
	row.Children(func() {
		circle := ui.Box(c).Size(size, size).Radius(size/2).Border(1, border).Background(bg).
			Center().Shrink(0).Transition(checkboxTransition)
		if p.Disabled {
			circle.Opacity(0.25)
		}
		in := float32(0)
		anim := dotOut
		if checked {
			in, anim = 1, dotIn
		}
		scale := circle.AnimateTo("dot", in, anim)
		circle.Children(func() {
			if scale > 0 {
				dot := float32(int(size*0.375 + 0.5))
				ui.Box(c).Size(dot, dot).Radius(dot/2).Background(tk.BgLayout).Scale(scale, scale)
			}
		})
		if label != "" {
			ui.Text(c, label).SingleLine()
		}
	})
	return row
}

// RadioGroup holds the radios fn builds as one choice: Tab stops at the chosen one and the arrows
// choose among them.
func RadioGroup(c *ui.Context, p RadioGroupProps, fn func()) ui.Element {
	gap := p.Gap
	if gap == 0 {
		gap = 12
	}
	g := ui.RadioGroup(c, fn).Gap(gap)
	if !p.Vertical {
		g.Row().Wrap().AlignItems(ui.Center)
	}
	return g
}
