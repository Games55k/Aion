package field

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SearchBarProps struct {
	// Placeholder shows while the query is empty: "Type keywords..." when empty.
	Placeholder string
	// ShortKey, written as display.Hotkey takes it ("mod+k"), focuses the search from anywhere in
	// the window, and shows as a tag at the end while the search is empty and not focused.
	ShortKey string
	// Platform names the ShortKey's keys and picks what mod means, as display.HotkeyProps's.
	Platform display.HotkeyPlatform
	// Loading spins a loader in place of the search icon.
	Loading bool
	// Variant picks the surface, as Input's.
	Variant InputVariant
	// Size sets the height (24, 32 or 40), padding and font size, as Input's.
	Size theme.Size
	// OnSearch runs with the query when Enter is pressed.
	OnSearch func(query string)
}

// SearchBar is an Input for a search query: a search icon before it, a button clearing the query,
// and an optional window-wide shortcut that focuses it. It returns the text input.
func SearchBar(c *ui.Context, query *string, p SearchBarProps) ui.Element {
	tk := theme.TokensOf(c)
	focusNow := false
	if p.ShortKey != "" {
		if mods, key, ok := display.HotkeyBinding(p.ShortKey, p.Platform); ok && c.Shortcut(mods, key) {
			focusNow = true
		}
	}
	placeholder := p.Placeholder
	if placeholder == "" {
		placeholder = "Type keywords..."
	}

	root := fieldRow(c, tk, p.Size).Gap(4)
	var input ui.Element
	root.Children(func() {
		icon := ui.Icon(c, lucide.Search()).TextColor(tk.TextPlaceholder).Shrink(0)
		if p.Loading {
			icon = ui.Icon(c.Key("loading"), lucide.LoaderCircle()).TextColor(tk.TextPlaceholder).Shrink(0)
			icon.Rotate(icon.Loop("spin", time.Second, ui.Linear) * 360)
		}
		icon.Margin(0, 4, 0, 0)
		input = ui.TextInputBase(c, query).Grow(1).FocusRing(false).
			Placeholder(placeholder).PlaceholderColor(tk.TextPlaceholder)
		if focusNow {
			input.Focus()
		}
		if p.ShortKey != "" {
			if mods, key, ok := display.HotkeyBinding(p.ShortKey, p.Platform); ok {
				input.KeyShortcuts(mods, key)
			}
		}
		if input.Submitted() && p.OnSearch != nil {
			p.OnSearch(*query)
		}
		switch {
		case *query != "":
			clear := ui.ButtonBase(c).Label("Clear").Size(16, 16).Radius(8).Center().Shrink(0).
				Cursor(ui.CursorPointer).TextColor(tk.TextTertiary)
			if clear.Hovered() {
				clear.TextColor(tk.TextSecondary)
			}
			if clear.Clicked() {
				*query = ""
				input.Focus()
			}
			clear.Children(func() { ui.Icon(c, lucide.CircleX()).FontSize(14) })
		case p.ShortKey != "" && !input.Focused():
			display.Hotkey(c, p.ShortKey, display.HotkeyProps{Compact: true, Platform: p.Platform}).TextColor(tk.TextDescription)
		}
	})
	styleField(root, tk, p.Variant, false, input.Focused())
	return input
}
