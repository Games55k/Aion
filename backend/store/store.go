// Package store owns SQLite lifecycle and the shared consistency lock.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"aion/backend/model"

	"golang.org/x/sys/unix"
	_ "modernc.org/sqlite"
)

const SchemaVersion = 1

type Store struct {
	mu     sync.Mutex
	db     *sql.DB
	root   string
	lock   *os.File
	closed bool
}

func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("data directory is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	for _, p := range []string{root, filepath.Join(root, "attachments")} {
		if st, e := os.Lstat(p); e == nil && (st.Mode()&os.ModeSymlink != 0 || !st.IsDir()) {
			return nil, fmt.Errorf("unsafe data directory: %s", p)
		}
	}
	if err = os.MkdirAll(filepath.Join(root, "attachments"), 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(root, ".aion.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("data directory is already open: %w", err)
	}
	s := &Store{root: root, lock: lock}
	s.db, err = openDB(s.DBPath())
	if err == nil {
		err = s.migrate()
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func openDB(path string) (*sql.DB, error) {
	if st, e := os.Lstat(path); e == nil && (!st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0) {
		return nil, errors.New("unsafe database path")
	}
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err = db.Ping(); err == nil {
		_, err = db.Exec("PRAGMA journal_mode=DELETE")
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func (s *Store) Root() string           { return s.root }
func (s *Store) DBPath() string         { return filepath.Join(s.root, "aion.sqlite") }
func (s *Store) AttachmentsDir() string { return filepath.Join(s.root, "attachments") }

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > SchemaVersion {
		return fmt.Errorf("unsupported database version %d", version)
	}
	if version == SchemaVersion {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range schema {
		if _, err = tx.Exec(statement); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("PRAGMA user_version=1"); err != nil {
		return err
	}
	return tx.Commit()
}

var schema = []string{
	`CREATE TABLE projects(id TEXT PRIMARY KEY,name TEXT NOT NULL,icon TEXT NOT NULL,description TEXT NOT NULL,archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN(0,1)),created_at TEXT NOT NULL,updated_at TEXT NOT NULL,deleted_at TEXT)`,
	`CREATE TABLE tasks(id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,title TEXT NOT NULL,description TEXT NOT NULL,status TEXT NOT NULL CHECK(status IN('planned','in_progress','done')),priority TEXT NOT NULL CHECK(priority IN('none','low','medium','high','urgent')),tags TEXT NOT NULL DEFAULT '[]',subtasks TEXT NOT NULL DEFAULT '[]',due_date TEXT NOT NULL DEFAULT '',position INTEGER NOT NULL DEFAULT 0,created_at TEXT NOT NULL,updated_at TEXT NOT NULL,completed_at TEXT,deleted_at TEXT)`,
	`CREATE INDEX tasks_board ON tasks(project_id,status,position)`,
	`CREATE TABLE attachments(id TEXT PRIMARY KEY,task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,name TEXT NOT NULL,storage_name TEXT NOT NULL UNIQUE,media_type TEXT NOT NULL,size INTEGER NOT NULL CHECK(size>=0),sha256 TEXT NOT NULL,created_at TEXT NOT NULL)`,
	`CREATE INDEX attachments_task ON attachments(task_id)`,
	`CREATE TABLE settings(key TEXT PRIMARY KEY,value TEXT NOT NULL)`,
}

func (s *Store) Read(ctx context.Context, fn func(*sql.Tx) error) error {
	return s.transaction(ctx, fn)
}
func (s *Store) Write(ctx context.Context, fn func(*sql.Tx) error) error {
	return s.transaction(ctx, fn)
}
func (s *Store) transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	return s.Exclusive(ctx, func(db *sql.DB) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err = fn(tx); err != nil {
			return err
		}
		return tx.Commit()
	})
}

// Exclusive serializes database AND managed-file operations. Callbacks must not
// reenter Store methods which acquire the lock.
func (s *Store) Exclusive(ctx context.Context, fn func(*sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("store is closed")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn(s.db)
}
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	var err error
	if s.db != nil {
		err = s.db.Close()
	}
	if s.lock != nil {
		err = errors.Join(err, unix.Flock(int(s.lock.Fd()), unix.LOCK_UN), s.lock.Close())
	}
	return err
}

// ReplaceDatabaseLocked atomically replaces the live database, reopening the old
// one on failure. Caller MUST hold Exclusive. Source is copied, never consumed.
func (s *Store) ReplaceDatabaseLocked(ctx context.Context, stagedPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src, err := os.Open(stagedPath)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(s.root, ".restore-db-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	_, err = io.Copy(tmp, src)
	if err == nil {
		err = tmp.Sync()
	}
	err = errors.Join(err, tmp.Close())
	if err != nil {
		return err
	}
	probe, err := openDB(name)
	if err != nil {
		return err
	}
	var v int
	var integrity string
	err = probe.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v)
	if err == nil {
		err = probe.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity)
	}
	probe.Close()
	if err != nil {
		return err
	}
	if v != SchemaVersion || integrity != "ok" {
		return errors.New("invalid restore database")
	}
	old := filepath.Join(s.root, ".previous-db-"+filepath.Base(name))
	if err = s.db.Close(); err != nil {
		return err
	}
	reopen := func() error { var e error; s.db, e = openDB(s.DBPath()); return e }
	if err = os.Rename(s.DBPath(), old); err != nil {
		return errors.Join(err, reopen())
	}
	rollback := func(cause error) error {
		os.Remove(s.DBPath())
		e := os.Rename(old, s.DBPath())
		return errors.Join(cause, e, reopen())
	}
	if err = os.Rename(name, s.DBPath()); err != nil {
		return rollback(err)
	}
	if err = reopen(); err != nil {
		return rollback(err)
	}
	_ = os.Remove(old)
	return nil
}

func validExportFileName(name string) bool {
	return strings.TrimSpace(name) != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, "/\\:\x00") && strings.IndexFunc(name, unicode.IsControl) < 0
}

func isWithinManagedDir(dirPath string, managedStat, rootStat os.FileInfo) bool {
	cur := filepath.Clean(dirPath)
	var prevChild string
	for {
		if st, err := os.Stat(cur); err == nil {
			if managedStat != nil && os.SameFile(st, managedStat) {
				return true
			}
			if rootStat != nil && os.SameFile(st, rootStat) && prevChild != "" {
				lowerChild := strings.ToLower(prevChild)
				if lowerChild == "attachments" || strings.HasPrefix(lowerChild, ".") {
					return true
				}
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		prevChild = filepath.Base(cur)
		cur = parent
	}
	return false
}

// ValidateBackupDir normalizes dirPath and verifies that an automatic backup directory
// does not point to the store root itself or inside managed attachment storage.
func (s *Store) ValidateBackupDir(dirPath string) (string, error) {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return "", fmt.Errorf("%w: backup directory is required", model.ErrInvalid)
	}
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return "", err
	}
	if st, err := os.Lstat(absDir); err == nil && (!st.IsDir() || st.Mode()&os.ModeSymlink != 0) {
		return "", fmt.Errorf("%w: backup directory must be a real directory", model.ErrInvalid)
	}
	managedStat, err := os.Lstat(s.AttachmentsDir())
	if err != nil {
		return "", err
	}
	rootStat, err := os.Stat(s.root)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(absDir); err == nil && os.SameFile(st, rootStat) {
		return "", fmt.Errorf("%w: backup directory cannot be the store root", model.ErrInvalid)
	}
	if filepath.Clean(absDir) == filepath.Clean(s.root) || isWithinManagedDir(absDir, managedStat, rootStat) {
		return "", fmt.Errorf("%w: backup directory overlaps protected store path", model.ErrInvalid)
	}
	return absDir, nil
}

// ValidateExportTarget normalizes destinationPath and verifies that it does not
// target or alias any managed store file (live SQLite database, WAL/SHM/journal sidecars,
// lock file, automatic backup state, internal staging files, or managed attachments directory).
// When allowOverwrite is false, any existing destination fails with model.ErrConflict.
// When allowOverwrite is true, only an existing regular file outside protected store paths may be replaced.
func (s *Store) ValidateExportTarget(destinationPath string, allowOverwrite bool) (string, error) {
	destinationPath = strings.TrimSpace(destinationPath)
	if destinationPath == "" {
		return "", fmt.Errorf("%w: destination is required", model.ErrInvalid)
	}
	base := filepath.Base(destinationPath)
	if !validExportFileName(base) {
		return "", fmt.Errorf("%w: invalid destination file name", model.ErrInvalid)
	}
	absDest, err := filepath.Abs(destinationPath)
	if err != nil {
		return "", err
	}
	managedStat, e := os.Lstat(s.AttachmentsDir())
	if e != nil {
		return "", e
	} else if !managedStat.IsDir() || managedStat.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%w: unsafe attachment directory", model.ErrInvalid)
	}
	rootStat, err := os.Stat(s.root)
	if err != nil {
		return "", err
	}
	managedReal, err := filepath.EvalSymlinks(s.AttachmentsDir())
	if err != nil {
		return "", err
	}
	rootReal, err := filepath.EvalSymlinks(s.root)
	if err != nil {
		rootReal = s.root
	}
	if isWithinManagedDir(filepath.Dir(absDest), managedStat, rootStat) {
		return "", fmt.Errorf("%w: export into managed attachment storage is forbidden", model.ErrInvalid)
	}
	if allowOverwrite {
		if _, statErr := os.Lstat(filepath.Dir(absDest)); errors.Is(statErr, os.ErrNotExist) {
			if mkErr := os.MkdirAll(filepath.Dir(absDest), 0o755); mkErr != nil {
				return "", mkErr
			}
		}
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absDest))
	if err != nil {
		return "", err
	}
	parentStat, err := os.Stat(parent)
	if err != nil {
		return "", err
	}
	resolvedDest := filepath.Join(parent, base)

	if isWithinManagedDir(parent, managedStat, rootStat) {
		return "", fmt.Errorf("%w: export into managed attachment storage is forbidden", model.ErrInvalid)
	}

	relManaged, err := filepath.Rel(managedReal, resolvedDest)
	if err != nil {
		return "", err
	}
	if relManaged == "." || (relManaged != ".." && !strings.HasPrefix(relManaged, ".."+string(os.PathSeparator))) {
		return "", fmt.Errorf("%w: export into managed attachment storage is forbidden", model.ErrInvalid)
	}

	protectedPaths := []string{
		s.DBPath(),
		s.DBPath() + "-wal",
		s.DBPath() + "-shm",
		s.DBPath() + "-journal",
		filepath.Join(s.root, ".aion.lock"),
		filepath.Join(s.root, ".backup-auto-state.json"),
		filepath.Join(rootReal, "aion.sqlite"),
		filepath.Join(rootReal, "aion.sqlite-wal"),
		filepath.Join(rootReal, "aion.sqlite-shm"),
		filepath.Join(rootReal, "aion.sqlite-journal"),
		filepath.Join(rootReal, ".aion.lock"),
		filepath.Join(rootReal, ".backup-auto-state.json"),
	}
	for _, p := range protectedPaths {
		if resolvedDest == p || absDest == p {
			return "", fmt.Errorf("%w: destination overlaps live store file: %s", model.ErrInvalid, base)
		}
	}

	if os.SameFile(parentStat, rootStat) {
		lowerBase := strings.ToLower(base)
		if strings.HasPrefix(lowerBase, "aion.sqlite") || strings.HasPrefix(base, ".") || lowerBase == "attachments" || lowerBase == "backups" {
			return "", fmt.Errorf("%w: destination overlaps protected store path: %s", model.ErrInvalid, base)
		}
	}

	if relRoot, err := filepath.Rel(rootReal, resolvedDest); err == nil {
		if relRoot == "." {
			return "", fmt.Errorf("%w: destination overlaps store root", model.ErrInvalid)
		}
		if filepath.Dir(relRoot) == "." {
			lowerBase := strings.ToLower(base)
			if strings.HasPrefix(lowerBase, "aion.sqlite") || strings.HasPrefix(base, ".") || lowerBase == "attachments" || lowerBase == "backups" {
				return "", fmt.Errorf("%w: destination overlaps protected store path: %s", model.ErrInvalid, base)
			}
		}
	}

	st, err := os.Lstat(resolvedDest)
	if err == nil {
		if !allowOverwrite {
			return "", errors.Join(model.ErrConflict, fmt.Errorf("destination already exists: %s: %w", resolvedDest, os.ErrExist))
		}
		if !st.Mode().IsRegular() {
			return "", errors.Join(model.ErrConflict, model.ErrInvalid, fmt.Errorf("destination is not a regular file: %s", resolvedDest))
		}
		if os.SameFile(st, managedStat) || os.SameFile(st, rootStat) {
			return "", fmt.Errorf("%w: destination aliases store directory", model.ErrInvalid)
		}
		for _, p := range protectedPaths {
			if pst, perr := os.Stat(p); perr == nil && os.SameFile(st, pst) {
				return "", fmt.Errorf("%w: destination aliases live store file", model.ErrInvalid)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return resolvedDest, nil
}

// PublishExportFile atomically publishes a completed staging file to destinationPath
// after re-verifying all store boundary and overwrite protections.
func (s *Store) PublishExportFile(stagedPath, destinationPath string, allowOverwrite bool) error {
	resolvedDest, err := s.ValidateExportTarget(destinationPath, allowOverwrite)
	if err != nil {
		return err
	}
	st, err := os.Lstat(stagedPath)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("%w: staged file is not a regular file", model.ErrInvalid)
	}
	if !allowOverwrite {
		if err := os.Link(stagedPath, resolvedDest); err != nil {
			if errors.Is(err, os.ErrExist) {
				return errors.Join(model.ErrConflict, err)
			}
			return err
		}
		return nil
	}
	return os.Rename(stagedPath, resolvedDest)
}
