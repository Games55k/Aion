package frontend

import (
	"math"
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
	alertSecondary
)

type alertVariant int

const (
	alertSoft alertVariant = iota
	alertOutlined
	alertPlain
)

type alertProps struct {
	Type        alertType
	Variant     alertVariant
	Description string
	NoIcon      bool
	Closed      *bool
}

var (
	alertCloseTransition = ui.ElementTransition{Duration: 160 * time.Millisecond, Ease: easeOut, Colors: true}
	dialogCloseTransition = ui.ElementTransition{Duration: 160 * time.Millisecond, Ease: softEase, Colors: true}
)

func uiAlert(c *ui.Context, title string, p alertProps) ui.Element {
	if p.Closed != nil && *p.Closed {
		return ui.Element{}
	}
	tk := tokensOf(c)
	accent, icon, share, ring := alertTone(tk, p.Type)
	detailed := p.Description != ""

	root := ui.Row(c).Gap(10).AlignSelf(ui.Stretch).AlignItems(ui.Start).Radius(tk.BorderRadius).
		FontSize(14).TextColor(tk.Text).Transition(collapseTransition())
	switch {
	case p.Variant == alertPlain:
		root.Padding(2, 0)
	case detailed:
		root.Padding(12, 14)
	default:
		root.Padding(10, 12)
	}
	switch p.Variant {
	case alertSoft:
		root.Background(mixColor(tk.BgContainer, accent, share)).InsetShadow(0, 0, 0, 1, accent.Alpha(ring))
	case alertOutlined:
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
		if p.Closed != nil && alertCloseButton(c, tk).Clicked() {
			*p.Closed = true
		}
	})
	return root
}

func alertTone(tk *uiTokens, t alertType) (accent ui.Color, icon func() *ui.SVG, share, ring float32) {
	switch t {
	case alertSuccess:
		return tk.Success, iconCircleCheck, 0.05, 0.14
	case alertWarning:
		return tk.Warning, iconTriangleAlert, 0.05, 0.14
	case alertError:
		return tk.Error, iconCircleX, 0.05, 0.14
	case alertSecondary:
		return tk.TextSecondary, iconTriangleAlert, 0.04, 0.12
	}
	return tk.Info, iconInfo, 0.05, 0.14
}

