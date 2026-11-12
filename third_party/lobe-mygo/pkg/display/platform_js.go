//go:build js

package display

import (
	"strings"
	"syscall/js"
)

func appleDevice() bool {
	nav := js.Global().Get("navigator")
	if nav.IsUndefined() {
		return false
	}
	p := nav.Get("platform")
	if p.Type() != js.TypeString || p.String() == "" {
		p = nav.Get("userAgent")
	}
	s := strings.ToLower(p.String())
	return strings.Contains(s, "mac") || strings.Contains(s, "iphone") || strings.Contains(s, "ipad")
}

func browserOnMac() bool { return isApple() }
