package layout

import "github.com/egoist/mygo/ui"

type GridProps struct {
	// MaxItemWidth is the narrowest a column goes before the grid drops one: 240 DIPs when zero.
	MaxItemWidth float32
	// MaxColumns is the most columns the grid shows, however wide it is: 3 when zero.
	MaxColumns int
	// Gap is the room between columns and rows: 14 DIPs when zero.
	Gap float32
}

// Grid lays its children out in columns of equal width: as many as fit at MaxItemWidth or wider,
// up to MaxColumns, and at least one. It fills the width it is given; build the items in its
// Children.
func Grid(c *ui.Context, p GridProps) ui.Element {
	itemWidth, maxColumns, gap := p.MaxItemWidth, p.MaxColumns, p.Gap
	if itemWidth == 0 {
		itemWidth = 240
	}
	if maxColumns == 0 {
		maxColumns = 3
	}
	if gap == 0 {
		gap = 14
	}
	g := ui.Grid(c).AlignSelf(ui.Stretch).Gap(gap)
	columns := ui.Local(g, "columns", func() int { return 1 })
	n := max(1, min(maxColumns, int((g.Bounds().W+gap)/(itemWidth+gap))))
	if n != *columns {
		*columns = n
		c.Invalidate()
	}
	return g.Columns(*columns)
}
