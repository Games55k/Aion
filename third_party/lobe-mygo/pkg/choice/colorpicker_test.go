package choice

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func pickerTester(v *ui.Color, p ColorPickerProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			ColorPicker(c, v, p).Label("Accent")
		})
	}, 400, 500)
}

func role(t *testing.T, tt *ui.Tester, r ui.Role, name string) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(r, name)
	if !ok {
		t.Fatalf("no %v %q", r, name)
	}
	return info.Bounds
}

func openPicker(t *testing.T, tt *ui.Tester) {
	t.Helper()
	b := role(t, tt, ui.RoleButton, "Accent")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
}

func TestColorPickerSwatchOpensPanel(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	tt := pickerTester(&v, ColorPickerProps{})
	b := role(t, tt, ui.RoleButton, "Accent")
	if b.W != 28 || b.H != 28 {
		t.Errorf("swatch trigger %vx%v, want 28", b.W, b.H)
	}
	if got := testkit.PixelAt(tt, 400, b.X+14, b.Y+14); !testkit.Near(got, v) {
		t.Errorf("trigger shows %+v, want the color", got)
	}
	openPicker(t, tt)
	if info, ok := tt.AccessByRole(ui.RoleTextField, "Hex"); !ok || info.Value != "#1677FF" {
		t.Errorf("hex field %+v, want #1677FF", info)
	}
	area := role(t, tt, ui.RoleSlider, "Saturation and brightness")
	if area.W != 240 || area.H != 150 {
		t.Errorf("saturation area %vx%v, want 240x150", area.W, area.H)
	}
}

func TestColorPickerSaturationAndHue(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	tt := pickerTester(&v, ColorPickerProps{})
	openPicker(t, tt)
	area := role(t, tt, ui.RoleSlider, "Saturation and brightness")
	tt.ClickAt(area.X+area.W-1, area.Y+1)
	tt.Frame()
	if v.B < 250 || v.R > 3 {
		t.Errorf("top right of a blue hue: %+v, want the pure hue", v)
	}
	tt.ClickAt(area.X+1, area.Y+area.H-1)
	tt.Frame()
	if v.R > 3 || v.G > 3 || v.B > 3 {
		t.Errorf("bottom left: %+v, want black", v)
	}
	tt.ClickAt(area.X+area.W-1, area.Y+1)
	hue := role(t, tt, ui.RoleSlider, "Hue")
	tt.ClickAt(hue.X+7+(hue.W-14)/3, hue.Y+hue.H/2)
	tt.Frame()
	if v.G < 250 || v.R > 10 || v.B > 10 {
		t.Errorf("hue at a third: %+v, want green", v)
	}
}

