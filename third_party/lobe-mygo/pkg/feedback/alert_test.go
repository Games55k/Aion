package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func alertTester(title string, p AlertProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Padding(0).Children(func() {
			Alert(c, title, p)
			ui.Text(c, "after")
		})
	}, 400, 300)
}

func TestAlertSoftTintsByType(t *testing.T) {
	tk := theme.Light()
	for _, c := range []struct {
		typ    AlertType
		accent ui.Color
		share  float32
	}{{Info, tk.Info, 0.05}, {Success, tk.Success, 0.05}, {Warning, tk.Warning, 0.05}, {Error, tk.Error, 0.05}, {Secondary, tk.TextSecondary, 0.04}} {
		tt := alertTester("Saved", AlertProps{Type: c.typ})
		want := theme.Mix(tk.BgContainer, c.accent, c.share)
		if got := testkit.PixelAt(tt, 400, 200, 4); !testkit.Near(got, want) {
			t.Errorf("type %v: background %+v, want %+v", c.typ, got, want)
		}
	}
}

func TestAlertLaysOutIconTitleAndDescription(t *testing.T) {
	tt := alertTester("Saved", AlertProps{})
	if r, _ := tt.Find("Saved"); r.X != 12+16+10 || r.Y != 10 {
		t.Errorf("title at %+v, want after a 16 DIP icon at (38, 10)", r)
	}
	tt = alertTester("Saved", AlertProps{NoIcon: true})
	if r, _ := tt.Find("Saved"); r.X != 12 {
		t.Errorf("title without icon at x %v, want 12", r.X)
	}
	tt = alertTester("Saved", AlertProps{Description: "Your draft is safe."})
	title, _ := tt.Find("Saved")
	desc, ok := tt.Find("Your draft is safe.")
	if title.X != 14+18+10 || title.Y != 12 {
		t.Errorf("detailed title at %+v, want after an 18 DIP icon at (42, 12)", title)
	}
	if !ok || desc.X != title.X || desc.Y < title.Y+title.H {
		t.Errorf("description at %+v, want under the title %+v", desc, title)
	}
}

func TestAlertOutlinedAndPlainHaveNoFill(t *testing.T) {
	tk := theme.Light()
	for _, v := range []AlertVariant{Outlined, Plain} {
		tt := alertTester("Saved", AlertProps{Variant: v})
		if got := testkit.PixelAt(tt, 400, 200, 4); !testkit.Near(got, tk.BgLayout) {
			t.Errorf("variant %v: background %+v, want none", v, got)
		}
	}
	tt := alertTester("Saved", AlertProps{Variant: Outlined})
	if got, want := testkit.PixelAt(tt, 400, 200, 0.5), testkit.Over(tk.BgLayout, tk.BorderSecondary); !testkit.Near(got, want) {
		t.Errorf("outlined edge %+v, want BorderSecondary %+v", got, want)
	}
}

func TestAlertCloses(t *testing.T) {
	closed := false
	tt := alertTester("Saved", AlertProps{Closed: &closed})
	info, ok := tt.AccessByRole(ui.RoleButton, "Close")
	if !ok {
		t.Fatal("no Close button")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if !closed {
		t.Fatal("close did not set Closed")
	}
	if tt.HasText("Saved") {
		t.Error("closed alert still shows")
	}
	if r, _ := tt.Find("after"); r.Y != 0 {
		t.Errorf("content after a closed alert at y %v, want 0", r.Y)
	}
	if _, ok := alertTester("Saved", AlertProps{}).AccessByRole(ui.RoleButton, "Close"); ok {
		t.Error("alert without Closed shows a close button")
	}
}
