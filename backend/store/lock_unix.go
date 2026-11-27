//go:build !windows

package store

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openLockFile(lockPath string) (*os.File, error) {
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("data directory is already open: %w", err)
	}
	return lock, nil
}

func unlockLockFile(lock *os.File) error {
	return unix.Flock(int(lock.Fd()), unix.LOCK_UN)
}
