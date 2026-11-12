package code

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func qrTester(value string, p QRCodeProps, extra func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).AlignItems(ui.Start).Children(func() {
			QRCode(c, value, p).Label("qr")
			if extra != nil {
				extra(c)
			}
		})
	}, 400, 300)
}

func scan(t *testing.T, tt *ui.Tester) string {
	t.Helper()
	bmp, err := gozxing.NewBinaryBitmapFromImage(tt.Image())
	if err != nil {
		t.Fatal(err)
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("not a readable QR code: %v", err)
	}
	return res.GetText()
}

func TestQRCodeScans(t *testing.T) {
	for _, v := range []string{"https://lobehub.com", "LobeHub 你好", strings.Repeat("lobe ", 30)} {
		tt := qrTester(v, QRCodeProps{}, nil)
		if got := scan(t, tt); got != v {
			t.Errorf("scanned %q, want %q", got, v)
		}
	}
}

func TestQRCodeScansWithIcon(t *testing.T) {
	mark := ui.RGB(230, 30, 30)
	tt := qrTester("https://lobehub.com", QRCodeProps{Icon: func(c *ui.Context) {
		ui.Box(c).Fill().Background(mark)
	}}, nil)
	if got := scan(t, tt); got != "https://lobehub.com" {
		t.Errorf("scanned %q", got)
	}
	info, _ := tt.Access("qr")
	b := info.Bounds
	if got := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+b.H/2); !testkit.Near(got, mark) {
		t.Errorf("center %+v, want the icon", got)
	}
}

func TestQRCodeSizeColorsAndBorder(t *testing.T) {
	tt := qrTester("hi", QRCodeProps{}, nil)
	info, _ := tt.Access("qr")
	b := info.Bounds
	if b.W != 192 || b.H != 192 {
		t.Errorf("bounds %+v, want 160 plus 16 of padding around", b)
	}
	white := ui.RGB(255, 255, 255)
	if got := testkit.PixelAt(tt, 400, b.X+8, b.Y+b.H/2); !testkit.Near(got, white) {
		t.Errorf("padding %+v, want white", got)
	}
	if got := testkit.PixelAt(tt, 400, b.X+17, b.Y+17); !testkit.Near(got, ui.RGB(0, 0, 0)) {
		t.Errorf("finder corner %+v, want black", got)
	}
	border := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+0.5)
	if testkit.Near(border, white) {
		t.Error("no border")
	}

	ink, paper := ui.RGB(20, 60, 160), ui.RGB(250, 240, 210)
	tt = qrTester("hi", QRCodeProps{Size: 100, Color: ink, Background: paper, NoBorder: true}, nil)
	info, _ = tt.Access("qr")
	b = info.Bounds
	if b.W != 132 {
		t.Errorf("width %v, want 132", b.W)
	}
	if got := testkit.PixelAt(tt, 400, b.X+17, b.Y+17); !testkit.Near(got, ink) {
		t.Errorf("module %+v, want the custom color", got)
	}
	if got := testkit.PixelAt(tt, 400, b.X+8, b.Y+b.H/2); !testkit.Near(got, paper) {
		t.Errorf("padding %+v, want the custom background", got)
	}
	if got := testkit.PixelAt(tt, 400, b.X+b.W/2, b.Y+0.5); !testkit.Near(got, paper) {
		t.Errorf("edge %+v with NoBorder, want the background", got)
	}
}

func TestQRCodeErrorLevels(t *testing.T) {
	v := strings.Repeat("lobehub ", 8)
	low, high := len(qrModules(v, QRLevelL, false)), len(qrModules(v, QRLevelH, false))
	if high <= low {
		t.Errorf("H takes %d modules a side, L %d: want H larger", high, low)
	}
	if withIcon := len(qrModules(v, QRLevelL, true)); withIcon != high {
		t.Errorf("an icon takes %d modules, want H's %d", withIcon, high)
	}
}

