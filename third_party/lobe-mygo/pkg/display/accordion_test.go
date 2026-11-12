package display

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func accordionTester(open map[string]bool, p AccordionProps) *ui.Tester {
	p.Items = []AccordionItem{{Key: "a", Title: "Billing"}, {Key: "b", Title: "Security"}, {Key: "c", Title: "Members"}}
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(440).Padding(0, 20).Children(func() {
			Accordion(c, open, p, func(key string) { ui.Text(c, "Panel "+key) })
		})
	}, 440, 400)
}

func TestAccordionOpensAndClosesSections(t *testing.T) {
	open := map[string]bool{}
	tt := accordionTester(open, AccordionProps{})
	if tt.HasText("Panel a") {
		t.Fatal("panel shows while closed")
	}
	sec, _ := tt.Find("Security")
	if err := tt.Click("Billing"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if !open["a"] || !tt.HasText("Panel a") {
		t.Fatalf("clicking Billing: open %v, texts %q", open, tt.Texts())
	}
	if moved, _ := tt.Find("Security"); moved.Y <= sec.Y {
		t.Errorf("Security at %v after opening Billing, want pushed below %v", moved.Y, sec.Y)
	}
	if err := tt.Click("Security"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !open["a"] || !open["b"] {
		t.Errorf("sections open on their own: %v", open)
	}
	tt.Click("Billing")
	tt.Advance(time.Second)
	tt.Frame()
	if open["a"] || tt.HasText("Panel a") {
		t.Errorf("clicking Billing again: open %v", open)
	}
}

func TestAccordionSingleKeepsOneOpen(t *testing.T) {
	open := map[string]bool{}
	tt := accordionTester(open, AccordionProps{Single: true})
	tt.Click("Billing")
	tt.Advance(time.Second)
	tt.Click("Security")
	tt.Advance(time.Second)
	if open["a"] || !open["b"] {
		t.Errorf("single: %v, want only b", open)
	}
}

func TestAccordionPanelAnimatesOpen(t *testing.T) {
	open := map[string]bool{}
	tt := accordionTester(open, AccordionProps{})
	sec, _ := tt.Find("Security")
	tt.Click("Billing")
	tt.Advance(60 * time.Millisecond)
	tt.Frame()
	mid, _ := tt.Find("Security")
	tt.Advance(time.Second)
	tt.Frame()
	end, _ := tt.Find("Security")
	if !(mid.Y > sec.Y && mid.Y < end.Y) {
		t.Errorf("Security at %v 60 ms in, from %v to %v: want it on its way", mid.Y, sec.Y, end.Y)
	}
}

func TestAccordionVariants(t *testing.T) {
	tk := theme.Light()
	tt := accordionTester(map[string]bool{}, AccordionProps{Variant: AccordionFilled})
	b, _ := tt.Find("Billing")
	if got := testkit.PixelAt(tt, 440, 24, b.Y+b.H/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("filled header %+v, want FillTertiary", got)
	}
	tt = accordionTester(map[string]bool{}, AccordionProps{Variant: AccordionOutlined})
	if got := testkit.PixelAt(tt, 440, 20.5, 30); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.BorderSecondary)) {
		t.Errorf("outlined edge %+v, want BorderSecondary", got)
	}
	b, _ = tt.Find("Billing")
	s, _ := tt.Find("Security")
	if got := testkit.PixelAt(tt, 440, 200, (b.Y+b.H+s.Y)/2); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.BorderSecondary)) {
		t.Errorf("outlined divider between sections %+v, want BorderSecondary", got)
	}
}

func darkestIn(tt *ui.Tester, width int, r ui.Rect) float32 {
	d := float32(1)
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			d = min(d, luminance(testkit.PixelAt(tt, width, x, y)))
		}
	}
	return d
}

func TestAccordionContentFadesIn(t *testing.T) {
	open := map[string]bool{}
	tt := accordionTester(open, AccordionProps{})
	tt.Click("Billing")
	tt.Advance(time.Second)
	tt.Frame()
	r, _ := tt.Find("Panel a")
	settled := darkestIn(tt, 440, r)
	tt.Click("Billing")
	tt.Advance(time.Second)
	tt.Frame()
	tt.Click("Billing")
	tt.Advance(50 * time.Millisecond)
	tt.Frame()
	if r, ok := tt.Find("Panel a"); !ok || darkestIn(tt, 440, r) < settled+0.1 {
		t.Errorf("panel text 50 ms into opening is as dark as when open (%v): want it fading in", settled)
	}
}
