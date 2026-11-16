package frontend

import (
	"time"

	"github.com/egoist/mygo/ui"
)

type buttonVariant int

const (
	buttonDefault buttonVariant = iota
	buttonPrimary
	buttonText
)

type buttonProps struct {
	Variant buttonVariant
	Size    uiSize
	Danger  bool
}

type buttonMetrics struct {
	height, paddingX, fontSize float32
}

var buttonTransition = ui.ElementTransition{
	Duration: 160 * time.Millisecond,
	Ease:     ui.CubicBezier(0.32, 0.72, 0, 1),
	Colors:   true,
}

func uiButton(c *ui.Context, label string, p buttonProps) ui.Element {
	return styleButton(c, ui.ButtonBase(c), label, p)
}

func styleButton(c *ui.Context, b ui.Element, label string, p buttonProps) ui.Element {
	tk := tokensOf(c)
	m, radius := buttonSize(tk, p.Size)

	b.Height(m.height).PaddingX(m.paddingX).Gap(6).Radius(radius).
		FontSize(m.fontSize).FontWeight(500).
		Cursor(ui.CursorPointer).Transition(buttonTransition)

	bg, border, fg := buttonColors(tk, p.Variant, stateOfButton(b))
	if p.Danger {
		bg, border, fg = dangerButtonColors(tk, p.Variant, stateOfButton(b))
	}
	b.Background(bg).Border(1, border).TextColor(fg)

	if label != "" {
		b.Children(func() { ui.Text(c, label).SingleLine() })
	}
	return b
}

func buttonSize(tk *uiTokens, s uiSize) (buttonMetrics, float32) {
	if s == sizeSmall {
		return buttonMetrics{24, 8, 12}, tk.BorderRadiusSM
	}
	return buttonMetrics{32, 14, 13}, tk.BorderRadiusSM
}

type buttonState int

const (
	buttonRest buttonState = iota
	buttonHover
	buttonActive
)

func stateOfButton(e ui.Element) buttonState {
	switch {
	case e.Pressed():
		return buttonActive
	case e.Hovered():
		return buttonHover
	}
	return buttonRest
}

func buttonColors(tk *uiTokens, v buttonVariant, s buttonState) (bg, border, fg ui.Color) {
	pick := func(rest, hover, active ui.Color) ui.Color {
		switch s {
		case buttonActive:
			return active
		case buttonHover:
			return hover
		}
		return rest
	}
	switch v {
	case buttonPrimary:
		c := pick(tk.Primary, tk.PrimaryHover, tk.PrimaryActive)
		return c, c, tk.BgLayout
	case buttonText:
		return pick(ui.Transparent, tk.FillSecondary, tk.FillSecondary), ui.Transparent, tk.Text
	default:
		bg = pick(tk.BgContainer, layerColor(tk.BgContainer, tk.FillTertiary), layerColor(tk.BgContainer, tk.FillSecondary))
		border = tk.Border
		if s != buttonRest {
			border = mixColor(tk.Border, tk.Text, 0.08)
		}
		return bg, border, tk.Text
	}
}

func dangerButtonColors(tk *uiTokens, v buttonVariant, s buttonState) (bg, border, fg ui.Color) {
	pick := func(rest, hover, active ui.Color) ui.Color {
		switch s {
		case buttonActive:
			return active
		case buttonHover:
			return hover
		}
		return rest
	}
	fg = pick(tk.ErrorText, tk.ErrorTextHover, tk.ErrorTextActive)
	switch v {
	case buttonPrimary:
		c := pick(tk.Error, tk.ErrorHover, tk.ErrorActive)
		return c, c, tk.BgLayout
	case buttonText:
		return pick(ui.Transparent, tk.ErrorBgHover, tk.ErrorBgActive), ui.Transparent, fg
	default:
		return pick(tk.BgContainer, tk.ErrorBgHover, tk.ErrorBgActive),
			pick(tk.ErrorBorder, tk.ErrorBorderHover, tk.Error),
			fg
	}
}

func layerColor(base, top ui.Color) ui.Color {
	return base.Mix(ui.RGB(top.R, top.G, top.B), float32(top.A)/255)
}

type actionIconProps struct {
	Active bool
	Danger bool
	Title  string
}

func uiActionIcon(c *ui.Context, svg *ui.SVG, p actionIconProps) ui.Element {
	return styleActionIcon(c, ui.ButtonBase(c), svg, p)
}

func styleActionIcon(c *ui.Context, b ui.Element, svg *ui.SVG, p actionIconProps) ui.Element {
	tk := tokensOf(c)
	const (
		box    float32 = 24
		radius float32 = 4
		glyph  float32 = 14
	)

	b.Size(box, box).Radius(radius).
		Cursor(ui.CursorPointer).Transition(buttonTransition)
	if p.Title != "" {
		b.Label(p.Title)
		uiTooltip(c, b, p.Title)
	}

	state := stateOfButton(b)
	bg, border, _ := buttonColors(tk, buttonText, state)
	if p.Active {
		bg = tk.FillSecondary
	}
	b.Background(bg).Border(1, border).TextColor(actionIconGlyphColor(tk, p.Danger, state))

	b.Children(func() { ui.Icon(c, svg).FontSize(glyph) })
	return b
}

func actionIconGlyphColor(tk *uiTokens, danger bool, s buttonState) ui.Color {
	if danger {
		switch s {
		case buttonActive:
			return tk.ErrorActive
		case buttonHover:
			return tk.Error
		}
		return tk.Error.Alpha(0.66)
	}
	switch s {
	case buttonActive:
		return tk.Text
	case buttonHover:
		return tk.TextSecondary
	}
	return tk.TextTertiary
}

