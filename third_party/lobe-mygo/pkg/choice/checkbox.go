package choice

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"time"

	"github.com/egoist/mygo/ui"
)

type CheckboxShape int

const (
	CheckboxSquare CheckboxShape = iota
	CheckboxCircle
)

type CheckboxProps struct {
	// Indeterminate shows a dash instead of a check, for a box over a partly checked group.
	Indeterminate bool
	// Shape rounds the box: CheckboxSquare has 4 DIP corners, CheckboxCircle is round.
	Shape CheckboxShape
	// Size is the box's side in DIPs; 0 is 16.
	Size float32
}

var (
	checkGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`))
	dashGlyph  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/></svg>`))

	checkboxTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: ui.CubicBezier(0.215, 0.61, 0.355, 1), Colors: true}
)

func Checkbox(c *ui.Context, checked *bool, label string, p CheckboxProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 16
	}

	state := ui.CheckOff
	switch {
	case p.Indeterminate:
		state = ui.CheckMixed
	case *checked:
		state = ui.CheckOn
	}
	row := ui.CheckboxStateBase(c, &state).Gap(8).Cursor(ui.CursorPointer)
	if row.Changed() {
		*checked = state == ui.CheckOn
	}

	filled := state != ui.CheckOff
	border, bg := tk.BorderSecondary, tk.BgContainer
	switch {
	case filled:
		border, bg = tk.Primary, tk.Primary
	case row.Hovered():
		border = tk.Border
	}

	radius := float32(4)
	glyphScale := float32(0.75)
	if p.Shape == CheckboxCircle {
		radius, glyphScale = size/2, 0.66
	}

	row.Children(func() {
		box := ui.Box(c).Size(size, size).Radius(radius).Border(1, border).Background(bg).
			TextColor(tk.BgLayout).Center().Transition(checkboxTransition)
		box.Children(func() {
			switch state {
			case ui.CheckOn:
				ui.Icon(c, checkGlyph).FontSize(size * glyphScale)
			case ui.CheckMixed:
				ui.Icon(c, dashGlyph).FontSize(size * glyphScale)
			}
		})
		if label != "" {
			ui.Text(c, label).SingleLine()
		}
	})
	return row
}
