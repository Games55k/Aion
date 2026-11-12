package display

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"

	"github.com/egoist/mygo/ui"
)

type tagHarness struct {
	t      *testing.T
	tt     *ui.Tester
	closed int
}

func newTagHarness(t *testing.T, p TagProps) *tagHarness {
	h := &tagHarness{t: t}
	if p.Closable {
		p.OnClose = func() { h.closed++ }
	}
	h.tt = ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Center().Children(func() {
			Tag(c, "Beta", p).Label("TagBox")
		})
	}, 200, 120)
	return h
}

func (h *tagHarness) bounds() ui.Rect {
	h.t.Helper()
	b, ok := h.tt.Find("TagBox")
	if !ok {
		h.t.Fatalf("tag not found; texts %q", h.tt.Texts())
	}
	return b
}

func (h *tagHarness) pixel(x, y float32) ui.Color {
	img := h.tt.Image()
	scale := float32(img.Bounds().Dx()) / 200
	px := img.RGBAAt(int(x*scale), int(y*scale))
	return ui.RGB(px.R, px.G, px.B)
}

func (h *tagHarness) surface() ui.Color {
	b := h.bounds()
	return h.pixel(b.X+2.5, b.Y+b.H/2)
}

func TestTagHeights(t *testing.T) {
	for _, c := range []struct {
		size theme.Size
		want float32
	}{{theme.Small, 20}, {theme.Middle, 22}, {theme.Large, 28}} {
		if got := newTagHarness(t, TagProps{Size: c.size}).bounds().H; got != c.want {
			t.Errorf("size %v height = %v, want %v", c.size, got, c.want)
		}
	}
}

func TestTagSurfaces(t *testing.T) {
	tk := theme.Light()
	cases := []struct {
		name string
		p    TagProps
		want ui.Color
	}{
		{"default filled", TagProps{}, testkit.Over(tk.BgLayout, tk.FillTertiary)},
		{"red filled", TagProps{Color: TagRed}, testkit.Over(tk.BgLayout, tk.RedFillTertiary)},
		{"red solid", TagProps{Color: TagRed, Variant: TagSolid}, tk.Red},
		{"error filled", TagProps{Color: TagError}, testkit.Over(tk.BgLayout, tk.ErrorFillTertiary)},
		{"processing is info", TagProps{Color: TagProcessing}, testkit.Over(tk.BgLayout, tk.InfoFillTertiary)},
		{"custom hex", TagProps{Color: TagColor("#7c3aed")}, ui.Hex("#7c3aed")},
		{"borderless", TagProps{Color: TagRed, Variant: TagBorderless}, tk.BgLayout},
		{"outlined default", TagProps{Variant: TagOutlined}, tk.BgContainer},
	}
	for _, c := range cases {
		if got := newTagHarness(t, c.p).surface(); !testkit.Near(got, c.want) {
			t.Errorf("%s surface = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestTagOutlinedBorder(t *testing.T) {
	tk := theme.Light()
	h := newTagHarness(t, TagProps{Variant: TagOutlined})
	b := h.bounds()
	if got := h.pixel(b.X+b.W/2, b.Y+0.5); !testkit.Near(got, tk.BorderSecondary) {
		t.Errorf("outlined border = %+v, want BorderSecondary %+v", got, tk.BorderSecondary)
	}
}

func TestReadableOn(t *testing.T) {
	black, white := ui.RGB(0, 0, 0), ui.RGB(255, 255, 255)
	for _, c := range []struct {
		bg, want ui.Color
	}{
		{ui.Hex("#ffffff"), black},
		{ui.Hex("#fadb14"), black},
		{ui.Hex("#000000"), white},
		{ui.Hex("#0958d9"), white},
	} {
		if got := readableOn(c.bg); got != c.want {
			t.Errorf("readableOn(%+v) = %+v, want %+v", c.bg, got, c.want)
		}
	}
}

func TestTagClosable(t *testing.T) {
	h := newTagHarness(t, TagProps{Closable: true})
	if err := h.tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	if h.closed != 1 {
		t.Errorf("OnClose called %d times, want 1", h.closed)
	}
	if h.tt.HasText("Beta") {
		t.Error("tag still shown after closing")
	}
}
