package nav

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestBreadcrumb(t *testing.T) {
	var went []string
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			Breadcrumb(c, BreadcrumbProps{Items: []BreadcrumbItem{
				{Title: "Home", OnClick: func() { went = append(went, "home") }},
				{Title: "Agents", OnClick: func() { went = append(went, "agents") }},
				{Title: "Translator", OnClick: func() { went = append(went, "translator") }},
			}})
		})
	}, 400, 80)
	home, _ := tt.Find("Home")
	agents, _ := tt.Find("Agents")
	last, ok := tt.Find("Translator")
	if !ok || !(home.X < agents.X && agents.X < last.X) {
		t.Fatalf("order %v %v %v", home, agents, last)
	}
	if gap := agents.X - (home.X + home.W); gap < 4+14+4-0.5 {
		t.Errorf("%v DIPs between items, want a 14 DIP chevron 4 apart", gap)
	}
	tt.Click("Agents")
	tt.Frame()
	if len(went) != 1 || went[0] != "agents" {
		t.Errorf("went %q", went)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Translator"); ok {
		t.Error("the current page is a button")
	}
}

func TestBreadcrumbTextSeparator(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			Breadcrumb(c, BreadcrumbProps{Separator: "/", Items: []BreadcrumbItem{{Title: "Docs"}, {Title: "Guides"}}})
		})
	}, 400, 80)
	docs, _ := tt.Find("Docs")
	slash, ok := tt.Find("/")
	if !ok || slash.X < docs.X+docs.W || slash.W > 10 {
		t.Errorf("slash %v after Docs %v; texts %q", slash, docs, tt.Texts())
	}
}