func TestColorPickerHexPresetsAndRestore(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	red := ui.RGB(245, 34, 45)
	tt := pickerTester(&v, ColorPickerProps{Presets: []ui.Color{red, ui.RGB(82, 196, 26)}})
	openPicker(t, tt)
	hex := role(t, tt, ui.RoleTextField, "Hex")
	tt.ClickAt(hex.X+hex.W/2, hex.Y+hex.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("#00ff00")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v != ui.RGB(0, 255, 0) {
		t.Errorf("hex #00ff00 → %+v", v)
	}
	p := role(t, tt, ui.RoleButton, "#F5222D")
	tt.ClickAt(p.X+p.W/2, p.Y+p.H/2)
	tt.Frame()
	if v != red {
		t.Errorf("preset → %+v, want red", v)
	}
	r := role(t, tt, ui.RoleButton, "Restore")
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if v != ui.RGB(22, 119, 255) {
		t.Errorf("restore → %+v, want the color the picker opened with", v)
	}
}

func TestColorPickerAlpha(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	tt := pickerTester(&v, ColorPickerProps{Alpha: true})
	openPicker(t, tt)
	a := role(t, tt, ui.RoleSlider, "Alpha")
	tt.ClickAt(a.X+7+(a.W-14)/2, a.Y+a.H/2)
	tt.Frame()
	if v.A < 120 || v.A > 135 || v.B != 255 {
		t.Errorf("alpha at half: %+v", v)
	}
	if info, ok := tt.AccessByRole(ui.RoleTextField, "Alpha percent"); !ok || info.Value != "50" {
		t.Errorf("alpha field %+v, want 50", info)
	}
}

func TestColorSwatchesCustomPicker(t *testing.T) {
	red := ui.RGB(245, 34, 45)
	v := red
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			ColorSwatches(c, &v, ColorSwatchesProps{Colors: []ColorSwatch{{Color: red, Title: "Red"}}, EnableColorPicker: true})
		})
	}, 400, 500)
	b := role(t, tt, ui.RoleButton, "Custom color")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	hex := role(t, tt, ui.RoleTextField, "Hex")
	tt.ClickAt(hex.X+hex.W/2, hex.Y+hex.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("#123456")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v != ui.RGB(0x12, 0x34, 0x56) {
		t.Fatalf("custom color → %+v", v)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	tt.Frame()
	chip, ok := tt.AccessByRole(ui.RoleButton, "#123456")
	if !ok {
		t.Fatal("no chip for the custom color")
	}
	tt.Move(0, 0)
	tt.Advance(time.Second)
	tt.Frame()
	ink := 0
	cb := chip.Bounds
	for y := cb.Y + 6; y < cb.Y+cb.H-6; y++ {
		for x := cb.X + 6; x < cb.X+cb.W-6; x++ {
			if !testkit.Near(testkit.PixelAt(tt, 400, x, y), v) {
				ink++
			}
		}
	}
	if ink == 0 {
		t.Error("custom chip shows no check")
	}
}

func TestColorPickerPresetsFitNinePerRow(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	var colors []ui.Color
	for i := range 9 {
		colors = append(colors, ui.RGB(uint8(i*20), 0, 0))
	}
	tt := pickerTester(&v, ColorPickerProps{Presets: colors})
	openPicker(t, tt)
	first := role(t, tt, ui.RoleButton, formatHex(colors[0]))
	last := role(t, tt, ui.RoleButton, formatHex(colors[8]))
	if last.Y != first.Y {
		t.Errorf("9th preset at y %v, first at %v: want one row of nine", last.Y, first.Y)
	}
}

func TestColorPickerCompareShowsBothColors(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	tt := pickerTester(&v, ColorPickerProps{})
	openPicker(t, tt)
	area := role(t, tt, ui.RoleSlider, "Saturation and brightness")
	tt.ClickAt(area.X+1, area.Y+area.H-1)
	tt.Frame()
	r := role(t, tt, ui.RoleButton, "Restore")
	if got := testkit.PixelAt(tt, 400, r.X+r.W/2, r.Y+r.H/2); !testkit.Near(got, ui.RGB(22, 119, 255)) {
		t.Errorf("restore half %+v, want the color the picker opened with", got)
	}
	if got := testkit.PixelAt(tt, 400, r.X+r.W/2, r.Y+r.H+r.H/2); !testkit.Near(got, v) {
		t.Errorf("current half %+v, want %+v", got, v)
	}
}

func TestColorPickerThumbShowsPastTheEdge(t *testing.T) {
	v := ui.RGB(22, 119, 255)
	tt := pickerTester(&v, ColorPickerProps{})
	tt.SetDark(true)
	tt.Frame()
	openPicker(t, tt)
	area := role(t, tt, ui.RoleSlider, "Saturation and brightness")
	tt.ClickAt(area.X+area.W/2, area.Y)
	tt.Frame()
	if got := testkit.PixelAt(tt, 400, area.X+area.W/2, area.Y-6); got.R < 240 || got.G < 240 || got.B < 240 {
		t.Errorf("above the top edge, over the thumb's white ring: %+v", got)
	}
}
