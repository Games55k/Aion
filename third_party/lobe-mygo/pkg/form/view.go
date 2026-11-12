package form

import (
	"context"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type Variant int

const (
	Borderless Variant = iota
	Filled
	Outlined
)

type Layout int

const (
	// LayoutAuto takes the form's layout, which is Horizontal by default.
	LayoutAuto Layout = iota
	Horizontal
	Vertical
)

// Host is a *Form of any value type, as Group, Field and SubmitFooter take it.
type Host interface{ formHost() *host }

type Props[T any] struct {
	// Variant picks the groups' look: Borderless titles them over a line, Filled and Outlined box
	// them in cards that collapse.
	Variant Variant
	// Layout puts labels beside their controls (Horizontal) or above them (Vertical).
	Layout Layout
	// ItemMinWidth is the least width of a control beside its label.
	ItemMinWidth float32
	// ValidateOn is when fields check their values; ValidateAuto by default.
	ValidateOn ValidateOn
	// OnSubmit gets the values once every field is valid. The values become the form's baseline
	// unless it returns an error.
	OnSubmit func(values T) error
	// OnValuesChange gets the values after a change, ChangeDebounce after the last one, as to
	// save as the user types.
	OnValuesChange func(values T)
	ChangeDebounce time.Duration
}

// View lays out the groups and fields content builds for f, and runs f's checks and submission.
func View[T any](c *ui.Context, f *Form[T], p Props[T], content func()) ui.Element {
	h := &f.host
	h.drain()
	h.wake = c.Services().Invalidate
	layout := p.Layout
	if layout == LayoutAuto {
		layout = Horizontal
	}
	h.view = viewSettings{variant: p.Variant, layout: layout, itemMinWidth: p.ItemMinWidth, validateOn: p.ValidateOn}
	h.order = h.order[:0]
	for _, fs := range h.fields {
		fs.registered = false
	}
	h.dirty, h.reset = f.IsDirty, f.Reset
	h.submit = func() {
		h.validateAll()
		h.submitting = true
	}

	gap := float32(16)
	if p.Variant == Borderless {
		gap = 48
	}
	root := ui.Column(c).Gap(gap).AlignSelf(ui.Stretch)
	root.Children(content)

	for name, fs := range h.fields {
		if !fs.registered && fs.seen {
			if fs.cancel != nil {
				fs.cancel()
			}
			delete(h.fields, name)
		}
	}
	if h.submitting && !h.busy() {
		h.submitting = false
		if !h.hasErrors() && p.OnSubmit != nil && p.OnSubmit(deepCopy(f.Values)) == nil {
			f.baseline = deepCopy(f.Values)
			for _, fs := range h.fields {
				fs.touched, fs.edited = false, false
			}
			h.submitCount = 0
		}
	}
	if h.changed {
		h.changed = false
		if p.OnValuesChange != nil && p.ChangeDebounce <= 0 {
			p.OnValuesChange(deepCopy(f.Values))
		} else if p.OnValuesChange != nil {
			root.Jump("values-change", 1)
			h.changePending = true
		}
	}
	if h.changePending && root.AnimateTo("values-change", 0, ui.Animation{Duration: p.ChangeDebounce, Ease: ui.Linear}) == 0 {
		h.changePending = false
		p.OnValuesChange(deepCopy(f.Values))
	}
	return root
}

type FieldProps struct {
	// Name keys the field's state; a field without one only lays out its label and control.
	Name string
	// Label and Desc show beside the control, or above it in a vertical layout; Tag adds a small
	// tag after the label.
	Label, Desc, Tag string
	// Value is the field's value as the app holds it, for Required and to see it change.
	Value any
	// Required errs while Value is empty: "", nil, or an empty slice or map. RequiredText
	// replaces "This field is required".
	Required     bool
	RequiredText string
	// Validate returns the field's error, or "" when it is valid.
	Validate func() string
	// ValidateAsync checks the value off the UI thread once Validate passes, ValidateDebounce
	// after it changes; its context ends when a newer check starts.
	ValidateAsync    func(ctx context.Context, value any) string
	ValidateDebounce time.Duration
	// ValidateOn overrides the form's for this field.
	ValidateOn ValidateOn
	// Deps names fields whose changes check this one again once it was touched.
	Deps []string
	// Layout overrides the form's for this field.
	Layout Layout
	// MinWidth is the least width of the control beside its label.
	MinWidth float32
}

// Field lays out a control built by control with its label, description and error, and keeps the
// state of the field named p.Name: touched, edited, its error and its checks.
func Field(c *ui.Context, f Host, p FieldProps, control func()) ui.Element {
	tk := theme.TokensOf(c)
	h := f.formHost()
	layout := p.Layout
	if layout == LayoutAuto {
		layout = h.view.layout
	}
	if h.groupFields > 0 && h.view.variant != Borderless {
		ui.Box(c.Key("divider-" + p.Name)).Height(1).AlignSelf(ui.Stretch).Background(tk.BorderSecondary.Alpha(0.66))
	}
	h.groupFields++

	var root ui.Element
	if layout == Vertical {
		root = ui.Column(c.Key(p.Name)).Gap(8).AlignItems(ui.Stretch)
	} else {
		root = ui.Row(c.Key(p.Name)).Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween)
	}
	root.PaddingY(16).AlignSelf(ui.Stretch)
	root.Children(func() {
		if p.Label != "" || p.Desc != "" {
			title := fieldTitle(c, tk, p)
			if layout == Horizontal {
				title.Grow(1).Basis(0).MinWidth(0)
			}
		}
		col := ui.Column(c).Gap(4)
		if layout == Vertical {
			col.AlignItems(ui.Stretch).AlignSelf(ui.Stretch)
		} else {
			col.AlignItems(ui.End).Shrink(0)
			if w := max(p.MinWidth, h.view.itemMinWidth); w > 0 {
				col.MinWidth(w)
			}
		}
		var fs *fieldState
		if p.Name != "" {
			fs = h.observe(p.Name, p, col.FocusWithin())
		}
		col.Children(func() {
			control()
			if fs == nil {
				return
			}
			if err := h.Error(p.Name); err != "" {
				ui.Text(c, err).FontSize(12).TextColor(tk.Error)
			}
		})
	})
	return root
}

func fieldTitle(c *ui.Context, tk *theme.Tokens, p FieldProps) ui.Element {
	return ui.Column(c).Gap(4).Children(func() {
		if p.Label != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, p.Label).FontSize(14).TextColor(tk.Text)
				if p.Required {
					ui.Text(c, "*").FontSize(14).TextColor(tk.Error)
				}
				if p.Tag != "" {
					display.Tag(c, p.Tag, display.TagProps{Size: theme.Small})
				}
			})
		}
		if p.Desc != "" {
			ui.Text(c, p.Desc).FontSize(12).LineHeight(1.5).TextColor(tk.TextDescription)
		}
	})
}
