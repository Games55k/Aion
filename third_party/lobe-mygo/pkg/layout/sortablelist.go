package layout

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SortableVariant int

const (
	SortableBorderless SortableVariant = iota
	SortableFilled
	SortableOutlined
)

type SortableListProps struct {
	// Gap is the space between rows: 8 DIPs when zero.
	Gap float32
	// Variant gives each row a surface: Filled tints it and Outlined borders it, both padded.
	Variant SortableVariant
	// Overlay builds what follows the pointer for the row at index while it is dragged: the row
	// itself when nil.
	Overlay func(index int)
}

type sortState struct {
	active, over int
	keyboard     bool
	dy           float32
	bounds       []ui.Rect
	reset        bool
	dropping     string
	dropTop      float32
}

var (
	sortShift  = ui.Animation{Duration: 250 * time.Millisecond, Ease: motion.EaseOut}
	sortSettle = ui.Animation{Duration: 250 * time.Millisecond, Ease: motion.EaseOut}
)

// SortableList shows *items as rows that drag into a new order by the handle each row builds: row
// gets an item and a func building its handle where it goes. key names an item for as long as it
// lives. The handle reorders from the keyboard too: Space picks the row up and drops it, the arrow
// keys move it, Escape puts it back.
func SortableList[T any](c *ui.Context, items *[]T, key func(T) string, p SortableListProps, row func(item T, handle func())) ui.Element {
	tk := theme.TokensOf(c)
	gap := p.Gap
	if gap <= 0 {
		gap = 8
	}
	list := ui.Column(c).Gap(gap).AlignSelf(ui.Stretch)
	st := ui.Local(list, "sortable", func() sortState { return sortState{active: -1} })
	n := len(*items)
	if st.active >= n {
		st.active = -1
	}
	if st.active >= 0 && !st.keyboard && len(st.bounds) == n {
		st.over = sortOver(st.bounds, st.active, st.dy)
	}
	reset := st.reset
	st.reset = false
	bounds := make([]ui.Rect, n)
	released := false
	list.Children(func() {
		for i, it := range *items {
			k := key(it)
			r := sortRow(c.Key(k), tk, p.Variant)
			shift := sortShiftOf(st, i, gap)
			if reset {
				r.Translate(0, r.Jump("shift", 0))
			} else {
				r.Translate(0, r.AnimateTo("shift", shift, sortShift))
			}
			if i == st.active || k == st.dropping {
				r.Opacity(0.4)
			}
			bounds[i] = r.LayoutBounds()
			r.Children(func() {
				row(it, func() {
					if sortHandle(c, st, i, n) {
						released = true
					}
				})
			})
		}
	})
	st.bounds = bounds
	sortOverlay(c, tk, list, st, items, key, p, row)
	if released {
		sortDrop(c, st, items, key, gap)
	}
	return list
}

func sortRow(c *ui.Context, tk *theme.Tokens, v SortableVariant) ui.Element {
	r := ui.Row(c).Gap(4).AlignItems(ui.Center).Radius(tk.BorderRadius).Clip()
	switch v {
	case SortableFilled:
		r.Background(tk.FillTertiary).Padding(4, 16, 4, 4)
	case SortableOutlined:
		r.Border(1, tk.BorderSecondary).Background(tk.BgContainer).Padding(4, 16, 4, 4)
	}
	return r
}

func sortOver(bounds []ui.Rect, active int, dy float32) int {
	center := bounds[active].Y + bounds[active].H/2 + dy
	best := active
	for i, b := range bounds {
		if abs(b.Y+b.H/2-center) < abs(bounds[best].Y+bounds[best].H/2-center) {
			best = i
		}
	}
	return best
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func sortSlotTop(st *sortState) float32 {
	a, o := st.bounds[st.active], st.bounds[st.over]
	if st.over > st.active {
		return o.Y + o.H - a.H
	}
	return o.Y
}

func sortShiftOf(st *sortState, i int, gap float32) float32 {
	if st.active < 0 || len(st.bounds) <= max(st.active, st.over, i) {
		return 0
	}
	room := st.bounds[st.active].H + gap
	switch {
	case i == st.active:
		return sortSlotTop(st) - st.bounds[st.active].Y
	case st.active < i && i <= st.over:
		return -room
	case st.over <= i && i < st.active:
		return room
	}
	return 0
}

func sortHandle(c *ui.Context, st *sortState, i, n int) bool {
	h := action.ActionIcon(c, lucide.GripVertical(), action.ActionIconProps{Size: theme.Small}).Label("Reorder").Cursor(ui.CursorGrab)
	_, dy, held := h.Dragged()
	switch {
	case held && st.active != i:
		st.active, st.over, st.keyboard, st.dy = i, i, false, 0
		fallthrough
	case held:
		st.dy += dy
		h.Cursor(ui.CursorGrabbing)
		c.Invalidate()
		return false
	case st.active == i && !st.keyboard:
		return true
	}
	if !h.Focused() {
		return false
	}
	lifted := st.active == i && st.keyboard
	switch {
	case h.Shortcut(0, ui.KeySpace) || h.Shortcut(0, ui.KeyEnter):
		if lifted {
			return true
		}
		if st.active < 0 {
			st.active, st.over, st.keyboard, st.dy = i, i, true, 0
		}
	case lifted && h.Shortcut(0, ui.KeyDown):
		st.over = min(st.over+1, n-1)
	case lifted && h.Shortcut(0, ui.KeyUp):
		st.over = max(st.over-1, 0)
	case lifted && h.Shortcut(0, ui.KeyEscape):
		st.active = -1
	}
	return false
}

func sortOverlay[T any](c *ui.Context, tk *theme.Tokens, list ui.Element, st *sortState, items *[]T, key func(T) string, p SortableListProps, row func(item T, handle func())) {
	idx := st.active
	if idx < 0 && st.dropping != "" {
		for i, it := range *items {
			if key(it) == st.dropping {
				idx = i
			}
		}
	}
	if idx < 0 || idx >= len(st.bounds) || idx >= len(*items) {
		st.dropping = ""
		return
	}
	b := st.bounds[idx]
	var top float32
	switch {
	case st.dropping != "":
		top = list.AnimateTo("overlay", st.dropTop, sortSettle)
		if top == st.dropTop {
			st.dropping = ""
			c.Invalidate()
		}
	case st.keyboard:
		top = list.AnimateTo("overlay", sortSlotTop(st), sortShift)
	default:
		top = list.Jump("overlay", max(0, b.Y+st.dy))
	}
	it := (*items)[idx]
	ui.Overlay(c, func() {
		r := sortRow(c.Key("sortable-overlay"), tk, p.Variant).Absolute().Left(b.X).Top(top).Width(b.W).IgnoreInput(true)
		r.Children(func() {
			if p.Overlay != nil {
				p.Overlay(idx)
				return
			}
			row(it, func() {
				ui.Box(c).Children(func() {
					action.ActionIcon(c, lucide.GripVertical(), action.ActionIconProps{Size: theme.Small})
				})
			})
		})
	})
}

func sortDrop[T any](c *ui.Context, st *sortState, items *[]T, key func(T) string, gap float32) {
	from, to := st.active, st.over
	if len(st.bounds) > max(from, to) {
		st.dropTop = sortSlotTop(st)
	}
	st.dropping = key((*items)[from])
	st.active, st.keyboard, st.dy = -1, false, 0
	if from != to {
		moved := (*items)[from]
		s := append((*items)[:from:from], (*items)[from+1:]...)
		s = append(s[:to], append([]T{moved}, s[to:]...)...)
		*items = s
		st.reset = true
	}
	c.Invalidate()
}
