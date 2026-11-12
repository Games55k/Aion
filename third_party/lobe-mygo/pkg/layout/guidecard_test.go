package layout

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type guideApp struct {
	p      GuideCardProps
	closed int
}

func guideTester(a *guideApp) *ui.Tester {
	a.p.OnClose = func() { a.closed++ }
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).AlignItems(ui.Start).Children(func() {
			GuideCard(c, a.p).Label("guide").Width(280)
			ui.Text(c, "After")
		})
	}, 400, 400)
}

func coverBitmap(col ui.Color) *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.Set(x, y, color.RGBA{col.R, col.G, col.B, 255})
		}
	}
	return ui.NewBitmap(img)
}

func TestGuideCardShowsCoverTitleAndDesc(t *testing.T) {
	cover := ui.RGB(40, 120, 220)
	a := &guideApp{p: GuideCardProps{Title: "Welcome", Desc: "Start a chat", Cover: coverBitmap(cover), CoverHeight: 100}}
	tt := guideTester(a)
	info, _ := tt.Access("guide")
	b := info.Bounds
	if got := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+50); !testkit.Near(got, cover) {
		t.Errorf("cover %+v", got)
	}
	title, ok := tt.Find("Welcome")
	if !ok || title.Y < b.Y+100+16-1 || title.X != b.X+16 {
		t.Errorf("title at %+v, want 16 in under the cover", title)
	}
	if desc, _ := tt.Find("Start a chat"); desc.Y < title.Y+title.H+8-1 {
		t.Errorf("desc at %+v, want 8 under the title", desc)
	}
}

func TestGuideCardFilledGradient(t *testing.T) {
	tk := theme.Light()
	a := &guideApp{p: GuideCardProps{Title: "Welcome", Desc: "Start a chat", NoClose: true}}
	tt := guideTester(a)
	info, _ := tt.Access("guide")
	b := info.Bounds
	top := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+1)
	bottom := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+b.H-1)
	if !testkit.Near(top, testkit.Over(tk.BgLayout, tk.FillQuaternary)) {
		t.Errorf("top %+v, want FillQuaternary", top)
	}
	if !testkit.Near(bottom, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("bottom %+v, want FillTertiary", bottom)
	}
}

func TestGuideCardCloses(t *testing.T) {
	a := &guideApp{p: GuideCardProps{Title: "Welcome"}}
	tt := guideTester(a)
	after, _ := tt.Find("After")
	btn, ok := tt.AccessByRole(ui.RoleButton, "Close")
	if !ok {
		t.Fatal("no Close button")
	}
	tt.ClickAt(btn.Bounds.X+btn.Bounds.W/2, btn.Bounds.Y+btn.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if a.closed != 1 {
		t.Errorf("OnClose called %d times", a.closed)
	}
	if _, ok := tt.Find("Welcome"); ok {
		t.Error("the card still shows")
	}
	if moved, _ := tt.Find("After"); moved.Y >= after.Y {
		t.Errorf("what follows stays at %v: want the card's room given back", moved.Y)
	}

	a = &guideApp{p: GuideCardProps{Title: "Welcome", NoClose: true}}
	tt = guideTester(a)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Close"); ok {
		t.Error("a Close button with NoClose")
	}
}

func TestGuideCardVariants(t *testing.T) {
	tk := theme.Light()
	a := &guideApp{p: GuideCardProps{Title: "Welcome", NoClose: true, Variant: BlockOutlined}}
	tt := guideTester(a)
	info, _ := tt.Access("guide")
	b := info.Bounds
	if got := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+0.5); testkit.Near(got, tk.BgLayout) {
		t.Error("outlined card has no border")
	}
	a = &guideApp{p: GuideCardProps{Title: "Welcome", NoClose: true, Variant: BlockBorderless}}
	tt = guideTester(a)
	info, _ = tt.Access("guide")
	b = info.Bounds
	if got := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+b.H-1); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("borderless card %+v, want the page", got)
	}
}

func TestGuideCardDarkGradientFlips(t *testing.T) {
	tk := theme.Dark()
	a := &guideApp{p: GuideCardProps{Title: "Welcome", Desc: "Start a chat", NoClose: true}}
	tt := guideTester(a)
	tt.SetDark(true)
	tt.Frame()
	info, _ := tt.Access("guide")
	b := info.Bounds
	if got := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+1); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("dark top %+v, want FillTertiary", got)
	}
}

func TestGuideCardLeavesNothingOnceClosed(t *testing.T) {
	a := &guideApp{p: GuideCardProps{Title: "Welcome"}}
	tt := guideTester(a)
	btn, _ := tt.AccessByRole(ui.RoleButton, "Close")
	tt.ClickAt(btn.Bounds.X+btn.Bounds.W/2, btn.Bounds.Y+btn.Bounds.H/2)
	tt.Frame()
	if _, ok := tt.Access("guide"); ok {
		t.Error("the closed card is still in the accessibility tree")
	}
}
