package display

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestEmptyStacksIconTitleDescriptionAction(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Children(func() {
			Empty(c, EmptyProps{Icon: lucide.Inbox(), Title: "No messages", Description: "New messages show here.", Action: func() { ui.Text(c, "Compose") }})
		})
	}, 400, 400)
	title, _ := tt.Find("No messages")
	desc, ok := tt.Find("New messages show here.")
	action, ok2 := tt.Find("Compose")
	if !ok || !ok2 {
		t.Fatalf("texts %q", tt.Texts())
	}
	if title.Y < 16+32+8 {
		t.Errorf("title at y %v, want under a 32 DIP icon and 8 DIPs", title.Y)
	}
	if desc.Y < title.Y+title.H || action.Y < desc.Y+desc.H+8 {
		t.Errorf("order: title %+v desc %+v action %+v", title, desc, action)
	}
	for name, r := range map[string]ui.Rect{"title": title, "desc": desc, "action": action} {
		if mid := r.X + r.W/2; mid < 199 || mid > 201 {
			t.Errorf("%s centered at %v, want 200", name, mid)
		}
	}
}

func TestEmptyPageType(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Children(func() { Empty(c, EmptyProps{Title: "Nothing here", Page: true}) })
	}, 400, 400)
	r, _ := tt.Find("Nothing here")
	if r.H < 20 {
		t.Errorf("page title %v tall, want the 18 DIP size", r.H)
	}
}
