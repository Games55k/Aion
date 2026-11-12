package feedback

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
)

// DialogBase stops building the dialog the frame it closes, and builds the panel inside the
// backdrop, which appears in the same frame, so neither the panel's Enter nor its Exit would play:
// present keeps the dialog built while it leaves and animates values it jumps to their start,
// which hold across the two builds of a frame a click makes, where a first pose would not.
// Dialogs animate the backdrop's color, not its opacity, which the panel inside it would take too.
type dialogState struct{ shown, closing, entered bool }

type dialogMotion struct{ in, out, fade ui.Animation }

func motionOf(panel ui.ElementTransition) dialogMotion {
	b := motion.Backdrop()
	return dialogMotion{
		in:   ui.Animation{Duration: panel.Duration, Ease: panel.Ease},
		out:  ui.Animation{Duration: panel.ExitDuration, Ease: panel.ExitEase},
		fade: ui.Animation{Duration: b.Duration, Ease: b.Ease},
	}
}

// present shows a dialog while *open is true and while it animates out after: build styles the
// backdrop and the panel from shown and dim, which go from 0 to 1 as it opens and back as it
// closes, and builds the panel's content.
func present(c *ui.Context, open *bool, m dialogMotion, build func(backdrop, panel ui.Element, shown, dim float32, closing bool)) {
	st := ui.Local(ui.Box(c).Absolute().Size(0, 0), "dialog", func() dialogState { return dialogState{} })
	if *open {
		st.shown, st.closing = true, false
	} else if st.shown {
		st.closing = true
	}
	if !st.shown {
		return
	}
	gone := false
	ui.DialogBase(c, &st.shown, func(backdrop, panel ui.Element) {
		if !st.entered {
			st.entered = true
			panel.Jump("shown", 0)
			backdrop.Jump("dim", 0)
		}
		target, anim := float32(1), m.in
		if st.closing {
			target, anim = 0, m.out
			backdrop.IgnoreInput(true)
		}
		shown := panel.AnimateTo("shown", target, anim)
		dim := backdrop.AnimateTo("dim", target, m.fade)
		gone = st.closing && shown <= 0 && dim <= 0
		build(backdrop, panel, shown, dim, st.closing)
	})
	switch {
	case !st.shown:
		*open, st.shown, st.closing = false, true, true
		c.Invalidate()
	case gone:
		*st = dialogState{}
		c.Invalidate()
	}
}
