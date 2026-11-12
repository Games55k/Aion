package feedback

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ToastType int

const (
	ToastDefault ToastType = iota
	ToastInfo
	ToastSuccess
	ToastWarning
	ToastError
	ToastLoading
)

var toastTypeNames = []string{"", "info", "success", "warning", "error", "loading"}

type ToastProps struct {
	// ID names the toast: showing one with the ID of a toast on screen changes it. ShowToast makes
	// one up when it is empty.
	ID string
	// Title and Description are what the toast says; a toast with only a Description shows it in
	// primary text.
	Title, Description string
	// Type picks the icon and its color.
	Type ToastType
	// Timeout is how long the toast shows: 5 seconds when zero, until it is closed when negative.
	// The time stops while the pointer rests on the toasts.
	Timeout time.Duration
	// Action labels a button that runs OnAction and closes the toast, as Undo.
	Action   string
	OnAction func()
}

// ShowToast shows a toast in the ToastHost and returns its ID.
func ShowToast(c *ui.Context, o ToastProps) string {
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return c.AddToast(ui.Toast{
		ID: o.ID, Title: o.Title, Description: o.Description, Type: toastTypeNames[o.Type],
		Timeout: timeout, Action: o.Action, OnAction: o.OnAction,
	})
}

type ToastPlacement int

const (
	ToastBottomRight ToastPlacement = iota
	ToastBottomLeft
	ToastBottom
	ToastTopRight
	ToastTopLeft
	ToastTop
)

type ToastHostProps struct {
	// Placement is the corner, or the middle of the edge, the toasts stack in.
	Placement ToastPlacement
}

const (
	toastWidth = 360
	toastGap   = 12
	toastPeek  = 12
)

var (
	toastStack = ui.Animation{Duration: 400 * time.Millisecond, Ease: motion.ToastEase}
	toastFade  = ui.Animation{Duration: 200 * time.Millisecond, Ease: motion.EaseOut}
	toastSize  = ui.Animation{Duration: 150 * time.Millisecond, Ease: motion.CSSEase}
)

// ToastHost shows the toasts ShowToast adds; call it once in the view. The newest shows in front
// with the older ones peeking behind it, and the pointer resting on them spreads them out.
func ToastHost(c *ui.Context, p ToastHostProps) ui.Element {
	tk := theme.TokensOf(c)
	top := p.Placement >= ToastTopRight
	return ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(16)
		switch p.Placement {
		case ToastBottomLeft, ToastTopLeft:
			viewport.AlignItems(ui.Start)
		case ToastBottom, ToastTop:
			viewport.AlignItems(ui.Center)
		default:
			viewport.AlignItems(ui.End)
		}
		if top {
			viewport.Justify(ui.Start)
		} else {
			viewport.Justify(ui.End)
		}

		stack := ui.Box(c).Width(toastWidth).MaxWidthPercent(100)
		heights := *ui.Local(stack, "heights", func() map[string]float32 { return map[string]float32{} })
		expanded := stack.Hovered()
		n := len(toasts)
		front := float32(0)
		if n > 0 {
			front = heights[toasts[n-1].ID]
		}
		total := front + toastPeek*float32(max(n-1, 0))
		if expanded {
			total = 0
			for _, t := range toasts {
				total += heights[t.ID]
			}
			total += toastGap * float32(max(n-1, 0))
		}
		stack.Height(total)

		layouts := make([]toastLayout, n)
		offset := float32(0)
		for k := n - 1; k >= 0; k-- {
			depth := n - 1 - k
			scale := 1 - 0.05*float32(depth)
			l := toastLayout{depth: depth, y: float32(depth)*toastPeek + (1-scale)*front, scale: scale,
				collapsed: !expanded && depth > 0, front: front, top: top}
			if expanded {
				l.y, l.scale = offset, 1
			}
			offset += heights[toasts[k].ID] + toastGap
			layouts[k] = l
		}
		stack.Children(func() {
			for k, t := range toasts {
				toast(c, tk, t, heights, layouts[k])
			}
		})
	})
}

type toastLayout struct {
	depth           int
	y, scale, front float32
	collapsed, top  bool
}

