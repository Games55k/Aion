package field

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type DatePickerProps struct {
	// Placeholder shows while no day is chosen: "Select date" when empty.
	Placeholder string
	// Format writes the chosen day in the trigger, as time.Format takes it: "2006-01-02" when
	// empty.
	Format string
	// AllowClear adds a button that empties the value.
	AllowClear bool
	// DisabledDate reports the days that cannot be chosen, as the Calendar's.
	DisabledDate func(day time.Time) bool
	// WeekStart is the first day of each week: Sunday when zero.
	WeekStart time.Weekday
	// NoToday hides the calendar's Today button.
	NoToday bool
	// Variant picks the trigger's surface, as Input's.
	Variant InputVariant
	// Size sets the trigger's height (24, 32 or 40), padding and font size, as Input's.
	Size theme.Size
	// Disabled keeps the picker from opening.
	Disabled bool
}

// DatePicker chooses a day into *value, the zero time while none is chosen, from a Calendar that
// opens below its trigger and closes once a day is chosen. It returns the trigger.
func DatePicker(c *ui.Context, value *time.Time, p DatePickerProps) ui.Element {
	tk := theme.TokensOf(c)
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
		if p.Disabled {
			trigger.Disabled(true).Cursor(ui.CursorNotAllowed)
			box.Opacity(0.5)
		}
		clearable := p.AllowClear && !value.IsZero() && !p.Disabled
		trigger.Children(func() {
			if value.IsZero() {
				ui.Text(c, placeholder).TextColor(tk.TextPlaceholder).Grow(1).SingleLine()
			} else {
				ui.Text(c, value.Format(format)).FontFeatures("tnum").Grow(1).SingleLine()
			}
			icon := ui.Icon(c, lucide.Calendar()).TextColor(tk.TextTertiary).Shrink(0)
			if clearable {
				icon.Opacity(0)
			}
		})
		styleField(trigger, tk, p.Variant, false, *open || trigger.Focused())
		if clearable {
			x := ui.ButtonBase(c).Label("Clear").Absolute().Right(10).Top(0).Bottom(0).Width(16).Radius(8).Center().
				Cursor(ui.CursorPointer).TextColor(tk.TextTertiary)
			if x.Hovered() {
				x.TextColor(tk.TextSecondary)
			}
			if x.Clicked() {
				*value = time.Time{}
			}
			x.Children(func() { ui.Icon(c, lucide.CircleX()).FontSize(14) })
		}
		if p.Disabled {
			return
		}
		before := *value
		panel := overlay.Popover(c, trigger, open, overlay.PopoverProps{Placement: overlay.Bottom}, func() {
			Calendar(c, value, CalendarProps{DisabledDate: p.DisabledDate, WeekStart: p.WeekStart, NoToday: p.NoToday})
		})
		panel.Padding(0).Radius(16)
		if !value.Equal(before) {
			*open = false
		}
	})
	return trigger
}
