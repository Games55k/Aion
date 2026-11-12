package form

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type profile struct {
	Name, Email string
	Tags        []string
}

type harness struct {
	t         *testing.T
	tt        *ui.Tester
	f         *Form[profile]
	submitted []profile
	changes   int
	taken     map[string]bool
}

func newHarness(t *testing.T, p Props[profile]) *harness {
	h := &harness{t: t, f: New(profile{Name: "Ada"}), taken: map[string]bool{"admin": true}}
	p.OnSubmit = func(v profile) error { h.submitted = append(h.submitted, v); return nil }
	if p.ChangeDebounce > 0 {
		p.OnValuesChange = func(profile) { h.changes++ }
	}
	h.tt = ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(600).Children(func() {
			View(c, h.f, p, func() {
				Group(c, h.f, GroupProps{Title: "Profile"}, func() {
					Field(c, h.f, FieldProps{Name: "name", Label: "Name", Desc: "Shown to other members", Required: true, Value: h.f.Values.Name}, func() {
						field.Input(c, &h.f.Values.Name, field.InputProps{Placeholder: "Name"}).Label("name input")
					})
					Field(c, h.f, FieldProps{Name: "email", Label: "Email", Value: h.f.Values.Email,
						Validate: func() string {
							if h.f.Values.Email != "" && !strings.Contains(h.f.Values.Email, "@") {
								return "Not an email"
							}
							return ""
						},
						ValidateAsync: func(ctx context.Context, v any) string {
							if h.taken[strings.TrimSuffix(v.(string), "@x.io")] {
								return "Taken"
							}
							return ""
						}}, func() {
						field.Input(c, &h.f.Values.Email, field.InputProps{Placeholder: "Email"}).Label("email input")
					})
				})
				SubmitFooter(c, h.f, SubmitFooterProps{})
			})
		})
	}, 640, 480)
	return h
}

func (h *harness) focus(label string) {
	h.t.Helper()
	r, ok := h.tt.Find(label)
	if !ok {
		h.t.Fatalf("no %q in %q", label, h.tt.Texts())
	}
	h.tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	h.tt.Frame()
}

func (h *harness) settle() {
	for i := 0; i < 20; i++ {
		h.tt.Advance(100 * time.Millisecond)
		h.tt.Frame()
	}
}

func (h *harness) click(name string) {
	h.t.Helper()
	info, ok := h.tt.AccessByRole(ui.RoleButton, name)
	if !ok {
		h.t.Fatalf("no %q button; texts %q", name, h.tt.Texts())
	}
	h.tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	h.settle()
}

func TestFieldLayoutAndRequiredMark(t *testing.T) {
	h := newHarness(t, Props[profile]{})
	label, _ := h.tt.Find("Name")
	desc, _ := h.tt.Find("Shown to other members")
	input, _ := h.tt.Find("name input")
	if !h.tt.HasText("*") {
		t.Errorf("no required mark: %q", h.tt.Texts())
	}
	if desc.Y < label.Y+label.H-0.5 || desc.X != label.X {
		t.Errorf("description %v not under label %v", desc, label)
	}
	if input.X < label.X+label.W || abs(input.X+input.W+12+1-580) > 1 {
		t.Errorf("control %v: want it ending at the form's right edge, right of the label", input)
	}
	title, _ := h.tt.Find("Profile")
	if label.Y < title.Y+title.H+16 {
		t.Errorf("first field at %v right under the group title %v", label.Y, title)
	}
}

func TestRequiredShowsOnBlurAfterAnEdit(t *testing.T) {
	h := newHarness(t, Props[profile]{})
	h.f.ResetTo(profile{})
	h.tt.Frame()
	h.focus("name input")
	h.focus("email input")
	h.settle()
	if h.tt.HasText("This field is required") {
		t.Fatal("an untouched field errs on blur")
	}
	h.focus("name input")
	h.tt.Type("x")
	h.tt.Frame()
	h.tt.Key(0, ui.KeyBackspace)
	h.tt.Frame()
	if h.tt.HasText("This field is required") {
		t.Error("the error shows while still typing")
	}
	h.focus("email input")
	h.settle()
	if !h.tt.HasText("This field is required") {
		t.Fatalf("emptied and left: %q, want the required error", h.tt.Texts())
	}
	h.focus("name input")
	h.tt.Type("B")
	h.tt.Frame()
	if h.tt.HasText("This field is required") {
		t.Error("an erring field did not revalidate as it changed")
	}
}

