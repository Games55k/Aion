package nav

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type StepStatus int

const (
	StepAuto StepStatus = iota
	StepWait
	StepProcess
	StepFinish
	StepError
)

type StepItem struct {
	// Title and Description are what the step says.
	Title, Description string
	// Icon replaces the step's number or check.
	Icon *ui.SVG
	// Status overrides the status Current gives the step.
	Status StepStatus
}

type StepsProps struct {
	// Items are the steps, in order.
	Items []StepItem
	// Current is the index of the step in process: the ones before it are finished and the ones
	// after wait. Negative leaves every step without a status, in a neutral look.
	Current int
	// Vertical stacks the steps with the connector running down beside them.
	Vertical bool
	// Dot shows a small dot for each step instead of a numbered circle.
	Dot bool
}

const stepGuide StepStatus = -1

func stepStatus(i, current int, it StepItem) StepStatus {
	switch {
	case it.Status != StepAuto:
		return it.Status
	case current < 0:
		return stepGuide
	case i < current:
		return StepFinish
	case i == current:
		return StepProcess
	}
	return StepWait
}

// Steps shows the steps of a process, numbered circles linked by connectors that turn primary
// after the steps finished.
func Steps(c *ui.Context, p StepsProps) ui.Element {
	tk := theme.TokensOf(c)
	var root ui.Element
	if p.Vertical {
		root = ui.Column(c).AlignSelf(ui.Stretch)
	} else {
		root = ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch)
	}
	root.Children(func() {
		for i, it := range p.Items {
			status := stepStatus(i, p.Current, it)
			last := i == len(p.Items)-1
			item := ui.Row(c.Key(i)).Gap(8).AlignItems(ui.Center).MinWidth(0)
			if p.Vertical {
				item.Gap(12).AlignItems(ui.Start)
				if !last {
					item.Padding(0, 0, 18, 0)
				}
			} else if !last {
				item.Grow(1).Basis(0)
			}
			item.Children(func() {
				stepIndicator(c, tk, i, it, status, p.Dot)
				if it.Title != "" || it.Description != "" {
					body := ui.Column(c).Gap(2).MinWidth(0)
					if p.Vertical {
						body.Grow(1)
					}
					body.Children(func() {
						if it.Title != "" {
							title := ui.Text(c, it.Title).FontSize(14).LineHeight(22.0 / 14)
							switch status {
							case StepError:
								title.TextColor(tk.Error)
							case StepWait:
								title.TextColor(tk.TextDescription)
							case StepProcess:
								title.TextColor(tk.Text).FontWeight(500)
							default:
								title.TextColor(tk.Text)
							}
							if !p.Vertical {
								title.SingleLine()
							}
						}
						if it.Description != "" {
							ui.Text(c, it.Description).FontSize(13).LineHeight(1.5).TextColor(tk.TextSecondary)
						}
					})
				}
				if !last {
					line := tk.BorderSecondary
					if status == StepFinish {
						line = tk.Primary
					}
					conn := ui.Box(c).Background(line)
					switch {
					case p.Vertical && p.Dot:
						conn.Absolute().Top(20).Bottom(4).Left(3).Width(1)
					case p.Vertical:
						conn.Absolute().Top(26).Bottom(4).Left(11).Width(1)
					default:
						conn.Height(1).Grow(1).MinWidth(16)
					}
				}
			})
		}
	})
	return root
}

func stepIndicator(c *ui.Context, tk *theme.Tokens, i int, it StepItem, status StepStatus, dot bool) {
	if dot {
		bg := tk.TextQuaternary
		if status == StepProcess || status == StepFinish {
			bg = tk.Primary
		}
		ui.Box(c).Size(7, 7).Radius(3.5).Shrink(0).Background(bg).Margin(8, 0, 0, 0)
		return
	}
	border, fg := tk.Border, tk.TextDescription
	ind := ui.Row(c).Size(22, 22).Radius(11).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).
		FontSize(12).FontWeight(500)
	switch status {
	case StepError:
		border, fg = tk.Error, tk.Error
	case StepFinish:
		border, fg = tk.Primary, tk.Primary
	case StepProcess:
		border, fg = tk.Primary, tk.BgContainer
		ind.Background(tk.Primary)
	case stepGuide:
		fg = tk.TextSecondary
	}
	ind.Border(1, border).TextColor(fg).Children(func() {
		switch {
		case it.Icon != nil:
			ui.Icon(c, it.Icon).FontSize(12)
		case status == StepFinish:
			ui.Icon(c, lucide.Check()).FontSize(12)
		default:
			ui.Text(c, strconv.Itoa(i+1)).FontFeatures("tnum")
		}
	})
}
