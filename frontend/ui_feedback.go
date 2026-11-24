package frontend

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

type alertType int

const (
	alertInfo alertType = iota
	alertSuccess
	alertWarning
	alertError
)

type alertProps struct {
	Type        alertType
	Description string
}

var dialogCloseTransition = ui.ElementTransition{Duration: 160 * time.Millisecond, Ease: softEase, Colors: true}

func uiAlert(c *ui.Context, title string, p alertProps) ui.Element {
	tk := tokensOf(c)
	accent, icon := alertTone(tk, p.Type)
	detailed := p.Description != ""

	root := ui.Row(c).Gap(10).AlignSelf(ui.Stretch).AlignItems(ui.Start).Radius(tk.BorderRadius).
		FontSize(14).TextColor(tk.Text).Transition(collapseTransition()).
		Background(mixColor(tk.BgContainer, accent, 0.05)).InsetShadow(0, 0, 0, 1, accent.Alpha(0.14))
	if detailed {
		root.Padding(12, 14)
	} else {
		root.Padding(10, 12)
	}

	root.Children(func() {
		glyph := float32(16)
		if detailed {
			glyph = 18
		}
		ui.Row(c).Height(20).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Icon(c, icon()).FontSize(glyph).TextColor(accent)
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, title).FontWeight(500).LineHeight(20.0 / 14)
			if detailed {
				ui.Text(c, p.Description).FontSize(13).LineHeight(20.0 / 13).TextColor(tk.TextSecondary)
			}
		})
	})
	return root
}

func alertTone(tk *uiTokens, t alertType) (accent ui.Color, icon func() *ui.SVG) {
	switch t {
	case alertSuccess:
		return tk.Success, iconCircleCheck
	case alertWarning:
		return tk.Warning, iconTriangleAlert
	case alertError:
		return tk.Error, iconCircleX
	}
	return tk.Info, iconInfo
}

type dialogState struct{ shown, closing, entered bool }

type dialogMotion struct{ in, out, fade ui.Animation }

func motionOfDialog(panel ui.ElementTransition) dialogMotion {
	b := backdropTransition()
	return dialogMotion{
		in:   ui.Animation{Duration: panel.Duration, Ease: panel.Ease},
		out:  ui.Animation{Duration: panel.ExitDuration, Ease: panel.ExitEase},
		fade: ui.Animation{Duration: b.Duration, Ease: b.Ease},
	}
}

func dialogPresent(c *ui.Context, open *bool, m dialogMotion, build func(backdrop, panel ui.Element, shown, dim float32, closing bool)) {
	st := ui.Local(ui.Box(c).Absolute().Size(0, 0), "dialog", func() dialogState { return dialogState{} })
	if *open {
		st.shown, st.closing = true, false
	} else if st.shown {
		st.closing = true
	}
	if !st.shown {
		return
	}
	gone := false
	ui.DialogBase(c, &st.shown, func(backdrop, panel ui.Element) {
		if !st.entered {
			st.entered = true
			panel.Jump("shown", 0)
			backdrop.Jump("dim", 0)
		}
		target, anim := float32(1), m.in
		if st.closing {
			target, anim = 0, m.out
			backdrop.IgnoreInput(true)
		}
		shown := panel.AnimateTo("shown", target, anim)
		dim := backdrop.AnimateTo("dim", target, m.fade)
		gone = st.closing && shown <= 0 && dim <= 0
		build(backdrop, panel, shown, dim, st.closing)
	})
	switch {
	case !st.shown:
		*open, st.shown, st.closing = false, true, true
		c.Invalidate()
	case gone:
		*st = dialogState{}
		c.Invalidate()
	}
}

type modalProps struct {
	Title      string
	Width      float32
	OkText     string
	CancelText string
	OkDisabled bool
}

func uiModal(c *ui.Context, open *bool, p modalProps, content func()) bool {
	tk := tokensOf(c)
	m := modalTransition()
	ok := false
	dialogPresent(c, open, motionOfDialog(m), func(backdrop, panel ui.Element, shown, dim float32, closing bool) {
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
		tk.Elevate(panel, boxShadow)
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
			ui.Row(c).Gap(8).Padding(12, 16).Justify(ui.End).Children(func() {
				if uiButton(c, textOr(p.CancelText, "Cancel"), buttonProps{}).Clicked() {
					*open = false
				}
				if uiButton(c, textOr(p.OkText, "OK"), buttonProps{Variant: buttonPrimary}).Disabled(p.OkDisabled).Clicked() {
					ok = true
				}
			})
			if dialogCloseButton(c, tk, 18).Absolute().Top(8).Right(12).Radius(12).Clicked() {
				*open = false
			}
		})
	})
	return ok
}

func dialogCloseButton(c *ui.Context, tk *uiTokens, glyph float32) ui.Element {
	b := ui.ButtonBase(c).Label("Close").Size(32, 32).Radius(8).Cursor(ui.CursorPointer).Transition(dialogCloseTransition)
	if b.Hovered() {
		b.Background(tk.FillSecondary).TextColor(tk.Text)
	} else {
		b.TextColor(tk.TextTertiary)
	}
	b.Children(func() { ui.Icon(c, iconX()).FontSize(glyph) })
	return b
}

func textOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

type progressProps struct {
	ShowInfo bool
	Format   func(percent float32) string
}

var progressMove = ui.Animation{Duration: 300 * time.Millisecond, Ease: cssEase}

func uiProgress(c *ui.Context, percent float32, p progressProps) ui.Element {
	tk := tokensOf(c)
	pct := min(max(percent, 0), 100)
	const thick = float32(2)

	root := ui.Column(c).Gap(6).FillWidth().Role(ui.RoleProgress).Range(0, 100, float64(pct))
	shown := root.AnimateTo("pct", pct, progressMove) / 100
	root.Children(func() {
		if p.ShowInfo {
			text := strconv.FormatFloat(float64(pct), 'f', -1, 32) + "%"
			if p.Format != nil {
				text = p.Format(pct)
			}
			ui.Row(c).Justify(ui.End).FontSize(12).Font("monospace").TextColor(tk.Text).Children(func() {
				ui.Text(c, text).SingleLine().FontFeatures("tnum")
			})
		}
		track := ui.Box(c).FillWidth().Height(thick).Radius(thick / 2).Clip().Background(tk.FillTertiary)
		track.Children(func() {
			ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).WidthPercent(100 * shown).Radius(thick / 2).Background(tk.Primary)
		})
	})
	return root
}
