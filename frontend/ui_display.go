package frontend

import (
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

type tagVariant int

const (
	tagFilled tagVariant = iota
	tagOutlined
	tagBorderless
)

type tagColor string

const (
	tagOrange tagColor = "orange"
	tagGreen  tagColor = "green"
	tagCyan   tagColor = "cyan"
	tagBlue   tagColor = "blue"
	tagPurple tagColor = "purple"
	tagGray   tagColor = "gray"

	tagError      tagColor = "error"
	tagWarning    tagColor = "warning"
	tagSuccess    tagColor = "success"
	tagProcessing tagColor = "processing"
)

type tagProps struct {
	Variant tagVariant
	Color   tagColor
	Size    uiSize
	Round   bool
}

type tagColors struct {
	text, bg, border ui.Color
	hasText          bool
}

func uiTag(c *ui.Context, label string, p tagProps) ui.Element {
	tk := tokensOf(c)
	tag := ui.Row(c).Gap(5).FontSize(tk.FontSizeSM)

	height, padding, radius := tagSize(p.Size, p.Round)
	tag.Height(height).PaddingX(padding).Radius(radius)

	colors := tagColorsFor(tk, p.Color, p.Variant)
	switch p.Variant {
	case tagOutlined:
		border := tk.BorderSecondary
		if colors.border.A > 0 {
			border = colors.border
		}
		bg := tk.BgContainer
		if p.Color != "" {
			bg = colors.bg
		}
		tag.Background(bg).Border(1, border)
	case tagBorderless:
	default:
		bg := tk.FillTertiary
		if p.Color != "" {
			bg = colors.bg
		}
		tag.Background(bg)
	}
	text := tk.TextSecondary
	if colors.hasText {
		text = colors.text
	}
	tag.TextColor(text)

	tag.Children(func() {
		ui.Text(c, label).SingleLine()
	})
	return tag
}

func tagSize(s uiSize, round bool) (height, padding, radius float32) {
	if s == sizeSmall {
		height, padding, radius = 20, 4, 3
		if round {
			padding = 8
		}
	} else {
		height, padding, radius = 22, 8, 3
		if round {
			padding = 10
		}
	}
	if round {
		radius = 999
	}
	return height, padding, radius
}

func tagColorsFor(tk *uiTokens, color tagColor, v tagVariant) tagColors {
	borderless, filled := v == tagBorderless, v == tagFilled
	tinted := func(text, fillTertiary, fillQuaternary ui.Color) tagColors {
		out := tagColors{hasText: true, text: text, bg: fillTertiary, border: fillTertiary}
		if filled {
			out.border = fillQuaternary
		}
		if borderless {
			out.bg = ui.Transparent
		}
		return out
	}

	if status, ok := tagStatus(tk, color); ok {
		return tinted(status[0], status[1], status[2])
	}
	if name := string(color); name != "" {
		if _, okBase := tk.Color(strings.ToUpper(name[:1]) + name[1:]); okBase {
			title := strings.ToUpper(name[:1]) + name[1:]
			active, _ := tk.Color(title + "Active")
			fill3, _ := tk.Color(title + "FillTertiary")
			fill4, _ := tk.Color(title + "FillQuaternary")
			return tinted(active, fill3, fill4)
		}
	}
	return tagColors{}
}

func tagStatus(tk *uiTokens, color tagColor) ([3]ui.Color, bool) {
	switch color {
	case tagError:
		return [3]ui.Color{tk.Error, tk.ErrorFillTertiary, tk.ErrorFillQuaternary}, true
	case tagWarning:
		return [3]ui.Color{tk.Warning, tk.WarningFillTertiary, tk.WarningFillQuaternary}, true
	case tagSuccess:
		return [3]ui.Color{tk.Success, tk.SuccessFillTertiary, tk.SuccessFillQuaternary}, true
	case tagProcessing:
		return [3]ui.Color{tk.Info, tk.InfoFillTertiary, tk.InfoFillQuaternary}, true
	}
	return [3]ui.Color{}, false
}

type emptyProps struct {
	Icon               *ui.SVG
	Title, Description string
	Action             func()
}

func uiEmpty(c *ui.Context, p emptyProps) ui.Element {
	tk := tokensOf(c)
	root := ui.Column(c).Gap(4).Padding(16, 8).AlignItems(ui.Center).FillWidth()
	root.Children(func() {
		if p.Icon != nil {
			ui.Icon(c, p.Icon).FontSize(32).TextColor(tk.TextQuaternary)
		}
		if p.Title != "" {
			ui.Text(c, p.Title).FontWeight(500).TextColor(tk.Text).TextAlign(ui.Center).Margin(8, 0, 0, 0)
		}
		if p.Description != "" {
			ui.Text(c, p.Description).FontSize(13).TextColor(tk.TextTertiary).TextAlign(ui.Center).MaxWidth(13 * 0.55 * 34)
		}
		if p.Action != nil {
			ui.Row(c).Margin(8, 0, 0, 0).Children(p.Action)
		}
	})
	return root
}

type listItem struct {
	Key     string
	Label   string
	Icon    *ui.SVG
	Extra   string
	Actions func()
}

type listProps struct {
	Items     []listItem
	Active    string
	Collapse  float32
	RailWidth float32
}

var listTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: easeOut, Colors: true}

