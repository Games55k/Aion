package action

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"time"

	"github.com/egoist/mygo/ui"
)

type ButtonVariant int

const (
	Default ButtonVariant = iota
	Primary
	Dashed
	Filled
	Text
	Link
)

type ButtonProps struct {
	// Variant picks the surface: Primary fills with the primary color, Default and Dashed sit on
	// the container with a border, Filled uses a neutral fill, Text and Link have no surface.
	Variant ButtonVariant
	// Size sets the height (24, 32 or 40), padding, font size and corner radius.
	Size theme.Size
}

type buttonMetrics struct {
	height, paddingX, fontSize float32
}

var buttonTransition = ui.ElementTransition{
	Duration: 160 * time.Millisecond,
	Ease:     ui.CubicBezier(0.32, 0.72, 0, 1),
	Colors:   true,
}

func Button(c *ui.Context, label string, p ButtonProps) ui.Element {
	return StyleButton(c, ui.ButtonBase(c), label, p)
}

// StyleButton gives b, a button of any kind, as a menu's trigger (ui.MenuBase), the look of Button
// with label.
func StyleButton(c *ui.Context, b ui.Element, label string, p ButtonProps) ui.Element {
	tk := theme.TokensOf(c)
	m, radius := buttonSize(tk, p.Size)

	b.Height(m.height).PaddingX(m.paddingX).Gap(6).Radius(radius).
		FontSize(m.fontSize).FontWeight(500).
		Cursor(ui.CursorPointer).Transition(buttonTransition)

	bg, border, fg := buttonColors(tk, p.Variant, stateOf(b))
	b.Background(bg).Border(1, border).TextColor(fg)
	switch p.Variant {
	case Dashed:
		b.BorderStyle(ui.BorderDashed)
	case Link:
		b.PaddingX(0)
	}

	if label != "" {
		b.Children(func() { ui.Text(c, label).SingleLine() })
	}
	return b
}

func buttonSize(tk *theme.Tokens, s theme.Size) (buttonMetrics, float32) {
	switch s {
	case theme.Small:
		return buttonMetrics{24, 8, 12}, tk.BorderRadiusSM
	case theme.Large:
		return buttonMetrics{40, 16, 14}, tk.BorderRadius
	default:
		return buttonMetrics{32, 14, 13}, tk.BorderRadiusSM
	}
}

type buttonState int

const (
	buttonRest buttonState = iota
	buttonHover
	buttonActive
)

func stateOf(e ui.Element) buttonState {
	switch {
	case e.Pressed():
		return buttonActive
	case e.Hovered():
		return buttonHover
	}
	return buttonRest
}

func buttonColors(tk *theme.Tokens, v ButtonVariant, s buttonState) (bg, border, fg ui.Color) {
	pick := func(rest, hover, active ui.Color) ui.Color {
		switch s {
		case buttonActive:
			return active
		case buttonHover:
			return hover
		}
		return rest
	}
	switch v {
	case Primary:
		c := pick(tk.Primary, tk.PrimaryHover, tk.PrimaryActive)
		return c, c, tk.BgLayout
	case Filled:
		return pick(tk.FillTertiary, tk.FillSecondary, tk.Fill), ui.Transparent, tk.Text
	case Text:
		return pick(ui.Transparent, tk.FillSecondary, tk.FillSecondary), ui.Transparent, tk.Text
	case Link:
		return ui.Transparent, ui.Transparent, pick(tk.Primary, tk.PrimaryHover, tk.PrimaryHover)
	default:
		bg = pick(tk.BgContainer, layer(tk.BgContainer, tk.FillTertiary), layer(tk.BgContainer, tk.FillSecondary))
		border = tk.Border
		if s != buttonRest {
			border = theme.Mix(tk.Border, tk.Text, 0.08)
		}
		return bg, border, tk.Text
	}
}

func layer(base, top ui.Color) ui.Color {
	return base.Mix(ui.RGB(top.R, top.G, top.B), float32(top.A)/255)
}
