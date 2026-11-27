package main

import (
	"math"

	"github.com/egoist/mygo/plugins/render"
)

const sampleRate = 48000

type soundBank struct {
	Score       *render.Sound
	ScoreRMSDB  float64
	ChimeHigh   *render.Sound
	ChimeMid    *render.Sound
	WhooshSoft  *render.Sound
	WhooshSweep *render.Sound
	KeyClick    *render.Sound
	PopSoft     *render.Sound
	FinaleChord *render.Sound
}

func buildSoundBank() *soundBank {
	score, rmsDB := newAmbientScore(54.2)
	return &soundBank{
		Score:       score,
		ScoreRMSDB:  rmsDB,
		ChimeHigh:   newChime(880.0, 0.80),  // A5
		ChimeMid:    newChime(587.33, 0.75), // D5
		WhooshSoft:  newWhoosh(0.65, 0.28),
		WhooshSweep: newWhoosh(1.20, 0.36),
		KeyClick:    newKeyClick(),
		PopSoft:     newPop(740.0),
		FinaleChord: newFinaleChord(4.0),
	}
}

// newAmbientScore synthesizes a 48kHz stereo Apple-style minimalist product film score
// mastered to a clear, warm commercial loudness (~ -14 dBFS RMS, peak <= -1.0 dBFS):
//   - Lush detuned Dmaj9 -> Bm9 -> Gmaj9 -> A6/9 warm analog-style pad
//   - Deep clean sub-bass + steady 108 BPM gentle heart-beat pulse
//   - Felt-piano / glass marimba pentatonic motif with warm stereo delay
func newAmbientScore(durationSec float64) (*render.Sound, float64) {
	n := int(durationSec * sampleRate)
	bufL := make([]float64, n)
	bufR := make([]float64, n)

	// 4-chord progression (each chord lasts 4.44s = 8 beats at 108 BPM)
	const beatDur = 60.0 / 108.0 // 0.5556s
	const chordDur = 8.0 * beatDur

	chords := [][]float64{
		{146.83, 220.00, 277.18, 329.63, 369.99}, // Dmaj9
		{123.47, 185.00, 220.00, 277.18, 293.66}, // Bm9
		{98.00, 146.83, 220.00, 246.94, 369.99},  // Gmaj9
		{110.00, 164.81, 220.00, 277.18, 329.63}, // A6/9
	}

	// Melodic felt-piano / marimba sequence (0 = intentional rest for breathing room)
	motif := []float64{
		587.33, 0, 739.99, 880.00, 0, 739.99, 659.25, 0,
		587.33, 0, 880.00, 1174.66, 0, 880.00, 739.99, 659.25,
	}

	// Deterministic subtle high-frequency air texture for soft rhythmic groove
	var rng uint64 = 0x7a3b91f5c8e2041d
	var lpShaker float64

	for i := range n {
		t := float64(i) / sampleRate

		// Global envelope: smooth 0.9s swell in, graceful 3.2s fade out at end
		master := 1.0
		if t < 0.9 {
			master = smoothstep(t / 0.9)
		} else if t > durationSec-3.2 {
			master = smoothstep(max(0, (durationSec-t)/3.2))
		}

		// Chord morph
		chordPos := t / chordDur
		idx := int(chordPos) % len(chords)
		frac := chordPos - math.Floor(chordPos)
		nextIdx := (idx + 1) % len(chords)
		blend := 0.0
		if frac > 0.84 {
			blend = smoothstep((frac - 0.84) / 0.16)
		}

		lfo := 0.88 + 0.12*math.Sin(2*math.Pi*0.22*t)

		// 1. Warm stereo pad
		var padL, padR float64
		for k := range chords[idx] {
			f1 := chords[idx][k]
			f2 := chords[nextIdx][k]
			w1 := 1.0 - blend
			w2 := blend

			amp := 0.085
			if k == 0 {
				amp = 0.12
			}
			sL := w1*math.Sin(2*math.Pi*(f1*0.9987)*t+float64(k)*0.65) +
				w2*math.Sin(2*math.Pi*(f2*0.9987)*t+float64(k)*0.65)
			sR := w1*math.Sin(2*math.Pi*(f1*1.0013)*t+float64(k)*1.25) +
				w2*math.Sin(2*math.Pi*(f2*1.0013)*t+float64(k)*1.25)
			// Warm 2nd + gentle 3rd harmonic richness
			sL += 0.24*(w1*math.Sin(2*math.Pi*2*f1*t)+w2*math.Sin(2*math.Pi*2*f2*t)) +
				0.07*(w1*math.Sin(2*math.Pi*3*f1*t)+w2*math.Sin(2*math.Pi*3*f2*t))
			sR += 0.24*(w1*math.Sin(2*math.Pi*2*f1*t+0.35)+w2*math.Sin(2*math.Pi*2*f2*t+0.35)) +
				0.07*(w1*math.Sin(2*math.Pi*3*f1*t+0.5)+w2*math.Sin(2*math.Pi*3*f2*t+0.5))

			padL += amp * sL
			padR += amp * sR
		}
		padL *= lfo
		padR *= lfo

		// 2. Clean warm bass & gentle Apple-style muted kick pulse (enters after opening at 2.6s, lifts at 50s)
		var bass, kick, shaker float64
		if t >= 2.6 && t < 50.2 {
			gate := 1.0
			if t < 4.2 {
				gate = smoothstep((t - 2.6) / 1.6)
			} else if t > 48.0 {
				gate = smoothstep((50.2 - t) / 2.2)
			}
			rootFreq := chords[idx][0] * 0.5
			beatT := math.Mod(t, beatDur)
			barBeat := int(t/beatDur) % 4

			// Smooth bass pulse on every beat
			bassEnv := math.Exp(-3.2*beatT) * min(beatT/0.012, 1.0)
			bass = gate * 0.20 * bassEnv * (math.Sin(2*math.Pi*rootFreq*t) + 0.30*math.Sin(2*math.Pi*2*rootFreq*t))

			// Soft muted thump on beats 1 and 3
			if barBeat == 0 || barBeat == 2 {
				kickEnv := math.Exp(-14.0*beatT) * min(beatT/0.004, 1.0)
				kickFreq := 46.0 + 64.0*math.Exp(-32.0*beatT)
				kick = gate * 0.16 * kickEnv * math.Sin(2*math.Pi*kickFreq*beatT)
			}

			// Feather-light off-beat pulse tick (gives forward momentum without intruding)
			offT := math.Mod(t+beatDur*0.5, beatDur)
			rng = rng*6364136223846793005 + 1442695040888963407
			noise := float64(int32(rng>>32)) / float64(1<<31)
			lpShaker += 0.30 * (noise - lpShaker)
			hp := noise - lpShaker
			shakerEnv := math.Exp(-42.0*offT) * min(offT/0.003, 1.0)
			shaker = gate * 0.032 * shakerEnv * hp
		}

		// 3. Felt-piano / crystalline marimba motif (1.0s to 49.6s)
		var melL, melR float64
		if t >= 1.0 && t < 49.6 {
			mGate := 1.0
			if t < 2.0 {
				mGate = smoothstep((t - 1.0) / 1.0)
			} else if t > 47.6 {
				mGate = smoothstep((49.6 - t) / 2.0)
			}
			step := int((t - 1.0) / beatDur)
			localT := math.Mod(t-1.0, beatDur)
			note := motif[step%len(motif)]
			if note > 0 {
				env := math.Exp(-5.2*localT) * min(localT/0.007, 1.0)
				tone := math.Sin(2*math.Pi*note*localT) +
					0.32*math.Sin(2*math.Pi*2*note*localT)*math.Exp(-10*localT) +
					0.10*math.Sin(2*math.Pi*3*note*localT)*math.Exp(-18*localT)
				pan := 0.5 + 0.30*math.Sin(float64(step)*1.25)
				melL = mGate * 0.16 * env * tone * (1 - 0.45*pan)
				melR = mGate * 0.16 * env * tone * (0.55 + 0.45*pan)
			}
		}

		bufL[i] = master * (padL + bass + kick + shaker + melL)
		bufR[i] = master * (padR + bass + kick + shaker + melR)
	}

	// Add warm stereo ping-pong echo tail (0.37s / 0.55s) for spacious studio depth.
	dL := int(0.37 * sampleRate)
	dR := int(0.555 * sampleRate)
	for i := dR; i < n; i++ {
		bufL[i] += 0.22 * bufR[i-dL]
		bufR[i] += 0.20 * bufL[i-dR]
	}

	// Normalize / master cleanly to target RMS ~ -14.2 dBFS (linear ~0.195) with soft peak ceiling at 0.88.
	var sumSq float64
	for i := range n {
		sumSq += bufL[i]*bufL[i] + bufR[i]*bufR[i]
	}
	rms := math.Sqrt(sumSq / float64(n*2))
	gain := 1.0
	if rms > 1e-6 {
		gain = 0.20 / rms // ~ -14 dBFS RMS
	}

	pcm := make([]float32, n*2)
	var finalSq float64
	for i := range n {
		l := 0.90 * math.Tanh(bufL[i]*gain/0.90)
		r := 0.90 * math.Tanh(bufR[i]*gain/0.90)
		pcm[i*2] = float32(l)
		pcm[i*2+1] = float32(r)
		finalSq += l*l + r*r
	}
	finalRMS := math.Sqrt(finalSq / float64(n*2))
	rmsDB := 20.0 * math.Log10(max(finalRMS, 1e-9))
	return render.SoundOf(sampleRate, 2, pcm), rmsDB
}

