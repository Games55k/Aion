package motion

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

func Smooth(settle time.Duration) ui.Animation { return Spring(1, 0.001, settle) }

func Snappy(settle time.Duration) ui.Animation { return Spring(0.8, 0.001, settle) }

func Bouncy(settle time.Duration) ui.Animation { return Spring(0.7, 0.003, settle) }

func Spring(zeta, threshold float64, settle time.Duration) ui.Animation {
	omega := unitSettle(zeta, threshold) / settle.Seconds()
	return ui.Animation{Spring: ui.SpringOptions{
		Stiffness: omega * omega,
		Damping:   2 * zeta * omega,
		Mass:      1,
	}}
}

func unitSettle(zeta, threshold float64) float64 {
	const step, horizon = 0.0005, 60.0
	last := 0.0
	for t := 0.0; t < horizon; t += step {
		if math.Abs(stepResponse(zeta, t)-1) > threshold {
			last = t
		}
	}
	return last
}

func stepResponse(zeta, t float64) float64 {
	if zeta >= 1 {
		return 1 - math.Exp(-t)*(1+t)
	}
	wd := math.Sqrt(1 - zeta*zeta)
	return 1 - math.Exp(-zeta*t)*(math.Cos(wd*t)+zeta/wd*math.Sin(wd*t))
}
