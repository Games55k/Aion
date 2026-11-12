package field

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

const inputWidth = 320

type inputHarness struct {
	t     *testing.T
	tt    *ui.Tester
	value string
}

func newInputHarness(t *testing.T, p InputProps) *inputHarness {
	h := &inputHarness{t: t}
	h.tt = ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Input(c, &h.value, p).Label("Name")
	}), inputWidth, 140)
	return h
}

func (h *inputHarness) field() ui.Rect {
	h.t.Helper()
	info, ok := h.tt.AccessByRole(ui.RoleTextField, "Name")
	if !ok {
		h.t.Fatal("no text field named Name")
	}
	return info.Bounds
}

func (h *inputHarness) px(x, y float32) ui.Color { return testkit.PixelAt(h.tt, inputWidth, x, y) }

func (h *inputHarness) frame() (top, bottom float32, border ui.Color) {
	h.t.Helper()
	f := h.field()
	x := f.X + f.W/2
	inside := h.px(x, f.Y+f.H/2)
	top = f.Y + f.H/2
	for top > 0 && h.px(x, top-1) == inside {
		top--
	}
	bottom = f.Y + f.H/2
	for bottom < 140 && h.px(x, bottom+1) == inside {
		bottom++
	}
	return top - 1, bottom + 1, h.px(x, top-0.5)
}

func (h *inputHarness) focus() {
	f := h.field()
	h.tt.ClickAt(f.X+f.W/2, f.Y+f.H/2)
	h.tt.Move(0, 0)
	h.tt.Advance(time.Second)
}

func TestInputHeights(t *testing.T) {
	for _, c := range []struct {
		size theme.Size
		want float32
	}{{theme.Small, 24}, {theme.Middle, 32}, {theme.Large, 40}} {
		top, bottom, _ := newInputHarness(t, InputProps{Size: c.size}).frame()
		if got := bottom - top + 1; got != c.want {
			t.Errorf("size %v height = %v, want %v", c.size, got, c.want)
		}
	}
}

func TestInputTyping(t *testing.T) {
	h := newInputHarness(t, InputProps{Placeholder: "Your name"})
	h.focus()
	h.tt.Type("Ada")
	if h.value != "Ada" {
		t.Errorf("value = %q, want Ada", h.value)
	}
}

func TestInputOutlinedBorderFollowsFocus(t *testing.T) {
	tk := theme.Light()
	h := newInputHarness(t, InputProps{})
	f := h.field()
	if got := h.px(f.X-6, f.Y+f.H/2); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("outlined surface = %+v, want BgContainer", got)
	}
	if _, _, border := h.frame(); !testkit.Near(border, tk.BorderSecondary) {
		t.Errorf("rest border = %+v, want BorderSecondary %+v", border, tk.BorderSecondary)
	}
	h.focus()
	top, _, border := h.frame()
	if !testkit.Near(border, tk.Primary) {
		t.Errorf("focused border = %+v, want Primary %+v", border, tk.Primary)
	}
	if got, want := h.px(f.X+f.W/2, top-1.5), testkit.Over(tk.BgLayout, tk.PrimaryBg); !testkit.Near(got, want) {
		t.Errorf("focus ring = %+v, want PrimaryBg %+v", got, want)
	}
}

func TestInputFilledAndInvalid(t *testing.T) {
	tk := theme.Light()
	h := newInputHarness(t, InputProps{Variant: InputFilled})
	f := h.field()
	if got, want := h.px(f.X-6, f.Y+f.H/2), testkit.Over(tk.BgLayout, tk.FillTertiary); !testkit.Near(got, want) {
		t.Errorf("filled surface = %+v, want %+v", got, want)
	}
	if _, _, border := newInputHarness(t, InputProps{Invalid: true}).frame(); !testkit.Near(border, tk.Error) {
		t.Errorf("invalid border = %+v, want Error %+v", border, tk.Error)
	}
}

func TestTextAreaGrowsWithLines(t *testing.T) {
	value := ""
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		TextArea(c, &value, TextAreaProps{MinRows: 2, MaxRows: 6}).Label("Notes")
	}), inputWidth, 300)
	info, ok := tt.AccessByRole(ui.RoleTextField, "Notes")
	if !ok {
		t.Fatal("no text field named Notes")
	}
	before := info.Bounds.H
	tt.ClickAt(info.Bounds.X+10, info.Bounds.Y+info.Bounds.H/2)
	tt.Type("one\ntwo\nthree\nfour")
	tt.Advance(time.Second)
	after, _ := tt.AccessByRole(ui.RoleTextField, "Notes")
	if value != "one\ntwo\nthree\nfour" {
		t.Errorf("value = %q", value)
	}
	if after.Bounds.H <= before {
		t.Errorf("textarea stayed %v tall after four lines (was %v)", after.Bounds.H, before)
	}
}
