//go:build !windows

package attachments_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func createTestFIFO(t *testing.T, dir string) (string, bool) {
	t.Helper()
	fifo := filepath.Join(dir, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	return fifo, true
}
