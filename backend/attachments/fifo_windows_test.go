//go:build windows

package attachments_test

import "testing"

func createTestFIFO(_ *testing.T, _ string) (string, bool) {
	return "", false
}
