package code

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const goCode = `
package main

// Greet says hello.
func Greet() string {
	return "hi"
}
`

func spanColor(t *testing.T, spans []ui.Span, text string) ui.Span {
	t.Helper()
	for _, s := range spans {
		if s.Text == text {
			return s
		}
	}
	var all []string
	for _, s := range spans {
		all = append(all, s.Text)
	}
	t.Fatalf("no span %q in %q", text, all)
	return ui.Span{}
}

func TestHighlightSpansUseLobeColors(t *testing.T) {
	tk := theme.Light()
	ct := Theme(tk)
	spans := Spans(ct, "go", strings.TrimSpace(goCode))
	if s := spanColor(t, spans, "func"); s.Color != tk.Info {
		t.Errorf("keyword %+v, want info %+v", s.Color, tk.Info)
	}
	if s := spanColor(t, spans, `"hi"`); s.Color != tk.Success {
		t.Errorf("string %+v, want success %+v", s.Color, tk.Success)
	}
	if s := spanColor(t, spans, "Greet"); s.Color != tk.Geekblue {
		t.Errorf("function %+v, want geekblue %+v", s.Color, tk.Geekblue)
	}
	if s := spanColor(t, spans, "string"); s.Color != tk.Warning {
		t.Errorf("type %+v, want warning %+v", s.Color, tk.Warning)
	}
	if s := spanColor(t, spans, "// Greet says hello."); s.Color != tk.TextQuaternary || !s.Italic {
		t.Errorf("comment %+v, want italic quaternary", s)
	}
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	if b.String() != strings.TrimSpace(goCode) {
		t.Errorf("spans spell %q", b.String())
	}
}

func TestHighlightSpansPlainForUnknownLanguage(t *testing.T) {
	tk := theme.Dark()
	spans := Spans(Theme(tk), "klingon", "a\nb")
	if len(spans) != 3 || spans[0].Text != "a" || spans[1].Text != "\n" || spans[2].Text != "b" {
		t.Fatalf("spans %+v", spans)
	}
	if spans[0].Color != tk.Text {
		t.Errorf("plain text %+v, want %+v", spans[0].Color, tk.Text)
	}
}

func TestLanguageName(t *testing.T) {
	for in, want := range map[string]string{"go": "Go", "ts": "TypeScript", "TypeScript": "TypeScript", "": "Plaintext", "klingon": "Plaintext"} {
		if got := LanguageName(in); got != want {
			t.Errorf("LanguageName(%q) = %q, want %q", in, got, want)
		}
	}
}

func highlighterTester(code string, p HighlighterProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			Highlighter(c, code, p).Label("code")
		})
	}, 500, 400)
}

func codeBounds(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.Access("code")
	if !ok {
		t.Fatal("no highlighter")
	}
	return info.Bounds
}

