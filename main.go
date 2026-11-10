package main

import (
	"context"
	"flag"
	"image/png"
	"log"
	"os"

	"aion/backend"
	"aion/frontend"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	dataDir := flag.String("data-dir", "", "custom local data directory (defaults to ~/Library/Application Support/Aion)")
	snapshotPath := flag.String("snapshot", "", "render a headless PNG snapshot of the UI to this path and exit")
	flag.Parse()

	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{
		DataDir: *dataDir,
	})
	if err != nil {
		log.Fatalf("aion: failed to open backend: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("aion: failed to close backend cleanly: %v", err)
		}
	}()

	wb := frontend.New(app, frontend.Host{})
	defer func() {
		wb.WaitForTasks()
		_ = wb.FlushPending()
	}()

	// Headless snapshot rendering mode for verification
	if *snapshotPath != "" {
		img := ui.Render(wb.View, 1360, 860, 1.0)
		f, err := os.Create(*snapshotPath)
		if err != nil {
			log.Fatalf("aion: create snapshot file: %v", err)
		}
		if err := png.Encode(f, img); err != nil {
			_ = f.Close()
			log.Fatalf("aion: encode snapshot png: %v", err)
		}
		_ = f.Close()
		return
	}

	// Sync initial saved theme before starting the native desktop window
	if savedTheme, err := app.Settings.Theme(ctx); err == nil {
		applyNativeThemeSource(savedTheme)
	}

	mygo.App.SetName("Aion")
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:                "Aion",
			Width:                1320,
			Height:               840,
			MinWidth:             960,
			MinHeight:            620,
			TitleBarStyle:        mygo.TitleBarHiddenInset,
			TrafficLightPosition: &mygo.Point{X: 13, Y: 14},
			Content:              ui.View(wb.View),
		})

		wb.SetHost(frontend.Host{
			Dispatch: win.Update,
			PickOpenFiles: func(title string, multiple bool, extensions ...string) ([]string, error) {
				var filters []mygo.FileFilter
				if len(extensions) > 0 {
					filters = []mygo.FileFilter{{Name: "Files", Extensions: extensions}}
				}
				return mygo.Dialog.Open(mygo.OpenDialogOptions{
					Parent:   win,
					Title:    title,
					Multiple: multiple,
					Filters:  filters,
				})
			},
			PickDirectory: func(title, defaultPath string) (string, error) {
				paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
					Parent:            win,
					Title:             title,
					DefaultPath:       defaultPath,
					Directory:         true,
					CreateDirectories: true,
				})
				if err != nil || len(paths) == 0 {
					return "", err
				}
				return paths[0], nil
			},
			PickSaveFile: func(title, defaultName string, extensions ...string) (string, error) {
				var filters []mygo.FileFilter
				if len(extensions) > 0 {
					filters = []mygo.FileFilter{{Name: "Files", Extensions: extensions}}
				}
				return mygo.Dialog.Save(mygo.SaveDialogOptions{
					Parent:            win,
					Title:             title,
					DefaultPath:       defaultName,
					Filters:           filters,
					CreateDirectories: true,
				})
			},
			OpenPath: func(path string) error {
				return mygo.Shell.OpenPath(path)
			},
			RevealPath: func(path string) error {
				mygo.Shell.ShowItemInFolder(path)
				return nil
			},
			SetTheme: func(th string) {
				applyNativeThemeSource(th)
			},
		})

		win.OnClose(func(ev *mygo.CloseEvent) {
			if !wb.PrepareClose() {
				ev.PreventDefault()
				return
			}
			wb.WaitForTasks()
		})

		// Listen for background scheduler errors and surface them in the UI
		go func() {
			for bgErr := range app.Errors() {
				if bgErr == nil {
					continue
				}
				errCopy := bgErr
				win.Update(func() {
					wb.RecordAsyncError(errCopy)
				})
			}
		}()
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatalf("aion: app run error: %v", err)
	}
}

func applyNativeThemeSource(th string) {
	switch th {
	case frontend.ThemeLight:
		mygo.Theme.SetSource(mygo.ThemeLight)
	case frontend.ThemeDark:
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}
