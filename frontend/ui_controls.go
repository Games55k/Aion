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
		return buttonMetrics{28, 12, 14}, tk.BorderRadiusSM
	}
	return buttonMetrics{36, 16, 14}, tk.BorderRadius
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
		return c, ui.Transparent, ui.RGB(255, 255, 255)
	case buttonText:
		return pick(ui.Transparent, tk.FillTertiary, tk.FillSecondary), ui.Transparent, tk.Text
	default:
		bg = pick(tk.FillTertiary, tk.FillSecondary, tk.Fill)
		return bg, ui.Transparent, tk.Text
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
		return c, ui.Transparent, ui.RGB(255, 255, 255)
	case buttonText:
		return pick(ui.Transparent, tk.ErrorBgHover, tk.ErrorBgActive), ui.Transparent, fg
	default:
		return pick(tk.FillTertiary, tk.ErrorBgHover, tk.ErrorBgActive),
			ui.Transparent,
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
		box    float32 = 28
		radius float32 = 8
		glyph  float32 = 18
	)

	b.Size(box, box).Radius(radius).
		Cursor(ui.CursorPointer).Transition(buttonTransition)
	if p.Title != "" {
		b.Label(p.Title)
		uiTooltip(c, b, p.Title)
	}

	state := stateOfButton(b)
	bg, _, _ := buttonColors(tk, buttonText, state)
	if p.Active {
		bg = tk.FillSecondary
	}
	b.Background(bg).TextColor(actionIconGlyphColor(tk, p.Danger, state))

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

type segmentedOption struct {
	Label string
	Icon  *ui.SVG
}

type segmentedProps struct {
	Size uiSize
}

type segmentedMatch struct{ selected *int }

var segmentPress = ui.Animation{Duration: 120 * time.Millisecond, Ease: easeOut}

func uiSegmented(c *ui.Context, selected *int, options []segmentedOption, p segmentedProps) ui.Element {
	tk := tokensOf(c)
	height, padding, fontSize, iconSize, gap := float32(32), float32(12), float32(13), float32(14), float32(6)
	if p.Size == sizeSmall {
		height, padding, fontSize, iconSize, gap = 26, 10, 12, 13, 5
	}

	parts := ui.SegmentedBase(c, selected, len(options))
	track := parts.Track.Padding(4).Gap(4).Radius(40).
		Background(tk.FillTertiary)

	slide := ui.ElementTransition{Duration: 240 * time.Millisecond, Ease: easeOut, Match: segmentedMatch{selected}}

	track.Children(func() {
		for i, option := range options {
			seg := parts.Segment(i).Height(height).PaddingX(padding).Gap(gap).Radius(40).
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
					ind := ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
						Radius(40).Background(tk.BgElevated).ZIndex(-1).Transition(slide)
					if !tk.Dark {
						ind.Shadow(0, 2, 8, 0, ui.RGBA(24, 24, 24, 0.08))
					}
				}
				if option.Icon != nil {
					ui.Icon(c, option.Icon).FontSize(iconSize).Shrink(0)
				}
				if option.Label != "" {
					ui.Text(c, option.Label).SingleLine()
				}
			})
		}
	})
	return track
}

var (
	checkGlyph         = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`))
	checkboxTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: ui.CubicBezier(0.215, 0.61, 0.355, 1), Colors: true}
)

func uiCheckbox(c *ui.Context, checked *bool, label string) ui.Element {
	tk := tokensOf(c)
	const size float32 = 20

	state := ui.CheckOff
	if *checked {
		state = ui.CheckOn
	}
	row := ui.CheckboxStateBase(c, &state).Gap(12).Cursor(ui.CursorPointer)
	if row.Changed() {
		*checked = state == ui.CheckOn
	}

	filled := state != ui.CheckOff
	border, bg := tk.Border, ui.Transparent
	switch {
	case filled:
		border, bg = tk.Primary, tk.Primary
	case row.Hovered():
		border = tk.Primary
	}

	row.Children(func() {
		box := ui.Box(c).Size(size, size).Radius(40).Border(1.5, border).Background(bg).
			TextColor(ui.RGB(255, 255, 255)).Center().Transition(checkboxTransition)
		box.Children(func() {
			if state == ui.CheckOn {
				ui.Icon(c, checkGlyph).FontSize(12)
			}
		})
		if label != "" {
			ui.Text(c, label).SingleLine()
		}
	})
	return row
}

var switchTransition = ui.ElementTransition{Duration: 200 * time.Millisecond, Ease: easeOut, Colors: true}

func uiSwitch(c *ui.Context, on *bool) ui.Element {
	tk := tokensOf(c)
	const (
		w               float32 = 44
		h               float32 = 24
		thumbSize       float32 = 20
		checkedX        float32 = 20
		pressedCheckedX float32 = 16
		pressedWidth    float32 = 24
	)

	track := ui.SwitchBase(c, on).Size(w, h).Padding(2).Radius(40).
		Cursor(ui.CursorPointer).Transition(switchTransition)

	hovered, pressed := track.Hovered() && !track.IsDisabled(), track.Pressed()
	bg := tk.Fill
	switch {
	case *on && hovered:
		bg = tk.PrimaryHover
	case *on:
		bg = tk.Primary
	case hovered:
		bg = tk.FillSecondary
	}
	track.Background(bg)

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
		thumb := ui.Box(c).Height(thumbSize).Radius(40).Background(ui.RGB(255, 255, 255))
		thumb.Width(thumb.AnimateTo("w", targetW, switchThumbAnim))
		thumb.Translate(thumb.AnimateTo("x", targetX, switchThumbAnim), 0)
	})
	return track
}
