package field

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
)

type InputVariant int

const (
	InputOutlined InputVariant = iota
	InputFilled
	InputBorderless
)

type InputProps struct {
	// Variant picks the surface: Outlined sits on the container with a border, Filled uses a
	// neutral fill, Borderless has neither.
	Variant InputVariant
	// Size sets the height (24, 32 or 40), padding, font size and corner radius.
	Size theme.Size
	// Placeholder shows while the input is empty.
	Placeholder string
	// Prefix shows an icon before the text.
	Prefix *ui.SVG
	// Suffix shows an icon after the text.
	Suffix *ui.SVG
	// Invalid draws the error border, and the error ring while focused.
	Invalid bool
}

type TextAreaProps struct {
	// Variant picks the surface, as Input's.
	Variant InputVariant
	// Placeholder shows while the text area is empty.
	Placeholder string
	// MinRows is the height in lines when empty; 0 is 2.
	MinRows int
	// MaxRows is how many lines it grows to before it scrolls; 0 is 8.
	MaxRows int
	// Invalid draws the error border, and the error ring while focused.
	Invalid bool
}

var fieldTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: motion.EaseOut, Colors: true, Shadows: true}

func Input(c *ui.Context, value *string, p InputProps) ui.Element {
	tk := theme.TokensOf(c)
	root := fieldRow(c, tk, p.Size).Gap(8)
	var input ui.Element
	root.Children(func() {
		affix(c, tk, p.Prefix)
		input = ui.TextInputBase(c, value).Grow(1).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder)
		affix(c, tk, p.Suffix)
	})
	styleField(root, tk, p.Variant, p.Invalid, input.Focused())
	return input
}

func TextArea(c *ui.Context, value *string, p TextAreaProps) ui.Element {
	tk := theme.TokensOf(c)
	minRows, maxRows := p.MinRows, p.MaxRows
	if minRows <= 0 {
		minRows = 2
	}
	if maxRows <= 0 {
		maxRows = 8
	}
	root := ui.Column(c).PaddingX(12).PaddingY(8).Radius(tk.BorderRadius).
		FontSize(14).TextColor(tk.Text).Cursor(ui.CursorText).Transition(fieldTransition)
	var area ui.Element
	root.Children(func() {
		area = ui.TextAreaBase(c, value).Lines(minRows, maxRows).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder)
	})
	styleField(root, tk, p.Variant, p.Invalid, area.Focused())
	return area
}

func affix(c *ui.Context, tk *theme.Tokens, svg *ui.SVG) {
	if svg != nil {
		ui.Icon(c, svg).TextColor(tk.TextTertiary)
	}
}

func styleField(root ui.Element, tk *theme.Tokens, v InputVariant, invalid, focused bool) {
	hovered := root.Hovered() && !focused && !root.IsDisabled()
	switch v {
	case InputFilled:
		bg := tk.FillTertiary
		if hovered {
			bg = tk.FillSecondary
		}
		root.Background(bg)
	case InputBorderless:
		root.Background(ui.Transparent)
	default:
		root.Background(tk.BgContainer)
	}

	border := ui.Transparent
	if v == InputOutlined {
		border = tk.BorderSecondary
		if hovered {
			border = tk.Border
		}
		if focused && !invalid {
			border = tk.Primary
		}
	}
	if invalid && !hovered {
		border = tk.Error
	}
	root.Border(1, border)

	switch {
	case focused && invalid:
		root.Shadow(0, 0, 0, 2, tk.ErrorBg)
	case focused && v == InputOutlined:
		root.Shadow(0, 0, 0, 2, tk.PrimaryBg)
	}
}

func fieldRow(c *ui.Context, tk *theme.Tokens, size theme.Size) ui.Element {
	return styleFieldRow(ui.Row(c), tk, size)
}

func styleFieldRow(e ui.Element, tk *theme.Tokens, size theme.Size) ui.Element {
	minHeight, fontSize, radius, padding := float32(32), float32(14), tk.BorderRadius, float32(12)
	switch size {
	case theme.Small:
		minHeight, fontSize, radius, padding = 24, 12, tk.BorderRadiusSM, 8
	case theme.Large:
		minHeight, fontSize, radius = 40, 16, tk.BorderRadiusLG
	}
	return e.MinHeight(minHeight).PaddingX(padding).Radius(radius).AlignItems(ui.Center).
		FontSize(fontSize).TextColor(tk.Text).Cursor(ui.CursorText).Transition(fieldTransition)
}
