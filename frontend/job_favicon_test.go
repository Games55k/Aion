package frontend

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"aion/backend/jobapplications"
	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

func makeTestPNG(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeTestICOFromPNG(pngData []byte, w, h byte) []byte {
	out := make([]byte, 6+16+len(pngData))
	binary.LittleEndian.PutUint16(out[0:2], 0)
	binary.LittleEndian.PutUint16(out[2:4], 1)
	binary.LittleEndian.PutUint16(out[4:6], 1)
	out[6] = w
	out[7] = h
	binary.LittleEndian.PutUint16(out[12:14], 32)
	binary.LittleEndian.PutUint32(out[14:18], uint32(len(pngData)))
	binary.LittleEndian.PutUint32(out[18:22], 22)
	copy(out[22:], pngData)
	return out
}

func makeTestDIBICO(w, h int, bgra color.NRGBA) []byte {
	dibHeaderSize := 40
	xorSize := w * h * 4
	maskStride := ((w + 31) / 32) * 4
	maskSize := maskStride * h
	payloadSize := dibHeaderSize + xorSize + maskSize

	out := make([]byte, 22+payloadSize)
	binary.LittleEndian.PutUint16(out[2:4], 1)
	binary.LittleEndian.PutUint16(out[4:6], 1)
	out[6] = byte(w)
	out[7] = byte(h)
	binary.LittleEndian.PutUint16(out[12:14], 32)
	binary.LittleEndian.PutUint32(out[14:18], uint32(payloadSize))
	binary.LittleEndian.PutUint32(out[18:22], 22)

	dib := out[22:]
	binary.LittleEndian.PutUint32(dib[0:4], uint32(dibHeaderSize))
	binary.LittleEndian.PutUint32(dib[4:8], uint32(w))
	binary.LittleEndian.PutUint32(dib[8:12], uint32(h*2))
	binary.LittleEndian.PutUint16(dib[12:14], 1)
	binary.LittleEndian.PutUint16(dib[14:16], 32)

	px := dib[dibHeaderSize:]
	for i := 0; i < w*h; i++ {
		px[i*4] = bgra.B
		px[i*4+1] = bgra.G
		px[i*4+2] = bgra.R
		px[i*4+3] = bgra.A
	}
	return out
}

// Company initial extraction uppercases ASCII letters, preserves CJK characters, trims spaces, and falls back to "?".
func TestCompanyInitialExtraction(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"星辰实验室", "星"},
		{"  raycast", "R"},
		{"Linear", "L"},
		{"   ", "?"},
		{"", "?"},
	}
	for _, tc := range cases {
		if got := companyInitial(tc.in); got != tc.want {
			t.Fatalf("companyInitial(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Favicon decoder handles PNG, PNG-embedded ICO, and 32-bit DIB ICO payloads while capping dimensions and rejecting corrupt bytes.
func TestFaviconDecodingAndHTMLDiscovery(t *testing.T) {
	pngBytes := makeTestPNG(t, 128, 128, color.RGBA{R: 0, G: 113, B: 227, A: 255})
	img, err := decodeFaviconBytes(pngBytes)
	if err != nil {
		t.Fatalf("decode PNG favicon: %v", err)
	}
	if img.Bounds().Dx() != maxFaviconSide || img.Bounds().Dy() != maxFaviconSide {
		t.Fatalf("expected scaled %dx%d favicon, got %v", maxFaviconSide, maxFaviconSide, img.Bounds())
	}

	if _, err := decodeFaviconBytes(makeTestICOFromPNG(pngBytes, 128, 128)); err != nil {
		t.Fatalf("decode PNG-in-ICO: %v", err)
	}
	if _, err := decodeFaviconBytes(makeTestDIBICO(16, 16, color.NRGBA{R: 24, G: 24, B: 24, A: 255})); err != nil {
		t.Fatalf("decode DIB-in-ICO: %v", err)
	}
	if _, err := decodeFaviconBytes([]byte("not-an-image")); err == nil {
		t.Fatal("expected invalid bytes to fail decoding")
	}

	// HTML <link rel="icon"> discovery fetches icon when root /favicon.ico returns 404.
	iconBytes := makeTestPNG(t, 32, 32, color.RGBA{R: 10, G: 132, B: 255, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jobs/1":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><link rel="shortcut icon" href="/static/brand.png"></head></html>`))
		case "/static/brand.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(iconBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	fetched, err := fetchFaviconImage(context.Background(), srv.Client(), srv.URL+"/jobs/1", false)
	if err != nil || fetched.Bounds().Dx() != 32 {
		t.Fatalf("expected HTML-discovered 32px icon, got %v err=%v", fetched, err)
	}

	// Regression: Feishu Hire and similar SPA job portals inject their tab favicon via embedded JSON
	// ("browser_label_img_url") rather than static <link rel="icon"> tags and return 404 for /favicon.ico.
	var spaSrv *httptest.Server
	spaSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/referral/position/123/detail":
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<html><head><script id="js-websiteInfo" type="text/json">{"website_info":{"web_ui_config":{"browser_label_img_url":"%s/cdn/sun-icon.png"}}}</script></head></html>`, spaSrv.URL)
		case "/cdn/sun-icon.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(iconBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer spaSrv.Close()

	spaFetched, err := fetchFaviconImage(context.Background(), spaSrv.Client(), spaSrv.URL+"/referral/position/123/detail", false)
	if err != nil || spaFetched.Bounds().Dx() != 32 {
		t.Fatalf("expected SPA JSON-discovered 32px icon, got %v err=%v", spaFetched, err)
	}
}

// Job applications show website favicons when a URL loads and company-initial monograms when URL is blank, loading, or failed, across both Focus and Jobs views with bounded caching.
func TestJobAvatarFaviconAndMonogramInFocusAndJobsViews(t *testing.T) {
	ctx := context.Background()
	fixedNow := func() time.Time { return time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local) }
	dispatched := make(chan func(), 16)
	gate := make(chan struct{})
	var callsMu sync.Mutex
	loadCalls := make(map[string]int)
	sampleBmp := ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 32, 32)))

	app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{
		Now: fixedNow,
		Dispatch: func(fn func()) {
			dispatched <- fn
		},
		LoadFavicon: func(_ context.Context, rawURL string) (*ui.Bitmap, error) {
			callsMu.Lock()
			loadCalls[rawURL]++
			callsMu.Unlock()
			<-gate
			if rawURL == "https://broken.example.com/jobs" {
				return nil, errors.New("404")
			}
			return sampleBmp, nil
		},
	})

	if _, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "Raycast",
		PositionName:   "Mac Engineer",
		AppliedDate:    "2026-11-26",
		Stage:          model.JobStageInterview1,
		ApplicationURL: "https://raycast.com/careers",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:  "星辰实验室",
		PositionName: "系统工程师",
		AppliedDate:  "2026-11-25",
		Stage:        model.JobStageOffer,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.JobApplications.Create(ctx, jobapplications.CreateInput{
		CompanyName:    "BrokenCorp",
		PositionName:   "Backend",
		AppliedDate:    "2026-11-24",
		Stage:          model.JobStageApplied,
		ApplicationURL: "https://broken.example.com/jobs",
	}); err != nil {
		t.Fatal(err)
	}

	wb.ReloadAll()
	drainDispatched := func() {
		wb.WaitForTasks()
		for len(dispatched) > 0 {
			(<-dispatched)()
		}
	}

	// 1. Focus ("今天") view: while favicon fetch is pending, all jobs show initial monograms without blocking.
	wb.NavigateTo(NavFocus)
	tt.Frame()
	if _, ok := tt.Find("首字图标 Raycast"); !ok {
		t.Fatal("expected initial monogram fallback while Raycast favicon is still loading")
	}
	if _, ok := tt.Find("首字图标 星辰实验室"); !ok {
		t.Fatal("expected initial monogram for job without URL in Focus view")
	}

	// Release background fetches and apply results on UI thread.
	close(gate)
	drainDispatched()
	tt.Frame()

	if _, ok := tt.Find("网站图标 Raycast"); !ok {
		t.Fatal("expected website favicon rendered for Raycast in Focus view")
	}
	if _, ok := tt.Find("首字图标 星辰实验室"); !ok {
		t.Fatal("expected monogram tile retained for 星辰实验室 without URL in Focus view")
	}
	if _, ok := tt.Find("首字图标 BrokenCorp"); !ok {
		t.Fatal("expected monogram fallback when BrokenCorp favicon fails in Focus view")
	}

	// 2. Jobs ("求职") view reuses cached favicon & failure state without triggering duplicate fetches.
	wb.NavigateTo(NavJobAll)
	drainDispatched()
	tt.Frame()
	if _, ok := tt.Find("网站图标 Raycast"); !ok {
		t.Fatal("expected website favicon rendered for Raycast in Jobs table")
	}
	if _, ok := tt.Find("首字图标 星辰实验室"); !ok {
		t.Fatal("expected monogram tile for 星辰实验室 in Jobs table")
	}
	if _, ok := tt.Find("首字图标 BrokenCorp"); !ok {
		t.Fatal("expected monogram fallback for BrokenCorp in Jobs table")
	}
	if loadCalls["https://raycast.com/careers"] != 1 || loadCalls["https://broken.example.com/jobs"] != 1 {
		t.Fatalf("expected deduplicated single fetch per host, got %+v", loadCalls)
	}

	// 3. Favicon memory cache enforces bounded LRU limit.
	for i := 0; i < maxFaviconCacheEntries+10; i++ {
		wb.cacheFavicon(fmt.Sprintf("host-%d.example.com", i), sampleBmp)
	}
	if len(wb.faviconCache) > maxFaviconCacheEntries {
		t.Fatalf("favicon cache exceeded cap %d: got %d", maxFaviconCacheEntries, len(wb.faviconCache))
	}
}
