package nav

import (
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type PaginationAlign int

const (
	AlignNone PaginationAlign = iota
	AlignStart
	AlignCenter
	AlignEnd
)

type PaginationProps struct {
	// Total is the number of items paged through.
	Total int
	// PageSize is how many items a page holds: 10 when zero.
	PageSize int
	// ShowLessItems shows one page on each side of the current page instead of two, and the
	// ellipses jump 3 pages instead of 5.
	ShowLessItems bool
	// ShowTotal, if set, gives the text before the pages from the total and the 1-based range of
	// items on the current page.
	ShowTotal func(total, from, to int) string
	// HideOnSinglePage hides the pager when every item fits on one page.
	HideOnSinglePage bool
	// Disabled dims the pager and keeps its pages from being chosen.
	Disabled bool
	// Size is Middle (28 DIP items) or Small (24).
	Size theme.Size
	// Align stretches the pager across its parent and puts it at the start, center or end.
	Align PaginationAlign
}

type pageItem int

const (
	ellipsisPrev pageItem = -1
	ellipsisNext pageItem = -2
)

func pageItems(current, count, buffer int) []pageItem {
	if count <= 1 {
		return []pageItem{1}
	}
	items := []pageItem{1}
	start, end := max(2, current-buffer), min(count-1, current+buffer)
	if start > 2 {
		items = append(items, ellipsisPrev)
	}
	for p := start; p <= end; p++ {
		items = append(items, pageItem(p))
	}
	if end < count-1 {
		items = append(items, ellipsisNext)
	}
	return append(items, pageItem(count))
}

var pageTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: motion.EaseOut, Colors: true}

// Pagination pages through p.Total items, of which *current is the 1-based page shown: previous and
// next buttons, the first and last pages, the pages around the current one, and ellipses between
// that jump several pages and show a double chevron under the pointer.
func Pagination(c *ui.Context, current *int, p PaginationProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.PageSize
	if size <= 0 {
		size = 10
	}
	count := max(1, (p.Total+size-1)/size)
	*current = min(max(*current, 1), count)
	buffer, jump := 2, 5
	if p.ShowLessItems {
		buffer, jump = 1, 3
	}
	side, fontSize := float32(28), float32(13)
	if p.Size == theme.Small {
		side, fontSize = 24, 12
	}

	root := ui.Row(c).Gap(4).AlignItems(ui.Center).Label("Pagination")
	if p.HideOnSinglePage && count == 1 {
		return root
	}
	switch p.Align {
	case AlignStart:
		root.AlignSelf(ui.Stretch).Justify(ui.Start)
	case AlignCenter:
		root.AlignSelf(ui.Stretch).Justify(ui.Center)
	case AlignEnd:
		root.AlignSelf(ui.Stretch).Justify(ui.End)
	}
	goTo := func(page int) { *current = min(max(page, 1), count) }

	button := func(label string, disabled, selected bool) ui.Element {
		b := ui.ButtonBase(c).Label(label).Disabled(disabled).Height(side).MinWidth(side).PaddingX(6).
			Radius(tk.BorderRadius).Justify(ui.Center).AlignItems(ui.Center).FontSize(fontSize).
			FontWeight(500).Transition(pageTransition)
		hover := b.Hovered() && !disabled
		fg := tk.TextSecondary
		switch {
		case hover:
			fg = tk.Text
			b.Background(tk.FillTertiary)
		case selected:
			fg = tk.Text
			b.Background(tk.FillSecondary)
		case disabled:
			fg = tk.TextQuaternary
		}
		if selected {
			b.FontWeight(600)
		}
		if disabled {
			b.Cursor(ui.CursorDefault)
		} else {
			b.Cursor(ui.CursorPointer)
		}
		return b.TextColor(fg)
	}

	root.Children(func() {
		if p.ShowTotal != nil {
			from, to := (*current-1)*size+1, min(*current*size, p.Total)
			if p.Total == 0 {
				from = 0
			}
			ui.Text(c, p.ShowTotal(p.Total, from, to)).FontSize(12).TextColor(tk.TextTertiary).Margin(0, 8, 0, 0)
		}
		prev := button("Previous page", p.Disabled || *current == 1, false)
		prev.Children(func() { ui.Icon(c, lucide.ChevronLeft()).FontSize(fontSize + 3) })
		if prev.Clicked() {
			goTo(*current - 1)
		}
		for _, it := range pageItems(*current, count, buffer) {
			switch it {
			case ellipsisPrev, ellipsisNext:
				label, icon, target := fmt.Sprintf("Next %d pages", jump), lucide.ChevronsRight, *current+jump
				if it == ellipsisPrev {
					label, icon, target = fmt.Sprintf("Previous %d pages", jump), lucide.ChevronsLeft, *current-jump
				}
				e := ui.ButtonBase(c.Key(int(it))).Label(label).Disabled(p.Disabled).Size(side, side).
					Radius(tk.BorderRadius).Justify(ui.Center).AlignItems(ui.Center)
				hover := e.Hovered() && !p.Disabled
				if hover {
					e.TextColor(tk.Text)
				} else {
					e.TextColor(tk.TextQuaternary)
				}
				if p.Disabled {
					e.Cursor(ui.CursorDefault)
				} else {
					e.Cursor(ui.CursorPointer)
				}
				e.Children(func() {
					if hover {
						ui.Icon(c, icon()).FontSize(14)
					} else {
						ui.Text(c, "•••").FontSize(8).LetterSpacing(2).Margin(0, 0, 0, 2)
					}
				})
				if e.Clicked() {
					goTo(target)
				}
			default:
				page := int(it)
				b := button(strconv.Itoa(page), p.Disabled, page == *current)
				b.Children(func() { ui.Text(c, strconv.Itoa(page)).FontFeatures("tnum") })
				if b.Clicked() {
					goTo(page)
				}
			}
		}
		next := button("Next page", p.Disabled || *current == count, false)
		next.Children(func() { ui.Icon(c, lucide.ChevronRight()).FontSize(fontSize + 3) })
		if next.Clicked() {
			goTo(*current + 1)
		}
	})
	return root
}
