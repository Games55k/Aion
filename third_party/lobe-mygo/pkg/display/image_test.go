package display

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var red = ui.RGB(230, 30, 30)

func solid(w, h int) *ui.Bitmap { return solidOf(w, h, red) }

func solidOf(w, h int, c ui.Color) *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{c.R, c.G, c.B, 255})
		}
	}
	return ui.NewBitmap(img)
}

const winW, winH = 800, 600

func imageTester(src ui.ImageSource, p ImageProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(40).Children(func() {
			Image(c, src, p).Label("thumb")
		})
	}, winW, winH)
}

func previewBox(t *testing.T, tt *ui.Tester) (ui.Rect, bool) {
	t.Helper()
	info, ok := tt.Access("Image preview")
	return info.Bounds, ok
}

func openPreview(t *testing.T, tt *ui.Tester) {
	t.Helper()
	thumb, _ := tt.Find("thumb")
	tt.ClickAt(thumb.X+thumb.W/2, thumb.Y+thumb.H/2)
	tt.Advance(time.Second)
	tt.Frame()
}

func TestImageThumbnail(t *testing.T) {
	tk := theme.Light()
	tt := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80})
	thumb, _ := tt.Find("thumb")
	if thumb.W != 120 || thumb.H != 80 {
		t.Errorf("thumbnail %v, want 120×80", thumb)
	}
	if got := testkit.PixelAt(tt, winW, thumb.X+thumb.W/2, thumb.Y+thumb.H/2); !testkit.Near(got, red) {
		t.Errorf("center %+v, want the image", got)
	}
	if got := testkit.PixelAt(tt, winW, thumb.X+60, thumb.Y+4); !testkit.Near(got, red) {
		t.Errorf("top %+v: want the 2:1 image to cover the 3:2 box", got)
	}
	loading := imageTester(nil, ImageProps{Width: 120, Height: 80, Loading: true})
	box, _ := loading.Find("thumb")
	if got := testkit.PixelAt(loading, winW, box.X+60, box.Y+40); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillContent)) {
		t.Errorf("loading %+v, want a skeleton block", got)
	}
	failed := imageTester(nil, ImageProps{Width: 120, Height: 80, Failed: true})
	box, _ = failed.Find("thumb")
	if got := testkit.PixelAt(failed, winW, box.X+6, box.Y+6); !testkit.Near(got, ui.RGB(0xEC, 0xEC, 0xEC)) {
		t.Errorf("failed %+v, want the light fallback fill", got)
	}
}

func TestImagePreviewOpensFromTheThumbnail(t *testing.T) {
	tt := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80})
	thumb, _ := tt.Find("thumb")
	tt.ClickAt(thumb.X+60, thumb.Y+40)
	tt.Advance(60 * time.Millisecond)
	tt.Frame()
	mid, ok := previewBox(t, tt)
	if !ok {
		t.Fatalf("no preview after a click; texts %q", tt.Texts())
	}
	if mid.W <= 121 || mid.W >= 399 {
		t.Errorf("60 ms in the preview is %v wide: want it growing from the thumbnail", mid.W)
	}
	tt.Advance(time.Second)
	tt.Frame()
	fit, _ := previewBox(t, tt)
	want := ui.Rect{X: 200, Y: 200, W: 400, H: 200}
	if abs(fit.X-want.X) > 1 || abs(fit.Y-want.Y) > 1 || abs(fit.W-want.W) > 1 || abs(fit.H-want.H) > 1 {
		t.Errorf("settled at %v, want %v: centered, at most natural size", fit, want)
	}
	if !tt.HasText("100%") {
		t.Errorf("texts %q, want the zoom at 100%%", tt.Texts())
	}
	tk := theme.Light()
	behind := testkit.Over(testkit.Over(tk.BgLayout, tk.FillTertiary), tk.BgLayout.Alpha(0.9))
	if got := testkit.PixelAt(tt, winW, thumb.X+60, thumb.Y+40); !testkit.Near(got, behind) {
		t.Errorf("thumbnail under the backdrop %+v, want it emptied while the preview flies out of it", got)
	}
}

