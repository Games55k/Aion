package field

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestInputOTPFillsCells(t *testing.T) {
	v := ""
	var done []string
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			InputOTP(c, &v, InputOTPProps{Length: 4, OnComplete: func(s string) { done = append(done, s) }}).Label("code")
		})
	}, 400, 200)
	tt.Click("code")
	tt.Type("1a2")
	tt.Frame()
	if v != "12" {
		t.Fatalf("value %q after typing 1a2, want digits only", v)
	}
	one, ok1 := tt.Find("1")
	two, ok2 := tt.Find("2")
	if !ok1 || !ok2 || two.X-one.X < 32 {
		t.Errorf("1 at %+v, 2 at %+v: want each digit in its own 32 DIP cell", one, two)
	}
	tt.Type("345")
	tt.Frame()
	if v != "1234" || len(done) != 1 || done[0] != "1234" {
		t.Errorf("value %q done %q, want 1234 once", v, done)
	}
}