const copiedFor = 2 * time.Second

func uiCopyButton(c *ui.Context, content string) ui.Element {
	b := ui.ButtonBase(c)
	if b.Clicked() {
		c.WriteClipboard(content)
		b.Jump("copied", 1)
	}
	copied := b.AnimateTo("copied", 0, ui.Animation{Duration: copiedFor}) > 0
	icon, title := iconCopy(), "Copy"
	if copied {
		icon, title = iconCheck(), "Copied"
	}
	return styleActionIcon(c, b, icon, actionIconProps{Title: title, Active: copied})
}

type segmentedProps struct {
	Size uiSize
}

type segmentedMatch struct{ selected *int }

var segmentPress = ui.Animation{Duration: 120 * time.Millisecond, Ease: easeOut}

func uiSegmented(c *ui.Context, selected *int, options []string, p segmentedProps) ui.Element {
	tk := tokensOf(c)
	height, padding, fontSize := float32(32), float32(12), float32(13)
	if p.Size == sizeSmall {
		height, padding, fontSize = 26, 10, 12
	}

	parts := ui.SegmentedBase(c, selected, len(options))
	track := parts.Track.Padding(3).Gap(4).Radius(tk.BorderRadiusLG).
		Background(tk.BgLayout).Border(1, tk.FillQuaternary)

	slide := ui.ElementTransition{Duration: 240 * time.Millisecond, Ease: easeOut, Match: segmentedMatch{selected}}

	track.Children(func() {
		for i, option := range options {
			seg := parts.Segment(i).Height(height).PaddingX(padding).Radius(tk.BorderRadius).
				FontSize(fontSize).FontWeight(500).Cursor(ui.CursorPointer)
			text := tk.TextSecondary
			if i == *selected || seg.Hovered() {
				text = tk.Text
			}
			pose := ui.Pose{}
			if seg.Pressed() {
				pose.Scale = 0.98
			}
			seg.TextColor(text).AnimatePose(pose, segmentPress)
			seg.Children(func() {
				if i == *selected {
					ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
						Radius(tk.BorderRadius).Background(tk.BgElevated).ZIndex(-1).Transition(slide)
				}
				ui.Text(c, option).SingleLine()
			})
		}
	})
	return track
}

var (
	checkGlyph         = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`))
	checkboxTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: ui.CubicBezier(0.215, 0.61, 0.355, 1), Colors: true}
)

func uiCheckbox(c *ui.Context, checked *bool, label string) ui.Element {
	tk := tokensOf(c)
	const size float32 = 16

	state := ui.CheckOff
	if *checked {
		state = ui.CheckOn
	}
	row := ui.CheckboxStateBase(c, &state).Gap(8).Cursor(ui.CursorPointer)
	if row.Changed() {
		*checked = state == ui.CheckOn
	}

	filled := state != ui.CheckOff
	border, bg := tk.BorderSecondary, tk.BgContainer
	switch {
	case filled:
		border, bg = tk.Primary, tk.Primary
	case row.Hovered():
		border = tk.Border
	}

	row.Children(func() {
		box := ui.Box(c).Size(size, size).Radius(4).Border(1, border).Background(bg).
			TextColor(tk.BgLayout).Center().Transition(checkboxTransition)
		box.Children(func() {
			if state == ui.CheckOn {
				ui.Icon(c, checkGlyph).FontSize(size * 0.75)
			}
		})
		if label != "" {
			ui.Text(c, label).SingleLine()
		}
	})
	return row
}

var switchTransition = ui.ElementTransition{Duration: 200 * time.Millisecond, Ease: easeOut, Colors: true, Shadows: true}

func uiSwitch(c *ui.Context, on *bool) ui.Element {
	tk := tokensOf(c)
	const (
		w               float32 = 36
		h               float32 = 22
		thumbSize       float32 = 18
		checkedX        float32 = 14
		pressedCheckedX float32 = 10
		pressedWidth    float32 = 22
	)

	track := ui.SwitchBase(c, on).Size(w, h).Padding(2).Radius(100).
		Cursor(ui.CursorPointer).Transition(switchTransition)

	hovered, pressed := track.Hovered() && !track.IsDisabled(), track.Pressed()
	bg := tk.FillSecondary
	switch {
	case *on && hovered:
		bg = tk.PrimaryHover
	case *on:
		bg = tk.Primary
	case hovered:
		bg = tk.Fill
	}
	track.Background(bg)
	if *on {
		track.InsetShadow(0, 1.5, 3, 0, ui.RGBA(0, 0, 0, 0.18))
	} else {
		track.InsetShadow(0, 1.5, 2, 0, ui.RGBA(0, 0, 0, 0.08))
	}

	targetX, targetW := float32(0), thumbSize
	if pressed {
		targetW = pressedWidth
	}
	if *on {
		targetX = checkedX
		if pressed {
			targetX = pressedCheckedX
		}
	}

	track.Children(func() {
		thumb := ui.Box(c).Height(thumbSize).Radius(thumbSize/2).Background(tk.BgContainer).
			Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04)).
			Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.06)).
			Shadow(0, 3, 8, 0, ui.RGBA(0, 30, 80, 0.16))
		thumb.Width(thumb.AnimateTo("w", targetW, switchThumbAnim))
		thumb.Translate(thumb.AnimateTo("x", targetX, switchThumbAnim), 0)
	})
	return track
}
