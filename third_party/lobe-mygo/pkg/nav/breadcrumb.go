package nav

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type BreadcrumbItem struct {
	// Title is the item's text.
	Title string
	// OnClick makes the item a button that runs it; the last item, the current page, never is.
	OnClick func()
}

type BreadcrumbProps struct {
	// Items are the path to the current page, which is the last.
	Items []BreadcrumbItem
	// Separator is the text between items, as "/"; SeparatorIcon shows when it is empty, and a
	// chevron when both are.
	Separator     string
	SeparatorIcon *ui.SVG
}

// Breadcrumb shows the path to the current page, in description color with the current page last
// in text color, each item a button when it has OnClick.
func Breadcrumb(c *ui.Context, p BreadcrumbProps) ui.Element {
	tk := theme.TokensOf(c)
	root := ui.Row(c).Label("breadcrumb").Gap(4).AlignItems(ui.Center).Wrap().FontSize(14).LineHeight(1.5)
	root.Children(func() {
		for i, it := range p.Items {
			last := i == len(p.Items)-1
			item := ui.Row(c.Key(i)).Gap(4).AlignItems(ui.Center)
			item.Children(func() {
				switch {
				case last:
					ui.Text(c, it.Title).TextColor(tk.Text).FontWeight(500)
				case it.OnClick != nil:
					b := ui.ButtonBase(c).Radius(4).Cursor(ui.CursorPointer).Transition(pageTransition)
					if b.Hovered() {
						b.TextColor(tk.Text)
					} else {
						b.TextColor(tk.TextDescription)
					}
					b.Children(func() { ui.Text(c, it.Title) })
					if b.Clicked() {
						it.OnClick()
					}
				default:
					ui.Text(c, it.Title).TextColor(tk.TextDescription)
				}
				if !last {
					breadcrumbSeparator(c, tk, p)
				}
			})
		}
	})
	return root
}

func breadcrumbSeparator(c *ui.Context, tk *theme.Tokens, p BreadcrumbProps) {
	if p.Separator != "" {
		ui.Text(c, p.Separator).TextColor(tk.TextQuaternary)
		return
	}
	icon := p.SeparatorIcon
	if icon == nil {
		icon = lucide.ChevronRight()
	}
	ui.Icon(c, icon).FontSize(14).TextColor(tk.TextQuaternary)
}
