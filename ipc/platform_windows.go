//go:build windows

package ipc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func fallbackSocketDir() string {
	return filepath.Join(os.TempDir(), "aion-ipc")
}

func checkPrivateDirOwner(_ string, _ os.FileInfo) error {
	return nil
}

func isSocketUnavailable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, syscall.ENOENT) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, windows.WSAECONNREFUSED) ||
		errors.Is(err, windows.WSAENOTSOCK) ||
		errors.Is(err, windows.ERROR_FILE_NOT_FOUND)
}

func isLockBusyError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
		strings.Contains(err.Error(), "data directory is already open")
}
