package field

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func numberTester(v *float64, p InputNumberProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			InputNumber(c, v, p).Label("count")
			ui.Button(c, "Elsewhere")
		})
	}, 400, 200)
}

func clickRole(t *testing.T, tt *ui.Tester, role ui.Role, name string) {
	t.Helper()
	info, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("no %v %q", role, name)
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Frame()
}

func TestInputNumberTypesAndCommits(t *testing.T) {
	v := 3.0
	tt := numberTester(&v, InputNumberProps{Min: 0, Max: 10})
	tt.Click("count")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("42")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if v != 10 {
		t.Errorf("value %v after typing 42 into 0–10, want it clamped to 10", v)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if v != 9 {
		t.Errorf("value %v after Down, want 9", v)
	}
}

func TestInputNumberControlsStep(t *testing.T) {
	v := 1.0
	tt := numberTester(&v, InputNumberProps{Step: 0.5, Precision: 2})
	clickRole(t, tt, ui.RoleButton, "Increase")
	clickRole(t, tt, ui.RoleButton, "Increase")
	if v != 2 {
		t.Fatalf("value %v after two increases by 0.5, want 2", v)
	}
	clickRole(t, tt, ui.RoleButton, "Decrease")
	if v != 1.5 {
		t.Errorf("value %v after a decrease, want 1.5", v)
	}
	if info, _ := tt.AccessByRole(ui.RoleTextField, "count"); info.Value != "1.50" {
		t.Errorf("field shows %q, want 1.50 at precision 2", info.Value)
	}
}

func TestInputPasswordHidesUntilShown(t *testing.T) {
	v := "hunter2"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() { InputPassword(c, &v, InputProps{}).Label("password") })
	}, 400, 200)
	shown := func() string { info, _ := tt.AccessByRole(ui.RoleTextField, "password"); return info.Value }
	if shown() == "hunter2" {
		t.Fatal("password shows in plain text")
	}
	clickRole(t, tt, ui.RoleButton, "Show password")
	if got := shown(); got != "hunter2" {
		t.Errorf("field holds %q after Show password", got)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Hide password"); !ok {
		t.Error("toggle not renamed to Hide password")
	}
}
