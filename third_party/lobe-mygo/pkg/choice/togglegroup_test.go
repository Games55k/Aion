package choice

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func toggleTester(sel *[]string, p ToggleGroupProps) *ui.Tester {
	p.Items = []ToggleItem{{Value: "b", Label: "Bold"}, {Value: "i", Label: "Italic"}, {Value: "u", Label: "Underline"}}
	return ui.NewTester(testkit.Centered(func(c *ui.Context) { ToggleGroup(c, sel, p) }), 400, 100)
}

func TestToggleGroupSingle(t *testing.T) {
	sel := []string{"b"}
	tt := toggleTester(&sel, ToggleGroupProps{})
	tt.Click("Italic")
	tt.Frame()
	if !slices.Equal(sel, []string{"i"}) {
		t.Fatalf("single: %v after clicking Italic, want [i]", sel)
	}
	tt.Click("Italic")
	tt.Frame()
	if len(sel) != 0 {
		t.Errorf("single: %v after clicking Italic again, want none", sel)
	}
}

func TestToggleGroupMultiple(t *testing.T) {
	sel := []string{"b"}
	tt := toggleTester(&sel, ToggleGroupProps{Multiple: true})
	tt.Click("Underline")
	tt.Frame()
	if !slices.Equal(sel, []string{"b", "u"}) {
		t.Fatalf("multiple: %v, want [b u] in item order", sel)
	}
	tt.Click("Bold")
	tt.Frame()
	if !slices.Equal(sel, []string{"u"}) {
		t.Errorf("multiple: %v after clicking Bold, want [u]", sel)
	}
}

func TestToggleGroupPressedLook(t *testing.T) {
	tk := theme.Light()
	sel := []string{"b"}
	tt := toggleTester(&sel, ToggleGroupProps{})
	tt.Move(0, 0)
	b, _ := tt.Find("Bold")
	i, _ := tt.Find("Italic")
	if got := testkit.PixelAt(tt, 400, b.X+3, b.Y+b.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillSecondary)) {
		t.Errorf("pressed item %+v, want FillSecondary", got)
	}
	if got := testkit.PixelAt(tt, 400, i.X+3, i.Y+i.H/2); testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillSecondary)) {
		t.Errorf("unpressed item has the pressed fill")
	}
	if got := testkit.PixelAt(tt, 400, b.X+b.W+0.5, b.Y+b.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.BorderSecondary)) {
		t.Errorf("divider between items %+v, want BorderSecondary", got)
	}
}

func TestToggleGroupHugsItsItems(t *testing.T) {
	tk := theme.Light()
	sel := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Children(func() {
			ToggleGroup(c, &sel, ToggleGroupProps{Items: []ToggleItem{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}})
		})
	}, 400, 100)
	b, _ := tt.Find("B")
	if got := testkit.PixelAt(tt, 400, 399.5, b.Y+b.H/2); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("group border at the column's right edge %+v: want the group as wide as its items", got)
	}
}