func alertCloseButton(c *ui.Context, tk *uiTokens) ui.Element {
	b := ui.ButtonBase(c).Label("Close").Size(32, 32).Radius(tk.BorderRadiusSM).Margin(-6, -4, -6, 0).
		Shrink(0).Cursor(ui.CursorPointer).Transition(alertCloseTransition)
	if b.Hovered() {
		b.Background(tk.FillTertiary).TextColor(tk.Text)
	} else {
		b.TextColor(tk.TextTertiary)
	}
	if b.Pressed() {
		b.Scale(0.96, 0.96)
	}
	b.Children(func() { ui.Icon(c, iconX()).FontSize(16) })
	return b
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
	NoFooter   bool
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
			if !p.NoFooter {
				ui.Row(c).Gap(8).Padding(12, 16).Justify(ui.End).Children(func() {
					if uiButton(c, textOr(p.CancelText, "Cancel"), buttonProps{}).Clicked() {
						*open = false
					}
					if uiButton(c, textOr(p.OkText, "OK"), buttonProps{Variant: buttonPrimary}).Clicked() {
						ok = true
					}
				})
			}
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

type progressType int

const (
	progressLine progressType = iota
	progressCircleType
)

type progressVariant int

const (
	progressBar progressVariant = iota
	progressSegments
	progressInset
)

type progressStatus int

const (
	progressNormal progressStatus = iota
	progressActive
	progressSuccess
	progressException
)

type progressProps struct {
	Type     progressType
	Variant  progressVariant
	Status   progressStatus
	Size     uiSize
	ShowInfo bool
	Label    string
	Format   func(percent float32) string
	Segments int
	Color    ui.Color
}

var (
	progressMove  = ui.Animation{Duration: 300 * time.Millisecond, Ease: cssEase}
	progressShine = 1200 * time.Millisecond
)

func uiProgress(c *ui.Context, percent float32, p progressProps) ui.Element {
	tk := tokensOf(c)
	pct := min(max(percent, 0), 100)
	fill := p.Color
	if fill == (ui.Color{}) {
		fill = map[progressStatus]ui.Color{
			progressNormal: tk.Primary, progressActive: tk.Info,
			progressSuccess: tk.Success, progressException: tk.Error,
		}[p.Status]
	}
	info := func(size float32) {
		switch p.Status {
		case progressSuccess:
			ui.Icon(c, iconCheck()).FontSize(size).TextColor(fill)
		case progressException:
			ui.Icon(c, iconX()).FontSize(size).TextColor(fill)
		default:
			text := strconv.FormatFloat(float64(pct), 'f', -1, 32) + "%"
			if p.Format != nil {
				text = p.Format(pct)
			}
			ui.Text(c, text).SingleLine().FontFeatures("tnum")
		}
	}
	if p.Type == progressCircleType {
		return progressCircle(c, tk, pct, fill, p, info)
	}
	thick := map[uiSize]float32{sizeSmall: 2, sizeMiddle: 4, sizeLarge: 6}[p.Size]
	block := map[uiSize]float32{sizeSmall: 4, sizeMiddle: 8, sizeLarge: 12}[p.Size]

	root := ui.Column(c).Gap(6).FillWidth().Role(ui.RoleProgress).Range(0, 100, float64(pct))
	shown := root.AnimateTo("pct", pct, progressMove) / 100
	root.Children(func() {
		switch p.Variant {
		case progressSegments, progressInset:
			ui.Row(c).Gap(10).AlignItems(ui.Center).FillWidth().Children(func() {
				if p.Variant == progressSegments {
					n := p.Segments
					if n <= 0 {
						n = 20
					}
					filled := int(math.Round(float64(n) * float64(pct) / 100))
					ui.Row(c).Grow(1).MinWidth(0).Gap(2).Height(block).Children(func() {
						for i := range n {
							bg := tk.FillTertiary
							if i < filled {
								bg = fill
							}
							ui.Box(c.Key(i)).Grow(1).Basis(0).Height(block).Radius(1).Background(bg)
						}
					})
				} else {
					ui.Row(c).Grow(1).MinWidth(0).Gap(3).Height(block).Children(func() {
						if shown > 0 {
							ui.Box(c).Grow(shown).Basis(0).MinWidth(block).Height(block).Radius(block / 2).Background(fill)
						}
						if pct < 100 {
							ui.Box(c).Grow(1 - shown).Basis(0).Height(block).Radius(block / 2).Background(tk.FillSecondary)
						}
					})
				}
				if p.ShowInfo {
					ui.Row(c).Justify(ui.End).MinWidth(32).FontSize(12).Font("monospace").TextColor(tk.TextSecondary).
						Children(func() { info(14) })
				}
			})
		default:
			if p.Label != "" || p.ShowInfo {
				ui.Row(c).Gap(12).Justify(ui.SpaceBetween).FontSize(12).TextColor(tk.TextSecondary).Children(func() {
					ui.Text(c, p.Label)
					if p.ShowInfo {
						ui.Row(c).Font("monospace").TextColor(tk.Text).Children(func() { info(14) })
					}
				})
			}
			track := ui.Box(c).FillWidth().Height(thick).Radius(thick / 2).Clip().Background(tk.FillTertiary)
			track.Children(func() {
				bar := ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).WidthPercent(100 * shown).Radius(thick / 2).Background(fill)
				if p.Status == progressActive {
					t := bar.Loop("shine", progressShine, ui.Linear)
					w := track.Bounds().W
					bar.Clip().Children(func() {
						ui.Box(c).Absolute().Top(0).Bottom(0).Width(80).Left(-80 + (w+80)*t).LinearGradient(ui.LinearGradient{
							Angle: 90, Stops: []ui.GradientStop{{Color: ui.RGBA(255, 255, 255, 0), At: 0},
								{Color: ui.RGBA(255, 255, 255, 0.35), At: 0.5}, {Color: ui.RGBA(255, 255, 255, 0), At: 1}},
						})
					})
				}
			})
		}
	})
	return root
}

