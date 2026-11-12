package layout

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Placement is the edge of its container a DraggablePanel sits at; it resizes from the edge
// opposite.
type Placement int

const (
	Right Placement = iota
	Left
	Top
	Bottom
)

type DraggablePanelProps struct {
	// Placement is the edge the panel sits at.
	Placement Placement
	// MinSize and MaxSize bound the size a drag gives it; a MaxSize of zero sets no bound.
	MinSize, MaxSize float32
	// CollapseThreshold collapses the panel when a drag leaves it this size or smaller, and keeps
	// the size it had before the drag to expand to; zero never collapses it by dragging.
	CollapseThreshold float32
	// NoToggle hides the button on the seam that collapses and expands the panel.
	NoToggle bool
}

type panelDrag struct {
	pressing   bool
	start, raw float32
}

var (
	panelResize = ui.Animation{Duration: 250 * time.Millisecond, Ease: motion.EaseOut}
	seamFade    = ui.Animation{Duration: 160 * time.Millisecond, Ease: motion.EaseOut}
)

// DraggablePanel shows content in a panel at an edge of its container, *size DIPs wide (or tall at
// Top and Bottom) while *expanded is true. Dragging the seam on its inner edge resizes it; the
// button on the seam collapses and expands it, and the content keeps its size and slides out with
// the inner edge.
func DraggablePanel(c *ui.Context, size *float32, expanded *bool, p DraggablePanelProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
	horizontal := p.Placement == Right || p.Placement == Left

	root := ui.Box(c).Shrink(0).AlignSelf(ui.Stretch).ZIndex(1)
	drag := ui.Local(root, "drag", func() panelDrag { return panelDrag{} })

	var dx, dy float32
	var held bool
	root.Children(func() {
		ui.Box(c).Fill().Clip().Children(func() {
			body := ui.Column(c).Absolute()
			switch p.Placement {
			case Left:
				body.Right(0).Top(0).Bottom(0).Width(*size)
			case Top:
				body.Bottom(0).Left(0).Right(0).Height(*size)
			case Bottom:
				body.Top(0).Left(0).Right(0).Height(*size)
			default:
				body.Left(0).Top(0).Bottom(0).Width(*size)
			}
			body.Children(content)
		})

		handle := edge(ui.ButtonBase(c).Absolute(), p.Placement, -4, 8).Role(ui.RoleSeparator).Label("Resize panel")
		if horizontal {
			handle.Cursor(ui.CursorResizeColumn)
		} else {
			handle.Cursor(ui.CursorResizeRow)
		}
		dx, dy, held = handle.Dragged()
		seam := tk.BorderSecondary
		switch {
		case held:
			seam = tk.Primary
		case handle.Hovered():
			seam = tk.Fill
		}
		edge(ui.Box(c).Absolute().IgnoreInput(true), p.Placement, 0, 1).
			Background(root.AnimateColor("seam", seam, seamFade))

		if !p.NoToggle {
			panelToggle(c, tk, root, expanded, p.Placement)
		}
	})

	resize(dx, dy, held, size, expanded, drag, p)

	target := float32(0)
	if *expanded {
		target = *size
	}
	var current float32
	if drag.pressing {
		current = root.Jump("size", target)
	} else {
		current = root.AnimateTo("size", target, panelResize)
	}
	if horizontal {
		root.Width(current)
	} else {
		root.Height(current)
	}
	return root
}

func resize(dx, dy float32, held bool, size *float32, expanded *bool, d *panelDrag, p DraggablePanelProps) {
	if held && !d.pressing {
		d.pressing, d.start, d.raw = true, *size, *size
	}
	if held {
		switch p.Placement {
		case Right:
			d.raw -= dx
		case Left:
			d.raw += dx
		case Bottom:
			d.raw -= dy
		case Top:
			d.raw += dy
		}
		*size = max(d.raw, p.MinSize)
		if p.MaxSize > 0 {
			*size = min(*size, p.MaxSize)
		}
	}
	if !held && d.pressing {
		d.pressing = false
		if p.CollapseThreshold > 0 && d.raw <= p.CollapseThreshold {
			*expanded, *size = false, d.start
		}
	}
}

func edge(e ui.Element, placement Placement, offset, thickness float32) ui.Element {
	switch placement {
	case Left:
		return e.Top(0).Bottom(0).Right(offset).Width(thickness)
	case Top:
		return e.Left(0).Right(0).Bottom(offset).Height(thickness)
	case Bottom:
		return e.Left(0).Right(0).Top(offset).Height(thickness)
	}
	return e.Top(0).Bottom(0).Left(offset).Width(thickness)
}

func panelToggle(c *ui.Context, tk *theme.Tokens, root ui.Element, expanded *bool, placement Placement) {
	label, icon := "Collapse panel", collapseIcon(placement, *expanded)
	if !*expanded {
		label = "Expand panel"
	}
	offset := float32(-8)
	if !*expanded {
		offset = -16
	}
	b := ui.ButtonBase(c).Label(label).Absolute().Radius(8).Cursor(ui.CursorPointer).
		Background(tk.BgContainer).Border(1, tk.BorderSecondary)
	switch placement {
	case Left:
		b.Right(offset).TopPercent(50).TranslatePercent(0, -50).Size(16, 40)
	case Top:
		b.Bottom(offset).LeftPercent(50).TranslatePercent(-50, 0).Size(40, 16)
	case Bottom:
		b.Top(offset).LeftPercent(50).TranslatePercent(-50, 0).Size(40, 16)
	default:
		b.Left(offset).TopPercent(50).TranslatePercent(0, -50).Size(16, 40)
	}
	shown := float32(0)
	if !*expanded || root.Hovered() || b.Hovered() || b.FocusVisible() {
		shown = 1
	}
	b.Opacity(b.AnimateTo("shown", shown, seamFade))
	glyph := tk.TextTertiary
	if b.Hovered() {
		glyph = tk.TextSecondary
	}
	b.Children(func() { ui.Icon(c, icon).FontSize(12).TextColor(glyph) })
	if b.Clicked() {
		*expanded = !*expanded
	}
}

func collapseIcon(placement Placement, expanded bool) *ui.SVG {
	toward := map[Placement]func() *ui.SVG{
		Right: lucide.ChevronRight, Left: lucide.ChevronLeft, Top: lucide.ChevronUp, Bottom: lucide.ChevronDown,
	}
	away := map[Placement]func() *ui.SVG{
		Right: lucide.ChevronLeft, Left: lucide.ChevronRight, Top: lucide.ChevronDown, Bottom: lucide.ChevronUp,
	}
	if expanded {
		return toward[placement]()
	}
	return away[placement]()
}
