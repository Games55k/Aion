package frontend

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	mmd "github.com/egoist/mygo/ui/markdown"
)

type codeVariant int

const (
	codeFilled codeVariant = iota
	codeOutlined
	codeBorderless
)

type highlighterProps struct {
	Language         string
	Variant          codeVariant
	NoCopy           bool
	HideLanguage     bool
	Wrap             bool
	Shadow           bool
	FullFeatured     bool
	FileName         string
	DefaultCollapsed bool
	Actions          func()
	FontSize         float32
}

type snippetProps struct {
	Language string
	Prefix   string
	Variant  codeVariant
	NoCopy   bool
	Shadow   bool
	FontSize float32
}

var codeChromeFade = ui.Animation{Duration: 200 * time.Millisecond, Ease: easeOut}

type highlightedCode struct {
	src, lang string
	dark      bool
	spans     []ui.Span
}

func codeTheme(tk *uiTokens) *editor.Theme {
	ct := &editor.Theme{Background: tk.BgContainer, Text: tk.Text}
	set := func(k editor.TokenKind, col ui.Color) { ct.Tokens[k] = editor.TokenStyle{Color: col} }
	for _, k := range []editor.TokenKind{editor.TokenText, editor.TokenVariable, editor.TokenEscape} {
		set(k, tk.Text)
	}
	for _, k := range []editor.TokenKind{editor.TokenKeyword, editor.TokenOperator, editor.TokenPunctuation, editor.TokenConstant, editor.TokenHeading} {
		set(k, tk.Info)
	}
	for _, k := range []editor.TokenKind{editor.TokenString, editor.TokenRegexp, editor.TokenInserted, editor.TokenCode} {
		set(k, tk.Success)
	}
	for _, k := range []editor.TokenKind{editor.TokenProperty, editor.TokenNumber, editor.TokenTag, editor.TokenLink, editor.TokenDeleted} {
		set(k, tk.Volcano)
	}
	set(editor.TokenType, tk.Warning)
	set(editor.TokenFunction, tk.Geekblue)
	set(editor.TokenAttribute, tk.Purple)
	set(editor.TokenMeta, tk.TextQuaternary)
	set(editor.TokenInvalid, tk.Error)
	ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: tk.TextQuaternary, Italic: true}
	return ct
}

func codeSpans(ct *editor.Theme, lang, src string) []ui.Span {
	var tok editor.Tokenizer
	if l := editor.LanguageNamed(lang); l != nil {
		tok = l.Tokenizer
	}
	var spans []ui.Span
	var state uint64
	var toks []editor.Token
	for i, line := range strings.Split(src, "\n") {
		if i > 0 {
			spans = append(spans, ui.Span{Text: "\n"})
		}
		start := 0
		if tok != nil {
			toks, state = tok.Tokenize(line, state, toks[:0])
			for _, t := range toks {
				end := min(t.End, len(line))
				if end > start {
					spans = append(spans, ui.Span{Text: line[start:end], Color: ct.TokenColor(t.Kind), Italic: ct.Tokens[t.Kind].Italic})
				}
				start = end
			}
		}
		if start < len(line) {
			spans = append(spans, ui.Span{Text: line[start:], Color: ct.Text})
		}
	}
	return spans
}

func codeLanguageName(lang string) string {
	if l := editor.LanguageNamed(lang); l != nil {
		return l.Name
	}
	return "Plaintext"
}

