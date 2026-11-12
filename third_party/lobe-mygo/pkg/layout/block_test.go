package layout

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func blockTester(p BlockProps, clicks *int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			b := Block(c, p).Size(200, 100).Label("block")
			b.Children(func() { ui.Text(c, "Card") })
			if b.Clicked() {
				*clicks++
			}
		})
	}, 300, 160)
}

func TestBlockVariants(t *testing.T) {
	tk := theme.Light()
	clicks := 0
	at := func(tt *ui.Tester, x, y float32) ui.Color { return testkit.PixelAt(tt, 300, x, y) }
	filled := blockTester(BlockProps{}, &clicks)
	if got := at(filled, 120, 100); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("filled %+v, want FillTertiary", got)
	}
	outlined := blockTester(BlockProps{Variant: BlockOutlined}, &clicks)
	if got := at(outlined, 120, 100); !testkit.Near(got, tk.BgContainer) {
		t.Errorf("outlined inside %+v, want BgContainer", got)
	}
	if got := at(outlined, 20.5, 70); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.BorderSecondary)) && !testkit.Near(got, tk.BorderSecondary) {
		t.Errorf("outlined edge %+v, want BorderSecondary", got)
	}
	borderless := blockTester(BlockProps{Variant: BlockBorderless}, &clicks)
	if got := at(borderless, 120, 100); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("borderless %+v, want no surface", got)
	}
}

func TestBlockClickableHovers(t *testing.T) {
	tk := theme.Light()
	clicks := 0
	tt := blockTester(BlockProps{Clickable: true}, &clicks)
	tt.Move(120, 100)
	tt.Advance(1e9)
	tt.Frame()
	if got := testkit.PixelAt(tt, 300, 120, 100); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillSecondary)) {
		t.Errorf("hovered clickable %+v, want FillSecondary", got)
	}
	tt.ClickAt(120, 100)
	tt.Frame()
	if clicks != 1 {
		t.Errorf("clicks %d", clicks)
	}
	plain := blockTester(BlockProps{}, &clicks)
	plain.Move(120, 100)
	plain.Advance(1e9)
	plain.Frame()
	if got := testkit.PixelAt(plain, 300, 120, 100); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("hovered plain block %+v, want it to stay FillTertiary", got)
	}
}

func TestClickableBlockLeavesNoStrayElement(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Gap(16).Children(func() {
			Block(c.Key("a"), BlockProps{Clickable: true}).Size(100, 50).Label("A")
			Block(c.Key("b"), BlockProps{}).Size(100, 50).Label("B")
		})
	}, 400, 100)
	a, _ := tt.Find("A")
	b, _ := tt.Find("B")
	if a.X != 0 || b.X != 116 {
		t.Errorf("A at %v, B at %v: want 0 and 116, one gap apart", a.X, b.X)
	}
}
