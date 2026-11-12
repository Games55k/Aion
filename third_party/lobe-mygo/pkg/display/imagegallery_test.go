package display

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var (
	green = ui.RGB(30, 200, 60)
	blue  = ui.RGB(40, 60, 220)
)

func galleryTester() *ui.Tester {
	srcs := []ui.ImageSource{solidOf(400, 200, red), solidOf(400, 200, green), solidOf(400, 200, blue)}
	names := []string{"one", "two", "three"}
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(40).Children(func() {
			ImageGroup(c, func(g *ImageGallery) {
				for i, src := range srcs {
					Image(c.Key(i), src, ImageProps{Size: 80, Gallery: g}).Label(names[i])
				}
			}).Row().Gap(8)
		})
	}, winW, winH)
}

func previewColor(tt *ui.Tester) ui.Color { return testkit.PixelAt(tt, winW, 400, 300) }

func TestImageGroupSwitchesBetweenImages(t *testing.T) {
	tt := galleryTester()
	two, _ := tt.Find("two")
	tt.ClickAt(two.X+40, two.Y+40)
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("2 / 3") || !testkit.Near(previewColor(tt), green) {
		t.Fatalf("opened the second: texts %q, color %+v", tt.Texts(), previewColor(tt))
	}
	clickTitle(t, tt, "Next image")
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("3 / 3") || !testkit.Near(previewColor(tt), blue) {
		t.Errorf("next: texts %q, color %+v", tt.Texts(), previewColor(tt))
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Next image"); ok {
		t.Error("Next shows on the last image")
	}
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyLeft)
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("1 / 3") || !testkit.Near(previewColor(tt), red) {
		t.Errorf("left twice: texts %q, color %+v", tt.Texts(), previewColor(tt))
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Previous image"); ok {
		t.Error("Previous shows on the first image")
	}
	one, _ := tt.Find("one")
	if got := testkit.PixelAt(tt, winW, one.X+40, one.Y+40); testkit.Near(got, testkit.Over(red, tk().BgLayout.Alpha(0.9))) {
		t.Errorf("the shown image's thumbnail stays visible under the backdrop: %+v", got)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(250 * time.Millisecond)
	tt.Frame()
	r, ok := previewBox(t, tt)
	cx := r.X + r.W/2
	if d1, d2 := abs(cx-(one.X+40)), abs(cx-(two.X+40)); !ok || d1 >= d2 {
		t.Errorf("closing from the first image: center %v is %v from its thumbnail and %v from the one opened, want it flying to the first", cx, d1, d2)
	}
}

func TestImagePreviewRotates(t *testing.T) {
	tt := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80})
	openPreview(t, tt)
	clickTitle(t, tt, "Rotate right")
	tt.Advance(time.Second)
	tt.Frame()
	r, _ := previewBox(t, tt)
	if abs(r.W-200) > 1 || abs(r.H-400) > 1 || abs(r.X-300) > 1 || abs(r.Y-100) > 1 {
		t.Errorf("rotated right: %v, want 200×400 centered", r)
	}
	clickTitle(t, tt, "Rotate left")
	tt.Advance(time.Second)
	tt.Frame()
	if r, _ := previewBox(t, tt); abs(r.W-400) > 1 || abs(r.H-200) > 1 {
		t.Errorf("rotated back: %v, want 400×200", r)
	}
}

func TestImagePreviewDownload(t *testing.T) {
	var got []ui.ImageSource
	src := solid(400, 200)
	tt := imageTester(src, ImageProps{Width: 120, Height: 80, OnDownload: func(s ui.ImageSource) { got = append(got, s) }})
	openPreview(t, tt)
	clickTitle(t, tt, "Download")
	if len(got) != 1 || got[0] != ui.ImageSource(src) {
		t.Errorf("downloads %v, want the image once", got)
	}
	plain := imageTester(src, ImageProps{Width: 120, Height: 80})
	openPreview(t, plain)
	if _, ok := plain.AccessByRole(ui.RoleButton, "Download"); ok {
		t.Error("Download shows without OnDownload")
	}
}

func tk() *theme.Tokens { return theme.Light() }
