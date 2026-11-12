package display

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func hotkeyTester(keys string, p HotkeyProps) *ui.Tester {
	return ui.NewTester(testkit.Centered(func(c *ui.Context) { Hotkey(c, keys, p) }), 300, 100)
}

func TestHotkeyNamesKeysForThePlatform(t *testing.T) {
	tt := hotkeyTester("mod+shift+k", HotkeyProps{Platform: HotkeyOther})
	for _, s := range []string{"Ctrl", "Shift", "K"} {
		if !tt.HasText(s) {
			t.Errorf("other platform: no %q in %q", s, tt.Texts())
		}
	}
	tt = hotkeyTester("mod+shift+k", HotkeyProps{Platform: HotkeyApple})
	if tt.HasText("Ctrl") || tt.HasText("Shift") || !tt.HasText("K") {
		t.Errorf("apple: texts %q, want glyphs for mod and shift and K", tt.Texts())
	}
	tt = hotkeyTester("enter", HotkeyProps{Platform: HotkeyOther})
	if !tt.HasText("Enter") {
		t.Errorf("enter on other platforms: %q", tt.Texts())
	}
}

func TestHotkeyChipsAndCompact(t *testing.T) {
	tk := theme.Light()
	tt := hotkeyTester("ctrl+k", HotkeyProps{Platform: HotkeyOther})
	ctrl, _ := tt.Find("Ctrl")
	k, _ := tt.Find("K")
	if gap := k.X - (ctrl.X + ctrl.W); gap < 2+16-0.5 {
		t.Errorf("separate chips: K %v DIPs after Ctrl, want 2 between chips plus 8 padding each side", gap)
	}
	if got := testkit.PixelAt(tt, 300, ctrl.X-4, ctrl.Y+ctrl.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("filled chip %+v, want FillTertiary", got)
	}
	tt = hotkeyTester("ctrl+k", HotkeyProps{Platform: HotkeyOther, Compact: true})
	ctrl, _ = tt.Find("Ctrl")
	k, _ = tt.Find("K")
	if gap := k.X - (ctrl.X + ctrl.W); gap < 5.5 || gap > 6.5 {
		t.Errorf("compact: K %v DIPs after Ctrl, want 6 inside one chip", gap)
	}
	if got := testkit.PixelAt(tt, 300, ctrl.X+ctrl.W+3, ctrl.Y+ctrl.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("compact: between keys %+v, want the one chip's fill", got)
	}
}

func TestParseHotkey(t *testing.T) {
	for _, c := range []struct {
		keys  string
		apple bool
		mods  ui.Modifiers
		key   ui.Key
	}{
		{"mod+k", true, ui.Super, ui.KeyK},
		{"mod+k", false, ui.Ctrl, ui.KeyK},
		{"ctrl+shift+p", true, ui.Ctrl | ui.Shift, ui.KeyP},
		{"alt+3", false, ui.Alt, ui.Key3},
		{"escape", false, 0, ui.KeyEscape},
	} {
		mods, key, ok := parseHotkey(c.keys, c.apple)
		if !ok || mods != c.mods || key != c.key {
			t.Errorf("%q apple %v: %v %v %v, want %v %v", c.keys, c.apple, mods, key, ok, c.mods, c.key)
		}
	}
	if _, _, ok := parseHotkey("mod+", false); ok {
		t.Error("mod+ parsed without a key")
	}
}

func TestBrowserOnMacSwapsCommandAndControl(t *testing.T) {
	if got := swapCmd(ui.Super|ui.Shift, true); got != ui.Ctrl|ui.Shift {
		t.Errorf("⌘⇧ in a browser on a Mac: %v, want Ctrl|Shift as mygo's web backend reports it", got)
	}
	if got := swapCmd(ui.Ctrl, true); got != ui.Super {
		t.Errorf("⌃ in a browser on a Mac: %v, want Super", got)
	}
	if got := swapCmd(ui.Super|ui.Alt, false); got != ui.Super|ui.Alt {
		t.Errorf("native: %v changed", got)
	}
}

func TestFormatHotkey(t *testing.T) {
	for _, c := range []struct {
		mods  ui.Modifiers
		key   ui.Key
		apple bool
		want  string
	}{
		{ui.Super | ui.Shift, ui.KeyK, true, "mod+shift+k"},
		{ui.Ctrl, ui.KeyK, false, "mod+k"},
		{ui.Ctrl | ui.Alt, ui.KeyEnter, true, "ctrl+alt+enter"},
		{ui.Super, ui.Key1, false, "meta+1"},
		{0, ui.KeyF5, false, "f5"},
	} {
		if got, ok := formatHotkey(c.mods, c.key, c.apple); !ok || got != c.want {
			t.Errorf("%v %v apple %v: %q %v, want %q", c.mods, c.key, c.apple, got, ok, c.want)
		}
		if got, ok := formatHotkey(c.mods, c.key, c.apple); ok {
			mods, key, _ := parseHotkey(got, c.apple)
			if mods != c.mods || key != c.key {
				t.Errorf("%q parses back to %v %v", got, mods, key)
			}
		}
	}
	if _, ok := formatHotkey(ui.Ctrl, ui.KeyUnknown, false); ok {
		t.Error("a modifier alone formatted")
	}
}
