package overlay

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var tipContentTransition = ui.ElementTransition{Duration: 120 * time.Millisecond, Ease: motion.EaseOut, Enter: &ui.Motion{}}

// One tip shows at a time, so moving to another anchor rebuilds the same tip there: animating its
// box would lay the text out at every size on the way. It jumps instead, and its content fades in.
func tipTransition(p Placement) ui.ElementTransition {
	tr := motion.Toward(motion.Tooltip(), p.side())
	tr.Colors = true
	return tr
}

type TooltipProps struct {
	// Placement is the side of the anchor the tip shows on.
	Placement Placement
	// Hotkey shows a key combination after the text, written as display.Hotkey takes it, as "mod+k".
	Hotkey string
}

// Tooltip shows text by anchor once the pointer rests on it, or the keyboard focus comes to it,
// and describes anchor to assistive technology. It returns the tip, empty while it does not show.
func Tooltip(c *ui.Context, anchor ui.Element, text string, p TooltipProps) ui.Element {
	if text == "" {
		return ui.Element{}
	}
	anchor.Description(text)
	tk := theme.TokensOf(c)
	return ui.TooltipBase(c, anchor, func(tip ui.Element) {
		attach(tip, anchor, p.Placement, 6)
		tip.Padding(4, 8).Radius(tk.BorderRadiusSM).MaxWidth(320).
			Background(tk.BgElevated).TextColor(tk.TextLabel).FontSize(tk.FontSizeSM).
			Transition(tipTransition(p.Placement))
		tk.Elevate(tip, theme.BoxShadowTertiary)
		tip.Children(func() {
			ui.Row(c.Key(text)).Gap(6).Transition(tipContentTransition).Children(func() {
				ui.Text(c, text)
				if p.Hotkey != "" {
					display.Hotkey(c, p.Hotkey, display.HotkeyProps{Compact: true})
				}
			})
		})
	})
}
