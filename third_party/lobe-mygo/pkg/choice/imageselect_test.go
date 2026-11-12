package choice

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func solidBitmap(c ui.Color) *ui.Bitmap { return solidSized(c, 8, 8) }

func solidSized(c ui.Color, w, h int) *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{c.R, c.G, c.B, 255})
		}
	}
	return ui.NewBitmap(img)
}

var (
	lightPic = ui.RGB(200, 220, 255)
	darkPic  = ui.RGB(40, 60, 90)
)

func imageSelectTester(v *string, p ImageSelectProps[string]) *ui.Tester {
	if p.Options == nil {
		p.Options = []ImageSelectItem[string]{
			{Value: "light", Label: "Light", Image: solidBitmap(lightPic)},
			{Value: "dark", Label: "Dark", Image: solidBitmap(darkPic)},
		}
	}
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { ImageSelect(c, v, p) })
	}, 500, 200)
}

func TestImageSelectChoosesOnClick(t *testing.T) {
	v := "light"
	tt := imageSelectTester(&v, ImageSelectProps[string]{})
	info, ok := tt.AccessByRole(ui.RoleRadio, "Dark")
	if !ok {
		t.Fatal("no Dark option")
	}
	tt.ClickAt(info.Bounds.X+20, info.Bounds.Y+20)
	tt.Advance(time.Second)
	tt.Frame()
	if v != "dark" {
		t.Fatalf("value %q after clicking Dark", v)
	}
	if info, _ := tt.AccessByRole(ui.RoleRadio, "Dark"); !info.Checked {
		t.Error("Dark not checked")
	}
	tt.Key(0, ui.KeyLeft)
	tt.Advance(time.Second)
	tt.Frame()
	if v != "light" {
		t.Errorf("value %q after Left, want light", v)
	}
}

func TestImageSelectFrameSizeAndPicture(t *testing.T) {
	v := ""
	tt := imageSelectTester(&v, ImageSelectProps[string]{})
	info, _ := tt.AccessByRole(ui.RoleRadio, "Light")
	b := info.Bounds
	if b.W != 144 {
		t.Errorf("option width %v, want 144", b.W)
	}
	if got := testkit.PixelAt(tt, 500, b.X+72, b.Y+43); !testkit.Near(got, lightPic) {
		t.Errorf("picture pixel %+v, want %+v", got, lightPic)
	}
	if got := testkit.PixelAt(tt, 500, b.X+72, b.Y+84); !testkit.Near(got, lightPic) {
		t.Errorf("picture bottom %+v, want it 86 tall", got)
	}
	if got := testkit.PixelAt(tt, 500, b.X+72, b.Y+88); testkit.Near(got, lightPic) {
		t.Error("picture taller than 86")
	}
	if !tt.HasText("Light") || !tt.HasText("Dark") {
		t.Errorf("texts %q", tt.Texts())
	}

	tt = imageSelectTester(&v, ImageSelectProps[string]{Width: 100, Height: 60})
	info, _ = tt.AccessByRole(ui.RoleRadio, "Light")
	if info.Bounds.W != 100 {
		t.Errorf("option width %v, want 100", info.Bounds.W)
	}
	if got := testkit.PixelAt(tt, 500, info.Bounds.X+50, info.Bounds.Y+58); !testkit.Near(got, lightPic) {
		t.Errorf("picture bottom %+v, want it 60 tall", got)
	}
}

func TestImageSelectRingsTheChosen(t *testing.T) {
	v := "dark"
	tt := imageSelectTester(&v, ImageSelectProps[string]{})
	tk := theme.Light()
	page := tk.BgLayout
	chosen, _ := tt.AccessByRole(ui.RoleRadio, "Dark")
	other, _ := tt.AccessByRole(ui.RoleRadio, "Light")
	ringAt := func(b ui.Rect) ui.Color { return testkit.PixelAt(tt, 500, b.X-1, b.Y+43) }
	if got := ringAt(chosen.Bounds); testkit.Near(got, page) {
		t.Errorf("chosen ring %+v, want a ring outside the picture", got)
	}
	if got := ringAt(other.Bounds); !testkit.Near(got, page) {
		t.Errorf("other ring %+v, want none", got)
	}
}

func TestImageSelectLabelColors(t *testing.T) {
	v := "dark"
	tt := imageSelectTester(&v, ImageSelectProps[string]{})
	tk := theme.Light()
	label := func(name string) ui.Color {
		info, _ := tt.AccessByRole(ui.RoleRadio, name)
		b := info.Bounds
		return testkit.Ink(tt, 500, ui.Rect{X: b.X, Y: b.Y + 90, W: b.W, H: b.H - 90}, tk.BgLayout)
	}
	if got := label("Dark"); !testkit.Near(got, tk.Text) {
		t.Errorf("chosen label %+v, want text color %+v", got, tk.Text)
	}
	if got := label("Light"); !testkit.Near(got, tk.TextDescription) {
		t.Errorf("other label %+v, want description color %+v", got, tk.TextDescription)
	}
}

func TestImageSelectWidePictureCoversItsFrame(t *testing.T) {
	v := ""
	tt := imageSelectTester(&v, ImageSelectProps[string]{Options: []ImageSelectItem[string]{
		{Value: "wide", Label: "Wide", Image: solidSized(lightPic, 60, 8)},
	}})
	info, _ := tt.AccessByRole(ui.RoleRadio, "Wide")
	b := info.Bounds
	for _, y := range []float32{2, 43, 84} {
		if got := testkit.PixelAt(tt, 500, b.X+72, b.Y+y); !testkit.Near(got, lightPic) {
			t.Errorf("y=%v: %+v, want the picture covering the frame", y, got)
		}
	}
}

func TestImageSelectHoverRingsInText(t *testing.T) {
	v := ""
	tt := imageSelectTester(&v, ImageSelectProps[string]{})
	info, _ := tt.AccessByRole(ui.RoleRadio, "Light")
	b := info.Bounds
	tt.Move(b.X+72, b.Y+43)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.PixelAt(tt, 500, b.X-1, b.Y+43); !testkit.Near(got, theme.Light().Text) {
		t.Errorf("hover ring %+v, want the text color", got)
	}
}
