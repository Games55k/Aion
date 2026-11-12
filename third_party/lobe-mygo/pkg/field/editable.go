package field

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type EditableTextProps struct {
	// NoEditIcon hides the edit button beside the text; editing then starts by setting *editing.
	NoEditIcon bool
	// Variant picks the editing field's surface, as Input's.
	Variant InputVariant
	// Size sets the height (24, 32 or 40), as Input's.
	Size theme.Size
	// OnChangeEnd runs with the new text when an edit is saved.
	OnChangeEnd func(text string)
}

type editDraft struct {
	text    string
	editing bool
}

// EditableText shows *value as text with an edit button; while *editing it is a field holding a
// draft, which Enter or the save button writes to *value and Escape drops. A changed draft shows
// buttons to reset and save it.
func EditableText(c *ui.Context, value *string, editing *bool, p EditableTextProps) ui.Element {
	tk := theme.TokensOf(c)
	height := float32(32)
	switch p.Size {
	case theme.Small:
		height = 24
	case theme.Large:
		height = 40
	}
	root := ui.Row(c).Height(height).Gap(8).AlignItems(ui.Center).FillWidth()
	draft := ui.Local(root, "draft", func() editDraft { return editDraft{} })
	if *editing && !draft.editing {
		draft.text = *value
	}
	draft.editing = *editing

	root.Children(func() {
		if !*editing {
			ui.Text(c, *value).SingleLine().MinWidth(0)
			if !p.NoEditIcon {
				if action.ActionIcon(c, lucide.PenLine(), action.ActionIconProps{Size: theme.Small, Title: "Edit"}).Clicked() {
					*editing, draft.editing, draft.text = true, true, *value
					c.Invalidate()
				}
			}
			return
		}
		save := func() {
			*value, *editing, draft.editing = draft.text, false, false
			if p.OnChangeEnd != nil {
				p.OnChangeEnd(draft.text)
			}
			c.Invalidate()
		}
		field := fieldRow(c, tk, p.Size).Height(height).Gap(8).Grow(1)
		var input ui.Element
		field.Children(func() {
			input = ui.TextInputBase(c, &draft.text).Grow(1).MinWidth(0).FocusRing(false).AutoFocus()
			if p.Size == theme.Small {
				input.FontSize(14)
			}
			switch {
			case input.Submitted():
				save()
				return
			case input.Shortcut(0, ui.KeyEscape):
				*editing, draft.editing = false, false
				c.Invalidate()
				return
			}
			if draft.text != *value {
				ui.Row(c).Gap(2).Margin(0, -4, 0, 0).Children(func() {
					if action.ActionIcon(c, lucide.RotateCcw(), action.ActionIconProps{Size: theme.Small, Title: "Reset"}).Clicked() {
						draft.text = *value
					}
					if action.ActionIcon(c, lucide.Save(), action.ActionIconProps{Size: theme.Small, Title: "Submit", Variant: action.ActionIconFilled}).Clicked() {
						save()
					}
				})
			}
		})
		styleField(field, tk, p.Variant, false, input.Focused())
	})
	return root
}
