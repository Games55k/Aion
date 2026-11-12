package field

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.Local) }

func calendarTester(v *time.Time, p CalendarProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { Calendar(c, v, p) })
	}, 400, 460)
}

func TestCalendarShowsTheMonthAndPicksADay(t *testing.T) {
	tk := theme.Light()
	v := day(2026, time.November, 8)
	tt := calendarTester(&v, CalendarProps{})
	for _, s := range []string{"Nov", "2026", "Su", "Sa"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	sel := role(t, tt, ui.RoleButton, "2026-11-08")
	if sel.W < 36 || sel.H != 36 {
		t.Errorf("day cell %vx%v, want a 288 DIP panel's seventh by 36", sel.W, sel.H)
	}
	if got := testkit.PixelAt(tt, 400, sel.X+sel.W/2-12, sel.Y+sel.H/2); !testkit.Near(got, tk.Text) {
		t.Errorf("chosen day %+v, want filled with the text color", got)
	}
	b := role(t, tt, ui.RoleButton, "2026-11-15")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
	if !v.Equal(day(2026, time.November, 15)) {
		t.Errorf("value %v after clicking the 15th", v)
	}
	_, oct31 := tt.AccessByRole(ui.RoleButton, "2026-10-31")
	_, dec12 := tt.AccessByRole(ui.RoleButton, "2026-12-12")
	if oct31 || !dec12 {
		t.Errorf("Nov 2026 starts on a Sunday: want six weeks from Nov 1 to Dec 12, got Oct 31 %v Dec 12 %v", oct31, dec12)
	}
}

func TestCalendarNavigatesMonthsAndYears(t *testing.T) {
	v := day(2026, time.November, 8)
	tt := calendarTester(&v, CalendarProps{})
	click := func(name string) {
		b := role(t, tt, ui.RoleButton, name)
		tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
		tt.Frame()
	}
	click("Next month")
	if !tt.HasText("Dec") {
		t.Fatalf("after Next month: %q", tt.Texts())
	}
	click("Next year")
	if !tt.HasText("2027") {
		t.Fatalf("after Next year: %q", tt.Texts())
	}
	click("Dec")
	click("Mar")
	if _, ok := tt.AccessByRole(ui.RoleButton, "2027-03-01"); !ok {
		t.Fatalf("choosing Mar did not show March 2027: %q", tt.Texts())
	}
	click("2027")
	if !tt.HasText("2029") {
		t.Fatalf("year view %q", tt.Texts())
	}
	click("2024")
	if _, ok := tt.AccessByRole(ui.RoleButton, "2024-03-01"); !ok {
		t.Errorf("choosing 2024 did not show March 2024: %q", tt.Texts())
	}
	if !v.Equal(day(2026, time.November, 8)) {
		t.Errorf("navigating changed the value to %v", v)
	}
}

func TestCalendarDisabledDatesAndToday(t *testing.T) {
	v := day(2026, time.November, 8)
	tt := calendarTester(&v, CalendarProps{DisabledDate: func(d time.Time) bool { return d.Weekday() == time.Sunday }})
	b := role(t, tt, ui.RoleButton, "2026-11-15")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
	if !v.Equal(day(2026, time.November, 8)) {
		t.Errorf("a disabled Sunday was chosen: %v", v)
	}
	b = role(t, tt, ui.RoleButton, "Today")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
	if y, m, d := v.Date(); y != tt.Now().Year() || m != tt.Now().Month() || d != tt.Now().Day() {
		t.Errorf("Today chose %v, want %v", v, tt.Now())
	}
}

func TestCalendarKeys(t *testing.T) {
	v := day(2026, time.November, 30)
	tt := calendarTester(&v, CalendarProps{})
	b := role(t, tt, ui.RoleButton, "2026-11-30")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if !tt.HasText("Dec") {
		t.Errorf("the view did not follow the cursor into December: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !v.Equal(day(2026, time.December, 8)) {
		t.Errorf("Right, Down, Enter from Nov 30: %v, want Dec 8", v)
	}
	if !tt.HasText("Dec") {
		t.Errorf("the view did not follow the keys into December: %q", tt.Texts())
	}
}

func role(t *testing.T, tt *ui.Tester, r ui.Role, name string) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(r, name)
	if !ok {
		t.Fatalf("no %v %q; texts %q", r, name, tt.Texts())
	}
	return info.Bounds
}
