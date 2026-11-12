package layout

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// SideNavState tells a DraggableSideNav's content how the nav shows.
type SideNavState struct {
	// Expanded is whether to build the expanded content: true from the moment the nav starts
	// expanding, and while it collapses until Collapse reaches 1.
	Expanded bool
	// Collapse goes from 0, expanded, to 1, collapsed, along with the width, as a drag moves it
	// too: give it to List's Collapse for the rows to slide into the rail.
	Collapse float32
	// RailWidth is the content's width when collapsed.
	RailWidth float32
}

type DraggableSideNavProps struct {
	// Header and Footer build the rows above and under the body.
	Header, Footer func(nav SideNavState)
	// Right puts the nav at the right edge, resizing from its left.
	Right bool
	// MinWidth is the collapsed width and the least a drag leaves: 64 DIPs when zero.
	MinWidth float32
	// MaxWidth bounds the width a drag gives it; zero sets no bound.
	MaxWidth float32
	// NoExpand keeps the nav from collapsing: no handle, and a drag only resizes it.
	NoExpand bool
	// NoResize leaves out the seam that resizes it.
	NoResize bool
	// NoBorder leaves out the line along its inner edge.
	NoBorder bool
	// NoHandle leaves out the tab on the inner edge that collapses and expands it.
	NoHandle bool
	// ShowHandleWhenCollapsed keeps the tab shown while collapsed, not only on hover.
	ShowHandleWhenCollapsed bool
	// HighlightHandle lights the resize seam in the primary color on hover.
	HighlightHandle bool
	// Background fills the nav: the layout background when zero.
	Background ui.Color
}

type sideNavDrag struct {
	pressing bool
	raw      float32
}

const (
	sideNavCollapsed = 64
	sideNavExpanded  = 280
)

var (
	sideNavResize = ui.Animation{Duration: 300 * time.Millisecond, Ease: ui.CubicBezier(0.22, 1, 0.36, 1)}
	sideNavFade   = ui.Animation{Duration: 250 * time.Millisecond, Ease: ui.CubicBezier(0.22, 1, 0.36, 1)}
)

// DraggableSideNav is an app's side navigation, *width DIPs wide (280 when zero) while *expanded is
// true and MinWidth wide while not. The tab on its inner edge toggles it; dragging that edge
// resizes it, and leaving it under a third of the way from MinWidth to *width collapses it. body
// builds the content, told how the nav shows: on collapse the expanded content stays until the
// width is down, kept at its width and clipped.
func DraggableSideNav(c *ui.Context, width *float32, expanded *bool, p DraggableSideNavProps, body func(nav SideNavState)) ui.Element {
	tk := theme.TokensOf(c)
	if *width <= 0 {
		*width = sideNavExpanded
	}
	minW := p.MinWidth
	if minW <= 0 {
		minW = sideNavCollapsed
	}
	bg := p.Background
	if bg == (ui.Color{}) {
		bg = tk.BgLayout
	}

	root := ui.Box(c).Shrink(0).AlignSelf(ui.Stretch).ZIndex(1)
	drag := ui.Local(root, "drag", func() sideNavDrag { return sideNavDrag{} })
	var border float32
	if !p.NoBorder {
		border = 1
	}
	collapse := sideNavCollapse(root, drag, *width, *expanded, minW, p)
	shown := *expanded || (!drag.pressing && collapse < 1)
	nav := SideNavState{Expanded: shown, RailWidth: minW - border, Collapse: collapse}

	var dx, contentW float32
	var held bool
	var content ui.Element
	root.Children(func() {
		frame := ui.Row(c).Fill().Clip().Background(bg)
		if !p.NoBorder {
			if p.Right {
				frame.BorderWidth(0, 0, 0, 1)
			} else {
				frame.BorderWidth(0, 1, 0, 0)
			}
			frame.BorderColor(tk.BorderSecondary)
		}
		frame.Children(func() {
			// The content keeps the width of the state it shows while the nav animates, clipped
			// rather than squeezed, so its rows neither wrap nor ellipsize on the way.
			content = ui.Column(c).AlignSelf(ui.Stretch)
			switch {
			case drag.pressing:
				content.Grow(1)
			case shown:
				contentW = *width - border
			default:
				contentW = minW - border
			}
			if contentW > 0 {
				content.Width(contentW)
			}
			content.Children(func() {
				if p.Header != nil {
					ui.Column(c).Shrink(0).Children(func() { p.Header(nav) })
				}
				ui.Scroll(c).Grow(1).Basis(0).Children(func() { body(nav) })
				if p.Footer != nil {
					ui.Column(c).Shrink(0).Children(func() { p.Footer(nav) })
				}
			})
		})
		if !p.NoResize {
			dx, held = sideNavSeam(c, tk, root, p)
		}
		if !p.NoHandle && !p.NoExpand {
			sideNavHandle(c, tk, root, expanded, bg, p)
		}
	})

	if p.Right {
		dx = -dx
	}
	resizeSideNav(dx, held, width, expanded, drag, minW, p)

	target := minW
	if *expanded {
		target = *width
	}
	var current float32
	if drag.pressing {
		current = root.Jump("width", clampSideNav(drag.raw, minW, p.MaxWidth))
	} else {
		current = root.AnimateTo("width", target, sideNavResize)
	}
	if p.Right && contentW > 0 {
		content.Margin(0, 0, 0, min(0, current-border-contentW))
	}
	return root.Width(current)
}