func uiList(c *ui.Context, p listProps) string {
	tk := tokensOf(c)
	root := ui.Column(c).Gap(2).AlignSelf(ui.Stretch)
	clicked := ""
	root.Children(func() {
		for _, it := range p.Items {
			if listRow(c.Key(it.Key), tk, it, it.Key == p.Active, p, root.LayoutBounds().W) {
				clicked = it.Key
			}
		}
	})
	return clicked
}

func listRow(c *ui.Context, tk *uiTokens, it listItem, active bool, p listProps, width float32) bool {
	clicked := false
	cell := ui.Box(c).AlignSelf(ui.Stretch)
	cell.Children(func() {
		row := ui.ButtonBase(c).Label(it.Label).Row().Gap(12).AlignItems(ui.Center).
			Justify(ui.Start).MinHeight(32).PaddingY(6).PaddingX(12).Radius(tk.BorderRadius).
			FontSize(14).AlignSelf(ui.Stretch).Transition(listTransition)
		const pad = float32(12)
		k := min(max(p.Collapse, 0), 1)
		fg := tk.TextSecondary
		switch {
		case active:
			fg = tk.Text
			row.Background(tk.FillSecondary).Cursor(ui.CursorPointer)
		case row.Hovered():
			fg = tk.Text
			row.Background(tk.FillTertiary).Cursor(ui.CursorPointer)
		default:
			row.Cursor(ui.CursorPointer)
		}
		row.TextColor(fg)
		var lead ui.Element
		row.Children(func() {
			lead = ui.Row(c).Shrink(0).Children(func() {
				if it.Icon != nil {
					ui.Icon(c, it.Icon).FontSize(16)
				}
			})
			text := ui.Column(c).Gap(1).Grow(1).Basis(0).MinWidth(0)
			if k > 0 {
				text.Clip().Opacity(1 - k)
			}
			text.Children(func() {
				label := ui.Text(c, it.Label).SingleLine()
				if active {
					label.FontWeight(500)
				}
			})
			if it.Extra != "" && k < 1 {
				ui.Text(c, it.Extra).FontSize(12).TextColor(tk.TextDescription).Shrink(0).Opacity(1 - k)
			}
		})
		if k > 0 {
			railRow(c, row, lead, k, pad, p.RailWidth, width)
		}
		clicked = row.Clicked()
		if it.Actions != nil && (cell.Hovered() || cell.FocusWithin()) {
			ui.Row(c).Absolute().Right(8).Top(0).Bottom(0).Gap(4).AlignItems(ui.Center).Children(it.Actions)
		}
	})
	return clicked
}