func progressCircle(c *ui.Context, tk *uiTokens, pct float32, fill ui.Color, p progressProps, info func(float32)) ui.Element {
	d := map[uiSize]float32{sizeSmall: 20, sizeMiddle: 40, sizeLarge: 72}[p.Size]
	stroke := map[uiSize]float32{sizeSmall: 2, sizeMiddle: 3, sizeLarge: 4}[p.Size]
	root := ui.Row(c).Size(d, d).Center().Shrink(0).AlignSelf(ui.Start).Role(ui.RoleProgress).Range(0, 100, float64(pct))
	shown := root.AnimateTo("pct", pct, progressMove) / 100
	track := tk.FillTertiary
	root.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy, rad := r.X+r.W/2, r.Y+r.H/2, d/2-stroke
		var ring ui.Path
		pt.StrokePath(ring.Circle(cx, cy, rad), stroke, track)
		if shown <= 0 {
			return
		}
		var arc ui.Path
		steps := max(2, int(96*shown))
		for i := 0; i <= steps; i++ {
			a := float64(shown) * 2 * math.Pi * float64(i) / float64(steps)
			x, y := cx+rad*float32(math.Sin(a)), cy-rad*float32(math.Cos(a))
			if i == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		pt.StrokePath(arc.Cap(ui.RoundCap), stroke, fill)
	})
	if p.ShowInfo && d >= 40 {
		font := float32(12)
		if d >= 72 {
			font = 17
		}
		root.FontSize(font).TextColor(tk.Text).Children(func() { info(font + 2) })
	}
	return root
}

// ToastType selects the severity icon and color of a toast message.
type ToastType int

const (
	ToastDefault ToastType = iota
	ToastInfo
	ToastSuccess
	ToastWarning
	ToastError
	ToastLoading
)

var toastTypeNames = []string{"", "info", "success", "warning", "error", "loading"}

// ToastProps describes an in-app toast notification.
type ToastProps struct {
	ID                 string
	Title, Description string
	Type               ToastType
	Timeout            time.Duration
	Action             string
	OnAction           func()
}

func uiShowToast(c *ui.Context, o ToastProps) string {
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return c.AddToast(ui.Toast{
		ID: o.ID, Title: o.Title, Description: o.Description, Type: toastTypeNames[o.Type],
		Timeout: timeout, Action: o.Action, OnAction: o.OnAction,
	})
}

type toastPlacement int

const (
	toastBottomRight toastPlacement = iota
	toastBottomLeft
	toastBottom
	toastTopRight
	toastTopLeft
	toastTop
)

type toastHostProps struct {
	Placement toastPlacement
}

const (
	toastWidth = 360
	toastGap   = 12
	toastPeek  = 12
)

var (
	toastStackAnim = ui.Animation{Duration: 400 * time.Millisecond, Ease: toastEase}
	toastFadeAnim  = ui.Animation{Duration: 200 * time.Millisecond, Ease: easeOut}
	toastSizeAnim  = ui.Animation{Duration: 150 * time.Millisecond, Ease: cssEase}
)

func uiToastHost(c *ui.Context, p toastHostProps) ui.Element {
	tk := tokensOf(c)
	top := p.Placement >= toastTopRight
	return ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(16)
		switch p.Placement {
		case toastBottomLeft, toastTopLeft:
			viewport.AlignItems(ui.Start)
		case toastBottom, toastTop:
			viewport.AlignItems(ui.Center)
		default:
			viewport.AlignItems(ui.End)
		}
		if top {
			viewport.Justify(ui.Start)
		} else {
			viewport.Justify(ui.End)
		}

		stack := ui.Box(c).Width(toastWidth).MaxWidthPercent(100)
		heights := *ui.Local(stack, "heights", func() map[string]float32 { return map[string]float32{} })
		expanded := stack.Hovered()
		n := len(toasts)
		front := float32(0)
		if n > 0 {
			front = heights[toasts[n-1].ID]
		}
		total := front + toastPeek*float32(max(n-1, 0))
		if expanded {
			total = 0
			for _, t := range toasts {
				total += heights[t.ID]
			}
			total += toastGap * float32(max(n-1, 0))
		}
		stack.Height(total)

		layouts := make([]toastLayout, n)
		offset := float32(0)
		for k := n - 1; k >= 0; k-- {
			depth := n - 1 - k
			scale := 1 - 0.05*float32(depth)
			l := toastLayout{
				depth: depth, y: float32(depth)*toastPeek + (1-scale)*front, scale: scale,
				collapsed: !expanded && depth > 0, front: front, top: top,
			}
			if expanded {
				l.y, l.scale = offset, 1
			}
			offset += heights[toasts[k].ID] + toastGap
			layouts[k] = l
		}
		stack.Children(func() {
			for k, t := range toasts {
				renderToast(c, tk, t, heights, layouts[k])
			}
		})
	})
}

