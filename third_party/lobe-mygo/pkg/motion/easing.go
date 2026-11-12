package motion

import (
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	Fast = 100 * time.Millisecond
	Mid  = 200 * time.Millisecond
	Slow = 300 * time.Millisecond
)

var (
	SoftEase     = ui.CubicBezier(0.32, 0.72, 0, 1)
	ExitEase     = ui.CubicBezier(0.4, 0, 1, 1)
	EaseOut      = ui.CubicBezier(0.215, 0.61, 0.355, 1)
	EaseInOut    = ui.CubicBezier(0.645, 0.045, 0.355, 1)
	EaseOutBack  = ui.CubicBezier(0.12, 0.4, 0.29, 1.46)
	EaseOutQuint = ui.CubicBezier(0.23, 1, 0.32, 1)
	EaseOutCirc  = ui.CubicBezier(0.08, 0.82, 0.17, 1)
	CSSEase      = ui.CubicBezier(0.25, 0.1, 0.25, 1)
	CSSEaseIn    = ui.CubicBezier(0.42, 0, 1, 1)
	ToastEase    = ui.CubicBezier(0.22, 1, 0.36, 1)
)

var SwitchThumb = ui.Animation{Spring: ui.SpringOptions{Stiffness: 360, Damping: 24, Mass: 1}}
