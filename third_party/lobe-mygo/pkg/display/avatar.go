package display

import (
	"hash/fnv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type AvatarShape int

const (
	AvatarSquare AvatarShape = iota
	AvatarCircle
)

type AvatarFallback int

const (
	AvatarMarble AvatarFallback = iota
	AvatarText
)

type AvatarProps struct {
	// Size is the width and height: 48 DIPs when zero.
	Size float32
	// Shape rounds the corners by a sixth of the size, or makes the avatar a circle.
	Shape AvatarShape
	// Image shows instead of the text, covering the avatar.
	Image ui.ImageSource
	// Fallback is what a name shows without an Image: marbled colors seeded by the name, the same
	// for the same name, or its first two letters.
	Fallback AvatarFallback
	// Full shows all of the text in the AvatarText fallback, not only its first two letters.
	Full bool
	// Background fills the avatar, and the text takes the color that reads on it.
	Background ui.Color
}

var avatarPalette = []ui.Color{ui.Hex("#383838"), ui.Hex("#7858f9"), ui.Hex("#f8f7bf"), ui.Hex("#ffdd1f"), ui.Hex("#f88c13")}

// Avatar shows a user or an agent: an Image, an emoji, or a name drawn as the Fallback says.
func Avatar(c *ui.Context, avatar string, p AvatarProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size == 0 {
		size = 48
	}
	root := ui.Row(c).Size(size, size).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Clip().Unselectable()
	switch {
	case p.Shape == AvatarCircle:
		root.Radius(size / 2)
	case size < 24:
		root.Radius(size * 0.33)
	default:
		root.Radius(max(size/6, 2))
	}
	emoji := isEmoji(avatar)
	bg := p.Background
	if bg == (ui.Color{}) && !emoji {
		bg = tk.Border
	}
	root.Background(bg)

	root.Children(func() {
		switch {
		case p.Image != nil:
			ui.Image(c, p.Image).Fill().Fit(ui.Cover)
		case emoji:
			scale := float32(0.8)
			if p.Background != (ui.Color{}) {
				scale *= 0.85
			}
			ui.Text(c, avatar).FontSize(size * scale).LineHeight(1).SingleLine()
		case p.Fallback == AvatarText:
			text := strings.ToUpper(avatar)
			if !p.Full {
				text = string([]rune(text)[:min(2, utf8.RuneCountInString(text))])
			}
			ui.Text(c, text).FontSize(size * 0.5).FontWeight(700).LineHeight(1).SingleLine().TextColor(readable(bg))
		default:
			marble(c, avatar, size)
		}
	})
	return root
}

func isEmoji(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return r >= 0x1F000 || unicode.Is(unicode.So, r)
}

func readable(bg ui.Color) ui.Color {
	if luminance(bg) > 0.4 {
		return ui.RGB(0, 0, 0)
	}
	return ui.RGB(255, 255, 255)
}

func luminance(c ui.Color) float32 {
	return (0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)) / 255
}

func marble(c *ui.Context, seed string, size float32) {
	h := fnv.New32a()
	h.Write([]byte(seed))
	n := h.Sum32()
	pick := func(i uint32) ui.Color { return avatarPalette[(n>>(i*3)+i)%uint32(len(avatarPalette))] }
	unit := func(i uint32) float32 { return float32((n>>(i*4))&0xf)/15 - 0.5 }
	base, a, b := pick(0), pick(1), pick(2)
	if a == base {
		a = avatarPalette[(n+1)%uint32(len(avatarPalette))]
	}
	if b == a {
		b = avatarPalette[(n+2)%uint32(len(avatarPalette))]
	}
	blur := size * 7 / 80
	ui.Box(c).Absolute().Fill().Background(base)
	ui.Box(c).Absolute().Size(size*0.9, size*1.1).Radius(size * 0.45).Background(a).
		Left(size * (0.35 + 0.3*unit(1))).Top(size * (-0.2 + 0.3*unit(2))).Rotate(180 * unit(3)).Blur(blur)
	ui.Box(c).Absolute().Size(size*0.8, size*0.7).Radius(size * 0.35).Background(b).
		Left(size * (-0.15 + 0.3*unit(4))).Top(size * (0.4 + 0.3*unit(5))).Rotate(180 * unit(6)).Blur(blur)
}
