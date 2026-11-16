package frontend

import (
	"time"

	"github.com/egoist/mygo/ui"
)

type sideNavState struct {
	Expanded  bool
	Collapse  float32
	RailWidth float32
}

type draggableSideNavProps struct {
	Header, Footer          func(nav sideNavState)
	Right                   bool
	MinWidth                float32
	MaxWidth                float32
	NoExpand                bool
	NoResize                bool
	NoBorder                bool
	NoHandle                bool
	ShowHandleWhenCollapsed bool
	HighlightHandle         bool
	Background              ui.Color
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

func uiDraggableSideNav(c *ui.Context, width *float32, expanded *bool, p draggableSideNavProps, body func(nav sideNavState)) ui.Element {
	tk := tokensOf(c)
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
	nav := sideNavState{Expanded: shown, RailWidth: minW - border, Collapse: collapse}

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

func sideNavCollapse(root ui.Element, drag *sideNavDrag, width float32, expanded bool, minW float32, p draggableSideNavProps) float32 {
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

func resizeSideNav(dx float32, held bool, width *float32, expanded *bool, d *sideNavDrag, minW float32, p draggableSideNavProps) {
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

func sideNavSeam(c *ui.Context, tk *uiTokens, root ui.Element, p draggableSideNavProps) (float32, bool) {
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

func sideNavHandle(c *ui.Context, tk *uiTokens, root ui.Element, expanded *bool, bg ui.Color, p draggableSideNavProps) {
	label, icon := "Collapse side nav", iconChevronLeft()
	if p.Right {
		icon = iconChevronRight()
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
	angle := b.AnimateTo("turn", turn, ui.Animation{Duration: 200 * time.Millisecond, Ease: easeOut})
	b.Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(glyph).Rotate(angle)
	})
	if b.Clicked() {
		*expanded = !*expanded
	}
}

type scrollDirection int

const (
	scrollVertical scrollDirection = iota
	scrollHorizontal
	scrollBoth
)

type scrollAreaProps struct {
	Direction scrollDirection
	Fade      bool
	State     *ui.ScrollState
}

const (
	scrollFadeSize      = 40
	scrollBarInset      = 3
	scrollBarHit        = 12
	scrollThumbRest     = 3
	scrollThumbHover    = 6
	scrollThumbMin      = 16
	scrollHoldDuration  = 500 * time.Millisecond
	scrollBarTransition = 150 * time.Millisecond
)

type scrollBars struct{ lastX, lastY float32 }

func uiScrollArea(c *ui.Context, p scrollAreaProps, content func()) ui.Element {
	tk := tokensOf(c)
	root := ui.Box(c)
	bars := ui.Local(root, "bars", func() scrollBars { return scrollBars{} })
	state := p.State
	if state == nil {
		state = ui.Local(root, "scroll", func() ui.ScrollState { return ui.ScrollState{} })
	}
	moved := state.X != bars.lastX || state.Y != bars.lastY
	bars.lastX, bars.lastY = state.X, state.Y
	if moved {
		root.Jump("hold", 1)
	}
	scrolling := root.AnimateTo("hold", 0, ui.Animation{Duration: scrollHoldDuration}) > 0
	size := root.Bounds()

	root.Children(func() {
		var s ui.Element
		switch p.Direction {
		case scrollHorizontal:
			s = ui.ScrollHorizontal(c)
		case scrollBoth:
			s = ui.ScrollBoth(c)
		default:
			s = ui.Scroll(c)
		}
		s.Fill().Scrollbars(false).TrackScroll(state).Children(content)
		if p.Fade {
			fadeEdges(s, p.Direction, size, state)
		}
		if p.Direction != scrollHorizontal && state.MaxY > 0 {
			scrollBar(c, tk, root, state, size, moved, scrolling, true)
		}
		if p.Direction != scrollVertical && state.MaxX > 0 {
			scrollBar(c, tk, root, state, size, moved, scrolling, false)
		}
	})
	return root
}

func scrollBar(c *ui.Context, tk *uiTokens, root ui.Element, state *ui.ScrollState, size ui.Rect, moved, scrolling, vertical bool) {
	view, limit, at := size.H, state.MaxY, state.Y
	strip := ui.Box(c.Key(vertical)).Absolute()
	if vertical {
		strip.Top(0).Bottom(0).Right(0).Width(scrollBarHit)
	} else {
		view, limit, at = size.W, state.MaxX, state.X
		strip.Left(0).Right(0).Bottom(0).Height(scrollBarHit)
	}
	track := view - 2*scrollBarInset
	length := min(track, max(scrollThumbMin, track*view/(view+limit)))
	offset := scrollBarInset + (track-length)*at/limit

	strip.Children(func() {
		thumb := ui.ButtonBase(c).Absolute().Label("Scroll").Background(tk.TextQuaternary)
		dx, dy, held := thumb.Dragged()
		if held && track > length {
			if vertical {
				state.Y = clampScroll(state.Y+dy*limit/(track-length), 0, limit)
			} else {
				state.X = clampScroll(state.X+dx*limit/(track-length), 0, limit)
			}
		}

		width := float32(scrollThumbRest)
		if strip.Hovered() || held {
			width = scrollThumbHover
		}
		width = thumb.AnimateTo("width", width, ui.Animation{Duration: scrollBarTransition, Ease: easeOut})
		thumb.Radius(width / 2)
		if vertical {
			thumb.Right(scrollBarInset).Top(offset).Width(width).Height(length)
		} else {
			thumb.Bottom(scrollBarInset).Left(offset).Height(width).Width(length)
		}

		fade := ui.Animation{Duration: scrollBarTransition, Ease: easeOut}
		var shown float32
		switch {
		case moved || held:
			shown = thumb.Jump("shown", 1)
		case scrolling || root.Hovered():
			shown = thumb.AnimateTo("shown", 1, fade)
		default:
			shown = thumb.AnimateTo("shown", 0, fade)
		}
		thumb.Opacity(shown)
		strip.IgnoreInput(shown == 0 && !root.Hovered())
	})
}

func fadeEdges(s ui.Element, d scrollDirection, size ui.Rect, state *ui.ScrollState) {
	vertical := edgeFadeGradient(180, size.H, state.Y, state.MaxY-state.Y)
	horizontal := edgeFadeGradient(90, size.W, state.X, state.MaxX-state.X)
	switch d {
	case scrollHorizontal:
		s.MaskLinearGradient(horizontal)
	case scrollBoth:
		s.Masks(ui.Mask{Image: vertical, Composite: ui.MaskIntersect}, ui.Mask{Image: horizontal})
	default:
		s.MaskLinearGradient(vertical)
	}
}

func edgeFadeGradient(angle, length, before, after float32) ui.LinearGradient {
	black := ui.RGB(0, 0, 0)
	if length <= 0 {
		return ui.LinearGradient{From: black, To: black}
	}
	start := min(scrollFadeSize, max(before, 0)) / length
	end := 1 - min(scrollFadeSize, max(after, 0))/length
	return ui.LinearGradient{Angle: angle, Stops: []ui.GradientStop{
		{Color: ui.Transparent, At: 0}, {Color: black, At: start}, {Color: black, At: end}, {Color: ui.Transparent, At: 1},
	}}
}

func clampScroll(v, lo, hi float32) float32 { return min(max(v, lo), hi) }
