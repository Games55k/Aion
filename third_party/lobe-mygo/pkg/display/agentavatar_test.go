package display

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestAgentTraitsMatchLobe(t *testing.T) {
	cases := []struct {
		seed  string
		shape AgentShape
		color AgentColor
		tilt  float32
		phase time.Duration
	}{
		{"", AgentHex, AgentOrange, 2, -617 * time.Millisecond},
		{"LobeChat", AgentCapsule, AgentGray, -2, -2027 * time.Millisecond},
		{"agent-1", AgentStar, AgentPurple, -1, -843 * time.Millisecond},
		{"Claude", AgentCat, AgentGeekblue, 1, -966 * time.Millisecond},
		{"设计助手", AgentBowl, AgentGeekblue, -3, -2899 * time.Millisecond},
	}
	for _, k := range cases {
		got := agentTraits(k.seed)
		if got.shape != k.shape || got.color != k.color || got.tilt != k.tilt || got.phase != k.phase {
			t.Errorf("agentTraits(%q) = %+v, want shape %d color %d tilt %v phase %v", k.seed, got, k.shape, k.color, k.tilt, k.phase)
		}
	}
}

func TestAgentShapesHoldTheirEyes(t *testing.T) {
	for s := AgentFace; s <= AgentCat; s++ {
		g := agentGeometry(s)
		if g.body == nil {
			t.Fatalf("shape %d has no body", s)
		}
		for _, x := range []float32{50 - g.eyeGap/2, 50, 50 + g.eyeGap/2} {
			if !g.body.Contains(x, g.eyeY) {
				t.Errorf("shape %d: body leaves out %v,%v", s, x, g.eyeY)
			}
		}
		for _, p := range [][2]float32{{2, 2}, {98, 98}, {2, 98}} {
			if g.body.Contains(p[0], p[1]) {
				t.Errorf("shape %d reaches the corner %v", s, p)
			}
		}
	}
}

type agentApp struct {
	p AgentAvatarProps
}

func agentTester(a *agentApp) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).AlignItems(ui.Start).Children(func() {
			AgentAvatar(c, a.p).Label("agent")
		})
	}, 300, 200)
}

var (
	scleraColor = ui.Hex("#fff8f6")
	pupilColor  = ui.Hex("#080808")
)

func agentAt(t *testing.T, tt *ui.Tester, x, y float32) ui.Color {
	t.Helper()
	info, ok := tt.Access("agent")
	if !ok {
		t.Fatal("no agent avatar")
	}
	b := info.Bounds
	return testkit.PixelAt(tt, 300, b.X+x*b.W/100, b.Y+y*b.H/100)
}

func TestAgentAvatarPaintsBodyAndEyes(t *testing.T) {
	tk := theme.Light()
	tt := agentTester(&agentApp{AgentAvatarProps{Size: 100, Shape: AgentFace, Color: AgentPurple, NoTilt: true}})
	if info, _ := tt.Access("agent"); info.Bounds.W != 100 || info.Bounds.H != 100 {
		t.Fatalf("bounds %+v", info.Bounds)
	}
	if got := agentAt(t, tt, 50, 80); !testkit.Near(got, tk.Purple8) {
		t.Errorf("body %+v, want purple8 %+v", got, tk.Purple8)
	}
	if got := agentAt(t, tt, 33, 30); !testkit.Near(got, scleraColor) {
		t.Errorf("eye white %+v", got)
	}
	if got := agentAt(t, tt, 33, 44); !testkit.Near(got, pupilColor) {
		t.Errorf("pupil %+v", got)
	}
	if got := agentAt(t, tt, 2, 2); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("corner %+v, want the page", got)
	}
}

func TestAgentAvatarDefaultSizeAndCustomColor(t *testing.T) {
	custom := ui.RGB(20, 160, 120)
	tt := agentTester(&agentApp{AgentAvatarProps{Shape: AgentFace, CustomColor: custom, NoTilt: true}})
	if info, _ := tt.Access("agent"); info.Bounds.W != 40 {
		t.Errorf("width %v, want 40", info.Bounds.W)
	}
	if got := agentAt(t, tt, 50, 80); !testkit.Near(got, custom) {
		t.Errorf("body %+v, want the custom color", got)
	}
}

func TestAgentAvatarStates(t *testing.T) {
	tk := theme.Light()
	base := AgentAvatarProps{Size: 100, Shape: AgentFace, Color: AgentPurple, NoTilt: true}

	done := base
	done.State = AgentDone
	tt := agentTester(&agentApp{done})
	if got := agentAt(t, tt, 33, 44); testkit.Near(got, pupilColor) {
		t.Error("a done agent still shows its pupils")
	}

	blocked := base
	blocked.State = AgentBlocked
	tt = agentTester(&agentApp{blocked})
	if got := agentAt(t, tt, 85.5, 9); !testkit.Near(got, tk.Error) {
		t.Errorf("badge %+v, want the error color", got)
	}

	thinking := base
	thinking.State = AgentThinking
	tt = agentTester(&agentApp{thinking})
	if got := agentAt(t, tt, 70, 10); testkit.Near(got, tk.BgLayout) {
		t.Error("no thought bubble")
	}
	tt = agentTester(&agentApp{base})
	if got := agentAt(t, tt, 70, 10); !testkit.Near(got, tk.BgLayout) {
		t.Errorf("an idle agent shows %+v where thinking shows a bubble", got)
	}
}

