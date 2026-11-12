package motion

import (
	"time"

	"github.com/egoist/mygo/ui"
)

type Side int

const (
	Bottom Side = iota
	Top
	Left
	Right
)

func Popup() ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     150 * time.Millisecond,
		Ease:         EaseOut,
		ExitDuration: 75 * time.Millisecond,
		ExitEase:     CSSEaseIn,
		Enter:        &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
		Exit:         &ui.Motion{Scale: 0.96, Y: -6, Flip: true},
	}
}

func Tooltip() ui.ElementTransition {
	enter := Bouncy(280 * time.Millisecond)
	return ui.ElementTransition{
		Animation:    &enter,
		ExitDuration: 100 * time.Millisecond,
		ExitEase:     ExitEase,
		Enter:        &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
		Exit:         &ui.Motion{Scale: 0.97, Y: -3, Flip: true},
	}
}

func Modal() ui.ElementTransition {
	return ui.ElementTransition{
		Duration:     220 * time.Millisecond,
		Ease:         SoftEase,
		ExitDuration: 120 * time.Millisecond,
		ExitEase:     ExitEase,
		Enter:        &ui.Motion{Scale: 0.97},
		Exit:         &ui.Motion{Scale: 0.98},
	}
}

func Backdrop() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Ease:     SoftEase,
		Enter:    &ui.Motion{},
		Exit:     &ui.Motion{},
	}
}

func Drawer(from Side) ui.ElementTransition {
	offscreen := func() *ui.Motion {
		m := &ui.Motion{Opacity: 1}
		switch from {
		case Right:
			m.XPercent = 100
		case Left:
			m.XPercent = -100
		case Top:
			m.YPercent = -100
		default:
			m.YPercent = 100
		}
		return m
	}
	return ui.ElementTransition{
		Duration:     300 * time.Millisecond,
		Ease:         SoftEase,
		ExitDuration: 220 * time.Millisecond,
		ExitEase:     ExitEase,
		Enter:        offscreen(),
		Exit:         offscreen(),
	}
}

func Toast(from Side) ui.ElementTransition {
	offset := float32(150)
	if from == Top {
		offset = -150
	}
	return ui.ElementTransition{
		Duration: 400 * time.Millisecond,
		Ease:     ToastEase,
		Enter:    &ui.Motion{YPercent: offset},
		Exit:     &ui.Motion{YPercent: offset},
	}
}

func Collapse() ui.ElementTransition {
	return ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Ease:     EaseOut,
		Enter:    &ui.Motion{Collapse: true, Y: 6},
		Exit:     &ui.Motion{Collapse: true, Y: -6},
	}
}

// Toward turns the offset of a preset made for an element below its anchor, as Popup and Tooltip
// are, so that it starts at the anchor from side.
func Toward(tr ui.ElementTransition, side Side) ui.ElementTransition {
	turn := func(m *ui.Motion) *ui.Motion {
		if m == nil {
			return nil
		}
		t := *m
		switch side {
		case Top:
			t.Y = -m.Y
		case Right:
			t.X, t.Y = m.Y, 0
		case Left:
			t.X, t.Y = -m.Y, 0
		}
		return &t
	}
	tr.Enter, tr.Exit = turn(tr.Enter), turn(tr.Exit)
	return tr
}
