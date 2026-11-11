package frontend

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"aion/backend/model"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
	"golang.org/x/sys/unix"
)

var errPreviewTooLarge = errors.New("image exceeds preview resource limit")

const (
	maxBitmapCacheEntries = 16
	maxBitmapCacheBytes   = 32 << 20
	maxPreviewSide        = 1024
	// Bound both encoded input and temporary decoder allocations. Larger files
	// remain ordinary attachments that can be opened or exported in full.
	maxPreviewFileBytes = 32 << 20
	maxPreviewPixels    = 32 << 20
)

// AttachmentBitmap returns a cached preview, scheduling at most one decode at
// a time off the native UI thread. Headless hosts without Dispatch load inline.
func (w *Workbench) AttachmentBitmap(att model.Attachment) *ui.Bitmap {
	if !IsImageAttachment(att) || w.IsRestoring() {
		return nil
	}
	if bmp := w.bitmapCache[att.ID]; bmp != nil {
		w.touchBitmap(att.ID)
		return bmp
	}
	if w.bitmapFailed[att.ID] || w.bitmapLoading {
		return nil
	}
	name := att.StorageName
	if name == "" || name == "." || name == ".." || name != filepath.Base(name) || strings.ContainsAny(name, "/\\:\x00") {
		w.bitmapFailed[att.ID] = true
		return nil
	}
	path := filepath.Join(w.app.DataDir(), "attachments", name)
	taskID, epoch := w.selectedTaskID, w.currentDataEpoch()
	load := w.bitmapLoader
	if load == nil {
		load = loadAttachmentPreview
	}
	apply := func(bmp *ui.Bitmap, err error) {
		w.bitmapLoading = false
		if w.IsRestoring() || epoch != w.currentDataEpoch() || taskID != w.selectedTaskID {
			return
		}
		// A decode may finish after the attachment was removed or its task closed.
		if !slices.ContainsFunc(w.detailAttachments, func(current model.Attachment) bool {
			return current.ID == att.ID && current.StorageName == name
		}) {
			return
		}
		if err != nil || !w.cacheBitmap(att.ID, bmp) {
			w.bitmapFailed[att.ID] = true
		}
	}
	w.bitmapLoading = true
	if dispatch := w.host.Dispatch; dispatch != nil {
		w.taskWG.Add(1)
		go func() {
			defer w.taskWG.Done()
			bmp, err := load(path)
			dispatch(func() { apply(bmp, err) })
		}()
		return nil
	}
	bmp, err := load(path)
	apply(bmp, err)
	return w.bitmapCache[att.ID]
}

func (w *Workbench) touchBitmap(id string) {
	w.bitmapOrder = slices.DeleteFunc(w.bitmapOrder, func(existing string) bool { return existing == id })
	w.bitmapOrder = append(w.bitmapOrder, id)
}

// Include all possible downsample levels created by MyGo when painting smaller
// thumbnails, as well as the base RGBA pixels. No encoded source is retained.
func bitmapBytes(bmp *ui.Bitmap) int64 {
	if bmp == nil {
		return 0
	}
	w, h := bmp.Size()
	var size int64
	for w > 0 && h > 0 {
		size += int64(w) * int64(h) * 4
		if w == 1 && h == 1 {
			break
		}
		w, h = (w+1)/2, (h+1)/2
	}
	return size
}

func (w *Workbench) cacheBitmap(id string, bmp *ui.Bitmap) bool {
	size := bitmapBytes(bmp)
	if size <= 0 || size > maxBitmapCacheBytes {
		return false
	}
	if _, exists := w.bitmapCache[id]; exists {
		delete(w.bitmapCache, id)
		w.bitmapOrder = slices.DeleteFunc(w.bitmapOrder, func(existing string) bool { return existing == id })
	}
	var total int64
	for _, cached := range w.bitmapCache {
		total += bitmapBytes(cached)
	}
	for len(w.bitmapCache) >= maxBitmapCacheEntries || total+size > maxBitmapCacheBytes {
		if len(w.bitmapOrder) == 0 {
			clear(w.bitmapCache)
			break
		}
		oldest := w.bitmapOrder[0]
		w.bitmapOrder = w.bitmapOrder[1:]
		total -= bitmapBytes(w.bitmapCache[oldest])
		delete(w.bitmapCache, oldest)
	}
	w.bitmapCache[id] = bmp
	w.bitmapOrder = append(w.bitmapOrder, id)
	return true
}

