package main

import (
	"testing"

	"github.com/egoist/mygo/plugins/render"
)

// Verify that demo seeding, studio-level stereo sound synthesis, and keyframe composition rendering succeed across light and dark acts.
func TestPromoCompositionRendersKeyframes(t *testing.T) {
	assets, err := captureDemoAssets()
	if err != nil {
		t.Fatalf("capture demo assets: %v", err)
	}
	if assets.LogoSVG == nil || assets.FocusLight == nil || assets.BoardLight == nil || assets.BoardDark == nil ||
		assets.TaskDetail == nil || assets.JobsLight == nil || assets.JobDetail == nil ||
		assets.PaletteDark == nil || assets.SettingsDark == nil {
		t.Fatalf("expected all demo visual assets to be populated")
	}

	sfx := buildSoundBank()
	if sfx.Score == nil || sfx.ChimeHigh == nil || sfx.WhooshSweep == nil || sfx.FinaleChord == nil {
		t.Fatalf("expected all synthesized sound assets to be populated")
	}
	// Ensure BGM sits at a clear, audible commercial loudness (~ -14 dBFS RMS).
	if sfx.ScoreRMSDB < -16.5 || sfx.ScoreRMSDB > -11.5 {
		t.Fatalf("ScoreRMSDB = %.2f dBFS, want within [-16.5, -11.5]", sfx.ScoreRMSDB)
	}

	comp := buildComposition(assets, sfx)
	for _, frame := range []int{0, 160, 520, 2145, 2900, 3220, totalFrames - 1} {
		img, err := render.Still(comp, frame, render.Options{})
		if err != nil {
			t.Fatalf("render still frame %d: %v", frame, err)
		}
		if img.Bounds().Dx() != videoWidth || img.Bounds().Dy() != videoHeight {
			t.Fatalf("frame %d bounds = %v, want %dx%d", frame, img.Bounds(), videoWidth, videoHeight)
		}
	}
}
