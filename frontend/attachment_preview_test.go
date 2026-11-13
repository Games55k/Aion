package frontend

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
)

// Regression: a single large photo previously exceeded the cache budget and retained its full-size pixels.
func TestLargeAttachmentPreviewPreservesOriginalAndBoundsCache(t *testing.T) {
	app, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "large photo"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "photo")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	var original bytes.Buffer
	if err := png.Encode(&original, image.NewNRGBA(image.Rect(0, 0, 3200, 3200))); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "large.png")
	if err := os.WriteFile(source, original.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	wb.ImportAttachmentsToSelectedTask([]string{source})
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	att := wb.detailAttachments[0]
	bmp := wb.AttachmentBitmap(att)
	if bmp == nil {
		t.Fatal("large photo should have a usable preview")
	}
	w, h := bmp.Size()
	if w > maxPreviewSide || h > maxPreviewSide || w != h || bitmapBytes(bmp) > maxBitmapCacheBytes {
		t.Fatalf("unbounded or distorted preview: %dx%d, %d bytes", w, h, bitmapBytes(bmp))
	}
	if wb.AttachmentBitmap(att) != bmp {
		t.Fatal("repeated frames should reuse the preview")
	}
	destination := filepath.Join(t.TempDir(), "original.png")
	if err := app.Attachments.Export(context.Background(), att.ID, destination); err != nil {
		t.Fatal(err)
	}
	exported, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(exported, original.Bytes()) {
		t.Fatalf("generating a small preview changed the exported original: %v", err)
	}
}

// Regression: eviction must enforce both byte and entry limits and reject a single bitmap larger than the entire budget.
func TestAttachmentBitmapCacheBoundsAndEviction(t *testing.T) {
	for _, side := range []int{24, 1024} {
		t.Run(fmt.Sprint(side), func(t *testing.T) {
			_, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})
			for i := 0; i < maxBitmapCacheEntries+2; i++ {
				if !wb.cacheBitmap(fmt.Sprint(i), ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, side, side)))) {
					t.Fatal("bounded preview rejected")
				}
				var total int64
				for _, bmp := range wb.bitmapCache {
					total += bitmapBytes(bmp)
				}
				if len(wb.bitmapCache) > maxBitmapCacheEntries || total > maxBitmapCacheBytes {
					t.Fatalf("cache exceeded budget: entries=%d, bytes=%d", len(wb.bitmapCache), total)
				}
			}
			if wb.bitmapCache["0"] != nil || wb.bitmapCache[fmt.Sprint(maxBitmapCacheEntries+1)] == nil {
				t.Fatal("cache did not evict old previews and retain the newest")
			}
		})
	}
	_, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})
	if wb.cacheBitmap("oversize", ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 3200, 3200)))) || len(wb.bitmapCache) != 0 {
		t.Fatal("a single oversized bitmap bypassed the byte limit")
	}
}

// Replacing an already-cached preview or evicting with desynced LRU order preserves cache bounds without panicking.
func TestAttachmentBitmapCacheReplaceAndDesyncGuard(t *testing.T) {
	_, wb, _ := openTestWorkbench(t, t.TempDir(), Host{})
	for i := 0; i < maxBitmapCacheEntries; i++ {
		if !wb.cacheBitmap(fmt.Sprint(i), ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 16, 16)))) {
			t.Fatal("bounded preview rejected")
		}
	}
	replacement := ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 24, 24)))
	if !wb.cacheBitmap("0", replacement) {
		t.Fatal("replacing existing entry failed")
	}
	if len(wb.bitmapCache) != maxBitmapCacheEntries || wb.bitmapCache["1"] == nil || wb.bitmapCache["0"] != replacement {
		t.Fatalf("replacing an existing key evicted unrelated entries: len=%d", len(wb.bitmapCache))
	}

	wb.bitmapOrder = nil
	if !wb.cacheBitmap("fresh", ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 16, 16)))) || len(wb.bitmapCache) != 1 {
		t.Fatalf("cache did not recover safely from empty LRU order: len=%d", len(wb.bitmapCache))
	}
}

