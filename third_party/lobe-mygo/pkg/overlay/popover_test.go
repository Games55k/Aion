package overlay

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func popoverTester(open *bool, p PopoverProps) *ui.Tester {
	return ui.NewTester(testkit.Centered(func(c *ui.Context) {
		anchor := ui.ButtonBase(c).Size(80, 32).Background(ui.RGB(200, 200, 200)).Label("Share")
		Popover(c, anchor, open, p, func() { ui.Text(c, "Invite people") })
	}), 400, 300)
}

func TestPopoverOpensFromItsAnchorAndClosesOutside(t *testing.T) {
	open := false
	tt := popoverTester(&open, PopoverProps{})
	if tt.HasText("Invite people") {
		t.Fatal("content shows while closed")
	}
	tt.ClickAt(200, 150)
	tt.Advance(time.Second)
	if !open {
		t.Fatal("clicking the anchor did not open the popover")
	}
	r, ok := tt.Find("Invite people")
	if !ok {
		t.Fatal("no content while open")
	}
	if anchorTop := float32(150 - 16); r.Y+r.H > anchorTop-6-12 {
		t.Errorf("content at %+v, want above the anchor past the 6 DIP gap and 12 DIP padding", r)
	}
	tt.ClickAt(10, 290)
	tt.Advance(time.Second)
	if open {
		t.Error("pressing outside did not close the popover")
	}
}

func TestPopoverClosesWhenItsAnchorIsClickedAgainOrEscape(t *testing.T) {
	open := true
	tt := popoverTester(&open, PopoverProps{Placement: Bottom})
	tt.Advance(time.Second)
	tt.ClickAt(200, 150)
	tt.Advance(time.Second)
	if open {
		t.Fatal("clicking the anchor of an open popover did not close it")
	}
	open = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if open {
		t.Error("Escape did not close the popover")
	}
}

func TestPopoverPanelLook(t *testing.T) {
	tk := theme.Light()
	open := true
	tt := popoverTester(&open, PopoverProps{Placement: Bottom})
	tt.Advance(time.Second)
	r, ok := tt.Find("Invite people")
	if !ok {
		t.Fatal("no content")
	}
	if r.Y < 150+16+6+12-1 {
		t.Errorf("content at %+v, want below the anchor past the gap and padding", r)
	}
	if got := testkit.PixelAt(tt, 400, r.X-6, r.Y+r.H/2); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("panel padding = %+v, want BgElevated %+v", got, tk.BgElevated)
	}
}

func TestPopoverAnimatesOutAfterClosing(t *testing.T) {
	open := true
	tt := popoverTester(&open, PopoverProps{Placement: Bottom})
	tt.Advance(time.Second)
	r, _ := tt.Find("Invite people")
	x, y := r.X-6, r.Y+r.H/2
	bg := testkit.PixelAt(tt, 400, 5, 5)
	tt.Key(0, ui.KeyEscape)
	tt.Advance(20 * time.Millisecond)
	if got := testkit.PixelAt(tt, 400, x, y); testkit.Near(got, bg) {
		t.Error("panel gone at once, want it animating out")
	}
	tt.Advance(time.Second)
	if got := testkit.PixelAt(tt, 400, x, y); !testkit.Near(got, bg) {
		t.Errorf("panel still painted %+v after its exit", got)
	}
}
