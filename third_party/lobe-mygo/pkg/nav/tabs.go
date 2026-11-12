package nav

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type TabsVariant int

const (
	TabsRounded TabsVariant = iota
	TabsSquare
	TabsPoint
)

type TabItem struct {
	// Label is the tab's text.
	Label string
	// Icon shows before the label.
	Icon *ui.SVG
	// Disabled dims the tab and keeps it from being chosen.
	Disabled bool
}

type TabsProps struct {
	// Items are the tabs, in order.
	Items []TabItem
	// Variant picks the look: Rounded slides a raised pill in a layout-colored track, Square
	// underlines the chosen tab in the primary color, Point puts a dot under it.
	Variant TabsVariant
	// Size sets the tab height (26, 32 or 36), padding and font size.
	Size theme.Size
	// Vertical stacks the tabs in a column beside the panel.
	Vertical bool
}

type tabsMatch struct{ selected *int }

var (
	tabSlide = 240 * time.Millisecond
	tabPress = ui.Animation{Duration: 120 * time.Millisecond, Ease: motion.EaseOut}
)

// Tabs shows p.Items as tabs of which *selected is the chosen one, with an indicator that slides to
// the tab chosen, and the chosen tab's panel below them, or beside them when Vertical. panel builds
// the content of tab i; nil shows the tabs alone.
func Tabs(c *ui.Context, selected *int, p TabsProps, panel func(i int)) ui.Element {
	tk := theme.TokensOf(c)
	height, padding, fontSize := float32(32), float32(12), float32(13)
	switch p.Size {
	case theme.Small:
		height, padding, fontSize = 26, 10, 12
	case theme.Large:
		height, padding, fontSize = 36, 16, 14
	}

	var root ui.Element
	if p.Vertical {
		root = ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Start)
	} else {
		root = ui.Column(c).AlignSelf(ui.Stretch)
	}
	slide := ui.ElementTransition{Duration: tabSlide, Ease: motion.EaseOut, Match: tabsMatch{selected}}
	root.Children(func() {
		tabs := ui.TabsBase(c, selected, len(p.Items))
		list := tabs.List.Gap(2)
		if p.Vertical {
			list.Column().AlignItems(ui.Stretch)
		} else {
			list.Row().AlignItems(ui.Center).AlignSelf(ui.Start)
		}
		switch p.Variant {
		case TabsRounded:
			list.Padding(3).Gap(4).Radius(tk.BorderRadiusLG).Background(tk.BgLayout)
			if p.Vertical {
				list.AlignSelf(ui.Stretch)
			}
		case TabsSquare:
			list.Gap(16)
			if p.Vertical {
				list.InsetShadow(-1, 0, 0, 0, tk.BorderSecondary)
			} else {
				list.InsetShadow(0, -1, 0, 0, tk.BorderSecondary)
			}
		}
		list.Children(func() {
			for i, it := range p.Items {
				tab := tabs.Tab(i).Disabled(it.Disabled).Gap(6).AlignItems(ui.Center).Justify(ui.Center).
					FontSize(fontSize).FontWeight(500).Shrink(0).Transition(ui.ElementTransition{Duration: 120 * time.Millisecond, Ease: motion.EaseOut, Colors: true})
				switch p.Variant {
				case TabsSquare:
					tab.PaddingY(8)
				case TabsPoint:
					tab.Padding(8, padding, 14, padding).Radius(tk.BorderRadius)
				default:
					tab.Height(height).PaddingX(padding).Radius(tk.BorderRadius)
				}
				fg := tk.TextSecondary
				switch {
				case it.Disabled:
					fg = tk.TextDisabled
					tab.Cursor(ui.CursorNotAllowed)
				case tab.Hovered():
					fg = tk.Text
					tab.Cursor(ui.CursorPointer)
				case i == *selected:
					fg = tk.Primary
					tab.Cursor(ui.CursorPointer)
				default:
					tab.Cursor(ui.CursorPointer)
				}
				pose := ui.Pose{}
				if tab.Pressed() && !it.Disabled {
					pose.Scale = 0.98
				}
				tab.TextColor(fg).AnimatePose(pose, tabPress)
				tab.Children(func() {
					if i == *selected {
						tabIndicator(c, tk, p.Variant, p.Vertical, slide)
					}
					if it.Icon != nil {
						ui.Icon(c, it.Icon).FontSize(fontSize + 2)
					}
					ui.Text(c, it.Label).SingleLine()
				})
			}
		})
		if panel != nil && *selected >= 0 && *selected < len(p.Items) {
			body := ui.Column(c.Key(*selected)).Grow(1).MinWidth(0)
			if p.Vertical {
				body.Padding(0, 0, 0, 12)
			} else {
				body.Padding(12, 0, 0, 0)
			}
			body.Children(func() { panel(*selected) })
		}
	})
	return root
}

func tabIndicator(c *ui.Context, tk *theme.Tokens, v TabsVariant, vertical bool, slide ui.ElementTransition) {
	ind := ui.Box(c).Absolute().ZIndex(-1).IgnoreInput(true).Transition(slide)
	switch v {
	case TabsSquare:
		ind.Background(tk.Primary)
		if vertical {
			ind.Top(0).Bottom(0).Right(0).Width(2)
		} else {
			ind.Left(0).Right(0).Bottom(0).Height(2)
		}
	case TabsPoint:
		ind.Size(5, 5).Radius(2.5).Background(tk.Primary).Bottom(6).LeftPercent(50).Margin(0, 0, 0, -2.5)
	default:
		ind.Top(0).Left(0).Right(0).Bottom(0).Radius(tk.BorderRadius).Background(tk.BgElevated)
		tk.Elevate(ind, theme.BoxShadowTertiary)
	}
}
