package choice

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type RateProps struct {
	// Count is how many stars show: 5 when zero.
	Count int
	// AllowHalf lets a click on a star's left half choose half of it.
	AllowHalf bool
	// Size is a star's size in DIPs: 20 when zero. The stars are half of it apart.
	Size float32
	// Character replaces the star.
	Character *ui.SVG
	// Color fills the chosen stars: the warning color when unset.
	Color ui.Color
	// Disabled shows the rating without letting it change.
	Disabled bool
}

var (
	starGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11.525 2.295a.53.53 0 0 1 .95 0l2.31 4.679a2.123 2.123 0 0 0 1.595 1.16l5.166.756a.53.53 0 0 1 .294.904l-3.736 3.638a2.123 2.123 0 0 0-.611 1.878l.882 5.14a.53.53 0 0 1-.771.56l-4.618-2.428a2.122 2.122 0 0 0-1.973 0L6.396 21.01a.53.53 0 0 1-.77-.56l.881-5.139a2.122 2.122 0 0 0-.611-1.879L2.16 9.795a.53.53 0 0 1 .294-.906l5.165-.755a2.122 2.122 0 0 0 1.597-1.16z"/></svg>`))
	starHover = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.CSSEase}
)

// Rate shows a rating of *value out of p.Count stars, which a click on a star sets and the
// pointer previews; the arrows, Home and End move it while it has the focus.
func Rate(c *ui.Context, value *float64, p RateProps) ui.Element {
	tk := theme.TokensOf(c)
	count := p.Count
	if count <= 0 {
		count = 5
	}
	size := p.Size
	if size <= 0 {
		size = 20
	}
	gap := size / 2
	step := 1.0
	if p.AllowHalf {
		step = 0.5
	}
	glyph, color := p.Character, p.Color
	if glyph == nil {
		glyph = starGlyph
	}
	if color == (ui.Color{}) {
		color = tk.Warning
	}
	clamp := func(v float64) float64 { return math.Min(math.Max(math.Round(v/step)*step, 0), float64(count)) }

	root := ui.ButtonBase(c).Role(ui.RoleSlider).Row().Gap(gap).AlignItems(ui.Center).Shrink(0).AlignSelf(ui.Start)
	if p.Disabled {
		root.Disabled(true).Cursor(ui.CursorNotAllowed).Opacity(0.5)
	} else {
		root.Cursor(ui.CursorPointer)
	}
	at := func(x float32) float64 {
		i := min(int(x/(size+gap)), count-1)
		v := float64(i + 1)
		if p.AllowHalf && x-float32(i)*(size+gap) < size/2 {
			v -= 0.5
		}
		return v
	}
	x, _, over := root.PointerPosition()
	interactive := !p.Disabled
	if interactive {
		if root.Clicked() {
			*value = at(x)
		}
		for _, k := range []struct {
			key ui.Key
			to  func() float64
		}{
			{ui.KeyRight, func() float64 { return *value + step }}, {ui.KeyUp, func() float64 { return *value + step }},
			{ui.KeyLeft, func() float64 { return *value - step }}, {ui.KeyDown, func() float64 { return *value - step }},
			{ui.KeyHome, func() float64 { return 0 }}, {ui.KeyEnd, func() float64 { return float64(count) }},
		} {
			if root.Shortcut(0, k.key) {
				*value = clamp(k.to())
			}
		}
	}
	shown := *value
	hovered := -1
	if interactive && over {
		shown = at(x)
		hovered = min(int(x/(size+gap)), count-1)
	}
	root.Range(0, float64(count), *value).Step(step)

	root.Children(func() {
		for i := range count {
			fill := float32(math.Min(math.Max(shown-float64(i), 0), 1))
			s := ui.Box(c.Key(i)).Size(size, size).Shrink(0)
			scale := float32(1)
			if i == hovered {
				scale = 1.1
			}
			scale = s.AnimateTo("scale", scale, starHover)
			s.Scale(scale, scale).Children(func() {
				ui.Icon(c, glyph).FontSize(size).TextColor(tk.Text).Opacity(0.12).Absolute().Left(0).Top(0)
				if fill > 0 {
					ui.Box(c).Absolute().Left(0).Top(0).Height(size).Width(size * fill).Clip().Children(func() {
						ui.Icon(c, glyph).FontSize(size).TextColor(color).Absolute().Left(0).Top(0)
					})
				}
			})
		}
	})
	return root
}
