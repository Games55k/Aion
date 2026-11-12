package field

import (
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// File is a file picked or dropped on an Upload.
type File struct {
	// Name is the file's name, without its folder.
	Name string
	// Path is where the file is, empty for a file picked in a browser, which has no path: Data
	// holds it then.
	Path string
	// Type is the file's media type, as "image/png", guessed from its extension; empty when
	// unknown.
	Type string
	// Size is the file's length in bytes.
	Size int64
	// Data is the content of a file without a Path.
	Data []byte
}

// ReadAll returns the file's content.
func (f File) ReadAll() ([]byte, error) {
	if f.Path == "" {
		return f.Data, nil
	}
	return os.ReadFile(f.Path)
}

// PickOptions are what a Picker is asked to pick.
type PickOptions struct {
	Accept    string
	Multiple  bool
	Directory bool
}

type UploadProps struct {
	// Accept limits the files taken, as HTML's accept: a comma-separated list of extensions
	// (".pdf"), media types ("application/pdf") and kinds of them ("image/*").
	Accept string
	// Multiple takes several files at once; without it only the first file of a pick or a drop is
	// taken.
	Multiple bool
	// Directory picks folders instead of files, which give the files in them.
	Directory bool
	// MaxCount is the most files taken at once, after BeforeUpload; no limit when zero.
	MaxCount int
	// Disabled neither opens the picker nor takes drops.
	Disabled bool
	// NoClick keeps a click or Enter from opening the picker, for a dragger that only takes drops.
	NoClick bool
	// Dragger makes a dashed drop zone that also takes files dropped on it; with no content it
	// shows an icon, Title and Description.
	Dragger bool
	// Title is the default dragger's text: "Click or drag file to this area to upload" when empty.
	Title string
	// Description is a line under the default dragger's Title.
	Description string
	// BeforeUpload is asked about each accepted file in turn; returning false drops it.
	BeforeUpload func(File) bool
	// OnFiles gets the files taken, once for each pick or drop that takes any.
	OnFiles func([]File)
	// Picker shows the picker the click opens, from a goroutine of its own, and returns the files
	// chosen, nil when canceled: PickFiles when nil, mygo's Dialog.Open in a native app and a file
	// input in a browser.
	Picker func(PickOptions) ([]File, error)
}

type uploadInbox struct {
	mu      sync.Mutex
	picked  [][]File
	picking bool
}

func (in *uploadInbox) take() [][]File {
	in.mu.Lock()
	defer in.mu.Unlock()
	out := in.picked
	in.picked = nil
	return out
}

// Upload opens a file picker when clicked, or as a dragger also takes files dropped on it, and
// hands the files it accepts to OnFiles. content is what it shows; a dragger shows its default
// texts when content is nil.
func Upload(c *ui.Context, p UploadProps, content func()) ui.Element {
	tk := theme.TokensOf(c)
	b := ui.ButtonBase(c).Disabled(p.Disabled)
	in := *ui.Local(b, "upload", func() *uploadInbox { return &uploadInbox{} })

	for _, files := range in.take() {
		deliver(files, p)
	}
	if p.Dragger {
		if dropped := b.DroppedFiles(); len(dropped) > 0 && !p.Disabled {
			files := make([]File, 0, len(dropped))
			for _, path := range dropped {
				files = append(files, fileAt(path))
			}
			deliver(files, p)
		}
	}
	if b.Clicked() && !p.Disabled && !p.NoClick {
		openPicker(c, in, p)
	}

	if p.Dragger {
		styleDragger(c, tk, b, p, content)
	} else {
		b.Cursor(ui.CursorPointer)
		if p.Disabled {
			b.Opacity(0.5).Cursor(ui.CursorNotAllowed)
		}
		if content != nil {
			b.Children(content)
		}
	}
	return b
}

func openPicker(c *ui.Context, in *uploadInbox, p UploadProps) {
	in.mu.Lock()
	if in.picking {
		in.mu.Unlock()
		return
	}
	in.picking = true
	in.mu.Unlock()
	pick := p.Picker
	if pick == nil {
		pick = PickFiles
	}
	opts := PickOptions{Accept: p.Accept, Multiple: p.Multiple, Directory: p.Directory}
	wake := c.Services().Invalidate
	go func() {
		files, _ := pick(opts)
		in.mu.Lock()
		in.picking = false
		if len(files) > 0 {
			in.picked = append(in.picked, files)
		}
		in.mu.Unlock()
		wake()
	}()
}

func styleDragger(c *ui.Context, tk *theme.Tokens, b ui.Element, p UploadProps, content func()) {
	over := b.FileDragOver() && !p.Disabled
	hot := over || (b.Hovered() && !p.Disabled)
	border, bg := tk.Border, tk.FillQuaternary
	if hot {
		border, bg = tk.TextTertiary, tk.FillTertiary
	}
	b.Column().AlignItems(ui.Center).Justify(ui.Center).Gap(6).Padding(28, 20).
		Radius(tk.BorderRadiusLG).Border(1, border).BorderStyle(ui.BorderDashed).Background(bg).
		Cursor(ui.CursorPointer).Transition(uploadTransition)
	if p.Disabled {
		b.Opacity(0.5).Cursor(ui.CursorNotAllowed)
	}
	if content != nil {
		b.Children(content)
		return
	}
	title := p.Title
	if title == "" {
		title = "Click or drag file to this area to upload"
	}
	b.Children(func() {
		ui.Icon(c, lucide.Upload()).FontSize(24).TextColor(tk.TextTertiary)
		ui.Text(c, title).FontWeight(500).TextColor(tk.Text).TextAlign(ui.Center)
		if p.Description != "" {
			ui.Text(c, p.Description).FontSize(12).TextColor(tk.TextTertiary).TextAlign(ui.Center)
		}
	})
}

var uploadTransition = ui.ElementTransition{Duration: 150 * time.Millisecond, Colors: true}

func deliver(files []File, p UploadProps) {
	files = expandFolders(files)
	taken := make([]File, 0, len(files))
	for _, f := range files {
		if !matchesAccept(f, p.Accept) {
			continue
		}
		if p.BeforeUpload != nil && !p.BeforeUpload(f) {
			continue
		}
		taken = append(taken, f)
	}
	limit := p.MaxCount
	if !p.Multiple {
		limit = 1
	}
	if limit > 0 && len(taken) > limit {
		taken = taken[:limit]
	}
	if len(taken) > 0 && p.OnFiles != nil {
		p.OnFiles(taken)
	}
}

func expandFolders(files []File) []File {
	out := make([]File, 0, len(files))
	for _, f := range files {
		if f.Path == "" {
			out = append(out, f)
			continue
		}
		info, err := os.Stat(f.Path)
		if err != nil || !info.IsDir() {
			out = append(out, f)
			continue
		}
		_ = filepath.WalkDir(f.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") && path != f.Path {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() {
				out = append(out, fileAt(path))
			}
			return nil
		})
	}
	return out
}

func fileAt(path string) File {
	f := File{Name: filepath.Base(path), Path: path, Type: mimeOf(path)}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		f.Size = info.Size()
	}
	return f
}

func mimeOf(name string) string {
	t := mime.TypeByExtension(filepath.Ext(name))
	if i := strings.IndexByte(t, ';'); i >= 0 {
		t = t[:i]
	}
	return t
}

func acceptPatterns(accept string) []string {
	var out []string
	for _, s := range strings.Split(accept, ",") {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func matchesAccept(f File, accept string) bool {
	patterns := acceptPatterns(accept)
	if len(patterns) == 0 {
		return true
	}
	name, typ := strings.ToLower(f.Name), strings.ToLower(f.Type)
	for _, p := range patterns {
		switch {
		case strings.HasPrefix(p, "."):
			if strings.HasSuffix(name, p) {
				return true
			}
		case strings.HasSuffix(p, "/*"):
			if strings.HasPrefix(typ, p[:len(p)-1]) {
				return true
			}
		case typ == p:
			return true
		}
	}
	return false
}
