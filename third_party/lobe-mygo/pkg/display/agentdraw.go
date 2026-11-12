package display

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

type agentDrawing struct {
	g                     *agentShapeGeometry
	state                 AgentState
	body, bg, badge, mark ui.Color
	lean                  float32
	small, animated       bool
	phase                 time.Duration
}

type pose struct{ tx, ty, sx, sy, rot, op float32 }

var still = pose{sx: 1, sy: 1, op: 1}

func move(tx, ty float32) pose      { return pose{tx: tx, ty: ty, sx: 1, sy: 1, op: 1} }
func squash(sx, sy float32) pose    { return pose{sx: sx, sy: sy, op: 1} }
func turn(deg float32) pose         { return pose{sx: 1, sy: 1, rot: deg, op: 1} }
func hop(ty, sx, sy float32) pose   { return pose{ty: ty, sx: sx, sy: sy, op: 1} }
func rise(op, ty, s float32) pose   { return pose{ty: ty, sx: s, sy: s, op: op} }
func lerp(a, b, t float32) float32  { return a + (b-a)*t }
func at(u float32, p pose) keyframe { return keyframe{u, p} }
func cycle(ms int) time.Duration    { return time.Duration(ms) * time.Millisecond }
func blinkAt(u float32) []keyframe  { return blink(u-0.025, u, u+0.025) }
func blink(a, m, b float32) []keyframe {
	return []keyframe{at(a, still), at(m, squash(1, 0.08)), at(b, still)}
}

type keyframe struct {
	u float32
	p pose
}

type track struct {
	dur    time.Duration
	frames []keyframe
	linear bool
}

var agentEase = ui.CubicBezier(0.45, 0, 0.55, 1)

func (tr track) at(t time.Duration) pose {
	u := float32(((t%tr.dur)+tr.dur)%tr.dur) / float32(tr.dur)
	f := tr.frames
	if u <= f[0].u {
		return f[0].p
	}
	for i := 1; i < len(f); i++ {
		if u > f[i].u {
			continue
		}
		k := (u - f[i-1].u) / (f[i].u - f[i-1].u)
		if !tr.linear {
			k = agentEase(k)
		}
		a, b := f[i-1].p, f[i].p
		return pose{lerp(a.tx, b.tx, k), lerp(a.ty, b.ty, k), lerp(a.sx, b.sx, k), lerp(a.sy, b.sy, k), lerp(a.rot, b.rot, k), lerp(a.op, b.op, k)}
	}
	return f[len(f)-1].p
}

func frames(points ...[]keyframe) []keyframe {
	var out []keyframe
	for _, p := range points {
		out = append(out, p...)
	}
	return out
}

var bodyTracks = map[AgentState]track{
	AgentIdle:     {cycle(3000), []keyframe{at(0, still), at(0.5, squash(1.015, 0.982)), at(1, still)}, false},
	AgentWorking:  {cycle(750), []keyframe{at(0, still), at(0.5, move(0, -1.6)), at(1, still)}, false},
	AgentWaiting:  {cycle(3000), []keyframe{at(0, still), at(0.25, turn(-4)), at(0.75, turn(4)), at(1, still)}, false},
	AgentBlocked:  {cycle(1500), []keyframe{at(0, still), at(0.06, move(-1.8, 0)), at(0.12, move(1.8, 0)), at(0.18, move(-1, 0)), at(0.24, still), at(1, still)}, false},
	AgentThinking: {cycle(3000), []keyframe{at(0, turn(-2)), at(0.5, turn(0)), at(1, turn(-2))}, false},
	AgentDone: {cycle(3000), []keyframe{at(0, still), at(0.08, hop(1, 1.05, 0.93)), at(0.18, hop(-5, 0.97, 1.04)),
		at(0.26, hop(0.5, 1.04, 0.95)), at(0.32, still), at(1, still)}, false},
}

var blinkTracks = map[AgentState]track{
	AgentIdle:     {cycle(6000), frames([]keyframe{at(0, still)}, blinkAt(0.325), blinkAt(0.845), []keyframe{at(1, still)}), false},
	AgentWorking:  {cycle(3000), frames([]keyframe{at(0, still)}, blinkAt(0.925), []keyframe{at(1, still)}), false},
	AgentWaiting:  {cycle(3000), frames([]keyframe{at(0, still)}, blinkAt(0.625), []keyframe{at(1, still)}), false},
	AgentThinking: {cycle(3000), frames([]keyframe{at(0, still)}, blinkAt(0.475), []keyframe{at(1, still)}), false},
}

var gazeTracks = map[AgentState]track{
	AgentIdle: {cycle(6000), []keyframe{at(0, move(0, 1.6)), at(0.18, move(-3, 1)), at(0.4, move(-3, 1)),
		at(0.62, move(3, 0.6)), at(0.84, move(3, 0.6)), at(1, move(0, 1.6))}, false},
	AgentWorking:  {cycle(1500), []keyframe{at(0, move(-3.2, 3.4)), at(0.4, move(3.2, 3.4)), at(0.5, move(3.2, 3.4)), at(0.9, move(-3.2, 3.4)), at(1, move(-3.2, 3.4))}, false},
	AgentThinking: {cycle(3000), []keyframe{at(0, move(1.8, -3.1)), at(0.5, move(-1, -3.4)), at(1, move(1.8, -3.1))}, false},
}

