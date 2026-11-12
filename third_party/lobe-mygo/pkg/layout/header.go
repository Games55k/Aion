package layout

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type HeaderProps struct {
	// Logo builds the brand at the start, or in the middle of a mobile header.
	Logo func()
	// Nav builds the links after the logo, or the menu button first in a mobile header.
	Nav func()
	// Actions builds the buttons at the end, 8 DIPs apart.
	Actions func()
	// Mobile puts Nav first and lets Logo take the room left, with 12 DIPs of padding
	// instead of 24.
	Mobile bool
}

// Header is a site's 64 DIP top bar: logo, links and actions over a line.
func Header(c *ui.Context, p HeaderProps) ui.Element {
	tk := theme.TokensOf(c)
	pad := float32(24)
	if p.Mobile {
		pad = 12
	}
	root := ui.Row(c).Height(64).PaddingX(pad).AlignItems(ui.Center).Justify(ui.SpaceBetween).
		AlignSelf(ui.Stretch).Shrink(0).Background(tk.BgLayout.Alpha(0.4)).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
	slot := func(fn func()) {
		if fn != nil {
			fn()
		}
	}
	return root.Children(func() {
		if p.Mobile {
			ui.Row(c).AlignItems(ui.Center).Shrink(0).Children(func() { slot(p.Nav) })
			ui.Row(c).AlignItems(ui.Center).Grow(1).Basis(0).Clip().Children(func() { slot(p.Logo) })
			ui.Row(c).AlignItems(ui.Center).Shrink(0).Children(func() { slot(p.Actions) })
			return
		}
		ui.Row(c).AlignItems(ui.Center).Shrink(0).Children(func() { slot(p.Logo) })
		ui.Row(c).AlignItems(ui.Center).Grow(1).Basis(0).MarginStart(48).Clip().Children(func() { slot(p.Nav) })
		ui.Row(c).AlignItems(ui.Center).Grow(1).Basis(0).Justify(ui.End).Gap(8).Children(func() { slot(p.Actions) })
	})
}

type SideNavProps struct {
	// Avatar builds the account at the top.
	Avatar func()
	// TopActions builds the buttons under the avatar, 8 DIPs apart.
	TopActions func()
	// BottomActions builds the buttons at the bottom, 4 DIPs apart.
	BottomActions func()
}

// SideNav is an app's 58 DIP rail of icon buttons along the left edge, at least 640 DIPs tall.
func SideNav(c *ui.Context, p SideNavProps) ui.Element {
	tk := theme.TokensOf(c)
	return ui.Column(c).Width(58).MinHeight(640).AlignSelf(ui.Stretch).Shrink(0).PaddingY(12).
		AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(tk.BgContainer).
		BorderWidth(0, 1, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(12).Children(func() {
			if p.Avatar != nil {
				p.Avatar()
			}
			ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
				if p.TopActions != nil {
					p.TopActions()
				}
			})
		})
		ui.Column(c).AlignItems(ui.Center).Gap(4).Children(func() {
			if p.BottomActions != nil {
				p.BottomActions()
			}
		})
	})
}
