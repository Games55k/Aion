package display

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

type TagVariant int

const (
	TagFilled TagVariant = iota
	TagOutlined
	TagBorderless
	TagSolid
)

type TagColor string

const (
	TagRed      TagColor = "red"
	TagVolcano  TagColor = "volcano"
	TagOrange   TagColor = "orange"
	TagGold     TagColor = "gold"
	TagYellow   TagColor = "yellow"
	TagLime     TagColor = "lime"
	TagGreen    TagColor = "green"
	TagCyan     TagColor = "cyan"
	TagBlue     TagColor = "blue"
	TagGeekblue TagColor = "geekblue"
	TagPurple   TagColor = "purple"
	TagMagenta  TagColor = "magenta"
	TagGray     TagColor = "gray"

	TagError      TagColor = "error"
	TagWarning    TagColor = "warning"
	TagSuccess    TagColor = "success"
	TagInfo       TagColor = "info"
	TagProcessing TagColor = "processing"
)

type TagProps struct {
	// Variant picks the surface: Filled tints the background, Outlined adds a border on the
	// container, Borderless keeps only the text color, Solid fills with the color itself.
	Variant TagVariant
	// Color is a preset palette (TagRed…TagGray), a status (TagError, TagWarning, TagSuccess,
	// TagInfo, TagProcessing) or a hex color such as TagColor("#7c3aed"). Empty is neutral.
	Color TagColor
	// Size sets the height (20, 22 or 28), padding and corner radius.
	Size theme.Size
	// Round makes the corners fully round.
	Round bool
	// Icon shows before the label, in the label's color.
	Icon *ui.SVG
	// Closable adds a close button that hides the tag.
	Closable bool
	// OnClose runs when the close button hides the tag.
	OnClose func()
}

type tagColors struct {
	text, bg, border ui.Color
	hasText          bool
}

func Tag(c *ui.Context, label string, p TagProps) ui.Element {
	tk := theme.TokensOf(c)
	tag := ui.Row(c).Gap(5).FontSize(tk.FontSizeSM)
	closed := ui.Local(tag, "closed", func() bool { return false })
	tag.Hidden(*closed)

	height, padding, radius := tagSize(p.Size, p.Round)
	tag.Height(height).PaddingX(padding).Radius(radius)

	colors := tagColorsFor(tk, p.Color, p.Variant)
	switch p.Variant {
	case TagOutlined:
		border := tk.BorderSecondary
		if colors.border.A > 0 {
			border = colors.border
		}
		bg := tk.BgContainer
		if p.Color != "" {
			bg = colors.bg
		}
		tag.Background(bg).Border(1, border)
	case TagBorderless:
	default:
		bg := tk.FillTertiary
		if p.Color != "" {
			bg = colors.bg
		}
		tag.Background(bg)
	}
	text := tk.TextSecondary
	if colors.hasText {
		text = colors.text
	}
	tag.TextColor(text)

	tag.Children(func() {
		if p.Icon != nil {
			ui.Icon(c, p.Icon)
		}
		ui.Text(c, label).SingleLine()
		if p.Closable {
			closeButton := ui.ButtonBase(c).Label("Close").Cursor(ui.CursorPointer)
			glyph := tk.Icon
			if closeButton.Hovered() {
				glyph = tk.TextHeading
			}
			closeButton.TextColor(glyph).Children(func() { ui.Icon(c, lucide.X()).FontSize(10) })
			if closeButton.Clicked() {
				*closed = true
				tag.Hidden(true)
				if p.OnClose != nil {
					p.OnClose()
				}
			}
		}
	})
	return tag
}

func tagSize(s theme.Size, round bool) (height, padding, radius float32) {
	switch s {
	case theme.Small:
		height, padding, radius = 20, 4, 3
		if round {
			padding = 8
		}
	case theme.Large:
		height, padding, radius = 28, 12, 6
		if round {
			padding = 14
		}
	default:
		height, padding, radius = 22, 8, 3
		if round {
			padding = 10
		}
	}
	if round {
		radius = 999
	}
	return height, padding, radius
}

func tagColorsFor(tk *theme.Tokens, color TagColor, v TagVariant) tagColors {
	solid, borderless, filled := v == TagSolid, v == TagBorderless, v == TagFilled
	tinted := func(base, text, fillTertiary, fillQuaternary ui.Color) tagColors {
		out := tagColors{hasText: true, text: text, bg: fillTertiary, border: fillTertiary}
		if filled {
			out.border = fillQuaternary
		}
		if borderless {
			out.bg = ui.Transparent
		}
		if solid {
			out = tagColors{hasText: true, text: readableOn(base), bg: base, border: base}
		}
		return out
	}

	if strings.HasPrefix(string(color), "#") {
		hex := ui.Hex(string(color))
		switch {
		case solid:
			return tagColors{hasText: true, text: readableOn(hex), bg: hex, border: hex}
		case borderless:
			return tagColors{hasText: true, text: hex}
		}
		return tagColors{hasText: true, text: tk.BgLayout, bg: hex}
	}

	if status, ok := tagStatus(tk, color); ok {
		return tinted(status[0], status[0], status[1], status[2])
	}
	if name := string(color); name != "" {
		base, okBase := tk.Color(strings.ToUpper(name[:1]) + name[1:])
		if okBase {
			title := strings.ToUpper(name[:1]) + name[1:]
			active, _ := tk.Color(title + "Active")
			fill3, _ := tk.Color(title + "FillTertiary")
			fill4, _ := tk.Color(title + "FillQuaternary")
			return tinted(base, active, fill3, fill4)
		}
	}
	return tagColors{}
}

func tagStatus(tk *theme.Tokens, color TagColor) ([3]ui.Color, bool) {
	switch color {
	case TagError:
		return [3]ui.Color{tk.Error, tk.ErrorFillTertiary, tk.ErrorFillQuaternary}, true
	case TagWarning:
		return [3]ui.Color{tk.Warning, tk.WarningFillTertiary, tk.WarningFillQuaternary}, true
	case TagSuccess:
		return [3]ui.Color{tk.Success, tk.SuccessFillTertiary, tk.SuccessFillQuaternary}, true
	case TagInfo, TagProcessing:
		return [3]ui.Color{tk.Info, tk.InfoFillTertiary, tk.InfoFillQuaternary}, true
	}
	return [3]ui.Color{}, false
}

func readableOn(bg ui.Color) ui.Color {
	if oklabLightness(bg) > 0.62 {
		return ui.RGB(0, 0, 0)
	}
	return ui.RGB(255, 255, 255)
}

func oklabLightness(c ui.Color) float64 {
	lin := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	r, g, b := lin(c.R), lin(c.G), lin(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
}
