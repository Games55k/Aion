package markdown

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func near(a, b float32) bool { return a-b < 0.01 && b-a < 0.01 }

func TestThemeDefaultVariant(t *testing.T) {
	tk := theme.Light()
	th := themeFor(tk, MarkdownProps{})
	if th.FontSize != 16 || th.LineHeight != 1.8 || !near(th.BlockSpacing, 1) || !near(th.ItemSpacing, 0.66) {
		t.Errorf("size %v, line %v, blocks %v, items %v", th.FontSize, th.LineHeight, th.BlockSpacing, th.ItemSpacing)
	}
	sizes := [6]float32{2.5, 2, 1.5, 1.25, 1, 1}
	tops := [6]float32{2, 1.6, 1.2, 1, 1, 1}
	for i, h := range th.Headings {
		if !near(h.Size, sizes[i]) || !near(h.Top, tops[i]) || !near(h.Bottom, tops[i]) || h.Weight != 700 {
			t.Errorf("h%d %+v, want size %v and room %v", i+1, h, sizes[i], tops[i])
		}
	}
	if th.Link != tk.InfoText || th.QuoteText != tk.TextSecondary || th.QuoteBar != tk.Border || th.QuoteBarWidth != 4 {
		t.Errorf("link %+v, quote %+v/%+v/%v", th.Link, th.QuoteText, th.QuoteBar, th.QuoteBarWidth)
	}
	if th.CodeBackground != tk.FillSecondary || th.TableBorder != tk.BorderSecondary || th.TableHeader != tk.FillQuaternary ||
		th.TableCellPadding != [2]float32{1, 0.75} {
		t.Errorf("code %+v, table %+v/%+v/%v", th.CodeBackground, th.TableBorder, th.TableHeader, th.TableCellPadding)
	}
	if th.Alerts != [5]ui.Color{tk.Info, tk.Success, tk.Purple, tk.Warning, tk.Error} {
		t.Errorf("alerts %+v", th.Alerts)
	}
}

func TestThemeChatVariantAndOverrides(t *testing.T) {
	tk := theme.Dark()
	th := themeFor(tk, MarkdownProps{Variant: Chat})
	if th.FontSize != 14 || th.LineHeight != 1.6 || !near(th.BlockSpacing, 0.5) {
		t.Errorf("size %v, line %v, blocks %v", th.FontSize, th.LineHeight, th.BlockSpacing)
	}
	if h := th.Headings[0]; !near(h.Size, 1.375) || !near(h.Top, 1) {
		t.Errorf("h1 %+v", h)
	}
	th = themeFor(tk, MarkdownProps{Variant: Chat, FontSize: 18, LineHeight: 2, MarginMultiple: 3, HeaderMultiple: 1})
	if th.FontSize != 18 || th.LineHeight != 2 || !near(th.BlockSpacing, 1.5) || !near(th.Headings[1].Size, 2) {
		t.Errorf("overrides: size %v, line %v, blocks %v, h2 %v", th.FontSize, th.LineHeight, th.BlockSpacing, th.Headings[1].Size)
	}
}

const doc = "# Title\n\nSome **bold** text and a [link](https://lobehub.com).\n\n" +
	"```go\npackage main\n\nfunc main() {}\n```\n\n```sh\nnpm i @lobehub/ui\n```\n"

func mdTester(src string, p MarkdownProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			Markdown(c, src, p)
		})
	}, 600, 600)
}

func TestMarkdownRendersWithLobeCodeBlocks(t *testing.T) {
	tt := mdTester(doc, MarkdownProps{})
	for _, s := range []string{"Title", "Some", "bold"} {
		if !tt.HasText(s) {
			t.Errorf("texts %q, want %q", tt.Texts(), s)
		}
	}
	block, ok := tt.AccessByRole(ui.RoleGroup, "Code, Go")
	if !ok {
		t.Fatal("the go block is not a Highlighter")
	}
	if block.Bounds.H <= 38 {
		t.Errorf("go block %+v, want a multi-line Highlighter", block.Bounds)
	}
	line, ok := tt.AccessByRole(ui.RoleGroup, "Code, Shell")
	if !ok || line.Bounds.H != 38 {
		t.Errorf("sh block %+v (found %v), want a 38-high Snippet", line.Bounds, ok)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copy code"); ok {
		t.Error("mygo's default code block still shows")
	}
}

