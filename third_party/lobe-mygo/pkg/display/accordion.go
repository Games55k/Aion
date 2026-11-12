package display

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type AccordionVariant int

const (
	AccordionBorderless AccordionVariant = iota
	AccordionFilled
	AccordionOutlined
)

type AccordionIndicator int

const (
	AccordionIndicatorStart AccordionIndicator = iota
	AccordionIndicatorEnd
	AccordionIndicatorNone
)

type AccordionItem struct {
	// Key names the section in the open map and to the panel function.
	Key string
	// Title labels the section's header.
	Title string
	// Disabled keeps the section from opening or closing.
	Disabled bool
}

type AccordionProps struct {
	// Items are the sections, in order.
	Items []AccordionItem
	// Variant picks the look: Borderless shows the headers on hover only, Filled tints them,
	// Outlined draws a border around the sections and dividers between them.
	Variant AccordionVariant
	// Single keeps one section open: opening one closes the others.
	Single bool
	// Indicator places the arrow before the title, turning right to down, or after it, turning
	// down to up, or hides it.
	Indicator AccordionIndicator
	// Gap is the space between sections.
	Gap float32
}

type accordionState struct{ open, last bool }

var (
	accordionHover = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.EaseOut}
)

// Accordion shows sections that open and close below their headers; open[key] says which are
// open and changes as they are clicked. panel builds the content of the section key.
func Accordion(c *ui.Context, open map[string]bool, p AccordionProps, panel func(key string)) ui.Element {
	tk := theme.TokensOf(c)
	acc := ui.AccordionBase(c)
	root := acc.Root.FillWidth().Gap(p.Gap)
	if p.Variant == AccordionOutlined {
		root.Border(1, tk.BorderSecondary).Radius(tk.BorderRadiusLG).Clip()
	}

	states := make([]*accordionState, len(p.Items))
	toggled := ""
	for i, it := range p.Items {
		st := ui.Local(root, "item:"+it.Key, func() accordionState { return accordionState{open: open[it.Key], last: open[it.Key]} })
		if st.open != st.last {
			open[it.Key] = st.open
			if st.open {
				toggled = it.Key
			}
		}
		states[i] = st
	}
	if p.Single && toggled != "" {
		for _, it := range p.Items {
			if it.Key != toggled {
				open[it.Key] = false
			}
		}
	}

	root.Children(func() {
		for i, it := range p.Items {
			st := states[i]
			st.open, st.last = open[it.Key], open[it.Key]
			accordionSection(c.Key(it.Key), tk, acc, st, it, p, i, panel)
		}
	})
	return root
}

func accordionSection(c *ui.Context, tk *theme.Tokens, acc ui.AccordionParts, st *accordionState, it AccordionItem, p AccordionProps, index int, panel func(string)) {
	ui.Column(c).FillWidth().Children(func() {
		if p.Variant == AccordionOutlined && index > 0 {
			ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary)
		}
		item := acc.Item(&st.open)
		progress := item.Progress()
		tr := item.Trigger.FillWidth().Gap(8).AlignItems(ui.Center).FontSize(14).TextColor(tk.Text).Cursor(ui.CursorPointer)
		if it.Disabled {
			tr.Disabled(true).TextColor(tk.TextDisabled).Cursor(ui.CursorNotAllowed)
		}
		var bg, hover ui.Color
		switch p.Variant {
		case AccordionFilled:
			tr.Padding(8, 16).Radius(tk.BorderRadius)
			bg, hover = tk.FillTertiary, tk.FillSecondary
		case AccordionOutlined:
			bottom := float32(12)
			if st.open {
				bottom = 8
			}
			tr.Padding(12, 16, bottom, 16)
			hover = tk.FillTertiary
		default:
			tr.Padding(8, 8).Margin(0, -8).Radius(tk.BorderRadius)
			hover = tk.FillTertiary
		}
		if tr.Hovered() && !it.Disabled {
			bg = hover
		}
		tr.Background(tr.AnimateColor("bg", bg, accordionHover))

		tr.Children(func() {
			if p.Indicator == AccordionIndicatorStart {
				ui.Icon(c, lucide.ChevronRight()).FontSize(16).TextColor(tk.TextDescription).Shrink(0).Rotate(90 * progress)
			}
			ui.Text(c, it.Title).Grow(1).MinWidth(0)
			if p.Indicator == AccordionIndicatorEnd {
				ui.Icon(c, lucide.ChevronDown()).FontSize(16).TextColor(tk.TextDescription).Shrink(0).Rotate(180 * progress)
			}
		})

		content := item.Panel(func() { panel(it.Key) })
		if content == (ui.Element{}) {
			return
		}
		indent := float32(0)
		if p.Indicator == AccordionIndicatorStart {
			indent = 24
		}
		switch p.Variant {
		case AccordionOutlined:
			if indent > 0 {
				indent = 40
			} else {
				indent = 16
			}
			content.Padding(4, 16, 14, indent)
		case AccordionFilled:
			content.Padding(8, 16, 12, max(indent, 16))
		default:
			content.Padding(0, 0, 12, indent)
		}
		shift := float32(6)
		if !st.open {
			shift = -6
		}
		content.FontSize(14).LineHeight(1.6).Opacity(progress).Translate(0, shift*(1-progress))
	})
}
