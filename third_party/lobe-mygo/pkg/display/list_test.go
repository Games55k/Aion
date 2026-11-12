package display

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func listTester(active *string, clicked *[]string, actions *int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(320).Children(func() {
			k := List(c, ListProps{Active: *active, Items: []ListItem{
				{Key: "chat", Label: "Chat", Description: "Talk to agents", Icon: lucide.MessageSquare(), Extra: "12"},
				{Key: "files", Label: "Files", Actions: func() {
					if ui.ButtonBase(c).Label("Pin").Size(20, 20).Clicked() {
						*actions++
					}
				}},
				{Divider: true},
				{Key: "trash", Label: "Trash", Danger: true},
				{Key: "locked", Label: "Locked", Disabled: true},
			}})
			if k != "" {
				*clicked = append(*clicked, k)
				*active = k
			}
		})
	}, 360, 300)
}

func TestListClicksAndActive(t *testing.T) {
	tk := theme.Light()
	active, clicked, actions := "chat", []string{}, 0
	tt := listTester(&active, &clicked, &actions)
	chat, _ := tt.AccessByRole(ui.RoleButton, "Chat")
	if got := testkit.PixelAt(tt, 360, chat.Bounds.X+4, chat.Bounds.Y+chat.Bounds.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillSecondary)) {
		t.Errorf("active row %+v, want FillSecondary", got)
	}
	if chat.Bounds.W < 280-0.5 {
		t.Errorf("row %v wide, want the list's width", chat.Bounds.W)
	}
	desc, ok := tt.Find("Talk to agents")
	if want := chat.Bounds.X + 12 + 16 + 12; !ok || desc.X != want || desc.Y < chat.Bounds.Y+chat.Bounds.H/2-1 {
		t.Errorf("description %v, want under the label at x %v past the 16 DIP icon", desc, want)
	}
	extra, _ := tt.Find("12")
	if extra.X+extra.W < chat.Bounds.X+chat.Bounds.W-12-0.5 {
		t.Errorf("extra %v not at the row end %v", extra, chat.Bounds)
	}
	tt.Click("Trash")
	tt.Frame()
	tt.Click("Locked")
	tt.Frame()
	if len(clicked) != 1 || clicked[0] != "trash" || active != "trash" {
		t.Errorf("clicked %q active %q", clicked, active)
	}
}

func TestListActionsShowOnHover(t *testing.T) {
	active, clicked, actions := "", []string{}, 0
	tt := listTester(&active, &clicked, &actions)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Pin"); ok {
		t.Fatal("actions show while the row is not hovered")
	}
	files, _ := tt.AccessByRole(ui.RoleButton, "Files")
	tt.Move(files.Bounds.X+40, files.Bounds.Y+files.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	pin, ok := tt.AccessByRole(ui.RoleButton, "Pin")
	if !ok {
		t.Fatal("no actions on hover")
	}
	if end := files.Bounds.X + files.Bounds.W - 8; pin.Bounds.X+pin.Bounds.W < end-0.5 || pin.Bounds.X+pin.Bounds.W > end+0.5 {
		t.Errorf("actions end at %v, want 8 DIPs from the row end %v", pin.Bounds.X+pin.Bounds.W, end)
	}
	tt.ClickAt(pin.Bounds.X+10, pin.Bounds.Y+10)
	tt.Frame()
	if actions != 1 || len(clicked) != 0 {
		t.Errorf("action clicks %d, row clicks %q: want the action alone", actions, clicked)
	}
}

func railTester(collapse *float32, width float32) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(width).Children(func() {
			List(c, ListProps{Active: "home", Collapse: *collapse, RailWidth: 53, Items: []ListItem{
				{Key: "home", Label: "Home", Icon: lucide.House()},
				{Key: "files", Label: "Files", Icon: lucide.Folder()},
				{Key: "rene", Label: "Rene Wang", Description: "Playing Cursor", Avatar: func() {
					ui.Box(c).Size(36, 36).Background(ui.RGB(20, 120, 220)).Label("avatar")
				}},
			}})
		})
	}, 400, 300)
}

func glyphX(t *testing.T, tt *ui.Tester, bg ui.Color, row ui.Rect) float32 {
	t.Helper()
	for x := row.X; x < row.X+row.W; x += 0.5 {
		if !testkit.Near(testkit.PixelAt(tt, 400, x, row.Y+row.H/2), bg) {
			return x
		}
	}
	t.Fatalf("no ink in %+v", row)
	return 0
}

func TestListCollapsedRailCentersSquareRows(t *testing.T) {
	tk := theme.Light()
	collapse := float32(1)
	tt := railTester(&collapse, 53)
	home, _ := tt.AccessByRole(ui.RoleButton, "Home")
	if home.Bounds.W != home.Bounds.H {
		t.Errorf("collapsed active row %+v, want a square", home.Bounds)
	}
	if mid := home.Bounds.X + home.Bounds.W/2; abs32(mid-26.5) > 0.5 {
		t.Errorf("collapsed row centered at %v, want 26.5", mid)
	}
	files, _ := tt.AccessByRole(ui.RoleButton, "Files")
	if x := glyphX(t, tt, tk.BgLayout, ui.Rect{X: 0, Y: files.Bounds.Y, W: 53, H: files.Bounds.H}); abs32(x-18.5) > 1.5 {
		t.Errorf("collapsed icon starts at %v, want centered at 18.5", x)
	}
	av, _ := tt.Find("avatar")
	if abs32(av.X-8.5) > 0.5 {
		t.Errorf("collapsed avatar at %v, want centered at 8.5", av.X)
	}
	rene, _ := tt.AccessByRole(ui.RoleButton, "Rene Wang")
	if got := testkit.Ink(tt, 400, ui.Rect{X: 46, Y: rene.Bounds.Y, W: 7, H: rene.Bounds.H}, tk.BgLayout); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("collapsed rail shows ink %+v beside the avatar: want the label gone", got)
	}
}

func TestListCollapseMovesRowsAlongTheWay(t *testing.T) {
	collapse := float32(0)
	tt := railTester(&collapse, 300)
	av, _ := tt.Find("avatar")
	open := av.X
	collapse = 0.5
	tt.Frame()
	av, _ = tt.Find("avatar")
	half := av.X
	if open != 12 {
		t.Errorf("avatar at %v open, want 12", open)
	}
	if abs32(half-(12+(8.5-12)*0.5)) > 0.5 {
		t.Errorf("avatar halfway at %v, want %v", half, 12+(8.5-12)*0.5)
	}
	home, _ := tt.AccessByRole(ui.RoleButton, "Home")
	if home.Bounds.W >= 300 || home.Bounds.W <= home.Bounds.H {
		t.Errorf("active row halfway %+v, want narrowing toward a square", home.Bounds)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
