package overlay

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var menuItems = []MenuItem{
	{Key: "rename", Label: "Rename", Hotkey: "f2"},
	{Key: "duplicate", Label: "Duplicate", Extra: "x2"},
	{Key: "share", Label: "Share", Disabled: true},
	{Divider: true},
	{Key: "delete", Label: "Delete", Danger: true},
}

func menuTester(open *bool, chosen *[]string) *ui.Tester {
	return ui.NewTester(testkit.Centered(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			key := DropdownMenu(c, open, DropdownMenuProps{Items: menuItems}, func(t ui.Element) {
				t.Padding(6, 12).Children(func() { ui.Text(c, "Actions") })
			})
			if key != "" {
				*chosen = append(*chosen, key)
			}
		})
	}), 400, 400)
}

func clickText(t *testing.T, tt *ui.Tester, s string) {
	t.Helper()
	if err := tt.Click(s); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
}

func TestDropdownMenuOpensBelowTriggerAndReturnsTheChoice(t *testing.T) {
	open := false
	var chosen []string
	tt := menuTester(&open, &chosen)
	if tt.HasText("Rename") {
		t.Fatal("items show while closed")
	}
	trigger, _ := tt.Find("Actions")
	tt.Press(trigger.X+trigger.W/2, trigger.Y+trigger.H/2)
	tt.Release(trigger.X+trigger.W/2, trigger.Y+trigger.H/2)
	tt.Advance(time.Second)
	if !open {
		t.Fatal("pressing the trigger did not open the menu")
	}
	first, ok := tt.Find("Rename")
	if !ok {
		t.Fatal("no items while open")
	}
	if first.Y < trigger.Y+trigger.H+6 {
		t.Errorf("first item at %+v, want below the trigger %+v past the gap", first, trigger)
	}
	tk := theme.Light()
	rowEnd := first.X + 220 - 2*(4+12)
	key, ok := tt.Find("F2")
	if capEnd := key.X + key.W + 8; !ok || capEnd < rowEnd+7-0.5 || capEnd > rowEnd+7+0.5 {
		t.Errorf("hotkey F2 at %+v (shown %v), want its cap 5 DIPs from the row edge, as far as above and below it", key, ok)
	}
	if got := testkit.PixelAt(tt, 400, key.X-4, key.Y+key.H/2); !testkit.Near(got, testkit.Over(tk.BgElevated, tk.FillTertiary)) {
		t.Errorf("behind the hotkey %+v, want a filled key cap", got)
	}
	dup, _ := tt.Find("Duplicate")
	extra, ok := tt.Find("x2")
	if !ok || extra.X+extra.W < rowEnd-0.5 || extra.X+extra.W > rowEnd+0.5 {
		t.Errorf("extra at %+v (shown %v), want flush with the row end %v after %+v", extra, ok, rowEnd, dup)
	}
	clickText(t, tt, "Duplicate")
	if len(chosen) != 1 || chosen[0] != "duplicate" {
		t.Errorf("chosen = %v, want [duplicate]", chosen)
	}
	if open {
		t.Error("choosing an item did not close the menu")
	}
}

func TestDropdownMenuIgnoresDisabledItems(t *testing.T) {
	open := true
	var chosen []string
	tt := menuTester(&open, &chosen)
	tt.Advance(time.Second)
	clickText(t, tt, "Share")
	if len(chosen) != 0 {
		t.Errorf("disabled item chosen: %v", chosen)
	}
}

func TestDropdownMenuItemLook(t *testing.T) {
	tk := theme.Light()
	open := true
	var chosen []string
	tt := menuTester(&open, &chosen)
	tt.Advance(time.Second)
	rename, _ := tt.Find("Rename")
	duplicate, _ := tt.Find("Duplicate")
	if rowPitch := duplicate.Y - rename.Y; rowPitch != 32 {
		t.Errorf("rows are %v DIPs apart, want 32", rowPitch)
	}
	tt.Move(duplicate.X+duplicate.W/2, duplicate.Y+duplicate.H/2)
	tt.Advance(time.Second)
	hovered := testkit.PixelAt(tt, 400, duplicate.X-6, duplicate.Y+duplicate.H/2)
	if want := testkit.Over(tk.BgElevated, tk.FillTertiary); !testkit.Near(hovered, want) {
		t.Errorf("hovered item = %+v, want FillTertiary over BgElevated %+v", hovered, want)
	}
	if rest := testkit.PixelAt(tt, 400, rename.X-6, rename.Y+rename.H/2); !testkit.Near(rest, tk.BgElevated) {
		t.Errorf("resting item = %+v, want BgElevated %+v", rest, tk.BgElevated)
	}
	share, _ := tt.Find("Share")
	del, _ := tt.Find("Delete")
	if gap := del.Y - share.Y; gap != 32+9 {
		t.Errorf("Delete is %v DIPs below Share, want a row plus the 9 DIP divider", gap)
	}
}
