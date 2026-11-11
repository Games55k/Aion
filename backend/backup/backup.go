// Package backup implements consistent full backups and validated replacement.
package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"

	"aion/backend/model"
	"aion/backend/store"
	"golang.org/x/sys/unix"
)

const (
	FormatVersion          = 1
	MaxEntries             = 10002
	MaxFileBytes     int64 = 256 << 20
	MaxTotalBytes    int64 = 2 << 30
	MaxArchiveBytes  int64 = 3 << 30
	maxManifestBytes int64 = 2 << 20
)

type Service struct {
	store           *store.Store
	replaceDatabase func(context.Context, string) error
	now             func() time.Time
}

// Options customizes a Service. The zero value uses the wall clock.
type Options struct {
	// Now supplies the clock for schedules, manifests, and rotation order.
	Now func() time.Time
}

func New(s *store.Store, options ...Options) *Service {
	service := &Service{store: s, replaceDatabase: s.ReplaceDatabaseLocked, now: time.Now}
	for _, o := range options {
		if o.Now != nil {
			service.now = o.Now
		}
	}
	return service
}

type RestoreResult struct {
	SafetyBackupPath string `json:"safety_backup_path"`
}
type fileRecord struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type manifest struct {
	Version       int          `json:"version"`
	SchemaVersion int          `json:"schema_version"`
	CreatedAt     time.Time    `json:"created_at"`
	Kind          string       `json:"kind"`
	Files         []fileRecord `json:"files"`
}
type attachment struct {
	name string
	size int64
	hash string
}
type schemaObject struct{ Kind, Name, Table, SQL string }

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: backup: %s", model.ErrInvalid, fmt.Sprintf(format, args...))
}

func wrapZipErr(err error) error {
	if errors.Is(err, zip.ErrFormat) || errors.Is(err, zip.ErrChecksum) || errors.Is(err, zip.ErrAlgorithm) || errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.Join(invalid("corrupt ZIP entry"), err)
	}
	return err
}

func safeComponent(s string) bool {
	return strings.TrimSpace(s) != "" && s != "." && s != ".." && len(s) <= 255 &&
		!strings.ContainsAny(s, "/\\:") && strings.IndexFunc(s, unicode.IsControl) < 0
}
func validEntry(s string) bool {
	return s == "manifest.json" || s == "database.sqlite" ||
		(strings.HasPrefix(s, "attachments/") && safeComponent(strings.TrimPrefix(s, "attachments/")))
}
func validHash(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func regular(path string) (os.FileInfo, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, invalid("not a regular file: %s", path)
	}
	return st, nil
}

// openRegular rejects symlinks and checks the opened inode against Lstat.
func openRegular(path string) (*os.File, error) {
	st, err := regular(path)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(st, opened) {
		_ = f.Close()
		return nil, errors.Join(invalid("file changed while opening"), err)
	}
	return f, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func copyHash(ctx context.Context, dst io.Writer, src io.Reader, limit int64) (int64, string, error) {
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(dst, h), io.LimitReader(contextReader{ctx, src}, limit+1))
	if err == nil && n > limit {
		err = invalid("file exceeds size limit")
	}
	return n, hex.EncodeToString(h.Sum(nil)), err
}

