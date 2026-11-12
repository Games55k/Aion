package feedback

import (
	"math"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type FloatingSheetProps struct {
	// Title shows in the header, beside HeaderActions.
	Title string
	// HeaderActions builds buttons at the end of the header; a drag starting on them does not
	// resize the sheet.
	HeaderActions func()
	// SnapPoints are the heights a drag settles at: up to 1 a fraction of the container's height,
	// above it DIPs. The sheet rests at the lowest; a quick flick goes to the next one.
	SnapPoints []float32
	// ActiveSnapPoint is the snap point the sheet rests at, one of SnapPoints, as
	// OnSnapPointChange reported it.
	ActiveSnapPoint float32
	// OnSnapPointChange gets the snap point a drag settled at.
	OnSnapPointChange func(snapPoint float32)
	// MinHeight is the least height a drag leaves: 200 DIPs when zero, a fraction of the
	// container's height up to 1.
	MinHeight float32
	// MaxHeight is the most: 0.8 of the container's height when zero.
	MaxHeight float32
	// RestingHeight is the height it opens at, between the two: MinHeight when zero.
	RestingHeight float32
	// CloseThreshold closes the sheet when a drag leaves it under this fraction of MinHeight, or of
	// the lowest snap point: 0.25 when zero.
	CloseThreshold float32
	// NoDismiss keeps a drag from closing it.
	NoDismiss bool
	// Inline puts the sheet in its container's flow under the content, which keeps the room it
	// rests at; dragged taller, it covers the content. Otherwise it floats over the content.
	Inline bool
	// Embedded draws a border instead of a shadow, for a sheet inside a panel.
	Embedded bool
	// Width is the sheet's width: the container's when zero.
	Width float32
}

type sheetState struct {
	probed, measured     bool
	wasOpen              bool
	height, lastResting  float32
	pressing, fromAction bool
	startH, dist, dragH  float32
	startT               time.Time
}

const (
	sheetVelocity  = 0.4
	sheetFlickPart = 0.4
)

var sheetMotion = ui.Animation{Duration: 300 * time.Millisecond, Ease: ui.CubicBezier(0.32, 0.72, 0, 1)}

type sheetSizes struct {
	minH, maxH, resting, base float32
	snaps                     []float32
	resolve                   func(v float32) float32
}

func sheetSizesOf(p FloatingSheetProps, container float32) sheetSizes {
	resolve := func(v float32) float32 {
		if v > 0 && v <= 1 {
			return v * container
		}
		return v
	}
	s := sheetSizes{resolve: resolve, minH: resolve(orDefault(p.MinHeight, 200)), maxH: resolve(orDefault(p.MaxHeight, 0.8))}
	s.base = s.minH
	if p.RestingHeight > 0 {
		s.base = clampFloat(resolve(p.RestingHeight), s.minH, s.maxH)
	}
	for _, sp := range p.SnapPoints {
		h := clampFloat(resolve(sp), s.minH, s.maxH)
		if !slices.Contains(s.snaps, h) {
			s.snaps = append(s.snaps, h)
		}
	}
	slices.Sort(s.snaps)
	switch {
	case container <= 0:
	case len(s.snaps) > 0 && p.ActiveSnapPoint > 0:
		s.resting = clampFloat(resolve(p.ActiveSnapPoint), s.minH, s.maxH)
	case len(s.snaps) > 0:
		s.resting = s.snaps[0]
	default:
		s.resting = s.base
	}
	return s
}

func (s sheetSizes) closest(h float32) int {
	best := 0
	for i, v := range s.snaps {
		if abs32(v-h) < abs32(s.snaps[best]-h) {
			best = i
		}
	}
	return best
}

func (s sheetSizes) drag(h float32) float32 {
	if len(s.snaps) == 0 {
		return clampFloat(h, 0, s.maxH)
	}
	hi, lo := s.snaps[len(s.snaps)-1], s.snaps[0]
	switch {
	case h > hi:
		return hi + dampen(h-hi)
	case h < lo:
		return max(0, lo-dampen(lo-h))
	}
	return h
}

func dampen(v float32) float32 {
	return float32(8 * math.Log1p(float64(v)/8))
}

func abs32(v float32) float32 { return float32(math.Abs(float64(v))) }

// release returns the height a drag that ended at h settles at, or -1 to close the sheet.
func (s sheetSizes) release(start, dist, velocity float32, p FloatingSheetProps) float32 {
	threshold := orDefault(p.CloseThreshold, 0.25)
	h := start + dist
	if len(s.snaps) == 0 {
		if !p.NoDismiss && h < s.minH*threshold {
			return -1
		}
		return clampFloat(h, s.minH, s.maxH)
	}
	at := s.closest(start)
	last := len(s.snaps) - 1
	if velocity > sheetVelocity && abs32(dist) < s.snaps[at]*sheetFlickPart {
		switch {
		case dist > 0:
			return s.snaps[min(at+1, last)]
		case at > 0:
			return s.snaps[at-1]
		case !p.NoDismiss:
			return -1
		}
		return s.snaps[0]
	}
	if !p.NoDismiss && at == 0 && dist <= 0 && h < s.snaps[0]*threshold {
		return -1
	}
	return s.snaps[s.closest(h)]
}

// FloatingSheet is a bottom sheet inside its container, open while *open is true: its header
// drags it taller or shorter, to its SnapPoints when given, and down to close.
func FloatingSheet(c *ui.Context, open *bool, p FloatingSheetProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
	probe := ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).Width(0).IgnoreInput(true)
	st := ui.Local(probe, "sheet", func() sheetState { return sheetState{} })
	container := probe.LayoutBounds().H
	sz := sheetSizesOf(p, container)

	if !st.probed || (container > 0 && !st.measured) {
		st.probed, st.measured = true, container > 0
		c.Invalidate()
	}
	switch {
	case *open && !st.wasOpen:
		st.height = sz.resting
	case !*open && st.wasOpen:
		st.height = 0
	case *open && !st.pressing && sz.resting != st.lastResting:
		st.height = sz.resting
	}
	jump := *open && st.wasOpen && st.lastResting == 0 && sz.resting > 0
	st.wasOpen, st.lastResting = *open, sz.resting

	root := ui.Column(c).Clip().Background(tk.BgContainer).TextColor(tk.Text).FontSize(tk.FontSize)
	var shown float32
	switch {
	case st.pressing:
		shown = root.Jump("height", st.dragH)
	case jump:
		shown = root.Jump("height", st.height)
	default:
		shown = root.AnimateTo("height", st.height, sheetMotion)
	}
	if p.Inline {
		root.Shrink(0).Radius(12).ZIndex(1)
		if over := shown - sz.base; over > 0 && *open {
			root.Margin(-over, 0, 0, 0)
		}
	} else {
		root.Absolute().Left(0).Bottom(0).ZIndex(10).Radius(12, 12, 0, 0)
		if p.Width <= 0 {
			root.Right(0)
		}
	}
	if p.Width > 0 {
		root.Width(p.Width)
	} else {
		root.AlignSelf(ui.Stretch)
	}
	if p.Embedded {
		root.Border(1, tk.BorderSecondary)
	} else {
		tk.Elevate(root, theme.BoxShadowSecondary)
	}
	if !*open && shown <= 0 {
		return root.Height(0).Hidden(true)
	}
	root.Height(shown)
	return root.Children(func() {
		sheetHeader(c, tk, st, sz, open, p)
		ui.Scroll(c).Grow(1).Basis(0).Children(content)
	})
}