// Regression: image decoding previously blocked rendering; late results must not resurrect closed, deleted, or restored attachment state.
func TestAttachmentPreviewAsyncRenderAndStaleCompletion(t *testing.T) {
	for _, change := range []string{"close", "delete", "restore"} {
		t.Run(change, func(t *testing.T) {
			app, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
			wb.OpenCreateProjectModal()
			wb.projectModalName = "async photo"
			wb.SubmitProjectModal()
			task, err := wb.QuickCreateTask(model.Planned, "photo")
			if err != nil {
				t.Fatal(err)
			}
			wb.SelectTask(task.ID)
			path := filepath.Join(t.TempDir(), "photo.png")
			writeSamplePNG(t, path)
			wb.ImportAttachmentsToSelectedTask([]string{path})
			att := wb.detailAttachments[0]
			dispatched := make(chan func(), 1)
			wb.host.Dispatch = func(fn func()) { dispatched <- fn }
			started, release := make(chan struct{}), make(chan struct{})
			wb.bitmapLoader = func(path string) (*ui.Bitmap, error) {
				close(started)
				<-release
				return loadAttachmentPreview(path)
			}
			if wb.AttachmentBitmap(att) != nil {
				t.Fatal("first request should return without waiting for decoding")
			}
			<-started
			// Returning this frame while the loader is held proves it never waits for I/O.
			tt.Frame()
			if wb.AttachmentBitmap(att) != nil || len(wb.bitmapCache) != 0 {
				t.Fatal("preview should still be pending and duplicate requests deduplicated")
			}
			close(release)
			wb.WaitForTasks()
			if len(wb.bitmapCache) != 0 {
				t.Fatal("worker published state outside the UI callback")
			}
			switch change {
			case "close":
				wb.ClearSelectedTask()
			case "delete":
				if err := app.Attachments.Remove(context.Background(), att.ID); err != nil {
					t.Fatal(err)
				}
				wb.ReloadBoardTasks()
			case "restore":
				wb.bumpDataEpoch()
			}
			(<-dispatched)()
			if wb.bitmapLoading || len(wb.bitmapCache) != 0 || len(wb.bitmapFailed) != 0 {
				t.Fatal("late preview completion repopulated stale state")
			}
			// A later, current request can still finish normally after a stale one.
			if change == "restore" {
				wb.bitmapLoader = nil
				wb.AttachmentBitmap(att)
				wb.WaitForTasks()
				(<-dispatched)()
				if wb.AttachmentBitmap(att) == nil {
					t.Fatal("current preview did not load after discarding a stale result")
				}
			}
		})
	}
}

// Regression: a tiny encoded file can advertise huge dimensions and must be rejected before allocating full-size pixels.
func TestAttachmentPreviewRejectsExcessivePixelDimensions(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	binary.BigEndian.PutUint32(data[16:20], 100000)
	binary.BigEndian.PutUint32(data[20:24], 100000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	if _, err := decodeAttachmentPreview(bytes.NewReader(data)); !errors.Is(err, errPreviewTooLarge) {
		t.Fatalf("oversized dimensions reached the image decoder: %v", err)
	}
}

// Large, transparent previews keep their proportions and alpha while releasing the full-size decoded source.
func TestAttachmentPreviewResizePreservesAspectAndAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2048, 512))
	for y := 0; y < 512; y++ {
		for x := 0; x < 2048; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 255, A: 128})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	preview, err := decodeAttachmentPreview(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Bounds() != image.Rect(0, 0, 1024, 256) {
		t.Fatalf("preview changed aspect ratio: %v", preview.Bounds())
	}
	red, _, _, alpha := preview.At(512, 128).RGBA()
	if alpha != 128*257 || red != alpha {
		t.Fatalf("preview lost premultiplied transparency: red=%d alpha=%d", red, alpha)
	}
}

