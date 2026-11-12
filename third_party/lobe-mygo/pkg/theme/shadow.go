package theme

import "github.com/egoist/mygo/ui"

// Elevation is one of Lobe UI's shadows: BoxShadow for dialogs, BoxShadowSecondary for popovers
// and menus, BoxShadowTertiary for tooltips.
type Elevation int

const (
	BoxShadow Elevation = iota
	BoxShadowSecondary
	BoxShadowTertiary
)

type shadowLayer struct{ y, blur, spread, light, dark float32 }

var shadowLayers = map[Elevation][]shadowLayer{
	BoxShadow: {
		{40, 80, 0, 0.06, 0.09}, {20, 40, 0, 0.05, 0.08}, {10, 20, 0, 0.04, 0.06},
		{5, 10, 0, 0.03, 0.05}, {2, 4, 0, 0.02, 0.03},
	},
	BoxShadowSecondary: {
		{17.5, 23.4, 0, 0.04, 0.06}, {9.4, 12.5, 0, 0.03, 0.05}, {5.25, 7, 0, 0.02, 0.03},
		{2.8, 3.7, -2, 0.01, 0.02}, {1.2, 1.5, 0, 0.01, 0.02},
	},
	BoxShadowTertiary: {{1, 2, 0, 0.04, 0.06}, {2, 4, 0, 0.02, 0.03}},
}

// Elevate gives e the shadow of level and Lobe UI's one-pixel ring around it.
func (t *Tokens) Elevate(e ui.Element, level Elevation) ui.Element {
	for _, l := range shadowLayers[level] {
		a := l.light
		if t.Dark {
			a = l.dark
		}
		e.Shadow(0, l.y, l.blur, l.spread, ui.RGBA(0, 0, 0, a))
	}
	if t.Dark {
		return e.Shadow(0, 0, 0, 1, ui.RGBA(255, 255, 255, 0.18))
	}
	return e.Shadow(0, 0, 0, 1, ui.RGBA(0, 0, 0, 0.05))
}
