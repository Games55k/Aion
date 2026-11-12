package field

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func searchTester(q *string, p SearchBarProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			SearchBar(c, q, p).Label("search")
			ui.Button(c, "Elsewhere")
		})
	}, 400, 200)
}

func TestSearchBarShortKeyTagAndFocus(t *testing.T) {
	q := ""
	tt := searchTester(&q, SearchBarProps{ShortKey: "mod+k", Platform: 2})
	if !tt.HasText("Ctrl") || !tt.HasText("K") {
		t.Fatalf("texts %q, want the Ctrl K tag", tt.Texts())
	}
	tt.Key(ui.Ctrl, ui.KeyK)
	tt.Frame()
	if !tt.Focused("search") {
		t.Fatal("Ctrl+K did not focus the search")
	}
	if tt.HasText("Ctrl") {
		t.Error("tag still shows while the search has the focus")
	}
	tt.Type("lobe")
	tt.Frame()
	if q != "lobe" {
		t.Errorf("query %q", q)
	}
}

func TestSearchBarEnterSearchesAndClearEmpties(t *testing.T) {
	q := ""
	var searched []string
	tt := searchTester(&q, SearchBarProps{OnSearch: func(s string) { searched = append(searched, s) }})
	if err := tt.Click("search"); err != nil {
		t.Fatal(err)
	}
	tt.Type("agents")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if len(searched) != 1 || searched[0] != "agents" {
		t.Fatalf("searched %q, want [agents]", searched)
	}
	info, ok := tt.AccessByRole(ui.RoleButton, "Clear")
	if !ok {
		t.Fatal("no clear button with a query")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
	if q != "" {
		t.Errorf("query %q after clearing", q)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Clear"); ok {
		t.Error("clear button shows with an empty query")
	}
}

// Regression: a Small (24px) SearchBar previously rendered the 21.6px default hotkey cap flush against its top and bottom borders.
func TestSmallSearchBarHotkeyHasVerticalMargin(t *testing.T) {
	tk := theme.Light()
	q := ""
	tt := searchTester(&q, SearchBarProps{ShortKey: "mod+k", Size: theme.Small, Platform: 2})
	k, ok := tt.Find("K")
	if !ok {
		t.Fatal("missing hotkey text K")
	}
	cy := k.Y + k.H/2
	chipBg := testkit.Over(tk.BgContainer, tk.FillTertiary)
	// 10 DIPs above/below the vertical center is inside the 24px bar (±12) but outside the 16px chip (±8).
	for _, dy := range []float32{-10, 10} {
		got := testkit.PixelAt(tt, 400, k.X, cy+dy)
		if testkit.Near(got, chipBg) || !testkit.Near(got, tk.BgContainer) {
			t.Fatalf("at dy=%v: pixel=%+v, want BgContainer %+v (outside chipBg %+v)", dy, got, tk.BgContainer, chipBg)
		}
	}
}
