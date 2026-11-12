package display

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func avatarTester(avatar string, p AvatarProps) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(10).Children(func() { Avatar(c, avatar, p).Label("avatar").Role(ui.RoleImage) })
	}, 200, 200)
}

func avatarBox(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleImage, "avatar")
	if !ok {
		t.Fatal("no avatar")
	}
	return info.Bounds
}

func TestAvatarSizeAndShape(t *testing.T) {
	tk := theme.Light()
	tt := avatarTester("Ada Lovelace", AvatarProps{Fallback: AvatarText})
	if b := avatarBox(t, tt); b.W != 48 || b.H != 48 {
		t.Errorf("default size %vx%v, want 48", b.W, b.H)
	}
	if got := testkit.PixelAt(tt, 200, 10.5, 10.5); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("square corner %+v, want rounded off (radius 8)", got)
	}
	if got := testkit.PixelAt(tt, 200, 10+4, 10+24); !testkit.Near(got, tk.Border) {
		t.Errorf("text avatar fill %+v, want Border %+v", got, tk.Border)
	}
	tt = avatarTester("Ada Lovelace", AvatarProps{Fallback: AvatarText, Size: 32, Shape: AvatarCircle})
	if b := avatarBox(t, tt); b.W != 32 {
		t.Errorf("size 32: %v", b.W)
	}
	if got := testkit.PixelAt(tt, 200, 10+2, 10+2); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("circle near its corner %+v, want background", got)
	}
}

func TestAvatarTextShowsFirstTwoLetters(t *testing.T) {
	tt := avatarTester("ada lovelace", AvatarProps{Fallback: AvatarText})
	if !tt.HasText("AD") {
		t.Errorf("texts %q, want AD", tt.Texts())
	}
	tt = avatarTester("Lobe", AvatarProps{Fallback: AvatarText, Full: true})
	if !tt.HasText("LOBE") {
		t.Errorf("texts %q, want the whole text LOBE", tt.Texts())
	}
}

func TestAvatarMarbleIsStableAndDiffersBySeed(t *testing.T) {
	px := func(name string) ui.Color {
		return testkit.PixelAt(avatarTester(name, AvatarProps{}), 200, 10+24, 10+24)
	}
	if a, b := px("ada"), px("ada"); a != b {
		t.Errorf("same seed painted %+v and %+v", a, b)
	}
	if a, b := px("ada"), px("grace"); a == b {
		t.Errorf("different seeds painted the same %+v", a)
	}
	if tt := avatarTester("ada", AvatarProps{}); tt.HasText("AD") {
		t.Error("marble avatar shows text")
	}
}

func TestAvatarEmoji(t *testing.T) {
	tt := avatarTester("🦊", AvatarProps{})
	if !tt.HasText("🦊") {
		t.Errorf("texts %q, want the emoji", tt.Texts())
	}
}