func railRow(c *ui.Context, row, lead ui.Element, k, pad, rail, width float32) {
	sq, glyph := row.LayoutBounds().H, lead.LayoutBounds().W
	if sq <= 0 || glyph <= 0 || width <= 0 {
		c.Invalidate()
		return
	}
	if rail <= 0 {
		rail = width
	}
	lerp := func(a, b float32) float32 { return a + (b-a)*k }
	row.AlignSelf(ui.Start).Width(lerp(width, sq)).Gap(lerp(12, 0)).
		Margin(0, 0, 0, lerp(0, (rail-sq)/2)).Padding(6, lerp(pad, 0), 6, lerp(pad, (sq-glyph)/2))
}

type skeletonAnimation int

const (
	skeletonFade skeletonAnimation = iota
	skeletonStatic
)

type skeletonProps struct {
	Width, Height float32
	Animation     skeletonAnimation
}

func uiSkeleton(c *ui.Context, p skeletonProps) ui.Element {
	tk := tokensOf(c)
	e := ui.Box(c).Background(tk.FillContent).Radius(tk.BorderRadius).Shrink(0).Unselectable()
	if p.Width > 0 {
		e.Width(p.Width)
	} else {
		e.FillWidth()
	}
	h := p.Height
	if h == 0 {
		h = tk.FontSize
	}
	e.Height(h)
	if p.Animation == skeletonFade {
		e.Opacity(1 - 0.5*e.Loop("fade", 1600*time.Millisecond, ui.Bounce(ui.EaseInOut)))
	}
	return e
}

type hotkeyProps struct {
	Size uiSize
}

var isApple = sync.OnceValue(appleDevice)

func appleDevice() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "ios" }

func uiHotkey(c *ui.Context, keys string, p hotkeyProps) ui.Element {
	tk := tokensOf(c)
	apple := isApple()
	parts := splitHotkeyParts(keys)
	root := ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0)
	root.Children(func() {
		capGap := float32(6)
		if p.Size == sizeSmall {
			capGap = 4
		}
		hotkeyCap(c, tk, p.Size).Gap(capGap).Children(func() {
			for i, k := range parts {
				hotkeyKey(c.Key(i), k, apple, p.Size)
			}
		})
	})
	return root
}

func hotkeyCap(c *ui.Context, tk *uiTokens, size uiSize) ui.Element {
	font := float32(12)
	height := font * 1.8
	padX := float32(8)
	if size == sizeSmall {
		font = 11
		height = 16
		padX = 6
	}
	return ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Height(height).MinWidth(height).
		PaddingX(padX).Radius(tk.BorderRadiusSM).FontSize(font).LineHeight(1.1).TextColor(tk.TextSecondary).
		Background(tk.FillTertiary).Shrink(0)
}

type keyNameSpec struct {
	apple func() *ui.SVG
	scale float32
	text  string
	icon  func() *ui.SVG
}

var keyNames = map[string]keyNameSpec{
	"mod":       {apple: iconCommand, scale: 0.95, text: "Ctrl"},
	"alt":       {apple: iconOption, scale: 0.95, text: "Alt"},
	"option":    {apple: iconOption, scale: 0.95, text: "Alt"},
	"ctrl":      {apple: iconChevronUp, scale: 1, text: "Ctrl"},
	"control":   {apple: iconChevronUp, scale: 1, text: "Ctrl"},
	"shift":     {apple: iconArrowBigUp, scale: 1.15, text: "Shift"},
	"meta":      {apple: iconCommand, scale: 0.95, icon: iconGrid2x2},
	"cmd":       {apple: iconCommand, scale: 0.95, icon: iconGrid2x2},
	"command":   {apple: iconCommand, scale: 0.95, icon: iconGrid2x2},
	"enter":     {apple: iconCornerDownLeft, scale: 1, text: "Enter"},
	"backspace": {apple: iconDelete, scale: 1, text: "Backspace"},
	"tab":       {apple: iconArrowRightToLine, scale: 1, text: "Tab"},
	"up":        {icon: iconArrowUp},
	"down":      {icon: iconArrowDown},
	"left":      {icon: iconArrowLeft},
	"right":     {icon: iconArrowRight},
	"space":     {icon: iconSpace},
	"comma":     {text: ","},
	"period":    {text: "."},
	"slash":     {text: "?"},
	"semicolon": {text: ";"},
	"minus":     {text: "-"},
	"equal":     {text: "+"},
}

