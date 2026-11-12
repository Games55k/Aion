package display

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The silhouettes and eyes of AgentAvatar on a 100 × 100 canvas, as lobe-ui's AgentAvatar/shapes.ts
// draws them: each fits the 6–94 box, a flat body under the logo's big eyes.

type agentShapeGeometry struct {
	body                *ui.Path
	eyeGap, eyeR, eyeY  float32
	badgeX, badgeY      float32
	thoughtX, thoughtY  float32
	thoughtFlip, noHalo bool
}

type point struct{ x, y float64 }

func polygonPath(pts []point) string {
	var b strings.Builder
	for i, p := range pts {
		if i == 0 {
			b.WriteString("M")
		} else {
			b.WriteString("L")
		}
		fmt.Fprintf(&b, "%.2f %.2f", p.x, p.y)
	}
	b.WriteString("Z")
	return b.String()
}

func polarPath(radius func(a float64) float64, steps int) string {
	pts := make([]point, steps)
	for i := range pts {
		a := float64(i) / float64(steps) * 2 * math.Pi
		r := radius(a)
		pts[i] = point{50 + r*math.Cos(a), 50 + r*math.Sin(a)}
	}
	return polygonPath(pts)
}

func signPow(v, e float64) float64 {
	if v < 0 {
		return -math.Pow(-v, e)
	}
	return math.Pow(v, e)
}

func superellipsePath(cx, cy, a, b, n float64) string {
	pts := make([]point, 120)
	for i := range pts {
		t := float64(i) / 120 * 2 * math.Pi
		pts[i] = point{cx + a*signPow(math.Cos(t), 2/n), cy + b*signPow(math.Sin(t), 2/n)}
	}
	return polygonPath(pts)
}

func facePath() string {
	const r, cx, cy, cut, fillet, inset = 44.0, 50.0, 40.5, 15.5, 0.24, 9.0
	a0 := math.Asin((cy - cut) / r)
	at := func(a float64) point { return point{cx + r*math.Cos(a), cy - r*math.Sin(a)} }
	right, left := at(a0-fillet), at(math.Pi-a0+fillet)
	hx := math.Sqrt(r*r - (cy-cut)*(cy-cut))
	return fmt.Sprintf("M%.2f %.2fQ%.2f %.2f %.2f %.2fQ%.2f %.2f %.2f %.2fA%.0f %.0f 0 1 1 %.2f %.2fQ%.2f %.2f %.2f %.2fZ",
		cx-hx+inset, cut, cx, cut-1.5, cx+hx-inset, cut, cx+hx, cut, right.x, right.y, r, r, left.x, left.y, cx-hx, cut, cx-hx+inset, cut)
}

func bowlPath() string {
	const n = 2.6
	var b strings.Builder
	b.WriteString("M6 30Q6 14 22 14H78Q94 14 94 30L91 52")
	for i := 1; i <= 40; i++ {
		a := float64(i) / 40 * math.Pi
		fmt.Fprintf(&b, "L%.2f %.2f", 50+41*signPow(math.Cos(a), 2/n), 52+42*math.Pow(math.Sin(a), 2/n))
	}
	b.WriteString("L6 30Z")
	return b.String()
}

func cloudPath() string {
	circles := [][3]float64{{34, 54, 30}, {66, 54, 30}, {50, 34, 27}}
	pts := make([]point, 180)
	for i := range pts {
		a := float64(i)/180*2*math.Pi - math.Pi/2
		dx, dy := math.Cos(a), math.Sin(a)
		far := 0.0
		for _, c := range circles {
			fx, fy := 50-c[0], 50-c[1]
			b := 2 * (fx*dx + fy*dy)
			disc := b*b - 4*(fx*fx+fy*fy-c[2]*c[2])
			if disc >= 0 {
				far = math.Max(far, (-b+math.Sqrt(disc))/2)
			}
		}
		pts[i] = point{50 + far*dx, 50 + far*dy}
	}
	return polygonPath(pts)
}

func hexPath() string {
	var v [6]point
	for i := range v {
		a := math.Pi/6 + float64(i)*math.Pi/3
		v[i] = point{50 + 44*math.Cos(a), 50 + 44*math.Sin(a)}
	}
	lerp := func(a, b point, t float64) point { return point{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t} }
	var b strings.Builder
	for i, corner := range v {
		start, end := lerp(corner, v[(i+5)%6], 0.22), lerp(corner, v[(i+1)%6], 0.22)
		op := "L"
		if i == 0 {
			op = "M"
		}
		fmt.Fprintf(&b, "%s%.2f %.2fQ%.2f %.2f %.2f %.2f", op, start.x, start.y, corner.x, corner.y, end.x, end.y)
	}
	b.WriteString("Z")
	return b.String()
}