func sheetHeader(c *ui.Context, tk *theme.Tokens, st *sheetState, sz sheetSizes, open *bool, p FloatingSheetProps) {
	header := ui.Column(c).Shrink(0).Padding(8, 16, 4, 16).AlignItems(ui.Center).Cursor(ui.CursorGrab)
	if st.pressing {
		header.Cursor(ui.CursorGrabbing)
	}
	_, dy, held := header.Dragged()
	var actions ui.Element
	header.Children(func() {
		ui.Box(c).Size(32, 4).Radius(2).Margin(0, 0, 8, 0).Background(tk.BorderSecondary)
		ui.Row(c).AlignSelf(ui.Stretch).MinHeight(24).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, p.Title).Grow(1).Basis(0).SingleLine()
			if p.HeaderActions != nil {
				actions = ui.Row(c).Shrink(0).Gap(4).AlignItems(ui.Center).Children(p.HeaderActions)
			}
		})
	})
	switch {
	case held && !st.pressing && !st.fromAction:
		if actions != (ui.Element{}) && actions.Hovered() {
			st.fromAction = true
			return
		}
		st.pressing, st.startH, st.dist, st.startT = true, st.height, 0, c.Now()
		fallthrough
	case held && st.pressing:
		st.dist -= dy
		st.dragH = sz.drag(st.startH + st.dist)
		c.Invalidate()
	case !held && st.pressing:
		st.pressing = false
		c.Invalidate()
		elapsed := float32(c.Now().Sub(st.startT).Milliseconds())
		velocity := float32(0)
		if elapsed > 0 {
			velocity = abs32(st.dist) / elapsed
		}
		h := sz.release(st.startH, st.dist, velocity, p)
		if h < 0 {
			*open, st.height = false, 0
			return
		}
		st.height = h
		if p.OnSnapPointChange != nil {
			for _, sp := range p.SnapPoints {
				if sz.resolve(sp) == h || clampFloat(sz.resolve(sp), sz.minH, sz.maxH) == h {
					p.OnSnapPointChange(sp)
					break
				}
			}
		}
	case !held:
		st.fromAction = false
	}
}
