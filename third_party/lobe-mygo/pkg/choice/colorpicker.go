package choice

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ColorPickerTrigger int

const (
	ColorTriggerSwatch ColorPickerTrigger = iota
	ColorTriggerField
	ColorTriggerChip
)

type ColorPickerProps struct {
	// Alpha adds an opacity slider and field.
	Alpha bool
	// Presets show as a grid of swatches under the inputs.
	Presets []ui.Color
	// Trigger is what opens the picker: a color tile, an Input-like field with the hex value, or a
	// round chip for toolbars.
	Trigger ColorPickerTrigger
	// Size sets the trigger's size (24, 28 or 32 for the tile; Input's heights otherwise).
	Size theme.Size
	// Disabled keeps the picker from opening.
	Disabled bool
}

// ColorPicker chooses *value in a panel that opens below its trigger: a saturation and brightness
// area, a hue slider, an optional alpha slider, a hex field and presets. Its value changes as the
// controls move, and the half of the compare tile showing the color it opened with brings it back.
// It returns the trigger.
func ColorPicker(c *ui.Context, value *ui.Color, p ColorPickerProps) ui.Element {
	tk := theme.TokensOf(c)
	b := ui.ButtonBase(c).Cursor(ui.CursorPointer).Shrink(0).AlignSelf(ui.Start)
	if p.Disabled {
		b.Disabled(true).Cursor(ui.CursorNotAllowed).Opacity(0.5)
	}
	switch p.Trigger {
	case ColorTriggerField:
		fieldTrigger(c, tk, b, *value, p)
	case ColorTriggerChip:
		chipTrigger(c, tk, b, *value, p.Size)
	default:
		side := float32(28)
		switch p.Size {
		case theme.Small:
			side = 24
		case theme.Large:
			side = 32
		}
		b.Size(side, side).Radius(8).Clip().InsetShadow(0, 0, 0, 1, tk.FillSecondary).Children(func() {
			colorTile(c, tk, *value)
		})
	}
	ColorPopover(c, b, value, p)
	return b
}

// ColorPopover opens the ColorPicker panel by anchor, a button, when it is clicked, choosing
// *value; ColorPicker and ColorSwatches open theirs with it.
func ColorPopover(c *ui.Context, anchor ui.Element, value *ui.Color, p ColorPickerProps) {
	tk := theme.TokensOf(c)
	st := ui.Local(anchor, "picker", func() pickerState { return pickerState{} })
	if p.Disabled {
		return
	}
	overlay.Popover(c, anchor, &st.open, overlay.PopoverProps{Placement: overlay.Bottom}, func() {
		if st.open && !st.wasOpen {
			st.original = *value
			st.synced = false
		}
		colorPanel(c, tk, value, st, p)
	})
	st.wasOpen = st.open
}

func colorTile(c *ui.Context, tk *theme.Tokens, col ui.Color) {
	if col.A < 255 {
		checkerboard(c, tk, 4)
	}
	ui.Box(c).Absolute().Fill().Background(col).IgnoreInput(true)
}

func fieldTrigger(c *ui.Context, tk *theme.Tokens, b ui.Element, col ui.Color, p ColorPickerProps) {
	height, font, padL, padR, minW := float32(32), float32(14), float32(8), float32(10), float32(140)
	switch p.Size {
	case theme.Small:
		height, font, padL, padR, minW = 24, 12, 5, 8, 112
	case theme.Large:
		height, font, padL, padR, minW = 40, 16, 10, 12, 168
	}
	b.Height(height).MinWidth(minW).Padding(0, padR, 0, padL).Gap(8).AlignItems(ui.Center).Radius(tk.BorderRadius).
		FontSize(font).TextColor(tk.Text).Border(1, tk.BorderSecondary).Background(tk.BgContainer)
	if b.Hovered() {
		b.Border(1, tk.Border)
	}
	b.Children(func() {
		ui.Box(c).Size(height-12, height-12).Radius(4).Clip().Shrink(0).InsetShadow(0, 0, 0, 1, tk.FillSecondary).
			Children(func() { colorTile(c, tk, col) })
		ui.Text(c, formatHex(col)).FontFeatures("tnum").SingleLine()
		if p.Alpha {
			ui.Text(c, strconv.Itoa(int(float32(col.A)/255*100+0.5))+"%").FontFeatures("tnum").TextColor(tk.TextTertiary).SingleLine()
		}
		ui.Box(c).Grow(1)
		ui.Icon(c, lucide.ChevronDown()).FontSize(14).TextColor(tk.TextTertiary)
	})
}

func chipTrigger(c *ui.Context, tk *theme.Tokens, b ui.Element, col ui.Color, size theme.Size) {
	height, font, padL, padR := float32(32), float32(13), float32(6), float32(11)
	switch size {
	case theme.Small:
		height, font, padL, padR = 24, 12, 4, 8
	case theme.Large:
		height, font, padL, padR = 40, 14, 8, 14
	}
	bg := tk.FillTertiary
	if b.Hovered() {
		bg = tk.FillSecondary
	}
	b.Height(height).Padding(0, padR, 0, padL).Gap(6).AlignItems(ui.Center).Radius(height / 2).
		FontSize(font).TextColor(tk.Text).Background(bg)
	b.Children(func() {
		dot := height - 2*padL - 4
		ui.Box(c).Size(dot, dot).Radius(dot/2).Clip().Shrink(0).InsetShadow(0, 0, 0, 1, tk.FillSecondary).
			Children(func() { colorTile(c, tk, col) })
		ui.Text(c, formatHex(col)).FontFeatures("tnum").SingleLine()
	})
}
