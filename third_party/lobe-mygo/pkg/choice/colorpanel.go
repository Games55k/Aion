package choice

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const (
	panelWidth = 240
	thumbSize  = 14
)

type pickerState struct {
	hsv              hsva
	last, original   ui.Color
	hue, hueSeen     float64
	alpha, alphaSeen float64
	hex, alphaText   string
	hexWas, alphaWas bool
	open, wasOpen    bool
	synced           bool
}

var rainbow = []ui.GradientStop{
	{Color: ui.RGB(255, 0, 0), At: 0}, {Color: ui.RGB(255, 255, 0), At: 1.0 / 6}, {Color: ui.RGB(0, 255, 0), At: 2.0 / 6},
	{Color: ui.RGB(0, 255, 255), At: 3.0 / 6}, {Color: ui.RGB(0, 0, 255), At: 4.0 / 6}, {Color: ui.RGB(255, 0, 255), At: 5.0 / 6},
	{Color: ui.RGB(255, 0, 0), At: 1},
}

func (st *pickerState) sync(value *ui.Color) {
	if st.synced && *value == st.last {
		return
	}
	next := toHSV(*value)
	if st.synced && (next.s == 0 || next.v == 0) {
		next.h = st.hsv.h
	}
	st.hsv, st.last, st.synced = next, *value, true
}

func (st *pickerState) set(value *ui.Color, h hsva) {
	st.hsv = h
	*value = h.color()
	st.last = *value
}

func colorPanel(c *ui.Context, tk *theme.Tokens, value *ui.Color, st *pickerState, p ColorPickerProps) {
	if st.hue != st.hueSeen {
		h := st.hsv
		h.h = float32(st.hue)
		st.set(value, h)
	}
	if st.alpha != st.alphaSeen {
		h := st.hsv
		h.a = float32(st.alpha)
		st.set(value, h)
	}
	st.sync(value)
	st.hue, st.hueSeen = float64(st.hsv.h), float64(st.hsv.h)
	st.alpha, st.alphaSeen = float64(st.hsv.a), float64(st.hsv.a)

	ui.Column(c).Width(panelWidth).Gap(12).Children(func() {
		saturationArea(c, value, st)
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
				opaque := st.hsv
				opaque.a = 1
				pickerSlider(c, tk, "Hue", &st.hue, 0, 360, hsva{st.hsv.h, 1, 1, 1}.color(), func(track ui.Element) {
					track.LinearGradient(ui.LinearGradient{Angle: 90, Stops: rainbow})
				})
				if p.Alpha {
					pickerSlider(c, tk, "Alpha", &st.alpha, 0, 1, *value, func(track ui.Element) {
						track.Children(func() {
							checkerboard(c, tk, 4)
							ui.Box(c).Absolute().Fill().LinearGradient(ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
								{Color: opaque.color().Alpha(0), At: 0}, {Color: opaque.color(), At: 1},
							}})
						})
					})
				}
			})
			compare(c, tk, value, st, p.Alpha)
		})
		colorInputs(c, tk, value, st, p.Alpha)
		if len(p.Presets) > 0 {
			presets(c, tk, value, st, p.Presets)
		}
	})
}

func saturationArea(c *ui.Context, value *ui.Color, st *pickerState) {
	area := ui.ButtonBase(c).Label("Saturation and brightness").Role(ui.RoleSlider).Size(panelWidth, 150).
		Radius(10).Cursor(ui.CursorCrosshair).FocusRing(false)
	x, y, _ := area.PointerPosition()
	_, _, held := area.Dragged()
	if held || area.Clicked() {
		h := st.hsv
		h.s, h.v = clamp01(x/panelWidth), 1-clamp01(y/150)
		st.set(value, h)
	}
	for _, k := range []struct {
		key    ui.Key
		ds, dv float32
	}{{ui.KeyLeft, -0.01, 0}, {ui.KeyRight, 0.01, 0}, {ui.KeyUp, 0, 0.01}, {ui.KeyDown, 0, -0.01}} {
		if area.Shortcut(0, k.key) {
			h := st.hsv
			h.s, h.v = clamp01(h.s+k.ds), clamp01(h.v+k.dv)
			st.set(value, h)
		}
	}
	area.Background(hsva{st.hsv.h, 1, 1, 1}.color()).InsetShadow(0, 0, 0, 1, theme.TokensOf(c).FillSecondary)
	area.Children(func() {
		ui.Box(c).Absolute().Fill().Radius(10).IgnoreInput(true).LinearGradient(ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
			{Color: ui.RGB(255, 255, 255), At: 0}, {Color: ui.RGBA(255, 255, 255, 0), At: 1},
		}})
		ui.Box(c).Absolute().Fill().Radius(10).IgnoreInput(true).LinearGradient(ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
			{Color: ui.RGBA(0, 0, 0, 0), At: 0}, {Color: ui.RGB(0, 0, 0), At: 1},
		}})
		opaque := st.hsv
		opaque.a = 1
		thumb(c, opaque.color()).Left(st.hsv.s*panelWidth - thumbSize/2).Top((1-st.hsv.v)*150 - thumbSize/2)
	})
}

