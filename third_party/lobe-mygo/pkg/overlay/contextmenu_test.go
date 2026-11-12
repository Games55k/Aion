package overlay

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestContextMenuOpensOnRightClick(t *testing.T) {
	var chosen []string
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			target := ui.Box(c).Size(200, 100).Label("file")
			if k := ContextMenu(c, target, DropdownMenuProps{Items: []MenuItem{{Key: "rename", Label: "Rename"}, {Key: "delete", Label: "Delete", Danger: true}}}); k != "" {
				chosen = append(chosen, k)
			}
		})
	}, 400, 300)
	if tt.HasText("Rename") {
		t.Fatal("menu shows before a right click")
	}
	tt.RightClickAt(120, 70)
	tt.Advance(time.Second)
	tt.Frame()
	r, ok := tt.Find("Rename")
	if !ok {
		t.Fatalf("no menu after right click: %q", tt.Texts())
	}
	if r.X < 120 || r.Y < 70 {
		t.Errorf("menu at %+v, want it where the click was", r)
	}
	tt.Click("Delete")
	tt.Advance(time.Second)
	tt.Frame()
	if len(chosen) != 1 || chosen[0] != "delete" || tt.HasText("Rename") {
		t.Errorf("chosen %q, texts %q", chosen, tt.Texts())
	}
}
