package frontend

import (
	"time"

	"github.com/egoist/mygo/ui"
)

type overlayPlacement int

const (
	placementTop overlayPlacement = iota
	placementBottom
	placementLeft
	placementRight
)

func (p overlayPlacement) side() motionSide {
	switch p {
	case placementBottom:
		return sideBottom
	case placementLeft:
		return sideLeft
	case placementRight:
		return sideRight
	}
	return sideTop
}

func attachOverlay(e, anchor ui.Element, p overlayPlacement, gap float32) {
	switch p {
	case placementBottom:
		e.AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).Margin(gap, 0, 0, 0)
	case placementLeft:
		e.AttachTo(anchor, ui.AnchorLeft, ui.AnchorRight).Margin(0, gap, 0, 0)
	case placementRight:
		e.AttachTo(anchor, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, gap)
	default:
		e.AttachTo(anchor, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, gap, 0)
	}
	e.AttachedOrigin()
}

type popoverProps struct {
	Placement overlayPlacement
}

func uiPopover(c *ui.Context, anchor ui.Element, open *bool, p popoverProps, content func()) ui.Element {
	if anchor.Clicked() {
		*open = !*open
	}
	tk := tokensOf(c)
	return ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
		attachOverlay(panel, anchor, p.Placement, 6)
		panel.Padding(12).Radius(tk.BorderRadius).MinWidth(120).
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Transition(towardSide(popupTransition(), p.Placement.side()))
		tk.Elevate(panel, boxShadowSecondary)
		panel.Children(content)
	})
}

var tipContentTransition = ui.ElementTransition{Duration: 120 * time.Millisecond, Ease: easeOut, Enter: &ui.Motion{}}

func tipTransition(p overlayPlacement) ui.ElementTransition {
	tr := towardSide(tooltipTransition(), p.side())
	tr.Colors = true
	return tr
}

type tooltipProps struct {
	Placement overlayPlacement
	Hotkey    string
}

func uiTooltip(c *ui.Context, anchor ui.Element, text string, p tooltipProps) ui.Element {
	if text == "" {
		return ui.Element{}
	}
	anchor.Description(text)
	tk := tokensOf(c)
	return ui.TooltipBase(c, anchor, func(tip ui.Element) {
		attachOverlay(tip, anchor, p.Placement, 6)
		tip.Padding(4, 8).Radius(tk.BorderRadiusSM).MaxWidth(320).
			Background(tk.BgElevated).TextColor(tk.TextLabel).FontSize(tk.FontSizeSM).
			Transition(tipTransition(p.Placement))
		tk.Elevate(tip, boxShadowTertiary)
		tip.Children(func() {
			ui.Row(c.Key(text)).Gap(6).Transition(tipContentTransition).Children(func() {
				ui.Text(c, text)
				if p.Hotkey != "" {
					uiHotkey(c, p.Hotkey, hotkeyProps{Compact: true})
				}
			})
		})
	})
}

type menuItem struct {
	Key      string
	Label    string
	Icon     *ui.SVG
	Hotkey   string
	Extra    string
	Danger   bool
	Disabled bool
	Divider  bool
}

type dropdownMenuProps struct {
	Items []menuItem
}

var menuItemTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: easeOut, Colors: true}

func uiDropdownMenu(c *ui.Context, open *bool, p dropdownMenuProps, trigger func(t ui.Element)) string {
	tk := tokensOf(c)
	m := ui.MenuBase(c, open)
	trigger(m.Trigger)
	chosen := ""
	m.Popup(func(panel ui.Element) {
		panel.Margin(6, 0, 0, 0).Padding(4).Radius(tk.BorderRadius).MinWidth(220).
			Background(tk.BgElevated).Transition(popupTransition()).AttachedOrigin()
		tk.Elevate(panel, boxShadowSecondary)
		panel.Children(func() { menuRows(c, tk, m, p.Items, &chosen) })
	})
	return chosen
}

func renderMenuItem(c *ui.Context, tk *uiTokens, m ui.MenuParts, it menuItem) ui.Element {
	item := m.Item().Disabled(it.Disabled).
		PaddingY(6).PaddingX(12).MinHeight(32).Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).
		FontSize(14).Transition(menuItemTransition)
	fg, icon := tk.Text, tk.TextSecondary
	if it.Danger {
		fg, icon = tk.Error, tk.Error
	}
	switch {
	case it.Disabled:
		fg, icon = tk.TextDisabled, tk.TextDisabled
		item.Opacity(0.5)
	case item.Pressed():
		item.Background(tk.FillSecondary)
	case item.Highlighted() && it.Danger:
		item.Background(tk.ErrorBg)
	case item.Highlighted():
		item.Background(tk.FillTertiary)
	}
	item.TextColor(fg)
	item.Children(func() {
		ui.Row(c).Grow(1).Children(func() {
			if it.Icon != nil {
				ui.Icon(c, it.Icon).FontSize(14).TextColor(icon).Margin(0, 12, 0, 0)
			}
			ui.Text(c, it.Label).Grow(1).SingleLine()
			if it.Extra != "" {
				ui.Text(c, it.Extra).Font("monospace").FontSize(12).TextColor(tk.TextTertiary).Padding(0, 0, 0, 16).SingleLine()
			}
			if it.Hotkey != "" {
				uiHotkey(c, it.Hotkey, hotkeyProps{Compact: true}).Margin(-2, -7, -2, 16)
			}
		})
	})
	return item
}

func uiContextMenu(c *ui.Context, target ui.Element, p dropdownMenuProps) string {
	tk := tokensOf(c)
	open := ui.Local(target, "contextMenu", func() bool { return false })
	m := ui.ContextMenuBase(c, target, open)
	chosen := ""
	m.Popup(func(panel ui.Element) {
		panel.Padding(4).Radius(tk.BorderRadius).MinWidth(220).
			Background(tk.BgElevated).Transition(popupTransition()).AttachedOrigin()
		tk.Elevate(panel, boxShadowSecondary)
		panel.Children(func() { menuRows(c, tk, m, p.Items, &chosen) })
	})
	return chosen
}

func menuRows(c *ui.Context, tk *uiTokens, m ui.MenuParts, items []menuItem, chosen *string) {
	for i, it := range items {
		if it.Divider {
			ui.Box(c.Key(i)).Height(1).Margin(4, 0).AlignSelf(ui.Stretch).Background(tk.Border)
			continue
		}
		if renderMenuItem(c, tk, m, it).Clicked() && !it.Disabled {
			*chosen = it.Key
		}
	}
}
