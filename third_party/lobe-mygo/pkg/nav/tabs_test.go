package nav

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func tabsTester(sel *int, p TabsProps) *ui.Tester {
	p.Items = []TabItem{{Label: "Chat"}, {Label: "Files"}, {Label: "Settings"}, {Label: "Billing", Disabled: true}}
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(400).Children(func() {
			Tabs(c, sel, p, func(i int) { ui.Text(c, "Panel "+p.Items[i].Label) })
		})
	}, 440, 200)
}

func TestTabsChooseAndShowPanel(t *testing.T) {
	sel := 0
	tt := tabsTester(&sel, TabsProps{})
	if !tt.HasText("Panel Chat") {
		t.Fatalf("texts %q", tt.Texts())
	}
	tt.Click("Files")
	tt.Advance(time.Second)
	tt.Frame()
	if sel != 1 || !tt.HasText("Panel Files") || tt.HasText("Panel Chat") {
		t.Errorf("after clicking Files: sel %d texts %q", sel, tt.Texts())
	}
	tt.Click("Billing")
	tt.Frame()
	if sel != 1 {
		t.Errorf("a disabled tab was chosen: %d", sel)
	}
}

func TestTabsIndicatorSlides(t *testing.T) {
	tk := theme.Light()
	sel := 0
	tt := tabsTester(&sel, TabsProps{})
	tt.Advance(time.Second)
	tt.Frame()
	chat, files, settings := tabBox(t, tt, "Chat"), tabBox(t, tt, "Files"), tabBox(t, tt, "Settings")
	under := func(r ui.Rect) ui.Color { return testkit.PixelAt(tt, 440, r.X+2, r.Y+r.H/2) }
	if !testkit.Near(under(chat), tk.BgElevated) {
		t.Errorf("rounded indicator under Chat %+v, want BgElevated", under(chat))
	}
	tt.Click("Settings")
	tt.Advance(80 * time.Millisecond)
	tt.Frame()
	gap := func() ui.Color { return testkit.PixelAt(tt, 440, files.X+files.W+2, files.Y+files.H/2) }
	if !testkit.Near(gap(), tk.BgElevated) {
		t.Errorf("80 ms into the move, between Files and Settings %+v: want the indicator sliding across", gap())
	}
	tt.Advance(time.Second)
	tt.Frame()
	if !testkit.Near(under(settings), tk.BgElevated) || testkit.Near(under(chat), tk.BgElevated) || testkit.Near(gap(), tk.BgElevated) {
		t.Error("indicator did not settle under Settings")
	}
}

func TestTabsSquareUnderline(t *testing.T) {
	tk := theme.Light()
	sel := 1
	tt := tabsTester(&sel, TabsProps{Variant: TabsSquare})
	tt.Advance(time.Second)
	tt.Frame()
	files := tabBox(t, tt, "Files")
	if got := testkit.PixelAt(tt, 440, files.X+files.W/2, files.Y+files.H-1); !testkit.Near(got, tk.Primary) {
		t.Errorf("square underline under Files %+v, want Primary", got)
	}
}

func tabBox(t *testing.T, tt *ui.Tester, name string) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleTab, name)
	if !ok {
		t.Fatalf("no tab %q", name)
	}
	return info.Bounds
}
