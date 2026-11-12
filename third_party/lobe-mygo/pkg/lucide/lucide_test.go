package lucide

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestIconsParseOnceAndDraw(t *testing.T) {
	first := Save()
	if first == nil {
		t.Fatal("Save() returned nil")
	}
	if Save() != first {
		t.Error("Save() parsed twice")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Icon(c, Trash()).Label("Delete")
	}, 64, 64)
	if _, ok := tt.Find("Delete"); !ok {
		t.Error("Trash2 icon did not render")
	}
}

func TestLegacyAliasesStillResolve(t *testing.T) {
	for name, icon := range map[string]func() *ui.SVG{"Smile": Smile, "Trash2": Trash2} {
		if icon() == nil {
			t.Errorf("%s() returned nil", name)
		}
	}
}
