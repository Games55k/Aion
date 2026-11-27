package main

import (
	"context"
	"errors"
	"image/color"
	"net/url"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

var (
	demoLogosOnce  sync.Once
	demoLogosCache map[string]*ui.Bitmap
)

// demoFaviconLoader supplies crisp, deterministic 64×64 vector-rendered company
// logos for the demo job applications so favicons render reliably offline.
func demoFaviconLoader(_ context.Context, rawURL string) (*ui.Bitmap, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(u.Hostname())

	demoLogosOnce.Do(func() {
		demoLogosCache = map[string]*ui.Bitmap{
			"google":    renderLogoSVG(svgLogoGoogle),
			"apple":     renderLogoSVG(svgLogoApple),
			"microsoft": renderLogoSVG(svgLogoMicrosoft),
			"nvidia":    renderLogoSVG(svgLogoNvidia),
			"bytedance": renderLogoSVG(svgLogoByteDance),
			"tencent":   renderLogoSVG(svgLogoTencent),
			"alibaba":   renderLogoSVG(svgLogoAlibaba),
		}
	})

	for key, bmp := range demoLogosCache {
		if strings.Contains(host, key) {
			return bmp, nil
		}
	}
	return nil, errors.New("unknown demo favicon host")
}

func renderLogoSVG(src string) *ui.Bitmap {
	svg := ui.MustParseSVG([]byte(src))
	return ui.NewBitmap(svg.Image(64, 64, color.Black))
}

const svgLogoGoogle = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#FFFFFF"/>
  <path d="M51.6 32.5c0-1.5-.1-2.9-.4-4.3H32v8.2h11c-.5 2.6-2 4.8-4.2 6.3v5.2h6.8c4-3.7 6-9.1 6-15.4z" fill="#4285F4"/>
  <path d="M32 52.5c5.5 0 10.2-1.8 13.6-5l-6.8-5.2c-1.8 1.2-4.2 2-6.8 2-5.2 0-9.7-3.5-11.3-8.3h-7v5.4c3.4 6.7 10.4 11.1 18.3 11.1z" fill="#34A853"/>
  <path d="M20.7 36c-.4-1.2-.6-2.6-.6-4s.2-2.8.6-4v-5.4h-7A20.5 20.5 0 0 0 11.5 32c0 3.3.8 6.4 2.2 9.4l7-5.4z" fill="#FBBC05"/>
  <path d="M32 19.7c3 0 5.7 1 7.8 3l5.8-5.8C42.1 13.6 37.5 11.5 32 11.5c-7.9 0-14.9 4.4-18.3 11.1l7 5.4c1.6-4.8 6.1-8.3 11.3-8.3z" fill="#EA4335"/>
</svg>`

const svgLogoMicrosoft = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#FFFFFF"/>
  <rect x="13" y="13" width="17.5" height="17.5" rx="2" fill="#F25022"/>
  <rect x="33.5" y="13" width="17.5" height="17.5" rx="2" fill="#7FBA00"/>
  <rect x="13" y="33.5" width="17.5" height="17.5" rx="2" fill="#00A4EF"/>
  <rect x="33.5" y="33.5" width="17.5" height="17.5" rx="2" fill="#FFB900"/>
</svg>`

const svgLogoApple = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#1D1D1F"/>
  <path d="M39.8 33.4c0-4.9 4-7.3 4.2-7.4-2.3-3.4-5.9-3.8-7.1-3.9-3-.3-5.9 1.8-7.4 1.8-1.5 0-3.9-1.7-6.4-1.7-3.3 0-6.3 1.9-8 4.9-3.5 6-.9 14.9 2.4 19.8 1.7 2.4 3.6 5.1 6.2 5 2.5-.1 3.4-1.6 6.4-1.6 3 0 3.8 1.6 6.4 1.5 2.7 0 4.4-2.4 6-4.8 1.9-2.8 2.7-5.5 2.7-5.6-.1 0-5.4-2.1-5.4-8z" fill="#F5F5F7"/>
  <path d="M35.1 19.1c1.4-1.7 2.3-4 2-6.3-2 .1-4.4 1.3-5.8 3-1.2 1.4-2.3 3.8-2 6 2.2.2 4.5-1.1 5.8-2.7z" fill="#F5F5F7"/>
</svg>`

const svgLogoNvidia = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#76B900"/>
  <path d="M12 31.5C18.5 22 27.5 18 38 19.5c5.8.8 10.7 3.7 14 7.8-4.5 5.6-11.2 9.4-19.5 9.4-6 0-11-2-14.5-5.2zm6.5.2c2.8 2.4 6.6 3.8 11.2 3.8 5.8 0 10.6-2.4 14-6.2-2.6-2.5-6.2-4.2-10.5-4.6-6.2-.6-11.3 2.1-14.7 7z" fill="#FFFFFF"/>
  <path d="M24 31.6c2.2-2.4 5.2-3.6 8.6-3.2 2.4.3 4.4 1.3 5.9 2.8-2 1.8-4.6 2.8-7.6 2.8-2.8 0-5.2-.9-6.9-2.4z" fill="#FFFFFF"/>
  <path d="M14 43.5c6.2 3.5 13.8 4.8 22 3.6 6.2-.9 11.6-3.5 16-7.2v4.8c-4.8 3.6-10.6 5.9-17.2 6.5-8.2.7-15.6-1.2-20.8-4.5v-3.2z" fill="#FFFFFF" opacity="0.9"/>
</svg>`

const svgLogoByteDance = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#FFFFFF"/>
  <polygon points="14,22 21,19 21,45 14,42" fill="#325AB4"/>
  <polygon points="24,27 31,24 31,42 24,39" fill="#3C8CFF"/>
  <polygon points="34,21 41,18 41,47 34,44" fill="#00B4D8"/>
  <polygon points="44,16 51,13 51,44 44,41" fill="#00D2D3"/>
</svg>`

const svgLogoTencent = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#0052D9"/>
  <path d="M15 20h34l-3.2 8H35.6l-5.8 18H21l5.8-18H15z" fill="#FFFFFF"/>
  <path d="M36 33h13l-2.4 6.5H34z" fill="#68B5FF"/>
</svg>`

const svgLogoAlibaba = `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64">
  <rect width="64" height="64" rx="14" fill="#FF6A00"/>
  <path d="M16 33c0-7.5 6.4-13 15.8-13 9.8 0 16.2 5.4 16.2 13.2 0 7.6-6.2 12.8-16 12.8-9.6 0-16-5.2-16-13zm9 0c0 3.6 2.9 6.1 7 6.1 4.2 0 7-2.5 7-6.1s-2.8-6.1-7-6.1c-4.1 0-7 2.5-7 6.1z" fill="#FFFFFF"/>
  <path d="M21 41c4 3.4 9.5 4.8 15.5 3.8 4.4-.7 8-2.6 10.5-5.3" fill="none" stroke="#FFFFFF" stroke-width="3.6" stroke-linecap="round"/>
</svg>`
