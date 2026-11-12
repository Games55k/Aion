package display

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type EmptyProps struct {
	// Icon shows above the title, 32 DIPs in the quaternary text color.
	Icon *ui.SVG
	// Title and Description say what is missing and what to do about it.
	Title, Description string
	// Action builds a button or link below the text.
	Action func()
	// Page lays it out for a whole page: the icon beside larger text, aligned to the start.
	Page bool
}

// Empty fills the place of content there is none of yet.
func Empty(c *ui.Context, p EmptyProps) ui.Element {
	tk := theme.TokensOf(c)
	if p.Page {
		root := ui.Row(c).Padding(8).Gap(20).AlignItems(ui.Start).FillWidth()
		root.Children(func() {
			if p.Icon != nil {
				ui.Icon(c, p.Icon).FontSize(36).TextColor(tk.TextQuaternary).Shrink(0)
			}
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				if p.Title != "" {
					ui.Text(c, p.Title).FontSize(18).FontWeight(600).TextColor(tk.Text).Margin(0, 0, 4, 0)
				}
				if p.Description != "" {
					ui.Text(c, p.Description).FontSize(13).TextColor(tk.TextTertiary).MaxWidth(13 * 0.55 * 52)
				}
				if p.Action != nil {
					ui.Row(c).Gap(8).Margin(14, 0, 0, 0).Children(p.Action)
				}
			})
		})
		return root
	}
	root := ui.Column(c).Gap(4).Padding(16, 8).AlignItems(ui.Center).FillWidth()
	root.Children(func() {
		if p.Icon != nil {
			ui.Icon(c, p.Icon).FontSize(32).TextColor(tk.TextQuaternary)
		}
		if p.Title != "" {
			ui.Text(c, p.Title).FontWeight(500).TextColor(tk.Text).TextAlign(ui.Center).Margin(8, 0, 0, 0)
		}
		if p.Description != "" {
			ui.Text(c, p.Description).FontSize(13).TextColor(tk.TextTertiary).TextAlign(ui.Center).MaxWidth(13 * 0.55 * 34)
		}
		if p.Action != nil {
			ui.Row(c).Margin(8, 0, 0, 0).Children(p.Action)
		}
	})
	return root
}
