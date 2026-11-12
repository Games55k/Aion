package field

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func selectTester(v *string, p SelectProps) *ui.Tester {
	p.Options = []SelectOption{{Value: "gpt", Label: "GPT-5"}, {Value: "claude", Label: "Claude"}, {Value: "gemini", Label: "Gemini", Disabled: true}}
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { Select(c, v, p).Width(240) })
	}, 400, 300)
}

func TestSelectShowsPlaceholderThenChoice(t *testing.T) {
	v := ""
	tt := selectTester(&v, SelectProps{Placeholder: "Pick a model"})
	if !tt.HasText("Pick a model") {
		t.Fatalf("texts %q, want the placeholder", tt.Texts())
	}
	if err := tt.Click("Pick a model"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !tt.HasText("Claude") {
		t.Fatalf("popup not open: %q", tt.Texts())
	}
	tt.Click("Claude")
	tt.Advance(time.Second)
	tt.Frame()
	if v != "claude" {
		t.Fatalf("value %q after choosing Claude", v)
	}
	if tt.HasText("GPT-5") || !tt.HasText("Claude") {
		t.Errorf("after choosing: %q, want the popup closed and Claude in the trigger", tt.Texts())
	}
}

func TestSelectPopupMatchesTriggerAndMarksChoice(t *testing.T) {
	tk := theme.Light()
	v := "gpt"
	tt := selectTester(&v, SelectProps{})
	tt.Click("GPT-5")
	tt.Advance(time.Second)
	tt.Frame()
	claude, ok := tt.Find("Claude")
	if !ok {
		t.Fatal("popup not open")
	}
	if claude.X < 20 || claude.X > 40 {
		t.Errorf("option at x %v, want the popup under the trigger at 20", claude.X)
	}
	check := false
	row := claude.Y - 32 + claude.H/2
	for x := float32(200); x < 260; x++ {
		for dy := float32(-3); dy <= 3; dy++ {
			if testkit.Near(testkit.PixelAt(tt, 400, x, row+dy), tk.Primary) {
				check = true
			}
		}
	}
	if !check {
		t.Error("chosen option has no Primary check at the end of its row")
	}
	tt.Click("Gemini")
	tt.Advance(time.Second)
	if v != "gpt" {
		t.Errorf("disabled option chosen: %q", v)
	}
}
