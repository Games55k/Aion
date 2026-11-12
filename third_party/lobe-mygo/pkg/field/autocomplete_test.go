package field

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestAutoCompleteFiltersAndFills(t *testing.T) {
	v := ""
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(300).Children(func() {
			AutoComplete(c, &v, AutoCompleteProps{
				Options:   []AutoCompleteOption{{Value: "gpt-5"}, {Value: "claude-opus"}, {Value: "claude-sonnet", Label: "Claude Sonnet"}},
				EmptyText: "No models",
			}).Label("model")
		})
	}, 400, 300)
	tt.Click("model")
	tt.Type("clau")
	tt.Advance(time.Second)
	tt.Frame()
	if tt.HasText("gpt-5") || !tt.HasText("claude-opus") || !tt.HasText("Claude Sonnet") {
		t.Fatalf("typing clau shows %q, want only the claude options", tt.Texts())
	}
	tt.Click("Claude Sonnet")
	tt.Advance(time.Second)
	tt.Frame()
	if v != "claude-sonnet" {
		t.Errorf("value %q after choosing Claude Sonnet", v)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("zzz")
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("No models") {
		t.Errorf("no match shows %q, want the empty text", tt.Texts())
	}
}

func TestAutoCompletePopupLinesUpWithTheField(t *testing.T) {
	v := ""
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(300).Children(func() {
			AutoComplete(c, &v, AutoCompleteProps{Options: []AutoCompleteOption{{Value: "gpt-5"}}, Prefix: lucide.Bot()}).Label("model")
		})
	}, 400, 300)
	tt.Click("model")
	tt.Key(0, ui.KeyDown)
	tt.Advance(time.Second)
	tt.Frame()
	r, ok := tt.Find("gpt-5")
	if !ok {
		t.Fatalf("popup not open: %q", tt.Texts())
	}
	if want := float32(20 + 4 + 12); r.X < want-1 || r.X > want+1 {
		t.Errorf("option text at x %v, want %v: the popup's edge under the field's", r.X, want)
	}
}
