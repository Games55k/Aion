package display

import (
	"time"

	"github.com/egoist/mygo/ui"
)

type IconSize int

const (
	IconInherit IconSize = iota
	IconSmall
	IconMiddle
	IconLarge
)

type IconProps struct {
	// Size picks 14, 20 or 24 DIPs; IconInherit follows the font size around the icon.
	Size IconSize
	// Spin turns the icon once a second, as a loading indicator.
	Spin bool
}

func iconPixels(s IconSize) float32 {
	switch s {
	case IconSmall:
		return 14
	case IconMiddle:
		return 20
	case IconLarge:
		return 24
	}
	return 0
}

func Icon(c *ui.Context, svg *ui.SVG, p IconProps) ui.Element {
	icon := ui.Icon(c, svg)
	if px := iconPixels(p.Size); px > 0 {
		icon.FontSize(px)
	}
	if p.Spin {
		icon.Rotate(icon.Loop("spin", time.Second, ui.Linear) * 360)
	}
	return icon
}
