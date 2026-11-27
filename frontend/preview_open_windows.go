//go:build windows

package frontend

import (
	"errors"
	"os"
)

func openPreviewFile(path string) (*os.File, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	return os.OpenFile(path, os.O_RDONLY, 0)
}
