package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"aion/backend"
	"aion/cli"
	"aion/frontend"
	"aion/ipc"
	"aion/mcp"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
)

const releasesURL = "https://github.com/Games55k/Aion/releases"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		if err := runMCP(os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "aion mcp: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && cli.IsSubcommand(os.Args[1]) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
		stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "aion: %v\n", err)
			os.Exit(1)
		}
		return
	}

	dataDir := flag.String("data-dir", "", "custom local data directory (defaults to ~/Library/Application Support/Aion)")
	snapshotPath := flag.String("snapshot", "", "render a headless PNG snapshot of the UI to this path and exit")
	flag.Parse()

	if err := run(*dataDir, *snapshotPath); err != nil {
		log.Fatalf("aion: %v", err)
	}
}

func runMCP(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataDir := fs.String("data-dir", "", "custom local data directory (defaults to ~/Library/Application Support/Aion)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	router, err := ipc.NewRouter(ipc.RouterOptions{DataDir: *dataDir})
	if err != nil {
		return fmt.Errorf("init local router: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := mcp.NewServer(router, mcp.ServerInfo{Name: "aion", Version: mygo.App.Version()})
	err = srv.Serve(ctx, in, out)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func resolveExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
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

	wb := frontend.New(app, frontend.Host{})
	var (
		ipcMu  sync.Mutex
		ipcSrv *ipc.Server
	)
	closeIPC := func() {
		ipcMu.Lock()
		srv := ipcSrv
		ipcSrv = nil
		ipcMu.Unlock()
		if srv != nil {
			_ = srv.Close()
		}
	}
	shutdown := newShutdownFunc(wb, app, closeIPC)
	defer shutdown()

	// Headless snapshot rendering mode for verification
	if snapshotPath != "" {
		return writeSnapshot(snapshotPath, app.DataDir(), wb.View)
	}

	// Sync initial saved preferences before starting the native desktop window
	if savedTheme, err := app.Settings.Theme(ctx); err == nil {
		applyNativeThemeSource(savedTheme)
	}
	initialUpdatePolicy := backend.DefaultUpdateCheckPolicy()
	if savedPolicy, err := app.Settings.UpdateCheckPolicy(ctx); err == nil {
		initialUpdatePolicy = savedPolicy
	}

	if dataDir != "" {
		mygo.App.SetPath(mygo.PathUserData, app.DataDir())
	}
	mygo.App.SetLanguages("zh-Hans")
	mygo.App.SetName("Aion")

	mygo.App.WhenReady(func() {
		var openSettings func()
		var checkUpdates func()
		triggerOpenSettings := func() {
			if openSettings != nil {
				openSettings()
			}
		}
		triggerCheckUpdates := func() {
			if checkUpdates != nil {
				checkUpdates()
			}
		}
		mygo.App.SetMenu(buildAppMenu(triggerOpenSettings, triggerCheckUpdates))
		mygo.App.OnLocaleChanged(func(string) {
			mygo.App.SetMenu(buildAppMenu(triggerOpenSettings, triggerCheckUpdates))
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

		focusWindow := func() {
			if win.IsMinimized() {
				win.Restore()
			}
			if !win.IsVisible() {
				win.Show()
			}
			if !win.IsFocused() {
				win.Focus()
			}
		}

		openSettings = func() {
			focusWindow()
			win.Update(func() {
				wb.OpenSettings()
			})
		}

		checkUpdates = func() {
			focusWindow()
			win.Update(func() {
				wb.CheckForUpdatesAndShow()
			})
		}

		updCtrl := newUpdateController(updateControllerOptions{
			DataDir:       app.DataDir(),
			Version:       mygo.App.Version(),
			IsDev:         mygo.IsDev(),
			CanSelfUpdate: mygo.Updater.Enabled(),
			ReleasesURL:   releasesURL,
			Policy:        initialUpdatePolicy,
			Check: func(checkCtx context.Context) (*updateRelease, error) {
				up, err := mygo.Updater.Check(checkCtx)
				if err != nil || up == nil {
					return nil, err
				}
				return &updateRelease{
					Version: up.Version,
					Notes:   up.Notes,
					Date:    up.Date,
					Install: up.Install,
				}, nil
			},
			Relaunch: func() {
				mygo.App.Relaunch()
			},
			OnChange: func(info frontend.UpdateCapability) {
				win.Update(func() {
					wb.SetUpdateCapability(info)
				})
			},
		})

		mcpCap := frontend.MCPCapability{
			ExecutablePath: resolveExecutablePath(),
			DataDirArg:     dataDir,
		}
		if srv, err := ipc.StartServer(app.DataDir(), ipc.NewHandler(app, func(ev ipc.ChangeEvent) {
			win.Update(func() {
				wb.HandleExternalChange(frontend.ExternalChange{
					Scope:     string(ev.Scope),
					ProjectID: ev.ProjectID,
					TaskID:    ev.TaskID,
					JobID:     ev.JobID,
				})
			})
		})); err != nil {
			mcpCap.LastError = err.Error()
			log.Printf("aion: local IPC server unavailable: %v", err)
		} else {
			ipcMu.Lock()
			ipcSrv = srv
			ipcMu.Unlock()
			mcpCap.Ready = true
			mcpCap.SocketPath = srv.SocketPath()
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
			UpdateInfo:           updCtrl.Snapshot(),
			MCPInfo:              mcpCap,
			CheckForUpdates:      updCtrl.CheckForUpdates,
			InstallUpdate:        updCtrl.InstallUpdate,
			CancelUpdate:         updCtrl.CancelUpdate,
			SkipUpdateVersion:    updCtrl.SkipVersion,
			RelaunchToUpdate:     updCtrl.Relaunch,
			SetUpdateCheckPolicy: updCtrl.SetPolicy,
			LoadFavicon:          frontend.DefaultFaviconLoader,
		})

		offResume := mygo.Power.OnResume(updCtrl.OnResume)
		updCtrl.Start()

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
			offResume()
			updCtrl.Close()
			shutdown()
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

func newShutdownFunc(wb *frontend.Workbench, app *backend.App, beforeClose ...func()) func() {
	return sync.OnceFunc(func() {
		for _, fn := range beforeClose {
			if fn != nil {
				fn()
			}
		}
		wb.WaitForTasks()
		if err := wb.FlushPending(); err != nil {
			log.Printf("aion: failed to flush pending edits: %v", err)
		}
		wb.WaitForTasks()
		if err := app.Close(); err != nil {
			log.Printf("aion: failed to close backend cleanly: %v", err)
		}
	})
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

func buildAppMenu(onOpenSettings, onCheckUpdates func()) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: mygo.App.Name(), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			{
				Label: "检查更新…",
				Click: func(*mygo.MenuItem, *mygo.Window) {
					if onCheckUpdates != nil {
						onCheckUpdates()
					}
				},
			},
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
