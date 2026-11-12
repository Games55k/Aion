package feedback

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type AlertType int

const (
	Info AlertType = iota
	Success
	Warning
	Error
	Secondary
)

type AlertVariant int

const (
	Soft AlertVariant = iota
	Outlined
	Plain
)

type AlertProps struct {
	// Type picks the color and icon.
	Type AlertType
	// Variant picks the surface: Soft tints it with the type's color, Outlined draws a border,
	// Plain shows no surface.
	Variant AlertVariant
	// Description shows under the title in secondary text, and makes the alert roomier.
	Description string
	// NoIcon hides the type's icon.
	NoIcon bool
	// Closed, if set, shows a close button, which sets *Closed to true; the alert shows nothing
	// while it is true.
	Closed *bool
}

var closeTransition = ui.ElementTransition{Duration: 160 * time.Millisecond, Ease: motion.EaseOut, Colors: true}

// Alert shows a message with the icon and color of its type.
func Alert(c *ui.Context, title string, p AlertProps) ui.Element {
	if p.Closed != nil && *p.Closed {
		return ui.Element{}
	}
	tk := theme.TokensOf(c)
	accent, icon, share, ring := alertTone(tk, p.Type)
	detailed := p.Description != ""

	root := ui.Row(c).Gap(10).AlignSelf(ui.Stretch).AlignItems(ui.Start).Radius(tk.BorderRadius).
		FontSize(14).TextColor(tk.Text).Transition(motion.Collapse())
	switch {
	case p.Variant == Plain:
		root.Padding(2, 0)
	case detailed:
		root.Padding(12, 14)
	default:
		root.Padding(10, 12)
	}
	switch p.Variant {
	case Soft:
		root.Background(theme.Mix(tk.BgContainer, accent, share)).InsetShadow(0, 0, 0, 1, accent.Alpha(ring))
	case Outlined:
		root.InsetShadow(0, 0, 0, 1, tk.BorderSecondary)
	}

	root.Children(func() {
		if !p.NoIcon {
			glyph := float32(16)
			if detailed {
				glyph = 18
			}
			ui.Row(c).Height(20).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Icon(c, icon()).FontSize(glyph).TextColor(accent)
			})
		}
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, title).FontWeight(500).LineHeight(20.0 / 14)
			if detailed {
				ui.Text(c, p.Description).FontSize(13).LineHeight(20.0 / 13).TextColor(tk.TextSecondary)
			}
		})
		if p.Closed != nil && closeButton(c, tk).Clicked() {
			*p.Closed = true
		}
	})
	return root
}

func alertTone(tk *theme.Tokens, t AlertType) (accent ui.Color, icon func() *ui.SVG, share, ring float32) {
	switch t {
	case Success:
		return tk.Success, lucide.CircleCheck, 0.05, 0.14
	case Warning:
		return tk.Warning, lucide.TriangleAlert, 0.05, 0.14
	case Error:
		return tk.Error, lucide.CircleX, 0.05, 0.14
	case Secondary:
		return tk.TextSecondary, lucide.TriangleAlert, 0.04, 0.12
	}
	return tk.Info, lucide.Info, 0.05, 0.14
}

func closeButton(c *ui.Context, tk *theme.Tokens) ui.Element {
	b := ui.ButtonBase(c).Label("Close").Size(32, 32).Radius(tk.BorderRadiusSM).Margin(-6, -4, -6, 0).
		Shrink(0).Cursor(ui.CursorPointer).Transition(closeTransition)
	if b.Hovered() {
		b.Background(tk.FillTertiary).TextColor(tk.Text)
	} else {
		b.TextColor(tk.TextTertiary)
	}
	if b.Pressed() {
		b.Scale(0.96, 0.96)
	}
	b.Children(func() { ui.Icon(c, lucide.X()).FontSize(16) })
	return b
}
