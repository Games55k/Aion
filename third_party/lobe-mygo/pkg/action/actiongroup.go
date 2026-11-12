package action

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ActionGroupItem struct {
	// Key is what ActionIconGroup returns in the frame the icon is clicked.
	Key string
	// Icon is the button's glyph; Label names it and shows as its tooltip.
	Icon  *ui.SVG
	Label string
	// Danger colors the glyph with the error color.
	Danger bool
	// Disabled keeps the icon from being clicked.
	Disabled bool
}

type ActionIconGroupVariant int

const (
	GroupFilled ActionIconGroupVariant = iota
	GroupOutlined
	GroupBorderless
)

type ActionIconGroupProps struct {
	// Items are the icons, in order.
	Items []ActionGroupItem
	// Menu, if set, adds a More icon at the end that opens a menu of these items.
	Menu []overlay.MenuItem
	// Variant picks the group's surface: Filled tints it, Outlined borders it on the container
	// color, Borderless shows a fill only under the pointer.
	Variant ActionIconGroupVariant
	// Size sets the icons' size, as ActionIcon's.
	Size theme.Size
	// Vertical stacks the icons in a column.
	Vertical bool
	// Disabled dims the group and keeps its icons and menu from being used.
	Disabled bool
	// Shadow lifts the group with a soft layered shadow in the border colors.
	Shadow bool
}

// ActionIconGroup shows a row of icon buttons on one surface, as the actions under a message, with
// an optional More menu; it returns the Key of the icon or menu item clicked in the frame, or "".
func ActionIconGroup(c *ui.Context, p ActionIconGroupProps) string {
	tk := theme.TokensOf(c)
	root := ui.Row(c).Padding(2).Radius(tk.BorderRadius).AlignItems(ui.Center).AlignSelf(ui.Start).
		Transition(buttonTransition)
	if p.Vertical {
		root.Column()
	}
	switch p.Variant {
	case GroupOutlined:
		root.Border(1, tk.BorderSecondary).Background(tk.BgContainer)
	case GroupBorderless:
		if root.Hovered() {
			root.Background(tk.FillTertiary)
		}
	default:
		root.Background(tk.FillTertiary)
	}
	if p.Disabled {
		root.Opacity(0.5).Cursor(ui.CursorNotAllowed)
	}
	if p.Shadow {
		root.Shadow(0, 1, 0, -1, tk.Border).Shadow(0, 1, 2, -0.5, tk.Border).
			Shadow(0, 2, 2, -1, tk.BorderSecondary).Shadow(0, 3, 6, -4, tk.BorderSecondary)
	}
	icon := ActionIconProps{Size: p.Size}
	chosen := ""
	open := ui.Local(root, "more", func() bool { return false })
	root.Children(func() {
		for _, it := range p.Items {
			ip := icon
			ip.Danger, ip.Title = it.Danger, it.Label
			b := ActionIcon(c.Key(it.Key), it.Icon, ip).Disabled(p.Disabled || it.Disabled)
			if b.Clicked() {
				chosen = it.Key
			}
		}
		if len(p.Menu) > 0 {
			if p.Disabled {
				*open = false
			}
			mp := icon
			mp.Title, mp.Active = "More", *open
			if k := overlay.DropdownMenu(c, open, overlay.DropdownMenuProps{Items: p.Menu}, func(t ui.Element) {
				StyleActionIcon(c, t.Disabled(p.Disabled), lucide.Ellipsis(), mp)
			}); k != "" {
				chosen = k
			}
		}
	})
	return chosen
}
