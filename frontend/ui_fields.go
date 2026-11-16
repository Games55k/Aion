package frontend

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

type inputVariant int

const (
	inputOutlined inputVariant = iota
	inputFilled
)

type inputProps struct {
	Variant     inputVariant
	Size        uiSize
	Placeholder string
	Invalid     bool
}

type textAreaProps struct {
	Placeholder string
	MinRows     int
	MaxRows     int
	Invalid     bool
}

var (
	fieldTransition  = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: easeOut, Colors: true, Shadows: true}
	optionTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: easeOut, Colors: true}
	arrowTurn        = ui.Animation{Duration: 150 * time.Millisecond, Ease: easeOut}
)

func uiInput(c *ui.Context, value *string, p inputProps) ui.Element {
	tk := tokensOf(c)
	root := fieldRow(c, tk, p.Size).Gap(8)
	var input ui.Element
	root.Children(func() {
		input = ui.TextInputBase(c, value).Grow(1).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder)
	})
	styleField(root, tk, p.Variant, p.Invalid, input.Focused())
	return input
}

func uiTextArea(c *ui.Context, value *string, p textAreaProps) ui.Element {
	tk := tokensOf(c)
	minRows, maxRows := p.MinRows, p.MaxRows
	if minRows <= 0 {
		minRows = 2
	}
	if maxRows <= 0 {
		maxRows = 8
	}
	root := ui.Column(c).PaddingX(12).PaddingY(8).Radius(tk.BorderRadius).
		FontSize(14).TextColor(tk.Text).Cursor(ui.CursorText).Transition(fieldTransition)
	var area ui.Element
	root.Children(func() {
		area = ui.TextAreaBase(c, value).Lines(minRows, maxRows).FocusRing(false).
			Placeholder(p.Placeholder).PlaceholderColor(tk.TextPlaceholder)
	})
	styleField(root, tk, inputOutlined, p.Invalid, area.Focused())
	return area
}

func styleField(root ui.Element, tk *uiTokens, v inputVariant, invalid, focused bool) {
	hovered := root.Hovered() && !focused && !root.IsDisabled()
	switch v {
	case inputFilled:
		bg := tk.FillTertiary
		if hovered {
			bg = tk.FillSecondary
		}
		root.Background(bg)
	default:
		root.Background(tk.BgContainer)
	}

	border := ui.Transparent
	if v == inputOutlined {
		border = tk.BorderSecondary
		if hovered {
			border = tk.Border
		}
		if focused && !invalid {
			border = tk.Primary
		}
	}
	if invalid && !hovered {
		border = tk.Error
	}
	root.Border(1, border)

	switch {
	case focused && invalid:
		root.Shadow(0, 0, 0, 2, tk.ErrorBg)
	case focused && v == inputOutlined:
		root.Shadow(0, 0, 0, 2, tk.PrimaryBg)
	}
}

func fieldRow(c *ui.Context, tk *uiTokens, size uiSize) ui.Element {
	return styleFieldRow(ui.Row(c), tk, size)
}

func styleFieldRow(e ui.Element, tk *uiTokens, size uiSize) ui.Element {
	minHeight, fontSize, radius, padding := float32(32), float32(14), tk.BorderRadius, float32(12)
	switch size {
	case sizeSmall:
		minHeight, fontSize, radius, padding = 24, 12, tk.BorderRadiusSM, 8
	case sizeLarge:
		minHeight, fontSize, radius = 40, 16, tk.BorderRadiusLG
	}
	return e.MinHeight(minHeight).PaddingX(padding).Radius(radius).AlignItems(ui.Center).
		FontSize(fontSize).TextColor(tk.Text).Cursor(ui.CursorText).Transition(fieldTransition)
}

type searchBarProps struct {
	Placeholder string
	ShortKey    string
	Size        uiSize
}

