package display

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ListVariant int

const (
	ListBorderless ListVariant = iota
	ListFilled
	ListOutlined
)

type ListItem struct {
	// Key is what List returns in the frame the item is clicked, and what Active names.
	Key string
	// Label is the item's text, on one line; Description shows under it in small type.
	Label, Description string
	// Icon shows before the label; Avatar, if set, builds what shows there instead.
	Icon   *ui.SVG
	Avatar func()
	// Extra shows small text at the end of the row.
	Extra string
	// Actions builds buttons that show at the end of the row while the pointer is on it or the
	// focus is in it.
	Actions func()
	// ShowAction keeps the Actions shown.
	ShowAction bool
	// Danger colors the item with the error color.
	Danger bool
	// Disabled dims the item and keeps it from being clicked.
	Disabled bool
	// Divider makes the item a line between groups; its other fields are ignored.
	Divider bool
}

type ListProps struct {
	// Items are the rows, in order.
	Items []ListItem
	// Active is the Key of the row shown as chosen.
	Active string
	// Variant picks the container: Filled tints it and Outlined borders it, both padded by 4.
	Variant ListVariant
	// Compact narrows the rows' side padding from 12 to 8.
	Compact bool
	// Collapse turns the rows, from 0 to 1, into squares centered in RailWidth that show only
	// their icon or avatar, as a side nav collapsing to a rail: follow its progress for the rows
	// to slide along with its width.
	Collapse float32
	// RailWidth is the width the collapsed rows center in: the list's own when zero.
	RailWidth float32
}

var listTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: motion.EaseOut, Colors: true}

// List shows rows of navigation or settings, each with an icon, a label, an optional description
// and extra text; it returns the Key of the row clicked in the frame, or "".
func List(c *ui.Context, p ListProps) string {
	tk := theme.TokensOf(c)
	root := ui.Column(c).Gap(2).AlignSelf(ui.Stretch)
	switch p.Variant {
	case ListFilled:
		root.Padding(4).Radius(tk.BorderRadiusLG).Background(tk.FillQuaternary)
	case ListOutlined:
		root.Padding(4).Radius(tk.BorderRadiusLG).Border(1, tk.BorderSecondary)
	}
	clicked := ""
	root.Children(func() {
		for i, it := range p.Items {
			if it.Divider {
				ui.Box(c.Key(i)).Height(1).Margin(4, 0).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
				continue
			}
			if listRow(c.Key(it.Key), tk, it, it.Key == p.Active, p, root.LayoutBounds().W) {
				clicked = it.Key
			}
		}
	})
	return clicked
}

func listRow(c *ui.Context, tk *theme.Tokens, it ListItem, active bool, p ListProps, width float32) bool {
	clicked := false
	cell := ui.Box(c).AlignSelf(ui.Stretch)
	cell.Children(func() {
		row := ui.ButtonBase(c).Label(it.Label).Disabled(it.Disabled).Row().Gap(12).AlignItems(ui.Center).
			Justify(ui.Start).MinHeight(32).PaddingY(6).PaddingX(12).Radius(tk.BorderRadius).
			FontSize(14).AlignSelf(ui.Stretch).Transition(listTransition)
		pad := float32(12)
		if p.Compact {
			pad = 8
			row.PaddingX(8)
		}
		k := min(max(p.Collapse, 0), 1)
		fg := tk.TextSecondary
		switch {
		case it.Disabled:
			row.Opacity(0.45).Cursor(ui.CursorNotAllowed)
		case active:
			fg = tk.Text
			row.Background(tk.FillSecondary).Cursor(ui.CursorPointer)
		case row.Hovered():
			fg = tk.Text
			row.Background(tk.FillTertiary).Cursor(ui.CursorPointer)
		default:
			row.Cursor(ui.CursorPointer)
		}
		if it.Danger {
			fg = tk.Error
		}
		row.TextColor(fg)
		var lead ui.Element
		row.Children(func() {
			lead = ui.Row(c).Shrink(0).Children(func() {
				switch {
				case it.Avatar != nil:
					it.Avatar()
				case it.Icon != nil:
					ui.Icon(c, it.Icon).FontSize(16)
				}
			})
			text := ui.Column(c).Gap(1).Grow(1).Basis(0).MinWidth(0)
			if k > 0 {
				text.Clip().Opacity(1 - k)
			}
			text.Children(func() {
				label := ui.Text(c, it.Label).SingleLine()
				if active {
					label.FontWeight(500)
				}
				if it.Description != "" {
					ui.Text(c, it.Description).SingleLine().FontSize(12).LineHeight(1.4).TextColor(tk.TextDescription)
				}
			})
			if it.Extra != "" && k < 1 {
				ui.Text(c, it.Extra).FontSize(12).TextColor(tk.TextDescription).Shrink(0).Opacity(1 - k)
			}
		})
		if k > 0 {
			railRow(c, row, lead, k, pad, p.RailWidth, width)
		}
		clicked = row.Clicked()
		if it.Actions != nil && (it.ShowAction || cell.Hovered() || cell.FocusWithin()) {
			ui.Row(c).Absolute().Right(8).Top(0).Bottom(0).Gap(4).AlignItems(ui.Center).Children(it.Actions)
		}
	})
	return clicked
}

// railRow narrows row toward a square centered in the rail, its lead centered in the square, as
// k goes to 1. The sizes are those of the last frame, as the rows keep their height.
func railRow(c *ui.Context, row, lead ui.Element, k, pad, rail, width float32) {
	sq, glyph := row.LayoutBounds().H, lead.LayoutBounds().W
	if sq <= 0 || glyph <= 0 || width <= 0 {
		c.Invalidate()
		return
	}
	if rail <= 0 {
		rail = width
	}
	lerp := func(a, b float32) float32 { return a + (b-a)*k }
	row.AlignSelf(ui.Start).Width(lerp(width, sq)).Gap(lerp(12, 0)).
		Margin(0, 0, 0, lerp(0, (rail-sq)/2)).Padding(6, lerp(pad, 0), 6, lerp(pad, (sq-glyph)/2))
}
