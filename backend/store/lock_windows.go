//go:build windows

package store

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openLockFile(lockPath string) (*os.File, error) {
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var ol windows.Overlapped
	err = windows.LockFileEx(
		windows.Handle(lock.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&ol,
	)
	if err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("data directory is already open: %w", err)
	}
	return lock, nil
}

func unlockLockFile(lock *os.File) error {
	var ol windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(lock.Fd()), 0, 1, 0, &ol)
}
