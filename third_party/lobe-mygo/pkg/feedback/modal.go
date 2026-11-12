package feedback

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ModalProps struct {
	// Title shows in the header; without one, the content starts at the top.
	Title string
	// Width is the panel's width, 520 DIPs when zero, and at most the window's less 32.
	Width float32
	// OkText labels the primary button, "OK" when empty.
	OkText string
	// CancelText labels the other button, "Cancel" when empty.
	CancelText string
	// NoFooter hides both buttons.
	NoFooter bool
}

var dialogCloseTransition = ui.ElementTransition{Duration: 160 * time.Millisecond, Ease: motion.SoftEase, Colors: true}

// Modal shows content in a dialog over the window while *open is true. Cancel, the close button,
// Escape and a click on the backdrop close it. It returns true in the frame OK is clicked and
// leaves *open to the caller, which closes it once the action succeeds.
func Modal(c *ui.Context, open *bool, p ModalProps, content func()) bool {
	tk := theme.TokensOf(c)
	m := motion.Modal()
	ok := false
	present(c, open, motionOf(m), func(backdrop, panel ui.Element, shown, dim float32, closing bool) {
		backdrop.Padding(32, 16).Background(tk.BgContainer.Alpha(0.6 * dim))
		width := p.Width
		if width == 0 {
			width = 520
		}
		from := m.Enter.Scale
		if closing {
			from = m.Exit.Scale
		}
		scale := from + (1-from)*shown
		panel.Width(width).MaxWidthPercent(100).MaxHeightPercent(100).Radius(12).Clip().
			Background(tk.BgElevated).TextColor(tk.Text).FontSize(tk.FontSize).
			Scale(scale, scale).Opacity(shown)
		tk.Elevate(panel, theme.BoxShadow)
		panel.Children(func() {
			if p.Title != "" {
				ui.Row(c).MinHeight(48).Padding(8, 56, 8, 16).Children(func() {
					ui.Text(c, p.Title).FontSize(14).FontWeight(600)
				})
			}
			ui.Scroll(c).Shrink(1).Children(func() {
				col := ui.Column(c).Padding(0, 16, 16, 16)
				if p.Title == "" {
					col.Padding(16)
				}
				col.Children(content)
			})
			if !p.NoFooter {
				ui.Row(c).Gap(8).Padding(12, 16).Justify(ui.End).Children(func() {
					if action.Button(c, textOr(p.CancelText, "Cancel"), action.ButtonProps{}).Clicked() {
						*open = false
					}
					if action.Button(c, textOr(p.OkText, "OK"), action.ButtonProps{Variant: action.Primary}).Clicked() {
						ok = true
					}
				})
			}
			if dialogClose(c, tk, 18).Absolute().Top(8).Right(12).Radius(12).Clicked() {
				*open = false
			}
		})
	})
	return ok
}

func dialogClose(c *ui.Context, tk *theme.Tokens, glyph float32) ui.Element {
	b := ui.ButtonBase(c).Label("Close").Size(32, 32).Radius(8).Cursor(ui.CursorPointer).Transition(dialogCloseTransition)
	if b.Hovered() {
		b.Background(tk.FillSecondary).TextColor(tk.Text)
	} else {
		b.TextColor(tk.TextTertiary)
	}
	b.Children(func() { ui.Icon(c, lucide.X()).FontSize(glyph) })
	return b
}

func textOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
