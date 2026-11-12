package display

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type TextType int

const (
	TextDefault TextType = iota
	TextSecondary
	TextSuccess
	TextWarning
	TextDanger
	TextInfo
)

type TextProps struct {
	// Type colors the text: Secondary in the description color, the others in their status color.
	Type TextType
	// Heading sets a heading's size from 1 (2.5× the font size) to 5 (the font size), bold at line
	// height 1.25; 0 is body text.
	Heading int
	// FontSize overrides the size Heading or the theme gives.
	FontSize float32
	// Strong, Italic, Underline and Delete set the weight, slant and lines of the text.
	Strong, Italic, Underline, Delete bool
	// Code sets the text in the code font.
	Code bool
	// Mark highlights the text in yellow with black text.
	Mark bool
	// Disabled dims the text whatever its Type.
	Disabled bool
	// Rows shows at most that many lines, ending in an ellipsis; 0 shows them all.
	Rows int
	// Shiny dims the text and sweeps a bright band across it every 1.5 seconds, for a pending
	// state such as "Thinking…".
	Shiny bool
}

var headingScale = [6]float32{1, 2.5, 2, 1.5, 1.25, 1}

const shinyPeriod = 1500 * time.Millisecond

// Text shows s in the Lobe UI type styles.
func Text(c *ui.Context, s string, p TextProps) ui.Element {
	tk := theme.TokensOf(c)
	t := ui.Text(c, s)
	color := tk.Text
	switch p.Type {
	case TextSecondary:
		color = tk.TextDescription
	case TextSuccess:
		color = tk.Success
	case TextWarning:
		color = tk.Warning
	case TextDanger:
		color = tk.Error
	case TextInfo:
		color = tk.Info
	}
	if p.Heading >= 1 && p.Heading <= 5 {
		t.FontSize(tk.FontSize * headingScale[p.Heading]).FontWeight(700).LineHeight(1.25)
	}
	if p.FontSize > 0 {
		t.FontSize(p.FontSize)
	}
	if p.Strong {
		t.FontWeight(700)
	}
	if p.Italic {
		t.Italic()
	}
	if p.Underline {
		t.Underline()
	}
	if p.Delete {
		t.Strikethrough()
	}
	if p.Code {
		t.Font("monospace")
	}
	if p.Mark {
		t.Background(tk.Yellow)
		color = ui.RGB(0, 0, 0)
	}
	if p.Disabled {
		color = tk.TextDisabled
		t.Cursor(ui.CursorNotAllowed)
	}
	switch {
	case p.Rows == 1:
		t.SingleLine()
	case p.Rows > 1:
		t.MaxLines(p.Rows)
	}
	t.TextColor(color)
	if p.Shiny {
		t.MaskLinearGradient(shinyMask(t.Loop("shiny", shinyPeriod, ui.Linear)))
	}
	return t
}

// The band is CSS's repeating 200%-wide gradient sliding from background-position 100% to -100%:
// its peak, half a text width wide on each side, crosses from the left edge to two widths right
// while the previous tile's peak comes in behind it, with the rest of the text at 28% opacity.
func shinyMask(progress float32) ui.LinearGradient {
	const dim = 0.28
	peaks := [2]float32{2 * progress, 2*progress - 2}
	at := func(x float32) ui.Color {
		a := float32(dim)
		for _, p := range peaks {
			d := x - p
			if d < 0 {
				d = -d
			}
			if d < 0.5 {
				a = max(a, 1-(1-dim)*d/0.5)
			}
		}
		return ui.RGBA(0, 0, 0, a)
	}
	stops := []ui.GradientStop{{Color: at(0), At: 0}}
	for _, p := range []float32{peaks[1] - 0.5, peaks[1], peaks[1] + 0.5, peaks[0] - 0.5, peaks[0], peaks[0] + 0.5} {
		if p > 0 && p < 1 {
			stops = append(stops, ui.GradientStop{Color: at(p), At: p})
		}
	}
	stops = append(stops, ui.GradientStop{Color: at(1), At: 1})
	return ui.LinearGradient{Angle: 120, Stops: stops}
}
