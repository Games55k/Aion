package field

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type HotkeyInputProps struct {
	// Placeholder shows while no shortcut is set: "Press keys to record shortcut" when empty.
	Placeholder string
	// ResetValue is the default shortcut; a reset button brings it back while the value differs.
	ResetValue string
	// AllowClear adds a button that removes the shortcut.
	AllowClear bool
	// Conflicts are shortcuts taken elsewhere, which recording refuses.
	Conflicts []string
	// OnConflict runs with a refused shortcut.
	OnConflict func(keys string)
	// Platform names the keys and picks what mod means, as display.HotkeyProps's.
	Platform display.HotkeyPlatform
	// Variant picks the surface, as Input's.
	Variant InputVariant
	// Disabled keeps the shortcut from changing.
	Disabled bool
}

type hotkeyRecord struct {
	invalid, conflict bool
}

// HotkeyInput records a keyboard shortcut into *value, written as display.Hotkey takes it: focus
// it and press the keys. A shortcut needs a modifier and one other key, and may not be one of
// p.Conflicts. It returns the focusable control.
func HotkeyInput(c *ui.Context, value *string, p HotkeyInputProps) ui.Element {
	tk := theme.TokensOf(c)
	col := ui.Column(c).Gap(8).FillWidth()
	var control ui.Element
	col.Children(func() {
		box := ui.Row(c).MinHeight(32).PaddingX(12).Gap(8).Radius(tk.BorderRadius).AlignItems(ui.Center).
			FontSize(14).TextColor(tk.Text).Transition(fieldTransition)
		rec := ui.Local(box, "record", func() hotkeyRecord { return hotkeyRecord{} })
		focused := false
		box.Children(func() {
			control = ui.ButtonBase(c).Role(ui.RoleTextField).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.Start).
				Cursor(ui.CursorText).FocusRing(false)
			if p.Disabled {
				control.Disabled(true).Cursor(ui.CursorNotAllowed)
				box.Opacity(0.5)
			}
			control.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind != ui.InputKeyDown || p.Disabled {
					return false
				}
				if ev.Key == ui.KeyTab && ev.Mods&^ui.Shift == 0 {
					return false
				}
				keys, ok := display.FormatHotkey(ev.Mods, ev.Key, p.Platform)
				if !ok {
					return true
				}
				mods, _, _ := display.HotkeyBinding(keys, p.Platform)
				rec.invalid, rec.conflict = mods == 0, false
				switch {
				case rec.invalid:
				case slices.Contains(p.Conflicts, keys) && keys != p.ResetValue:
					rec.conflict = true
					if p.OnConflict != nil {
						p.OnConflict(keys)
					}
				default:
					*value = keys
				}
				return true
			})
			focused = control.Focused()
			control.Children(func() {
				if *value == "" {
					placeholder := p.Placeholder
					if placeholder == "" {
						placeholder = "Press keys to record shortcut"
					}
					ui.Text(c, placeholder).TextColor(tk.TextDescription).SingleLine()
					return
				}
				display.Hotkey(c, *value, display.HotkeyProps{Platform: p.Platform})
			})
			if focused || *value == "" || p.Disabled {
				return
			}
			ui.Row(c).Gap(4).Margin(0, -6, 0, 0).Children(func() {
				if p.ResetValue != "" && *value != p.ResetValue {
					if action.ActionIcon(c, lucide.Undo2(), action.ActionIconProps{Size: theme.Small, Variant: action.ActionIconFilled, Title: "Reset to default"}).Clicked() {
						*value, rec.invalid, rec.conflict = p.ResetValue, false, false
					}
				}
				if p.AllowClear {
					if action.ActionIcon(c, lucide.X(), action.ActionIconProps{Size: theme.Small, Variant: action.ActionIconFilled, Title: "Clear binding"}).Clicked() {
						*value, rec.invalid, rec.conflict = "", false, false
					}
				}
			})
		})
		styleField(box, tk, p.Variant, rec.invalid || rec.conflict, false)
		if focused {
			box.Background(tk.FillSecondary)
		}
		switch {
		case rec.conflict:
			ui.Text(c, "This shortcut conflicts with an existing one.").FontSize(12).TextColor(tk.Error)
		case rec.invalid:
			ui.Text(c, "Shortcut must include a modifier key (Ctrl, Alt, Shift) and only one regular key.").FontSize(12).TextColor(tk.Error)
		}
	})
	return control
}
