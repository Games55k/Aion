package form

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type GroupProps struct {
	// Title heads the group, after Icon; Desc follows it in small type; Extra builds controls at
	// the end of the header.
	Title, Desc string
	Icon        *ui.SVG
	Extra       func()
	// Fixed keeps a Filled or Outlined group open; Borderless groups never collapse.
	Fixed bool
	// DefaultCollapsed starts a collapsible group closed.
	DefaultCollapsed bool
}

type groupState struct{ open bool }

// Group heads a set of fields with a title, in the form's variant: over a line when Borderless,
// in a card that collapses when Filled or Outlined.
func Group(c *ui.Context, f Host, p GroupProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
	h := f.formHost()
	h.groupFields = 0
	variant := h.view.variant
	root := ui.Column(c).AlignSelf(ui.Stretch)
	if variant == Borderless {
		root.Children(func() {
			if p.Title != "" {
				ui.Row(c).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingY(0).Children(func() {
					groupTitle(c, tk, p, 18)
					if p.Extra != nil {
						p.Extra()
					}
				})
				ui.Box(c).Height(1).Margin(16, 0, 0, 0).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
			}
			ui.Column(c).AlignSelf(ui.Stretch).Children(content)
		})
		return root
	}

	root.Radius(tk.BorderRadiusLG)
	if variant == Filled {
		bg := tk.FillQuaternary
		if tk.Dark {
			bg = tk.BgLayout
		}
		root.Background(bg)
	} else {
		root.Border(1, tk.BorderSecondary).Background(tk.BgContainer)
	}
	st := ui.Local(root, "group", func() groupState { return groupState{open: !p.DefaultCollapsed} })
	if p.Fixed {
		st.open = true
	}
	root.Children(func() {
		col := ui.CollapsibleBase(c, &st.open)
		progress := col.Progress()
		tr := col.Trigger.Row().Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(16).AlignSelf(ui.Stretch).
			Cursor(ui.CursorPointer)
		if p.Fixed {
			tr.Disabled(true).Cursor(ui.CursorDefault)
		}
		tr.Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				groupTitle(c, tk, p, 16)
				if !p.Fixed {
					ui.Icon(c, lucide.ChevronDown()).FontSize(16).TextColor(tk.TextDescription).Rotate(180 * progress)
				}
			})
			if p.Extra != nil {
				p.Extra()
			}
		})
		body := col.Panel(func() {
			inner := ui.Column(c).PaddingX(16).AlignSelf(ui.Stretch)
			if variant == Filled {
				inner.Margin(0, 3, 3, 3).Radius(tk.BorderRadius).Border(1, tk.BorderSecondary).Background(tk.BgContainer)
				if !tk.Dark {
					inner.Shadow(0, 1, 0, -1, tk.Border).Shadow(0, 1, 2, -0.5, tk.Border).
						Shadow(0, 2, 2, -1, tk.BorderSecondary).Shadow(0, 3, 6, -4, tk.BorderSecondary)
				}
			}
			inner.Children(content)
		})
		if body != (ui.Element{}) {
			body.AlignSelf(ui.Stretch)
		}
	})
	return root
}

func groupTitle(c *ui.Context, tk *theme.Tokens, p GroupProps, size float32) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		if p.Icon != nil {
			ui.Icon(c, p.Icon).FontSize(size).TextColor(tk.Text)
		}
		ui.Text(c, p.Title).FontSize(size).FontWeight(700).TextColor(tk.Text)
		if p.Desc != "" {
			ui.Text(c, p.Desc).FontSize(12).TextColor(tk.TextDescription)
		}
	})
}
