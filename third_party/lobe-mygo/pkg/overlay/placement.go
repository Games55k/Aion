package overlay

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
)

// Placement is the side of its anchor an overlay shows on. Where there is no room there, it shows
// on the opposite side.
type Placement int

const (
	Top Placement = iota
	Bottom
	Left
	Right
)

func (p Placement) side() motion.Side {
	switch p {
	case Bottom:
		return motion.Bottom
	case Left:
		return motion.Left
	case Right:
		return motion.Right
	}
	return motion.Top
}

func attach(e, anchor ui.Element, p Placement, gap float32) {
	switch p {
	case Bottom:
		e.AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).Margin(gap, 0, 0, 0)
	case Left:
		e.AttachTo(anchor, ui.AnchorLeft, ui.AnchorRight).Margin(0, gap, 0, 0)
	case Right:
		e.AttachTo(anchor, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, gap)
	default:
		e.AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, gap, 0)
	}
	e.AttachedOrigin()
}
