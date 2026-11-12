package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestGeneratedTokensMatchLobeUI(t *testing.T) {
	cases := []struct {
		name string
		got  ui.Color
		want ui.Color
	}{
		{"light Text", Light().Text, ui.Hex("#1f1f1f")},
		{"light BgContainer", Light().BgContainer, ui.Hex("#ffffff")},
		{"light Fill", Light().Fill, ui.RGBA(0, 0, 0, 0.12)},
		{"light Primary", Light().Primary, ui.Hex("#222222")},
		{"dark Text", Dark().Text, ui.Hex("#e6e6e6")},
		{"dark BgContainer", Dark().BgContainer, ui.Hex("#0d0d0d")},
		{"dark FillSecondary", Dark().FillSecondary, ui.RGBA(255, 255, 255, 0.10)},
		{"dark BgLayout", Dark().BgLayout, ui.Hex("#000000")},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %+v, want %+v", c.name, c.got, c.want)
		}
	}
}

func TestGeneratedMetricsMatchLobeUI(t *testing.T) {
	l := Light()
	if l.BorderRadius != 8 || l.BorderRadiusSM != 6 || l.BorderRadiusLG != 12 {
		t.Errorf("radii = %v/%v/%v, want 8/6/12", l.BorderRadius, l.BorderRadiusSM, l.BorderRadiusLG)
	}
	if l.FontSize != 14 || l.FontSizeSM != 12 {
		t.Errorf("font sizes = %v/%v, want 14/12", l.FontSize, l.FontSizeSM)
	}
}

func TestColorLookupByName(t *testing.T) {
	l := Light()
	if got, ok := l.Color("Text"); !ok || got != l.Text {
		t.Errorf("Color(Text) = %+v, %v; want %+v, true", got, ok, l.Text)
	}
	if _, ok := l.Color("Nope"); ok {
		t.Error("Color(Nope) reported found")
	}
	names := map[string]bool{}
	for _, n := range ColorTokenNames() {
		names[n] = true
	}
	for _, want := range []string{"Text", "BgContainer", "Primary", "Error"} {
		if !names[want] {
			t.Errorf("ColorTokenNames lacks %s", want)
		}
	}
	if names["Yellow9"] {
		t.Error("ColorTokenNames includes palette scale Yellow9")
	}
}

func TestTokensOfFollowsTheme(t *testing.T) {
	var got *Tokens
	tt := ui.NewTester(func(c *ui.Context) {
		Apply(c)
		got = TokensOf(c)
	}, 100, 100)
	if got == nil || got.Dark {
		t.Fatalf("light: TokensOf = %+v", got)
	}
	tt.SetDark(true)
	tt.Frame()
	if !got.Dark || got.BgContainer != Dark().BgContainer {
		t.Errorf("dark: TokensOf Dark=%v", got.Dark)
	}
}

func TestTokensOfResultIsACopy(t *testing.T) {
	var first, second *Tokens
	tt := ui.NewTester(func(c *ui.Context) {
		Apply(c)
		if first == nil {
			first = TokensOf(c)
			first.Text = ui.RGB(1, 2, 3)
		} else {
			second = TokensOf(c)
		}
	}, 100, 100)
	tt.Frame()
	if second == nil {
		t.Fatal("view built only once")
	}
	if second.Text != ui.Hex("#1f1f1f") {
		t.Errorf("mutating one TokensOf result changed the next: Text = %+v", second.Text)
	}
}

func TestPresetPaletteTokens(t *testing.T) {
	l := Light()
	cases := []struct {
		name      string
		got, want ui.Color
	}{
		{"Red", l.Red, ui.Hex("#f4416c")},
		{"RedActive", l.RedActive, ui.Hex("#d6225d")},
		{"RedFillTertiary", l.RedFillTertiary, ui.RGBA(255, 15, 15, 0.05)},
		{"Gray", l.Gray, ui.Hex("#888888")},
		{"GeekblueFillQuaternary", l.GeekblueFillQuaternary, l.GeekblueFillQuaternary},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %+v, want %+v", c.name, c.got, c.want)
		}
	}
	if got, ok := l.Color("RedActive"); !ok || got != l.RedActive {
		t.Errorf("Color(RedActive) = %+v, %v", got, ok)
	}
	for _, n := range ColorTokenNames() {
		if n == "Red" || n == "GrayFillTertiary" {
			t.Errorf("ColorTokenNames includes palette token %s", n)
		}
	}
}
