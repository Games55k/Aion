package action

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
)

type SegmentedVariant int

const (
	SegmentedFilled SegmentedVariant = iota
	SegmentedOutlined
)

type SegmentedProps struct {
	// Variant picks the track: Filled sits on the layout color with a faint border, Outlined
	// has only a border.
	Variant SegmentedVariant
	// Size sets the item height (26, 32 or 36), padding and font size.
	Size theme.Size
	// Block stretches the track across its parent and shares the width between the items.
	Block bool
}

type segmentedMatch struct{ selected *int }

var segmentPress = ui.Animation{Duration: 120 * time.Millisecond, Ease: motion.EaseOut}

func Segmented(c *ui.Context, selected *int, options []string, p SegmentedProps) ui.Element {
	tk := theme.TokensOf(c)
	height, padding, fontSize := float32(32), float32(12), float32(13)
	switch p.Size {
	case theme.Small:
		height, padding, fontSize = 26, 10, 12
	case theme.Large:
		height, padding, fontSize = 36, 16, 14
	}

	indicator := tk.BgElevated
	parts := ui.SegmentedBase(c, selected, len(options))
	track := parts.Track.Padding(3).Gap(4).Radius(tk.BorderRadiusLG)
	if p.Variant == SegmentedOutlined {
		track.Border(1, tk.BorderSecondary)
		indicator = tk.FillSecondary
	} else {
		track.Background(tk.BgLayout).Border(1, tk.FillQuaternary)
	}
	if p.Block {
		track.AlignSelf(ui.Stretch)
	}

	slide := ui.ElementTransition{Duration: 240 * time.Millisecond, Ease: motion.EaseOut, Match: segmentedMatch{selected}}

	track.Children(func() {
		for i, option := range options {
			seg := parts.Segment(i).Height(height).PaddingX(padding).Radius(tk.BorderRadius).
				FontSize(fontSize).FontWeight(500).Cursor(ui.CursorPointer)
			if p.Block {
				seg.Grow(1).Basis(0)
			}
			text := tk.TextSecondary
			if i == *selected || seg.Hovered() {
				text = tk.Text
			}
			pose := ui.Pose{}
			if seg.Pressed() {
				pose.Scale = 0.98
			}
			seg.TextColor(text).AnimatePose(pose, segmentPress)
			seg.Children(func() {
				if i == *selected {
					ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
						Radius(tk.BorderRadius).Background(indicator).ZIndex(-1).Transition(slide)
				}
				ui.Text(c, option).SingleLine()
			})
		}
	})
	return track
}