func TestImagePreviewFitsLargeImagesInsideTheMargin(t *testing.T) {
	tt := imageTester(solid(1600, 800), ImageProps{Width: 120, Height: 80})
	openPreview(t, tt)
	fit, _ := previewBox(t, tt)
	if abs(fit.W-752) > 1 || abs(fit.X-24) > 1 {
		t.Errorf("fit %v, want 752 wide, 24 DIPs from the sides", fit)
	}
	if !tt.HasText("47%") {
		t.Errorf("texts %q, want 752/1600 as 47%%", tt.Texts())
	}
}

func TestImagePreviewZooms(t *testing.T) {
	tt := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80})
	openPreview(t, tt)
	tt.Pinch(350, 250, 3)
	tt.Frame()
	z, _ := previewBox(t, tt)
	if abs(z.W-1200) > 1 || abs(z.X-(-100)) > 1 || abs(z.Y) > 1 {
		t.Errorf("pinched ×3 at (350,250): %v, want 1200 wide at x -100 with the point kept, centered down as it fits the height", z)
	}
	if !tt.HasText("300%") {
		t.Errorf("texts %q, want 300%%", tt.Texts())
	}
	drag := func(dx float32) {
		tt.Press(400, 300)
		tt.Move(400+dx/2, 300)
		tt.Move(400+dx, 300)
		tt.Release(400+dx, 300)
		tt.Frame()
	}
	drag(80)
	if p, _ := previewBox(t, tt); abs(p.X-(-20)) > 1 {
		t.Errorf("dragged 80 right from -100: at %v", p.X)
	}
	drag(300)
	if p, _ := previewBox(t, tt); abs(p.X) > 1 {
		t.Errorf("dragged past the edge: at %v, want the left edge held at 0", p.X)
	}
	if _, ok := previewBox(t, tt); !ok {
		t.Fatal("a drag closed the preview")
	}
	clickTitle(t, tt, "Zoom out")
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("200%") {
		t.Errorf("zoom out from 300%%: %q, want 200%%", tt.Texts())
	}
	clickTitle(t, tt, "Zoom in")
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("300%") {
		t.Errorf("zoom in: %q, want 300%%", tt.Texts())
	}
}

func TestImagePreviewWheelAndDoubleClick(t *testing.T) {
	tt := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80})
	openPreview(t, tt)
	tt.Scroll(400, 300, 0, -347)
	tt.Frame()
	if !tt.HasText("200%") {
		t.Errorf("scrolled up 347: %q, want exp(347×0.002) ≈ 2×", tt.Texts())
	}
	tt.Scroll(400, 300, 0, 2000)
	tt.Frame()
	if !tt.HasText("100%") {
		t.Errorf("scrolled far down: %q, want the zoom floor at 100%%", tt.Texts())
	}
	tt.ClickAt(400, 300)
	tt.ClickAt(400, 300)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := previewBox(t, tt); !ok || !tt.HasText("200%") {
		t.Errorf("double-click: open %v texts %q, want zoomed to 2×", ok, tt.Texts())
	}
}

func TestImagePreviewCloses(t *testing.T) {
	tt := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80})
	openPreview(t, tt)
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := previewBox(t, tt); ok {
		t.Error("Escape left the preview open")
	}
	openPreview(t, tt)
	tt.ClickAt(30, 30)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := previewBox(t, tt); ok {
		t.Error("clicking the backdrop left the preview open")
	}
	openPreview(t, tt)
	tt.ClickAt(400, 300)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := previewBox(t, tt); ok {
		t.Error("a single click on the unzoomed image left the preview open")
	}
	none := imageTester(solid(400, 200), ImageProps{Width: 120, Height: 80, NoPreview: true})
	openPreview(t, none)
	if _, ok := previewBox(t, none); ok {
		t.Error("NoPreview opened a preview")
	}
}

func clickTitle(t *testing.T, tt *ui.Tester, title string) {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleButton, title)
	if !ok {
		var names []string
		for _, a := range tt.AccessAll() {
			names = append(names, a.Name)
		}
		t.Fatalf("no %q button among %s", title, strings.Join(names, ", "))
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
