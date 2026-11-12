package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func drawerTester(open *bool, p DrawerProps) *ui.Tester {
	return ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Drawer(c, open, p, func() { ui.Text(c, "Body text") })
	}), 800, 600)
}

func TestDrawerSlidesInAtItsEdge(t *testing.T) {
	tk := theme.Light()
	open := true
	tt := drawerTester(&open, DrawerProps{Title: "Settings"})
	settle(tt)
	title, ok := tt.Find("Settings")
	if !ok || title.X != 800-378+16 {
		t.Fatalf("title at %+v (shown %v), want x %v", title, ok, 800-378+16)
	}
	if body, _ := tt.Find("Body text"); body.Y < title.Y+title.H {
		t.Errorf("body at %+v, want under the header", body)
	}
	if got, want := testkit.PixelAt(tt, 800, 10, 10), testkit.Over(tk.BgLayout, tk.BgMask); !testkit.Near(got, want) {
		t.Errorf("backdrop %+v, want BgMask %+v", got, want)
	}
	if got := testkit.PixelAt(tt, 800, 790, 300); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("panel %+v, want BgContainer", got)
	}
	if edge := panelEdge(tt, 300); edge < 800-378-1 || edge > 800-378+2 {
		t.Errorf("panel starts at x %v, want %v", edge, 800-378)
	}
	open = false
	settle(tt)
	open = true
	tt.Frame()
	tt.Advance(30 * time.Millisecond)
	if edge := panelEdge(tt, 300); edge < 800-378+20 {
		t.Errorf("panel starts at x %v 30 ms into opening: want it still sliding in from the right", edge)
	}
}

func panelEdge(tt *ui.Tester, y float32) float32 {
	for x := float32(0.5); x < 800; x++ {
		if testkit.Near(testkit.PixelAt(tt, 800, x, y), theme.Light().BgContainer) {
			return x
		}
	}
	return 800
}

func TestDrawerSlidesOutAndThenGoes(t *testing.T) {
	open := true
	tt := drawerTester(&open, DrawerProps{Title: "Settings"})
	settle(tt)
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if open {
		t.Fatal("Escape did not close the drawer")
	}
	tt.Advance(100 * time.Millisecond)
	if !tt.HasText("Settings") {
		t.Fatal("drawer gone 100 ms into its exit")
	}
	if edge := panelEdge(tt, 300); edge < 800-378+10 || edge >= 800 {
		t.Errorf("panel starts at x %v 100 ms into closing: want it partway out", edge)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if tt.HasText("Settings") {
		t.Error("drawer still built after its exit")
	}
}

func TestDrawerPlacementAndSize(t *testing.T) {
	open := true
	tt := drawerTester(&open, DrawerProps{Title: "Filters", Placement: Left, Size: 300})
	settle(tt)
	if r, _ := tt.Find("Filters"); r.X != 16 {
		t.Errorf("left drawer title at x %v, want 16", r.X)
	}
	tt = drawerTester(&open, DrawerProps{Title: "Terminal", Placement: Bottom})
	settle(tt)
	if r, _ := tt.Find("Terminal"); r.Y < 600-378 || r.Y > 600-378+20 || r.X != 16 {
		t.Errorf("bottom drawer title at %+v, want in a 378 DIP panel along the bottom", r)
	}
}

func TestDrawerClosesFromItsButtonAndBackdrop(t *testing.T) {
	for name, dismiss := range map[string]func(tt *ui.Tester){
		"close": func(tt *ui.Tester) {
			info, ok := tt.AccessByRole(ui.RoleButton, "Close")
			if !ok {
				t.Fatal("no Close button")
			}
			tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		},
		"backdrop": func(tt *ui.Tester) { tt.ClickAt(100, 300) },
	} {
		open := true
		tt := drawerTester(&open, DrawerProps{Title: "Settings"})
		settle(tt)
		dismiss(tt)
		tt.Advance(time.Second)
		if open {
			t.Errorf("%s did not close the drawer", name)
		}
	}
}

func TestDrawerFooter(t *testing.T) {
	open := true
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Drawer(c, &open, DrawerProps{Title: "Settings", Footer: func() { ui.Text(c, "Save changes") }}, func() { ui.Text(c, "Body text") })
	}), 800, 600)
	settle(tt)
	f, ok := tt.Find("Save changes")
	if !ok || f.Y < 600-60 || f.X+f.W < 800-16-1 || f.X+f.W > 800-16+1 {
		t.Errorf("footer at %+v (shown %v), want at the bottom end of the panel", f, ok)
	}
}
