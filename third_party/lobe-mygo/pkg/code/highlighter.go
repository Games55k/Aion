package code

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type Variant int

const (
	// Filled sits on a light fill.
	Filled Variant = iota
	// Outlined draws a border instead.
	Outlined
	// Borderless shows only the code.
	Borderless
)

type HighlighterProps struct {
	// Language is the code's language, by name, alias or extension ("go", "ts", "tsx"); code of a
	// language mygo's editor does not know shows plain.
	Language string
	// Variant picks the surface.
	Variant Variant
	// NoCopy leaves out the copy button.
	NoCopy bool
	// HideLanguage leaves out the language: the tag the pointer brings up, or the header's name.
	HideLanguage bool
	// Wrap wraps long lines rather than scrolling them sideways.
	Wrap bool
	// Shadow lifts the block with a soft shadow.
	Shadow bool
	// FullFeatured adds a header with the language or FileName, the actions and a toggle that
	// collapses the code.
	FullFeatured bool
	// FileName replaces the language's name in the full-featured header.
	FileName string
	// DefaultCollapsed starts a full-featured block collapsed.
	DefaultCollapsed bool
	// Actions builds more buttons before the copy button, as a Run or a Download.
	Actions func()
	// FontSize is the code's size in DIPs: 12 when zero.
	FontSize float32
}

var chromeFade = ui.Animation{Duration: 200 * time.Millisecond, Ease: motion.EaseOut}

type highlighted struct {
	src, lang string
	dark      bool
	spans     []ui.Span
}

// Highlighter shows src, trimmed of blank lines around it, highlighted in Lobe's code colors. The
// copy button and the language show while the pointer is over the block.
func Highlighter(c *ui.Context, src string, p HighlighterProps) ui.Element {
	tk := theme.TokensOf(c)
	src = strings.TrimRight(strings.TrimLeft(src, "\r\n"), " \t\r\n")
	root := ui.Column(c).AlignSelf(ui.Stretch).Radius(tk.BorderRadius).Clip().Role(ui.RoleGroup).Label(groupLabel(p.Language))
	switch p.Variant {
	case Filled:
		root.Background(tk.FillQuaternary)
	case Outlined:
		root.Border(1, tk.BorderSecondary)
	}
	if p.Shadow {
		tk.Elevate(root, theme.BoxShadowSecondary)
	}
	hl := ui.Local(root, "spans", func() highlighted { return highlighted{} })
	if hl.src != src || hl.lang != p.Language || hl.dark != tk.Dark {
		*hl = highlighted{src: src, lang: p.Language, dark: tk.Dark, spans: Spans(Theme(tk), p.Language, src)}
	}
	if p.FullFeatured {
		fullFeatured(c, tk, root, src, hl.spans, p)
		return root
	}
	shown := float32(0)
	if root.Hovered() {
		shown = 1
	}
	opacity := root.AnimateTo("chrome", shown, chromeFade)
	root.Children(func() {
		body(c, hl.spans, p.Wrap, p.FontSize)
		if !p.NoCopy || p.Actions != nil {
			ui.Row(c).Absolute().Top(8).Right(8).Gap(4).AlignItems(ui.Center).Opacity(opacity).Children(func() {
				actions(c, src, p)
			})
		}
		if !p.HideLanguage && p.Language != "" {
			display.Tag(c, LanguageName(p.Language), display.TagProps{}).Absolute().Bottom(8).Right(8).
				Font("monospace").TextColor(tk.TextSecondary).Background(tk.FillQuaternary).Opacity(opacity)
		}
	})
	return root
}

func actions(c *ui.Context, src string, p HighlighterProps) {
	if p.Actions != nil {
		p.Actions()
	}
	if !p.NoCopy {
		action.CopyButton(c, src, action.CopyButtonProps{Size: theme.Small})
	}
}

func groupLabel(lang string) string {
	if lang == "" {
		return "Code"
	}
	return "Code, " + LanguageName(lang)
}

func fontSize(size float32) float32 {
	if size <= 0 {
		return 12
	}
	return size
}

func body(c *ui.Context, spans []ui.Span, wrap bool, size float32) ui.Element {
	var outer ui.Element
	if wrap {
		outer = ui.Column(c)
	} else {
		outer = ui.ScrollHorizontal(c)
	}
	outer.AlignSelf(ui.Stretch).Direction(ui.LeftToRight).Font("monospace").FontSize(fontSize(size)).LineHeight(1.8).TabSize(4)
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

func fullFeatured(c *ui.Context, tk *theme.Tokens, root ui.Element, src string, spans []ui.Span, p HighlighterProps) {
	open := ui.Local(root, "open", func() bool { return !p.DefaultCollapsed })
	name := p.FileName
	if name == "" {
		name = LanguageName(p.Language)
	}
	label := "Collapse"
	if !*open {
		label = "Expand"
	}
	root.Children(func() {
		var col ui.CollapsibleParts
		head := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(4).AlignSelf(ui.Stretch)
		if p.Variant == Borderless {
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
			actions(c, src, p)
			chevron := lucide.ChevronDown()
			if !*open {
				chevron = lucide.ChevronRight()
			}
			if action.ActionIcon(c, chevron, action.ActionIconProps{Size: theme.Small}).Label(label).Clicked() {
				*open = !*open
			}
		})
		progress := col.Progress()
		col.Panel(func() {
			panel := ui.Column(c).AlignSelf(ui.Stretch).Opacity(progress)
			if p.Variant != Borderless {
				panel.BorderWidth(1, 0, 0, 0).BorderColor(tk.FillQuaternary)
			}
			panel.Children(func() { body(c, spans, p.Wrap, p.FontSize) })
		})
	})
}
