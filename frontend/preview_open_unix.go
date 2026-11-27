//go:build !windows

package frontend

import (
	"os"

	"golang.org/x/sys/unix"
)

func openPreviewFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}