// Scrolling a native-host attachment preview into view starts its load and publishes the result only on the UI queue.
func TestAttachmentPreviewLoadsWhenScrolledIntoView(t *testing.T) {
	_, wb, tt := openTestWorkbench(t, t.TempDir(), Host{})
	wb.OpenCreateProjectModal()
	wb.projectModalName = "scroll photo"
	wb.SubmitProjectModal()
	task, err := wb.QuickCreateTask(model.Planned, "photo")
	if err != nil {
		t.Fatal(err)
	}
	wb.SelectTask(task.ID)
	for i := 0; i < 16; i++ {
		wb.editSubtasks = append(wb.editSubtasks, model.Subtask{
			ID:    fmt.Sprintf("sub-%d", i),
			Title: fmt.Sprintf("前置子任务 %d", i),
		})
	}
	if err := wb.SaveSelectedTask(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "photo.png")
	writeSamplePNG(t, path)
	wb.ImportAttachmentsToSelectedTask([]string{path})
	wb.pendingToasts = nil
	att := wb.detailAttachments[0]
	dispatched := make(chan func(), 1)
	wb.host.Dispatch = func(fn func()) { dispatched <- fn }
	tt.Frame()
	tt.Advance(time.Second)
	beforeRect, found := tt.Find("图片预览 " + att.Name)
	if !found || beforeRect.Y < 1000 {
		t.Fatalf("expected thumbnail below viewport + InViewMargin before scroll, got found=%v rect=%+v", found, beforeRect)
	}
	if len(dispatched) != 0 {
		t.Fatal("offscreen thumbnail requested a preview before scrolling into view")
	}
	if tt.FramePending() {
		t.Fatal("offscreen thumbnail must not keep requesting animation frames")
	}
	modalRect, ok := tt.Find("任务详情弹窗")
	if !ok {
		t.Fatal("could not find 任务详情弹窗")
	}
	tt.Scroll(modalRect.X+modalRect.W/2, modalRect.Y+modalRect.H/2, 0, 800)
	tt.Frame()
	afterRect, found := tt.Find("图片预览 " + att.Name)
	if !found || afterRect.Y >= 900 {
		t.Fatalf("expected thumbnail inside viewport after scroll, got found=%v rect=%+v", found, afterRect)
	}
	wb.WaitForTasks()
	select {
	case apply := <-dispatched:
		apply()
	default:
		t.Fatal("scrolling into view did not request a preview")
	}
	if wb.AttachmentBitmap(att) == nil {
		t.Fatal("scrolled attachment preview did not load")
	}
}

// Resizing previews preserves JPEG camera orientation and the source's aspect ratio.
func TestAttachmentPreviewPreservesJPEGOrientation(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			if x < 20 {
				source.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
			} else {
				source.SetRGBA(x, y, color.RGBA{B: 255, A: 255})
			}
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, source, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	for orientation := 1; orientation <= 8; orientation++ {
		t.Run(fmt.Sprint(orientation), func(t *testing.T) {
			// Big-endian TIFF with one Orientation SHORT in IFD0.
			exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 1, 18, 0, 3, 0, 0, 0, 1, 0, byte(orientation), 0, 0, 0, 0, 0, 0}
			data := append([]byte{0xff, 0xd8, 0xff, 0xe1, 0, byte(len(exif) + 2)}, exif...)
			data = append(data, encoded.Bytes()[2:]...)
			preview, err := decodeAttachmentPreview(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			w, h := 40, 20
			if orientation >= 5 {
				w, h = h, w
			}
			if preview.Bounds() != image.Rect(0, 0, w, h) {
				t.Fatalf("preview lost orientation: %v", preview.Bounds())
			}
			red, _, blue, _ := preview.At(0, 0).RGBA()
			wantRed := orientation == 1 || orientation == 4 || orientation == 5 || orientation == 6
			if (red > blue) != wantRed {
				t.Fatalf("incorrect camera rotation or reflection: red=%d blue=%d", red, blue)
			}
		})
	}
}

// Regression: JPEG padding 0xff fill bytes and standalone markers previously misaligned EXIF orientation parsing.
func TestAttachmentPreviewHandlesJPEGFillAndStandaloneMarkers(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 40, 20))
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, source, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 1, 18, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}

	// Verify standalone TEM (0x01), RST0 (0xd0), and padded 0xff bytes directly in previewOrientation.
	withTEM := []byte{0xff, 0xd8, 0xff, 0xff, 0x01, 0xff, 0xd0, 0xff, 0xff, 0xe1, 0, byte(len(exif) + 2)}
	withTEM = append(withTEM, exif...)
	withTEM = append(withTEM, encoded.Bytes()[2:]...)
	if got := previewOrientation(bytes.NewReader(withTEM)); got != 6 {
		t.Fatalf("expected orientation 6 after TEM/RST0/padded markers, got %d", got)
	}

	// Verify full JPEG decode + orientation with markers supported by Go's JPEG decoder (0xff fill + RST0).
	decodable := []byte{0xff, 0xd8, 0xff, 0xd0, 0xff, 0xff, 0xff, 0xe1, 0, byte(len(exif) + 2)}
	decodable = append(decodable, exif...)
	decodable = append(decodable, encoded.Bytes()[2:]...)
	preview, err := decodeAttachmentPreview(bytes.NewReader(decodable))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Bounds() != image.Rect(0, 0, 20, 40) {
		t.Fatalf("expected rotated 20x40 bounds, got %v", preview.Bounds())
	}
}
