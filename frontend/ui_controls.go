package frontend

import (
	"time"

	"github.com/egoist/mygo/ui"
)

type buttonVariant int

const (
	buttonDefault buttonVariant = iota
	buttonPrimary
	buttonDashed
	buttonFilled
	buttonText
	buttonLink
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
	switch p.Variant {
	case buttonDashed:
		b.BorderStyle(ui.BorderDashed)
	case buttonLink:
		b.PaddingX(0)
	}

	if label != "" {
		b.Children(func() { ui.Text(c, label).SingleLine() })
	}
	return b
}

func buttonSize(tk *uiTokens, s uiSize) (buttonMetrics, float32) {
	switch s {
	case sizeSmall:
		return buttonMetrics{24, 8, 12}, tk.BorderRadiusSM
	case sizeLarge:
		return buttonMetrics{40, 16, 14}, tk.BorderRadius
	default:
		return buttonMetrics{32, 14, 13}, tk.BorderRadiusSM
	}
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
	case buttonFilled:
		return pick(tk.FillTertiary, tk.FillSecondary, tk.Fill), ui.Transparent, tk.Text
	case buttonText:
		return pick(ui.Transparent, tk.FillSecondary, tk.FillSecondary), ui.Transparent, tk.Text
	case buttonLink:
		return ui.Transparent, ui.Transparent, pick(tk.Primary, tk.PrimaryHover, tk.PrimaryHover)
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
	case buttonFilled:
		return pick(tk.ErrorFillTertiary, tk.ErrorFillSecondary, tk.ErrorFill), ui.Transparent, fg
	case buttonText:
		return pick(ui.Transparent, tk.ErrorBgHover, tk.ErrorBgActive), ui.Transparent, fg
	case buttonLink:
		return ui.Transparent, ui.Transparent, fg
	default:
		return pick(tk.BgContainer, tk.ErrorBgHover, tk.ErrorBgActive),
			pick(tk.ErrorBorder, tk.ErrorBorderHover, tk.Error),
			fg
	}
}

func layerColor(base, top ui.Color) ui.Color {
	return base.Mix(ui.RGB(top.R, top.G, top.B), float32(top.A)/255)
}

type actionIconVariant int

const (
	actionIconBorderless actionIconVariant = iota
	actionIconFilled
	actionIconOutlined
)

type actionIconProps struct {
	Variant actionIconVariant
	Size    uiSize
	Active  bool
	Danger  bool
	Title   string
}

func uiActionIcon(c *ui.Context, svg *ui.SVG, p actionIconProps) ui.Element {
	return styleActionIcon(c, ui.ButtonBase(c), svg, p)
}

func styleActionIcon(c *ui.Context, b ui.Element, svg *ui.SVG, p actionIconProps) ui.Element {
	tk := tokensOf(c)
	box, radius, glyph := actionIconSize(p.Size)

	b.Size(box, box).Radius(radius).
		Cursor(ui.CursorPointer).Transition(buttonTransition)
	if p.Title != "" {
		b.Label(p.Title)
		uiTooltip(c, b, p.Title, tooltipProps{})
	}

	state := stateOfButton(b)
	bg, border, _ := buttonColors(tk, actionIconSurface(p.Variant), state)
	if p.Active {
		bg = actionIconActiveFill(tk, p.Variant, state)
	}
	b.Background(bg).Border(1, border).TextColor(actionIconGlyphColor(tk, p.Danger, state))

	b.Children(func() { ui.Icon(c, svg).FontSize(glyph) })
	return b
}

func actionIconSize(s uiSize) (box, radius, glyph float32) {
	switch s {
	case sizeSmall:
		return 24, 4, 14
	case sizeLarge:
		return 44, 8, 24
	}
	return 36, 6, 20
}

func actionIconSurface(v actionIconVariant) buttonVariant {
	switch v {
	case actionIconFilled:
		return buttonFilled
	case actionIconOutlined:
		return buttonDefault
	}
	return buttonText
}

