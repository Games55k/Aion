package overlay

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type PopoverProps struct {
	// Placement is the side of the anchor the panel shows on, centered along it.
	Placement Placement
}

// Popover shows content in a panel by anchor while *open is true. Clicking anchor opens and closes
// it; pressing outside it or Escape closes it. It returns the panel, empty while closed.
func Popover(c *ui.Context, anchor ui.Element, open *bool, p PopoverProps, content func()) ui.Element {
	if anchor.Clicked() {
		*open = !*open
	}
	tk := theme.TokensOf(c)
	return ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
		attach(panel, anchor, p.Placement, 6)
		panel.Padding(12).Radius(tk.BorderRadius).MinWidth(120).
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Transition(motion.Toward(motion.Popup(), p.Placement.side()))
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(content)
	})
}