var thoughtTrack = track{cycle(1800), []keyframe{at(0, rise(0, 6, 0)), at(0.2, rise(0.95, 3.2, 0.8)),
	at(0.25, rise(0.925, 2.5, 1)), at(0.7, rise(0.7, -3.8, 0.9)), at(1, rise(0, -8, 0))}, true}

var badgeTrack = track{cycle(1500), []keyframe{at(0, still), at(0.08, move(0, -2)), at(0.16, still), at(1, still)}, false}

func (d *agentDrawing) paint(p *ui.Painter, r ui.Rect, now time.Time) {
	t := time.Duration(now.UnixNano()) - d.phase
	view := ui.Scaling(r.W/100, r.H/100).Then(ui.Translation(r.X, r.Y))
	p.Transform(view, 0, 0, func() {
		p.Transform(ui.Rotation(d.lean), 50, 94, func() {
			bp := still
			if d.state == AgentWaiting {
				bp = turn(-3)
			}
			if d.animated {
				bp = bodyTracks[d.state].at(t)
			}
			p.Transform(poseTransform(bp), 50, 94, func() {
				p.FillPath(d.g.body, d.body)
				d.eyes(p, t)
			})
		})
		if d.state == AgentThinking {
			d.thoughts(p, t)
		}
		if d.state == AgentBlocked {
			d.alert(p, t)
		}
	})
}

func poseTransform(ps pose) ui.Transform {
	return ui.Scaling(ps.sx, ps.sy).Then(ui.Rotation(ps.rot)).Then(ui.Translation(ps.tx, ps.ty))
}

func (d *agentDrawing) eyes(p *ui.Painter, t time.Duration) {
	g := d.g
	xs := [2]float32{50 - g.eyeGap/2, 50 + g.eyeGap/2}
	unit := g.eyeR / 14.5
	if d.state == AgentDone {
		open := d.animated && float32(t%cycle(3000))/float32(cycle(3000)) >= 0.75
		if !open {
			width := float32(6.5)
			if d.small {
				width = 7.5
			}
			for _, x := range xs {
				p.StrokePath(happyEyePath(x, g.eyeY, g.eyeR), width*unit, agentSclera)
			}
			return
		}
	}
	radius, ratio := g.eyeR, float32(0.6)
	switch {
	case d.state == AgentBlocked:
		radius, ratio = g.eyeR*1.1, 0.36
	case d.small:
		ratio = 0.5
	}
	gaze := move(stillGaze[d.state][0], stillGaze[d.state][1])
	if tr, ok := gazeTracks[d.state]; ok && d.animated {
		gaze = tr.at(t)
	}
	lid := still
	if tr, ok := blinkTracks[d.state]; ok && d.animated {
		lid = tr.at(t)
	}
	p.Transform(ui.Scaling(1, lid.sy), 50, g.eyeY, func() {
		for _, x := range xs {
			sclera := scleraPath(x, g.eyeY, radius)
			p.FillPath(sclera, agentSclera)
			p.ClipPath(sclera, func() {
				p.FillPath(new(ui.Path).Circle(x+gaze.tx*unit, g.eyeY+gaze.ty*unit, g.eyeR*ratio), agentPupil)
			})
		}
	})
}

func (d *agentDrawing) thoughts(p *ui.Painter, t time.Duration) {
	g := d.g
	dir := float32(1)
	if g.thoughtFlip {
		dir = -1
	}
	for i, dot := range thoughtDots {
		cx, cy, rad := g.thoughtX+dot[0]*dir, g.thoughtY+dot[1], dot[2]
		opacity := 0.9 - float32(i)*0.15
		ps := still
		if d.animated {
			ps = thoughtTrack.at(t - time.Duration(i)*350*time.Millisecond)
		}
		if ps.sx <= 0 || ps.op <= 0 {
			continue
		}
		p.Transform(poseTransform(ps), cx, cy, func() {
			circle := new(ui.Path).Circle(cx, cy, rad)
			if !g.noHalo {
				p.StrokePath(circle, 2.5, d.bg.Alpha(opacity*ps.op))
			}
			p.FillPath(circle, d.body.Alpha(opacity*ps.op))
		})
	}
}

func (d *agentDrawing) alert(p *ui.Painter, t time.Duration) {
	g := d.g
	ps := still
	if d.animated {
		ps = badgeTrack.at(t)
	}
	x, y := g.badgeX, g.badgeY+ps.ty
	p.FillPath(new(ui.Path).Circle(x, y, 11.5), d.bg)
	p.FillPath(new(ui.Path).Circle(x, y, 9), d.badge)
	p.FillPath(ui.MustParsePath(fmt.Sprintf("M%.2f %.2fh2.5v6.8h-2.5Z", x-1.25, y-6.2)), d.mark)
	p.FillPath(new(ui.Path).Circle(x, y+4.4, 1.45), d.mark)
}