func actionIconActiveFill(tk *uiTokens, v actionIconVariant, s buttonState) ui.Color {
	switch {
	case v == actionIconFilled && s == buttonActive:
		return tk.Fill
	case v == actionIconOutlined && s != buttonRest:
		return tk.Fill
	}
	return tk.FillSecondary
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

type copyButtonProps struct {
	Variant actionIconVariant
	Size    uiSize
	Icon    *ui.SVG
}

const copiedFor = 2 * time.Second

func uiCopyButton(c *ui.Context, content string, p copyButtonProps) ui.Element {
	b := ui.ButtonBase(c)
	if b.Clicked() {
		c.WriteClipboard(content)
		b.Jump("copied", 1)
	}
	copied := b.AnimateTo("copied", 0, ui.Animation{Duration: copiedFor}) > 0
	icon, title := p.Icon, "Copy"
	if icon == nil {
		icon = iconCopy()
	}
	if copied {
		icon, title = iconCheck(), "Copied"
	}
	return styleActionIcon(c, b, icon, actionIconProps{Variant: p.Variant, Size: p.Size, Title: title, Active: copied})
}

type segmentedVariant int

const (
	segmentedFilled segmentedVariant = iota
	segmentedOutlined
)

type segmentedProps struct {
	Variant segmentedVariant
	Size    uiSize
	Block   bool
}

type segmentedMatch struct{ selected *int }

var segmentPress = ui.Animation{Duration: 120 * time.Millisecond, Ease: easeOut}

func uiSegmented(c *ui.Context, selected *int, options []string, p segmentedProps) ui.Element {
	tk := tokensOf(c)
	height, padding, fontSize := float32(32), float32(12), float32(13)
	switch p.Size {
	case sizeSmall:
		height, padding, fontSize = 26, 10, 12
	case sizeLarge:
		height, padding, fontSize = 36, 16, 14
	}

	indicator := tk.BgElevated
	parts := ui.SegmentedBase(c, selected, len(options))
	track := parts.Track.Padding(3).Gap(4).Radius(tk.BorderRadiusLG)
	if p.Variant == segmentedOutlined {
		track.Border(1, tk.BorderSecondary)
		indicator = tk.FillSecondary
	} else {
		track.Background(tk.BgLayout).Border(1, tk.FillQuaternary)
	}
	if p.Block {
		track.AlignSelf(ui.Stretch)
	}

	slide := ui.ElementTransition{Duration: 240 * time.Millisecond, Ease: easeOut, Match: segmentedMatch{selected}}

	track.Children(func() {
		for i, option := range options {
			seg := parts.Segment(i).Height(height).PaddingX(padding).Radius(tk.BorderRadius).
				FontSize(fontSize).FontWeight(500).Cursor(ui.CursorPointer)
			if p.Block {
				seg.Grow(1).Basis(0)
			}
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
						Radius(tk.BorderRadius).Background(indicator).ZIndex(-1).Transition(slide)
				}
				ui.Text(c, option).SingleLine()
			})
		}
	})
	return track
}

type checkboxShape int

const (
	checkboxSquare checkboxShape = iota
	checkboxCircle
)

type checkboxProps struct {
	Indeterminate bool
	Shape         checkboxShape
	Size          float32
}

var (
	checkGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`))
	dashGlyph  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/></svg>`))

	checkboxTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: ui.CubicBezier(0.215, 0.61, 0.355, 1), Colors: true}
)

func uiCheckbox(c *ui.Context, checked *bool, label string, p checkboxProps) ui.Element {
	tk := tokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 16
	}

	state := ui.CheckOff
	switch {
	case p.Indeterminate:
		state = ui.CheckMixed
	case *checked:
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

	radius := float32(4)
	glyphScale := float32(0.75)
	if p.Shape == checkboxCircle {
		radius, glyphScale = size/2, 0.66
	}

	row.Children(func() {
		box := ui.Box(c).Size(size, size).Radius(radius).Border(1, border).Background(bg).
			TextColor(tk.BgLayout).Center().Transition(checkboxTransition)
		box.Children(func() {
			switch state {
			case ui.CheckOn:
				ui.Icon(c, checkGlyph).FontSize(size * glyphScale)
			case ui.CheckMixed:
				ui.Icon(c, dashGlyph).FontSize(size * glyphScale)
			}
		})
		if label != "" {
			ui.Text(c, label).SingleLine()
		}
	})
	return row
}

type switchProps struct {
	Size uiSize
}

type switchMetrics struct {
	w, h, thumb, checkedX, pressedCheckedX, pressedWidth float32
}

var switchTransition = ui.ElementTransition{Duration: 200 * time.Millisecond, Ease: easeOut, Colors: true, Shadows: true}

func uiSwitch(c *ui.Context, on *bool, p switchProps) ui.Element {
	tk := tokensOf(c)
	m := switchMetrics{36, 22, 18, 14, 10, 22}
	if p.Size == sizeSmall {
		m = switchMetrics{28, 16, 12, 12, 8, 16}
	}

	track := ui.SwitchBase(c, on).Size(m.w, m.h).Padding(2).Radius(100).
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

	targetX, targetW := float32(0), m.thumb
	if pressed {
		targetW = m.pressedWidth
	}
	if *on {
		targetX = m.checkedX
		if pressed {
			targetX = m.pressedCheckedX
		}
	}

	track.Children(func() {
		thumb := ui.Box(c).Height(m.thumb).Radius(m.thumb/2).Background(tk.BgContainer).
			Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04)).
			Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.06)).
			Shadow(0, 3, 8, 0, ui.RGBA(0, 30, 80, 0.16))
		thumb.Width(thumb.AnimateTo("w", targetW, switchThumbAnim))
		thumb.Translate(thumb.AnimateTo("x", targetX, switchThumbAnim), 0)
	})
	return track
}