func TestQRCodeStatuses(t *testing.T) {
	tt := qrTester("hi", QRCodeProps{Status: QRLoading}, nil)
	spin, ok := tt.AccessByRole(ui.RoleStatus, "Loading")
	if !ok {
		t.Error("no loading status")
	}
	card, _ := tt.Access("qr")
	if mid := spin.Bounds.X + spin.Bounds.W/2; mid < card.Bounds.X+card.Bounds.W/2-2 || mid > card.Bounds.X+card.Bounds.W/2+2 {
		t.Errorf("spinner centered at x %v, card at %v", mid, card.Bounds.X+card.Bounds.W/2)
	}
	refreshed := false
	tt = qrTester("hi", QRCodeProps{Status: QRExpired, OnRefresh: func() { refreshed = true }}, nil)
	if !tt.HasText("Expired") {
		t.Errorf("texts %q", tt.Texts())
	}
	btn, ok := tt.AccessByRole(ui.RoleButton, "Refresh")
	if !ok {
		t.Fatal("no Refresh button")
	}
	tt.ClickAt(btn.Bounds.X+btn.Bounds.W/2, btn.Bounds.Y+btn.Bounds.H/2)
	tt.Frame()
	if !refreshed {
		t.Error("OnRefresh not called")
	}
	tt = qrTester("hi", QRCodeProps{Status: QRExpired}, nil)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Refresh"); ok {
		t.Error("a Refresh button without OnRefresh")
	}
}

func TestQRCodeNamesItsValue(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		QRCode(c, "https://lobehub.com", QRCodeProps{})
	}, 300, 300)
	if _, ok := tt.AccessByRole(ui.RoleImage, "https://lobehub.com"); !ok {
		t.Error("no image named by the value")
	}
}

func TestQRCodeTooLongShowsNothing(t *testing.T) {
	tt := qrTester(strings.Repeat("x", 5000), QRCodeProps{}, nil)
	info, ok := tt.Access("qr")
	if !ok || info.Bounds.W != 192 {
		t.Errorf("bounds %+v, want the empty frame", info.Bounds)
	}
}

func TestQRCodeLeavesTheIconHoleEmpty(t *testing.T) {
	v := "https://lobehub.com/some/longer/path"
	tt := qrTester(v, QRCodeProps{Icon: func(c *ui.Context) {}}, nil)
	info, _ := tt.Access("qr")
	b := info.Bounds
	n := len(qrModules(v, QRLevelH, true))
	hole := int(float32(n)*qrIconRatio+0.999) | 1
	cx, cy, r := b.X+b.W/2, b.Y+b.H/2, float32(hole)*160/float32(n)/2-1
	for y := cy - r; y < cy+r; y++ {
		for x := cx - r; x < cx+r; x++ {
			if testkit.Near(testkit.PixelAt(tt, 400, x, y), ui.RGB(0, 0, 0)) {
				t.Fatalf("a module shows under the icon at %v,%v", x-b.X, y-b.Y)
			}
		}
	}
}

func TestQRCodeFollowsItsValue(t *testing.T) {
	v := "first"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { QRCode(c, v, QRCodeProps{}) })
	}, 400, 300)
	v = "second"
	tt.Frame()
	if got := scan(t, tt); got != "second" {
		t.Errorf("scanned %q after the value changed", got)
	}
}

func TestQRPathCutsTheHole(t *testing.T) {
	grid := make([][]bool, 9)
	for y := range grid {
		grid[y] = []bool{true, true, true, true, true, true, true, true, true}
	}
	path := qrPath(grid, 3)
	for _, p := range [][2]float32{{4.5, 4.5}, {3.5, 3.5}, {5.5, 5.5}} {
		if path.Contains(p[0], p[1]) {
			t.Errorf("module %v in the hole is drawn", p)
		}
	}
	for _, p := range [][2]float32{{2.5, 4.5}, {6.5, 4.5}, {4.5, 2.5}, {0.5, 0.5}} {
		if !path.Contains(p[0], p[1]) {
			t.Errorf("module %v outside the hole is missing", p)
		}
	}
}

func TestQRCodeModulesAreCrisp(t *testing.T) {
	tt := qrTester("https://lobehub.com", QRCodeProps{Size: 150}, nil)
	info, _ := tt.Access("qr")
	b := info.Bounds
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / 400
	soft := 0
	for y := int((b.Y + 16) * scale); y < int((b.Y+16+150)*scale); y++ {
		for x := int((b.X + 16) * scale); x < int((b.X+16+150)*scale); x++ {
			if v := img.RGBAAt(x, y).R; v > 24 && v < 232 {
				soft++
			}
		}
	}
	if soft > 0 {
		t.Errorf("%d gray pixels: want modules on whole pixels", soft)
	}
}
