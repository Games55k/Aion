package frontend

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	mmd "github.com/egoist/mygo/ui/markdown"
)

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

func codeGroupLabel(lang string) string {
	if lang == "" {
		return "Code"
	}
	return "Code, " + codeLanguageName(lang)
}

func renderCodeBlock(c *ui.Context, tk *uiTokens, src, lang string, fontSize float32) ui.Element {
	src = strings.TrimRight(strings.TrimLeft(src, "\r\n"), " \t\r\n")
	root := ui.Column(c).AlignSelf(ui.Stretch).Radius(tk.BorderRadius).Clip().
		Background(tk.FillQuaternary).Role(ui.RoleGroup).Label(codeGroupLabel(lang))
	hl := ui.Local(root, "spans", func() highlightedCode { return highlightedCode{} })
	if hl.src != src || hl.lang != lang || hl.dark != tk.Dark {
		*hl = highlightedCode{src: src, lang: lang, dark: tk.Dark, spans: codeSpans(codeTheme(tk), lang, src)}
	}

	open := ui.Local(root, "open", func() bool { return true })
	name := codeLanguageName(lang)
	label := "Collapse"
	if !*open {
		label = "Expand"
	}
	root.Children(func() {
		var col ui.CollapsibleParts
		ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(4).AlignSelf(ui.Stretch).Children(func() {
			col = ui.CollapsibleBase(c, open)
			col.Trigger.Row().Grow(1).MinWidth(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(8).
				Cursor(ui.CursorPointer).Label(name).Children(func() {
				ui.Text(c, name).FontSize(13).SingleLine().TextColor(tk.Text)
			})
			uiCopyButton(c, src, copyButtonProps{Size: sizeSmall})
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
			ui.Column(c).AlignSelf(ui.Stretch).Opacity(progress).
				BorderWidth(1, 0, 0, 0).BorderColor(tk.FillQuaternary).Children(func() {
				ui.ScrollHorizontal(c).AlignSelf(ui.Stretch).Direction(ui.LeftToRight).
					Font("monospace").FontSize(fontSize).LineHeight(1.8).TabSize(4).Children(func() {
					ui.Column(c).Padding(16).Shrink(0).Children(func() {
						ui.RichText(c, hl.spans...).NoWrap()
					})
				})
			})
		})
	})
	return root
}

func markdownThemeFor(tk *uiTokens) *mmd.Theme {
	const (
		fontSize       float32 = 14
		headerMultiple float32 = 0.25
		marginMultiple float32 = 1
		lineHeight     float32 = 1.6
	)
	th := &mmd.Theme{
		FontSize:         fontSize,
		CodeFont:         "monospace",
		LineHeight:       lineHeight,
		Text:             tk.Text,
		Muted:            tk.TextSecondary,
		Link:             tk.InfoText,
		Strong:           tk.Text,
		StrongWeight:     600,
		Code:             tk.Text,
		CodeBackground:   tk.FillSecondary,
		CodeSize:         0.875,
		BlockSpacing:     marginMultiple * 0.5,
		ItemSpacing:      marginMultiple * 0.33,
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
		size := 1 + grow[i]*headerMultiple
		room := max(headerMultiple*marginMultiple*0.4*size, 1)
		th.Headings[i] = mmd.HeadingStyle{Size: size, Weight: 700, Color: tk.Text, Top: room, Bottom: room}
	}
	return th
}

type markdownViewState struct {
	opts mmd.Options
	dark bool
	set  bool
}

func uiMarkdown(c *ui.Context, src string) ui.Element {
	tk := tokensOf(c)
	root := ui.Column(c).AlignSelf(ui.Stretch)
	v := ui.Local(root, "markdown", func() *markdownViewState { return &markdownViewState{} })
	if !(*v).set || (*v).dark != tk.Dark {
		(*v).opts.Theme = markdownThemeFor(tk)
		(*v).dark, (*v).set = tk.Dark, true
	}
	o := &(*v).opts
	size := o.Theme.FontSize * 0.85
	o.CodeBlock = func(c *ui.Context, b *mmd.CodeBlockView) {
		renderCodeBlock(c, tokensOf(c), b.Code(), b.Language, size)
	}
	var md ui.Element
	root.Children(func() {
		md = mmd.View(c, src, o)
	})
	return md
}
