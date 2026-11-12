package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func longReply() string {
	var b strings.Builder
	for i := range 12 {
		b.WriteString("## Section\n\nA paragraph of prose with **bold**, `code` and a [link](https://lobehub.com) that goes on for a while to wrap across lines in the reply.\n\n")
		b.WriteString("- First point\n- Second point with more words\n- Third point\n\n")
		if i%3 == 0 {
			b.WriteString("```go\n")
			for range 40 {
				b.WriteString("func handler(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, \"hi %s\", r.URL.Path) }\n")
			}
			b.WriteString("```\n\n")
		}
		if i%4 == 1 {
			b.WriteString("| Name | Type | Default |\n| --- | --- | --- |\n| size | number | 16 |\n| variant | string | default |\n\n")
		}
	}
	return b.String()
}

func streamFrames(b *testing.B, chunk int) {
	reply := longReply()
	for range b.N {
		src := ""
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Column(c).Padding(20).Children(func() {
				Markdown(c, src, MarkdownProps{Animated: true})
			})
		}, 800, 900)
		frames := 0
		start := time.Now()
		for n := 0; n < len(reply); n += chunk {
			src = reply[:min(n+chunk, len(reply))]
			tt.Advance(16 * time.Millisecond)
			tt.Frame()
			frames++
		}
		b.ReportMetric(float64(time.Since(start).Microseconds())/float64(frames)/1000, "ms/frame")
		b.ReportMetric(float64(len(reply)), "bytes")
	}
}

func BenchmarkStreamLongReply(b *testing.B) { streamFrames(b, 30) }
