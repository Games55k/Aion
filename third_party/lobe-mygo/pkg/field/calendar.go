package field

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type CalendarProps struct {
	// DisabledDate reports the days that cannot be chosen.
	DisabledDate func(day time.Time) bool
	// WeekStart is the first day of each week: Sunday when zero.
	WeekStart time.Weekday
	// NoToday hides the footer's Today button.
	NoToday bool
}

type calendarMode int

const (
	modeDate calendarMode = iota
	modeMonth
	modeYear
)

type calendarState struct {
	view, cursor, last time.Time
	mode               calendarMode
	init, moved        bool
}

const (
	calendarWidth = 288
	calendarBody  = calendarWidth - 24
)

// Calendar chooses a day into *value from a month of days, with buttons that turn the months and
// years and views to choose a month and a year. The arrows, Page Up and Page Down move between
// days while a day has the focus, and Enter chooses one.
func Calendar(c *ui.Context, value *time.Time, p CalendarProps) ui.Element {
	tk := theme.TokensOf(c)
	root := ui.Column(c).Width(calendarWidth).TextColor(tk.Text).Shrink(0)
	st := ui.Local(root, "calendar", func() calendarState { return calendarState{} })
	today := dateOf(c.Now())
	if !st.init || !value.Equal(st.last) {
		at := today
		if !value.IsZero() {
			at = dateOf(*value)
		}
		st.view, st.cursor, st.last, st.init = monthOf(at), at, *value, true
	}
	disabled := func(d time.Time) bool { return p.DisabledDate != nil && p.DisabledDate(d) }
	choose := func(d time.Time) {
		*value = d
		st.last, st.cursor, st.view = d, d, monthOf(d)
	}

	root.Children(func() {
		calendarHeader(c, tk, st)
		body := ui.Column(c).Padding(4, 12, 12, 12)
		body.Children(func() {
			switch st.mode {
			case modeMonth:
				pickGrid(c, tk, 3, 12, func(i int) (string, string, bool, bool) {
					m := time.Month(i + 1)
					return m.String()[:3], m.String()[:3], true, st.view.Month() == m
				}, func(i int) {
					st.view = time.Date(st.view.Year(), time.Month(i+1), 1, 0, 0, 0, 0, time.Local)
					st.mode = modeDate
				})
			case modeYear:
				start := st.view.Year()/10*10 - 1
				pickGrid(c, tk, 3, 12, func(i int) (string, string, bool, bool) {
					y := strconv.Itoa(start + i)
					return y, y, i > 0 && i < 11, st.view.Year() == start+i
				}, func(i int) {
					st.view = time.Date(start+i, st.view.Month(), 1, 0, 0, 0, 0, time.Local)
					st.mode = modeDate
				})
			default:
				dayGrid(c, tk, value, st, p, today, disabled, choose)
			}
		})
		if !p.NoToday && st.mode == modeDate {
			ui.Row(c).Justify(ui.Center).Padding(8, 12).FillWidth().Children(func() {
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(1).Background(tk.FillSecondary)
				b := ui.ButtonBase(c).Padding(2, 8).Radius(6).FontSize(13).Cursor(ui.CursorPointer).Transition(optionTransition)
				if disabled(today) {
					b.Disabled(true).Opacity(0.3).Cursor(ui.CursorNotAllowed)
				} else if b.Hovered() {
					b.Background(tk.FillTertiary)
				}
				if b.Clicked() {
					choose(today)
				}
				b.Children(func() { ui.Text(c, "Today") })
			})
		}
	})
	return root
}

func calendarHeader(c *ui.Context, tk *theme.Tokens, st *calendarState) {
	shift := func(years, months int) { st.view = st.view.AddDate(years, months, 0) }
	ui.Row(c).Gap(2).AlignItems(ui.Center).Padding(12, 12, 4, 12).Children(func() {
		switch st.mode {
		case modeYear:
			headerButton(c, tk, lucide.ChevronsLeft(), "Previous decade", func() { shift(-10, 0) })
		default:
			headerButton(c, tk, lucide.ChevronsLeft(), "Previous year", func() { shift(-1, 0) })
		}
		if st.mode == modeDate {
			headerButton(c, tk, lucide.ChevronLeft(), "Previous month", func() { shift(0, -1) })
		}
		ui.Row(c).Grow(1).MinWidth(0).Gap(4).Justify(ui.Center).FontWeight(600).FontSize(14).Children(func() {
			switch st.mode {
			case modeDate:
				viewButton(c, tk, st.view.Month().String()[:3], func() { st.mode = modeMonth })
				viewButton(c, tk, strconv.Itoa(st.view.Year()), func() { st.mode = modeYear })
			case modeMonth:
				viewButton(c, tk, strconv.Itoa(st.view.Year()), func() { st.mode = modeYear })
			case modeYear:
				start := st.view.Year() / 10 * 10
				ui.Text(c, strconv.Itoa(start)+"-"+strconv.Itoa(start+9)).Padding(2, 4)
			}
		})
		if st.mode == modeDate {
			headerButton(c, tk, lucide.ChevronRight(), "Next month", func() { shift(0, 1) })
		}
		switch st.mode {
		case modeYear:
			headerButton(c, tk, lucide.ChevronsRight(), "Next decade", func() { shift(10, 0) })
		default:
			headerButton(c, tk, lucide.ChevronsRight(), "Next year", func() { shift(1, 0) })
		}
	})
}

