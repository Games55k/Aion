package main

import (
	"math"

	"github.com/egoist/mygo/plugins/render"
)

const sampleRate = 48000

type soundBank struct {
	Score       *render.Sound
	ChimeHigh   *render.Sound
	ChimeMid    *render.Sound
	WhooshSoft  *render.Sound
	WhooshSweep *render.Sound
	KeyClick    *render.Sound
	PopSoft     *render.Sound
	FinaleChord *render.Sound
}

func buildSoundBank() *soundBank {
	return &soundBank{
		Score:       newAmbientScore(36.2),
		ChimeHigh:   newChime(880.0, 0.70),  // A5
		ChimeMid:    newChime(587.33, 0.65), // D5
		WhooshSoft:  newWhoosh(0.55, 0.16),
		WhooshSweep: newWhoosh(0.95, 0.24),
		KeyClick:    newKeyClick(),
		PopSoft:     newPop(740.0),
		FinaleChord: newFinaleChord(3.4),
	}
}

// newAmbientScore synthesizes a 48kHz stereo warm ambient electronic score:
// lush detuned Dmaj9 -> Bm9 -> Gmaj9 -> A6/9 pad + soft pulse sub-bass +
// delicate pentatonic kalimba plucks with stereo shimmer.
func newAmbientScore(durationSec float64) *render.Sound {
	n := int(durationSec * sampleRate)
	pcm := make([]float32, n*2)

	// 4-bar chord cycle (each chord = 4.0s, clean harmonic motion across 36s)
	chords := [][]float64{
		{146.83, 220.00, 277.18, 329.63, 369.99}, // Dmaj9
		{123.47, 185.00, 220.00, 277.18, 293.66}, // Bm9
		{98.00, 146.83, 220.00, 246.94, 369.99},  // Gmaj9
		{110.00, 164.81, 220.00, 277.18, 329.63}, // A6/9
	}
	// Pentatonic upper plucks (D major pentatonic)
	arpeggio := []float64{587.33, 659.25, 739.99, 880.00, 1174.66, 880.00, 739.99, 659.25}

	const chordDur = 4.0
	const beatDur = 0.50 // 120 BPM crisp modern pulse

	for i := range n {
		t := float64(i) / sampleRate

		// Global envelope: smooth 0.8s rise, warm 2.2s fade at end
		master := 1.0
		if t < 0.8 {
			master = smoothstep(t / 0.8)
		} else if t > durationSec-2.4 {
			master = smoothstep(max(0, (durationSec-t)/2.4))
		}

		// Chord crossfade (0.7s smooth morph between chords)
		chordPos := t / chordDur
		idx := int(chordPos) % len(chords)
		frac := chordPos - math.Floor(chordPos)
		nextIdx := (idx + 1) % len(chords)
		blend := 0.0
		if frac > 0.82 {
			blend = smoothstep((frac - 0.82) / 0.18)
		}

		// Subtle breathing LFO
		lfo := 0.82 + 0.18*math.Sin(2*math.Pi*0.25*t)

		var padL, padR float64
		for k := range chords[idx] {
			f1 := chords[idx][k]
			f2 := chords[nextIdx][k]
			w1 := 1.0 - blend
			w2 := blend

			// Warm detuned stereo voices
			amp := 0.036
			if k == 0 {
				amp = 0.052 // subtle root warmth
			}
			sL := w1*math.Sin(2*math.Pi*(f1*0.9988)*t+float64(k)*0.7) +
				w2*math.Sin(2*math.Pi*(f2*0.9988)*t+float64(k)*0.7)
			sR := w1*math.Sin(2*math.Pi*(f1*1.0012)*t+float64(k)*1.3) +
				w2*math.Sin(2*math.Pi*(f2*1.0012)*t+float64(k)*1.3)
			// Gentle 2nd harmonic silk
			sL += 0.18 * (w1*math.Sin(2*math.Pi*2*f1*t) + w2*math.Sin(2*math.Pi*2*f2*t))
			sR += 0.18 * (w1*math.Sin(2*math.Pi*2*f1*t+0.4) + w2*math.Sin(2*math.Pi*2*f2*t+0.4))

			padL += amp * sL
			padR += amp * sR
		}
		padL *= lfo
		padR *= lfo

		// Soft rhythmic pulse bass (starts gently at t=2.0s, lifts out at t=32.5s)
		var pulse float64
		if t >= 2.0 && t < 32.8 {
			gate := 1.0
			if t < 3.5 {
				gate = (t - 2.0) / 1.5
			} else if t > 31.5 {
				gate = (32.8 - t) / 1.3
			}
			beatFrac := math.Mod(t, beatDur*2) / (beatDur * 2)
			subEnv := math.Exp(-6.5*beatFrac) * min(beatFrac/0.02, 1.0)
			rootFreq := chords[idx][0] * 0.5
			pulse = gate * 0.068 * subEnv * math.Sin(2*math.Pi*rootFreq*t)
		}

		// Crystalline stereo kalimba arpeggio (gentle sparse notes every 0.5s from t=1.0s to t=31.5s)
		var arpL, arpR float64
		if t >= 1.0 && t < 32.0 {
			step := int((t - 1.0) / beatDur)
			localT := math.Mod(t-1.0, beatDur)
			// Leave intentional musical space on some beats
			if step%8 != 3 && step%8 != 7 {
				note := arpeggio[step%len(arpeggio)]
				env := math.Exp(-9.5*localT) * min(localT/0.006, 1.0)
				tone := math.Sin(2*math.Pi*note*localT) + 0.25*math.Sin(2*math.Pi*2*note*localT)*math.Exp(-14*localT)
				// Gentle stereo alternation
				pan := 0.5 + 0.32*math.Sin(float64(step)*1.4)
				arpL = 0.055 * env * tone * (1 - pan*0.5)
				arpR = 0.055 * env * tone * (0.5 + pan*0.5)
			}
		}

		left := master * (padL + pulse + arpL)
		right := master * (padR + pulse + arpR)

		pcm[i*2] = float32(math.Tanh(left * 1.15))
		pcm[i*2+1] = float32(math.Tanh(right * 1.15))
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func newChime(freq, durationSec float64) *render.Sound {
	n := int(durationSec * sampleRate)
	pcm := make([]float32, n*2)
	for i := range n {
		t := float64(i) / sampleRate
		env := math.Exp(-5.5*t) * min(t/0.005, 1.0)
		s := math.Sin(2*math.Pi*freq*t) +
			0.35*math.Sin(2*math.Pi*2*freq*t)*math.Exp(-9*t) +
			0.12*math.Sin(2*math.Pi*3*freq*t)*math.Exp(-16*t)
		val := float32(0.18 * env * s)
		pcm[i*2] = val * 0.92
		pcm[i*2+1] = val * 1.0
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func newWhoosh(durationSec, gain float64) *render.Sound {
	n := int(durationSec * sampleRate)
	pcm := make([]float32, n*2)
	// Deterministic LCG noise shaped with a simple 1-pole resonant lowpass sweep
	var seed uint64 = 0x51a7f39b2c1d4e77
	var lpL, lpR float64
	for i := range n {
		t := float64(i) / sampleRate
		u := t / durationSec
		// Smooth bell envelope peaking at 45%
		env := math.Sin(math.Pi * math.Pow(u, 0.85))
		env = env * env

		seed = seed*6364136223846793005 + 1442695040888963407
		noise1 := (float64(int32(seed>>32)) / float64(1<<31))
		seed = seed*6364136223846793005 + 1442695040888963407
		noise2 := (float64(int32(seed>>32)) / float64(1<<31))

		// Cutoff sweeps gently upward then down
		alpha := 0.02 + 0.14*math.Sin(math.Pi*u)
		lpL += alpha * (noise1 - lpL)
		lpR += alpha * (noise2 - lpR)

		// Subtle airy pitch sheen
		air := 0.15 * math.Sin(2*math.Pi*(180+340*u)*t)
		pcm[i*2] = float32(gain * env * (lpL + air))
		pcm[i*2+1] = float32(gain * env * (lpR + air))
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func newKeyClick() *render.Sound {
	const dur = 0.045
	n := int(dur * sampleRate)
	pcm := make([]float32, n*2)
	var seed uint64 = 0x9e3779b97f4a7c15
	for i := range n {
		t := float64(i) / sampleRate
		env := math.Exp(-110*t) * min(t/0.001, 1.0)
		seed = seed*6364136223846793005 + 1
		noise := float64(int32(seed>>32)) / float64(1<<31)
		body := math.Sin(2*math.Pi*1950*t) + 0.5*math.Sin(2*math.Pi*820*t)
		val := float32(0.18 * env * (0.65*body + 0.35*noise))
		pcm[i*2] = val
		pcm[i*2+1] = val
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func newPop(freq float64) *render.Sound {
	const dur = 0.12
	n := int(dur * sampleRate)
	pcm := make([]float32, n*2)
	for i := range n {
		t := float64(i) / sampleRate
		env := math.Exp(-34*t) * min(t/0.003, 1.0)
		// Subtle upward pitch blip
		f := freq * (0.85 + 0.30*min(t/0.04, 1.0))
		val := float32(0.14 * env * math.Sin(2*math.Pi*f*t))
		pcm[i*2] = val
		pcm[i*2+1] = val
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func newFinaleChord(durationSec float64) *render.Sound {
	n := int(durationSec * sampleRate)
	pcm := make([]float32, n*2)
	notes := []float64{293.66, 369.99, 440.00, 554.37, 659.25, 880.00} // Dmaj9 shimmer
	for i := range n {
		t := float64(i) / sampleRate
		env := math.Exp(-1.35*t) * min(t/0.08, 1.0)
		var l, r float64
		for k, f := range notes {
			w := 0.045 / (1 + 0.12*float64(k))
			l += w * math.Sin(2*math.Pi*(f*0.999)*t+float64(k)*0.5)
			r += w * math.Sin(2*math.Pi*(f*1.001)*t+float64(k)*0.9)
		}
		pcm[i*2] = float32(env * l)
		pcm[i*2+1] = float32(env * r)
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func smoothstep(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	return x * x * (3 - 2*x)
}
