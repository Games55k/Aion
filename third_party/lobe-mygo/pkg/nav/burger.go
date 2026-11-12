package nav

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type BurgerProps struct {
	// Items are the menu's rows, as List's: Key, Label, Icon, Disabled and Divider.
	Items []display.ListItem
	// Active is the Key of the page shown, its row filled.
	Active string
	// OnSelect gets the Key of the row clicked; the menu closes after.
	OnSelect func(key string)
	// Fullscreen covers the whole window with a header of its own and large rows without
	// icons, as a mobile site's menu, instead of a drawer under the app's header.
	Fullscreen bool
	// HeaderHeight is the height of the app's header the drawer opens under, or of the
	// fullscreen menu's own: 64 DIPs when zero.
	HeaderHeight float32
	// Footer builds actions at the bottom of the fullscreen menu, as Sign in.
	Footer func()
	// Size and Variant style the toggle, as ActionIcon's.
	Size    theme.Size
	Variant action.ActionIconVariant
}

// Burger is a mobile header's menu button: it opens a drawer of rows from the top while *opened
// is true and turns into a close button.
func Burger(c *ui.Context, opened *bool, p BurgerProps) ui.Element {
	tk := theme.TokensOf(c)
	header := p.HeaderHeight
	if header <= 0 {
		header = 64
	}
	icon, label := lucide.Menu(), "Open menu"
	if *opened {
		icon, label = lucide.X(), "Close menu"
	}
	toggle := action.ActionIcon(c, icon, action.ActionIconProps{Size: p.Size, Variant: p.Variant}).Label(label)
	if toggle.Clicked() {
		*opened = !*opened
	}
	burgerPanel(c, tk, toggle, opened, header, p)
	return toggle
}

func burgerRow(c *ui.Context, tk *theme.Tokens, it display.ListItem, current, large bool) ui.Element {
	b := ui.ButtonBase(c).Label(it.Label).Row().AlignItems(ui.Center).Justify(ui.Start).AlignSelf(ui.Stretch).Disabled(it.Disabled).Cursor(ui.CursorPointer)
	if it.Disabled {
		b.Opacity(0.5).Cursor(ui.CursorNotAllowed)
	}
	hovered := b.Hovered() && !it.Disabled
	if large {
		fg := tk.TextQuaternary
		if current || hovered {
			fg = tk.Text
		}
		b.MinHeight(48).PaddingX(8).Radius(12).FontSize(32).FontWeight(600).LineHeight(1.15).TextColor(fg)
		if hovered && !current {
			b.Background(tk.FillTertiary)
		}
		return b.Children(func() { ui.Text(c, it.Label) })
	}
	b.Height(52).Gap(14).PaddingX(16).Radius(999).FontSize(17).TextColor(tk.Text)
	switch {
	case current:
		b.Background(tk.Text).TextColor(tk.BgContainer).FontWeight(600)
	case hovered:
		b.Background(tk.FillTertiary)
	}
	return b.Children(func() {
		if it.Icon != nil {
			ui.Icon(c, it.Icon).FontSize(20)
		}
		ui.Text(c, it.Label).SingleLine()
	})
}

func burgerPanel(c *ui.Context, tk *theme.Tokens, anchor ui.Element, opened *bool, header float32, p BurgerProps) {
	slide, fade := motion.Drawer(motion.Top), motion.Backdrop()
	target, anim := float32(0), ui.Animation{Duration: slide.ExitDuration, Ease: slide.ExitEase}
	if *opened {
		target, anim = 1, ui.Animation{Duration: slide.Duration, Ease: slide.Ease}
	}
	shown := anchor.AnimateTo("burger", target, anim)
	dim := anchor.AnimateTo("burger-dim", target, ui.Animation{Duration: fade.Duration, Ease: fade.Ease})
	if !*opened && shown <= 0 && dim <= 0 {
		return
	}
	top := header
	if p.Fullscreen {
		top = 0
	}
	ui.Overlay(c, func() {
		ui.Box(c).Absolute().Left(0).Right(0).Top(top).Bottom(0).Background(tk.BgMask.Alpha(dim))
		frame := ui.Box(c).Absolute().Left(0).Right(0).Top(top).Bottom(0).Clip()
		frame.Children(func() {
			panel := ui.Column(c).Fill().Background(tk.BgContainer).TextColor(tk.Text).TranslatePercent(0, -100*(1-shown))
			if panel.OverlayShortcut(0, ui.KeyEscape) {
				*opened = false
			}
			if !p.Fullscreen {
				panel.Padding(8, 12)
			}
			panel.Children(func() { burgerContent(c, tk, opened, header, p) })
		})
	})
}

func burgerContent(c *ui.Context, tk *theme.Tokens, opened *bool, header float32, p BurgerProps) {
	if p.Fullscreen {
		ui.Row(c).Height(header).PaddingX(12).AlignItems(ui.Center).Justify(ui.End).Shrink(0).Children(func() {
			if action.ActionIcon(c, lucide.X(), action.ActionIconProps{Size: p.Size}).Label("Close menu").Clicked() {
				*opened = false
			}
		})
	}
	for i, it := range p.Items {
		if it.Divider {
			if !p.Fullscreen {
				ui.Box(c.Key(i)).Height(1).Margin(4, 16).AlignSelf(ui.Stretch).Background(tk.Split)
			}
			continue
		}
		if burgerRow(c.Key(i), tk, it, it.Key == p.Active, p.Fullscreen).Clicked() {
			if p.OnSelect != nil {
				p.OnSelect(it.Key)
			}
			*opened = false
		}
	}
	if p.Fullscreen && p.Footer != nil {
		ui.Spacer(c)
		ui.Row(c).Padding(20).Gap(8).Children(p.Footer)
	}
}
