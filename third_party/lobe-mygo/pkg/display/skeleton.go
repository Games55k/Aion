package display

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SkeletonAnimation int

const (
	SkeletonFade SkeletonAnimation = iota
	SkeletonSweep
	SkeletonStatic
)

type SkeletonProps struct {
	// Width is the width, the full width when zero; Height the height, the font size when zero.
	Width, Height float32
	// Circle makes the block round.
	Circle bool
	// Animation pulses the block's opacity, sweeps a highlight across it, or keeps it still.
	Animation SkeletonAnimation
}

// Skeleton holds the place of content that is loading.
func Skeleton(c *ui.Context, p SkeletonProps) ui.Element {
	tk := theme.TokensOf(c)
	e := ui.Box(c).Background(tk.FillContent).Radius(tk.BorderRadius).Shrink(0).Unselectable()
	if p.Width > 0 {
		e.Width(p.Width)
	} else {
		e.FillWidth()
	}
	h := p.Height
	if h == 0 {
		h = tk.FontSize
	}
	e.Height(h)
	if p.Circle {
		e.Radius(h / 2)
	}
	switch p.Animation {
	case SkeletonFade:
		e.Opacity(1 - 0.5*e.Loop("fade", 1600*time.Millisecond, ui.Bounce(ui.EaseInOut)))
	case SkeletonSweep:
		e.Clip().Children(func() {
			at := e.Loop("sweep", 1400*time.Millisecond, motion.CSSEase)
			ui.Box(c).Absolute().Fill().TranslatePercent(200*at-100, 0).
				LinearGradient(ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
					{Color: ui.Transparent, At: 0}, {Color: tk.Fill, At: 0.5}, {Color: ui.Transparent, At: 1},
				}})
		})
	}
	return e
}

type SkeletonTextProps struct {
	// Rows is how many lines show: one when zero. The last of several is 66% wide.
	Rows int
	// FontSize sizes the lines as text of that size: the theme's when zero.
	FontSize float32
	// Animation pulses, sweeps or keeps the lines still, as Skeleton's does.
	Animation SkeletonAnimation
}

// SkeletonText holds the place of a paragraph that is loading: lines as tall as its text.
func SkeletonText(c *ui.Context, p SkeletonTextProps) ui.Element {
	tk := theme.TokensOf(c)
	font := p.FontSize
	if font == 0 {
		font = tk.FontSize
	}
	rows := max(p.Rows, 1)
	margin := float32(int(font*0.15 + 0.5))
	col := ui.Column(c).FillWidth()
	col.Children(func() {
		for i := range rows {
			line := ui.Column(c.Key(i)).PaddingY(margin)
			if rows > 1 && i == rows-1 {
				line.WidthPercent(66)
			} else {
				line.FillWidth()
			}
			line.Children(func() {
				Skeleton(c, SkeletonProps{Height: float32(int(font*1.3 + 0.5)), Animation: p.Animation})
			})
		}
	})
	return col
}

// SkeletonAvatar holds the place of an Avatar: 40 DIPs when Size is zero.
func SkeletonAvatar(c *ui.Context, size float32, shape AvatarShape, a SkeletonAnimation) ui.Element {
	if size == 0 {
		size = 40
	}
	e := Skeleton(c, SkeletonProps{Width: size, Height: size, Circle: shape == AvatarCircle, Animation: a})
	if shape == AvatarSquare {
		e.Radius(max(size/6, 2))
	}
	return e
}
