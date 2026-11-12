package display

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ImageVariant int

const (
	ImageFilled ImageVariant = iota
	ImageOutlined
	ImageBorderless
)

type ImageProps struct {
	// Width and Height size the box; Size sets both. The image shows at its own size, at most its
	// parent's width, when none is set.
	Width, Height, Size float32
	// Contain fits the whole image inside the box instead of cropping it to cover the box.
	Contain bool
	// Variant picks the frame: Filled tints the box behind the image and borders it, Outlined
	// borders it on the container color, Borderless has neither.
	Variant ImageVariant
	// Loading shows a skeleton in the box, as while the image downloads.
	Loading bool
	// Failed shows a placeholder picture in the box, as when the image could not be loaded.
	Failed bool
	// NoPreview keeps a click from opening the image in the preview viewer.
	NoPreview bool
	// Alt names the image for assistive technology.
	Alt string
	// Actions builds buttons at the top right of the image, shown while the pointer is on it
	// unless AlwaysShowActions.
	Actions           func()
	AlwaysShowActions bool
	// OnDownload, if set, adds a Download button to the viewer's toolbar that calls it with the
	// image shown. Saving is the app's: the original file is its own, and a native app can ask
	// where with mygo's Dialog.Save, which a browser does not have.
	OnDownload func(src ui.ImageSource)
	// Gallery, from ImageGroup, opens the image in a viewer shared with the group's other images,
	// which steps between them.
	Gallery *ImageGallery
}

// ImageGallery is the state ImageGroup shares between its images.
type ImageGallery struct {
	items  []viewerItem
	viewer imageViewer
}

// ImageGroup builds content, whose Images given the gallery open in one viewer that steps between
// them with its arrows, Left and Right, counting where it is. It returns the box content is built
// in, for the caller to lay out.
func ImageGroup(c *ui.Context, content func(g *ImageGallery)) ui.Element {
	tk := theme.TokensOf(c)
	root := ui.Box(c)
	g := ui.Local(root, "gallery", func() ImageGallery { return ImageGallery{} })
	g.items = g.items[:0]
	root.Children(func() { content(g) })
	g.viewer.build(c, tk, root, g.items)
	return root
}

var (
	fallbackLight = [2]ui.Color{ui.RGB(0xEC, 0xEC, 0xEC), ui.RGB(0xD7, 0xD7, 0xD7)}
	fallbackDark  = [2]ui.Color{ui.RGB(0x3B, 0x3B, 0x3B), ui.RGB(0x62, 0x62, 0x62)}
)

// Image shows src in a rounded frame and, clicked, opens it in a viewer over the window that
// zooms with the wheel, a pinch, a double click or its toolbar and pans when dragged. Give src
// nil with Loading or Failed while there is no picture to show.
func Image(c *ui.Context, src ui.ImageSource, p ImageProps) ui.Element {
	tk := theme.TokensOf(c)
	w, h := p.Width, p.Height
	if w == 0 {
		w = p.Size
	}
	if h == 0 {
		h = p.Size
	}
	if p.Loading {
		return Skeleton(c, SkeletonProps{Width: w, Height: h})
	}
	previewable := !p.NoPreview && src != nil && !p.Failed
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
	switch p.Variant {
	case ImageOutlined:
		root.Border(1, tk.BorderSecondary).Background(tk.BgContainer)
	case ImageBorderless:
	default:
		root.Border(1, tk.BorderSecondary).Background(tk.FillTertiary)
	}

	v, index := ui.Local(root, "viewer", func() imageViewer { return imageViewer{} }), 0
	if p.Gallery != nil {
		v, index = &p.Gallery.viewer, len(p.Gallery.items)
		p.Gallery.items = append(p.Gallery.items, viewerItem{src: src, alt: p.Alt, onDownload: p.OnDownload, bounds: root.Bounds()})
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
			if p.Contain {
				img.Fit(ui.Contain)
			}
			if w > 0 || h > 0 {
				img.Fill()
			} else {
				img.MaxWidthPercent(100)
			}
			if v.shown && v.index == index {
				img.Opacity(0)
			}
		}
		if p.Actions != nil && (p.AlwaysShowActions || root.Hovered() || root.FocusWithin()) {
			ui.Row(c).Absolute().Top(0).Right(0).ZIndex(1).Children(p.Actions)
		}
	})
	if previewable && p.Gallery == nil {
		v.build(c, tk, root, []viewerItem{{src: src, alt: p.Alt, onDownload: p.OnDownload, bounds: root.Bounds()}})
	}
	return root
}

func imageFallback(c *ui.Context, tk *theme.Tokens, w, h float32) {
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
	box.Children(func() { ui.Icon(c, lucide.Image()).FontSize(max(16, side*0.28)) })
}
