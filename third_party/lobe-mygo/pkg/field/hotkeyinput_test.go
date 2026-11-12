package field

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func hotkeyInputTester(v *string, p HotkeyInputProps) *ui.Tester {
	p.Platform = display.HotkeyOther
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(320).Children(func() {
			HotkeyInput(c, v, p).Label("shortcut")
			ui.Button(c, "Elsewhere")
		})
	}, 400, 200)
}

func TestHotkeyInputRecords(t *testing.T) {
	v := ""
	tt := hotkeyInputTester(&v, HotkeyInputProps{})
	if !tt.HasText("Press keys to record shortcut") {
		t.Fatalf("texts %q, want the placeholder", tt.Texts())
	}
	tt.Click("shortcut")
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyP)
	tt.Frame()
	if v != "mod+shift+p" {
		t.Fatalf("value %q after Ctrl+Shift+P, want mod+shift+p", v)
	}
	if !tt.HasText("Shift") || !tt.HasText("P") {
		t.Errorf("texts %q, want the keys as caps", tt.Texts())
	}
}

func TestHotkeyInputRejectsInvalidAndConflicts(t *testing.T) {
	v := "mod+k"
	var conflicted []string
	tt := hotkeyInputTester(&v, HotkeyInputProps{Conflicts: []string{"mod+s"}, OnConflict: func(k string) { conflicted = append(conflicted, k) }})
	tt.Click("shortcut")
	tt.Key(0, ui.KeyJ)
	tt.Frame()
	if v != "mod+k" || !tt.HasText("Shortcut must include a modifier key (Ctrl, Alt, Shift) and only one regular key.") {
		t.Errorf("J alone: value %q, texts %q", v, tt.Texts())
	}
	tt.Key(ui.Ctrl, ui.KeyS)
	tt.Frame()
	if v != "mod+k" || len(conflicted) != 1 || !tt.HasText("This shortcut conflicts with an existing one.") {
		t.Errorf("Ctrl+S: value %q conflicted %q texts %q", v, conflicted, tt.Texts())
	}
}

func TestHotkeyInputResetAndClear(t *testing.T) {
	v := "mod+j"
	tt := hotkeyInputTester(&v, HotkeyInputProps{ResetValue: "mod+k", AllowClear: true})
	clickRole(t, tt, ui.RoleButton, "Reset to default")
	if v != "mod+k" {
		t.Fatalf("value %q after reset", v)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Reset to default"); ok {
		t.Error("reset shows while the value is the default")
	}
	clickRole(t, tt, ui.RoleButton, "Clear binding")
	if v != "" {
		t.Errorf("value %q after clear", v)
	}
}

func TestHotkeyInputAlignsToTheStart(t *testing.T) {
	v := ""
	tt := hotkeyInputTester(&v, HotkeyInputProps{})
	if r, _ := tt.Find("Press keys to record shortcut"); r.X > 20+12+1 {
		t.Errorf("placeholder at x %v, want it at the start, after the 12 DIP padding", r.X)
	}
}