func uiSearchBar(c *ui.Context, query *string, p searchBarProps) ui.Element {
	tk := tokensOf(c)
	focusNow := false
	if p.ShortKey != "" {
		if mods, key, ok := hotkeyBinding(p.ShortKey, hotkeyAuto); ok && c.Shortcut(mods, key) {
			focusNow = true
		}
	}
	placeholder := p.Placeholder
	if placeholder == "" {
		placeholder = "Type keywords..."
	}

	root := fieldRow(c, tk, p.Size).Gap(4)
	var input ui.Element
	root.Children(func() {
		ui.Icon(c, iconSearch()).TextColor(tk.TextPlaceholder).Shrink(0).Margin(0, 4, 0, 0)
		input = ui.TextInputBase(c, query).Grow(1).FocusRing(false).
			Placeholder(placeholder).PlaceholderColor(tk.TextPlaceholder)
		if focusNow {
			input.Focus()
		}
		if p.ShortKey != "" {
			if mods, key, ok := hotkeyBinding(p.ShortKey, hotkeyAuto); ok {
				input.KeyShortcuts(mods, key)
			}
		}
		switch {
		case *query != "":
			clearBtn := ui.ButtonBase(c).Label("Clear").Size(16, 16).Radius(8).Center().Shrink(0).
				Cursor(ui.CursorPointer).TextColor(tk.TextTertiary)
			if clearBtn.Hovered() {
				clearBtn.TextColor(tk.TextSecondary)
			}
			if clearBtn.Clicked() {
				*query = ""
				input.Focus()
			}
			clearBtn.Children(func() { ui.Icon(c, iconCircleX()).FontSize(14) })
		case p.ShortKey != "" && !input.Focused():
			hk := uiHotkey(c, p.ShortKey, hotkeyProps{Compact: true, Size: p.Size}).TextColor(tk.TextDescription)
			if p.Size == sizeSmall {
				hk.Margin(0, -4, 0, 0)
			}
		}
	})
	styleField(root, tk, inputOutlined, false, input.Focused())
	return input
}

type selectOption struct {
	Value    string
	Label    string
	Icon     *ui.SVG
	Disabled bool
}

type selectProps struct {
	Options     []selectOption
	Placeholder string
	Size        uiSize
}

func uiSelect(c *ui.Context, selected *string, p selectProps) ui.Element {
	tk := tokensOf(c)
	minHeight, fontSize, radius, padX, padY := float32(32), float32(14), tk.BorderRadius, float32(11), float32(4)
	if p.Size == sizeSmall {
		minHeight, fontSize, radius, padX, padY = 24, 12, tk.BorderRadiusSM, 8, 0
	}
	sel := ui.SelectBase(c, selected)
	tr := sel.Trigger.MinHeight(minHeight).Padding(padY, padX).Gap(8).Radius(radius).FontSize(fontSize).
		AlignItems(ui.Center).TextColor(tk.Text).Cursor(ui.CursorPointer).Unselectable().Transition(fieldTransition)
	styleField(tr, tk, inputOutlined, false, sel.Open())

	var current *selectOption
	for i := range p.Options {
		if p.Options[i].Value == *selected {
			current = &p.Options[i]
		}
	}
	tr.Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			switch {
			case current != nil:
				if current.Icon != nil {
					ui.Icon(c, current.Icon).TextColor(tk.TextSecondary)
				}
				ui.Text(c, current.Label).SingleLine()
			default:
				ui.Text(c, p.Placeholder).TextColor(tk.TextPlaceholder).SingleLine()
			}
		})
		turn := float32(0)
		if sel.Open() {
			turn = 180
		}
		arrow := ui.Icon(c, iconChevronDown()).FontSize(16).TextColor(tk.TextSecondary).Shrink(0)
		arrow.Rotate(arrow.AnimateTo("turn", turn, arrowTurn))
	})

	width := tr.Bounds().W
	sel.Popup(func(panel ui.Element) {
		panel.Margin(4, 0, 0, 0).Radius(tk.BorderRadius).MinWidth(width).Clip().
			Background(tk.BgElevated).Transition(popupTransition()).AttachedOrigin()
		tk.Elevate(panel, boxShadowSecondary)
		ui.Scroll(c).MaxHeight(384).Padding(4).Children(func() {
			for _, o := range p.Options {
				renderSelectOption(c.Key(o.Value), tk, sel, o, o.Value == *selected)
			}
		})
	})
	return tr
}

func renderSelectOption(c *ui.Context, tk *uiTokens, sel ui.SelectParts[string], o selectOption, chosen bool) {
	item := sel.Item(o.Value).Disabled(o.Disabled).Padding(6, 12).MinHeight(32).Gap(8).AlignItems(ui.Center).
		Radius(tk.BorderRadiusSM).AlignSelf(ui.Stretch).FontSize(14).TextColor(tk.Text).Transition(optionTransition)
	switch {
	case o.Disabled:
		item.TextColor(tk.TextDisabled).Cursor(ui.CursorNotAllowed)
	case item.Highlighted():
		item.Background(tk.FillTertiary).Cursor(ui.CursorPointer)
	default:
		item.Cursor(ui.CursorPointer)
	}
	item.Children(func() {
		if o.Icon != nil {
			ui.Icon(c, o.Icon).FontSize(14).TextColor(tk.TextSecondary)
		}
		ui.Text(c, o.Label).Grow(1).SingleLine()
		if chosen {
			ui.Icon(c, iconCheck()).FontSize(14).TextColor(tk.Primary).Margin(0, 0, 0, 8)
		}
	})
}