func TestHighlighterCopiesTrimmedCode(t *testing.T) {
	tt := highlighterTester(goCode, HighlighterProps{Language: "go"})
	b := codeBounds(t, tt)
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	info, ok := tt.AccessByRole(ui.RoleButton, "Copy")
	if !ok {
		t.Fatal("no Copy button")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
	if got := tt.Clipboard(); got != strings.TrimSpace(goCode) {
		t.Errorf("clipboard %q", got)
	}
}

func TestHighlighterChromeShowsOnHover(t *testing.T) {
	tk := theme.Light()
	bg := testkit.Over(tk.BgLayout, tk.FillQuaternary)
	tt := highlighterTester(goCode, HighlighterProps{Language: "go"})
	b := codeBounds(t, tt)
	copyBtn, _ := tt.AccessByRole(ui.RoleButton, "Copy")
	cb := copyBtn.Bounds
	if got := testkit.Ink(tt, 500, cb, bg); !testkit.Near(got, bg) {
		t.Errorf("copy button shows %+v before hover", got)
	}
	if !tt.HasText("Go") {
		t.Errorf("texts %q, want the language tag", tt.Texts())
	}
	tt.Move(b.X+b.W/2, b.Y+b.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if got := testkit.Ink(tt, 500, cb, bg); testkit.Near(got, bg) {
		t.Error("copy button still hidden on hover")
	}
}

func TestHighlighterNoCopyHideLanguage(t *testing.T) {
	tt := highlighterTester(goCode, HighlighterProps{Language: "go", NoCopy: true, HideLanguage: true})
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copy"); ok {
		t.Error("Copy button with NoCopy")
	}
	if tt.HasText("Go") {
		t.Errorf("texts %q, want no language tag", tt.Texts())
	}
}

func TestHighlighterVariants(t *testing.T) {
	tk := theme.Light()
	cases := []struct {
		v    Variant
		fill ui.Color
		edge ui.Color
	}{
		{Filled, testkit.Over(tk.BgLayout, tk.FillQuaternary), testkit.Over(tk.BgLayout, tk.FillQuaternary)},
		{Outlined, tk.BgLayout, testkit.Over(tk.BgLayout, tk.BorderSecondary)},
		{Borderless, tk.BgLayout, tk.BgLayout},
	}
	for _, k := range cases {
		tt := highlighterTester("x", HighlighterProps{Variant: k.v, NoCopy: true, HideLanguage: true})
		b := codeBounds(t, tt)
		if got := testkit.PixelAt(tt, 500, b.X+b.W-30, b.Y+b.H/2); !testkit.Near(got, k.fill) {
			t.Errorf("variant %d fill %+v, want %+v", k.v, got, k.fill)
		}
		if got := testkit.PixelAt(tt, 500, b.X+b.W/2, b.Y+0.5); !testkit.Near(got, k.edge) {
			t.Errorf("variant %d edge %+v, want %+v", k.v, got, k.edge)
		}
	}
}

func TestHighlighterTabsAreFourSpaces(t *testing.T) {
	tk := theme.Light()
	bg := testkit.Over(tk.BgLayout, tk.FillQuaternary)
	firstInk := func(code string) float32 {
		tt := highlighterTester(code, HighlighterProps{NoCopy: true, HideLanguage: true})
		b := codeBounds(t, tt)
		for x := b.X + 1; x < b.X+b.W; x++ {
			for y := b.Y + 4; y < b.Y+b.H-4; y++ {
				if !testkit.Near(testkit.PixelAt(tt, 500, x, y), bg) {
					return x - b.X
				}
			}
		}
		return -1
	}
	tab, spaces := firstInk("\tX"), firstInk("    X")
	if tab < 0 || tab-spaces > 1 || spaces-tab > 1 {
		t.Errorf("tab starts at %v, four spaces at %v", tab, spaces)
	}
}

func TestHighlighterWrap(t *testing.T) {
	long := strings.Repeat("word ", 60)
	one := codeBounds(t, highlighterTester(long, HighlighterProps{NoCopy: true, HideLanguage: true})).H
	wrapped := codeBounds(t, highlighterTester(long, HighlighterProps{NoCopy: true, HideLanguage: true, Wrap: true})).H
	if wrapped < one*2 {
		t.Errorf("wrapped height %v, unwrapped %v", wrapped, one)
	}
}

func TestHighlighterFullFeaturedCollapses(t *testing.T) {
	tt := highlighterTester(goCode, HighlighterProps{Language: "go", FullFeatured: true, FileName: "main.go"})
	if !tt.HasText("main.go") {
		t.Fatalf("texts %q, want the file name", tt.Texts())
	}
	copyBtn, ok := tt.AccessByRole(ui.RoleButton, "Copy")
	if !ok {
		t.Error("no Copy button in the header")
	}
	if title, ok := tt.Access("main.go"); !ok {
		t.Error("no title trigger")
	} else if mid := func(r ui.Rect) float32 { return r.Y + r.H/2 }; mid(title.Bounds) < mid(copyBtn.Bounds)-2 || mid(title.Bounds) > mid(copyBtn.Bounds)+2 {
		t.Errorf("title at %+v, copy at %+v: want one header row", title.Bounds, copyBtn.Bounds)
	}
	open := codeBounds(t, tt).H
	toggle, ok := tt.AccessByRole(ui.RoleButton, "Collapse")
	if !ok {
		t.Fatal("no Collapse toggle")
	}
	tt.ClickAt(toggle.Bounds.X+toggle.Bounds.W/2, toggle.Bounds.Y+toggle.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	closed := codeBounds(t, tt).H
	if closed >= open-40 {
		t.Errorf("height %v after collapsing, %v open", closed, open)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Expand"); !ok {
		t.Error("no Expand toggle once collapsed")
	}

	tt = highlighterTester(goCode, HighlighterProps{Language: "go", FullFeatured: true, DefaultCollapsed: true})
	if !tt.HasText("Go") {
		t.Errorf("texts %q, want the language name", tt.Texts())
	}
	if h := codeBounds(t, tt).H; h > 50 {
		t.Errorf("DefaultCollapsed height %v", h)
	}
}

func TestHighlighterFullFeaturedNoCopy(t *testing.T) {
	tt := highlighterTester(goCode, HighlighterProps{Language: "go", FullFeatured: true, NoCopy: true})
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copy"); ok {
		t.Error("Copy button with NoCopy")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Collapse"); !ok {
		t.Error("no Collapse toggle")
	}
}

func TestCodeBlocksNameTheirLanguage(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Gap(8).Children(func() {
			Highlighter(c.Key(1), "x := 1", HighlighterProps{Language: "go"})
			Highlighter(c.Key(2), "x", HighlighterProps{})
			Snippet(c.Key(3), "ls", SnippetProps{Language: "sh"})
		})
	}, 500, 400)
	for _, name := range []string{"Code, Go", "Code", "Code, Shell"} {
		if info, ok := tt.AccessByRole(ui.RoleGroup, name); !ok || info.Bounds.H == 0 {
			t.Errorf("no group %q", name)
		}
	}
}

func TestCodeFontSize(t *testing.T) {
	height := func(p HighlighterProps) float32 {
		tt := highlighterTester("a\nb\nc\nd", p)
		return codeBounds(t, tt).H
	}
	small, big := height(HighlighterProps{}), height(HighlighterProps{FontSize: 24})
	if big < small+30 {
		t.Errorf("height %v at 24 DIPs, %v at the default 12", big, small)
	}
	wide := func(p SnippetProps) float32 {
		tt := snippetTester("echo hello world", p)
		b, _ := tt.Access("snippet")
		return b.Bounds.W
	}
	if a, b := wide(SnippetProps{}), wide(SnippetProps{FontSize: 24}); b < a+40 {
		t.Errorf("snippet width %v at 24 DIPs, %v at 12", b, a)
	}
}
