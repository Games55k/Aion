package frontend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	maxFaviconCacheEntries = 64
	maxFaviconInFlight     = 4
	maxFaviconFileBytes    = 512 << 10
	maxFaviconHTMLBytes    = 128 << 10
	maxFaviconSide         = 64
	maxFaviconPixels       = 4 << 20
	faviconFetchTimeout    = 4 * time.Second
)

var (
	errInvalidFavicon = errors.New("invalid or unsupported favicon")
	pngMagicHeader    = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	linkTagRe         = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	metaTagRe         = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	htmlAttrRe        = regexp.MustCompile(`(?is)([a-zA-Z0-9_-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)
	// Matches SPA/SSR embedded favicon JSON fields such as Feishu Hire's "browser_label_img_url",
	// or common "favicon_url" / "faviconUrl" / "siteIcon" / "web_icon" keys.
	embeddedIconJSONRe = regexp.MustCompile(`(?i)"(?:browser_label_img_url|favicon_url|faviconUrl|favicon|site_icon|siteIcon|web_icon)"\s*:\s*"([^"]+)"`)
)

// companyInitial returns the uppercased first visible character of a company name
// for use as a minimalist monogram icon when no website favicon is available.
func companyInitial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsSpace(r) {
			continue
		}
		if r >= 'a' && r <= 'z' {
			r = unicode.ToUpper(r)
		}
		return string(r)
	}
	return "?"
}

// faviconHostKey validates an http(s) URL and extracts its origin and normalized host key.
func faviconHostKey(rawURL string) (origin, hostKey string, ok bool) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", "", false
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", "", false
	}
	hostname := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if hostname == "" {
		return "", "", false
	}
	hostKey = strings.ToLower(u.Host)
	origin = scheme + "://" + u.Host
	return origin, hostKey, true
}

// FaviconBitmap returns a cached website favicon for rawURL, scheduling a background
// fetch through Host.LoadFavicon when needed. Returns nil while loading, on error,
// or when Host.LoadFavicon is unset (e.g., offline headless tests).
func (w *Workbench) FaviconBitmap(rawURL string) *ui.Bitmap {
	if w.IsRestoring() {
		return nil
	}
	_, hostKey, ok := faviconHostKey(rawURL)
	if !ok {
		return nil
	}
	if bmp := w.faviconCache[hostKey]; bmp != nil {
		w.touchFavicon(hostKey)
		return bmp
	}
	if w.faviconFailed[hostKey] || w.faviconLoading[hostKey] {
		return nil
	}
	loader := w.host.LoadFavicon
	if loader == nil {
		return nil
	}
	if len(w.faviconLoading) >= maxFaviconInFlight {
		return nil
	}

	epoch := w.currentDataEpoch()
	apply := func(bmp *ui.Bitmap, err error) {
		delete(w.faviconLoading, hostKey)
		if w.IsRestoring() || epoch != w.currentDataEpoch() {
			return
		}
		if err != nil || bmp == nil {
			w.faviconFailed[hostKey] = true
			return
		}
		w.cacheFavicon(hostKey, bmp)
	}

	w.faviconLoading[hostKey] = true
	if dispatch := w.host.Dispatch; dispatch != nil {
		w.taskWG.Add(1)
		go func() {
			defer w.taskWG.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			bmp, err := loader(ctx, rawURL)
			dispatch(func() { apply(bmp, err) })
		}()
		return nil
	}
	bmp, err := loader(context.Background(), rawURL)
	apply(bmp, err)
	return w.faviconCache[hostKey]
}

func (w *Workbench) touchFavicon(hostKey string) {
	w.faviconOrder = slices.DeleteFunc(w.faviconOrder, func(existing string) bool { return existing == hostKey })
	w.faviconOrder = append(w.faviconOrder, hostKey)
}

func (w *Workbench) cacheFavicon(hostKey string, bmp *ui.Bitmap) {
	if bmp == nil {
		return
	}
	if _, exists := w.faviconCache[hostKey]; exists {
		delete(w.faviconCache, hostKey)
		w.faviconOrder = slices.DeleteFunc(w.faviconOrder, func(existing string) bool { return existing == hostKey })
	}
	for len(w.faviconCache) >= maxFaviconCacheEntries {
		if len(w.faviconOrder) == 0 {
			clear(w.faviconCache)
			break
		}
		oldest := w.faviconOrder[0]
		w.faviconOrder = w.faviconOrder[1:]
		delete(w.faviconCache, oldest)
	}
	w.faviconCache[hostKey] = bmp
	w.faviconOrder = append(w.faviconOrder, hostKey)
}

