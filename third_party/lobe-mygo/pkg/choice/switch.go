package choice

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
)

type SwitchProps struct {
	// Size is theme.Middle (36 by 22 DIPs) or theme.Small (28 by 16).
	Size theme.Size
}

type switchMetrics struct {
	w, h, thumb, checkedX, pressedCheckedX, pressedWidth float32
}

var switchTransition = ui.ElementTransition{Duration: 200 * time.Millisecond, Ease: motion.EaseOut, Colors: true, Shadows: true}

func Switch(c *ui.Context, on *bool, p SwitchProps) ui.Element {
	tk := theme.TokensOf(c)
	m := switchMetrics{36, 22, 18, 14, 10, 22}
	if p.Size == theme.Small {
		m = switchMetrics{28, 16, 12, 12, 8, 16}
	}

	track := ui.SwitchBase(c, on).Size(m.w, m.h).Padding(2).Radius(100).
		Cursor(ui.CursorPointer).Transition(switchTransition)

	hovered, pressed := track.Hovered() && !track.IsDisabled(), track.Pressed()
	bg := tk.FillSecondary
	switch {
	case *on && hovered:
		bg = tk.PrimaryHover
	case *on:
		bg = tk.Primary
	case hovered:
		bg = tk.Fill
	}
	track.Background(bg)
	if *on {
		track.InsetShadow(0, 1.5, 3, 0, ui.RGBA(0, 0, 0, 0.18))
	} else {
		track.InsetShadow(0, 1.5, 2, 0, ui.RGBA(0, 0, 0, 0.08))
	}

	targetX, targetW := float32(0), m.thumb
	if pressed {
		targetW = m.pressedWidth
	}
	if *on {
		targetX = m.checkedX
		if pressed {
			targetX = m.pressedCheckedX
		}
	}

	track.Children(func() {
		thumb := ui.Box(c).Height(m.thumb).Radius(m.thumb/2).Background(tk.BgContainer).
			Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04)).
			Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.06)).
			Shadow(0, 3, 8, 0, ui.RGBA(0, 30, 80, 0.16))
		thumb.Width(thumb.AnimateTo("w", targetW, motion.SwitchThumb))
		thumb.Translate(thumb.AnimateTo("x", targetX, motion.SwitchThumb), 0)
	})
	return track
}
