package nav

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var burgerItems = []display.ListItem{
	{Key: "home", Label: "Home", Icon: lucide.House()},
	{Key: "docs", Label: "Docs", Icon: lucide.BookOpen()},
	{Divider: true},
	{Key: "about", Label: "About"},
}

type burgerApp struct {
	open     bool
	selected string
	p        BurgerProps
}

func burgerTester(a *burgerApp) *ui.Tester {
	a.p.OnSelect = func(k string) { a.selected = k }
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Height(64).Padding(0, 12).AlignItems(ui.Center).Justify(ui.End).Children(func() {
			Burger(c, &a.open, a.p)
		})
	}, 400, 700)
}

func settleBurger(tt *ui.Tester) {
	tt.Advance(time.Second)
	tt.Frame()
}

func burgerClick(t *testing.T, tt *ui.Tester, role ui.Role, name string) {
	t.Helper()
	info, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("no %q", name)
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	settleBurger(tt)
}

func TestBurgerOpensUnderTheHeaderAndSelects(t *testing.T) {
	a := &burgerApp{p: BurgerProps{Items: burgerItems, Active: "home"}}
	tt := burgerTester(a)
	burgerClick(t, tt, ui.RoleButton, "Open menu")
	if !a.open {
		t.Fatal("the toggle did not open the menu")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Close menu"); !ok {
		t.Error("the toggle does not turn into Close menu")
	}
	home, ok := tt.Find("Home")
	if !ok || home.Y < 64 {
		t.Fatalf("Home at %+v, want under the 64 DIP header", home)
	}
	burgerClick(t, tt, ui.RoleButton, "Docs")
	if a.selected != "docs" || a.open {
		t.Errorf("selected %q, open %v: want docs and the menu closed", a.selected, a.open)
	}
}

func TestBurgerRows(t *testing.T) {
	tk := theme.Light()
	a := &burgerApp{open: true, p: BurgerProps{Items: burgerItems, Active: "home"}}
	tt := burgerTester(a)
	settleBurger(tt)
	home, _ := tt.AccessByRole(ui.RoleButton, "Home")
	docs, _ := tt.AccessByRole(ui.RoleButton, "Docs")
	if home.Bounds.H != 52 {
		t.Errorf("row height %v, want 52", home.Bounds.H)
	}
	if got := testkit.PixelAt(tt, 400, home.Bounds.X+home.Bounds.W-30, home.Bounds.Y+home.Bounds.H/2); !testkit.Near(got, tk.Text) {
		t.Errorf("current row %+v, want filled with the text color", got)
	}
	if got := testkit.PixelAt(tt, 400, docs.Bounds.X+docs.Bounds.W-30, docs.Bounds.Y+docs.Bounds.H/2); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("other row %+v, want the drawer's background", got)
	}
	if got := testkit.PixelAt(tt, 400, home.Bounds.X+1, home.Bounds.Y+1); testkit.Near(got, tk.Text) {
		t.Error("the current row has square corners: want a pill")
	}
}

func TestBurgerFullscreen(t *testing.T) {
	a := &burgerApp{open: true, p: BurgerProps{Items: burgerItems, Fullscreen: true}}
	tt := burgerTester(a)
	settleBurger(tt)
	home, ok := tt.AccessByRole(ui.RoleButton, "Home")
	if !ok || home.Bounds.H < 48 {
		t.Fatalf("Home %+v, want a large row", home.Bounds)
	}
	if home.Bounds.Y < 64 {
		t.Errorf("Home at %v, want under the drawer's own 64 DIP header", home.Bounds.Y)
	}
	closes := 0
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleButton && n.Name == "Close menu" {
			closes++
		}
	}
	if closes < 2 {
		t.Errorf("%d Close menu buttons, want the drawer's own besides the toggle", closes)
	}
	about, _ := tt.AccessByRole(ui.RoleButton, "About")
	if gap := about.Bounds.Y - (home.Bounds.Y + home.Bounds.H*2); gap > 4 {
		t.Errorf("About sits %v below Docs: want the divider left out", gap)
	}
}

func TestBurgerFullscreenFooter(t *testing.T) {
	a := &burgerApp{open: true, p: BurgerProps{Items: burgerItems, Fullscreen: true}}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.p.Footer = func() { ui.Text(c, "Sign in") }
		Burger(c, &a.open, a.p)
	}, 400, 700)
	settleBurger(tt)
	if r, ok := tt.Find("Sign in"); !ok || r.Y < 600 {
		t.Errorf("footer at %+v, want at the bottom", r)
	}
}

func TestBurgerClosesOnEscape(t *testing.T) {
	a := &burgerApp{open: true, p: BurgerProps{Items: burgerItems}}
	tt := burgerTester(a)
	settleBurger(tt)
	tt.Key(0, ui.KeyEscape)
	settleBurger(tt)
	if a.open {
		t.Error("Escape left the menu open")
	}
	if _, ok := tt.Find("Home"); ok {
		t.Error("the rows stay after the menu closed")
	}
}

func TestBurgerRowsAlignToTheStart(t *testing.T) {
	for _, full := range []bool{false, true} {
		a := &burgerApp{open: true, p: BurgerProps{Items: burgerItems, Fullscreen: full}}
		tt := burgerTester(a)
		settleBurger(tt)
		row, _ := tt.AccessByRole(ui.RoleButton, "About")
		bg := theme.Light().BgContainer
		start := ui.Rect{X: row.Bounds.X + 8, Y: row.Bounds.Y + 4, W: 24, H: row.Bounds.H - 8}
		if got := testkit.Ink(tt, 400, start, bg); testkit.Near(got, bg) {
			t.Errorf("fullscreen %v: nothing at the row's start: want the label there", full)
		}
	}
}
