package layout

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ScrollDirection int

const (
	Vertical ScrollDirection = iota
	Horizontal
	Both
)

type ScrollAreaProps struct {
	// Direction is the axis, or both, the content scrolls along.
	Direction ScrollDirection
	// Fade fades the content out over 40 DIPs toward each edge it can scroll toward, and not toward
	// an edge it is scrolled to.
	Fade bool
	// State, if set, keeps the scroll position in the app's state, to read it or scroll by setting
	// it; the area keeps its own otherwise.
	State *ui.ScrollState
}

const (
	fadeSize      = 40
	barInset      = 3
	barHit        = 12
	thumbRest     = 3
	thumbHover    = 6
	thumbMin      = 16
	scrollHold    = 500 * time.Millisecond
	barTransition = 150 * time.Millisecond
)

type scrollBars struct{ lastX, lastY float32 }

// ScrollArea scrolls content in a box with Lobe UI's scroll bars: thin thumbs inset from the edges
// that show at once while the content scrolls and for half a second after, fade in while the
// pointer is over the box, widen under the pointer, and scroll the content as they are dragged.
// Give the returned box its size.
func ScrollArea(c *ui.Context, p ScrollAreaProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
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
	scrolling := root.AnimateTo("hold", 0, ui.Animation{Duration: scrollHold}) > 0
	size := root.Bounds()

	root.Children(func() {
		var s ui.Element
		switch p.Direction {
		case Horizontal:
			s = ui.ScrollHorizontal(c)
		case Both:
			s = ui.ScrollBoth(c)
		default:
			s = ui.Scroll(c)
		}
		s.Fill().Scrollbars(false).TrackScroll(state).Children(content)
		if p.Fade {
			fadeEdges(s, p.Direction, size, state)
		}
		if p.Direction != Horizontal && state.MaxY > 0 {
			scrollBar(c, tk, root, state, size, moved, scrolling, true)
		}
		if p.Direction != Vertical && state.MaxX > 0 {
			scrollBar(c, tk, root, state, size, moved, scrolling, false)
		}
	})
	return root
}

func scrollBar(c *ui.Context, tk *theme.Tokens, root ui.Element, state *ui.ScrollState, size ui.Rect, moved, scrolling, vertical bool) {
	view, limit, at := size.H, state.MaxY, state.Y
	strip := ui.Box(c.Key(vertical)).Absolute()
	if vertical {
		strip.Top(0).Bottom(0).Right(0).Width(barHit)
	} else {
		view, limit, at = size.W, state.MaxX, state.X
		strip.Left(0).Right(0).Bottom(0).Height(barHit)
	}
	track := view - 2*barInset
	length := min(track, max(thumbMin, track*view/(view+limit)))
	offset := barInset + (track-length)*at/limit

	strip.Children(func() {
		thumb := ui.ButtonBase(c).Absolute().Label("Scroll").Background(tk.TextQuaternary)
		dx, dy, held := thumb.Dragged()
		if held && track > length {
			if vertical {
				state.Y = clamp(state.Y+dy*limit/(track-length), 0, limit)
			} else {
				state.X = clamp(state.X+dx*limit/(track-length), 0, limit)
			}
		}

		width := float32(thumbRest)
		if strip.Hovered() || held {
			width = thumbHover
		}
		width = thumb.AnimateTo("width", width, ui.Animation{Duration: barTransition, Ease: motion.EaseOut})
		thumb.Radius(width / 2)
		if vertical {
			thumb.Right(barInset).Top(offset).Width(width).Height(length)
		} else {
			thumb.Bottom(barInset).Left(offset).Height(width).Width(length)
		}

		fade := ui.Animation{Duration: barTransition, Ease: motion.EaseOut}
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

func fadeEdges(s ui.Element, d ScrollDirection, size ui.Rect, state *ui.ScrollState) {
	vertical := fade(180, size.H, state.Y, state.MaxY-state.Y)
	horizontal := fade(90, size.W, state.X, state.MaxX-state.X)
	switch d {
	case Horizontal:
		s.MaskLinearGradient(horizontal)
	case Both:
		s.Masks(ui.Mask{Image: vertical, Composite: ui.MaskIntersect}, ui.Mask{Image: horizontal})
	default:
		s.MaskLinearGradient(vertical)
	}
}

func fade(angle, length, before, after float32) ui.LinearGradient {
	black := ui.RGB(0, 0, 0)
	if length <= 0 {
		return ui.LinearGradient{From: black, To: black}
	}
	start := min(fadeSize, max(before, 0)) / length
	end := 1 - min(fadeSize, max(after, 0))/length
	return ui.LinearGradient{Angle: angle, Stops: []ui.GradientStop{
		{Color: ui.Transparent, At: 0}, {Color: black, At: start}, {Color: black, At: end}, {Color: ui.Transparent, At: 1},
	}}
}

func clamp(v, lo, hi float32) float32 { return min(max(v, lo), hi) }
