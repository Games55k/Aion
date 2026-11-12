package feedback

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type BadgeStatus int

const (
	BadgeDefault BadgeStatus = iota
	BadgeSuccess
	BadgeProcessing
	BadgeError
	BadgeWarning
)

type BadgeProps struct {
	// Count shows on the top right corner of the content; zero hides it unless ShowZero.
	Count int
	// OverflowCount caps the count shown, as "99+": 99 when zero.
	OverflowCount int
	// ShowZero shows a count of zero.
	ShowZero bool
	// Dot shows an 8 DIP dot instead of the count.
	Dot bool
	// Small makes the count 14 DIPs tall instead of 18.
	Small bool
	// Color fills the count or the dot: the error color when unset.
	Color ui.Color
	// Status colors the status dot shown when the badge has no content; Processing pulses.
	Status BadgeStatus
	// Text shows beside the status dot.
	Text string
}

var badgePulse = 1200 * time.Millisecond

// Badge shows a count or a dot on the top right corner of what content builds, or, without
// content, a status dot beside p.Text.
func Badge(c *ui.Context, p BadgeProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
	if content == nil {
		return badgeStatus(c, tk, p)
	}
	root := ui.Box(c).AlignSelf(ui.Start).Shrink(0)
	root.Children(func() {
		content()
		max := p.OverflowCount
		if max <= 0 {
			max = 99
		}
		if !p.Dot && p.Count == 0 && !p.ShowZero {
			return
		}
		bg := p.Color
		if bg == (ui.Color{}) {
			bg = tk.Error
		}
		pill := ui.Row(c).Absolute().Top(0).Right(0).TranslatePercent(50, -50).Background(bg).Center().IgnoreInput(true)
		if p.Dot {
			pill.Size(8, 8).Radius(4)
			return
		}
		h := float32(18)
		if p.Small {
			h = 14
		}
		text := strconv.Itoa(p.Count)
		if p.Count > max {
			text = strconv.Itoa(max) + "+"
		}
		pill.Height(h).MinWidth(h).PaddingX(5).Radius(h/2).Shadow(0, 0, 0, 1.5, tk.BgContainer).
			TextColor(ui.RGB(255, 255, 255)).FontSize(11).FontWeight(500).Font("monospace").FontFeatures("tnum")
		pill.Children(func() { ui.Text(c, text).SingleLine().LineHeight(1) })
	})
	return root
}

func badgeStatus(c *ui.Context, tk *theme.Tokens, p BadgeProps) ui.Element {
	color := p.Color
	if color == (ui.Color{}) {
		color = map[BadgeStatus]ui.Color{
			BadgeDefault: tk.TextQuaternary, BadgeSuccess: tk.Success, BadgeProcessing: tk.Info,
			BadgeError: tk.Error, BadgeWarning: tk.Warning,
		}[p.Status]
	}
	root := ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Start)
	root.Children(func() {
		dot := ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
		if p.Status == BadgeProcessing {
			t := dot.Loop("pulse", badgePulse, ui.EaseOut)
			dot.Children(func() {
				s := 1 + 1.4*t
				ui.Box(c).Absolute().Left(-1).Top(-1).Size(8, 8).Radius(4).Border(1, color).Scale(s, s).Opacity(1 - t)
			})
		}
		if p.Text != "" {
			ui.Text(c, p.Text).TextColor(tk.Text)
		}
	})
	return root
}
