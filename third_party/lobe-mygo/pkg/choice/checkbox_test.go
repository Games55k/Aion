package choice

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func checkboxBox(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleCheckBox, "Agree")
	if !ok {
		t.Fatalf("no checkbox named Agree; texts %q", tt.Texts())
	}
	return info.Bounds
}

func TestCheckboxTogglesAndFills(t *testing.T) {
	tk := theme.Light()
	checked := false
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Checkbox(c, &checked, "Agree", CheckboxProps{})
	}), 200, 100)
	label, ok := tt.Find("Agree")
	if !ok {
		t.Fatal("label not shown")
	}
	b := checkboxBox(t, tt)
	glyph := ui.Rect{X: b.X, Y: b.Y + (b.H-16)/2, W: 16, H: 16}
	if gap := label.X - (glyph.X + 16); gap != 8 {
		t.Errorf("label starts %v DIPs after the box, want 8", gap)
	}
	if got := testkit.PixelAt(tt, 200, glyph.X+3, glyph.Y+3); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("unchecked fill = %+v, want BgContainer", got)
	}
	if err := tt.Click("Agree"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !checked {
		t.Fatal("clicking the label did not check the box")
	}
	if got := testkit.PixelAt(tt, 200, glyph.X+2, glyph.Y+2); !testkit.Near(got, tk.Primary) {
		t.Errorf("checked fill = %+v, want Primary %+v", got, tk.Primary)
	}
}

func TestCheckboxIndeterminateFills(t *testing.T) {
	tk := theme.Light()
	checked := false
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Checkbox(c, &checked, "Agree", CheckboxProps{Indeterminate: true})
	}), 200, 100)
	b := checkboxBox(t, tt)
	glyph := ui.Rect{X: b.X, Y: b.Y + (b.H-16)/2, W: 16, H: 16}
	if got := testkit.PixelAt(tt, 200, glyph.X+2, glyph.Y+2); !testkit.Near(got, tk.Primary) {
		t.Errorf("indeterminate fill = %+v, want Primary", got)
	}
	if info, _ := tt.AccessByRole(ui.RoleCheckBox, "Agree"); !info.Mixed {
		t.Error("indeterminate checkbox does not report mixed to assistive technology")
	}
}