func headerButton(c *ui.Context, tk *theme.Tokens, glyph *ui.SVG, label string, do func()) {
	b := ui.ButtonBase(c.Key(label)).Label(label).Size(28, 28).Radius(14).Center().Shrink(0).
		Cursor(ui.CursorPointer).TextColor(tk.TextSecondary).Transition(optionTransition)
	if b.Hovered() {
		b.Background(tk.FillTertiary).TextColor(tk.Text)
	}
	if b.Clicked() {
		do()
	}
	b.Children(func() { ui.Icon(c, glyph).FontSize(16) })
}

func viewButton(c *ui.Context, tk *theme.Tokens, text string, do func()) {
	b := ui.ButtonBase(c.Key(text)).Padding(2, 4).Radius(6).Cursor(ui.CursorPointer).Transition(optionTransition)
	if b.Hovered() {
		b.TextColor(tk.Primary)
	}
	if b.Clicked() {
		do()
	}
	b.Children(func() { ui.Text(c, text) })
}

func dayGrid(c *ui.Context, tk *theme.Tokens, value *time.Time, st *calendarState, p CalendarProps, today time.Time, disabled func(time.Time) bool, choose func(time.Time)) {
	cell := float32(calendarBody) / 7
	start := st.view.AddDate(0, 0, -int((st.view.Weekday()-p.WeekStart+7)%7))
	grid := ui.Column(c)
	for _, k := range []struct {
		key  ui.Key
		days int
	}{{ui.KeyLeft, -1}, {ui.KeyRight, 1}, {ui.KeyUp, -7}, {ui.KeyDown, 7}} {
		if grid.Shortcut(0, k.key) {
			st.cursor = st.cursor.AddDate(0, 0, k.days)
			st.view, st.moved = monthOf(st.cursor), true
			start = st.view.AddDate(0, 0, -int((st.view.Weekday()-p.WeekStart+7)%7))
		}
	}
	for _, k := range []struct {
		key    ui.Key
		months int
	}{{ui.KeyPageUp, -1}, {ui.KeyPageDown, 1}} {
		if grid.Shortcut(0, k.key) {
			st.cursor = st.cursor.AddDate(0, k.months, 0)
			st.view, st.moved = monthOf(st.cursor), true
			start = st.view.AddDate(0, 0, -int((st.view.Weekday()-p.WeekStart+7)%7))
		}
	}
	grid.Children(func() {
		ui.Row(c).Children(func() {
			for i := range 7 {
				name := time.Weekday((int(p.WeekStart) + i) % 7).String()[:2]
				ui.Text(c.Key(i), name).Width(cell).Height(28).TextAlign(ui.Center).FontSize(12).TextColor(tk.TextTertiary).
					LineHeight(28.0 / 12)
			}
		})
		for w := range 6 {
			ui.Row(c.Key(w)).Children(func() {
				for i := range 7 {
					d := start.AddDate(0, 0, w*7+i)
					dayCell(c.Key(i), tk, d, cell, st, value, today, disabled(d), choose)
				}
			})
		}
	})
	st.moved = false
}

func dayCell(c *ui.Context, tk *theme.Tokens, d time.Time, width float32, st *calendarState, value *time.Time, today time.Time, off bool, choose func(time.Time)) {
	b := ui.ButtonBase(c).Label(d.Format("2006-01-02")).Size(width, 36).Center().FocusRing(false).Cursor(ui.CursorPointer)
	inView := d.Month() == st.view.Month()
	chosen := !value.IsZero() && sameDay(d, *value)
	if off {
		b.Disabled(true).Cursor(ui.CursorNotAllowed)
	}
	if st.moved && sameDay(d, st.cursor) {
		b.Focus()
	}
	if b.Clicked() {
		choose(d)
	}
	fg, bg := tk.TextQuaternary, ui.Transparent
	weight := 400
	switch {
	case off:
	case chosen && inView:
		fg, bg, weight = tk.BgContainer, tk.Text, 600
	case b.Hovered():
		bg = tk.FillTertiary
		if inView {
			fg = tk.Text
		}
	case inView:
		fg = tk.Text
	}
	if inView && sameDay(d, today) && weight < 600 {
		weight = 600
	}
	focused := b.Focused() && !chosen
	b.Children(func() {
		inner := ui.Row(c).Size(32, 32).Radius(16).Center().Background(bg).TextColor(fg).FontSize(13).
			FontWeight(weight).FontFeatures("tnum").Transition(optionTransition)
		if focused {
			inner.InsetShadow(0, 0, 0, 1, tk.Text)
		}
		inner.Children(func() { ui.Text(c, strconv.Itoa(d.Day())) })
	})
}

func pickGrid(c *ui.Context, tk *theme.Tokens, cols, n int, cellOf func(i int) (label, text string, inView, chosen bool), pick func(i int)) {
	width := float32(calendarBody) / float32(cols)
	for r := range (n + cols - 1) / cols {
		ui.Row(c.Key(r)).Children(func() {
			for k := range cols {
				i := r*cols + k
				label, text, inView, chosen := cellOf(i)
				b := ui.ButtonBase(c.Key(i)).Label(label).Size(width, 52).Center().Cursor(ui.CursorPointer)
				if b.Clicked() {
					pick(i)
				}
				fg, bg := tk.Text, ui.Transparent
				if !inView {
					fg = tk.TextQuaternary
				}
				switch {
				case chosen:
					fg, bg = tk.BgContainer, tk.Text
				case b.Hovered():
					bg = tk.FillTertiary
				}
				b.Children(func() {
					ui.Row(c).MinWidth(64).Height(36).PaddingX(8).Radius(18).Center().Background(bg).TextColor(fg).
						FontSize(14).FontWeight(map[bool]int{true: 600, false: 400}[chosen]).Transition(optionTransition).
						Children(func() { ui.Text(c, text) })
				})
			}
		})
	}
}

func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func monthOf(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local) }

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
