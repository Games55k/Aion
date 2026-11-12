package display

import (
	"strings"
	"sync"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type HotkeyVariant int

const (
	HotkeyFilled HotkeyVariant = iota
	HotkeyOutlined
	HotkeyBorderless
)

type HotkeyPlatform int

const (
	HotkeyAuto HotkeyPlatform = iota
	HotkeyApple
	HotkeyOther
)

type HotkeyProps struct {
	// Variant picks the key caps: Filled tints them, Outlined draws their border, Borderless shows
	// the keys as text in one run.
	Variant HotkeyVariant
	// Compact puts all the keys in one cap.
	Compact bool
	// Platform names the keys as Apple devices (⌘ ⌥ ⇧ glyphs) or as others do (Ctrl, Alt, Shift):
	// Auto asks the device the view runs on.
	Platform HotkeyPlatform
}

var isApple = sync.OnceValue(appleDevice)

// Hotkey shows a keyboard shortcut written as keys joined by "+", such as "mod+shift+k": mod is
// Command on Apple devices and Ctrl elsewhere.
func Hotkey(c *ui.Context, keys string, p HotkeyProps) ui.Element {
	tk := theme.TokensOf(c)
	apple := p.Platform == HotkeyApple || p.Platform == HotkeyAuto && isApple()
	parts := splitKeys(keys)
	joined := p.Compact || p.Variant == HotkeyBorderless
	gap := float32(2)
	if p.Variant == HotkeyBorderless {
		gap = 6
	}
	root := ui.Row(c).Gap(gap).AlignItems(ui.Center).Shrink(0)
	root.Children(func() {
		if joined {
			hotkeyCap(c, tk, p.Variant).Gap(6).Children(func() {
				for i, k := range parts {
					hotkeyKey(c.Key(i), k, apple)
				}
			})
			return
		}
		for i, k := range parts {
			hotkeyCap(c.Key(i), tk, p.Variant).Children(func() { hotkeyKey(c, k, apple) })
		}
	})
	return root
}

func hotkeyCap(c *ui.Context, tk *theme.Tokens, v HotkeyVariant) ui.Element {
	const font = 12
	e := ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Height(font * 1.8).MinWidth(font * 1.8).
		PaddingX(8).Radius(tk.BorderRadiusSM).FontSize(font).LineHeight(1.1).TextColor(tk.TextSecondary).Shrink(0)
	switch v {
	case HotkeyOutlined:
		e.Border(1, tk.BorderSecondary)
	case HotkeyBorderless:
		e.PaddingX(4)
	default:
		e.Background(tk.FillTertiary)
	}
	return e
}

type keyName struct {
	apple func() *ui.SVG
	scale float32
	text  string
	icon  func() *ui.SVG
}

var keyNames = map[string]keyName{
	"mod":       {apple: lucide.Command, scale: 0.95, text: "Ctrl"},
	"alt":       {apple: lucide.Option, scale: 0.95, text: "Alt"},
	"option":    {apple: lucide.Option, scale: 0.95, text: "Alt"},
	"ctrl":      {apple: lucide.ChevronUp, scale: 1, text: "Ctrl"},
	"control":   {apple: lucide.ChevronUp, scale: 1, text: "Ctrl"},
	"shift":     {apple: lucide.ArrowBigUp, scale: 1.15, text: "Shift"},
	"meta":      {apple: lucide.Command, scale: 0.95, icon: lucide.Grid2x2},
	"cmd":       {apple: lucide.Command, scale: 0.95, icon: lucide.Grid2x2},
	"command":   {apple: lucide.Command, scale: 0.95, icon: lucide.Grid2x2},
	"enter":     {apple: lucide.CornerDownLeft, scale: 1, text: "Enter"},
	"backspace": {apple: lucide.Delete, scale: 1, text: "Backspace"},
	"tab":       {apple: lucide.ArrowRightToLine, scale: 1, text: "Tab"},
	"up":        {icon: lucide.ArrowUp},
	"down":      {icon: lucide.ArrowDown},
	"left":      {icon: lucide.ArrowLeft},
	"right":     {icon: lucide.ArrowRight},
	"space":     {icon: lucide.Space},
	"comma":     {text: ","},
	"period":    {text: "."},
	"slash":     {text: "?"},
	"semicolon": {text: ";"},
	"minus":     {text: "-"},
	"equal":     {text: "+"},
}

func hotkeyKey(c *ui.Context, key string, apple bool) {
	n, ok := keyNames[strings.ToLower(key)]
	switch {
	case ok && apple && n.apple != nil:
		ui.Icon(c, n.apple()).FontSize(12 * n.scale)
	case ok && n.text != "":
		ui.Text(c, n.text).SingleLine()
	case ok && n.icon != nil:
		ui.Icon(c, n.icon()).FontSize(12)
	default:
		ui.Text(c, startCase(key)).SingleLine()
	}
}

func splitKeys(keys string) []string {
	var out []string
	for _, k := range strings.Split(strings.ReplaceAll(keys, "++", "+plus"), "+") {
		switch k = strings.TrimSpace(k); k {
		case "":
		case "plus":
			out = append(out, "+")
		default:
			out = append(out, k)
		}
	}
	return out
}

func startCase(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// HotkeyBinding reads keys written as Hotkey takes them into the modifiers and key that
// Context.Shortcut takes, with mod as Command on Apple devices and Ctrl elsewhere. It reports
// false when keys names no key it knows.
func HotkeyBinding(keys string, platform HotkeyPlatform) (ui.Modifiers, ui.Key, bool) {
	mods, key, ok := parseHotkey(keys, platform == HotkeyApple || platform == HotkeyAuto && isApple())
	return swapCmd(mods, browserOnMac()), key, ok
}

// mygo's web backend reports Command as Control on a Mac, and Control as Super, so that its
// Cmd shortcuts work in browsers on every system.
func swapCmd(m ui.Modifiers, swap bool) ui.Modifiers {
	if !swap {
		return m
	}
	ctrl, super := m&ui.Ctrl != 0, m&ui.Super != 0
	m &^= ui.Ctrl | ui.Super
	if ctrl {
		m |= ui.Super
	}
	if super {
		m |= ui.Ctrl
	}
	return m
}

var namedKeys = map[string]ui.Key{
	"enter": ui.KeyEnter, "escape": ui.KeyEscape, "esc": ui.KeyEscape, "backspace": ui.KeyBackspace,
	"tab": ui.KeyTab, "space": ui.KeySpace, "delete": ui.KeyDelete, "home": ui.KeyHome, "end": ui.KeyEnd,
	"pageup": ui.KeyPageUp, "pagedown": ui.KeyPageDown, "left": ui.KeyLeft, "right": ui.KeyRight,
	"up": ui.KeyUp, "down": ui.KeyDown, "minus": ui.KeyMinus, "equal": ui.KeyEqual, "comma": ui.KeyComma,
	"period": ui.KeyPeriod, "slash": ui.KeySlash, "semicolon": ui.KeySemicolon,
	"f1": ui.KeyF1, "f2": ui.KeyF2, "f3": ui.KeyF3, "f4": ui.KeyF4, "f5": ui.KeyF5, "f6": ui.KeyF6,
	"f7": ui.KeyF7, "f8": ui.KeyF8, "f9": ui.KeyF9, "f10": ui.KeyF10, "f11": ui.KeyF11, "f12": ui.KeyF12,
}

func parseHotkey(keys string, apple bool) (ui.Modifiers, ui.Key, bool) {
	var mods ui.Modifiers
	var key ui.Key
	found := false
	for _, k := range splitKeys(keys) {
		switch k = strings.ToLower(k); k {
		case "mod":
			if apple {
				mods |= ui.Super
			} else {
				mods |= ui.Ctrl
			}
		case "ctrl", "control":
			mods |= ui.Ctrl
		case "shift":
			mods |= ui.Shift
		case "alt", "option":
			mods |= ui.Alt
		case "meta", "cmd", "command":
			mods |= ui.Super
		default:
			switch r := []rune(k); {
			case len(r) == 1 && r[0] >= 'a' && r[0] <= 'z':
				key = ui.KeyA + ui.Key(r[0]-'a')
			case len(r) == 1 && r[0] >= '0' && r[0] <= '9':
				key = ui.Key0 + ui.Key(r[0]-'0')
			default:
				named, ok := namedKeys[k]
				if !ok {
					return 0, 0, false
				}
				key = named
			}
			found = true
		}
	}
	return mods, key, found
}

// FormatHotkey writes a key pressed with mods, as an InputEvent reports them, the way Hotkey and
// HotkeyBinding take keys ("mod+shift+k"). It reports false for a key it cannot name, as a
// modifier pressed alone.
func FormatHotkey(mods ui.Modifiers, key ui.Key, platform HotkeyPlatform) (string, bool) {
	return formatHotkey(swapCmd(mods, browserOnMac()), key, platform == HotkeyApple || platform == HotkeyAuto && isApple())
}

func formatHotkey(mods ui.Modifiers, key ui.Key, apple bool) (string, bool) {
	var name string
	switch {
	case key >= ui.KeyA && key <= ui.KeyA+25:
		name = string(rune('a' + key - ui.KeyA))
	case key >= ui.Key0 && key <= ui.Key0+9:
		name = string(rune('0' + key - ui.Key0))
	default:
		for n, k := range namedKeys {
			if k == key && n != "esc" {
				name = n
			}
		}
	}
	if name == "" {
		return "", false
	}
	var parts []string
	cmd, other := ui.Ctrl, ui.Super
	otherName := "meta"
	if apple {
		cmd, other, otherName = ui.Super, ui.Ctrl, "ctrl"
	}
	if mods&cmd != 0 {
		parts = append(parts, "mod")
	}
	if mods&other != 0 {
		parts = append(parts, otherName)
	}
	if mods&ui.Alt != 0 {
		parts = append(parts, "alt")
	}
	if mods&ui.Shift != 0 {
		parts = append(parts, "shift")
	}
	return strings.Join(append(parts, name), "+"), true
}
