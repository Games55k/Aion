package code

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Theme returns Lobe's code colors for tk, as lobe-ui's lobe-theme colors them: keywords and
// punctuation in the info color, strings in the success color, types in the warning color,
// functions in geekblue and comments in quaternary text, in italics.
func Theme(tk *theme.Tokens) *editor.Theme {
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

// Spans highlights src, code in lang, into spans of text colored by ct, its lines joined by
// newline spans; code of a language mygo's editor does not know comes as plain text.
func Spans(ct *editor.Theme, lang, src string) []ui.Span {
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

// LanguageName is how a language shows in a code block's header: its name as mygo's editor
// knows it, from a name, an alias or an extension ("ts" is "TypeScript"), or "Plaintext".
func LanguageName(lang string) string {
	if l := editor.LanguageNamed(lang); l != nil {
		return l.Name
	}
	return "Plaintext"
}
