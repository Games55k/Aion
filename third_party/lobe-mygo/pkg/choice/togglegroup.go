package choice

import (
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ToggleGroupVariant int

const (
	ToggleOutlined ToggleGroupVariant = iota
	ToggleBorderless
	ToggleSegmented
)

type ToggleItem struct {
	// Value is what the item puts in the selection.
	Value string
	// Label is the item's text, and its name to assistive technology.
	Label string
	// Icon shows before the label; an item may have either or both.
	Icon *ui.SVG
	// Disabled keeps the item from toggling.
	Disabled bool
}

type ToggleGroupProps struct {
	// Items are the toggles, in order.
	Items []ToggleItem
	// Variant picks the look: Outlined joins the items in a border with dividers, Borderless
	// spaces them apart, Segmented puts them on a track as Segmented does.
	Variant ToggleGroupVariant
	// Size sets the item height: 24 or 28 (26 or 32 Segmented).
	Size theme.Size
	// Multiple lets any number of items be pressed; one or none is otherwise.
	Multiple bool
}

type toggleState struct{ on, last bool }

var toggleFade = ui.ElementTransition{Duration: 120 * time.Millisecond, Ease: motion.EaseOut, Colors: true}

// ToggleGroup shows toggle buttons whose pressed values *selected holds, in the order of the items.
func ToggleGroup(c *ui.Context, selected *[]string, p ToggleGroupProps) ui.Element {
	tk := theme.TokensOf(c)
	root := ui.Row(c).AlignItems(ui.Center).AlignSelf(ui.Start).Shrink(0)
	switch p.Variant {
	case ToggleOutlined:
		root.Border(1, tk.BorderSecondary).Radius(tk.BorderRadius).Clip()
	case ToggleBorderless:
		root.Gap(2)
	case ToggleSegmented:
		root.Padding(3).Gap(4).Radius(tk.BorderRadiusLG).Border(1, tk.FillQuaternary).Background(tk.BgLayout)
	}

	states := make([]*toggleState, len(p.Items))
	toggled := -1
	for i, it := range p.Items {
		on := slices.Contains(*selected, it.Value)
		st := ui.Local(root, "toggle:"+it.Value, func() toggleState { return toggleState{on, on} })
		if st.on != st.last {
			toggled = i
		}
		states[i] = st
	}
	if toggled >= 0 {
		turnedOn := states[toggled].on
		next := []string{}
		for i, it := range p.Items {
			on := slices.Contains(*selected, it.Value)
			switch {
			case i == toggled:
				on = turnedOn
			case !p.Multiple && turnedOn:
				on = false
			}
			if on {
				next = append(next, it.Value)
			}
		}
		*selected = next
	}

	root.Children(func() {
		for i, it := range p.Items {
			st := states[i]
			st.on = slices.Contains(*selected, it.Value)
			st.last = st.on
			if p.Variant == ToggleOutlined && i > 0 {
				ui.Box(c.Key("div" + it.Value)).Width(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary)
			}
			toggleItem(c.Key(it.Value), tk, st, it, p)
		}
	})
	return root
}

func toggleItem(c *ui.Context, tk *theme.Tokens, st *toggleState, it ToggleItem, p ToggleGroupProps) {
	b := ui.ToggleBase(c, &st.on).Gap(4).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
		Cursor(ui.CursorPointer).Unselectable().Transition(toggleFade)
	if it.Disabled {
		b.Disabled(true).Cursor(ui.CursorNotAllowed)
	}
	if it.Label != "" {
		b.Label(it.Label)
	}
	hovered := b.Hovered() && !it.Disabled
	var bg, fg ui.Color
	if p.Variant == ToggleSegmented {
		height, padding, font := float32(32), float32(12), float32(13)
		if p.Size == theme.Small {
			height, padding, font = 26, 10, 12
		}
		b.Height(height).PaddingX(padding).FontSize(font).FontWeight(500).Radius(tk.BorderRadius)
		fg = tk.TextSecondary
		switch {
		case it.Disabled:
			fg = tk.TextDisabled
		case st.on:
			bg, fg = tk.BgElevated, tk.Text
			tk.Elevate(b, theme.BoxShadowTertiary)
		case hovered:
			fg = tk.Text
		}
	} else {
		height, padding := float32(28), float32(8)
		if p.Size == theme.Small {
			height, padding = 24, 6
		}
		b.Height(height).PaddingX(padding).FontSize(12)
		if p.Variant == ToggleBorderless {
			b.Radius(tk.BorderRadius)
		}
		fg = tk.TextTertiary
		switch {
		case it.Disabled && st.on:
			bg, fg = tk.FillSecondary, tk.TextDisabled
		case it.Disabled:
			fg = tk.TextDisabled
		case st.on && hovered:
			bg, fg = tk.Fill, tk.TextSecondary
		case st.on:
			bg, fg = tk.FillSecondary, tk.Text
		case hovered:
			bg, fg = tk.FillSecondary, tk.TextSecondary
		}
	}
	b.Background(bg).TextColor(fg)
	b.Children(func() {
		if it.Icon != nil {
			ui.Icon(c, it.Icon).FontSize(16)
		}
		if it.Label != "" {
			ui.Text(c, it.Label).SingleLine()
		}
	})
}