func TestMarkdownFullFeaturedCodeBlocks(t *testing.T) {
	tt := mdTester(doc, MarkdownProps{FullFeaturedCodeBlock: true})
	if _, ok := tt.AccessByRole(ui.RoleButton, "Collapse"); !ok {
		t.Error("no full-featured header")
	}
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Code, Shell"); !ok {
		t.Fatal("no sh block")
	}
	if info, _ := tt.AccessByRole(ui.RoleGroup, "Code, Shell"); info.Bounds.H == 38 {
		t.Error("a full-featured Markdown still shows a one-line block as a Snippet")
	}
}

func TestMarkdownLongSingleLineIsAHighlighter(t *testing.T) {
	tt := mdTester("```sh\necho this line is longer than thirty-two characters\n```\n", MarkdownProps{})
	if info, ok := tt.AccessByRole(ui.RoleGroup, "Code, Shell"); !ok || info.Bounds.H == 38 {
		t.Errorf("block %+v (found %v), want a Highlighter", info.Bounds, ok)
	}
}

func TestMarkdownChatIsSmaller(t *testing.T) {
	heading := func(p MarkdownProps) float32 {
		tt := mdTester("# Title\n\nText", p)
		info, ok := tt.AccessByRole(ui.RoleHeading, "Title")
		if !ok {
			t.Fatal("no heading")
		}
		return info.Bounds.H
	}
	if chat, def := heading(MarkdownProps{Variant: Chat}), heading(MarkdownProps{}); chat >= def*0.7 {
		t.Errorf("chat heading %v tall, default %v", chat, def)
	}
}

func TestMarkdownOnLink(t *testing.T) {
	var got string
	tt := mdTester("[LobeHub](https://lobehub.com)", MarkdownProps{OnLink: func(url string) { got = url }})
	info, ok := tt.AccessByRole(ui.RoleLink, "LobeHub")
	if !ok {
		t.Fatal("no link")
	}
	tt.ClickAt(info.Bounds.X+5, info.Bounds.Y+info.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if got != "https://lobehub.com" {
		t.Errorf("OnLink got %q", got)
	}
}

func TestMarkdownCodeFollowsTextSize(t *testing.T) {
	block := func(p MarkdownProps) float32 {
		tt := mdTester("```go\na\nb\nc\nd\n```\n", p)
		info, ok := tt.AccessByRole(ui.RoleGroup, "Code, Go")
		if !ok {
			t.Fatal("no code block")
		}
		return info.Bounds.H
	}
	if small, big := block(MarkdownProps{FontSize: 12}), block(MarkdownProps{FontSize: 28}); big < small+40 {
		t.Errorf("code block %v tall at 28 DIP text, %v at 12", big, small)
	}
}

func TestMarkdownThemeFollowsPropsChanges(t *testing.T) {
	p := MarkdownProps{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { Markdown(c, "# Title\n\nText", p) })
	}, 600, 600)
	h := func() float32 {
		info, _ := tt.AccessByRole(ui.RoleHeading, "Title")
		return info.Bounds.H
	}
	before := h()
	p.Variant = Chat
	tt.Frame()
	if after := h(); after >= before*0.7 {
		t.Errorf("heading %v tall after switching to Chat, %v before", after, before)
	}
}

func TestMarkdownLoadsImages(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	load := func(url string) (ui.ImageSource, error) {
		mu.Lock()
		asked = append(asked, url)
		mu.Unlock()
		return nil, errors.New("offline")
	}
	tt := mdTester("![cat](https://example.com/cat.png)", MarkdownProps{LoadImage: load})
	for range 100 {
		tt.Frame()
		mu.Lock()
		n := len(asked)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(asked) == 0 || asked[0] != "https://example.com/cat.png" {
		t.Errorf("LoadImage asked for %q", asked)
	}
}
