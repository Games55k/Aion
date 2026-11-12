//go:build !js

package display

import "runtime"

func appleDevice() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "ios" }

func browserOnMac() bool { return false }