func ghostPath() string {
	var b strings.Builder
	b.WriteString("M14 42A36 36 0 0 1 86 42V84")
	for i := range 3 {
		fmt.Fprintf(&b, "A12 9 0 0 1 %d 84", 86-(i+1)*24)
	}
	b.WriteString("Z")
	return b.String()
}

type shapeSpec struct {
	d                                   func() string
	gap, r, y, badgeX, badgeY, thX, thY float32
	flip, noHalo                        bool
}

func fixed(d string) func() string { return func() string { return d } }

var shapeSpecs = map[AgentShape]shapeSpec{
	AgentFace:  {d: facePath, gap: 34, r: 16, y: 42, badgeX: 91, badgeY: 9, thX: 70, thY: 2},
	AgentBowl:  {d: bowlPath, gap: 34, r: 16, y: 46, badgeX: 91, badgeY: 9, thX: 70, thY: 2},
	AgentCloud: {d: cloudPath, gap: 34, r: 16, y: 52, badgeX: 88, badgeY: 14, thX: 70, thY: 2},
	AgentTile: {d: func() string { return superellipsePath(50, 50, 42, 42, 5) },
		gap: 33, r: 15.5, y: 46, badgeX: 91, badgeY: 9, thX: 70, thY: 2},
	AgentHex: {d: hexPath, gap: 27, r: 12.5, y: 47, badgeX: 89, badgeY: 13, thX: 70, thY: 2},
	AgentShield: {d: fixed("M18 18Q18 8 28 8H72Q82 8 82 18L88 48Q90 72 50 94Q10 72 12 48Z"),
		gap: 26, r: 12, y: 42, badgeX: 91, badgeY: 9, thX: 70, thY: 2},
	AgentPudding: {d: fixed("M30 14H70Q79 14 81 23Q87 50 93 80Q95 92 83 92H17Q5 92 7 80Q13 50 19 23Q21 14 30 14Z"),
		gap: 29, r: 13.5, y: 52, badgeX: 91, badgeY: 9, thX: 70, thY: 2},
	AgentCookie: {d: func() string {
		return polarPath(func(a float64) float64 { return 39.5 + 4.5*math.Pow(math.Abs(math.Cos(5*a)), 0.55) }, 400)
	}, gap: 31, r: 14.5, y: 48, badgeX: 89, badgeY: 9, thX: 70, thY: 2},
	AgentStar: {d: func() string {
		return polarPath(func(a float64) float64 { return 45 * (0.76 + 0.24*math.Pow(math.Abs(math.Cos(2*a)), 2.2)) }, 240)
	}, gap: 27, r: 12.5, y: 49, badgeX: 88, badgeY: 14, thX: 70, thY: 2},
	AgentCapsule: {d: fixed("M36 20H64A30 30 0 0 1 64 80H36A30 30 0 0 1 36 20Z"),
		gap: 34, r: 16, y: 49, badgeX: 88, badgeY: 14, thX: 70, thY: 2},
	AgentGhost: {d: ghostPath, gap: 29, r: 13.5, y: 46, badgeX: 91, badgeY: 9, thX: 70, thY: 2},
	AgentCat: {d: fixed("M12 57A38 38 0 1 1 88 57A38 38 0 1 1 12 57ZM19 40L17 12Q17 6 22.5 8.5L44 21ZM56 21L77.5 8.5Q83 6 83 12L81 40Z"),
		gap: 30, r: 14, y: 56, badgeX: 91, badgeY: 91, thX: 48, thY: 0, flip: true, noHalo: true},
}

var agentGeometries sync.Map

func agentGeometry(s AgentShape) *agentShapeGeometry {
	if g, ok := agentGeometries.Load(s); ok {
		return g.(*agentShapeGeometry)
	}
	spec := shapeSpecs[s]
	body := ui.MustParsePath(spec.d())
	g := &agentShapeGeometry{body: body, eyeGap: spec.gap, eyeR: spec.r, eyeY: spec.y,
		badgeX: spec.badgeX, badgeY: spec.badgeY, thoughtX: spec.thX, thoughtY: spec.thY,
		thoughtFlip: spec.flip, noHalo: spec.noHalo}
	agentGeometries.Store(s, g)
	return g
}

const agentCheek = 0.74

func scleraPath(x, y, r float32) *ui.Path {
	cut := y + r*agentCheek
	half := float32(math.Sqrt(float64(r*r - (r*agentCheek)*(r*agentCheek))))
	return ui.MustParsePath(fmt.Sprintf("M%.2f %.2fA%.2f %.2f 0 1 1 %.2f %.2fQ%.2f %.2f %.2f %.2fZ",
		x-half, cut, r, r, x+half, cut, x, cut-r*0.2, x-half, cut))
}

func happyEyePath(x, y, r float32) *ui.Path {
	k := r / 14.5
	return ui.MustParsePath(fmt.Sprintf("M%.2f %.2fQ%.2f %.2f %.2f %.2f", x-9.5*k, y+4*k, x, y-10*k, x+9.5*k, y+4*k))
}
