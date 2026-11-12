package display

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

func iconBox(t *testing.T, view func(c *ui.Context)) (ui.Rect, *ui.Tester) {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Center().Children(func() { view(c) })
	}, 200, 200)
	box, ok := tt.Find("Glyph")
	if !ok {
		t.Fatal("icon labelled Glyph not found")
	}
	return box, tt
}

func TestIconPresetSizes(t *testing.T) {
	for _, c := range []struct {
		size IconSize
		want float32
	}{{IconSmall, 14}, {IconMiddle, 20}, {IconLarge, 24}} {
		box, _ := iconBox(t, func(ctx *ui.Context) {
			Icon(ctx, lucide.Save(), IconProps{Size: c.size}).Label("Glyph")
		})
		if box.W != c.want || box.H != c.want {
			t.Errorf("size %v = %vx%v, want %v", c.size, box.W, box.H, c.want)
		}
	}
}

func TestIconInheritsFontSize(t *testing.T) {
	box, _ := iconBox(t, func(c *ui.Context) {
		ui.Row(c).FontSize(18).Children(func() {
			Icon(c, lucide.Save(), IconProps{}).Label("Glyph")
		})
	})
	if box.H != 18 {
		t.Errorf("inherited icon height = %v, want 18", box.H)
	}
}

func TestIconSpinKeepsAnimating(t *testing.T) {
	_, tt := iconBox(t, func(c *ui.Context) {
		Icon(c, lucide.LoaderCircle(), IconProps{Spin: true}).Label("Glyph")
	})
	tt.Advance(300 * time.Millisecond)
	if !tt.FramePending() {
		t.Error("spinning icon stopped asking for frames")
	}
}
