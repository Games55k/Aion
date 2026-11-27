//go:build !windows

package ipc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func fallbackSocketDir() string {
	return filepath.Join("/tmp", fmt.Sprintf("aion-ipc-%d", os.Getuid()))
}

func checkPrivateDirOwner(dir string, st os.FileInfo) error {
	if stat, ok := st.Sys().(*syscall.Stat_t); ok && int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("socket directory owned by unexpected user: %s", dir)
	}
	if st.Mode().Perm() != 0o700 {
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func isSocketUnavailable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, syscall.ENOENT) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ENOTSOCK)
}

func isLockBusyError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, unix.EWOULDBLOCK) ||
		errors.Is(err, unix.EAGAIN) ||
		strings.Contains(err.Error(), "data directory is already open")
}
