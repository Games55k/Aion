package markdown

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	mmd "github.com/egoist/mygo/ui/markdown"
	"github.com/lobehub/lobe-mygo/pkg/code"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type Variant int

const (
	// Default is for documents: 16 DIP text, large headings and generous room.
	Default Variant = iota
	// Chat is for messages: 14 DIP text, headings barely larger than it and tighter room.
	Chat
)

type Smoothing int

const (
	// Balanced paces text evenly, catching up within 120 ms of its arrival.
	Balanced Smoothing = iota
	// Realtime shows text closest to as it comes, within 40 ms.
	Realtime
	// Silky paces text the most smoothly, within 170 ms.
	Silky
)

type Granularity int

const (
	// ByChar shows a character at a time.
	ByChar Granularity = iota
	// ByWord shows a word at a time.
	ByWord
)

type MarkdownProps struct {
	// Variant picks the type scale and the room between blocks.
	Variant Variant
	// FontSize is the size of body text in DIPs: 16, or 14 in Chat, when zero.
	FontSize float32
	// HeaderMultiple scales how much larger headings are than text: 1, or 0.25 in Chat, when
	// zero. A heading of level 1 is 1 + 1.5 × HeaderMultiple times the text.
	HeaderMultiple float32
	// MarginMultiple scales the room between blocks: 2, or 1 in Chat, when zero.
	MarginMultiple float32
	// LineHeight is how many times its size a line of text is: 1.8, or 1.6 in Chat, when zero.
	LineHeight float32
	// FullFeaturedCodeBlock shows code blocks with Highlighter's full-featured header; without
	// it a block of one short line shows as a Snippet.
	FullFeaturedCodeBlock bool
	// OnLink handles a click on a link, given its destination; links open in the browser or the
	// router around the view without it.
	OnLink func(url string)
	// LoadImage loads an image's URL off the UI goroutine; images show as their descriptions
	// without it.
	LoadImage func(url string) (ui.ImageSource, error)
	// Animated tells a source still being written, as a language model's reply: it shows at an
	// even pace, each unit fading in, what is half written showing as it will most likely end.
	// Clear it once the reply is complete, in the same place: what is still coming in carries on,
	// and Revealed tells when it is done.
	Animated bool
	// Smoothing is how evenly an animated source comes in.
	Smoothing Smoothing
	// Granularity is what an animated source shows at a time.
	Granularity Granularity
}

type scale struct {
	fontSize, headerMultiple, marginMultiple, lineHeight float32
}

func scaleOf(p MarkdownProps) scale {
	s := scale{16, 1, 2, 1.8}
	if p.Variant == Chat {
		s = scale{14, 0.25, 1, 1.6}
	}
	set := func(v *float32, o float32) {
		if o > 0 {
			*v = o
		}
	}
	set(&s.fontSize, p.FontSize)
	set(&s.headerMultiple, p.HeaderMultiple)
	set(&s.marginMultiple, p.MarginMultiple)
	set(&s.lineHeight, p.LineHeight)
	return s
}

func themeFor(tk *theme.Tokens, p MarkdownProps) *mmd.Theme {
	s := scaleOf(p)
	th := &mmd.Theme{
		FontSize:         s.fontSize,
		CodeFont:         "monospace",
		LineHeight:       s.lineHeight,
		Text:             tk.Text,
		Muted:            tk.TextSecondary,
		Link:             tk.InfoText,
		Strong:           tk.Text,
		StrongWeight:     600,
		Code:             tk.Text,
		CodeBackground:   tk.FillSecondary,
		CodeSize:         0.875,
		BlockSpacing:     s.marginMultiple * 0.5,
		ItemSpacing:      s.marginMultiple * 0.33,
		QuoteBar:         tk.Border,
		QuoteText:        tk.TextSecondary,
		QuoteBarWidth:    4,
		Alerts:           [5]ui.Color{tk.Info, tk.Success, tk.Purple, tk.Warning, tk.Error},
		Rule:             tk.Border,
		TableBorder:      tk.BorderSecondary,
		TableHeader:      tk.FillQuaternary,
		TableCellPadding: [2]float32{1, 0.75},
		Checkbox:         tk.Primary,
	}
	grow := [6]float32{1.5, 1, 0.5, 0.25, 0, 0}
	for i := range th.Headings {
		size := 1 + grow[i]*s.headerMultiple
		room := max(s.headerMultiple*s.marginMultiple*0.4*size, 1)
		th.Headings[i] = mmd.HeadingStyle{Size: size, Weight: 700, Color: tk.Text, Top: room, Bottom: room}
	}
	return th
}

type themeKey struct {
	dark bool
	s    scale
}

type view struct {
	opts mmd.Options
	key  themeKey
	set  bool
}

// Markdown shows src, Markdown finished or, Animated, still being written, in Lobe's typography: CommonMark with tables, task
// lists, footnotes, GitHub's alerts and the rest of mygo's extensions, its code blocks as
// Highlighter shows them. Its text is selectable and copies as text, HTML or Markdown.
func Markdown(c *ui.Context, src string, p MarkdownProps) ui.Element {
	tk := theme.TokensOf(c)
	root := ui.Column(c).AlignSelf(ui.Stretch)
	v := ui.Local(root, "markdown", func() *view { return &view{} })
	key := themeKey{dark: tk.Dark, s: scaleOf(p)}
	if !(*v).set || (*v).key != key {
		(*v).opts.Theme = themeFor(tk, p)
		(*v).key, (*v).set = key, true
	}
	o := &(*v).opts
	reveal := revealFor(p)
	o.Reveal = &reveal
	o.OnLink = p.OnLink
	o.LoadImage = p.LoadImage
	size := o.Theme.FontSize * 0.85
	full := p.FullFeaturedCodeBlock
	o.CodeBlock = func(c *ui.Context, b *mmd.CodeBlockView) {
		src := b.Code()
		if !full && singleLine(src) {
			code.Snippet(c, src, code.SnippetProps{Language: b.Language, FontSize: size})
			return
		}
		code.Highlighter(c, src, code.HighlighterProps{Language: b.Language, FullFeatured: full, FontSize: size})
	}
	var md ui.Element
	root.Children(func() {
		if p.Animated {
			md = mmd.Stream(c, src, o)
		} else {
			md = mmd.View(c, src, o)
		}
	})
	return md
}

// Revealed reports whether el, a Markdown built this frame, shows all of its source, done fading
// in: a chat shows a reply's actions once it is complete and Revealed.
func Revealed(el ui.Element) bool { return mmd.Revealed(el) }

func revealFor(p MarkdownProps) mmd.Reveal {
	r := mmd.Reveal{Unit: mmd.ByCharacter, CatchUp: 120 * time.Millisecond, Fade: 300 * time.Millisecond}
	switch p.Smoothing {
	case Realtime:
		r.CatchUp = 40 * time.Millisecond
	case Silky:
		r.CatchUp = 170 * time.Millisecond
	}
	if p.Granularity == ByWord {
		r.Unit = mmd.ByWord
	}
	return r
}

func singleLine(src string) bool {
	src = strings.TrimRight(src, "\n")
	return !strings.Contains(src, "\n") && utf8.RuneCountInString(src) <= 32
}
