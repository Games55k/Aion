package display

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var (
	gaRed    = ui.RGB(230, 30, 30)
	gaGreen  = ui.RGB(30, 200, 60)
	gaBlue   = ui.RGB(30, 80, 230)
	gaYellow = ui.RGB(240, 200, 20)
	gaPurple = ui.RGB(140, 40, 200)
)

func gaItems(colors ...ui.Color) []GroupAvatarItem {
	items := make([]GroupAvatarItem, len(colors))
	for i, col := range colors {
		items[i] = GroupAvatarItem{Avatar: " ", Background: col, Fallback: AvatarText}
	}
	return items
}

func groupTester(p GroupAvatarProps) (*ui.Tester, ui.Rect) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).AlignItems(ui.Start).Children(func() {
			GroupAvatar(c, p).Label("group")
		})
	}, 300, 200)
	info, _ := tt.Access("group")
	return tt, info.Bounds
}

func TestGroupAvatarTwoByTwo(t *testing.T) {
	tt, b := groupTester(GroupAvatarProps{Size: 64, Avatars: gaItems(gaRed, gaGreen, gaBlue, gaYellow, gaPurple)})
	if b.W != 64 || b.H != 64 {
		t.Fatalf("bounds %+v, want 64 square", b)
	}
	at := func(x, y float32) ui.Color { return testkit.PixelAt(tt, 300, b.X+x, b.Y+y) }
	for _, k := range []struct {
		x, y float32
		col  ui.Color
	}{{18, 18, gaRed}, {46, 18, gaGreen}, {18, 46, gaBlue}, {46, 46, gaYellow}} {
		if got := at(k.x, k.y); !testkit.Near(got, k.col) {
			t.Errorf("avatar at %v,%v is %+v, want %+v", k.x, k.y, got, k.col)
		}
	}
	tk := theme.Light()
	fill := testkit.Over(tk.BgLayout, tk.FillTertiary)
	if got := at(32, 18); !testkit.Near(got, fill) {
		t.Errorf("gap %+v, want the group's fill %+v", got, fill)
	}
	if got := at(4, 32); !testkit.Near(got, fill) {
		t.Errorf("margin %+v, want the group's fill", got)
	}
	for y := float32(0); y < 64; y++ {
		for x := float32(0); x < 64; x++ {
			if testkit.Near(at(x, y), gaPurple) {
				t.Fatal("a fifth avatar shows in a 2 × 2 grid")
			}
		}
	}
}

func TestGroupAvatarAutoGridGoesThreeByThree(t *testing.T) {
	tt, b := groupTester(GroupAvatarProps{Size: 64, Grid: GroupGridAuto, Avatars: gaItems(gaRed, gaGreen, gaBlue, gaYellow, gaPurple)})
	at := func(x, y float32) ui.Color { return testkit.PixelAt(tt, 300, b.X+x, b.Y+y) }
	if got := at(14, 22); !testkit.Near(got, gaRed) {
		t.Errorf("first cell %+v, want red", got)
	}
	if got := at(32, 22); !testkit.Near(got, gaGreen) {
		t.Errorf("second cell %+v, want green", got)
	}
	if got := at(14, 41); !testkit.Near(got, gaYellow) {
		t.Errorf("fourth cell %+v, want yellow on the second row", got)
	}
}

func TestGroupAvatarSingleFillsMost(t *testing.T) {
	tt, b := groupTester(GroupAvatarProps{Size: 64, Avatars: gaItems(gaRed)})
	at := func(x, y float32) ui.Color { return testkit.PixelAt(tt, 300, b.X+x, b.Y+y) }
	if got := at(9, 32); !testkit.Near(got, gaRed) {
		t.Errorf("left edge %+v, want the one avatar 51 DIPs wide", got)
	}
	if got := at(5, 32); testkit.Near(got, gaRed) {
		t.Error("the one avatar is wider than 80% of the group")
	}
}

func TestGroupAvatarCorners(t *testing.T) {
	tk := theme.Light()
	tt, b := groupTester(GroupAvatarProps{Size: 64, Corner: GroupCornerCircle, Avatars: gaItems(gaRed, gaGreen)})
	if got := testkit.PixelAt(tt, 300, b.X+8, b.Y+8); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("circle corner %+v, want the page", got)
	}
	if got := testkit.PixelAt(tt, 300, b.X+12, b.Y+32); !testkit.Near(got, gaRed) {
		t.Errorf("circle's first avatar at 12,32 is %+v, want red", got)
	}
	if got := testkit.PixelAt(tt, 300, b.X+8, b.Y+32); testkit.Near(got, gaRed) {
		t.Error("a circle's avatars are as large as a square's: they reach its curve")
	}
	tt, b = groupTester(GroupAvatarProps{Size: 64, Avatars: gaItems(gaRed, gaGreen)})
	if got := testkit.PixelAt(tt, 300, b.X+1, b.Y+1); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("square corner %+v, want the page", got)
	}
	if got := testkit.PixelAt(tt, 300, b.X+8, b.Y+8); testkit.Near(got, tk.BgLayout) {
		t.Error("square corner rounds as far in as a circle")
	}
}
