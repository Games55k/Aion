package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func toastTester(show *[]ToastProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		for _, t := range *show {
			ShowToast(c, t)
		}
		*show = nil
		ToastHost(c, ToastHostProps{})
	}, 800, 600)
}

func TestToastShowsAtTheBottomRight(t *testing.T) {
	tk := theme.Light()
	show := []ToastProps{{Title: "Saved", Description: "Your draft is safe.", Type: ToastSuccess}}
	tt := toastTester(&show)
	tt.Frame()
	tt.Advance(time.Second)
	title, ok := tt.Find("Saved")
	if !ok {
		t.Fatal("no toast")
	}
	if left := float32(800 - 16 - 360); title.X != left+16+18+12 {
		t.Errorf("title at x %v, want after the 18 DIP icon in a 360 DIP toast at %v", title.X, left)
	}
	desc, _ := tt.Find("Your draft is safe.")
	if bottom := desc.Y + desc.H; bottom < 600-16-12-2 || bottom > 600-16-12+1 {
		t.Errorf("description ends at %v, want the toast 16 DIPs above the bottom", bottom)
	}
	if got := testkit.PixelAt(tt, 800, title.X-40, title.Y); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("toast surface %+v, want BgElevated", got)
	}
}

func TestToastEntersFromBelow(t *testing.T) {
	show := []ToastProps{{Title: "Saved"}}
	tt := toastTester(&show)
	tt.Frame()
	tt.Advance(time.Second)
	rest, _ := tt.Find("Saved")
	show = []ToastProps{{Title: "Copied"}}
	tt.Frame()
	tt.Advance(40 * time.Millisecond)
	tt.Frame()
	if r, _ := tt.Find("Copied"); r.Y <= rest.Y+4 {
		t.Errorf("new toast at y %v 40 ms in, rest %v: want it rising from below", r.Y, rest.Y)
	}
}

func TestToastsStackAndExpandOnHover(t *testing.T) {
	tk := theme.Light()
	show := []ToastProps{{Title: "First"}}
	tt := toastTester(&show)
	tt.Frame()
	tt.Advance(time.Second)
	show = []ToastProps{{Title: "Second"}}
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	front, _ := tt.Find("Second")
	frontTop := front.Y - 12
	if got := testkit.PixelAt(tt, 800, 600, frontTop-6); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("collapsed: %+v just above the front toast, want the older one peeking", got)
	}
	if got := testkit.PixelAt(tt, 800, 600, frontTop-20); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("collapsed: %+v 20 DIPs above the front toast, want only a 12 DIP peek", got)
	}
	tt.Move(600, front.Y)
	tt.Advance(time.Second)
	tt.Frame()
	older, _ := tt.Find("First")
	front, _ = tt.Find("Second")
	if gap := (front.Y - 12) - (older.Y + older.H + 12); gap < 11 || gap > 13 {
		t.Errorf("expanded: %v DIPs between toasts, want 12", gap)
	}
}

func TestToastCloseButton(t *testing.T) {
	show := []ToastProps{{Title: "Saved", Timeout: -1}}
	tt := toastTester(&show)
	tt.Frame()
	tt.Advance(time.Second)
	info, ok := tt.AccessByRole(ui.RoleButton, "Close")
	if !ok {
		t.Fatal("no Close button")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if tt.HasText("Saved") {
		t.Error("toast still shows after Close")
	}
}

func twoToasts(t *testing.T) (*ui.Tester, ui.Rect) {
	t.Helper()
	show := []ToastProps{{Title: "First"}}
	tt := toastTester(&show)
	tt.Frame()
	tt.Advance(time.Second)
	show = []ToastProps{{Title: "Second"}}
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	front, _ := tt.Find("Second")
	return tt, front
}

func TestToastOlderLayerPeeksExactly12(t *testing.T) {
	tk := theme.Light()
	tt, front := twoToasts(t)
	frontTop := front.Y - 12
	if got := testkit.PixelAt(tt, 800, 600, frontTop-10.5); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("10.5 DIPs above the front toast %+v, want the older one", got)
	}
	if got := testkit.PixelAt(tt, 800, 600, frontTop-13.5); testkit.Near(got, tk.BgElevated) {
		t.Errorf("13.5 DIPs above the front toast %+v, want past the 12 DIP peek", got)
	}
}

func TestToastFrontStaysPutWhileTheStackSpreads(t *testing.T) {
	tt, front := twoToasts(t)
	tt.Move(600, front.Y)
	for i := 0; i < 8; i++ {
		tt.Advance(25 * time.Millisecond)
		tt.Frame()
		if r, _ := tt.Find("Second"); r.Y < front.Y-0.5 || r.Y > front.Y+0.5 {
			t.Fatalf("%d ms into spreading: front toast at y %v, want it to stay at %v", (i+1)*25, r.Y, front.Y)
		}
	}
}
