package feedback

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestResultStacksIconTitleAndExtra(t *testing.T) {
	tk := theme.Light()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Children(func() {
			Result(c, ResultProps{Status: ResultSuccess, Title: "Payment received", SubTitle: "A receipt is on its way.",
				Extra: func() { ui.Text(c, "Back home") }})
		})
	}, 400, 400)
	title, ok := tt.Find("Payment received")
	sub, ok2 := tt.Find("A receipt is on its way.")
	extra, ok3 := tt.Find("Back home")
	if !ok || !ok2 || !ok3 {
		t.Fatalf("texts %q", tt.Texts())
	}
	if title.Y < 16+72+12 || sub.Y < title.Y+title.H || extra.Y < sub.Y+sub.H+10 {
		t.Errorf("order: title %+v sub %+v extra %+v", title, sub, extra)
	}
	if title.H < 20*1.15 {
		t.Errorf("title %v tall, want 20 DIP type", title.H)
	}
	if got := testkit.PixelAt(tt, 400, 200-30, 16+36); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.Success.Alpha(0.12))) {
		t.Errorf("icon circle %+v, want 12%% success", got)
	}
}
