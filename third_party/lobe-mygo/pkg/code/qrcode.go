package code

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"rsc.io/qr"
)

type QRLevel int

const (
	// QRLevelM recovers from 15% of the code damaged.
	QRLevelM QRLevel = iota
	// QRLevelL recovers from 7%, for the smallest code.
	QRLevelL
	// QRLevelQ recovers from 25%.
	QRLevelQ
	// QRLevelH recovers from 30%, as a code with an Icon over its middle needs.
	QRLevelH
)

type QRStatus int

const (
	QRActive QRStatus = iota
	// QRLoading covers the code with a spinner.
	QRLoading
	// QRExpired covers the code with "Expired" and, given OnRefresh, a Refresh button.
	QRExpired
)

type QRCodeProps struct {
	// Size is the code's width and height, without its 16 DIPs of padding: 160 DIPs when zero.
	Size float32
	// Color colors the modules: black when zero.
	Color ui.Color
	// Background fills the card behind them: white when zero.
	Background ui.Color
	// NoBorder leaves out the hairline around the card.
	NoBorder bool
	// ErrorLevel is how much of the code may be damaged and still read; an Icon raises it to H.
	ErrorLevel QRLevel
	// Icon builds a logo in a circle over the middle of the code, a fifth of its size, the
	// modules under it left out.
	Icon func(c *ui.Context)
	// Status covers an expired code or one being fetched.
	Status QRStatus
	// OnRefresh adds a Refresh button to an expired code.
	OnRefresh func()
}

const qrIconRatio = 0.22

var qrLevels = map[QRLevel]qr.Level{QRLevelL: qr.L, QRLevelM: qr.M, QRLevelQ: qr.Q, QRLevelH: qr.H}

func qrModules(value string, level QRLevel, icon bool) [][]bool {
	if icon {
		level = QRLevelH
	}
	code, err := qr.Encode(value, qrLevels[level])
	if err != nil {
		return nil
	}
	grid := make([][]bool, code.Size)
	for y := range grid {
		grid[y] = make([]bool, code.Size)
		for x := range grid[y] {
			grid[y][x] = code.Black(x, y)
		}
	}
	return grid
}

func qrPath(grid [][]bool, hole int) *ui.Path {
	unit := func(k int) float32 { return float32(k) }
	return qrPathAt(grid, hole, unit, unit)
}

func qrPathAt(grid [][]bool, hole int, atX, atY func(k int) float32) *ui.Path {
	n := len(grid)
	from := (n - hole) / 2
	inHole := func(x, y int) bool { return hole > 0 && x >= from && x < from+hole && y >= from && y < from+hole }
	path := new(ui.Path)
	for y, row := range grid {
		for x := 0; x < n; {
			if !row[x] || inHole(x, y) {
				x++
				continue
			}
			end := x
			for end < n && row[end] && !inHole(end, y) {
				end++
			}
			path.MoveTo(atX(x), atY(y)).LineTo(atX(end), atY(y)).LineTo(atX(end), atY(y+1)).LineTo(atX(x), atY(y+1)).Close()
			x = end
		}
	}
	return path
}

type qrState struct {
	value string
	level QRLevel
	icon  bool
	grid  [][]bool
	hole  int
	drawn struct {
		r     ui.Rect
		scale float32
		path  *ui.Path
	}
}

// QRCode shows value as a QR code on a white card, as a link to scan or a login to approve.
func QRCode(c *ui.Context, value string, p QRCodeProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 160
	}
	ink, paper := p.Color, p.Background
	if ink == (ui.Color{}) {
		ink = ui.RGB(0, 0, 0)
	}
	if paper == (ui.Color{}) {
		paper = ui.RGB(255, 255, 255)
	}
	root := ui.Box(c).Padding(16).Radius(16).Background(paper).Shrink(0)
	if !p.NoBorder {
		root.InsetShadow(0, 0, 0, 1, tk.BorderSecondary)
	}
	st := ui.Local(root, "qr", func() qrState { return qrState{} })
	icon := p.Icon != nil
	if st.grid == nil || st.value != value || st.level != p.ErrorLevel || st.icon != icon {
		grid := qrModules(value, p.ErrorLevel, icon)
		hole := 0
		if icon {
			hole = int(float32(len(grid))*qrIconRatio+0.999) | 1
		}
		*st = qrState{value: value, level: p.ErrorLevel, icon: icon, grid: grid, hole: hole}
	}
	return root.Children(func() {
		code := ui.Box(c).Size(size, size).Role(ui.RoleImage).Label(value)
		if len(st.grid) > 0 {
			code.Draw(func(pt *ui.Painter, r ui.Rect) {
				d := &st.drawn
				if d.path == nil || d.r != r || d.scale != pt.Scale() {
					s, unit := pt.Scale(), r.W/float32(len(st.grid))
					snap := func(v float32) float32 { return float32(int(v*s+0.5)) / s }
					d.r, d.scale = r, s
					d.path = qrPathAt(st.grid, st.hole,
						func(k int) float32 { return snap(r.X + float32(k)*unit) },
						func(k int) float32 { return snap(r.Y + float32(k)*unit) })
				}
				pt.FillPath(d.path, ink)
			})
		}
		if icon {
			d := size * qrIconRatio
			ui.Box(c).Size(d, d).Absolute().Left(16+(size-d)/2).Top(16+(size-d)/2).Radius(d/2).Clip().
				Shadow(0, 0, 0, 4, ui.RGB(255, 255, 255)).Children(func() { p.Icon(c) })
		}
		switch p.Status {
		case QRLoading:
			qrOverlay(c).Children(func() {
				feedback.Spin(c, feedback.SpinProps{}, nil).Label("Loading").AlignSelf(ui.Center).TextColor(ui.Hex("#666666"))
			})
		case QRExpired:
			qrOverlay(c).Children(func() {
				ui.Text(c, "Expired").FontSize(17).FontWeight(600)
				if p.OnRefresh != nil {
					qrRefresh(c, p.OnRefresh)
				}
			})
		}
	})
}

func qrOverlay(c *ui.Context) ui.Element {
	return ui.Column(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).Radius(16).Gap(12).
		AlignItems(ui.Center).Justify(ui.Center).Background(ui.RGBA(255, 255, 255, 0.95)).TextColor(ui.Hex("#080808"))
}

func qrRefresh(c *ui.Context, onRefresh func()) {
	b := ui.ButtonBase(c).Label("Refresh").Row().Gap(6).AlignItems(ui.Center).Height(32).PaddingX(14).Radius(999).
		Background(ui.Hex("#080808")).TextColor(ui.RGB(255, 255, 255)).FontSize(13).FontWeight(500).Cursor(ui.CursorPointer)
	if b.Clicked() {
		onRefresh()
	}
	b.Children(func() {
		ui.Icon(c, lucide.RotateCw()).FontSize(14)
		ui.Text(c, "Refresh")
	})
}
