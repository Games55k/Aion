//go:build windows

package attachments

import (
	"errors"
	"os"
	"path/filepath"

	"aion/backend/model"
)

func openReadOnlyNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY, 0)
}

func (s *Service) openManagedDir() (*os.File, error) {
	dirPath := s.store.AttachmentsDir()
	st, err := os.Lstat(dirPath)
	if err != nil {
		return nil, err
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
		return nil, model.ErrInvalid
	}
	return os.Open(dirPath)
}

func linkStagedInManagedDir(stagedPath string, dir *os.File, storageName string) error {
	dest := filepath.Join(dir.Name(), storageName)
	return os.Link(stagedPath, dest)
}

func unlinkInManagedDir(dir *os.File, storageName string) error {
	return os.Remove(filepath.Join(dir.Name(), storageName))
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
	path := filepath.Join(dir.Name(), name)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, model.ErrInvalid
	}
	file, err := os.Open(path)
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
