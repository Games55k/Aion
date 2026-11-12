package choice

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ColorSwatch struct {
	// Color is the swatch's color; a fully transparent one shows a checkerboard.
	Color ui.Color
	// Title names the color in a tooltip and to assistive technology.
	Title string
}

type ColorSwatchesProps struct {
	// Colors are the swatches, in order.
	Colors []ColorSwatch
	// Size is a swatch's side in DIPs: 24 when zero.
	Size float32
	// Square rounds the swatches' corners instead of making them circles.
	Square bool
	// EnableColorPicker adds a button opening a ColorPicker for a color of one's own, which then
	// shows as a swatch of its own after the others.
	EnableColorPicker bool
	// Alpha lets the color picker choose the opacity too.
	Alpha bool
}

// ColorSwatches chooses one of p.Colors into *value from a row of swatches; the chosen one shows a
// check. The arrows move among them.
func ColorSwatches(c *ui.Context, value *ui.Color, p ColorSwatchesProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 24
	}
	radius := size / 2
	if p.Square {
		radius = tk.BorderRadius
	}
	return ui.RadioGroup(c, func() {
		for i, sw := range p.Colors {
			b := ui.RadioBase(c.Key(i), value, sw.Color).Size(size, size).Radius(radius).Center().Shrink(0).
				Cursor(ui.CursorPointer).Transition(checkboxTransition)
			if sw.Title != "" {
				b.Label(sw.Title)
				overlay.Tooltip(c, b, sw.Title, overlay.TooltipProps{})
			}
			chosen := *value == sw.Color
			ring := tk.FillSecondary
			if chosen {
				ring = tk.Fill
			}
			if sw.Color.A == 0 {
				b.Background(tk.BgContainer).Clip()
			} else {
				b.Background(sw.Color)
			}
			b.InsetShadow(0, 0, 0, 1, ring)
			if b.Hovered() {
				b.Shadow(0, 0, 0, 2, tk.Text)
			}
			b.Children(func() {
				if sw.Color.A == 0 {
					checker(c, tk, size)
				}
				if chosen {
					ui.Icon(c, lucide.Check()).FontSize(14).TextColor(checkColor(sw.Color, tk))
				}
			})
		}
		if p.EnableColorPicker {
			customPicker(c, tk, value, p, size, radius)
		}
	}).Row().Wrap().Gap(6)
}

func customPicker(c *ui.Context, tk *theme.Tokens, value *ui.Color, p ColorSwatchesProps, size, radius float32) {
	presets := make([]ui.Color, len(p.Colors))
	custom := *value != (ui.Color{})
	for i, sw := range p.Colors {
		presets[i] = sw.Color
		if sw.Color == *value {
			custom = false
		}
	}
	picker := ColorPickerProps{Alpha: p.Alpha, Presets: presets}
	ui.Box(c).Width(1).Height(size*0.75).Margin(0, 2).AlignSelf(ui.Center).Background(tk.FillSecondary)
	if custom {
		chip := ui.ButtonBase(c.Key("custom")).Label(formatHex(*value)).Size(size, size).Radius(radius).Clip().
			Center().Shrink(0).Cursor(ui.CursorPointer).InsetShadow(0, 0, 0, 1, tk.Fill)
		if chip.Hovered() {
			chip.Shadow(0, 0, 0, 2, tk.Text)
		}
		chip.Children(func() {
			colorTile(c, tk, *value)
			glyph, font := lucide.Check(), float32(14)
			if chip.Hovered() {
				glyph, font = lucide.Pencil(), 12
			}
			ui.Row(c).Absolute().Fill().Center().IgnoreInput(true).Children(func() {
				ui.Icon(c, glyph).FontSize(font).TextColor(checkColor(*value, tk))
			})
		})
		ColorPopover(c, chip, value, picker)
	}
	add := ui.ButtonBase(c.Key("add")).Label("Custom color").Size(size, size).Radius(radius).Center().Shrink(0).
		Cursor(ui.CursorPointer).TextColor(ui.RGB(255, 255, 255)).InsetShadow(0, 0, 0, 1, tk.FillSecondary).
		ConicGradient(ui.ConicGradient{Stops: hueTurn})
	if add.Hovered() {
		add.Shadow(0, 0, 0, 2, tk.Text)
	}
	overlay.Tooltip(c, add, "Custom color", overlay.TooltipProps{})
	add.Children(func() { ui.Icon(c, lucide.Plus()).FontSize(12) })
	ColorPopover(c, add, value, picker)
}

var hueTurn = []ui.GradientStop{
	{Color: ui.Hex("#f5222d"), At: 0}, {Color: ui.Hex("#fa541c"), At: 1.0 / 12}, {Color: ui.Hex("#fa8c16"), At: 2.0 / 12},
	{Color: ui.Hex("#faad14"), At: 3.0 / 12}, {Color: ui.Hex("#fadb14"), At: 4.0 / 12}, {Color: ui.Hex("#a0d911"), At: 5.0 / 12},
	{Color: ui.Hex("#52c41a"), At: 6.0 / 12}, {Color: ui.Hex("#13c2c2"), At: 7.0 / 12}, {Color: ui.Hex("#1677ff"), At: 8.0 / 12},
	{Color: ui.Hex("#2f54eb"), At: 9.0 / 12}, {Color: ui.Hex("#722ed1"), At: 10.0 / 12}, {Color: ui.Hex("#eb2f96"), At: 11.0 / 12},
	{Color: ui.Hex("#f5222d"), At: 1},
}

func checker(c *ui.Context, tk *theme.Tokens, size float32) {
	half := size / 2
	for _, at := range [][2]float32{{0, 0}, {half, half}} {
		ui.Box(c.Key(at)).Absolute().Left(at[0]).Top(at[1]).Size(half, half).Background(tk.FillSecondary).IgnoreInput(true)
	}
}

func checkColor(bg ui.Color, tk *theme.Tokens) ui.Color {
	if bg.A == 0 {
		return tk.Text.Alpha(0.33)
	}
	l := (0.2126*float32(bg.R) + 0.7152*float32(bg.G) + 0.0722*float32(bg.B)) / 255
	if l > 0.6 {
		return ui.RGB(0, 0, 0).Alpha(0.33)
	}
	return ui.RGB(255, 255, 255).Alpha(0.33)
}