type datePickerProps struct {
	Placeholder  string
	Format       string
	AllowClear   bool
	DisabledDate func(day time.Time) bool
	WeekStart    time.Weekday
	NoToday      bool
	Size         uiSize
}

func uiDatePicker(c *ui.Context, value *time.Time, p datePickerProps) ui.Element {
	tk := tokensOf(c)
	format, placeholder := p.Format, p.Placeholder
	if format == "" {
		format = "2006-01-02"
	}
	if placeholder == "" {
		placeholder = "Select date"
	}
	box := ui.Row(c).FillWidth().AlignItems(ui.Center)
	open := ui.Local(box, "open", func() bool { return false })
	var trigger ui.Element
	box.Children(func() {
		trigger = styleFieldRow(ui.ButtonBase(c), tk, p.Size).Grow(1).Gap(8).Cursor(ui.CursorPointer).FocusRing(false)
		clearable := p.AllowClear && !value.IsZero()
		trigger.Children(func() {
			if value.IsZero() {
				ui.Text(c, placeholder).TextColor(tk.TextPlaceholder).Grow(1).SingleLine()
			} else {
				ui.Text(c, value.Format(format)).FontFeatures("tnum").Grow(1).SingleLine()
			}
			icon := ui.Icon(c, iconCalendar()).TextColor(tk.TextTertiary).Shrink(0)
			if clearable {
				icon.Opacity(0)
			}
		})
		styleField(trigger, tk, inputOutlined, false, *open || trigger.Focused())
		if clearable {
			x := ui.ButtonBase(c).Label("Clear").Absolute().Right(10).Top(0).Bottom(0).Width(16).Radius(8).Center().
				Cursor(ui.CursorPointer).TextColor(tk.TextTertiary)
			if x.Hovered() {
				x.TextColor(tk.TextSecondary)
			}
			if x.Clicked() {
				*value = time.Time{}
			}
			x.Children(func() { ui.Icon(c, iconCircleX()).FontSize(14) })
		}
		before := *value
		panel := uiPopover(c, trigger, open, popoverProps{Placement: placementBottom}, func() {
			uiCalendar(c, value, calendarProps{DisabledDate: p.DisabledDate, WeekStart: p.WeekStart, NoToday: p.NoToday})
		})
		panel.Padding(0).Radius(16)
		if !value.Equal(before) {
			*open = false
		}
	})
	return trigger
}

type calendarProps struct {
	DisabledDate func(day time.Time) bool
	WeekStart    time.Weekday
	NoToday      bool
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

func uiCalendar(c *ui.Context, value *time.Time, p calendarProps) ui.Element {
	tk := tokensOf(c)
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

func calendarHeader(c *ui.Context, tk *uiTokens, st *calendarState) {
	shift := func(years, months int) { st.view = st.view.AddDate(years, months, 0) }
	ui.Row(c).Gap(2).AlignItems(ui.Center).Padding(12, 12, 4, 12).Children(func() {
		switch st.mode {
		case modeYear:
			headerButton(c, tk, iconChevronsLeft(), "Previous decade", func() { shift(-10, 0) })
		default:
			headerButton(c, tk, iconChevronsLeft(), "Previous year", func() { shift(-1, 0) })
		}
		if st.mode == modeDate {
			headerButton(c, tk, iconChevronLeft(), "Previous month", func() { shift(0, -1) })
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
			headerButton(c, tk, iconChevronRight(), "Next month", func() { shift(0, 1) })
		}
		switch st.mode {
		case modeYear:
			headerButton(c, tk, iconChevronsRight(), "Next decade", func() { shift(10, 0) })
		default:
			headerButton(c, tk, iconChevronsRight(), "Next year", func() { shift(1, 0) })
		}
	})
}

func headerButton(c *ui.Context, tk *uiTokens, glyph *ui.SVG, label string, do func()) {
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

func viewButton(c *ui.Context, tk *uiTokens, text string, do func()) {
	b := ui.ButtonBase(c.Key(text)).Padding(2, 4).Radius(6).Cursor(ui.CursorPointer).Transition(optionTransition)
	if b.Hovered() {
		b.TextColor(tk.Primary)
	}
	if b.Clicked() {
		do()
	}
	b.Children(func() { ui.Text(c, text) })
}

func dayGrid(c *ui.Context, tk *uiTokens, value *time.Time, st *calendarState, p calendarProps, today time.Time, disabled func(time.Time) bool, choose func(time.Time)) {
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

func dayCell(c *ui.Context, tk *uiTokens, d time.Time, width float32, st *calendarState, value *time.Time, today time.Time, off bool, choose func(time.Time)) {
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

func pickGrid(c *ui.Context, tk *uiTokens, cols, n int, cellOf func(i int) (label, text string, inView, chosen bool), pick func(i int)) {
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
