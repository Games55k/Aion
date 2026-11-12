package action

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func groupTester(p ActionIconGroupProps, got *[]string) *ui.Tester {
	p.Items = []ActionGroupItem{
		{Key: "copy", Icon: lucide.Copy(), Label: "Copy"},
		{Key: "edit", Icon: lucide.Pencil(), Label: "Edit"},
		{Key: "delete", Icon: lucide.Trash(), Label: "Delete", Danger: true},
	}
	p.Menu = []overlay.MenuItem{{Key: "share", Label: "Share"}, {Key: "export", Label: "Export"}}
	return ui.NewTester(testkit.Centered(func(c *ui.Context) {
		if k := ActionIconGroup(c, p); k != "" {
			*got = append(*got, k)
		}
	}), 400, 300)
}

func TestActionIconGroupItemsAndMenu(t *testing.T) {
	var got []string
	tt := groupTester(ActionIconGroupProps{Size: theme.Small}, &got)
	copyB, ok := tt.AccessByRole(ui.RoleButton, "Copy")
	edit, _ := tt.AccessByRole(ui.RoleButton, "Edit")
	if !ok || copyB.Bounds.W != 24 || edit.Bounds.X != copyB.Bounds.X+24 || edit.Bounds.Y != copyB.Bounds.Y {
		t.Fatalf("copy %v edit %v: want 24 DIP icons side by side", copyB.Bounds, edit.Bounds)
	}
	tt.ClickAt(edit.Bounds.X+12, edit.Bounds.Y+12)
	tt.Frame()
	more, ok := tt.AccessByRole(ui.RoleMenuButton, "More")
	if !ok {
		t.Fatal("no More button for the menu")
	}
	tt.ClickAt(more.Bounds.X+12, more.Bounds.Y+12)
	tt.Advance(time.Second)
	tt.Click("Export")
	tt.Advance(time.Second)
	if len(got) != 2 || got[0] != "edit" || got[1] != "export" {
		t.Errorf("got %q, want [edit export]", got)
	}
}

func TestActionIconGroupSurfaceAndVertical(t *testing.T) {
	tk := theme.Light()
	var got []string
	tt := groupTester(ActionIconGroupProps{Size: theme.Small}, &got)
	copyB, _ := tt.AccessByRole(ui.RoleButton, "Copy")
	if px := testkit.PixelAt(tt, 400, copyB.Bounds.X-1, copyB.Bounds.Y+12); !testkit.Near(px, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("group padding %+v, want the filled surface", px)
	}
	tt = groupTester(ActionIconGroupProps{Size: theme.Small, Vertical: true, Disabled: true}, &got)
	copyB, _ = tt.AccessByRole(ui.RoleButton, "Copy")
	edit, _ := tt.AccessByRole(ui.RoleButton, "Edit")
	if edit.Bounds.Y != copyB.Bounds.Y+24 || edit.Bounds.X != copyB.Bounds.X {
		t.Errorf("vertical: copy %v edit %v", copyB.Bounds, edit.Bounds)
	}
	tt.ClickAt(edit.Bounds.X+12, edit.Bounds.Y+12)
	tt.Frame()
	if len(got) != 0 {
		t.Errorf("a disabled group returned %q", got)
	}
}