// The stack moves its toasts with transforms of their own; their Transition plays only their Enter
// and Exit, as a layout move animated too would drag the front toast along while the stack resizes.
func toast(c *ui.Context, tk *theme.Tokens, t ui.Toast, heights map[string]float32, l toastLayout) {
	parts := ui.ToastBase(c, t)
	root := parts.Root.Absolute().Left(0).Right(0).Radius(tk.BorderRadiusLG).Clip().
		Background(tk.BgElevated).TextColor(tk.Text).FontSize(14)
	tk.Elevate(root, theme.BoxShadowSecondary)
	enter := motion.Toast(motion.Bottom)
	y := -l.y
	if l.top {
		enter = motion.Toast(motion.Top)
		root.Top(0).TransformOrigin(0.5, 0)
		y = l.y
	} else {
		root.Bottom(0).TransformOrigin(0.5, 1)
	}
	enter.Colors = true
	root.Transition(enter)
	height := heights[t.ID]
	if l.collapsed && l.front > 0 {
		height = l.front
	}
	if height > 0 {
		root.Height(root.AnimateTo("height", height, toastSize))
	}
	root.Translate(0, root.AnimateTo("y", y, toastStack))
	scale := root.AnimateTo("scale", l.scale, toastStack)
	root.Scale(scale, scale)

	shown := float32(1)
	if l.collapsed {
		shown = 0
	}
	var body ui.Element
	root.Children(func() {
		body = ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Start).Shrink(0).
			Opacity(root.AnimateTo("content", shown, toastFade))
		body.Children(func() {
			icon, color := toastIcon(tk, t.Type)
			ui.Row(c).Height(21).AlignItems(ui.Center).Shrink(0).Children(func() {
				ui.Icon(c, icon()).FontSize(18).TextColor(color)
			})
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
					switch {
					case t.Title != "":
						ui.Text(c, t.Title).FontWeight(500).LineHeight(1.5).Grow(1)
					default:
						ui.Text(c, t.Description).LineHeight(1.5).Grow(1)
					}
					toastClose(c, tk, parts.CloseButton())
				})
				if t.Title != "" && t.Description != "" {
					ui.Text(c, t.Description).FontSize(13).LineHeight(1.5).TextColor(tk.TextSecondary).Margin(4, 0, 0, 0)
				}
				if t.Action != "" {
					ui.Row(c).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
						toastAction(c, tk, parts.ActionButton(), t.Action)
					})
				}
			})
		})
	})
	if h := body.Bounds().H; h > 0 && h != heights[t.ID] {
		heights[t.ID] = h
		c.Invalidate()
	}
}

func toastIcon(tk *theme.Tokens, typ string) (func() *ui.SVG, ui.Color) {
	switch typ {
	case "info":
		return lucide.Info, tk.Info
	case "success":
		return lucide.CircleCheck, tk.Success
	case "warning":
		return lucide.TriangleAlert, tk.Warning
	case "error":
		return lucide.CircleX, tk.Error
	case "loading":
		return lucide.LoaderCircle, tk.TextSecondary
	}
	return lucide.Info, tk.Text
}

func toastClose(c *ui.Context, tk *theme.Tokens, b ui.Element) {
	b.Label("Close").Size(20, 20).Radius(tk.BorderRadiusSM).Shrink(0).Margin(1, 0, 0, 0).
		Cursor(ui.CursorPointer).Transition(dialogCloseTransition)
	if b.Hovered() {
		b.Background(tk.FillSecondary).TextColor(tk.Text)
	} else {
		b.TextColor(tk.TextSecondary)
	}
	b.Children(func() { ui.Icon(c, lucide.X()).FontSize(14) })
}

func toastAction(c *ui.Context, tk *theme.Tokens, b ui.Element, label string) {
	bg := tk.Primary
	switch {
	case b.Pressed():
		bg = tk.PrimaryActive
	case b.Hovered():
		bg = tk.PrimaryHover
	}
	b.Height(28).PaddingX(12).Radius(tk.BorderRadiusSM).Background(bg).TextColor(tk.BgLayout).
		FontSize(12).FontWeight(500).Cursor(ui.CursorPointer).Transition(dialogCloseTransition)
	b.Children(func() { ui.Text(c, label).SingleLine() })
}
