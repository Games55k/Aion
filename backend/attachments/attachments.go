// Package attachments manages local attachment copies independently of their sources.
package attachments

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"aion/backend/model"
	"aion/backend/store"
	"golang.org/x/sys/unix"
)

type Service struct{ store *store.Store }

// MaxFileBytes matches the V1 per-file backup limit so every successful import
// can be included in a full backup.
const MaxFileBytes int64 = 256 << 20

func New(s *store.Store) *Service { return &Service{store: s} }

// Import copies a regular source file into managed storage. A failed import
// rolls back its row and removes its new file before releasing the shared lock.
func (s *Service) Import(ctx context.Context, taskID, sourcePath string) (model.Attachment, error) {
	var attachment model.Attachment
	name := filepath.Base(sourcePath)
	if strings.TrimSpace(taskID) == "" || !safeName(name) {
		return attachment, model.ErrInvalid
	}
	err := s.store.Exclusive(ctx, func(db *sql.DB) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err = visibleTask(ctx, tx, taskID); err != nil {
			return err
		}
		source, err := openRegular(sourcePath)
		if err != nil {
			return fmt.Errorf("open attachment source: %w", err)
		}
		defer source.Close()
		info, err := source.Stat()
		if err != nil {
			return err
		}
		if info.Size() > MaxFileBytes {
			return fmt.Errorf("%w: attachment exceeds 256 MiB", model.ErrInvalid)
		}
		root, err := managedRoot(s.store.AttachmentsDir())
		if err != nil {
			return err
		}
		defer root.Close()
		storageName := model.NewID()
		target, err := root.OpenFile(storageName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		committed := false
		defer func() {
			if !committed {
				_ = root.Remove(storageName)
			}
		}()
		header := make([]byte, 512)
		n, readErr := io.ReadFull(contextReader{ctx, source}, header)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return errors.Join(readErr, target.Close())
		}
		header = header[:n]
		hash := sha256.New()
		size, copyErr := io.Copy(io.MultiWriter(target, hash), io.LimitReader(contextReader{ctx, io.MultiReader(bytes.NewReader(header), source)}, MaxFileBytes+1))
		if copyErr == nil && size > MaxFileBytes {
			copyErr = fmt.Errorf("%w: attachment exceeds 256 MiB", model.ErrInvalid)
		}
		if copyErr == nil {
			copyErr = target.Sync()
		}
		if err = errors.Join(copyErr, target.Close()); err != nil {
			return err
		}
		attachment = model.Attachment{
			ID: model.NewID(), TaskID: taskID, Name: name, StorageName: storageName,
			MediaType: http.DetectContentType(header), Size: size,
			SHA256: hex.EncodeToString(hash.Sum(nil)), CreatedAt: time.Now().UTC(),
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO attachments(id,task_id,name,storage_name,media_type,size,sha256,created_at) VALUES(?,?,?,?,?,?,?,?)`, attachment.ID, taskID, name, storageName, attachment.MediaType, size, attachment.SHA256, attachment.CreatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		committed = true
		return nil
	})
	if err != nil {
		return model.Attachment{}, err
	}
	return attachment, nil
}

// List returns metadata for a visible task, in import order.
func (s *Service) List(ctx context.Context, taskID string) ([]model.Attachment, error) {
	attachments := make([]model.Attachment, 0)
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		if err := visibleTask(ctx, tx, taskID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id,task_id,name,storage_name,media_type,size,sha256,created_at FROM attachments WHERE task_id=?`, taskID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			a, err := scanAttachment(rows)
			if err != nil {
				return err
			}
			attachments = append(attachments, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	// RFC3339Nano omits trailing fractional zeros, so sort parsed instants in Go.
	sort.Slice(attachments, func(i, j int) bool {
		if attachments[i].CreatedAt.Equal(attachments[j].CreatedAt) {
			return attachments[i].ID < attachments[j].ID
		}
		return attachments[i].CreatedAt.Before(attachments[j].CreatedAt)
	})
	return attachments, nil
}

// ManagedPath resolves an existing regular file for a visible attachment.
// The returned path is transient: a later removal, purge or restore can replace it.
func (s *Service) ManagedPath(ctx context.Context, attachmentID string) (string, error) {
	var path string
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		a, err := visibleAttachment(ctx, tx, attachmentID)
		if err != nil {
			return err
		}
		file, err := s.openManaged(a.StorageName)
		if err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
		path = filepath.Join(s.store.AttachmentsDir(), a.StorageName)
		return nil
	})
	return path, err
}

// Export creates a new file at destinationPath. Existing files and symlinks are
// never overwritten. Partial exports are removed on failure.
func (s *Service) Export(ctx context.Context, attachmentID, destinationPath string) error {
	return s.export(ctx, attachmentID, destinationPath, false)
}

// ExportOverwrite exports an attachment to destinationPath using a temporary staging file
// in the destination directory and atomically replaces an existing user regular file only
// after content verification succeeds. Protected store paths, symlinks, and directories are rejected.
func (s *Service) ExportOverwrite(ctx context.Context, attachmentID, destinationPath string) error {
	return s.export(ctx, attachmentID, destinationPath, true)
}

func (s *Service) export(ctx context.Context, attachmentID, destinationPath string, allowOverwrite bool) error {
	return s.store.Read(ctx, func(tx *sql.Tx) error {
		a, err := visibleAttachment(ctx, tx, attachmentID)
		if err != nil {
			return err
		}
		source, err := s.openManaged(a.StorageName)
		if err != nil {
			return err
		}
		defer source.Close()
		resolvedDest, err := s.store.ValidateExportTarget(destinationPath, allowOverwrite)
		if err != nil {
			return err
		}
		target, err := os.CreateTemp(filepath.Dir(resolvedDest), ".aion-attachment-export-*.tmp")
		if err != nil {
			return err
		}
		stagePath := target.Name()
		defer func() {
			_ = os.Remove(stagePath)
		}()
		hash := sha256.New()
		limit := min(max(a.Size, 0), MaxFileBytes) + 1
		size, copyErr := io.Copy(io.MultiWriter(target, hash), io.LimitReader(contextReader{ctx, source}, limit))
		if copyErr == nil && (size != a.Size || hex.EncodeToString(hash.Sum(nil)) != a.SHA256) {
			copyErr = fmt.Errorf("%w: attachment content mismatch", model.ErrInvalid)
		}
		if copyErr == nil {
			copyErr = target.Sync()
		}
		if err = errors.Join(copyErr, target.Close()); err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		return s.store.PublishExportFile(stagePath, resolvedDest, allowOverwrite)
	})
}

// Remove drops a visible attachment's metadata. GC reclaims its physical file;
// deferring file deletion preserves safety if the transaction fails to commit.
func (s *Service) Remove(ctx context.Context, attachmentID string) error {
	return s.store.Write(ctx, func(tx *sql.Tx) error {
		if _, err := visibleAttachment(ctx, tx, attachmentID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM attachments WHERE id=?`, attachmentID)
		return err
	})
}

// GC removes only unreferenced regular files directly under managed storage.
// References from soft-deleted tasks/projects are retained. Directories,
// symlinks and special files are never followed or removed.
func (s *Service) GC(ctx context.Context) (int, error) {
	removed := 0
	err := s.store.Read(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT storage_name FROM attachments`)
		if err != nil {
			return err
		}
		referenced := make(map[string]bool)
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				rows.Close()
				return err
			}
			referenced[name] = true
		}
		err = errors.Join(rows.Err(), rows.Close())
		if err != nil {
			return err
		}
		root, err := managedRoot(s.store.AttachmentsDir())
		if err != nil {
			return err
		}
		defer root.Close()
		dir, err := root.Open(".")
		if err != nil {
			return err
		}
		entries, err := dir.ReadDir(-1)
		err = errors.Join(err, dir.Close())
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err = ctx.Err(); err != nil {
				return err
			}
			if referenced[entry.Name()] {
				continue
			}
			info, err := root.Lstat(entry.Name())
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			if err = root.Remove(entry.Name()); err != nil {
				return err
			}
			removed++
		}
		return nil
	})
	return removed, err
}

