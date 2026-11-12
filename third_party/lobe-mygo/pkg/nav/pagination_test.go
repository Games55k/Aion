package nav

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func pagerTester(cur *int, p PaginationProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { Pagination(c, cur, p) })
	}, 600, 100)
}

func clickRole(t *testing.T, tt *ui.Tester, r ui.Role, name string) {
	t.Helper()
	info, ok := tt.AccessByRole(r, name)
	if !ok {
		t.Fatalf("no %v %q; texts %q", r, name, tt.Texts())
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
}

func TestPaginationPagesAndEllipses(t *testing.T) {
	cur := 10
	tt := pagerTester(&cur, PaginationProps{Total: 500})
	for _, s := range []string{"1", "8", "9", "10", "11", "12", "50"} {
		if !tt.HasText(s) {
			t.Errorf("no page %s in %q", s, tt.Texts())
		}
	}
	if tt.HasText("7") || tt.HasText("13") {
		t.Errorf("pages beyond two either side show: %q", tt.Texts())
	}
	clickRole(t, tt, ui.RoleButton, "Next 5 pages")
	if cur != 15 {
		t.Errorf("jump forward: %d, want 15", cur)
	}
	clickRole(t, tt, ui.RoleButton, "Previous page")
	if cur != 14 {
		t.Errorf("previous: %d", cur)
	}
	tt.Click("50")
	tt.Frame()
	if cur != 50 {
		t.Errorf("clicking 50: %d", cur)
	}
	clickRole(t, tt, ui.RoleButton, "Next page")
	if cur != 50 {
		t.Errorf("next on the last page moved to %d", cur)
	}
}

func TestPaginationTotalAndSmallCounts(t *testing.T) {
	cur := 2
	tt := pagerTester(&cur, PaginationProps{Total: 45, PageSize: 20, ShowTotal: func(total, from, to int) string {
		return fmt.Sprintf("%d-%d of %d", from, to, total)
	}})
	if !tt.HasText("21-40 of 45") {
		t.Errorf("texts %q, want the range", tt.Texts())
	}
	_, three := tt.AccessByRole(ui.RoleButton, "3")
	_, four := tt.AccessByRole(ui.RoleButton, "4")
	if !three || four {
		t.Errorf("45 items by 20: %q, want pages 1–3", tt.Texts())
	}
	one := 1
	tt = pagerTester(&one, PaginationProps{Total: 5, HideOnSinglePage: true})
	if _, ok := tt.AccessByRole(ui.RoleButton, "1"); ok {
		t.Error("a single page shows despite HideOnSinglePage")
	}
}
