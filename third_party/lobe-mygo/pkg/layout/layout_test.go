package layout

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func layoutTester(p LayoutProps, body float32) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		q := p
		if p.Header != nil {
			q.Header = func() { ui.Box(c).Fill().Label("header") }
		}
		if p.Sidebar != nil {
			q.Sidebar = func() { ui.Text(c, "Side") }
		}
		if p.Toc != nil {
			q.Toc = func() { ui.Text(c, "Toc") }
		}
		if p.Footer != nil {
			q.Footer = func() { ui.Box(c).Height(40).Label("footer") }
		}
		Layout(c, q, func() { ui.Box(c).Height(body).Label("content") })
	}, 1000, 600)
}

var allSlots = LayoutProps{Header: func() {}, Sidebar: func() {}, Toc: func() {}, Footer: func() {}}

func bounds(t *testing.T, tt *ui.Tester, name string) ui.Rect {
	t.Helper()
	info, ok := tt.Access(name)
	if !ok {
		t.Fatalf("no %s", name)
	}
	return info.Bounds
}

func TestLayoutDesktop(t *testing.T) {
	p := allSlots
	p.TocWidth = 180
	tt := layoutTester(p, 300)
	h := bounds(t, tt, "header")
	if h.Y != 0 || h.H != 64 || h.W != 1000 {
		t.Errorf("header %+v, want 1000 × 64 at the top", h)
	}
	side, ok := tt.Find("Side")
	if !ok || side.X > 4 || side.Y < 64 {
		t.Errorf("sidebar %+v, want at the left under the header", side)
	}
	content := bounds(t, tt, "content")
	toc, _ := tt.Find("Toc")
	if content.Y != 64 || content.X != 240 {
		t.Errorf("content %+v, want right of the sidebar under the header", content)
	}
	if toc.X != 1000-180 || content.X+content.W != toc.X {
		t.Errorf("toc at %v, content ends %v: want a 180 DIP toc at the right", toc.X, content.X+content.W)
	}
	foot := bounds(t, tt, "footer")
	if foot.Y != 364 || foot.X != content.X {
		t.Errorf("footer %+v, want under the content", foot)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Collapse panel"); !ok {
		t.Error("the sidebar does not collapse: want it in a DraggablePanel")
	}
}

func TestLayoutHeaderHeightAndAsideWidth(t *testing.T) {
	p := allSlots
	p.HeaderHeight, p.AsideWidth = 80, 260
	tt := layoutTester(p, 300)
	if h := bounds(t, tt, "header"); h.H != 80 {
		t.Errorf("header %v tall, want 80", h.H)
	}
	if c := bounds(t, tt, "content"); c.Y != 80 || c.X != 260 {
		t.Errorf("content %+v, want at 260, 80", c)
	}
}

func TestLayoutScrollsUnderAFixedHeader(t *testing.T) {
	tt := layoutTester(allSlots, 2000)
	before := bounds(t, tt, "content")
	tt.Scroll(500, 300, 0, 400)
	tt.Frame()
	after := bounds(t, tt, "content")
	if after.Y >= before.Y {
		t.Errorf("content at %v after scrolling, was %v", after.Y, before.Y)
	}
	if h := bounds(t, tt, "header"); h.Y != 0 {
		t.Errorf("header scrolled to %v", h.Y)
	}
	if toc, _ := tt.Find("Toc"); toc.H == 0 || toc.Y < 64 || toc.Y > 70 {
		t.Errorf("toc at %v after scrolling, want it held under the header", toc.Y)
	}
	if side, _ := tt.Find("Side"); side.H == 0 || side.Y < 64 || side.Y > 76 {
		t.Errorf("sidebar at %v after scrolling, want it held", side.Y)
	}
}

func TestLayoutWithoutSidebarKeepsTheTocRoomAtTheStart(t *testing.T) {
	p := LayoutProps{Toc: func() {}, TocWidth: 200}
	tt := layoutTester(p, 300)
	c := bounds(t, tt, "content")
	if c.X != 200 || c.W != 600 || c.Y != 0 {
		t.Errorf("content %+v, want centered between 200 DIP columns", c)
	}
}

func TestLayoutMobile(t *testing.T) {
	p := allSlots
	p.Mobile = true
	tt := layoutTester(p, 300)
	if tt.HasText("Side") {
		t.Error("a mobile layout shows the sidebar")
	}
	c := bounds(t, tt, "content")
	if c.X != 0 || c.W != 1000 {
		t.Errorf("content %+v, want the full width", c)
	}
	toc, ok := tt.Find("Toc")
	if !ok || toc.Y >= 64 {
		t.Errorf("toc %+v, want inside the header", toc)
	}
}

func TestLayoutAsideWidthBoundsTheDrag(t *testing.T) {
	p := allSlots
	p.AsideWidth = 260
	tt := layoutTester(p, 300)
	drag(tt, 260, 400, 300)
	if c := bounds(t, tt, "content"); c.X != 260 {
		t.Errorf("content at %v after dragging the sidebar past AsideWidth, want 260", c.X)
	}
}

func TestLayoutSidebarScrollsOnItsOwn(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		Layout(c, LayoutProps{Sidebar: func() {
			ui.Text(c, "Side")
			ui.Box(c).Height(2000)
		}}, func() { ui.Box(c).Height(300).Label("content") })
	}, 1000, 600)
	tt.Scroll(100, 300, 0, 400)
	tt.Frame()
	if side, _ := tt.Find("Side"); side.H > 0 && side.Y >= 0 {
		t.Errorf("sidebar text at %+v after scrolling the sidebar, want scrolled away", side)
	}
	if c := bounds(t, tt, "content"); c.Y != 0 {
		t.Errorf("content moved to %v", c.Y)
	}
}
