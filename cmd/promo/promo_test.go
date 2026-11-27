package main

import (
	"context"
	"testing"

	"github.com/egoist/mygo/plugins/render"
)

// Verify that demo seeding, BAT/Mag-7 vector logos, studio-level BGM, and keyframe rendering succeed across all acts.
func TestPromoCompositionRendersKeyframes(t *testing.T) {
	for _, rawURL := range []string{
		"https://careers.google.com/jobs",
		"https://jobs.apple.com/zh-cn",
		"https://careers.microsoft.com/us/en",
		"https://www.nvidia.com/careers",
		"https://jobs.bytedance.com/experienced",
		"https://careers.tencent.com/jobdesc.html",
		"https://talent.alibaba.com/off-campus",
	} {
		bmp, err := demoFaviconLoader(context.Background(), rawURL)
		if err != nil || bmp == nil {
			t.Fatalf("demoFaviconLoader(%q) = (%v, %v), want non-nil bitmap", rawURL, bmp, err)
		}
	}

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
	if sfx.ScoreRMSDB < -16.5 || sfx.ScoreRMSDB > -11.5 {
		t.Fatalf("ScoreRMSDB = %.2f dBFS, want within [-16.5, -11.5]", sfx.ScoreRMSDB)
	}

	comp := buildComposition(assets, sfx)
	for _, frame := range []int{0, 110, 400, 1420, 1975, 2680, 2950, totalFrames - 1} {
		img, err := render.Still(comp, frame, render.Options{})
		if err != nil {
			t.Fatalf("render still frame %d: %v", frame, err)
		}
		if img.Bounds().Dx() != videoWidth || img.Bounds().Dy() != videoHeight {
			t.Fatalf("frame %d bounds = %v, want %dx%d", frame, img.Bounds(), videoWidth, videoHeight)
		}
	}
}
