package lucide

//go:generate bun ../../scripts/gen-lucide.ts

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

const (
	svgOpen  = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`
	svgClose = `</svg>`
)

var (
	mu    sync.Mutex
	cache = map[string]*ui.SVG{}
)

func load(name, shapes string) *ui.SVG {
	mu.Lock()
	defer mu.Unlock()
	if svg, ok := cache[name]; ok {
		return svg
	}
	svg := ui.MustParseSVG([]byte(svgOpen + shapes + svgClose))
	cache[name] = svg
	return svg
}
