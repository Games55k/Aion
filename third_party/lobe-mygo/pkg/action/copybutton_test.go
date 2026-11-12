package action

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
)

func TestCopyButtonCopiesAndConfirms(t *testing.T) {
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) { CopyButton(c, "npm i @lobehub/ui", CopyButtonProps{}) }), 300, 100)
	info, ok := tt.AccessByRole(ui.RoleButton, "Copy")
	if !ok {
		t.Fatal("no Copy button")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
	if got := tt.Clipboard(); got != "npm i @lobehub/ui" {
		t.Errorf("clipboard %q", got)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copied"); !ok {
		t.Error("button does not say Copied after copying")
	}
	tt.Advance(2100 * time.Millisecond)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, "Copy"); !ok {
		t.Error("button still says Copied after 2 seconds")
	}
}
