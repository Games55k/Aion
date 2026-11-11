package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aion/backend/model"
)

const (
	policyKey         = "backup.auto.policy"
	maxStateBytes     = 2 << 20
	maxLastErrorBytes = 4096
)

type Policy struct {
	Enabled   bool          `json:"enabled"`
	Interval  time.Duration `json:"interval"`
	Keep      int           `json:"keep"`
	Directory string        `json:"directory"`
}

type AutoStatus struct {
	LastSuccess time.Time `json:"last_success"`
	LastError   string    `json:"last_error"`
}
type ownedBackup struct {
	Path      string    `json:"path"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}
type autoState struct {
	Version int `json:"version"`
	AutoStatus
	Files []ownedBackup `json:"files"`
}

func (s *Service) statePath() string { return filepath.Join(s.store.Root(), ".backup-auto-state.json") }
func (s *Service) defaultPolicy() Policy {
	return Policy{Enabled: true, Interval: 24 * time.Hour, Keep: 7, Directory: filepath.Join(s.store.Root(), "backups")}
}
func (s *Service) normalizePolicy(p Policy) (Policy, error) {
	if p.Interval < time.Minute || p.Interval > 365*24*time.Hour || p.Keep < 1 || p.Keep > 365 {
		return p, invalid("interval must be 1m..365d and keep must be 1..365")
	}
	if p.Directory == "" {
		p.Directory = s.defaultPolicy().Directory
	}
	var err error
	p.Directory, err = s.store.ValidateBackupDir(p.Directory)
	if err != nil {
		return p, err
	}
	return p, nil
}
func (s *Service) policyLocked(ctx context.Context, db *sql.DB) (Policy, error) {
	var raw string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", policyKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return s.defaultPolicy(), nil
	}
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	if len(raw) > 8192 {
		return p, invalid("policy too large")
	}
	if err = decodeJSON([]byte(raw), &p); err != nil {
		return p, invalid("invalid automatic backup policy: %v", err)
	}
	return s.normalizePolicy(p)
}
func (s *Service) GetPolicy(ctx context.Context) (Policy, error) {
	var p Policy
	err := s.store.Exclusive(ctx, func(db *sql.DB) error { var err error; p, err = s.policyLocked(ctx, db); return err })
	return p, err
}
func (s *Service) SetPolicy(ctx context.Context, p Policy) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		normalized, err := s.normalizePolicy(p)
		if err != nil {
			return err
		}
		// Persist the default directory as "" so it keeps resolving against the
		// current store root after the database is restored into another one.
		if normalized.Directory == s.defaultPolicy().Directory {
			normalized.Directory = ""
		}
		b, err := json.Marshal(normalized)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", policyKey, string(b))
		return err
	})
}

func (s *Service) loadState() (autoState, error) {
	st := autoState{Version: 1}
	f, err := openRegular(s.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxStateBytes+1))
	if err != nil {
		return st, err
	}
	if len(b) > maxStateBytes {
		return st, invalid("automatic state too large")
	}
	if err = decodeJSON(b, &st); err != nil {
		return st, err
	}
	if st.Version != 1 || len(st.Files) > 4096 {
		return st, invalid("unsupported automatic backup state")
	}
	for _, item := range st.Files {
		if !filepath.IsAbs(item.Path) || !validHash(item.SHA256) || !strings.HasPrefix(filepath.Base(item.Path), "auto-") || filepath.Ext(item.Path) != ".zip" || item.CreatedAt.IsZero() {
			return st, invalid("invalid automatic backup ownership")
		}
	}
	return st, nil
}
func (s *Service) saveState(st autoState) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	if len(b) > maxStateBytes {
		return invalid("automatic state too large")
	}
	if _, err = regular(s.statePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(s.store.Root(), ".backup-state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := f.Name()
	closed := false
	renamed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if !renamed {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err = f.Write(b); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	closed = true
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, s.statePath()); err != nil {
		return err
	}
	renamed = true
	return nil
}
func (s *Service) Status(ctx context.Context) (AutoStatus, error) {
	var status AutoStatus
	err := s.store.Exclusive(ctx, func(_ *sql.DB) error { st, err := s.loadState(); status = st.AutoStatus; return err })
	return status, err
}

func archiveHash(ctx context.Context, path string) (string, error) {
	f, err := openRegular(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, h, err := copyHash(ctx, io.Discard, f, MaxArchiveBytes)
	return h, err
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	var end int
	for i := range s {
		if i > maxBytes {
			break
		}
		end = i
	}
	return s[:end]
}

// rotate only touches paths recorded locally after successful creation; neither
// filenames nor an archive's own manifest establish ownership. Hash mismatches
// lose ownership and are preserved (a user may have replaced a file manually).
func (s *Service) rotate(ctx context.Context, st *autoState, keep int) error {
	sort.SliceStable(st.Files, func(i, j int) bool { return st.Files[i].CreatedAt.Before(st.Files[j].CreatedAt) })
	for len(st.Files) > keep {
		if err := ctx.Err(); err != nil {
			return err
		}
		old := st.Files[0]
		h, err := archiveHash(ctx, old.Path)
		if errors.Is(err, os.ErrNotExist) {
			st.Files = st.Files[1:]
			continue
		}
		if err != nil {
			// A replaced symlink/special file is never removed.
			if errors.Is(err, model.ErrInvalid) {
				st.Files = st.Files[1:]
				continue
			}
			return err
		}
		if h != old.SHA256 {
			st.Files = st.Files[1:]
			continue
		}
		if err = os.Remove(old.Path); err != nil {
			return err
		}
		st.Files = st.Files[1:]
	}
	return nil
}

// RunDue is synchronous. The application owns the periodic goroutine and must
// stop it before Store.Close. Last-success/error never come from restored SQL.
func (s *Service) RunDue(ctx context.Context) (string, error) {
	var created string
	err := s.store.Exclusive(ctx, func(db *sql.DB) error {
		st, err := s.loadState()
		if err != nil {
			return err
		}
		fail := func(cause error) error {
			st.LastError = truncateUTF8(cause.Error(), maxLastErrorBytes)
			return errors.Join(cause, s.saveState(st))
		}
		p, err := s.policyLocked(ctx, db)
		if err != nil {
			return fail(err)
		}
		now := time.Now().UTC()
		if !p.Enabled || (!st.LastSuccess.IsZero() && now.Before(st.LastSuccess.Add(p.Interval))) {
			return nil
		}
		if err = os.MkdirAll(p.Directory, 0700); err != nil {
			return fail(err)
		}
		path := filepath.Join(p.Directory, "auto-"+model.NewID()+".zip")
		h, err := s.exportLocked(ctx, db, path, "auto", false)
		if err != nil {
			return fail(err)
		}
		created = path
		st.LastSuccess = now
		st.LastError = ""
		st.Files = append(st.Files, ownedBackup{Path: path, SHA256: h, CreatedAt: now})
		// Persist success and ownership BEFORE deleting anything. If persistence
		// fails all previous valid archives remain intact.
		if err = s.saveState(st); err != nil {
			return fmt.Errorf("save automatic backup status: %w", err)
		}
		if err = s.rotate(ctx, &st, p.Keep); err != nil {
			return fail(fmt.Errorf("rotate automatic backups: %w", err))
		}
		return s.saveState(st)
	})
	return created, err
}
