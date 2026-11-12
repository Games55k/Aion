package display

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type GroupAvatarItem struct {
	// Avatar is the avatar's emoji or name, as Avatar's.
	Avatar string
	// Image shows instead, covering the avatar.
	Image ui.ImageSource
	// Background fills the avatar.
	Background ui.Color
	// Fallback is what a name shows without an Image, as Avatar's.
	Fallback AvatarFallback
}

type GroupGrid int

const (
	// GroupGridTwo lays out up to four avatars in two columns.
	GroupGridTwo GroupGrid = iota
	// GroupGridThree lays out up to nine avatars in three columns.
	GroupGridThree
	// GroupGridAuto takes three columns for more than four avatars, two otherwise.
	GroupGridAuto
)

type GroupCorner int

const (
	// GroupCornerSquare rounds the group's corners by 15% of its size.
	GroupCornerSquare GroupCorner = iota
	// GroupCornerCircle makes the group a circle, its avatars smaller to fit.
	GroupCornerCircle
	// GroupCornerSquircle, GroupCornerSmooth, GroupCornerIOS and GroupCornerSharp are Lobe's
	// superellipses, drawn as rounded squares nearest their curves: mygo has no masks of a path.
	GroupCornerSquircle
	GroupCornerSmooth
	GroupCornerIOS
	GroupCornerSharp
)

type GroupAvatarProps struct {
	// Avatars are the members, in order; those beyond the grid are left out.
	Avatars []GroupAvatarItem
	// Size is the group's width and height: 32 DIPs when zero.
	Size float32
	// Grid is how many columns the avatars take.
	Grid GroupGrid
	// Corner shapes the group.
	Corner GroupCorner
	// AvatarShape shapes each avatar.
	AvatarShape AvatarShape
}

var groupCornerRadius = map[GroupCorner]float32{
	GroupCornerSquare: 0.15, GroupCornerCircle: 0.5, GroupCornerSquircle: 0.28,
	GroupCornerSmooth: 0.35, GroupCornerIOS: 0.24, GroupCornerSharp: 0.2,
}

// GroupAvatar shows a group, as a group chat, by the avatars of its members in a grid on a light
// fill; one member's avatar shows large.
func GroupAvatar(c *ui.Context, p GroupAvatarProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 32
	}
	cols := 2
	if p.Grid == GroupGridThree || (p.Grid == GroupGridAuto && len(p.Avatars) > 4) {
		cols = 3
	}
	circle := p.Corner == GroupCornerCircle
	share, gaps := float32(0.75), float32(4)
	if circle {
		share, gaps = 0.65, 6
	}
	avatar := floor(size / float32(cols) * share)
	gap := floor((size - avatar*float32(cols)) / gaps)
	total := avatar*float32(cols) + gap
	pitch := (total-float32(cols-1)*gap)/float32(cols) + gap
	items := p.Avatars[:min(len(p.Avatars), cols*cols)]

	root := ui.Box(c).Size(size, size).Shrink(0).Clip().Background(tk.FillTertiary).Radius(size * groupCornerRadius[p.Corner])
	return root.Children(func() {
		if len(items) == 1 {
			one := size * 0.8
			groupMember(c.Key(0), items[0], one, p.AvatarShape).Absolute().Left((size - one) / 2).Top((size - one) / 2)
			return
		}
		rows := (len(items) + cols - 1) / cols
		top := (size - (float32(rows-1)*pitch + avatar)) / 2
		left := (size - total) / 2
		for i, it := range items {
			groupMember(c.Key(i), it, avatar, p.AvatarShape).Absolute().
				Left(left + float32(i%cols)*pitch).Top(top + float32(i/cols)*pitch)
		}
	})
}

func groupMember(c *ui.Context, it GroupAvatarItem, size float32, shape AvatarShape) ui.Element {
	return Avatar(c, it.Avatar, AvatarProps{Size: size, Shape: shape, Image: it.Image, Background: it.Background, Fallback: it.Fallback})
}

func floor(v float32) float32 { return float32(int(v)) }
