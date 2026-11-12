package layout

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type GuideCardProps struct {
	// Title and Desc are what the card says, the title bold above the description.
	Title, Desc string
	// Cover is a picture across the top of the card, CoverWidth wide (the card's width when
	// zero) and CoverHeight tall (its own height when zero).
	Cover                   ui.ImageSource
	CoverWidth, CoverHeight float32
	// Variant picks the surface: Filled is a soft gradient, Outlined a border, Borderless none.
	Variant BlockVariant
	// Shadow lifts the card with a soft shadow.
	Shadow bool
	// NoClose leaves out the close button.
	NoClose bool
	// OnClose runs when the close button hides the card.
	OnClose func()
}

// GuideCard is a dismissible tip, as a feature's introduction: a cover, a title and a
// description; the close button hides it for good.
func GuideCard(c *ui.Context, p GuideCardProps) ui.Element {
	tk := theme.TokensOf(c)
	root := ui.Column(c).Radius(tk.BorderRadiusLG).Clip()
	closed := ui.Local(root, "closed", func() bool { return false })
	if *closed {
		return root.Hidden(true)
	}
	switch p.Variant {
	case BlockOutlined:
		root.Border(1, tk.BorderSecondary).Background(tk.BgContainer)
	case BlockBorderless:
	default:
		from, to := tk.FillQuaternary, tk.FillTertiary
		if tk.Dark {
			from, to = to, from
		}
		root.Gradient(from, to, 180)
	}
	if p.Shadow {
		tk.Elevate(root, theme.BoxShadowSecondary)
	}
	return root.Children(func() {
		if p.Cover != nil {
			img := ui.Image(c, p.Cover).Fit(ui.Cover).AlignSelf(ui.Center)
			if p.CoverWidth > 0 {
				img.Width(p.CoverWidth)
			} else {
				img.AlignSelf(ui.Stretch)
			}
			if p.CoverHeight > 0 {
				img.Height(p.CoverHeight)
			}
		}
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			if p.Title != "" {
				ui.Text(c, p.Title).FontSize(16).FontWeight(700)
			}
			if p.Desc != "" {
				ui.Text(c, p.Desc).TextColor(tk.TextDescription)
			}
		})
		if !p.NoClose {
			if action.ActionIcon(c, lucide.X(), action.ActionIconProps{Size: theme.Small}).Label("Close").
				Absolute().Top(8).Right(8).Clicked() {
				*closed = true
				if p.OnClose != nil {
					p.OnClose()
				}
			}
		}
	})
}
