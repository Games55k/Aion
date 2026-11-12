package field

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestDatePickerOpensPicksAndClears(t *testing.T) {
	var v time.Time
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(260).Children(func() {
			DatePicker(c, &v, DatePickerProps{AllowClear: true}).Label("Due")
		})
	}, 400, 500)
	if !tt.HasText("Select date") {
		t.Fatalf("texts %q, want the placeholder", tt.Texts())
	}
	b := role(t, tt, ui.RoleButton, "Due")
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	now := tt.Now()
	name := time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	d := role(t, tt, ui.RoleButton, name)
	tt.ClickAt(d.X+d.W/2, d.Y+d.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if v.Format("2006-01-02") != name {
		t.Fatalf("value %v, want %s", v, name)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Today"); ok {
		t.Error("the calendar stays open after choosing a day")
	}
	if !tt.HasText(name) {
		t.Errorf("trigger shows %q, want %s", tt.Texts(), name)
	}
	c := role(t, tt, ui.RoleButton, "Clear")
	due := role(t, tt, ui.RoleButton, "Due")
	if d := (c.Y+c.H/2) - (due.Y+due.H/2); d > 1 || d < -1 {
		t.Fatalf("clear button not vertically centered in the date field: clear=%v field=%v delta=%.2f", c, due, d)
	}
	tt.ClickAt(c.X+c.W/2, c.Y+c.H/2)
	tt.Frame()
	if !v.IsZero() || !tt.HasText("Select date") {
		t.Errorf("after Clear: value %v texts %q", v, tt.Texts())
	}
}
