//go:build !windows

package attachments

import (
	"errors"
	"os"
	"path/filepath"

	"aion/backend/model"

	"golang.org/x/sys/unix"
)

func openReadOnlyNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}

func (s *Service) openManagedDir() (*os.File, error) {
	dir, err := os.OpenFile(s.store.AttachmentsDir(), os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOTDIR) || errors.Is(err, unix.ELOOP) {
		return nil, errors.Join(model.ErrInvalid, err)
	}
	return dir, err
}

func linkStagedInManagedDir(stagedPath string, dir *os.File, storageName string) error {
	fd := int(dir.Fd())
	if err := unix.Linkat(unix.AT_FDCWD, stagedPath, fd, storageName, 0); err != nil {
		return &os.LinkError{Op: "link", Old: stagedPath, New: filepath.Join(dir.Name(), storageName), Err: err}
	}
	return nil
}

func unlinkInManagedDir(dir *os.File, storageName string) error {
	return unix.Unlinkat(int(dir.Fd()), storageName, 0)
}

func (s *Service) openManaged(name string) (*os.File, error) {
	if !safeName(name) {
		return nil, model.ErrInvalid
	}
	dir, err := s.openManagedDir()
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	// os.Root.OpenFile resolves in-root symlinks. Open the single validated
	// filename against the pinned directory so the kernel rejects symlinks.
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	path := filepath.Join(dir.Name(), name)
	if err != nil {
		return nil, &os.PathError{Op: "openat", Path: path, Err: err}
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.Join(model.ErrInvalid, err)
	}
	return file, nil
}
