package ipc

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	socketFileName = "aion.sock"
	// macOS sockaddr_un.sun_path limit is 104 bytes including NUL terminator.
	maxDirectSocketPathLen = 96
)

// SocketPath returns the deterministic local Unix domain socket path for dataDir.
// Normally this is <dataDir>/aion.sock. If that path exceeds the OS sun_path
// limit (common in deep temporary test directories on macOS), it falls back to
// a private per-user directory under /tmp keyed by a hash of the canonical dataDir.
func SocketPath(dataDir string) (string, error) {
	if dataDir == "" {
		return "", errors.New("data directory is required")
	}
	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	if st, err := os.Lstat(absDir); err == nil {
		if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
			return "", fmt.Errorf("unsafe data directory: %s", absDir)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	canonical := absDir
	if resolved, err := filepath.EvalSymlinks(absDir); err == nil {
		canonical = resolved
	}
	direct := filepath.Join(canonical, socketFileName)
	if len(direct) <= maxDirectSocketPathLen {
		return direct, nil
	}

	// Fallback directory owned exclusively by the current user for deep paths.
	fallbackDir := fallbackSocketDir()
	if err := ensurePrivateDir(fallbackDir); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(canonical))
	return filepath.Join(fallbackDir, "a-"+hex.EncodeToString(sum[:12])+".sock"), nil
}

func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	st, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
		return fmt.Errorf("unsafe socket directory: %s", dir)
	}
	return checkPrivateDirOwner(dir, st)
}

func validateSocketPath(path string) error {
	parent := filepath.Dir(path)
	if st, err := os.Lstat(parent); err != nil {
		return err
	} else if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
		return fmt.Errorf("unsafe socket parent directory: %s", parent)
	}
	if st, err := os.Lstat(path); err == nil {
		if st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe symlink at socket path: %s", path)
		}
		if st.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("non-socket file exists at socket path: %s", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
