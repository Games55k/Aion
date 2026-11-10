package frontend

import (
	"strings"
	"sync"

	brand "aion/assets/icon"
	"aion/backend"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

var (
	aionSymbolOnce   sync.Once
	aionSymbolParsed *ui.SVG
)

// AionSymbolSVG returns the Aion symbol SVG.
func AionSymbolSVG() *ui.SVG {
	aionSymbolOnce.Do(func() {
		aionSymbolParsed = ui.MustParseSVG([]byte(brand.SVG))
	})
	return aionSymbolParsed
}

// ProjectIconOption defines a selectable project icon preset.
type ProjectIconOption struct {
	Key   string
	Label string
	SVG   func() *ui.SVG
}

// ProjectIconOptions lists the built-in project icon presets.
var ProjectIconOptions = []ProjectIconOption{
	{Key: "folder", Label: "文件夹", SVG: lucide.Folder},
	{Key: "code", Label: "代码", SVG: lucide.Code},
	{Key: "rocket", Label: "发布", SVG: lucide.Rocket},
	{Key: "book", Label: "文档", SVG: lucide.BookOpen},
	{Key: "target", Label: "目标", SVG: lucide.Target},
	{Key: "sparkles", Label: "灵感", SVG: lucide.Sparkles},
	{Key: "layers", Label: "架构", SVG: lucide.Layers},
	{Key: "terminal", Label: "终端", SVG: lucide.Terminal},
	{Key: "briefcase", Label: "工作", SVG: lucide.Briefcase},
	{Key: "heart", Label: "生活", SVG: lucide.Heart},
}

// ProjectIconSVG resolves a stored project icon key to an SVG icon.
func ProjectIconSVG(icon string) *ui.SVG {
	key := strings.TrimSpace(strings.ToLower(icon))
	for _, opt := range ProjectIconOptions {
		if opt.Key == key {
			return opt.SVG()
		}
	}
	return lucide.Folder()
}

// ModuleIconSVG resolves a registered module descriptor to an SVG icon.
func ModuleIconSVG(mod backend.ModuleDescriptor) *ui.SVG {
	switch mod.ID {
	case "projects":
		return lucide.FolderKanban()
	default:
		return lucide.Folder()
	}
}
