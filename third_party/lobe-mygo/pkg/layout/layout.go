package layout

import "github.com/egoist/mygo/ui"

type LayoutProps struct {
	// Header builds the bar across the top, as Header, held there as the page scrolls.
	Header func()
	// HeaderHeight is the bar's height: 64 DIPs when zero.
	HeaderHeight float32
	// Sidebar builds the navigation at the left in a DraggablePanel, scrolling on its own.
	Sidebar func()
	// AsideWidth is the sidebar's width and the most a drag gives it: 240 DIPs, unbounded,
	// when zero.
	AsideWidth float32
	// Toc builds the table of contents at the right, held at the top as the page scrolls.
	Toc func()
	// TocWidth is the width of the table of contents, and of the room kept at the left in its
	// place without a Sidebar: as wide as Toc builds when zero.
	TocWidth float32
	// Footer builds the end of the page, under the content.
	Footer func()
	// Mobile leaves out the sidebar and moves the table of contents into the header.
	Mobile bool
}

type layoutState struct {
	aside    float32
	expanded bool
}

// Layout is a documentation site's page: a header over a sidebar, the content and its table of
// contents, then a footer. The content, table of contents and footer scroll under the header.
func Layout(c *ui.Context, p LayoutProps, content func()) ui.Element {
	height := p.HeaderHeight
	if height <= 0 {
		height = 64
	}
	root := ui.Column(c).Fill()
	st := ui.Local(root, "layout", func() layoutState {
		aside := p.AsideWidth
		if aside <= 0 {
			aside = 240
		}
		return layoutState{aside: aside, expanded: true}
	})
	return root.Children(func() {
		if p.Header != nil {
			ui.Column(c).Height(height).Shrink(0).ZIndex(2).Children(func() {
				p.Header()
				if p.Mobile && p.Toc != nil {
					p.Toc()
				}
			})
		}
		ui.Row(c).Grow(1).Basis(0).AlignItems(ui.Stretch).Children(func() {
			if !p.Mobile && p.Sidebar != nil {
				DraggablePanel(c, &st.aside, &st.expanded, DraggablePanelProps{Placement: Left, MaxSize: p.AsideWidth}, func() {
					ui.Scroll(c).Fill().Children(p.Sidebar)
				})
			}
			ui.Scroll(c).Grow(1).Basis(0).Children(func() {
				ui.Row(c).AlignItems(ui.Start).Children(func() {
					if !p.Mobile && p.Sidebar == nil {
						ui.Box(c).Width(p.TocWidth).Shrink(0)
					}
					ui.Column(c).Grow(1).Basis(0).Children(content)
					if !p.Mobile && p.Toc != nil {
						toc := ui.Column(c).Shrink(0).Sticky().Top(0)
						if p.TocWidth > 0 {
							toc.Width(p.TocWidth)
						}
						toc.Children(p.Toc)
					}
				})
				if p.Footer != nil {
					ui.Column(c).Children(p.Footer)
				}
			})
		})
	})
}
