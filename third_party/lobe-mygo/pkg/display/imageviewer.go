package display

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

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
	viewerFade  = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.EaseOut}
	chromeFade  = ui.Animation{Duration: 200 * time.Millisecond, Ease: motion.EaseOut, Delay: 150 * time.Millisecond}
)

type viewerItem struct {
	src        ui.ImageSource
	alt        string
	onDownload func(ui.ImageSource)
	bounds     ui.Rect
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

// The image is laid out unrotated, at its own aspect, and turned about its center: its size and
// the turn animate apart, so a quarter turn refits the picture as it turns instead of jumping to
// the rotated box first.
func (v *imageViewer) build(c *ui.Context, tk *theme.Tokens, host ui.Element, items []viewerItem) {
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
	fade := host.AnimateTo("viewer-fade", b2f(v.open), viewerFade)
	chrome := host.AnimateTo("viewer-chrome", b2f(v.open && v.viewport.W > 0), chromeFade)
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

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