type toastLayout struct {
	depth           int
	y, scale, front float32
	collapsed, top  bool
}

func renderToast(c *ui.Context, tk *uiTokens, t ui.Toast, heights map[string]float32, l toastLayout) {
	parts := ui.ToastBase(c, t)
	root := parts.Root.Absolute().Left(0).Right(0).Radius(tk.BorderRadiusLG).Clip().
		Background(tk.BgElevated).TextColor(tk.Text).FontSize(14)
	tk.Elevate(root, boxShadowSecondary)
	enter := toastTransition(sideBottom)
	y := -l.y
	if l.top {
		enter = toastTransition(sideTop)
		root.Top(0).TransformOrigin(0.5, 0)
		y = l.y
	} else {
		root.Bottom(0).TransformOrigin(0.5, 1)
	}
	enter.Colors = true
	root.Transition(enter)
	height := heights[t.ID]
	if l.collapsed && l.front > 0 {
		height = l.front
	}
	if height > 0 {
		root.Height(root.AnimateTo("height", height, toastSizeAnim))
	}
	root.Translate(0, root.AnimateTo("y", y, toastStackAnim))
	scale := root.AnimateTo("scale", l.scale, toastStackAnim)
	root.Scale(scale, scale)

	shown := float32(1)
	if l.collapsed {
		shown = 0
	}
	var body ui.Element
	root.Children(func() {
		body = ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Start).Shrink(0).
			Opacity(root.AnimateTo("content", shown, toastFadeAnim))
		body.Children(func() {
			icon, color := toastIcon(tk, t.Type)
			ui.Row(c).Height(21).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Icon(c, icon()).FontSize(18).TextColor(color)
			})
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
					switch {
					case t.Title != "":
						ui.Text(c, t.Title).FontWeight(500).LineHeight(1.5).Grow(1)
					default:
						ui.Text(c, t.Description).LineHeight(1.5).Grow(1)
					}
					toastClose(c, tk, parts.CloseButton())
				})
				if t.Title != "" && t.Description != "" {
					ui.Text(c, t.Description).FontSize(13).LineHeight(1.5).TextColor(tk.TextSecondary).Margin(4, 0, 0, 0)
				}
				if t.Action != "" {
					ui.Row(c).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
						toastAction(c, tk, parts.ActionButton(), t.Action)
					})
				}
			})
		})
	})
	if h := body.Bounds().H; h > 0 && h != heights[t.ID] {
		heights[t.ID] = h
		c.Invalidate()
	}
}

func toastIcon(tk *uiTokens, typ string) (func() *ui.SVG, ui.Color) {
	switch typ {
	case "info":
		return iconInfo, tk.Info
	case "success":
		return iconCircleCheck, tk.Success
	case "warning":
		return iconTriangleAlert, tk.Warning
	case "error":
		return iconCircleX, tk.Error
	case "loading":
		return iconLoaderCircle, tk.TextSecondary
	}
	return iconInfo, tk.Text
}

func toastClose(c *ui.Context, tk *uiTokens, b ui.Element) {
	b.Label("Close").Size(20, 20).Radius(tk.BorderRadiusSM).Shrink(0).Margin(1, 0, 0, 0).
		Cursor(ui.CursorPointer).Transition(dialogCloseTransition)
	if b.Hovered() {
		b.Background(tk.FillSecondary).TextColor(tk.Text)
	} else {
		b.TextColor(tk.TextSecondary)
	}
	b.Children(func() { ui.Icon(c, iconX()).FontSize(14) })
}

func toastAction(c *ui.Context, tk *uiTokens, b ui.Element, label string) {
	bg := tk.Primary
	switch {
	case b.Pressed():
		bg = tk.PrimaryActive
	case b.Hovered():
		bg = tk.PrimaryHover
	}
	b.Height(28).PaddingX(12).Radius(tk.BorderRadiusSM).Background(bg).TextColor(tk.BgLayout).
		FontSize(12).FontWeight(500).Cursor(ui.CursorPointer).Transition(dialogCloseTransition)
	b.Children(func() { ui.Text(c, label).SingleLine() })
}
