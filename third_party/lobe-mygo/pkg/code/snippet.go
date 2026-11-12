package code

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type SnippetProps struct {
	// Language highlights the line, as Highlighter's.
	Language string
	// Prefix shows before the code and is left out of what Copy copies, as "$" before a command.
	Prefix string
	// Variant picks the surface.
	Variant Variant
	// NoCopy leaves out the copy button.
	NoCopy bool
	// Shadow lifts the snippet with a soft shadow.
	Shadow bool
	// FontSize is the code's size in DIPs: 12 when zero.
	FontSize float32
}

// Snippet shows a line of code, as a command to run, highlighted and scrolling sideways when long,
// with a button copying it.
func Snippet(c *ui.Context, src string, p SnippetProps) ui.Element {
	tk := theme.TokensOf(c)
	src = strings.TrimSpace(src)
	root := ui.Row(c).AlignItems(ui.Center).Gap(8).Height(38).Padding(0, 8, 0, 12).
		Radius(tk.BorderRadius).Clip().Role(ui.RoleGroup).Label(groupLabel(p.Language))
	switch p.Variant {
	case Filled:
		root.Background(tk.FillTertiary)
	case Outlined:
		root.Border(1, tk.BorderSecondary)
	}
	if p.Shadow {
		tk.Elevate(root, theme.BoxShadowSecondary)
	}
	shown := src
	if p.Prefix != "" {
		shown = p.Prefix + " " + src
	}
	hl := ui.Local(root, "spans", func() highlighted { return highlighted{} })
	if hl.src != shown || hl.lang != p.Language || hl.dark != tk.Dark {
		*hl = highlighted{src: shown, lang: p.Language, dark: tk.Dark, spans: Spans(Theme(tk), p.Language, shown)}
	}
	return root.Children(func() {
		ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Font("monospace").FontSize(fontSize(p.FontSize)).TabSize(4).Children(func() {
			ui.RichText(c, hl.spans...)
		})
		if !p.NoCopy {
			action.CopyButton(c, src, action.CopyButtonProps{Size: theme.Small}).Shrink(0)
		}
	})
}
