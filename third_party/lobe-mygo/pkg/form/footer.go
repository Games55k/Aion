package form

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type FooterTexts struct {
	// Submit, Reset and Unsaved replace "Submit", "Reset" and "Unsaved changes".
	Submit, Reset, Unsaved string
}

type SubmitFooterProps struct {
	// Float shows the footer as a pill floating at the bottom of the window while the form has
	// unsaved changes, as in a settings page.
	Float bool
	// NoReset hides the Reset button.
	NoReset bool
	// OnReset runs after Reset puts the values back.
	OnReset func()
	Texts   FooterTexts
}

var floatFade = ui.Animation{Duration: 100 * time.Millisecond, Ease: motion.EaseOut}

// SubmitFooter shows the form's Submit button and, while it has unsaved changes, a notice and a
// Reset button.
func SubmitFooter(c *ui.Context, f Host, p SubmitFooterProps) ui.Element {
	tk := theme.TokensOf(c)
	h := f.formHost()
	texts := p.Texts
	if texts.Submit == "" {
		texts.Submit = "Submit"
	}
	if texts.Reset == "" {
		texts.Reset = "Reset"
	}
	if texts.Unsaved == "" {
		texts.Unsaved = "Unsaved changes"
	}
	dirty := h.dirty != nil && h.dirty()
	content := func(round bool) {
		if p.Float || dirty {
			ui.Icon(c, lucide.Info()).FontSize(12).TextColor(tk.TextDescription).Margin(0, 0, 0, 8)
			unsaved := ui.Text(c, texts.Unsaved).FontSize(12).TextColor(tk.TextDescription).Shrink(0)
			if p.Float {
				unsaved.Margin(0, 16, 0, 0)
			} else {
				unsaved.Margin(0, 4, 0, 0)
			}
		}
		if !p.NoReset && (p.Float || dirty) {
			b := action.Button(c, texts.Reset, action.ButtonProps{Variant: action.Filled})
			if round {
				b.Radius(999)
			}
			if b.Clicked() && h.reset != nil {
				h.reset()
				if p.OnReset != nil {
					p.OnReset()
				}
			}
		}
		b := action.Button(c, texts.Submit, action.ButtonProps{Variant: action.Primary})
		if round {
			b.Radius(999)
		}
		if b.Clicked() && h.submit != nil {
			h.submit()
		}
	}
	if !p.Float {
		return ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() { content(false) })
	}
	host := ui.Box(c).Size(0, 0)
	shown := host.AnimateTo("float", b2f(dirty), floatFade)
	if shown > 0 {
		ui.Overlay(c, func() {
			ui.Row(c).Absolute().Left(0).Right(0).Bottom(24).Justify(ui.Center).IgnoreInput(shown < 1 && !dirty).Children(func() {
				pill := ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8).Radius(48).Border(1, tk.BorderSecondary).
					Background(tk.BgContainer).Opacity(shown).Translate(0, 20*(1-shown))
				tk.Elevate(pill, theme.BoxShadowSecondary)
				pill.Children(func() { content(true) })
			})
		})
	}
	return host
}

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
