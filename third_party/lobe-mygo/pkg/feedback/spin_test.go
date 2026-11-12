package feedback

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestSpinTurns(t *testing.T) {
	for _, v := range []SpinVariant{SpinDefault, SpinNetwork} {
		tt := ui.NewTester(testkit.Centered(func(c *ui.Context) { Spin(c, SpinProps{Variant: v, Size: theme.Large}, nil) }), 100, 100)
		tt.Frame()
		a := tt.Image()
		tt.Advance(300 * time.Millisecond)
		tt.Frame()
		if string(a.Pix) == string(tt.Image().Pix) {
			t.Errorf("variant %v did not move in 300 ms", v)
		}
		if !tt.FramePending() {
			t.Errorf("variant %v stopped asking for frames", v)
		}
	}
}

func TestSpinOverContent(t *testing.T) {
	spinning := true
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(200).Children(func() {
			Spin(c, SpinProps{Tip: "Loading", Spinning: &spinning}, func() { ui.Text(c, "Report").Padding(40) })
		})
	}, 240, 200)
	if !tt.HasText("Report") || !tt.HasText("Loading") {
		t.Fatalf("texts %q, want the content under the tip", tt.Texts())
	}
	spinning = false
	tt.Frame()
	if tt.HasText("Loading") || !tt.HasText("Report") {
		t.Errorf("not spinning: %q, want the content alone", tt.Texts())
	}
}

func TestSpinSizes(t *testing.T) {
	for _, c := range []struct {
		p    SpinProps
		want float32
	}{{SpinProps{Size: theme.Small}, 14}, {SpinProps{}, 20}, {SpinProps{Size: theme.Large}, 32}, {SpinProps{CustomSize: 48}, 48}} {
		tt := ui.NewTester(testkit.Centered(func(ctx *ui.Context) { Spin(ctx, c.p, nil).Label("spin") }), 100, 100)
		info, ok := tt.AccessByRole(ui.RoleStatus, "spin")
		if !ok || info.Bounds.W != c.want || info.Bounds.H != c.want {
			t.Errorf("%+v: %+v, want %v square", c.p, info.Bounds, c.want)
		}
	}
}

func TestSpinPercentRing(t *testing.T) {
	tk := theme.Light()
	pct := float32(30)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(10).Children(func() { Spin(c, SpinProps{CustomSize: 48, Percent: &pct}, nil) })
	}, 100, 100)
	k := float32(48) / 24
	cx, cy := 10+12*k, 10+12*k
	if got := testkit.PixelAt(tt, 100, cx+9*k*0.7071, cy-9*k*0.7071); !testkit.Near(got, tk.Primary) {
		t.Errorf("ring at 12.5%% %+v, want the primary arc", got)
	}
	if got := testkit.PixelAt(tt, 100, cx-9*k, cy); !testkit.Near(got, testkit.Over(tk.BgLayout, tk.FillTertiary)) {
		t.Errorf("ring at 75%% %+v, want the track", got)
	}
	a := tt.Image()
	tt.Advance(300 * time.Millisecond)
	tt.Frame()
	if string(a.Pix) != string(tt.Image().Pix) {
		t.Error("a determinate ring turns")
	}
}
