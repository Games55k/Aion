package display

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func viewerPill(c *ui.Context, tk *theme.Tokens) ui.Element {
	return ui.Row(c).Padding(4).Radius(999).Background(tk.BgLayout.Alpha(0.6)).AlignItems(ui.Center)
}

func (v *imageViewer) chrome(c *ui.Context, tk *theme.Tokens, items []viewerItem, fit ui.Rect) bool {
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
		if icon(lucide.X(), "Close") {
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
					if icon(lucide.ChevronLeft(), "Previous image") {
						v.showIndex(v.index - 1)
					}
				})
			}
		})
		ui.Column(c).Absolute().Right(16).Top(0).Bottom(0).Justify(ui.Center).Children(func() {
			if v.index < n-1 {
				viewerPill(c, tk).Children(func() {
					if icon(lucide.ChevronRight(), "Next image") {
						v.showIndex(v.index + 1)
					}
				})
			}
		})
	}
	ui.Row(c).Absolute().Bottom(16).Left(0).Right(0).Justify(ui.Center).Children(func() {
		viewerPill(c, tk).Padding(4, 6).Children(func() {
			if icon(lucide.RotateCcw(), "Rotate left") {
				v.rotate(-1)
			}
			if icon(lucide.RotateCw(), "Rotate right") {
				v.rotate(1)
			}
			if icon(lucide.ZoomOut(), "Zoom out") {
				v.zoomTo(fit, v.scale/zoomStep, cx, cy)
			}
			pct := int(math.Round(float64(v.scale / natural * 100)))
			ui.Row(c).Size(56, 36).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Text(c, strconv.Itoa(pct)+"%").FontSize(12).TextColor(tk.TextTertiary).FontFeatures("tnum")
			})
			if icon(lucide.ZoomIn(), "Zoom in") {
				v.zoomTo(fit, v.scale*zoomStep, cx, cy)
			}
			if natural > 1.001 {
				actual := math.Abs(float64(v.scale-natural)) > 0.01
				title, svg := "Fit to screen", lucide.Shrink()
				if actual {
					title, svg = "Actual size", lucide.Expand()
				}
				if icon(svg, title) {
					if actual {
						v.zoomTo(fit, natural, cx, cy)
					} else {
						v.reset()
					}
				}
			}
			if item.onDownload != nil && icon(lucide.Download(), "Download") {
				item.onDownload(item.src)
			}
		})
	})
	return used
}