func TestSubmitValidatesEverything(t *testing.T) {
	h := newHarness(t, Props[profile]{})
	h.f.Values.Name = ""
	h.tt.Frame()
	h.click("Submit")
	if len(h.submitted) != 0 || !h.tt.HasText("This field is required") {
		t.Fatalf("submitted %v texts %q, want blocked by the empty name", h.submitted, h.tt.Texts())
	}
	h.focus("name input")
	h.tt.Type("Grace")
	h.tt.Frame()
	h.click("Submit")
	if len(h.submitted) != 1 || h.submitted[0].Name != "Grace" {
		t.Errorf("submitted %v", h.submitted)
	}
	if h.f.IsDirty() {
		t.Error("still dirty after a submit")
	}
}

func TestAsyncValidationAndDirtyFooter(t *testing.T) {
	h := newHarness(t, Props[profile]{})
	if h.tt.HasText("Unsaved changes") {
		t.Error("unsaved before any change")
	}
	h.focus("email input")
	h.tt.Type("admin@x.io")
	h.tt.Frame()
	if !h.tt.HasText("Unsaved changes") || !h.f.IsDirty() {
		t.Errorf("after typing: %q, want the unsaved notice", h.tt.Texts())
	}
	h.focus("name input")
	h.settle()
	if !h.tt.HasText("Taken") {
		t.Fatalf("texts %q, want the async error", h.tt.Texts())
	}
	h.click("Submit")
	if len(h.submitted) != 0 {
		t.Error("submitted past the async error")
	}
	h.click("Reset")
	if h.f.Values.Email != "" || h.f.IsDirty() || h.tt.HasText("Taken") {
		t.Errorf("reset: values %+v, errors still %q", h.f.Values, h.tt.Texts())
	}
}

func TestValuesChangeDebounces(t *testing.T) {
	h := newHarness(t, Props[profile]{ChangeDebounce: 300 * time.Millisecond})
	h.focus("email input")
	h.tt.Type("a")
	h.tt.Frame()
	h.tt.Advance(100 * time.Millisecond)
	h.tt.Type("b")
	h.tt.Frame()
	h.tt.Advance(100 * time.Millisecond)
	h.tt.Frame()
	if h.changes != 0 {
		t.Errorf("%d changes reported mid-typing", h.changes)
	}
	h.settle()
	if h.changes != 1 {
		t.Errorf("%d changes after the pause, want 1", h.changes)
	}
}