func newChime(freq, durationSec float64) *render.Sound {
	n := int(durationSec * sampleRate)
	pcm := make([]float32, n*2)
	for i := range n {
		t := float64(i) / sampleRate
		env := math.Exp(-5.0*t) * min(t/0.005, 1.0)
		s := math.Sin(2*math.Pi*freq*t) +
			0.35*math.Sin(2*math.Pi*2*freq*t)*math.Exp(-9*t) +
			0.12*math.Sin(2*math.Pi*3*freq*t)*math.Exp(-16*t)
		val := float32(0.24 * env * s)
		pcm[i*2] = val * 0.94
		pcm[i*2+1] = val
	}
	return render.SoundOf(sampleRate, 2, pcm)
}

func newWhoosh(durationSec, gain float64) *render.Sound {
	n := int(durationSec * sampleRate)
	pcm := make([]float32, n*2)
	var seed uint64 = 0x51a7f39b2c1d4e77
	var lpL, lpR float64
	for i := range n {
		t := float64(i) / sampleRate
		u := t / durationSec
		env := math.Sin(math.Pi * math.Pow(u, 0.85))
		env = env * env

		seed = seed*6364136223846793005 + 1442695040888963407
		noise1 := float64(int32(seed>>32)) / float64(1<<31)
		seed = seed*6364136223846793005 + 1442695040888963407
		noise2 := float64(int32(seed>>32)) / float64(1<<31)

		alpha := 0.02 + 0.14*math.Sin(math.Pi*u)
		lpL += alpha * (noise1 - lpL)
		lpR += alpha * (noise2 - lpR)

		air := 0.14 * math.Sin(2*math.Pi*(180+300*u)*t)
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
		val := float32(0.24 * env * (0.65*body + 0.35*noise))
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
		f := freq * (0.88 + 0.24*min(t/0.04, 1.0))
		val := float32(0.18 * env * math.Sin(2*math.Pi*f*t))
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
		env := math.Exp(-1.15*t) * min(t/0.09, 1.0)
		var l, r float64
		for k, f := range notes {
			w := 0.085 / (1 + 0.12*float64(k))
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
