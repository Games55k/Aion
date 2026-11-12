package layout

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/motion"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type BlockVariant int

const (
	BlockFilled BlockVariant = iota
	BlockOutlined
	BlockBorderless
)

type BlockProps struct {
	// Variant picks the surface: Filled tints it, Outlined puts it on the container color inside a
	// border, Borderless has none.
	Variant BlockVariant
	// Clickable makes the block a button that deepens its surface under the pointer; ask the
	// returned element whether it was Clicked.
	Clickable bool
	// Shadow lifts the block with a soft layered shadow in the border colors.
	Shadow bool
}

var blockTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Ease: motion.EaseOut, Colors: true}

// Block is a rounded column on a Lobe UI surface; give it its size and children.
func Block(c *ui.Context, p BlockProps) ui.Element {
	tk := theme.TokensOf(c)
	var b ui.Element
	if p.Clickable {
		b = ui.ButtonBase(c).Column().AlignItems(ui.Stretch).Justify(ui.Start).Cursor(ui.CursorPointer)
	} else {
		b = ui.Box(c)
	}
	b.Radius(tk.BorderRadius).Transition(blockTransition)
	hover := p.Clickable && b.Hovered()
	switch p.Variant {
	case BlockOutlined:
		border := tk.BorderSecondary
		if hover {
			border = tk.Border
		}
		b.Border(1, border).Background(tk.BgContainer)
	case BlockBorderless:
		if hover {
			b.Background(tk.FillTertiary)
		}
	default:
		if hover {
			b.Background(tk.FillSecondary)
		} else {
			b.Background(tk.FillTertiary)
		}
	}
	if p.Shadow {
		b.Shadow(0, 1, 0, -1, tk.Border).Shadow(0, 1, 2, -0.5, tk.Border).
			Shadow(0, 2, 2, -1, tk.BorderSecondary).Shadow(0, 3, 6, -4, tk.BorderSecondary)
	}
	return b
}
