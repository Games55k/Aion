package code

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func snippetTester(src string, p SnippetProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).AlignItems(ui.Start).Children(func() {
			Snippet(c, src, p).Label("snippet")
		})
	}, 500, 200)
}

func TestSnippetShowsPrefixAndCopiesCode(t *testing.T) {
	tt := snippetTester("  npm i @lobehub/ui\n", SnippetProps{Language: "sh", Prefix: "$"})
	info, ok := tt.Access("snippet")
	if !ok {
		t.Fatal("no snippet")
	}
	if info.Bounds.H != 38 {
		t.Errorf("height %v, want 38", info.Bounds.H)
	}
	if !tt.HasText("$ npm i @lobehub/ui") {
		t.Errorf("texts %q", tt.Texts())
	}
	btn, ok := tt.AccessByRole(ui.RoleButton, "Copy")
	if !ok {
		t.Fatal("no Copy button")
	}
	tt.ClickAt(btn.Bounds.X+btn.Bounds.W/2, btn.Bounds.Y+btn.Bounds.H/2)
	tt.Frame()
	if got := tt.Clipboard(); got != "npm i @lobehub/ui" {
		t.Errorf("clipboard %q", got)
	}
}

func TestSnippetVariantsAndNoCopy(t *testing.T) {
	tk := theme.Light()
	for _, k := range []struct {
		v          Variant
		fill, edge ui.Color
	}{
		{Filled, testkit.Over(tk.BgLayout, tk.FillTertiary), testkit.Over(tk.BgLayout, tk.FillTertiary)},
		{Outlined, tk.BgLayout, testkit.Over(tk.BgLayout, tk.BorderSecondary)},
		{Borderless, tk.BgLayout, tk.BgLayout},
	} {
		tt := snippetTester("pnpm dev", SnippetProps{Variant: k.v, NoCopy: true})
		if _, ok := tt.AccessByRole(ui.RoleButton, "Copy"); ok {
			t.Error("Copy button with NoCopy")
		}
		b, _ := tt.Access("snippet")
		r := b.Bounds
		if got := testkit.PixelAt(tt, 500, r.X+r.W-3, r.Y+r.H/2); !testkit.Near(got, k.fill) {
			t.Errorf("variant %d fill %+v, want %+v", k.v, got, k.fill)
		}
		if got := testkit.PixelAt(tt, 500, r.X+r.W/2, r.Y+0.5); !testkit.Near(got, k.edge) {
			t.Errorf("variant %d edge %+v, want %+v", k.v, got, k.edge)
		}
	}
}

func TestSnippetStaysOneLine(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(240).AlignItems(ui.Start).Children(func() {
			Snippet(c, strings.Repeat("echo hi && ", 20), SnippetProps{Language: "sh"}).Label("snippet")
		})
	}, 500, 200)
	b, _ := tt.Access("snippet")
	if b.Bounds.H != 38 || b.Bounds.W > 200 {
		t.Errorf("bounds %+v, want one 38-high line within its 200-wide parent", b.Bounds)
	}
	tk := theme.Light()
	bg := testkit.Over(tk.BgLayout, tk.FillTertiary)
	r := b.Bounds
	for _, y := range []float32{r.Y + 4, r.Y + r.H - 4} {
		if got := testkit.Ink(tt, 500, ui.Rect{X: r.X + 12, Y: y, W: 100, H: 1}, bg); !testkit.Near(got, bg) {
			t.Errorf("ink %+v at y=%v: the line wrapped", got, y-r.Y)
		}
	}
}
