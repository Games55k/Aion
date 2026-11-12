package feedback

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type FloatingPanelPlacement int

const (
	FloatingBottomRight FloatingPanelPlacement = iota
	FloatingBottomLeft
	FloatingTopRight
	FloatingTopLeft
)

type FloatingPanelProps struct {
	// Title shows in the header and names the panel.
	Title string
	// Actions builds buttons in the header before the close button, 4 DIPs apart.
	Actions func()
	// Footer builds a row under the content, over a line.
	Footer func()
	// Placement is the corner of the window the panel sits in.
	Placement FloatingPanelPlacement
	// Offset is how far in from the corner it sits: 8 DIPs when zero.
	Offset float32
	// Width is the panel's width, at most the window's less 32: 640 DIPs when zero.
	Width float32
	// Height is the panel's height: as tall as its content when zero, at most the window's less 32.
	Height float32
	// MinWidth and MinHeight bound the size a drag gives it: 320 and 180 DIPs when zero.
	MinWidth, MinHeight float32
	// MaxWidth and MaxHeight bound the size a drag gives it: the window's less 16 when zero.
	MaxWidth, MaxHeight float32
	// NoResize leaves out the edges that resize it. Those facing away from its corner do.
	NoResize bool
	// NoClose leaves out the close button.
	NoClose bool
	// NoKeyboard keeps Escape from closing it.
	NoKeyboard bool
	// NoOutsideClose keeps a press outside it from closing it.
	NoOutsideClose bool
	// Mask dims the window behind the panel and takes its presses.
	Mask bool
	// OnResize gets the size a drag gives it, as it changes.
	OnResize func(width, height float32)
}

type floatEdge struct {
	left, right, top, bottom bool
}

type floatState struct {
	w, h           float32
	pressing       bool
	startW, startH float32
	accX, accY     float32
}

var (
	floatIn  = ui.Animation{Duration: 200 * time.Millisecond, Ease: ui.CubicBezier(0.32, 0.72, 0, 1)}
	floatOut = ui.Animation{Duration: 140 * time.Millisecond, Ease: ui.CubicBezier(0.4, 0, 1, 1)}
)

func (pl FloatingPanelPlacement) top() bool { return pl == FloatingTopLeft || pl == FloatingTopRight }
func (pl FloatingPanelPlacement) left() bool {
	return pl == FloatingTopLeft || pl == FloatingBottomLeft
}

func floatEdges(pl FloatingPanelPlacement) []floatEdge {
	v, h := floatEdge{top: true}, floatEdge{left: true}
	if pl.top() {
		v = floatEdge{bottom: true}
	}
	if pl.left() {
		h = floatEdge{right: true}
	}
	return []floatEdge{v, h, {left: h.left, right: h.right, top: v.top, bottom: v.bottom}}
}

// FloatingPanel shows content in a panel floating at a corner of the window while *open is true,
// beside the page rather than over it: the page stays usable, and a press on it, Escape or the
// close button closes the panel. Its free edges resize it.
func FloatingPanel(c *ui.Context, open *bool, p FloatingPanelProps, content func()) {
	tk := theme.TokensOf(c)
	holder := ui.Box(c).Absolute().Size(0, 0)
	st := ui.Local(holder, "floating", func() floatState { return floatState{} })
	target, anim := float32(0), floatOut
	if *open {
		target, anim = 1, floatIn
	}
	shown := holder.AnimateTo("shown", target, anim)
	fade := motion.Backdrop()
	dim := holder.AnimateTo("dim", target, ui.Animation{Duration: fade.Duration, Ease: fade.Ease})
	if !*open && shown <= 0 && dim <= 0 {
		return
	}
	offset := p.Offset
	if offset <= 0 {
		offset = 8
	}
	far := max(0, 32-offset)
	ui.Overlay(c, func() {
		if p.Mask {
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Background(tk.BgMask.Alpha(dim)).IgnoreInput(!*open)
		}
		frame := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).PassThrough()
		if p.Placement.top() {
			frame.Padding(offset, 0, far, 0)
		} else {
			frame.Padding(far, 0, offset, 0).Justify(ui.End)
		}
		if p.Placement.left() {
			frame.PaddingStart(offset).PaddingEnd(far).AlignItems(ui.Start)
		} else {
			frame.PaddingStart(far).PaddingEnd(offset).AlignItems(ui.End)
		}
		frame.Children(func() {
			floatingBody(c, tk, frame, open, st, shown, p, content)
		})
	})
}

