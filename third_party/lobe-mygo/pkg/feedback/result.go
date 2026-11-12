package feedback

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type ResultStatus int

const (
	ResultInfo ResultStatus = iota
	ResultSuccess
	ResultWarning
	ResultError
)

type ResultProps struct {
	// Status picks the icon and its color.
	Status ResultStatus
	// Title and SubTitle say what happened.
	Title, SubTitle string
	// Icon replaces the status's icon, without its circle.
	Icon *ui.SVG
	// Extra builds the actions below, as buttons.
	Extra func()
}

// Result shows the outcome of an operation: an icon in a tinted circle, a title, a subtitle and
// actions, centered.
func Result(c *ui.Context, p ResultProps) ui.Element {
	tk := theme.TokensOf(c)
	glyph, fg, bg := lucide.Info(), tk.TextSecondary, tk.FillTertiary
	switch p.Status {
	case ResultSuccess:
		glyph, fg, bg = lucide.Check(), tk.Success, tk.Success.Alpha(0.12)
	case ResultWarning:
		glyph, fg, bg = lucide.TriangleAlert(), tk.Warning, tk.Warning.Alpha(0.12)
	case ResultError:
		glyph, fg, bg = lucide.X(), tk.Error, tk.Error.Alpha(0.12)
	}
	root := ui.Column(c).Gap(6).Padding(16, 8).AlignItems(ui.Center).FillWidth()
	root.Children(func() {
		if p.Icon != nil {
			ui.Icon(c, p.Icon).FontSize(72).TextColor(fg).Margin(0, 0, 12, 0)
		} else {
			ui.Row(c).Size(72, 72).Radius(36).Center().Background(bg).TextColor(fg).Margin(0, 0, 12, 0).Shrink(0).
				Children(func() { ui.Icon(c, glyph).FontSize(36) })
		}
		if p.Title != "" {
			ui.Text(c, p.Title).FontSize(20).FontWeight(600).TextColor(tk.Text).TextAlign(ui.Center)
		}
		if p.SubTitle != "" {
			ui.Text(c, p.SubTitle).FontSize(13).TextColor(tk.TextTertiary).TextAlign(ui.Center).MaxWidth(13 * 0.55 * 36)
		}
		if p.Extra != nil {
			ui.Row(c).Gap(8).Wrap().Justify(ui.Center).AlignItems(ui.Center).Margin(10, 0, 0, 0).Children(p.Extra)
		}
	})
	return root
}
