package action

import (
	"github.com/lobehub/lobe-mygo/pkg/internal/testkit"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

var periods = []string{"Day", "Week", "Month"}

func segment(t *testing.T, tt *ui.Tester, name string) ui.Rect {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleRadio, name)
	if !ok {
		t.Fatalf("no segment named %s; texts %q", name, tt.Texts())
	}
	return info.Bounds
}

func TestSegmentedItemHeights(t *testing.T) {
	for _, c := range []struct {
		size theme.Size
		want float32
	}{{theme.Small, 26}, {theme.Middle, 32}, {theme.Large, 36}} {
		selected := 0
		tt := ui.NewTester(testkit.Centered(func(ctx *ui.Context) {
			Segmented(ctx, &selected, periods, SegmentedProps{Size: c.size})
		}), 400, 120)
		if got := segment(t, tt, "Week").H; got != c.want {
			t.Errorf("size %v item height = %v, want %v", c.size, got, c.want)
		}
	}
}

func TestSegmentedIndicatorFollowsSelection(t *testing.T) {
	tk := theme.Light()
	selected := 0
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Segmented(c, &selected, periods, SegmentedProps{})
	}), 400, 120)
	week := segment(t, tt, "Week")
	day := segment(t, tt, "Day")
	edge := func(r ui.Rect) ui.Color { return testkit.PixelAt(tt, 400, r.X+3, r.Y+r.H/2) }
	if got := edge(day); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("selected Day = %+v, want BgElevated indicator %+v", got, tk.BgElevated)
	}
	if got := edge(week); testkit.Near(got, tk.BgElevated) {
		t.Errorf("unselected Week already shows the indicator")
	}
	if err := tt.Click("Week"); err != nil {
		t.Fatal(err)
	}
	tt.Move(0, 0)
	tt.Advance(time.Second)
	if selected != 1 {
		t.Fatalf("selected = %d after clicking Week", selected)
	}
	if got := edge(week); !testkit.Near(got, tk.BgElevated) {
		t.Errorf("selected Week = %+v, want BgElevated indicator", got)
	}
	if got := edge(day); testkit.Near(got, tk.BgElevated) {
		t.Errorf("Day still shows the indicator after Week was chosen")
	}
}

func TestSegmentedBlockStretches(t *testing.T) {
	selected := 0
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Segmented(c, &selected, periods, SegmentedProps{Block: true})
	}), 400, 120)
	day, month := segment(t, tt, "Day"), segment(t, tt, "Month")
	if day.W != month.W {
		t.Errorf("block items differ in width: Day %v, Month %v", day.W, month.W)
	}
}

func TestSegmentedIndicatorSlides(t *testing.T) {
	selected := 0
	tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
		Segmented(c, &selected, periods, SegmentedProps{})
	}), 400, 120)
	tt.Advance(time.Second)
	day, week := segment(t, tt, "Day"), segment(t, tt, "Week")
	gap := testkit.PixelAt(tt, 400, day.X+day.W+2, day.Y+day.H/2)
	tt.ClickAt(week.X+week.W/2, week.Y+week.H/2)
	tt.Move(0, 0)
	tt.Advance(100 * time.Millisecond)
	mid := testkit.PixelAt(tt, 400, day.X+day.W+2, day.Y+day.H/2)
	if mid == gap {
		t.Error("the gap between Day and Week never shows the indicator: it jumped instead of sliding")
	}
}

func TestSegmentedSelectedLabelShowsOverIndicator(t *testing.T) {
	for _, v := range []SegmentedVariant{SegmentedFilled, SegmentedOutlined} {
		selected := 0
		tt := ui.NewTester(testkit.Centered(func(c *ui.Context) {
			Segmented(c, &selected, periods, SegmentedProps{Variant: v})
		}), 400, 120)
		day := segment(t, tt, "Day")
		darkest := 255
		for x := day.X; x < day.X+day.W; x++ {
			px := testkit.PixelAt(tt, 400, x, day.Y+day.H/2)
			darkest = min(darkest, int(px.R))
		}
		if darkest > 120 {
			t.Errorf("variant %v: the selected label is hidden (darkest pixel %d)", v, darkest)
		}
	}
}
