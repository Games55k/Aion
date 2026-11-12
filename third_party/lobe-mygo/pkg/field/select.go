package field

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SelectOption struct {
	// Value is what choosing the option puts in the selection.
	Value string
	// Label is the option's text, in the popup and in the trigger once chosen.
	Label string
	// Icon shows before the label.
	Icon *ui.SVG
	// Disabled dims the option and keeps it from being chosen.
	Disabled bool
}

type SelectProps struct {
	// Options are the choices, in order.
	Options []SelectOption
	// Placeholder shows in the trigger while nothing is chosen.
	Placeholder string
	// Variant picks the trigger's surface, as Input's.
	Variant InputVariant
	// Size sets the trigger height (24, 32 or 40), padding and font size.
	Size theme.Size
	// Disabled keeps the select from opening.
	Disabled bool
}

var (
	optionTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: motion.EaseOut, Colors: true}
	arrowTurn        = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.EaseOut}
)

// Select chooses one of p.Options into *selected from a popup below its trigger. The arrows open
// it and move among the options, Enter or Space chooses one and Escape closes it. It returns the
// trigger.
func Select(c *ui.Context, selected *string, p SelectProps) ui.Element {
	tk := theme.TokensOf(c)
	minHeight, fontSize, radius, padX, padY := float32(32), float32(14), tk.BorderRadius, float32(11), float32(4)
	switch p.Size {
	case theme.Small:
		minHeight, fontSize, radius, padX, padY = 24, 12, tk.BorderRadiusSM, 8, 0
	case theme.Large:
		minHeight, fontSize, radius, padX, padY = 40, 16, tk.BorderRadiusLG, 12, 6
	}
	sel := ui.SelectBase(c, selected)
	tr := sel.Trigger.MinHeight(minHeight).Padding(padY, padX).Gap(8).Radius(radius).FontSize(fontSize).
		AlignItems(ui.Center).TextColor(tk.Text).Cursor(ui.CursorPointer).Unselectable().Transition(fieldTransition)
	if p.Disabled {
		tr.Disabled(true).Cursor(ui.CursorNotAllowed).TextColor(tk.TextDisabled)
	}
	styleField(tr, tk, p.Variant, false, sel.Open())

	var current *SelectOption
	for i := range p.Options {
		if p.Options[i].Value == *selected {
			current = &p.Options[i]
		}
	}
	tr.Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			switch {
			case current != nil:
				if current.Icon != nil {
					ui.Icon(c, current.Icon).TextColor(tk.TextSecondary)
				}
				ui.Text(c, current.Label).SingleLine()
			default:
				ui.Text(c, p.Placeholder).TextColor(tk.TextPlaceholder).SingleLine()
			}
		})
		turn := float32(0)
		if sel.Open() {
			turn = 180
		}
		arrow := ui.Icon(c, lucide.ChevronDown()).FontSize(16).TextColor(tk.TextSecondary).Shrink(0)
		arrow.Rotate(arrow.AnimateTo("turn", turn, arrowTurn))
	})

	width := tr.Bounds().W
	sel.Popup(func(panel ui.Element) {
		panel.Margin(4, 0, 0, 0).Padding(4).Radius(tk.BorderRadius).MinWidth(width).MaxHeight(256).
			Background(tk.BgElevated).Transition(motion.Popup()).AttachedOrigin()
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(func() {
			for _, o := range p.Options {
				selectOption(c.Key(o.Value), tk, sel, o, o.Value == *selected)
			}
		})
	})
	return tr
}

func selectOption(c *ui.Context, tk *theme.Tokens, sel ui.SelectParts[string], o SelectOption, chosen bool) {
	item := sel.Item(o.Value).Disabled(o.Disabled).Padding(6, 12).MinHeight(32).Gap(8).AlignItems(ui.Center).
		Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).FontSize(14).TextColor(tk.Text).Transition(optionTransition)
	switch {
	case o.Disabled:
		item.TextColor(tk.TextDisabled).Cursor(ui.CursorNotAllowed)
	case item.Highlighted():
		item.Background(tk.FillTertiary).Cursor(ui.CursorPointer)
	default:
		item.Cursor(ui.CursorPointer)
	}
	item.Children(func() {
		if o.Icon != nil {
			ui.Icon(c, o.Icon).FontSize(14).TextColor(tk.TextSecondary)
		}
		ui.Text(c, o.Label).Grow(1).SingleLine()
		if chosen {
			ui.Icon(c, lucide.Check()).FontSize(14).TextColor(tk.Primary).Margin(0, 0, 0, 8)
		}
	})
}