func loadAttachmentPreview(path string) (*ui.Bitmap, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > maxPreviewFileBytes {
		return nil, errPreviewTooLarge
	}
	img, err := decodeAttachmentPreview(io.NewSectionReader(f, 0, maxPreviewFileBytes))
	if err != nil {
		return nil, err
	}
	return ui.NewBitmap(img), nil
}

func decodeAttachmentPreview(r io.ReadSeeker) (image.Image, error) {
	config, format, err := image.DecodeConfig(r)
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width) > maxPreviewPixels/int64(config.Height) {
		return nil, errPreviewTooLarge
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	orientation := 1
	if format == "jpeg" {
		orientation = previewOrientation(r)
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if largest := max(w, h); largest > maxPreviewSide {
		w, h = max(1, w*maxPreviewSide/largest), max(1, h*maxPreviewSide/largest)
	}
	preview := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(preview, preview.Bounds(), img, bounds, draw.Src, nil)
	if orientation > 1 {
		return orientedPreview{preview, orientation}, nil
	}
	return preview, nil
}

// Read JPEG metadata in bounded segments, stopping before the compressed image.
// Scaling the decoded image must preserve the camera orientation that MyGo's
// original DecodeBitmap path applied automatically.
func previewOrientation(r io.Reader) int {
	var soi [2]byte
	if _, err := io.ReadFull(r, soi[:]); err != nil || soi != [2]byte{0xff, 0xd8} {
		return 1
	}
	var b [1]byte
	for {
		if _, err := io.ReadFull(r, b[:]); err != nil || b[0] != 0xff {
			return 1
		}
		for b[0] == 0xff {
			if _, err := io.ReadFull(r, b[:]); err != nil {
				return 1
			}
		}
		marker := b[0]
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		// Standalone JPEG markers (TEM, RST0..RST7, SOI, stuffed 0x00) carry no segment length.
		if marker == 0x00 || marker == 0x01 || (marker >= 0xd0 && marker <= 0xd8) {
			continue
		}
		var length [2]byte
		if _, err := io.ReadFull(r, length[:]); err != nil {
			return 1
		}
		n := int64(binary.BigEndian.Uint16(length[:])) - 2
		if n < 0 {
			return 1
		}
		if marker != 0xe1 {
			if _, err := io.CopyN(io.Discard, r, n); err != nil {
				return 1
			}
			continue
		}
		segment := make([]byte, n)
		if _, err := io.ReadFull(r, segment); err != nil {
			return 1
		}
		if orientation, isExif := parseExifOrientation(segment); isExif {
			return orientation
		}
	}
}

func parseExifOrientation(segment []byte) (int, bool) {
	if len(segment) < 14 || !bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
		return 1, false
	}
	tiff := segment[6:]
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1, true
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 1, true
	}
	offset := int(order.Uint32(tiff[4:]))
	if offset < 8 || offset > len(tiff)-2 {
		return 1, true
	}
	count := int(order.Uint16(tiff[offset:]))
	for entry := offset + 2; count > 0 && entry <= len(tiff)-12; entry, count = entry+12, count-1 {
		if order.Uint16(tiff[entry:]) == 0x0112 && order.Uint16(tiff[entry+2:]) == 3 && order.Uint32(tiff[entry+4:]) == 1 {
			if value := int(order.Uint16(tiff[entry+8:])); value >= 1 && value <= 8 {
				return value, true
			}
			return 1, true
		}
	}
	return 1, true
}

// Orient only the small preview, keeping no reference to full-size source pixels.
type orientedPreview struct {
	image       *image.RGBA
	orientation int
}

func (p orientedPreview) ColorModel() color.Model { return color.RGBAModel }

func (p orientedPreview) Bounds() image.Rectangle {
	b := p.image.Bounds()
	if p.orientation >= 5 {
		return image.Rect(0, 0, b.Dy(), b.Dx())
	}
	return b
}

func (p orientedPreview) At(x, y int) color.Color {
	w, h := p.image.Bounds().Dx(), p.image.Bounds().Dy()
	switch p.orientation {
	case 2:
		x = w - 1 - x
	case 3:
		x, y = w-1-x, h-1-y
	case 4:
		y = h - 1 - y
	case 5:
		x, y = y, x
	case 6:
		x, y = y, h-1-x
	case 7:
		x, y = w-1-y, h-1-x
	case 8:
		x, y = w-1-y, x
	}
	return p.image.At(x, y)
}