func floatingBody(c *ui.Context, tk *theme.Tokens, frame ui.Element, open *bool, st *floatState, shown float32, p FloatingPanelProps, content func()) {
	minW, minH := orDefault(p.MinWidth, 320), orDefault(p.MinHeight, 180)
	panel := ui.Box(c).Role(ui.RoleDialog).Label(p.Title).Shrink(1).
		Width(orDefault(p.Width, 640)).MaxWidthPercent(100).MaxHeightPercent(100).MinWidth(minW).MinHeight(minH)
	switch {
	case st.w > 0:
		panel.Width(st.w).Height(st.h)
	case p.Height > 0:
		panel.Height(p.Height)
	}
	rise := float32(12)
	origin := float32(1)
	if p.Placement.top() {
		rise, origin = -12, 0
	}
	scale := 0.98 + 0.02*shown
	panel.TransformOrigin(1, origin).Scale(scale, scale).Translate(0, rise*(1-shown)).Opacity(shown)
	if !*open {
		panel.IgnoreInput(true)
	} else {
		if !p.NoKeyboard && panel.OverlayShortcut(0, ui.KeyEscape) {
			*open = false
		}
		if !p.NoOutsideClose && panel.PressedOutside() {
			*open = false
		}
	}
	tk.Elevate(panel, theme.BoxShadowSecondary)
	panel.Radius(12).Children(func() {
		ui.Column(c).Fill().Radius(12).Clip().Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).Children(func() {
			floatHeader(c, tk, open, p)
			ui.Scroll(c).Grow(1).Shrink(1).Children(content)
			if p.Footer != nil {
				ui.Row(c).Shrink(0).Gap(8).Padding(12, 16).AlignItems(ui.Center).Justify(ui.End).
					BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(p.Footer)
			}
		})
		if !p.NoResize {
			floatResize(c, tk, panel, frame, st, minW, minH, p)
		}
	})
}

func floatHeader(c *ui.Context, tk *theme.Tokens, open *bool, p FloatingPanelProps) {
	ui.Row(c).Shrink(0).MinHeight(48).Padding(8, 16).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		ui.Text(c, p.Title).FontSize(14).FontWeight(600).SingleLine().Grow(1).Basis(0)
		ui.Row(c).Shrink(0).Gap(4).AlignItems(ui.Center).MarginEnd(-4).Children(func() {
			if p.Actions != nil {
				p.Actions()
			}
			if !p.NoClose && dialogClose(c, tk, 16).Clicked() {
				*open = false
			}
		})
	})
}

func floatResize(c *ui.Context, tk *theme.Tokens, panel, frame ui.Element, st *floatState, minW, minH float32, p FloatingPanelProps) {
	window := frame.LayoutBounds()
	maxW, maxH := orDefault(p.MaxWidth, window.W-16), orDefault(p.MaxHeight, window.H-16)
	anyHeld := false
	for i, e := range floatEdges(p.Placement) {
		dx, dy, held := floatHandle(c.Key(i), tk, e)
		if !held {
			continue
		}
		anyHeld = true
		if !st.pressing {
			b := panel.LayoutBounds()
			st.pressing, st.startW, st.startH, st.accX, st.accY = true, b.W, b.H, 0, 0
		}
		st.accX += dx
		st.accY += dy
		w, h := st.startW, st.startH
		switch {
		case e.left:
			w -= st.accX
		case e.right:
			w += st.accX
		}
		switch {
		case e.top:
			h -= st.accY
		case e.bottom:
			h += st.accY
		}
		st.w, st.h = clampFloat(w, minW, max(minW, maxW)), clampFloat(h, minH, max(minH, maxH))
		if p.OnResize != nil {
			p.OnResize(st.w, st.h)
		}
	}
	if !anyHeld {
		st.pressing = false
	}
}

func floatHandle(c *ui.Context, tk *theme.Tokens, e floatEdge) (float32, float32, bool) {
	h := ui.Box(c).Absolute().ZIndex(1)
	corner := (e.left || e.right) && (e.top || e.bottom)
	switch {
	case corner:
		h.Size(16, 16)
		if e.left {
			h.Left(-5)
		} else {
			h.Right(-5)
		}
		if e.top {
			h.Top(-5)
		} else {
			h.Bottom(-5)
		}
		if e.left == e.top {
			h.Cursor(ui.CursorResizeNWSE)
		} else {
			h.Cursor(ui.CursorResizeNESW)
		}
	case e.top || e.bottom:
		h.Left(16).Right(16).Height(8).Cursor(ui.CursorResizeNS)
		if e.top {
			h.Top(-4)
		} else {
			h.Bottom(-4)
		}
	default:
		h.Top(16).Bottom(16).Width(8).Cursor(ui.CursorResizeEW)
		if e.left {
			h.Left(-4)
		} else {
			h.Right(-4)
		}
	}
	dx, dy, held := h.Dragged()
	if !corner {
		lit := float32(0)
		if held || h.Hovered() {
			lit = 0.55
		}
		alpha := h.AnimateTo("lit", lit, ui.Animation{Duration: 120 * time.Millisecond, Ease: ui.Linear})
		h.Children(func() {
			line := ui.Box(c).Absolute().Radius(999).IgnoreInput(true).Background(tk.Primary.Alpha(alpha))
			switch {
			case e.top:
				line.Left(0).Right(0).Top(3).Height(2)
			case e.bottom:
				line.Left(0).Right(0).Bottom(3).Height(2)
			case e.left:
				line.Top(0).Bottom(0).Left(3).Width(2)
			default:
				line.Top(0).Bottom(0).Right(3).Width(2)
			}
		})
	}
	return dx, dy, held
}

func orDefault(v, fallback float32) float32 {
	if v > 0 {
		return v
	}
	return fallback
}

func clampFloat(v, lo, hi float32) float32 {
	return min(max(v, lo), hi)
}