// renderJobAvatar renders a website favicon when the application has a URL and its icon
// is available, or falls back to a clean company-initial monogram tile.
func (w *Workbench) renderJobAvatar(c *ui.Context, tk *uiTokens, app model.JobApplication, size float32) {
	var bmp *ui.Bitmap
	if strings.TrimSpace(app.ApplicationURL) != "" {
		bmp = w.FaviconBitmap(app.ApplicationURL)
	}
	box := ui.Box(c.Key("job-avatar-"+app.ID)).Size(size, size).Radius(8).Clip().
		Center().Shrink(0).Background(tk.FillTertiary)
	if bmp != nil {
		imgSide := max(size-8, 16)
		box.Label("网站图标 " + app.CompanyName)
		box.Children(func() {
			ui.Image(c, bmp).Size(imgSide, imgSide).Fit(ui.Contain)
		})
		return
	}
	fontSize := float32(13)
	if size >= 32 {
		fontSize = 14
	}
	box.Label("首字图标 " + app.CompanyName)
	box.Children(func() {
		ui.Text(c, companyInitial(app.CompanyName)).FontSize(fontSize).FontWeight(600).TextColor(tk.Text).SingleLine()
	})
}

// DefaultFaviconLoader resolves, decodes, and caches a website favicon for rawURL.
// It checks an OS user cache directory first to avoid redundant network calls across restarts.
func DefaultFaviconLoader(ctx context.Context, rawURL string) (*ui.Bitmap, error) {
	_, hostKey, ok := faviconHostKey(rawURL)
	if !ok {
		return nil, errInvalidFavicon
	}
	cacheFile := faviconDiskCachePath(hostKey)
	if cacheFile != "" {
		if raw, err := os.ReadFile(cacheFile); err == nil {
			if img, err := decodeFaviconBytes(raw); err == nil {
				return ui.NewBitmap(img), nil
			}
		}
	}

	client := &http.Client{Timeout: faviconFetchTimeout}
	img, err := fetchFaviconImage(ctx, client, rawURL, true)
	if err != nil {
		return nil, err
	}
	if cacheFile != "" {
		saveFaviconDiskCache(cacheFile, img)
	}
	return ui.NewBitmap(img), nil
}

func faviconDiskCachePath(hostKey string) string {
	dir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(hostKey))
	return filepath.Join(dir, "Aion", "favicons", hex.EncodeToString(sum[:16])+".png")
}

func saveFaviconDiskCache(path string, img image.Image) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func fetchFaviconImage(ctx context.Context, client *http.Client, rawURL string, allowExternalFallback bool) (image.Image, error) {
	origin, _, ok := faviconHostKey(rawURL)
	if !ok {
		return nil, errInvalidFavicon
	}

	// 1. Try standard high-resolution & default site endpoints directly.
	candidates := []string{
		origin + "/apple-touch-icon.png",
		origin + "/favicon.ico",
		origin + "/favicon.png",
	}
	for _, target := range candidates {
		if img, err := fetchAndDecodeIconURL(ctx, client, target); err == nil {
			return img, nil
		}
	}

	// 2. Discover <link rel="...icon..." href="..."> from the page or site root HTML.
	htmlTargets := []string{rawURL}
	if strings.TrimRight(rawURL, "/") != origin {
		htmlTargets = append(htmlTargets, origin+"/")
	}
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		seen[c] = true
	}
	for _, pageURL := range htmlTargets {
		iconURLs := discoverHTMLIconURLs(ctx, client, pageURL)
		for _, target := range iconURLs {
			if seen[target] {
				continue
			}
			seen[target] = true
			if img, err := fetchAndDecodeIconURL(ctx, client, target); err == nil {
				return img, nil
			}
		}
	}

	// 3. Optional public favicon service fallback for anti-bot/CDN-gated hosts.
	if allowExternalFallback {
		if u, err := url.Parse(rawURL); err == nil {
			host := strings.ToLower(strings.TrimSpace(u.Hostname()))
			if host != "" && host != "localhost" && net.ParseIP(host) == nil && strings.Contains(host, ".") {
				s2 := "https://www.google.com/s2/favicons?domain=" + url.QueryEscape(host) + "&sz=64"
				if img, err := fetchAndDecodeIconURL(ctx, client, s2); err == nil {
					return img, nil
				}
			}
		}
	}

	return nil, errInvalidFavicon
}

