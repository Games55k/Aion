package display

import (
	"time"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type AgentShape int

const (
	// AgentShapeSeeded takes the shape the seed picks.
	AgentShapeSeeded AgentShape = iota
	AgentFace
	AgentBowl
	AgentCloud
	AgentTile
	AgentHex
	AgentShield
	AgentPudding
	AgentCookie
	AgentStar
	AgentCapsule
	AgentGhost
	AgentCat
)

type AgentColor int

const (
	// AgentColorSeeded takes the color the seed picks.
	AgentColorSeeded AgentColor = iota
	AgentPurple
	AgentOrange
	AgentGeekblue
	AgentVolcano
	AgentMagenta
	AgentInk
	AgentGray
)

type AgentState int

const (
	AgentIdle AgentState = iota
	// AgentWorking bobs and looks down at its work.
	AgentWorking
	// AgentWaiting sways, leaning.
	AgentWaiting
	// AgentBlocked shakes, wide-eyed, with an alert badge.
	AgentBlocked
	// AgentThinking looks up with thought bubbles rising.
	AgentThinking
	// AgentDone hops with happy closed eyes.
	AgentDone
)

type AgentAvatarProps struct {
	// Seed picks the shape, color, lean and animation phase, the same for the same seed: Name
	// when empty.
	Seed string
	// Name names the avatar for assistive technology.
	Name string
	// Shape and Color replace what the seed picks; CustomColor replaces the color with one of
	// its own.
	Shape       AgentShape
	Color       AgentColor
	CustomColor ui.Color
	// Size is the width and height: 40 DIPs when zero.
	Size float32
	// State is what the agent is doing, which its eyes and pose show.
	State AgentState
	// Animated moves the agent as its state does: blinking, glancing and bobbing.
	Animated bool
	// NoTilt stands the agent upright rather than at the lean its seed picks.
	NoTilt bool
}

type agentTrait struct {
	shape AgentShape
	color AgentColor
	tilt  float32
	phase time.Duration
}

func hashSeed(seed string) uint32 {
	h := uint32(2166136261)
	for _, u := range utf16.Encode([]rune(seed)) {
		h ^= uint32(u)
		h *= 16777619
	}
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func agentTraits(seed string) agentTrait {
	h := hashSeed("lh3174:" + seed)
	return agentTrait{
		shape: AgentShape(h%12) + AgentFace,
		color: AgentColor((h>>3)%7) + AgentPurple,
		tilt:  float32(int((h>>11)%7) - 3),
		phase: -time.Duration((h>>7)%3000) * time.Millisecond,
	}
}

func agentColor(tk *theme.Tokens, c AgentColor) ui.Color {
	pick := func(light, dark ui.Color) ui.Color {
		if tk.Dark {
			return dark
		}
		return light
	}
	switch c {
	case AgentOrange:
		return tk.Orange9
	case AgentGeekblue:
		return pick(tk.Geekblue8, tk.Geekblue9)
	case AgentVolcano:
		return pick(tk.Volcano8, tk.Volcano9)
	case AgentMagenta:
		return pick(tk.Magenta8, tk.Magenta9)
	case AgentInk:
		return pick(tk.Gray11, tk.Gray7)
	case AgentGray:
		return pick(tk.Gray9, tk.Gray10)
	}
	return pick(tk.Purple8, tk.Purple9)
}

var (
	agentSclera = ui.Hex("#fff8f6")
	agentPupil  = ui.Hex("#080808")
)

var stillGaze = map[AgentState][2]float32{
	AgentIdle: {0, 1.6}, AgentThinking: {1.8, -3.1}, AgentWaiting: {0, 1.4}, AgentWorking: {-2, 3.4},
}

var thoughtDots = [3][3]float32{{0, 8, 8.5}, {12, 2, 6.5}, {20, -5, 4.5}}

// AgentAvatar draws an agent as Lobe draws them: a flat silhouette with the logo's big eyes, its
// shape, color and lean seeded by its name, its eyes and pose showing its State.
func AgentAvatar(c *ui.Context, p AgentAvatarProps) ui.Element {
	tk := theme.TokensOf(c)
	size := p.Size
	if size <= 0 {
		size = 40
	}
	seed := p.Seed
	if seed == "" {
		seed = p.Name
	}
	tr := agentTraits(seed)
	shape, colorKey := p.Shape, p.Color
	if shape == AgentShapeSeeded {
		shape = tr.shape
	}
	if colorKey == AgentColorSeeded {
		colorKey = tr.color
	}
	body := p.CustomColor
	if body == (ui.Color{}) {
		body = agentColor(tk, colorKey)
	}
	lean := tr.tilt
	if p.NoTilt {
		lean = 0
	}
	d := agentDrawing{g: agentGeometry(shape), state: p.State, body: body, bg: tk.BgContainer,
		badge: tk.Error, mark: tk.TextLightSolid, lean: lean, small: size <= 24, animated: p.Animated, phase: tr.phase}
	root := ui.Box(c).Size(size, size).Shrink(0)
	if p.Name != "" {
		root.Role(ui.RoleImage).Label(p.Name)
	}
	return root.Draw(func(pt *ui.Painter, r ui.Rect) {
		if d.animated {
			pt.AnimationFrame()
		}
		d.paint(pt, r, pt.Now())
	})
}
