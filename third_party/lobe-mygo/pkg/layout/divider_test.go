package layout

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func columnTester(view func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(300).Padding(0).Children(func() { view(c) })
	}, 300, 200)
}

func TestDividerSpansItsColumn(t *testing.T) {
	tk := theme.Light()
	tt := columnTester(func(c *ui.Context) {
		ui.Text(c, "above")
		Divider(c, DividerProps{})
		ui.Text(c, "below")
	})
	a, _ := tt.Find("above")
	b, _ := tt.Find("below")
	if gap := b.Y - (a.Y + a.H); gap != 1 {
		t.Fatalf("divider takes %v DIPs, want 1", gap)
	}
	want := testkit.Over(tk.BgLayout, tk.BorderSecondary)
	for _, x := range []float32{2, 150, 298} {
		if got := testkit.PixelAt(tt, 300, x, a.Y+a.H+0.5); !testkit.Near(got, want) {
			t.Errorf("line at x %v = %+v, want BorderSecondary %+v", x, got, want)
		}
	}
}

func TestDividerDashedLeavesGaps(t *testing.T) {
	tk := theme.Light()
	tt := columnTester(func(c *ui.Context) {
		ui.Text(c, "above")
		Divider(c, DividerProps{Dashed: true})
	})
	a, _ := tt.Find("above")
	line, gaps := 0, 0
	for x := float32(0.5); x < 60; x++ {
		if testkit.Near(testkit.PixelAt(tt, 300, x, a.Y+a.H+0.5), tk.BgLayout) {
			gaps++
		} else {
			line++
		}
	}
	if line < 20 || gaps < 20 {
		t.Errorf("dashed line over 60 DIPs: %d painted, %d gaps, want both", line, gaps)
	}
}

func TestDividerLabelSitsCenteredBetweenLines(t *testing.T) {
	tk := theme.Light()
	tt := columnTester(func(c *ui.Context) { Divider(c, DividerProps{Label: "OR"}) })
	r, ok := tt.Find("OR")
	if !ok {
		t.Fatal("no label")
	}
	if mid := r.X + r.W/2; mid < 149 || mid > 151 {
		t.Errorf("label centered at %v, want 150", mid)
	}
	want := testkit.Over(tk.BgLayout, tk.BorderSecondary)
	y := r.Y + r.H/2
	for _, x := range []float32{4, r.X - 13, r.X + r.W + 13, 296} {
		if got := testkit.PixelAt(tt, 300, x, y); !testkit.Near(got, want) {
			t.Errorf("line at x %v = %+v, want BorderSecondary", x, got)
		}
	}
	if got := testkit.PixelAt(tt, 300, r.X-6, y); testkit.Near(got, want) {
		t.Errorf("line runs into the label's 12 DIP gap at x %v", r.X-6)
	}
}

func TestDividerVerticalSitsBetweenInlineItems(t *testing.T) {
	tk := theme.Light()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "Edit")
			Divider(c, DividerProps{Vertical: true})
			ui.Text(c, "View")
		})
	}, 300, 100)
	e, _ := tt.Find("Edit")
	v, _ := tt.Find("View")
	if gap := v.X - (e.X + e.W); gap != 17 {
		t.Fatalf("items %v DIPs apart, want 8 + 1 + 8", gap)
	}
	x, mid := e.X+e.W+8.5, e.Y+e.H/2
	if got, want := testkit.PixelAt(tt, 300, x, mid), testkit.Over(tk.BgLayout, tk.BorderSecondary); !testkit.Near(got, want) {
		t.Errorf("vertical line = %+v, want BorderSecondary %+v", got, want)
	}
}