func readAttachments(ctx context.Context, db *sql.DB) ([]attachment, error) {
	rows, err := db.QueryContext(ctx, "SELECT storage_name,size,sha256 FROM attachments ORDER BY storage_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []attachment
	for rows.Next() {
		var a attachment
		if err := rows.Scan(&a.name, &a.size, &a.hash); err != nil {
			return nil, err
		}
		if !safeComponent(a.name) || a.size < 0 || a.size > MaxFileBytes || !validHash(a.hash) {
			return nil, invalid("invalid attachment metadata")
		}
		out = append(out, a)
		if len(out)+2 > MaxEntries {
			return nil, invalid("too many attachments")
		}
	}
	return out, rows.Err()
}

func schema(ctx context.Context, db *sql.DB) ([]schemaObject, error) {
	rows, err := db.QueryContext(ctx, "SELECT type,name,tbl_name,coalesce(sql,'') FROM sqlite_schema WHERE name NOT GLOB 'sqlite_*' ORDER BY type,name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schemaObject
	for rows.Next() {
		var o schemaObject
		if err := rows.Scan(&o.Kind, &o.Name, &o.Table, &o.SQL); err != nil {
			return nil, err
		}
		out = append(out, o)
		if len(out) > 100 {
			return nil, invalid("too many schema objects")
		}
	}
	return out, rows.Err()
}

func (s *Service) Export(ctx context.Context, destination string) error {
	return s.store.Exclusive(ctx, func(db *sql.DB) error {
		_, err := s.exportLocked(ctx, db, destination, "manual", false)
		return err
	})
}

// ExportOverwrite exports a validated full backup archive to destination using a temporary staging
// file in the destination directory and atomically replaces an existing user regular file only after
// archive creation and verification succeed. Protected store paths, symlinks, and directories are rejected.
func (s *Service) ExportOverwrite(ctx context.Context, destination string) error {
	return s.store.Exclusive(ctx, func(db *sql.DB) error {
		_, err := s.exportLocked(ctx, db, destination, "manual", true)
		return err
	})
}

// exportLocked never reenters the store mutex. The snapshot and every referenced
// attachment (including trash) are read while all business writes are excluded.
func (s *Service) exportLocked(ctx context.Context, db *sql.DB, destination, kind string, allowOverwrite bool) (string, error) {
	resolvedDest, err := s.store.ValidateExportTarget(destination, allowOverwrite)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(resolvedDest)
	managed, err := filepath.EvalSymlinks(s.store.AttachmentsDir())
	if err != nil {
		return "", err
	}
	tmpdir, err := os.MkdirTemp(s.store.Root(), ".backup-export-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpdir)
	snapshot := filepath.Join(tmpdir, "database.sqlite")
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", snapshot); err != nil {
		return "", err
	}
	if err = os.Chmod(snapshot, 0600); err != nil {
		return "", err
	}
	refs, err := readAttachments(ctx, db)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp(parent, ".aion-backup-*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := f.Name()
	fileClosed := false
	defer func() {
		if !fileClosed {
			_ = f.Close()
		}
		_ = os.Remove(tempPath)
	}()
	archiveSum := sha256.New()
	zw := zip.NewWriter(io.MultiWriter(f, archiveSum))
	zipClosed := false
	defer func() {
		if !zipClosed {
			_ = zw.Close()
		}
	}()
	m := manifest{Version: FormatVersion, SchemaVersion: store.SchemaVersion, CreatedAt: s.now().UTC(), Kind: kind}
	var total int64
	add := func(name, path string, expected *attachment) error {
		src, err := openRegular(path)
		if err != nil {
			return err
		}
		defer src.Close()
		st, err := src.Stat()
		if err != nil {
			return err
		}
		if st.Size() > MaxFileBytes || st.Size() > MaxTotalBytes-maxManifestBytes-total {
			return invalid("backup exceeds size limit")
		}
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		n, h, err := copyHash(ctx, w, src, MaxFileBytes)
		if err != nil {
			return err
		}
		if n != st.Size() || (expected != nil && (n != expected.size || h != expected.hash)) {
			return invalid("attachment size/hash mismatch: %s", name)
		}
		total += n
		m.Files = append(m.Files, fileRecord{Path: name, Size: n, SHA256: h})
		return nil
	}
	if err = add("database.sqlite", snapshot, nil); err != nil {
		return "", err
	}
	for _, a := range refs {
		if err = add("attachments/"+a.name, filepath.Join(managed, a.name), &a); err != nil {
			return "", err
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	if int64(len(b)) > maxManifestBytes {
		return "", invalid("manifest too large")
	}
	w, err := zw.Create("manifest.json")
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	zipClosed = true
	if err = zw.Close(); err != nil {
		return "", err
	}
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if st.Size() > MaxArchiveBytes {
		return "", invalid("archive too large")
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	fileClosed = true
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = s.store.PublishExportFile(tempPath, resolvedDest, allowOverwrite); err != nil {
		return "", err
	}
	return hex.EncodeToString(archiveSum.Sum(nil)), nil
}

// checkDirectory bounds the central directory before archive/zip allocates its
// file list. ZIP64 and multi-disk archives are deliberately unsupported in V1.
func checkDirectory(f *os.File, size int64) error {
	if size < 22 || size > MaxArchiveBytes {
		return invalid("invalid archive size")
	}
	n := int64(65557)
	if size < n {
		n = size
	}
	b := make([]byte, n)
	if _, err := f.ReadAt(b, size-n); err != nil {
		return err
	}
	for i := len(b) - 22; i >= 0; i-- {
		if binary.LittleEndian.Uint32(b[i:]) != 0x06054b50 {
			continue
		}
		e := b[i:]
		if i+22+int(binary.LittleEndian.Uint16(e[20:])) != len(b) {
			continue
		}
		count := binary.LittleEndian.Uint16(e[10:])
		if binary.LittleEndian.Uint16(e[4:]) != 0 || binary.LittleEndian.Uint16(e[6:]) != 0 || binary.LittleEndian.Uint16(e[8:]) != count || count == 0 || int(count) > MaxEntries || binary.LittleEndian.Uint32(e[12:]) > 16<<20 || binary.LittleEndian.Uint32(e[16:]) == 0xffffffff {
			return invalid("unsupported central directory")
		}
		// Do not trust EOCD's declared count: archive/zip reads actual directory
		// records and may allocate more entries. Bound and count them BEFORE it runs.
		directorySize := int64(binary.LittleEndian.Uint32(e[12:]))
		directoryOffset := int64(binary.LittleEndian.Uint32(e[16:]))
		if directoryOffset+directorySize != size-n+int64(i) {
			return invalid("central directory bounds mismatch")
		}
		directory := make([]byte, directorySize)
		if _, err := f.ReadAt(directory, directoryOffset); err != nil {
			return err
		}
		actual := 0
		for cursor := 0; cursor < len(directory); {
			if len(directory)-cursor < 46 || binary.LittleEndian.Uint32(directory[cursor:]) != 0x02014b50 {
				return invalid("invalid central directory record")
			}
			h := directory[cursor:]
			length := 46 + int(binary.LittleEndian.Uint16(h[28:])) + int(binary.LittleEndian.Uint16(h[30:])) + int(binary.LittleEndian.Uint16(h[32:]))
			if length > len(h) || binary.LittleEndian.Uint16(h[34:]) != 0 ||
				binary.LittleEndian.Uint32(h[20:]) == 0xffffffff || binary.LittleEndian.Uint32(h[24:]) == 0xffffffff ||
				int64(binary.LittleEndian.Uint32(h[42:])) >= directoryOffset {
				return invalid("unsupported central directory record")
			}
			actual++
			if actual > MaxEntries || actual > int(count) {
				return invalid("actual ZIP entry count exceeds limit")
			}
			cursor += length
		}
		if actual != int(count) {
			return invalid("central directory count mismatch")
		}
		return nil
	}
	return invalid("missing ZIP directory")
}

func decodeJSON(b []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return invalid("trailing JSON")
	}
	return nil
}

// stage validates EVERYTHING before taking the replacement lock or writing any
// live content. Its staging directory shares the live filesystem for rename.
func (s *Service) stage(ctx context.Context, source string, expected []schemaObject) (string, error) {
	f, err := openRegular(source)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if err = checkDirectory(f, st.Size()); err != nil {
		return "", err
	}
	z, err := zip.NewReader(f, st.Size())
	if err != nil {
		return "", invalid("invalid ZIP: %v", err)
	}
	if len(z.File) > MaxEntries {
		return "", invalid("too many entries")
	}
	entries := make(map[string]*zip.File, len(z.File))
	var total uint64
	for _, e := range z.File {
		if !validEntry(e.Name) || !e.Mode().IsRegular() || e.Flags&1 != 0 || (e.Method != zip.Store && e.Method != zip.Deflate) {
			return "", invalid("unsafe ZIP entry: %q", e.Name)
		}
		if _, ok := entries[e.Name]; ok {
			return "", invalid("duplicate entry: %s", e.Name)
		}
		limit := uint64(MaxFileBytes)
		if e.Name == "manifest.json" {
			limit = uint64(maxManifestBytes)
		}
		if e.UncompressedSize64 > limit || e.UncompressedSize64 > uint64(MaxTotalBytes)-total {
			return "", invalid("archive exceeds resource limit")
		}
		total += e.UncompressedSize64
		entries[e.Name] = e
	}
	me, ok := entries["manifest.json"]
	if !ok {
		return "", invalid("missing manifest")
	}
	r, err := me.Open()
	if err != nil {
		return "", wrapZipErr(err)
	}
	b, err := io.ReadAll(io.LimitReader(contextReader{ctx, r}, maxManifestBytes+1))
	cerr := r.Close()
	if err = errors.Join(err, cerr); err != nil {
		return "", wrapZipErr(err)
	}
	if int64(len(b)) > maxManifestBytes {
		return "", invalid("manifest too large")
	}
	var m manifest
	if err = decodeJSON(b, &m); err != nil {
		return "", invalid("invalid manifest: %v", err)
	}
	if m.Version != FormatVersion || m.SchemaVersion != store.SchemaVersion || m.CreatedAt.IsZero() || (m.Kind != "manual" && m.Kind != "auto" && m.Kind != "safety") {
		return "", invalid("unsupported manifest")
	}
	if len(m.Files)+1 != len(entries) {
		return "", invalid("manifest entry count mismatch")
	}
	dir, err := os.MkdirTemp(s.store.Root(), ".backup-restore-*")
	if err != nil {
		return "", err
	}
	keep := false
	defer func() {
		if !keep {
			os.RemoveAll(dir)
		}
	}()
	if err = os.Mkdir(filepath.Join(dir, "attachments"), 0700); err != nil {
		return "", err
	}
	records := make(map[string]fileRecord, len(m.Files))
	for _, rec := range m.Files {
		e, ok := entries[rec.Path]
		if !ok || rec.Path == "manifest.json" || !validEntry(rec.Path) || rec.Size < 0 || rec.Size > MaxFileBytes || uint64(rec.Size) != e.UncompressedSize64 || !validHash(rec.SHA256) {
			return "", invalid("invalid file record")
		}
		if _, ok := records[rec.Path]; ok {
			return "", invalid("duplicate manifest record")
		}
		records[rec.Path] = rec
		r, err := e.Open()
		if err != nil {
			return "", wrapZipErr(err)
		}
		out, err := os.OpenFile(filepath.Join(dir, filepath.FromSlash(rec.Path)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			r.Close()
			return "", err
		}
		n, h, copyErr := copyHash(ctx, out, r, rec.Size)
		if zipErr := wrapZipErr(errors.Join(copyErr, r.Close())); zipErr != nil {
			_ = out.Close()
			return "", zipErr
		}
		if err = out.Close(); err != nil {
			return "", err
		}
		if n != rec.Size || h != rec.SHA256 {
			return "", invalid("file size/hash mismatch: %s", rec.Path)
		}
	}
	if _, ok := records["database.sqlite"]; !ok {
		return "", invalid("missing database")
	}
	refs, err := validateDB(ctx, filepath.Join(dir, "database.sqlite"), expected)
	if err != nil {
		return "", err
	}
	if len(refs)+1 != len(records) {
		return "", invalid("unreferenced or missing attachment")
	}
	for _, a := range refs {
		rec, ok := records["attachments/"+a.name]
		if !ok || rec.Size != a.size || rec.SHA256 != a.hash {
			return "", invalid("missing or invalid attachment reference: %s", a.name)
		}
	}
	keep = true
	return dir, nil
}

func validateDB(ctx context.Context, path string, expected []schemaObject) ([]attachment, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {"ro"}, "immutable": {"1"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, "PRAGMA trusted_schema=OFF; PRAGMA query_only=ON"); err != nil {
		return nil, invalid("invalid SQLite: %v", err)
	}
	var version int
	if err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return nil, invalid("invalid SQLite: %v", err)
	}
	if version != store.SchemaVersion {
		return nil, invalid("unsupported database schema %d", version)
	}
	actual, err := schema(ctx, db)
	if err != nil {
		return nil, invalid("invalid database schema: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		return nil, invalid("database schema mismatch")
	}
	var integrity string
	if err = db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		return nil, invalid("database integrity failed: %v %s", err, integrity)
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, err
	}
	bad := rows.Next()
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return nil, err
	}
	if bad {
		return nil, invalid("broken foreign key reference")
	}
	if err = validateValues(ctx, db); err != nil {
		return nil, err
	}
	return readAttachments(ctx, db)
}

func (s *Service) Restore(ctx context.Context, source string) (RestoreResult, error) {
	var result RestoreResult
	var expected []schemaObject
	err := s.store.Exclusive(ctx, func(db *sql.DB) error { var e error; expected, e = schema(ctx, db); return e })
	if err != nil {
		return result, err
	}
	dir, err := s.stage(ctx, source, expected)
	if err != nil {
		return result, err
	}
	restoring, _ := os.Stat(source)
	defer func() {
		if dir != "" {
			os.RemoveAll(dir)
		}
	}()
	err = s.store.Exclusive(ctx, func(db *sql.DB) error {
		backupDir := filepath.Join(s.store.Root(), "backups")
		if st, err := os.Lstat(backupDir); err == nil {
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return invalid("unsafe backups directory")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(backupDir, 0700); err != nil {
			return err
		}
		safety := filepath.Join(backupDir, "safety-"+model.NewID()+".zip")
		hash, err := s.exportLocked(ctx, db, safety, "safety", false)
		if err != nil {
			return fmt.Errorf("safety backup: %w", err)
		}
		result.SafetyBackupPath = safety
		s.recordSafetyBackup(ctx, safety, hash, restoring)
		if err := s.keepLocalBackupDirectory(ctx, db, filepath.Join(dir, "database.sqlite")); err != nil {
			return fmt.Errorf("prepare restored backup policy: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		old := filepath.Join(dir, "previous-attachments")
		live := s.store.AttachmentsDir()
		if err := os.Rename(live, old); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(dir, "attachments"), live); err != nil {
			rollbackErr := os.Rename(old, live)
			if rollbackErr != nil {
				dir = ""
				return errors.Join(err, fmt.Errorf("attachment rollback failed; retained staging at %s: %w", filepath.Dir(old), rollbackErr))
			}
			return err
		}
		if err := s.replaceDatabase(ctx, filepath.Join(dir, "database.sqlite")); err != nil {
			// Store restores its old database on failure. Preserve the old file
			// directory until rollback has completed; never delete it on failure.
			moveErr := os.Rename(live, filepath.Join(dir, "failed-attachments"))
			rollbackErr := os.Rename(old, live)
			if rollbackErr != nil {
				dir = ""
				return errors.Join(err, moveErr, fmt.Errorf("attachment rollback failed; retained staging at %s: %w", filepath.Dir(old), rollbackErr))
			}
			return errors.Join(err, moveErr)
		}
		return nil
	})
	return result, err
}

// keepLocalBackupDirectory rewrites the staged database's automatic backup
// policy to use this machine's backup directory. The archive's schedule and
// retention are restored, but its directory is a path on the machine that
// exported it and must not be created or written to here. Runs before any live
// file is replaced, so a failure aborts the restore cleanly.
func (s *Service) keepLocalBackupDirectory(ctx context.Context, live *sql.DB, staged string) error {
	local := ""
	if p, err := s.policyLocked(ctx, live); err == nil && p.Directory != s.defaultPolicy().Directory {
		local = p.Directory
	}
	u := url.URL{Scheme: "file", Path: staged}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	restored := s.defaultPolicy()
	var raw string
	err = db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", policyKey).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if local == "" {
			return nil
		}
	case err != nil:
		return err
	case decodeJSON([]byte(raw), &restored) != nil:
		// Leave an unreadable policy for RunDue to report, as it would live.
		return nil
	}
	restored.Directory = local
	b, err := json.Marshal(restored)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", policyKey, string(b))
	return errors.Join(err, db.Close())
}
