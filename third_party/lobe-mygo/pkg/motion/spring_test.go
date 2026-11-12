package motion

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

type springRun struct {
	values []float32
	step   time.Duration
}

func runSpring(t *testing.T, a ui.Animation, total time.Duration) springRun {
	t.Helper()
	var target float32
	var value float32
	tt := ui.NewTester(func(c *ui.Context) {
		box := ui.Box(c).Size(10, 10)
		value = box.AnimateTo("v", target, a)
	}, 50, 50)
	target = 1
	tt.Frame()
	run := springRun{step: 4 * time.Millisecond}
	for elapsed := time.Duration(0); elapsed < total; elapsed += run.step {
		tt.Advance(run.step)
		run.values = append(run.values, value)
	}
	return run
}

func (r springRun) lastOutside(threshold float64) time.Duration {
	last := time.Duration(0)
	for i, v := range r.values {
		if math.Abs(float64(v)-1) > threshold {
			last = time.Duration(i+1) * r.step
		}
	}
	return last
}

func (r springRun) peak() float32 {
	var m float32
	for _, v := range r.values {
		m = max(m, v)
	}
	return m
}

func TestSpringsSettleAtTheirDuration(t *testing.T) {
	cases := []struct {
		name      string
		anim      ui.Animation
		settle    time.Duration
		threshold float64
		overshoot float32
	}{
		{"smooth", Smooth(400 * time.Millisecond), 400 * time.Millisecond, 0.001, 0},
		{"snappy", Snappy(608 * time.Millisecond), 608 * time.Millisecond, 0.001, 0.015},
		{"bouncy", Bouncy(280 * time.Millisecond), 280 * time.Millisecond, 0.003, 0.046},
	}
	for _, c := range cases {
		run := runSpring(t, c.anim, 2*c.settle)
		got := run.lastOutside(c.threshold)
		if diff := got - c.settle; diff < -24*time.Millisecond || diff > 24*time.Millisecond {
			t.Errorf("%s settles at %v, want %v", c.name, got, c.settle)
		}
		if peak := run.peak() - 1; math.Abs(float64(peak-c.overshoot)) > 0.006 {
			t.Errorf("%s overshoots %.3f, want %.3f", c.name, peak, c.overshoot)
		}
	}
}