func frameOf(tt *ui.Tester) []byte {
	img := tt.Image()
	return append([]byte(nil), img.Pix...)
}

func differs(a, b []byte) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}

func TestAgentAvatarTilt(t *testing.T) {
	a := &agentApp{AgentAvatarProps{Size: 100, Seed: "LobeChat", NoTilt: true}}
	tt := agentTester(a)
	still := frameOf(tt)
	a.p.NoTilt = false
	tt.Frame()
	if !differs(still, frameOf(tt)) {
		t.Error("a seeded tilt of -2° draws the same as none")
	}
}

func TestAgentAvatarAnimates(t *testing.T) {
	a := &agentApp{AgentAvatarProps{Size: 100, Shape: AgentFace, Color: AgentPurple, NoTilt: true, State: AgentWorking}}
	tt := agentTester(a)
	first := frameOf(tt)
	tt.Advance(375 * time.Millisecond)
	tt.Frame()
	if differs(first, frameOf(tt)) {
		t.Error("a still agent moved")
	}
	a.p.Animated = true
	tt.Frame()
	start := frameOf(tt)
	moved := false
	for range 6 {
		tt.Advance(120 * time.Millisecond)
		tt.Frame()
		if differs(start, frameOf(tt)) {
			moved = true
			break
		}
	}
	if !moved {
		t.Error("an animated working agent does not bob")
	}
}

func TestAgentAvatarNamesItself(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		AgentAvatar(c, AgentAvatarProps{Name: "Research agent"})
	}, 200, 200)
	if _, ok := tt.AccessByRole(ui.RoleImage, "Research agent"); !ok {
		t.Error("no image named Research agent")
	}
}

func TestAgentAvatarBlockedShakes(t *testing.T) {
	a := &agentApp{AgentAvatarProps{Size: 100, Shape: AgentFace, Color: AgentPurple, NoTilt: true, State: AgentBlocked, Animated: true}}
	tt := agentTester(a)
	seen := map[ui.Color]bool{}
	for range 40 {
		tt.Advance(40 * time.Millisecond)
		tt.Frame()
		seen[agentAt(t, tt, 6.6, 45)] = true
	}
	if len(seen) < 2 {
		t.Error("the body's edge never moves while blocked")
	}
}

func TestAgentAvatarNameSeeds(t *testing.T) {
	render := func(p AgentAvatarProps) []byte {
		p.Size = 100
		return frameOf(agentTester(&agentApp{p}))
	}
	byName := render(AgentAvatarProps{Name: "LobeChat"})
	if differs(byName, render(AgentAvatarProps{Name: "LobeChat", Seed: "LobeChat"})) {
		t.Error("a name seeds differently from the same seed")
	}
	if !differs(byName, render(AgentAvatarProps{Name: "Claude"})) {
		t.Error("two names draw the same agent")
	}
}

func TestAgentAvatarBlockedPupilsShrink(t *testing.T) {
	base := AgentAvatarProps{Size: 100, Shape: AgentFace, Color: AgentPurple, NoTilt: true, State: AgentBlocked}
	tt := agentTester(&agentApp{base})
	if got := agentAt(t, tt, 33, 50); !testkit.Near(got, scleraColor) {
		t.Errorf("blocked pupil reaches %+v at 8 units from its center, want the eye white", got)
	}
	base.State = AgentWaiting
	tt = agentTester(&agentApp{base})
	if got := agentAt(t, tt, 33, 50); !testkit.Near(got, pupilColor) {
		t.Errorf("waiting pupil %+v at 8 units below center, want the pupil", got)
	}
}

func TestAgentAvatarSmallPupils(t *testing.T) {
	share := func(size float32) float32 {
		tt := agentTester(&agentApp{AgentAvatarProps{Size: size, Shape: AgentFace, Color: AgentPurple, NoTilt: true, State: AgentIdle}})
		info, _ := tt.Access("agent")
		b := info.Bounds
		dark := 0
		for y := b.Y; y < b.Y+b.H; y += 0.25 {
			for x := b.X; x < b.X+b.W/2; x += 0.25 {
				if testkit.Near(testkit.PixelAt(tt, 300, x, y), pupilColor) {
					dark++
				}
			}
		}
		return float32(dark) / (size * size)
	}
	if small, normal := share(24), share(25); small > normal*0.85 {
		t.Errorf("pupil share %v at 24 DIPs, %v at 25: want smaller pupils at sidebar sizes", small, normal)
	}
}

func TestAgentAvatarAsksForFramesOnlyWhenAnimated(t *testing.T) {
	a := &agentApp{AgentAvatarProps{Shape: AgentFace, State: AgentWorking}}
	tt := agentTester(a)
	tt.Advance(time.Second)
	tt.Frame()
	if tt.FramePending() {
		t.Error("a still agent asks for frames")
	}
	a.p.Animated = true
	tt.Frame()
	if !tt.FramePending() {
		t.Error("an animated agent asks for no next frame")
	}
}