func visibleTask(ctx context.Context, tx *sql.Tx, taskID string) error {
	var taskDeleted, projectDeleted sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT t.deleted_at,p.deleted_at FROM tasks t JOIN projects p ON p.id=t.project_id WHERE t.id=?`, taskID).Scan(&taskDeleted, &projectDeleted)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	if err != nil {
		return err
	}
	if taskDeleted.Valid {
		return model.ErrNotFound
	}
	if projectDeleted.Valid {
		return model.ErrParentDeleted
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanAttachment(row scanner) (model.Attachment, error) {
	var a model.Attachment
	var timestamp string
	err := row.Scan(&a.ID, &a.TaskID, &a.Name, &a.StorageName, &a.MediaType, &a.Size, &a.SHA256, &timestamp)
	if err != nil {
		return a, err
	}
	a.CreatedAt, err = time.Parse(time.RFC3339Nano, timestamp)
	return a, err
}

func visibleAttachment(ctx context.Context, tx *sql.Tx, id string) (model.Attachment, error) {
	a, err := scanAttachment(tx.QueryRowContext(ctx, `SELECT id,task_id,name,storage_name,media_type,size,sha256,created_at FROM attachments WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return a, model.ErrNotFound
	}
	if err != nil {
		return a, err
	}
	return a, visibleTask(ctx, tx, a.TaskID)
}

func safeName(name string) bool {
	return strings.TrimSpace(name) != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, "/\\:") && strings.IndexFunc(name, unicode.IsControl) < 0
}

func managedRoot(path string) (*os.Root, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, model.ErrInvalid
	}
	return os.OpenRoot(path)
}

func openRegular(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, model.ErrInvalid
	}
	file, err := os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, errors.Join(model.ErrInvalid, err)
	}
	return file, nil
}

func (s *Service) openManaged(name string) (*os.File, error) {
	if !safeName(name) {
		return nil, model.ErrInvalid
	}
	root, err := managedRoot(s.store.AttachmentsDir())
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.OpenFile(name, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.Join(model.ErrInvalid, err)
	}
	return file, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
