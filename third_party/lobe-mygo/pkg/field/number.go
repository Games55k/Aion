package field

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type InputNumberProps struct {
	// Min and Max bound the value; equal, as both zero, they leave it unbounded.
	Min, Max float64
	// Step is how far the arrows, Up and Down move the value: 1 when zero.
	Step float64
	// Precision fixes the digits shown after the point; 0 shows as many as the value has.
	Precision int
	// NoControls hides the up and down arrows at the end.
	NoControls bool
	// Placeholder shows while the field is empty.
	Placeholder string
	// Variant picks the surface, as Input's.
	Variant InputVariant
	// Size sets the height (24, 32 or 40), padding and font size, as Input's.
	Size theme.Size
}

type numberDraft struct {
	text    string
	focused bool
}

// InputNumber edits *value as a number: typed text takes effect on Enter or when the field loses
// the focus, clamped to the bounds, and the arrows at the end, Up and Down step it. It returns the
// text input.
func InputNumber(c *ui.Context, value *float64, p InputNumberProps) ui.Element {
	tk := theme.TokensOf(c)
	step := p.Step
	if step <= 0 {
		step = 1
	}
	format := func(v float64) string {
		if p.Precision > 0 {
			return strconv.FormatFloat(v, 'f', p.Precision, 64)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	clamp := func(v float64) float64 {
		if p.Min != p.Max {
			v = math.Min(math.Max(v, p.Min), p.Max)
		}
		if p.Precision > 0 {
			k := math.Pow(10, float64(p.Precision))
			v = math.Round(v*k) / k
		}
		return v
	}
	set := func(v float64) { *value = clamp(v) }

	root := fieldRow(c, tk, p.Size).Gap(8)
	draft := ui.Local(root, "draft", func() numberDraft { return numberDraft{text: format(*value)} })
	commit := func() {
		if f, err := strconv.ParseFloat(strings.TrimSpace(draft.text), 64); err == nil {
			set(f)
		}
		draft.text = format(*value)
	}

	var input ui.Element
	root.Children(func() {
		input = ui.TextInputBase(c, &draft.text).Grow(1).MinWidth(0).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder).FontFeatures("tnum")
		focused := input.Focused()
		switch {
		case input.Submitted(), draft.focused && !focused:
			commit()
		case input.Shortcut(0, ui.KeyUp):
			commit()
			set(*value + step)
			draft.text = format(*value)
		case input.Shortcut(0, ui.KeyDown):
			commit()
			set(*value - step)
			draft.text = format(*value)
		case !focused:
			draft.text = format(*value)
		}
		draft.focused = focused
		if !p.NoControls {
			ui.Column(c).Shrink(0).Margin(0, -6, 0, 0).Children(func() {
				if numberControl(c, tk, p.Size, lucide.ChevronUp(), "Increase") {
					set(*value + step)
					draft.text = format(*value)
				}
				if numberControl(c, tk, p.Size, lucide.ChevronDown(), "Decrease") {
					set(*value - step)
					draft.text = format(*value)
				}
			})
		}
	})
	styleField(root, tk, p.Variant, false, input.Focused())
	return input
}

func numberControl(c *ui.Context, tk *theme.Tokens, size theme.Size, glyph *ui.SVG, label string) bool {
	w, h, icon, radius := float32(20), float32(13), float32(12), float32(4)
	switch size {
	case theme.Small:
		w, h, icon, radius = 16, 10, 10, 3
	case theme.Large:
		w, h, icon = 22, 15, 13
	}
	b := ui.ButtonBase(c.Key(label)).Label(label).Size(w, h).Radius(radius).Center().Cursor(ui.CursorPointer).
		TextColor(tk.TextTertiary).Transition(optionTransition)
	if b.Hovered() {
		b.Background(tk.FillSecondary).TextColor(tk.Text)
	}
	b.Children(func() { ui.Icon(c, glyph).FontSize(icon) })
	return b.Clicked()
}

// InputPassword is an Input that hides what it holds, with an eye button at the end that shows
// it. It returns the text input.
func InputPassword(c *ui.Context, value *string, p InputProps) ui.Element {
	tk := theme.TokensOf(c)
	root := fieldRow(c, tk, p.Size).Gap(8)
	visible := ui.Local(root, "visible", func() bool { return false })
	var input ui.Element
	root.Children(func() {
		affix(c, tk, p.Prefix)
		input = ui.TextInputBase(c, value).Grow(1).MinWidth(0).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder)
		if !*visible {
			input.Password()
		}
		label, glyph := "Show password", lucide.EyeOff()
		if *visible {
			label, glyph = "Hide password", lucide.Eye()
		}
		b := ui.ButtonBase(c).Label(label).Center().Shrink(0).Cursor(ui.CursorPointer).
			TextColor(tk.TextTertiary).Transition(optionTransition)
		if b.Hovered() {
			b.TextColor(tk.Text)
		}
		if b.Clicked() {
			*visible = !*visible
		}
		b.Children(func() { ui.Icon(c, glyph).FontSize(16) })
	})
	styleField(root, tk, p.Variant, p.Invalid, input.Focused())
	return input
}