func fetchAndDecodeIconURL(ctx context.Context, client *http.Client, targetURL string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/png,image/jpeg,image/gif,image/x-icon,image/*,*/*;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errInvalidFavicon
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconFileBytes+1))
	if err != nil || len(raw) == 0 || len(raw) > maxFaviconFileBytes {
		return nil, errInvalidFavicon
	}
	return decodeFaviconBytes(raw)
}

func discoverHTMLIconURLs(ctx context.Context, client *http.Client, pageURL string) []string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconHTMLBytes))
	if err != nil || len(body) == 0 {
		return nil
	}
	return extractHTMLIconLinks(resp.Request.URL, string(body))
}

func extractHTMLIconLinks(base *url.URL, htmlText string) []string {
	if base == nil {
		return nil
	}
	var highPrio, normalPrio, embeddedPrio []string
	resolveCandidate := func(rawHref, typ string) (string, bool) {
		href := strings.TrimSpace(strings.ReplaceAll(rawHref, `\/`, `/`))
		if href == "" {
			return "", false
		}
		lowerHref := strings.ToLower(href)
		if strings.Contains(typ, "svg") || strings.HasSuffix(lowerHref, ".svg") || strings.HasPrefix(lowerHref, "data:") {
			return "", false
		}
		ref, err := url.Parse(href)
		if err != nil {
			return "", false
		}
		resolved := base.ResolveReference(ref)
		if resolved.Scheme != "http" && resolved.Scheme != "https" || resolved.Hostname() == "" {
			return "", false
		}
		return resolved.String(), true
	}

	// 1. Standard HTML <link rel="...icon..." href="...">
	for _, tag := range linkTagRe.FindAllString(htmlText, 48) {
		var rel, href, typ string
		for _, m := range htmlAttrRe.FindAllStringSubmatch(tag, -1) {
			name := strings.ToLower(m[1])
			val := m[2]
			if val == "" {
				val = m[3]
			}
			if val == "" {
				val = m[4]
			}
			switch name {
			case "rel":
				rel = strings.ToLower(strings.TrimSpace(val))
			case "href":
				href = strings.TrimSpace(val)
			case "type":
				typ = strings.ToLower(strings.TrimSpace(val))
			}
		}
		if !strings.Contains(rel, "icon") || strings.Contains(rel, "mask-icon") {
			continue
		}
		resolved, ok := resolveCandidate(href, typ)
		if !ok {
			continue
		}
		if strings.Contains(rel, "apple-touch-icon") {
			highPrio = append(highPrio, resolved)
		} else {
			normalPrio = append(normalPrio, resolved)
		}
	}

	// 2. <meta name="msapplication-TileImage" content="...">
	for _, tag := range metaTagRe.FindAllString(htmlText, 48) {
		var nameAttr, content string
		for _, m := range htmlAttrRe.FindAllStringSubmatch(tag, -1) {
			k := strings.ToLower(m[1])
			val := m[2]
			if val == "" {
				val = m[3]
			}
			if val == "" {
				val = m[4]
			}
			switch k {
			case "name", "property":
				nameAttr = strings.ToLower(strings.TrimSpace(val))
			case "content":
				content = strings.TrimSpace(val)
			}
		}
		if nameAttr == "msapplication-tileimage" {
			if resolved, ok := resolveCandidate(content, ""); ok {
				normalPrio = append(normalPrio, resolved)
			}
		}
	}

	// 3. Embedded SSR/SPA JSON favicon URLs (e.g. Feishu Hire's "browser_label_img_url")
	for _, m := range embeddedIconJSONRe.FindAllStringSubmatch(htmlText, 16) {
		if len(m) < 2 {
			continue
		}
		if resolved, ok := resolveCandidate(m[1], ""); ok {
			embeddedPrio = append(embeddedPrio, resolved)
		}
	}

	out := make([]string, 0, len(highPrio)+len(normalPrio)+len(embeddedPrio))
	out = append(out, highPrio...)
	out = append(out, normalPrio...)
	out = append(out, embeddedPrio...)
	return out
}