func sideNavCollapse(root ui.Element, drag *sideNavDrag, width float32, expanded bool, minW float32, p DraggableSideNavProps) float32 {
	if drag.pressing {
		w := clampSideNav(drag.raw, minW, p.MaxWidth)
		return root.Jump("collapse", min(max(1-(w-minW)/max(width-minW, 1), 0), 1))
	}
	target := float32(1)
	if expanded {
		target = 0
	}
	return root.AnimateTo("collapse", target, sideNavResize)
}

func clampSideNav(v, minW, maxW float32) float32 {
	v = max(v, minW)
	if maxW > 0 {
		v = min(v, maxW)
	}
	return v
}

func resizeSideNav(dx float32, held bool, width *float32, expanded *bool, d *sideNavDrag, minW float32, p DraggableSideNavProps) {
	if held && !d.pressing {
		d.pressing, d.raw = true, minW
		if *expanded {
			d.raw = *width
		}
	}
	if held {
		d.raw += dx
		return
	}
	if !d.pressing {
		return
	}
	d.pressing = false
	w := clampSideNav(d.raw, minW, p.MaxWidth)
	if p.NoExpand {
		*width = w
		return
	}
	threshold := minW + (*width-minW)/3
	switch {
	case d.raw <= minW || d.raw < threshold:
		*expanded = false
	default:
		*expanded, *width = true, w
	}
}

func sideNavSeam(c *ui.Context, tk *theme.Tokens, root ui.Element, p DraggableSideNavProps) (float32, bool) {
	seam := ui.ButtonBase(c).Absolute().Top(0).Bottom(0).Width(8).Role(ui.RoleSeparator).
		Label("Resize side nav").Cursor(ui.CursorResizeColumn)
	if p.Right {
		seam.Left(-4)
	} else {
		seam.Right(-4)
	}
	dx, _, held := seam.Dragged()
	line, thick := ui.Color{}, float32(2)
	if p.HighlightHandle {
		switch {
		case held:
			line = tk.PrimaryActive
		case seam.Hovered():
			line, thick = tk.Primary, 3
		}
	}
	color := root.AnimateColor("seam", line, sideNavFade)
	seam.Children(func() {
		ui.Box(c).Absolute().Top(0).Bottom(0).Left((8 - thick) / 2).Width(thick).IgnoreInput(true).Background(color)
	})
	return dx, held
}

func sideNavHandle(c *ui.Context, tk *theme.Tokens, root ui.Element, expanded *bool, bg ui.Color, p DraggableSideNavProps) {
	label, icon := "Collapse side nav", lucide.ChevronLeft()
	if p.Right {
		icon = lucide.ChevronRight()
	}
	if !*expanded {
		label = "Expand side nav"
	}
	b := ui.ButtonBase(c).Label(label).Absolute().TopPercent(50).TranslatePercent(0, -50).Size(16, 54).
		Background(bg).BorderColor(tk.Border).Cursor(ui.CursorPointer).ZIndex(50)
	if p.Right {
		b.Left(-15).Radius(8, 0, 0, 8).BorderWidth(1, 0, 1, 1)
	} else {
		b.Right(-15).Radius(0, 8, 8, 0).BorderWidth(1, 1, 1, 0)
	}
	visible := float32(0)
	if root.Hovered() || b.Hovered() || b.FocusVisible() || (!*expanded && p.ShowHandleWhenCollapsed) {
		visible = 1
	}
	b.Opacity(b.AnimateTo("shown", visible, sideNavFade))
	glyph := tk.TextTertiary
	switch {
	case b.Pressed():
		glyph = tk.Text
		b.Scale(0.95, 0.95)
	case b.Hovered():
		glyph = tk.TextSecondary
	}
	turn := float32(0)
	if !*expanded {
		turn = 180
	}
	angle := b.AnimateTo("turn", turn, ui.Animation{Duration: 200 * time.Millisecond, Ease: motion.EaseOut})
	b.Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(glyph).Rotate(angle)
	})
	if b.Clicked() {
		*expanded = !*expanded
	}
}
