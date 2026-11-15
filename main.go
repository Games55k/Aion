package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"aion/assets/icon"
	"aion/backend"
	"aion/frontend"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
)

const releasesURL = "https://github.com/Games55k/Aion/releases"

func main() {
	dataDir := flag.String("data-dir", "", "custom local data directory (defaults to ~/Library/Application Support/Aion)")
	snapshotPath := flag.String("snapshot", "", "render a headless PNG snapshot of the UI to this path and exit")
	flag.Parse()

	if err := run(*dataDir, *snapshotPath); err != nil {
		log.Fatalf("aion: %v", err)
	}
}

func run(dataDir, snapshotPath string) error {
	ctx := context.Background()
	app, err := backend.Open(ctx, backend.Options{
		DataDir:          dataDir,
		DisableScheduler: snapshotPath != "",
	})
	if err != nil {
		return fmt.Errorf("failed to open backend: %w", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("aion: failed to close backend cleanly: %v", err)
		}
	}()

	wb := frontend.New(app, frontend.Host{})
	defer func() {
		wb.WaitForTasks()
		if err := wb.FlushPending(); err != nil {
			log.Printf("aion: failed to flush pending edits: %v", err)
		}
	}()

	// Headless snapshot rendering mode for verification
	if snapshotPath != "" {
		return writeSnapshot(snapshotPath, app.DataDir(), wb.View)
	}

	// Sync initial saved theme before starting the native desktop window
	if savedTheme, err := app.Settings.Theme(ctx); err == nil {
		applyNativeThemeSource(savedTheme)
	}

	if dataDir != "" {
		mygo.App.SetPath(mygo.PathUserData, app.DataDir())
	}
	mygo.App.SetLanguages("zh-Hans")
	mygo.App.SetName("Aion")
	mygo.Use(native.New(updater.Options{
		DisableAutomaticChecks: true,
		Icon:                   icon.PNG,
		Language:               "zh-Hans",
	}))

	mygo.App.WhenReady(func() {
		var openSettings func()
		triggerOpenSettings := func() {
			if openSettings != nil {
				openSettings()
			}
		}
		mygo.App.SetMenu(buildAppMenu(triggerOpenSettings))
		mygo.App.OnLocaleChanged(func(string) {
			mygo.App.SetMenu(buildAppMenu(triggerOpenSettings))
		})

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

		openSettings = func() {
			if win.IsMinimized() {
				win.Restore()
			}
			if !win.IsVisible() {
				win.Show()
			}
			if !win.IsFocused() {
				win.Focus()
			}
			win.Update(func() {
				wb.OpenSettings()
			})
		}

		readUpdateInfo := func() frontend.UpdateCapability {
			return frontend.UpdateCapability{
				Version:          mygo.App.Version(),
				AvailableVersion: updater.AvailableVersion(),
				LastCheck:        updater.LastCheck(),
				IsDev:            mygo.IsDev(),
				CanSelfUpdate:    mygo.Updater.Enabled(),
				ReleasesURL:      releasesURL,
			}
		}

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
			OpenURL: func(url string) error {
				return mygo.Shell.OpenExternal(url)
			},
			SetTheme: func(th string) {
				applyNativeThemeSource(th)
			},
			UpdateInfo:      readUpdateInfo(),
			CheckForUpdates: updater.CheckForUpdates,
		})

		offUpdaterChange := updater.OnChange(func() {
			win.Update(func() {
				wb.SetUpdateCapability(readUpdateInfo())
			})
		})

		win.OnClose(func(ev *mygo.CloseEvent) {
			if !wb.PrepareClose() {
				ev.PreventDefault()
				win.Show()
				win.Focus()
				win.Update(func() {})
				return
			}
			wb.WaitForTasks()
		})

		mygo.App.OnQuit(func() {
			offUpdaterChange()
			wb.WaitForTasks()
			if err := wb.FlushPending(); err != nil {
				log.Printf("aion: failed to flush pending edits on quit: %v", err)
			}
			if err := app.Close(); err != nil {
				log.Printf("aion: failed to close backend cleanly on quit: %v", err)
			}
		})

		// Listen for background scheduler errors and surface them in the UI
		go func() {
			for bgErr := range app.Errors() {
				if bgErr == nil {
					continue
				}
				win.Update(func() {
					wb.RecordAsyncError(bgErr)
				})
			}
		}()
	})

	if err := mygo.App.Run(); err != nil {
		return fmt.Errorf("app run error: %w", err)
	}
	return nil
}

// writeSnapshot keeps diagnostic output outside managed data and publishes only
// a complete PNG, so an invalid target or encoding failure cannot truncate data.
func writeSnapshot(path, dataDir string, view func(*ui.Context)) error {
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return errors.New("snapshot destination must name a file")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(dataDir)
	if err != nil {
		return err
	}
	path = filepath.Join(parent, filepath.Base(absPath))
	rootInfo, err := os.Stat(root)
	if err != nil {
		return err
	}
	// Compare directory identity as well as resolving symlinks: macOS commonly
	// permits multiple spellings of the same path on a case-insensitive volume.
	for dir := parent; ; dir = filepath.Dir(dir) {
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if os.SameFile(info, rootInfo) {
			return errors.New("snapshot destination must be outside the Aion data directory")
		}
		if dir == filepath.Dir(dir) {
			break
		}
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("snapshot destination must be a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(parent, ".aion-snapshot-*.png")
	if err != nil {
		return fmt.Errorf("create snapshot file: %w", err)
	}
	defer os.Remove(f.Name())
	img := ui.Render(view, 1360, 860, 1.0)
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode snapshot png: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync snapshot file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close snapshot file: %w", err)
	}
	return os.Rename(f.Name(), path)
}

func buildAppMenu(onOpenSettings func()) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: mygo.App.Name(), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			updater.MenuItem(),
			mygo.Separator(),
			{
				Label:       "设置…",
				Accelerator: "CmdOrCtrl+,",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if onOpenSettings != nil {
						onOpenSettings()
					}
				},
			},
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleFileMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
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
