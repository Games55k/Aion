package code

import (
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestHighlighterDoesNotRehighlightEachFrame(t *testing.T) {
	big := strings.Repeat("func f(a int) string { return \"x\" } // c\n", 2000)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		Highlighter(c, big, HighlighterProps{Language: "go"})
	}, 600, 400)
	tt.Frame()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 5 {
		tt.Frame()
	}
	runtime.ReadMemStats(&after)
	perFrame := (after.TotalAlloc - before.TotalAlloc) / 5
	t.Logf("%d bytes a frame", perFrame)
	if perFrame > 1<<20 {
		t.Errorf("%d bytes allocated a frame for unchanged code: it is highlighted again", perFrame)
	}
}

func TestSnippetDoesNotRehighlightEachFrame(t *testing.T) {
	long := strings.Repeat("echo hi && ", 2000)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		Snippet(c, long, SnippetProps{Language: "sh"})
	}, 600, 200)
	tt.Frame()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 5 {
		tt.Frame()
	}
	runtime.ReadMemStats(&after)
	if perFrame := (after.TotalAlloc - before.TotalAlloc) / 5; perFrame > 1<<20 {
		t.Errorf("%d bytes allocated a frame for an unchanged snippet", perFrame)
	}
}