func thumb(c *ui.Context, fill ui.Color) ui.Element {
	t := ui.Box(c).Absolute().Size(thumbSize, thumbSize).Radius(thumbSize/2).Border(2, ui.RGB(255, 255, 255)).
		Background(fill).IgnoreInput(true)
	t.Shadow(0, 0, 0, 1, ui.RGBA(0, 0, 0, 0.18)).Shadow(0, 1, 4, 0, ui.RGBA(0, 0, 0, 0.3))
	return t
}

func pickerSlider(c *ui.Context, tk *theme.Tokens, label string, v *float64, lo, hi float64, fill ui.Color, track func(ui.Element)) {
	s := ui.SliderBase(c.Key(label), v, lo, hi).Label(label).Height(thumbSize).FillWidth().PaddingX(thumbSize / 2).
		Cursor(ui.CursorPointer).FocusRing(false)
	w := s.Bounds().W
	frac := float32((*v - lo) / (hi - lo))
	s.Children(func() {
		t := ui.Box(c).Absolute().Left(0).Right(0).Top(2).Height(10).Radius(5).Clip().IgnoreInput(true).
			InsetShadow(0, 0, 0, 1, tk.FillSecondary)
		track(t)
		thumb(c, fill).Top(0).Left(frac * (w - thumbSize))
	})
}

func checkerboard(c *ui.Context, tk *theme.Tokens, cell float32) {
	ui.Box(c).Absolute().Fill().IgnoreInput(true).Draw(func(pt *ui.Painter, r ui.Rect) {
		for y := float32(0); y < r.H; y += cell {
			for x := float32(0); x < r.W; x += cell {
				if int(x/cell+y/cell)%2 == 0 {
					pt.Fill(ui.Rect{X: r.X + x, Y: r.Y + y, W: min(cell, r.W-x), H: min(cell, r.H-y)}, tk.FillSecondary, 0)
				}
			}
		}
	})
}

func compare(c *ui.Context, tk *theme.Tokens, value *ui.Color, st *pickerState, alpha bool) {
	h := float32(28)
	if alpha {
		h = 30
	}
	ui.Column(c).Size(28, h).Radius(8).Clip().Shrink(0).InsetShadow(0, 0, 0, 1, tk.FillSecondary).Children(func() {
		checkerboard(c, tk, 4)
		ui.Column(c).Absolute().Fill().Children(func() {
			restore := ui.ButtonBase(c).Label("Restore").Grow(1).FillWidth().Background(st.original).Cursor(ui.CursorPointer)
			if restore.Clicked() {
				*value = st.original
				st.sync(value)
			}
			ui.Box(c).Grow(1).FillWidth().Background(*value)
		})
	})
}

func colorInputs(c *ui.Context, tk *theme.Tokens, value *ui.Color, st *pickerState, alpha bool) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		var hex ui.Element
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			hex = field.Input(c, &st.hex, field.InputProps{Size: theme.Small}).Label("Hex").FontFeatures("tnum")
		})
		focused := hex.Focused()
		if hex.Submitted() || st.hexWas && !focused {
			if col, ok := parseHex(st.hex); ok {
				if len(strings.TrimPrefix(strings.TrimSpace(st.hex), "#")) <= 6 && alpha {
					col.A = value.A
				}
				*value = col
				st.sync(value)
			}
		}
		if !focused || hex.Submitted() {
			st.hex = formatHex(*value)
		}
		st.hexWas = focused
		if !alpha {
			return
		}
		ui.Box(c).Width(52).Shrink(0).Children(func() {
			a := field.Input(c, &st.alphaText, field.InputProps{Size: theme.Small}).Label("Alpha percent").FontFeatures("tnum")
			af := a.Focused()
			if a.Submitted() || st.alphaWas && !af {
				if n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(st.alphaText), "%")); err == nil {
					h := st.hsv
					h.a = clamp01(float32(n) / 100)
					st.set(value, h)
				}
			}
			if !af || a.Submitted() {
				st.alphaText = strconv.Itoa(int(float32(value.A)/255*100 + 0.5))
			}
			st.alphaWas = af
		})
	})
}

func presets(c *ui.Context, tk *theme.Tokens, value *ui.Color, st *pickerState, colors []ui.Color) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary).Margin(0, 0, 4, 0)
		ui.Text(c, "Presets").FontSize(11).FontWeight(500).TextColor(tk.TextTertiary).Padding(0, 2)
		cell := float32(math.Floor((panelWidth-8*5)/9*100)) / 100
		ui.Row(c).Wrap().Gap(5).Children(func() {
			for i, col := range colors {
				b := ui.ButtonBase(c.Key(i)).Label(formatHex(col)).Size(cell, cell).Radius(6).Clip().
					Background(col).Cursor(ui.CursorPointer).InsetShadow(0, 0, 0, 1, tk.FillSecondary)
				if *value == col {
					b.Shadow(0, 0, 0, 2, tk.BgElevated).Shadow(0, 0, 0, 3.5, tk.Text)
				}
				if b.Clicked() {
					*value = col
					st.sync(value)
				}
			}
		})
	})
}

func clamp01(v float32) float32 { return min(max(v, 0), 1) }