func decodeFaviconBytes(raw []byte) (image.Image, error) {
	if len(raw) == 0 || len(raw) > maxFaviconFileBytes {
		return nil, errInvalidFavicon
	}
	if len(raw) >= 6 && raw[0] == 0 && raw[1] == 0 && raw[2] == 1 && raw[3] == 0 {
		img, err := decodeICO(raw)
		if err != nil {
			return nil, err
		}
		return scaleFavicon(img), nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxFaviconPixels {
		return nil, errInvalidFavicon
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return scaleFavicon(img), nil
}

func scaleFavicon(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return img
	}
	if largest := max(w, h); largest > maxFaviconSide {
		w = max(1, w*maxFaviconSide/largest)
		h = max(1, h*maxFaviconSide/largest)
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// decodeICO decodes standard Windows .ico containers containing either embedded PNG
// frames or uncompressed 32-bit BGRA / 24-bit BGR DIB bitmaps, selecting the highest-resolution frame.
func decodeICO(raw []byte) (image.Image, error) {
	if len(raw) < 6 {
		return nil, errInvalidFavicon
	}
	count := int(binary.LittleEndian.Uint16(raw[4:6]))
	if count <= 0 || count > 64 || len(raw) < 6+count*16 {
		return nil, errInvalidFavicon
	}
	type icoEntry struct {
		w, h, bpp int
		offset    int
		size      int
	}
	entries := make([]icoEntry, 0, count)
	for i := 0; i < count; i++ {
		off := 6 + i*16
		w := int(raw[off])
		if w == 0 {
			w = 256
		}
		h := int(raw[off+1])
		if h == 0 {
			h = 256
		}
		bpp := int(binary.LittleEndian.Uint16(raw[off+6 : off+8]))
		size := int(binary.LittleEndian.Uint32(raw[off+8 : off+12]))
		dataOff := int(binary.LittleEndian.Uint32(raw[off+12 : off+16]))
		if size <= 0 || dataOff < 6+count*16 || dataOff+size > len(raw) {
			continue
		}
		entries = append(entries, icoEntry{w: w, h: h, bpp: bpp, offset: dataOff, size: size})
	}
	slices.SortFunc(entries, func(a, b icoEntry) int {
		areaA, areaB := a.w*a.h, b.w*b.h
		if areaA != areaB {
			if areaA > areaB {
				return -1
			}
			return 1
		}
		if a.bpp > b.bpp {
			return -1
		}
		if a.bpp < b.bpp {
			return 1
		}
		return 0
	})

	for _, e := range entries {
		payload := raw[e.offset : e.offset+e.size]
		if bytes.HasPrefix(payload, pngMagicHeader) {
			cfg, err := png.DecodeConfig(bytes.NewReader(payload))
			if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxFaviconPixels {
				continue
			}
			if img, err := png.Decode(bytes.NewReader(payload)); err == nil {
				return img, nil
			}
			continue
		}
		if img, err := decodeICODIB(payload, e.w, e.h); err == nil {
			return img, nil
		}
	}
	return nil, errInvalidFavicon
}

func decodeICODIB(data []byte, fallbackW, fallbackH int) (image.Image, error) {
	if len(data) < 40 {
		return nil, errInvalidFavicon
	}
	headerSize := int(binary.LittleEndian.Uint32(data[0:4]))
	if headerSize < 40 || headerSize > len(data) {
		return nil, errInvalidFavicon
	}
	w := int(int32(binary.LittleEndian.Uint32(data[4:8])))
	totalH := int(int32(binary.LittleEndian.Uint32(data[8:12])))
	bpp := int(binary.LittleEndian.Uint16(data[14:16]))
	compression := binary.LittleEndian.Uint32(data[16:20])
	if compression != 0 || (bpp != 32 && bpp != 24) {
		return nil, errInvalidFavicon
	}
	if w <= 0 {
		w = fallbackW
	}
	h := totalH / 2
	if h <= 0 {
		h = fallbackH
	}
	if w <= 0 || h <= 0 || int64(w)*int64(h) > maxFaviconPixels {
		return nil, errInvalidFavicon
	}
	bytesPerPixel := bpp / 8
	rowStride := ((w*bytesPerPixel + 3) / 4) * 4
	xorSize := rowStride * h
	if headerSize+xorSize > len(data) {
		return nil, errInvalidFavicon
	}
	xorData := data[headerSize : headerSize+xorSize]
	maskStride := ((w + 31) / 32) * 4
	var maskData []byte
	if headerSize+xorSize+maskStride*h <= len(data) {
		maskData = data[headerSize+xorSize : headerSize+xorSize+maskStride*h]
	}

	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		row := xorData[srcY*rowStride:]
		for x := 0; x < w; x++ {
			px := x * bytesPerPixel
			b, g, r := row[px], row[px+1], row[px+2]
			a := uint8(255)
			if bpp == 32 {
				a = row[px+3]
			} else if maskData != nil {
				maskByte := maskData[srcY*maskStride+x/8]
				if (maskByte & (0x80 >> uint(x%8))) != 0 {
					a = 0
				}
			}
			out.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return out, nil
}
