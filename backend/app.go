// Package backend exposes Aion's local application services to a native host.
// It does not import UI packages, listen on a port, or contact a remote service.
package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aion/backend/attachments"
	"aion/backend/backup"
	"aion/backend/projects"
	"aion/backend/store"
)

type Options struct {
	// DataDir defaults to macOS Application Support/Aion. Tests must supply a temp directory.
	DataDir string
	// DisableScheduler stops background checks without changing the persisted policy.
	DisableScheduler bool
	// Now supplies the clock for automatic backup scheduling. Defaults to time.Now.
	Now func() time.Time
}
type App struct {
	Projects    *projects.Service
	Attachments *attachments.Service
	Backups     *backup.Service
	Settings    *Settings
	Modules     *Registry
	s           *store.Store
	cancel      context.CancelFunc
	workers     sync.WaitGroup
	once        sync.Once
	closeErr    error
	errors      chan error
}

func DefaultDataDir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Aion"), nil
}
func Open(ctx context.Context, options Options) (*App, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := options.DataDir
	if root == "" {
		var err error
		root, err = DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}
	s, err := store.Open(root)
	if err != nil {
		return nil, err
	}
	a := &App{Projects: projects.New(s), Attachments: attachments.New(s), Backups: backup.New(s, backup.Options{Now: options.Now}), Settings: &Settings{s: s}, Modules: &Registry{}, s: s, errors: make(chan error, 1)}
	if err = a.Modules.Register(ModuleDescriptor{ID: "projects", Name: "项目", Version: 1}); err != nil {
		s.Close()
		return nil, err
	}
	workerCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	if !options.DisableScheduler {
		a.workers.Add(1)
		go a.schedule(workerCtx)
	}
	return a, nil
}
func (a *App) DataDir() string { return a.s.Root() }

// PermanentlyDelete purges a recycled item and then reclaims files that no
// module references. A cleanup error does not undo the database deletion; GC
// can be retried safely. The host owns the user confirmation interaction.
func (a *App) PermanentlyDelete(ctx context.Context, kind, id string) error {
	if err := a.Projects.PermanentlyDelete(ctx, kind, id); err != nil {
		return err
	}
	if _, err := a.Attachments.GC(ctx); err != nil {
		return fmt.Errorf("item deleted; attachment cleanup: %w", err)
	}
	return nil
}

// ClearTrash purges all project recycle-bin contents, then collects unused files.
func (a *App) ClearTrash(ctx context.Context) error {
	if err := a.Projects.ClearTrash(ctx); err != nil {
		return err
	}
	if _, err := a.Attachments.GC(ctx); err != nil {
		return fmt.Errorf("trash cleared; attachment cleanup: %w", err)
	}
	return nil
}

// Errors reports background backup errors without blocking service shutdown.
// backup.Status remains the persisted source of the last success/failure.
func (a *App) Errors() <-chan error { return a.errors }
func (a *App) schedule(ctx context.Context) {
	defer a.workers.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if _, err := a.Backups.RunDue(ctx); err != nil && ctx.Err() == nil {
			select {
			case a.errors <- fmt.Errorf("automatic backup: %w", err):
			default:
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Close cancels and joins the in-process backup scheduler before closing SQLite.
// Call after UI commands have drained. Repeated calls are safe.
func (a *App) Close() error {
	a.once.Do(func() { a.cancel(); a.workers.Wait(); a.closeErr = a.s.Close(); close(a.errors) })
	return a.closeErr
}
