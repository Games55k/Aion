package choice

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ImageSelectItem[T comparable] struct {
	// Value is what choosing the option stores.
	Value T
	// Label is the text under the picture, which also names the option.
	Label string
	// Image is the option's picture, covering its frame.
	Image ui.ImageSource
	// Icon shows before the Label.
	Icon func() *ui.SVG
}

type ImageSelectProps[T comparable] struct {
	// Options are the choices, in a row.
	Options []ImageSelectItem[T]
	// Width is a picture's width in DIPs: 144 when zero.
	Width float32
	// Height is a picture's height in DIPs: 86 when zero.
	Height float32
}

// ImageSelect chooses one of p.Options into *value from a row of pictures with labels, as a theme
// or layout picker; the chosen picture shows a ring and the arrows move among them.
func ImageSelect[T comparable](c *ui.Context, value *T, p ImageSelectProps[T]) ui.Element {
	tk := theme.TokensOf(c)
	w, h := p.Width, p.Height
	if w <= 0 {
		w = 144
	}
	if h <= 0 {
		h = 86
	}
	return ui.RadioGroup(c, func() {
		for i, it := range p.Options {
			b := ui.RadioBase(c.Key(i), value, it.Value).Label(it.Label).Column().AlignItems(ui.Center).Gap(4).
				Cursor(ui.CursorPointer).Unselectable()
			chosen := *value == it.Value
			hovered := b.Hovered()
			fg, bg := tk.TextDescription, tk.FillTertiary
			if chosen {
				fg, bg = tk.Text, tk.FillSecondary
			}
			if hovered {
				bg = tk.FillSecondary
				if chosen {
					bg = tk.Fill
				}
			}
			b.TextColor(fg).Transition(checkboxTransition)
			b.Children(func() {
				frame := ui.Box(c).Size(w, h).Radius(tk.BorderRadius).Background(bg).Clip().Transition(checkboxTransition)
				switch {
				case hovered:
					frame.Shadow(0, 0, 0, 2, tk.Text)
				case chosen:
					frame.Shadow(0, 0, 0, 2, tk.TextTertiary)
				}
				frame.Children(func() {
					if it.Image != nil {
						ui.Image(c, it.Image).Fit(ui.Cover).Fill()
					}
				})
				ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
					if it.Icon != nil {
						ui.Icon(c, it.Icon()).FontSize(14)
					}
					ui.Text(c, it.Label).FontSize(14).SingleLine()
				})
			})
		}
	}).Row().Wrap().Gap(16)
}