func uiHighlighter(c *ui.Context, src string, p highlighterProps) ui.Element {
	tk := tokensOf(c)
	src = strings.TrimRight(strings.TrimLeft(src, "\r\n"), " \t\r\n")
	root := ui.Column(c).AlignSelf(ui.Stretch).Radius(tk.BorderRadius).Clip().Role(ui.RoleGroup).Label(codeGroupLabel(p.Language))
	switch p.Variant {
	case codeFilled:
		root.Background(tk.FillQuaternary)
	case codeOutlined:
		root.Border(1, tk.BorderSecondary)
	}
	if p.Shadow {
		tk.Elevate(root, boxShadowSecondary)
	}
	hl := ui.Local(root, "spans", func() highlightedCode { return highlightedCode{} })
	if hl.src != src || hl.lang != p.Language || hl.dark != tk.Dark {
		*hl = highlightedCode{src: src, lang: p.Language, dark: tk.Dark, spans: codeSpans(codeTheme(tk), p.Language, src)}
	}
	if p.FullFeatured {
		fullFeaturedCode(c, tk, root, src, hl.spans, p)
		return root
	}
	shown := float32(0)
	if root.Hovered() {
		shown = 1
	}
	opacity := root.AnimateTo("chrome", shown, codeChromeFade)
	root.Children(func() {
		codeBody(c, hl.spans, p.Wrap, p.FontSize)
		if !p.NoCopy || p.Actions != nil {
			ui.Row(c).Absolute().Top(8).Right(8).Gap(4).AlignItems(ui.Center).Opacity(opacity).Children(func() {
				codeActions(c, src, p)
			})
		}
		if !p.HideLanguage && p.Language != "" {
			uiTag(c, codeLanguageName(p.Language), tagProps{}).Absolute().Bottom(8).Right(8).
				Font("monospace").TextColor(tk.TextSecondary).Background(tk.FillQuaternary).Opacity(opacity)
		}
	})
	return root
}

func uiSnippet(c *ui.Context, src string, p snippetProps) ui.Element {
	tk := tokensOf(c)
	src = strings.TrimSpace(src)
	root := ui.Row(c).AlignItems(ui.Center).Gap(8).Height(38).Padding(0, 8, 0, 12).
		Radius(tk.BorderRadius).Clip().Role(ui.RoleGroup).Label(codeGroupLabel(p.Language))
	switch p.Variant {
	case codeFilled:
		root.Background(tk.FillTertiary)
	case codeOutlined:
		root.Border(1, tk.BorderSecondary)
	}
	if p.Shadow {
		tk.Elevate(root, boxShadowSecondary)
	}
	shown := src
	if p.Prefix != "" {
		shown = p.Prefix + " " + src
	}
	hl := ui.Local(root, "spans", func() highlightedCode { return highlightedCode{} })
	if hl.src != shown || hl.lang != p.Language || hl.dark != tk.Dark {
		*hl = highlightedCode{src: shown, lang: p.Language, dark: tk.Dark, spans: codeSpans(codeTheme(tk), p.Language, shown)}
	}
	return root.Children(func() {
		ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Font("monospace").FontSize(codeFontSize(p.FontSize)).TabSize(4).Children(func() {
			ui.RichText(c, hl.spans...)
		})
		if !p.NoCopy {
			uiCopyButton(c, src, copyButtonProps{Size: sizeSmall}).Shrink(0)
		}
	})
}

func codeActions(c *ui.Context, src string, p highlighterProps) {
	if p.Actions != nil {
		p.Actions()
	}
	if !p.NoCopy {
		uiCopyButton(c, src, copyButtonProps{Size: sizeSmall})
	}
}

func codeGroupLabel(lang string) string {
	if lang == "" {
		return "Code"
	}
	return "Code, " + codeLanguageName(lang)
}

func codeFontSize(size float32) float32 {
	if size <= 0 {
		return 12
	}
	return size
}

func codeBody(c *ui.Context, spans []ui.Span, wrap bool, size float32) ui.Element {
	var outer ui.Element
	if wrap {
		outer = ui.Column(c)
	} else {
		outer = ui.ScrollHorizontal(c)
	}
	outer.AlignSelf(ui.Stretch).Direction(ui.LeftToRight).Font("monospace").FontSize(codeFontSize(size)).LineHeight(1.8).TabSize(4)
	return outer.Children(func() {
		inner := ui.Column(c).Padding(16)
		if !wrap {
			inner.Shrink(0)
		}
		inner.Children(func() {
			text := ui.RichText(c, spans...)
			if !wrap {
				text.NoWrap()
			}
		})
	})
}

