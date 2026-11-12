package field

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type AutoCompleteOption struct {
	// Value is what choosing the option writes into the field.
	Value string
	// Label shows in the popup instead of the value.
	Label string
	// Disabled dims the option and keeps it from being chosen.
	Disabled bool
}

type AutoCompleteProps struct {
	// Options are the suggestions; those whose value or label holds what was typed show.
	Options []AutoCompleteOption
	// EmptyText shows in the popup when no option matches; the popup closes without it.
	EmptyText string
	// Placeholder shows while the field is empty.
	Placeholder string
	// Prefix shows an icon before the text.
	Prefix *ui.SVG
	// Variant picks the surface, as Input's.
	Variant InputVariant
	// Size sets the height (24, 32 or 40), padding and font size, as Input's.
	Size theme.Size
}

// AutoComplete is an Input that suggests p.Options below it as the user types, matching what was
// typed, and writes the one chosen into *value. It returns the text input.
func AutoComplete(c *ui.Context, value *string, p AutoCompleteProps) ui.Element {
	tk := theme.TokensOf(c)
	root := fieldRow(c, tk, p.Size).Gap(8)
	var input ui.Element
	root.Children(func() {
		affix(c, tk, p.Prefix)
		cb := ui.ComboboxBase(c, value)
		input = cb.Input.Grow(1).MinWidth(0).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder)
		q := strings.ToLower(strings.TrimSpace(*value))
		var shown []AutoCompleteOption
		for _, o := range p.Options {
			if q == "" || !cb.Filtering() || strings.Contains(strings.ToLower(o.Value), q) || strings.Contains(strings.ToLower(o.Label), q) {
				shown = append(shown, o)
			}
		}
		if len(shown) > 0 || p.EmptyText != "" {
			field, at := root.Bounds(), input.Bounds()
			cb.Popup(func(panel ui.Element) {
				panel.Margin(8, 0, 0, field.X-at.X).Padding(4).Radius(tk.BorderRadius).MinWidth(field.W).MaxHeight(256).
					Background(tk.BgElevated).Transition(motion.Popup()).AttachedOrigin()
				tk.Elevate(panel, theme.BoxShadowSecondary)
				panel.Children(func() {
					if len(shown) == 0 {
						ui.Text(c, p.EmptyText).FontSize(14).TextColor(tk.TextTertiary).Padding(6, 12)
					}
					for _, o := range shown {
						completeOption(c.Key(o.Value), tk, cb, o)
					}
				})
			})
		}
		if v, ok := cb.Chosen(); ok {
			*value = v
		}
	})
	styleField(root, tk, p.Variant, false, input.Focused())
	return input
}

func completeOption(c *ui.Context, tk *theme.Tokens, cb ui.ComboboxParts, o AutoCompleteOption) {
	item := cb.Item(o.Value).Disabled(o.Disabled).Padding(6, 12).MinHeight(32).AlignItems(ui.Center).
		Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).FontSize(14).TextColor(tk.Text).Transition(optionTransition)
	switch {
	case o.Disabled:
		item.TextColor(tk.TextDisabled).Cursor(ui.CursorNotAllowed)
	case item.Highlighted():
		item.Background(tk.FillTertiary).Cursor(ui.CursorPointer)
	default:
		item.Cursor(ui.CursorPointer)
	}
	label := o.Label
	if label == "" {
		label = o.Value
	}
	item.Children(func() { ui.Text(c, label).Grow(1).SingleLine() })
}
