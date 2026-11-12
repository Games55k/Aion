package field

import (
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type InputOTPProps struct {
	// Length is how many characters the code has: 6 when zero.
	Length int
	// Alphanumeric takes letters as well as digits.
	Alphanumeric bool
	// Variant picks the cells' surface, as Input's.
	Variant InputVariant
	// Size sets the cells' height (24, 32 or 40) and font size, as Input's.
	Size theme.Size
	// OnComplete runs with the code once all of its characters are in.
	OnComplete func(code string)
}

// InputOTP edits a one-time code in a row of cells, one character each, which typing fills from the
// first and Backspace empties from the last. It returns the text input behind the cells.
func InputOTP(c *ui.Context, value *string, p InputOTPProps) ui.Element {
	tk := theme.TokensOf(c)
	n := p.Length
	if n <= 0 {
		n = 6
	}
	clean := make([]rune, 0, n)
	for _, r := range *value {
		if len(clean) < n && (unicode.IsDigit(r) || p.Alphanumeric && (unicode.IsLetter(r))) {
			clean = append(clean, r)
		}
	}
	if string(clean) != *value {
		*value = string(clean)
	}

	root := ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Start).Shrink(0)
	last := ui.Local(root, "last", func() string { return *value })
	if *value != *last {
		*last = *value
		if len(clean) == n && p.OnComplete != nil {
			p.OnComplete(*value)
		}
	}
	var input ui.Element
	root.Children(func() {
		input = ui.TextInputBase(c, value).MaxLength(n).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Opacity(0).FocusRing(false)
		focused := input.Focused()
		for i := range n {
			cell := fieldRow(c.Key(i), tk, p.Size).Width(32).PaddingX(0).Justify(ui.Center).IgnoreInput(true)
			active := focused && (i == len(clean) || i == n-1 && len(clean) == n)
			styleField(cell, tk, p.Variant, false, active)
			cell.Children(func() {
				if i < len(clean) {
					ui.Text(c, string(clean[i])).FontFeatures("tnum")
				}
			})
		}
	})
	return input
}
