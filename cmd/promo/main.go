// Promo renders an introductory product video for Aion using MyGo's native
// render plugin: 1920×1080 at 60 FPS with GPU rendering, native hardware
// encoding, synthesized stereo audio, and live-captured Retina Aion UI screens.
//
//	go run ./cmd/promo                          # renders aion-promo.mp4
//	go run ./cmd/promo -preview                 # previews in a native window
//	go run ./cmd/promo -stills /tmp/aion-stills # exports act keyframe PNGs
//	go run ./cmd/promo -o out.mov -codec prores # exports ProRes MOV
package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/plugins/render"
	"github.com/egoist/mygo/ui"
)

const (
	videoWidth  = 1920
	videoHeight = 1080
	videoFPS    = 60
	videoScale  = 2 // 960×540 logical DIPs (Retina 2x)
	totalFrames = 60 * videoFPS
)

func main() {
	out := flag.String("o", "aion-promo.mp4", "output video (.mp4, .mov, .webm) or image sequence pattern (frames/%04d.png)")
	codec := flag.String("codec", "h264", "codec: h264, hevc, prores, prores4444, vp9")
	renderer := flag.String("renderer", "auto", "renderer: auto, gpu, cpu")
	bitrate := flag.Int("bitrate", 16_000_000, "video bitrate in bits per second (0 for encoder default)")
	preview := flag.Bool("preview", false, "play the composition in an interactive native preview window")
	stillsDir := flag.String("stills", "", "export keyframe still PNGs for every act to this directory and exit")
	flag.Parse()

	start := time.Now()
	fmt.Println("Capturing Retina Aion UI screens from real seeded Workbench...")
	assets, err := captureDemoAssets()
	if err != nil {
		log.Fatalf("capture demo assets: %v", err)
	}
	fmt.Printf("Captured 8 Retina UI screens in %v\n", time.Since(start).Round(time.Millisecond))

	sounds := buildSoundBank()
	comp := buildComposition(assets, sounds)

	if *preview {
		previewWindow(comp)
		return
	}

	opts := render.Options{
		Renderer: map[string]render.Renderer{
			"auto": render.Auto,
			"gpu":  render.GPU,
			"cpu":  render.CPU,
		}[strings.ToLower(*renderer)],
		Codec: map[string]render.Codec{
			"":           render.DefaultCodec,
			"h264":       render.H264,
			"hevc":       render.HEVC,
			"prores":     render.ProRes,
			"prores4444": render.ProRes4444,
			"vp9":        render.VP9,
		}[strings.ToLower(*codec)],
		Bitrate:      *bitrate,
		AudioBitrate: 256_000,
		Progress: func(p render.Progress) {
			if p.Done%60 == 0 || p.Done == p.Frames {
				fpsRate := float64(p.Done) / max(p.Elapsed.Seconds(), 0.001)
				fmt.Printf("\r[%s] %d/%d frames (%.1f%%) · %.0f fps", p.Renderer, p.Done, p.Frames, 100*float64(p.Done)/float64(p.Frames), fpsRate)
			}
		},
	}

	if *stillsDir != "" {
		if err := exportActStills(comp, *stillsDir, opts); err != nil {
			log.Fatalf("export stills: %v", err)
		}
		return
	}

	renderStart := time.Now()
	if strings.Contains(*out, "%") {
		err = render.Images(context.Background(), comp, *out, opts)
	} else {
		err = render.Video(context.Background(), comp, *out, opts)
	}
	fmt.Println()
	if err != nil {
		log.Fatalf("render failed: %v", err)
	}
	abs, _ := filepath.Abs(*out)
	info, _ := os.Stat(abs)
	sizeMB := 0.0
	if info != nil {
		sizeMB = float64(info.Size()) / (1024 * 1024)
	}
	fmt.Printf("Rendered %s (%.2f MB) in %v\n", abs, sizeMB, time.Since(renderStart).Round(time.Millisecond))
}

func exportActStills(comp render.Composition, dir string, opts render.Options) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keyframes := []struct {
		name  string
		frame int
	}{
		{"01_brand_opening", 160},
		{"02_today_focus", 520},
		{"03a_kanban_board", 900},
		{"03b_task_detail_drawer", 1220},
		{"04a_jobs_tracker", 1560},
		{"04b_job_timeline_modal", 1860},
		{"05a_light_dark_sweep", 2145},
		{"05b_command_palette", 2440},
		{"06_mcp_cli_sync", 2900},
		{"07_apple_pillars", 3220},
		{"08_finale_outro", 3460},
	}
	for _, kf := range keyframes {
		img, err := render.Still(comp, kf.frame, opts)
		if err != nil {
			return fmt.Errorf("still %s (frame %d): %w", kf.name, kf.frame, err)
		}
		p := filepath.Join(dir, kf.name+".png")
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		fmt.Printf("Saved %s (frame %d)\n", p, kf.frame)
	}
	return nil
}

func previewWindow(comp render.Composition) {
	player, err := render.NewPlayer(comp)
	if err != nil {
		log.Fatal(err)
	}
	player.SetLoop(true)
	mygo.App.SetName("Aion Promo Preview")
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:  "Aion Promo Preview",
			Width:  1120,
			Height: 680,
			Content: ui.View(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(16).Gap(12).Background(ui.Hex("#000000")).Children(func() {
					ui.Video(c, player).Grow(1).FillWidth().Radius(12)
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						label := "暂停"
						if !player.Playing() {
							label = "播放"
						}
						if ui.Button(c, label).Clicked() {
							if player.Playing() {
								player.Pause()
							} else {
								player.Play()
							}
						}
						at := player.Position().Seconds()
						if ui.Slider(c, &at, 0, player.Duration().Seconds()).Grow(1).Changed() {
							player.Seek(time.Duration(at * float64(time.Second)))
						}
						ui.Text(c, fmt.Sprintf("%.1fs / %.0fs", at, player.Duration().Seconds())).
							TextColor(ui.Hex("#86868B")).Width(96)
					})
				})
			}),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
