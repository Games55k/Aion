package feedback

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func badgeTester(p BadgeProps, child bool) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(40).Children(func() {
			var content func()
			if child {
				content = func() { ui.Box(c).Size(40, 40).Background(theme.TokensOf(c).FillSecondary).Label("inbox") }
			}
			Badge(c, p, content)
		})
	}, 200, 140)
}

func TestBadgeCountSitsOnTheCorner(t *testing.T) {
	tk := theme.Light()
	tt := badgeTester(BadgeProps{Count: 5}, true)
	r, ok := tt.Find("5")
	if !ok {
		t.Fatalf("texts %q", tt.Texts())
	}
	if cx, cy := r.X+r.W/2, r.Y+r.H/2; cx < 79 || cx > 81 || cy < 39 || cy > 41 {
		t.Errorf("count centered at %v,%v, want on the top right corner 80,40", cx, cy)
	}
	if got := testkit.PixelAt(tt, 200, 80+6, 40); !testkit.Near(got, tk.Error) {
		t.Errorf("pill %+v, want the error color", got)
	}
}

func TestBadgeOverflowZeroAndDot(t *testing.T) {
	tk := theme.Light()
	if tt := badgeTester(BadgeProps{Count: 120}, true); !tt.HasText("99+") {
		t.Errorf("120: %q, want 99+", tt.Texts())
	}
	if tt := badgeTester(BadgeProps{Count: 120, OverflowCount: 999}, true); !tt.HasText("120") {
		t.Errorf("120 under 999: %q", tt.Texts())
	}
	if tt := badgeTester(BadgeProps{}, true); tt.HasText("0") {
		t.Error("zero shows without ShowZero")
	}
	if tt := badgeTester(BadgeProps{ShowZero: true}, true); !tt.HasText("0") {
		t.Error("ShowZero hides zero")
	}
	tt := badgeTester(BadgeProps{Dot: true}, true)
	if got := testkit.PixelAt(tt, 200, 80, 40); !testkit.Near(got, tk.Error) {
		t.Errorf("dot at the corner %+v", got)
	}
	if got := testkit.PixelAt(tt, 200, 80+5, 40); testkit.Near(got, tk.Error) {
		t.Error("dot wider than 8")
	}
}

func TestBadgeStatus(t *testing.T) {
	tk := theme.Light()
	tt := badgeTester(BadgeProps{Status: BadgeSuccess, Text: "Online"}, false)
	r, ok := tt.Find("Online")
	if !ok {
		t.Fatalf("texts %q", tt.Texts())
	}
	if got := testkit.PixelAt(tt, 200, r.X-8-3, r.Y+r.H/2); !testkit.Near(got, tk.Success) {
		t.Errorf("status dot %+v, want the success color 8 DIPs before the text", got)
	}
}