func fullFeaturedCode(c *ui.Context, tk *uiTokens, root ui.Element, src string, spans []ui.Span, p highlighterProps) {
	open := ui.Local(root, "open", func() bool { return !p.DefaultCollapsed })
	name := p.FileName
	if name == "" {
		name = codeLanguageName(p.Language)
	}
	label := "Collapse"
	if !*open {
		label = "Expand"
	}
	root.Children(func() {
		var col ui.CollapsibleParts
		head := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(4).AlignSelf(ui.Stretch)
		if p.Variant == codeBorderless {
			head.PaddingX(0)
		}
		head.Children(func() {
			col = ui.CollapsibleBase(c, open)
			col.Trigger.Row().Grow(1).MinWidth(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(8).
				Cursor(ui.CursorPointer).Label(name).Children(func() {
				if !p.HideLanguage {
					ui.Text(c, name).FontSize(13).SingleLine().TextColor(tk.Text)
				}
			})
			codeActions(c, src, p)
			chevron := iconChevronDown()
			if !*open {
				chevron = iconChevronRight()
			}
			if uiActionIcon(c, chevron, actionIconProps{Size: sizeSmall}).Label(label).Clicked() {
				*open = !*open
			}
		})
		progress := col.Progress()
		col.Panel(func() {
			panel := ui.Column(c).AlignSelf(ui.Stretch).Opacity(progress)
			if p.Variant != codeBorderless {
				panel.BorderWidth(1, 0, 0, 0).BorderColor(tk.FillQuaternary)
			}
			panel.Children(func() { codeBody(c, spans, p.Wrap, p.FontSize) })
		})
	})
}

type markdownVariant int

const (
	markdownDefault markdownVariant = iota
	markdownChat
)

type markdownSmoothing int

const (
	mdBalanced markdownSmoothing = iota
	mdRealtime
	mdSilky
)

type markdownGranularity int

const (
	mdByChar markdownGranularity = iota
	mdByWord
)

type markdownProps struct {
	Variant               markdownVariant
	FontSize              float32
	HeaderMultiple        float32
	MarginMultiple        float32
	LineHeight            float32
	FullFeaturedCodeBlock bool
	OnLink                func(url string)
	LoadImage             func(url string) (ui.ImageSource, error)
	Animated              bool
	Smoothing             markdownSmoothing
	Granularity           markdownGranularity
}

type markdownScale struct {
	fontSize, headerMultiple, marginMultiple, lineHeight float32
}

func markdownScaleOf(p markdownProps) markdownScale {
	s := markdownScale{16, 1, 2, 1.8}
	if p.Variant == markdownChat {
		s = markdownScale{14, 0.25, 1, 1.6}
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

func markdownThemeFor(tk *uiTokens, p markdownProps) *mmd.Theme {
	s := markdownScaleOf(p)
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

type markdownThemeKey struct {
	dark bool
	s    markdownScale
}

type markdownViewState struct {
	opts mmd.Options
	key  markdownThemeKey
	set  bool
}

func uiMarkdown(c *ui.Context, src string, p markdownProps) ui.Element {
	tk := tokensOf(c)
	root := ui.Column(c).AlignSelf(ui.Stretch)
	v := ui.Local(root, "markdown", func() *markdownViewState { return &markdownViewState{} })
	key := markdownThemeKey{dark: tk.Dark, s: markdownScaleOf(p)}
	if !(*v).set || (*v).key != key {
		(*v).opts.Theme = markdownThemeFor(tk, p)
		(*v).key, (*v).set = key, true
	}
	o := &(*v).opts
	reveal := revealForMarkdown(p)
	o.Reveal = &reveal
	o.OnLink = p.OnLink
	o.LoadImage = p.LoadImage
	size := o.Theme.FontSize * 0.85
	full := p.FullFeaturedCodeBlock
	o.CodeBlock = func(c *ui.Context, b *mmd.CodeBlockView) {
		codeSrc := b.Code()
		if !full && singleLineCode(codeSrc) {
			uiSnippet(c, codeSrc, snippetProps{Language: b.Language, FontSize: size})
			return
		}
		uiHighlighter(c, codeSrc, highlighterProps{Language: b.Language, FullFeatured: full, FontSize: size})
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

func revealForMarkdown(p markdownProps) mmd.Reveal {
	r := mmd.Reveal{Unit: mmd.ByCharacter, CatchUp: 120 * time.Millisecond, Fade: 300 * time.Millisecond}
	switch p.Smoothing {
	case mdRealtime:
		r.CatchUp = 40 * time.Millisecond
	case mdSilky:
		r.CatchUp = 170 * time.Millisecond
	}
	if p.Granularity == mdByWord {
		r.Unit = mmd.ByWord
	}
	return r
}

func singleLineCode(src string) bool {
	src = strings.TrimRight(src, "\n")
	return !strings.Contains(src, "\n") && utf8.RuneCountInString(src) <= 32
}
