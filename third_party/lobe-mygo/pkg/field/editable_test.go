package field

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func editableTester(v *string, editing *bool, p EditableTextProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(360).Children(func() { EditableText(c, v, editing, p) })
	}, 400, 200)
}

func TestEditableTextEditsAndSaves(t *testing.T) {
	v, editing := "Lobe", false
	var ended []string
	tt := editableTester(&v, &editing, EditableTextProps{OnChangeEnd: func(s string) { ended = append(ended, s) }})
	if !tt.HasText("Lobe") {
		t.Fatalf("texts %q", tt.Texts())
	}
	clickRole(t, tt, ui.RoleButton, "Edit")
	if !editing {
		t.Fatal("Edit did not start editing")
	}
	tt.Type(" UI")
	tt.Frame()
	if v != "Lobe" {
		t.Errorf("value %q changed before saving", v)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Reset"); !ok {
		t.Error("no Reset button with a changed draft")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v != "Lobe UI" || editing || len(ended) != 1 {
		t.Errorf("after Enter: value %q editing %v ended %q", v, editing, ended)
	}
}

func TestEditableTextEscapeCancels(t *testing.T) {
	v, editing := "Lobe", false
	tt := editableTester(&v, &editing, EditableTextProps{})
	clickRole(t, tt, ui.RoleButton, "Edit")
	tt.Type("!!")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if v != "Lobe" || editing {
		t.Errorf("after Escape: value %q editing %v", v, editing)
	}
	editing = true
	tt.Frame()
	if info, _ := tt.AccessByRole(ui.RoleTextField, ""); info.Value != "Lobe" {
		t.Errorf("editing started from outside shows %q, want the value, not the cancelled draft", info.Value)
	}
}
