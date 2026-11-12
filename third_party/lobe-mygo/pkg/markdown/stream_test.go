package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	mmd "github.com/egoist/mygo/ui/markdown"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestRevealFromPresetAndGranularity(t *testing.T) {
	cases := []struct {
		p    MarkdownProps
		want mmd.Reveal
	}{
		{MarkdownProps{}, mmd.Reveal{Unit: mmd.ByCharacter, CatchUp: 120 * time.Millisecond, Fade: 300 * time.Millisecond}},
		{MarkdownProps{Smoothing: Realtime}, mmd.Reveal{Unit: mmd.ByCharacter, CatchUp: 40 * time.Millisecond, Fade: 300 * time.Millisecond}},
		{MarkdownProps{Smoothing: Silky, Granularity: ByWord}, mmd.Reveal{Unit: mmd.ByWord, CatchUp: 170 * time.Millisecond, Fade: 300 * time.Millisecond}},
	}
	for _, k := range cases {
		if got := revealFor(k.p); got != k.want {
			t.Errorf("revealFor(%+v) = %+v, want %+v", k.p, got, k.want)
		}
	}
}

const answer = "Here is a reply that streams in, word by word, until its very last word: done."

type streamApp struct {
	src      string
	p        MarkdownProps
	revealed bool
}

func streamTester(a *streamApp) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			a.revealed = Revealed(Markdown(c, a.src, a.p))
		})
	}, 600, 400)
}

func TestAnimatedRevealsOverTime(t *testing.T) {
	a := &streamApp{src: answer, p: MarkdownProps{Animated: true}}
	tt := streamTester(a)
	if a.revealed {
		t.Fatal("an animated reply is revealed on its first frame")
	}
	a.p.Animated = false
	for range 60 {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText("done.") || !a.revealed {
		t.Errorf("texts %q, revealed %v: want the whole reply in the end", tt.Texts(), a.revealed)
	}
}

func TestNotAnimatedShowsAtOnce(t *testing.T) {
	a := &streamApp{src: answer}
	tt := streamTester(a)
	if !tt.HasText("done.") || !a.revealed {
		t.Errorf("texts %q, revealed %v", tt.Texts(), a.revealed)
	}
}

func TestStreamingGrowsInPlace(t *testing.T) {
	a := &streamApp{p: MarkdownProps{Animated: true}}
	tt := streamTester(a)
	words := strings.Fields(answer)
	for i := range words {
		a.src = strings.Join(words[:i+1], " ")
		tt.Advance(30 * time.Millisecond)
		tt.Frame()
	}
	a.p.Animated = false
	for range 60 {
		tt.Advance(50 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText("done.") || !a.revealed {
		t.Errorf("texts %q, revealed %v", tt.Texts(), a.revealed)
	}
}
