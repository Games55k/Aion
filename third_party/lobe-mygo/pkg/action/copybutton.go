package action

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type CopyButtonProps struct {
	// Variant picks the surface, as ActionIcon's.
	Variant ActionIconVariant
	// Size sets the button and icon size, as ActionIcon's.
	Size theme.Size
	// Icon replaces the copy icon.
	Icon *ui.SVG
}

const copiedFor = 2 * time.Second

// CopyButton copies content to the clipboard when clicked, and shows a check and says "Copied"
// for two seconds after.
func CopyButton(c *ui.Context, content string, p CopyButtonProps) ui.Element {
	b := ui.ButtonBase(c)
	if b.Clicked() {
		c.WriteClipboard(content)
		b.Jump("copied", 1)
	}
	copied := b.AnimateTo("copied", 0, ui.Animation{Duration: copiedFor}) > 0
	icon, title := p.Icon, "Copy"
	if icon == nil {
		icon = lucide.Copy()
	}
	if copied {
		icon, title = lucide.Check(), "Copied"
	}
	return StyleActionIcon(c, b, icon, ActionIconProps{Variant: p.Variant, Size: p.Size, Title: title, Active: copied})
}