func TestErrorTextColorAndPlacement(t *testing.T) {
	h := newHarness(t, Props[profile]{})
	h.f.Values.Name = ""
	h.tt.Frame()
	h.click("Submit")
	msg, ok := h.tt.Find("This field is required")
	input, _ := h.tt.Find("name input")
	if !ok || msg.Y < input.Y+input.H || abs(msg.X+msg.W-580) > 1 {
		t.Errorf("error %v, want under the control %v, at its right", msg, input)
	}
	tk := theme.Light()
	if got := testkit.Ink(h.tt, 640, msg, tk.BgLayout); !testkit.Near(got, tk.Error) {
		t.Errorf("error ink %+v, want Error", got)
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestResetForgetsServerErrors(t *testing.T) {
	h := newHarness(t, Props[profile]{})
	h.f.SetError("email", "Server says no")
	h.tt.Frame()
	if !h.tt.HasText("Server says no") {
		t.Fatalf("texts %q, want the server error", h.tt.Texts())
	}
	h.focus("name input")
	h.tt.Type("x")
	h.tt.Frame()
	h.click("Reset")
	if h.tt.HasText("Server says no") {
		t.Error("Reset kept a server error on a field it did not change")
	}
}

func TestStaleAsyncResultsAreDropped(t *testing.T) {
	h := &host{fields: map[string]*fieldState{"email": {token: 2, err: ""}}}
	h.results = []asyncResult{{name: "email", token: 1, err: "Taken"}}
	h.drain()
	if got := h.fields["email"].err; got != "" {
		t.Errorf("a result of an older check set %q", got)
	}
}

type signup struct{ Password, Confirm string }

func TestDepsRecheckDependentFields(t *testing.T) {
	f := New(signup{})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(600).Children(func() {
			View(c, f, Props[signup]{}, func() {
				Field(c, f, FieldProps{Name: "password", Label: "Password", Value: f.Values.Password}, func() {
					field.Input(c, &f.Values.Password, field.InputProps{}).Label("password input")
				})
				Field(c, f, FieldProps{Name: "confirm", Label: "Confirm", Value: f.Values.Confirm, Deps: []string{"password"},
					Validate: func() string {
						if f.Values.Confirm != f.Values.Password {
							return "Mismatch"
						}
						return ""
					}}, func() {
					field.Input(c, &f.Values.Confirm, field.InputProps{}).Label("confirm input")
				})
			})
		})
	}, 640, 300)
	click := func(label string) {
		r, _ := tt.Find(label)
		tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
	}
	click("confirm input")
	tt.Type("abc")
	tt.Frame()
	click("password input")
	tt.Frame()
	if !tt.HasText("Mismatch") {
		t.Fatalf("texts %q, want the mismatch after leaving confirm", tt.Texts())
	}
	tt.Type("abc")
	tt.Frame()
	tt.Frame()
	if tt.HasText("Mismatch") {
		t.Error("typing the matching password left confirm's error")
	}
}

func TestBaselineSurvivesInPlaceEdits(t *testing.T) {
	f := New(profile{Tags: []string{"go", "ui"}})
	f.Values.Tags[0] = "rust"
	if !f.IsDirty() {
		t.Fatal("editing a slice element in place left the form clean")
	}
	f.Reset()
	if f.Values.Tags[0] != "go" || f.IsDirty() {
		t.Errorf("reset to %v, dirty %v", f.Values.Tags, f.IsDirty())
	}
	f.Values.Tags[1] = "web"
	if f.IsDirty() == false {
		t.Error("after Reset, Values share the baseline's slice")
	}
}

func TestVerticalLayoutPutsLabelsAbove(t *testing.T) {
	f := New(profile{})
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(600).Children(func() {
			View(c, f, Props[profile]{Layout: Vertical}, func() {
				Field(c, f, FieldProps{Name: "name", Label: "Name", Desc: "Shown to others", Value: f.Values.Name}, func() {
					field.Input(c, &f.Values.Name, field.InputProps{}).Label("name input")
				})
			})
		})
	}, 640, 300)
	desc, _ := tt.Find("Shown to others")
	input, _ := tt.Find("name input")
	if input.Y < desc.Y+desc.H+8-0.5 {
		t.Errorf("control at %v overlaps the description ending at %v", input.Y, desc.Y+desc.H)
	}
	if abs(input.X-(20+12+1)) > 1 {
		t.Errorf("control at x %v, want it stretched from the left edge", input.X)
	}
}

func TestAsyncErrorStaysWhileRechecking(t *testing.T) {
	type slug struct{ Slug string }
	f := New(slug{Slug: "admin"})
	release := make(chan struct{})
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Width(600).Children(func() {
			View(c, f, Props[slug]{}, func() {
				Field(c, f, FieldProps{Name: "slug", Label: "Slug", Value: f.Values.Slug,
					ValidateAsync: func(ctx context.Context, v any) string {
						calls++
						if calls > 1 {
							select {
							case <-release:
							case <-ctx.Done():
							}
						}
						return "Taken"
					}}, func() {
					field.Input(c, &f.Values.Slug, field.InputProps{})
				})
				SubmitFooter(c, f, SubmitFooterProps{})
			})
		})
	}, 640, 300)
	submit := func() {
		info, _ := tt.AccessByRole(ui.RoleButton, "Submit")
		tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	}
	submit()
	for i := 0; i < 10 && !tt.HasText("Taken"); i++ {
		time.Sleep(10 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText("Taken") {
		t.Fatal("no async error after the first submit")
	}
	submit()
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Taken") {
		t.Error("the error vanished while the field was checked again")
	}
	close(release)
}
