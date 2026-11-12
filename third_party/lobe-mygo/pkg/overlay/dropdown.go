package overlay

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type MenuItem struct {
	// Key is what DropdownMenu returns in the frame the item is chosen.
	Key string
	// Label is the item's text.
	Label string
	// Icon shows before the label.
	Icon *ui.SVG
	// Hotkey shows a key combination in a key cap at the end of the row, written as display.Hotkey
	// takes it, as "mod+d"; it does not bind the keys.
	Hotkey string
	// Extra shows text at the end of the row, after the label.
	Extra string
	// Danger colors the item with the error color.
	Danger bool
	// Disabled dims the item and keeps it from being chosen.
	Disabled bool
	// Divider makes the item a line between groups; its other fields are ignored.
	Divider bool
}

type DropdownMenuProps struct {
	// Items are the menu's rows, in order.
	Items []MenuItem
}

var menuItemTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: motion.EaseOut, Colors: true}

// DropdownMenu shows a menu of p.Items below a trigger while *open is true. trigger gives the
// trigger, a button without a look, its look and content, as action.StyleButton does. Pressing the
// trigger, or Enter, Space or Down on it, opens the menu; it returns the Key of the item chosen in
// the frame, which closes the menu, or "".
func DropdownMenu(c *ui.Context, open *bool, p DropdownMenuProps, trigger func(t ui.Element)) string {
	tk := theme.TokensOf(c)
	m := ui.MenuBase(c, open)
	trigger(m.Trigger)
	chosen := ""
	m.Popup(func(panel ui.Element) {
		panel.Margin(6, 0, 0, 0).Padding(4).Radius(tk.BorderRadius).MinWidth(220).
			Background(tk.BgElevated).Transition(motion.Popup()).AttachedOrigin()
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(func() { menuRows(c, tk, m, p.Items, &chosen) })
	})
	return chosen
}

func menuItem(c *ui.Context, tk *theme.Tokens, m ui.MenuParts, it MenuItem) ui.Element {
	item := m.Item().Disabled(it.Disabled).
		PaddingY(6).PaddingX(12).MinHeight(32).Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).
		FontSize(14).Transition(menuItemTransition)
	fg, icon := tk.Text, tk.TextSecondary
	if it.Danger {
		fg, icon = tk.Error, tk.Error
	}
	switch {
	case it.Disabled:
		fg, icon = tk.TextDisabled, tk.TextDisabled
		item.Opacity(0.5)
	case item.Pressed():
		item.Background(tk.FillSecondary)
	case item.Highlighted() && it.Danger:
		item.Background(tk.ErrorBg)
	case item.Highlighted():
		item.Background(tk.FillTertiary)
	}
	item.TextColor(fg)
	item.Children(func() {
		ui.Row(c).Grow(1).Children(func() {
			if it.Icon != nil {
				ui.Icon(c, it.Icon).FontSize(14).TextColor(icon).Margin(0, 12, 0, 0)
			}
			ui.Text(c, it.Label).Grow(1).SingleLine()
			if it.Extra != "" {
				ui.Text(c, it.Extra).Font("monospace").FontSize(12).TextColor(tk.TextTertiary).Padding(0, 0, 0, 16).SingleLine()
			}
			if it.Hotkey != "" {
				// The cap sits 5 DIPs from the row's top, bottom and end alike, so its glyphs line up
				// with the icon's 12 DIP inset; the negative margins keep it from growing the 32 DIP row.
				display.Hotkey(c, it.Hotkey, display.HotkeyProps{Compact: true}).Margin(-2, -7, -2, 16)
			}
		})
	})
	return item
}

// ContextMenu gives target a menu of p.Items that opens where target is clicked with the secondary
// button, or Control-clicked on macOS, and below it on the menu key or Shift+F10 while it has the
// focus. It returns the Key of the item chosen in the frame, which closes the menu, or "".
func ContextMenu(c *ui.Context, target ui.Element, p DropdownMenuProps) string {
	tk := theme.TokensOf(c)
	open := ui.Local(target, "contextMenu", func() bool { return false })
	m := ui.ContextMenuBase(c, target, open)
	chosen := ""
	m.Popup(func(panel ui.Element) {
		panel.Padding(4).Radius(tk.BorderRadius).MinWidth(220).
			Background(tk.BgElevated).Transition(motion.Popup()).AttachedOrigin()
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(func() { menuRows(c, tk, m, p.Items, &chosen) })
	})
	return chosen
}

func menuRows(c *ui.Context, tk *theme.Tokens, m ui.MenuParts, items []MenuItem, chosen *string) {
	for i, it := range items {
		if it.Divider {
			ui.Box(c.Key(i)).Height(1).Margin(4, 0).AlignSelf(ui.Stretch).Background(tk.Border)
			continue
		}
		if menuItem(c, tk, m, it).Clicked() && !it.Disabled {
			*chosen = it.Key
		}
	}
}