func hotkeyKey(c *ui.Context, key string, apple bool, size uiSize) {
	iconSize := float32(12)
	if size == sizeSmall {
		iconSize = 11
	}
	n, ok := keyNames[strings.ToLower(key)]
	switch {
	case ok && apple && n.apple != nil:
		ui.Icon(c, n.apple()).FontSize(iconSize * n.scale)
	case ok && n.text != "":
		ui.Text(c, n.text).SingleLine()
	case ok && n.icon != nil:
		ui.Icon(c, n.icon()).FontSize(iconSize)
	default:
		ui.Text(c, startCase(key)).SingleLine()
	}
}

func splitHotkeyParts(keys string) []string {
	var out []string
	for _, k := range strings.Split(strings.ReplaceAll(keys, "++", "+plus"), "+") {
		switch k = strings.TrimSpace(k); k {
		case "":
		case "plus":
			out = append(out, "+")
		default:
			out = append(out, k)
		}
	}
	return out
}

func startCase(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func hotkeyBinding(keys string) (ui.Modifiers, ui.Key, bool) {
	return parseHotkey(keys, isApple())
}

var namedKeys = map[string]ui.Key{
	"enter": ui.KeyEnter, "escape": ui.KeyEscape, "esc": ui.KeyEscape, "backspace": ui.KeyBackspace,
	"tab": ui.KeyTab, "space": ui.KeySpace, "delete": ui.KeyDelete, "home": ui.KeyHome, "end": ui.KeyEnd,
	"pageup": ui.KeyPageUp, "pagedown": ui.KeyPageDown, "left": ui.KeyLeft, "right": ui.KeyRight,
	"up": ui.KeyUp, "down": ui.KeyDown, "minus": ui.KeyMinus, "equal": ui.KeyEqual, "comma": ui.KeyComma,
	"period": ui.KeyPeriod, "slash": ui.KeySlash, "semicolon": ui.KeySemicolon,
	"f1": ui.KeyF1, "f2": ui.KeyF2, "f3": ui.KeyF3, "f4": ui.KeyF4, "f5": ui.KeyF5, "f6": ui.KeyF6,
	"f7": ui.KeyF7, "f8": ui.KeyF8, "f9": ui.KeyF9, "f10": ui.KeyF10, "f11": ui.KeyF11, "f12": ui.KeyF12,
}

func parseHotkey(keys string, apple bool) (ui.Modifiers, ui.Key, bool) {
	var mods ui.Modifiers
	var key ui.Key
	found := false
	for _, k := range splitHotkeyParts(keys) {
		switch k = strings.ToLower(k); k {
		case "mod":
			if apple {
				mods |= ui.Super
			} else {
				mods |= ui.Ctrl
			}
		case "ctrl", "control":
			mods |= ui.Ctrl
		case "shift":
			mods |= ui.Shift
		case "alt", "option":
			mods |= ui.Alt
		case "meta", "cmd", "command":
			mods |= ui.Super
		default:
			switch r := []rune(k); {
			case len(r) == 1 && r[0] >= 'a' && r[0] <= 'z':
				key = ui.KeyA + ui.Key(r[0]-'a')
			case len(r) == 1 && r[0] >= '0' && r[0] <= '9':
				key = ui.Key0 + ui.Key(r[0]-'0')
			default:
				named, ok := namedKeys[k]
				if !ok {
					return 0, 0, false
				}
				key = named
			}
			found = true
		}
	}
	return mods, key, found
}

type imageProps struct {
	Width, Height float32
	Loading       bool
	Failed        bool
	Alt           string
	Gallery       *imageGallery
}

type imageGallery struct {
	items  []viewerItem
	viewer imageViewer
}

func uiImageGroup(c *ui.Context, content func(g *imageGallery)) ui.Element {
	tk := tokensOf(c)
	root := ui.Box(c)
	g := ui.Local(root, "gallery", func() imageGallery { return imageGallery{} })
	g.items = g.items[:0]
	root.Children(func() { content(g) })
	g.viewer.build(c, tk, root, g.items)
	return root
}

var (
	fallbackLight = [2]ui.Color{ui.RGB(0xEC, 0xEC, 0xEC), ui.RGB(0xD7, 0xD7, 0xD7)}
	fallbackDark  = [2]ui.Color{ui.RGB(0x3B, 0x3B, 0x3B), ui.RGB(0x62, 0x62, 0x62)}
)

func uiImage(c *ui.Context, src ui.ImageSource, p imageProps) ui.Element {
	tk := tokensOf(c)
	w, h := p.Width, p.Height
	if p.Loading {
		return uiSkeleton(c, skeletonProps{Width: w, Height: h})
	}
	previewable := src != nil && !p.Failed
	var root ui.Element
	if previewable {
		root = ui.ButtonBase(c).Column().Cursor(ui.CursorZoomIn)
	} else {
		root = ui.Box(c)
	}
	root.Radius(tk.BorderRadius).Clip().Shrink(0).Unselectable()
	if p.Alt != "" {
		root.Label(p.Alt)
	}
	if w > 0 {
		root.Width(w)
	}
	if h > 0 {
		root.Height(h)
	}
	if w == 0 && h == 0 {
		root.MaxWidthPercent(100)
	}
	root.Border(1, tk.BorderSecondary).Background(tk.FillTertiary)

	v, index := ui.Local(root, "viewer", func() imageViewer { return imageViewer{} }), 0
	if p.Gallery != nil {
		v, index = &p.Gallery.viewer, len(p.Gallery.items)
		p.Gallery.items = append(p.Gallery.items, viewerItem{src: src, alt: p.Alt, bounds: root.Bounds()})
	}
	if previewable && root.Clicked() {
		v.openAt(index, root.Bounds())
	}
	root.Children(func() {
		switch {
		case p.Failed || src == nil:
			imageFallback(c, tk, w, h)
		default:
			img := ui.Image(c, src).Fit(ui.Cover)
			if w > 0 || h > 0 {
				img.Fill()
			} else {
				img.MaxWidthPercent(100)
			}
			if v.shown && v.index == index {
				img.Opacity(0)
			}
		}
	})
	if previewable && p.Gallery == nil {
		v.build(c, tk, root, []viewerItem{{src: src, alt: p.Alt, bounds: root.Bounds()}})
	}
	return root
}

func imageFallback(c *ui.Context, tk *uiTokens, w, h float32) {
	colors := fallbackLight
	if tk.Dark {
		colors = fallbackDark
	}
	box := ui.Row(c).Background(colors[0]).TextColor(colors[1]).Justify(ui.Center).AlignItems(ui.Center)
	if w > 0 || h > 0 {
		box.Fill()
	} else {
		box.Size(128, 128)
	}
	side := min(w, h)
	if side <= 0 {
		side = max(w, h, 128)
	}
	box.Children(func() { ui.Icon(c, iconImage()).FontSize(max(16, side*0.28)) })
}

const (
	viewerMargin     = 24
	viewerMaxScale   = 8
	wheelSensitivity = 0.002
	zoomStep         = 1.5
	doubleClickHold  = 250 * time.Millisecond
	switchFromScale  = 0.96
)

var (
	openSpring  = ui.Animation{Spring: ui.SpringOptions{Stiffness: 304.6, Damping: 29.67, Mass: 1}}
	resetSpring = ui.Animation{Spring: ui.SpringOptions{Stiffness: 300, Damping: 30, Mass: 1}}
	pushSpring  = ui.Animation{Spring: ui.SpringOptions{Stiffness: 304.6, Damping: 34.9, Mass: 1}}
	viewerFade  = ui.Animation{Duration: 150 * time.Millisecond, Ease: easeOut}
	chromeFade  = ui.Animation{Duration: 200 * time.Millisecond, Ease: easeOut, Delay: 150 * time.Millisecond}
)

type viewerItem struct {
	src    ui.ImageSource
	alt    string
	bounds ui.Rect
}

type imageViewer struct {
	open, shown  bool
	index        int
	from         ui.Rect
	viewport     ui.Rect
	scale, x, y  float32
	quarter      int
	snap         bool
	switched     bool
	pendingClick bool
}

func (v *imageViewer) openAt(index int, from ui.Rect) {
	*v = imageViewer{open: true, shown: true, index: index, from: from, viewport: v.viewport, scale: 1, snap: true}
}

func (v *imageViewer) close() { v.open, v.pendingClick = false, false }

func (v *imageViewer) showIndex(i int) {
	v.index, v.scale, v.x, v.y, v.quarter = i, 1, 0, 0, 0
	v.snap, v.switched, v.pendingClick = true, true, false
}

func (v *imageViewer) natural(src ui.ImageSource) (w, h float32) {
	w, h = ui.ImageSize(src)
	if v.quarter%2 != 0 {
		w, h = h, w
	}
	return w, h
}

func (v *imageViewer) fit(src ui.ImageSource) ui.Rect {
	nw, nh := v.natural(src)
	vw, vh := v.viewport.W, v.viewport.H
	if nw <= 0 || nh <= 0 || vw <= 0 || vh <= 0 {
		return v.from
	}
	s := max(min((vw-2*viewerMargin)/nw, (vh-2*viewerMargin)/nh, 1), 0)
	w, h := nw*s, nh*s
	return ui.Rect{X: (vw - w) / 2, Y: (vh - h) / 2, W: w, H: h}
}

func (v *imageViewer) zoomTo(fit ui.Rect, scale, ax, ay float32) {
	scale = min(max(scale, 1), viewerMaxScale)
	cx, cy := fit.X+fit.W/2, fit.Y+fit.H/2
	ratio := scale / v.scale
	v.x = ax - cx - ratio*(ax-cx-v.x)
	v.y = ay - cy - ratio*(ay-cy-v.y)
	v.scale = scale
	v.clampPan(fit)
}

func (v *imageViewer) clampPan(fit ui.Rect) {
	axis := func(at, center, size, view float32) float32 {
		scaled := size * v.scale
		if scaled <= view {
			return 0
		}
		return min(max(at, view-center-scaled/2), scaled/2-center)
	}
	v.x = axis(v.x, fit.X+fit.W/2, fit.W, v.viewport.W)
	v.y = axis(v.y, fit.Y+fit.H/2, fit.H, v.viewport.H)
}

func (v *imageViewer) reset() { v.scale, v.x, v.y = 1, 0, 0 }

func (v *imageViewer) rotate(by int) {
	v.quarter += by
	v.reset()
}

func lerpRect(a, b ui.Rect, t float32) ui.Rect {
	l := func(x, y float32) float32 { return x + (y-x)*t }
	return ui.Rect{X: l(a.X, b.X), Y: l(a.Y, b.Y), W: l(a.W, b.W), H: l(a.H, b.H)}
}

func (v *imageViewer) animate(host ui.Element, key string, to float32, a ui.Animation) float32 {
	if v.snap {
		return host.Jump(key, to)
	}
	return host.AnimateTo(key, to, a)
}

func (v *imageViewer) build(c *ui.Context, tk *uiTokens, host ui.Element, items []viewerItem) {
	if !v.shown || len(items) == 0 {
		return
	}
	v.index = min(max(v.index, 0), len(items)-1)
	item := items[v.index]
	if item.bounds.W > 0 {
		v.from = item.bounds
	}
	target := float32(0)
	if v.open && v.viewport.W > 0 {
		target = 1
	}
	progress := host.AnimateTo("viewer-open", target, openSpring)
	if !v.open && progress <= 0.001 {
		v.shown = false
		return
	}
	fade := host.AnimateTo("viewer-fade", boolToFloat(v.open), viewerFade)
	chrome := host.AnimateTo("viewer-chrome", boolToFloat(v.open && v.viewport.W > 0), chromeFade)
	if !v.open {
		chrome = 0
	}
	if v.pendingClick && host.AnimateTo("viewer-click", 0, ui.Animation{Duration: doubleClickHold}) == 0 {
		v.pendingClick = false
		if v.scale <= 1.01 {
			v.close()
		}
	}
	if v.switched {
		host.Jump("viewer-switch", 0)
		v.switched = false
	}
	arrival := host.AnimateTo("viewer-switch", 1, pushSpring)

	fit := v.fit(item.src)
	effW, effH := fit.W*v.scale, fit.H*v.scale
	uw, uh := effW, effH
	if v.quarter%2 != 0 {
		uw, uh = effH, effW
	}
	turn := float32(v.quarter * 90)
	if !v.open {
		turn = float32(math.Round(float64(v.quarter)/4) * 360)
	}
	w, h := v.animate(host, "viewer-w", uw, resetSpring), v.animate(host, "viewer-h", uh, resetSpring)
	x, y := v.animate(host, "viewer-x", v.x, resetSpring), v.animate(host, "viewer-y", v.y, resetSpring)
	angle := v.animate(host, "viewer-turn", turn, resetSpring)
	v.snap = false
	final := ui.Rect{X: v.viewport.W/2 + x - w/2, Y: v.viewport.H/2 + y - h/2, W: w, H: h}
	if v.viewport.W <= 0 {
		final = v.from
	}
	shown := lerpRect(v.from, final, progress)
	onScreen := ui.Rect{X: v.viewport.W/2 + v.x - effW/2, Y: v.viewport.H/2 + v.y - effH/2, W: effW, H: effH}

	ui.Overlay(c, func() {
		layer := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(tk.BgLayout.Alpha(0.9 * fade)).Cursor(ui.CursorDefault)
		if b := layer.Bounds(); b.W > 0 && b != v.viewport {
			v.viewport = b
			c.Invalidate()
		}
		v.keys(layer, len(items))
		layer.HandleInput(func(ev ui.InputEvent) bool {
			if !v.open {
				return false
			}
			switch ev.Kind {
			case ui.InputScroll:
				v.zoomTo(fit, v.scale*float32(math.Exp(float64(-ev.DY*wheelSensitivity))), ev.X, ev.Y)
			case ui.InputZoom:
				v.zoomTo(fit, v.scale*ev.Scale, ev.X, ev.Y)
			default:
				return false
			}
			v.snap = true
			return true
		})
		if dx, dy, held := layer.Dragged(); held && v.open && (dx != 0 || dy != 0) {
			v.x, v.y = v.x+dx, v.y+dy
			v.clampPan(fit)
			v.snap, v.pendingClick = true, false
		}

		usedChrome := false
		layer.Children(func() {
			k := switchFromScale + (1-switchFromScale)*arrival
			ui.Image(c.Key(v.index), item.src).Absolute().Left(shown.X).Top(shown.Y).Size(shown.W, shown.H).
				Fit(ui.Cover).Rotate(angle).Scale(k, k).Opacity(min(1, arrival*2)).
				Label("Image preview").IgnoreInput(true)
			ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Opacity(chrome).IgnoreInput(chrome == 0).
				Children(func() { usedChrome = v.chrome(c, tk, items, fit) })
		})

		if !v.open || usedChrome {
			return
		}
		px, py, _ := layer.PointerPosition()
		onImage := px >= onScreen.X && px < onScreen.X+onScreen.W && py >= onScreen.Y && py < onScreen.Y+onScreen.H
		switch {
		case !layer.Clicked():
		case !onImage:
			v.close()
		case v.pendingClick:
			v.pendingClick = false
			if v.scale > 1.01 {
				v.reset()
			} else {
				nw, _ := v.natural(item.src)
				v.zoomTo(fit, max(2, nw/fit.W), px, py)
			}
		default:
			v.pendingClick = true
			host.Jump("viewer-click", 1)
		}
	})
}

func (v *imageViewer) keys(layer ui.Element, n int) {
	if !v.open {
		return
	}
	switch {
	case layer.OverlayShortcut(0, ui.KeyEscape):
		if v.scale > 1.01 {
			v.reset()
		} else {
			v.close()
		}
	case layer.OverlayShortcut(0, ui.KeyLeft) && v.index > 0:
		v.showIndex(v.index - 1)
	case layer.OverlayShortcut(0, ui.KeyRight) && v.index < n-1:
		v.showIndex(v.index + 1)
	}
}

func viewerPill(c *ui.Context, tk *uiTokens) ui.Element {
	return ui.Row(c).Padding(4).Radius(999).Background(tk.BgLayout.Alpha(0.6)).AlignItems(ui.Center)
}

func (v *imageViewer) chrome(c *ui.Context, tk *uiTokens, items []viewerItem, fit ui.Rect) bool {
	used := false
	icon := func(svg *ui.SVG, title string) bool {
		b := ui.ButtonBase(c.Key(title)).Label(title).Size(36, 36).Radius(999).Justify(ui.Center).AlignItems(ui.Center).
			Cursor(ui.CursorPointer).Transition(listTransition)
		if b.Hovered() {
			b.Background(tk.FillTertiary).TextColor(tk.Text)
		} else {
			b.TextColor(tk.TextSecondary)
		}
		b.Children(func() { ui.Icon(c, svg).FontSize(20) })
		clicked := b.Clicked()
		used = used || clicked
		return clicked
	}
	item := items[v.index]
	nw, _ := v.natural(item.src)
	natural := float32(1)
	if fit.W > 0 {
		natural = nw / fit.W
	}
	cx, cy := v.viewport.W/2, v.viewport.H/2

	viewerPill(c, tk).Absolute().Top(16).Right(16).Children(func() {
		if icon(iconX(), "Close") {
			v.close()
		}
	})
	if n := len(items); n > 1 {
		ui.Row(c).Absolute().Top(16).Left(0).Right(0).Justify(ui.Center).IgnoreInput(true).Children(func() {
			viewerPill(c, tk).Padding(4, 12).Children(func() {
				ui.Text(c, strconv.Itoa(v.index+1)+" / "+strconv.Itoa(n)).FontSize(12).
					TextColor(tk.TextSecondary).FontFeatures("tnum")
			})
		})
		ui.Column(c).Absolute().Left(16).Top(0).Bottom(0).Justify(ui.Center).Children(func() {
			if v.index > 0 {
				viewerPill(c, tk).Children(func() {
					if icon(iconChevronLeft(), "Previous image") {
						v.showIndex(v.index - 1)
					}
				})
			}
		})
		ui.Column(c).Absolute().Right(16).Top(0).Bottom(0).Justify(ui.Center).Children(func() {
			if v.index < n-1 {
				viewerPill(c, tk).Children(func() {
					if icon(iconChevronRight(), "Next image") {
						v.showIndex(v.index + 1)
					}
				})
			}
		})
	}
	ui.Row(c).Absolute().Bottom(16).Left(0).Right(0).Justify(ui.Center).Children(func() {
		viewerPill(c, tk).Padding(4, 6).Children(func() {
			if icon(iconRotateCcw(), "Rotate left") {
				v.rotate(-1)
			}
			if icon(iconRotateCw(), "Rotate right") {
				v.rotate(1)
			}
			if icon(iconZoomOut(), "Zoom out") {
				v.zoomTo(fit, v.scale/zoomStep, cx, cy)
			}
			pct := int(math.Round(float64(v.scale / natural * 100)))
			ui.Row(c).Size(56, 36).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, strconv.Itoa(pct)+"%").FontSize(12).TextColor(tk.TextTertiary).FontFeatures("tnum")
			})
			if icon(iconZoomIn(), "Zoom in") {
				v.zoomTo(fit, v.scale*zoomStep, cx, cy)
			}
			if natural > 1.001 {
				actual := math.Abs(float64(v.scale-natural)) > 0.01
				title, svg := "Fit to screen", iconShrink()
				if actual {
					title, svg = "Actual size", iconExpand()
				}
				if icon(svg, title) {
					if actual {
						v.zoomTo(fit, natural, cx, cy)
					} else {
						v.reset()
					}
				}
			}
		})
	})
	return used
}
