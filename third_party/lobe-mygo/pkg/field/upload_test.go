package field

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func tempFiles(t *testing.T, names ...string) []string {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, len(names))
	for i, n := range names {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("data of "+n), 0o644); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
	}
	return paths
}

func uploadTester(p UploadProps, content func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Padding(20).Children(func() {
			var inner func()
			if content != nil {
				inner = func() { content(c) }
			}
			Upload(c, p, inner).Label("upload").Width(300)
		})
	}, 400, 300)
}

func names(files []File) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Name
	}
	return out
}

func dropCenter(tt *ui.Tester, t *testing.T, paths ...string) bool {
	t.Helper()
	info, ok := tt.Access("upload")
	if !ok {
		t.Fatal("no upload")
	}
	b := info.Bounds
	return tt.DropFiles(b.X+b.W/2, b.Y+b.H/2, paths...)
}

func TestUploadDraggerDropFiltersByAccept(t *testing.T) {
	paths := tempFiles(t, "a.png", "b.txt", "c.JPG")
	var got []File
	tt := uploadTester(UploadProps{Dragger: true, Multiple: true, Accept: "image/*", OnFiles: func(f []File) { got = f }}, nil)
	if !dropCenter(tt, t, paths...) {
		t.Fatal("dragger did not take the drop")
	}
	tt.Frame()
	if want := []string{"a.png", "c.JPG"}; !slices.Equal(names(got), want) {
		t.Fatalf("files %v, want %v", names(got), want)
	}
	if got[0].Path != paths[0] || got[0].Type != "image/png" || got[0].Size != int64(len("data of a.png")) {
		t.Errorf("file %+v", got[0])
	}
	data, err := got[0].ReadAll()
	if err != nil || string(data) != "data of a.png" {
		t.Errorf("ReadAll %q, %v", data, err)
	}
}

func TestUploadAcceptExtensionsAndExactTypes(t *testing.T) {
	cases := []struct {
		accept, name string
		ok           bool
	}{
		{".PDF, .md", "notes.md", true},
		{".pdf,.md", "a.pdf.txt", false},
		{"application/pdf", "doc.pdf", true},
		{"application/pdf", "doc.png", false},
		{"", "anything.bin", true},
		{" , ", "anything.bin", true},
	}
	for _, k := range cases {
		f := File{Name: k.name, Type: mimeOf(k.name)}
		if got := matchesAccept(f, k.accept); got != k.ok {
			t.Errorf("matchesAccept(%q, %q) = %v, want %v", k.name, k.accept, got, k.ok)
		}
	}
}

func TestUploadSingleTakesTheFirstFile(t *testing.T) {
	paths := tempFiles(t, "a.png", "b.png")
	var got []File
	tt := uploadTester(UploadProps{Dragger: true, OnFiles: func(f []File) { got = f }}, nil)
	dropCenter(tt, t, paths...)
	tt.Frame()
	if want := []string{"a.png"}; !slices.Equal(names(got), want) {
		t.Fatalf("files %v, want %v", names(got), want)
	}
}

func TestUploadBeforeUploadThenMaxCount(t *testing.T) {
	paths := tempFiles(t, "a.png", "skip.png", "b.png", "c.png")
	var got []File
	var seen []string
	tt := uploadTester(UploadProps{Dragger: true, Multiple: true, MaxCount: 2,
		BeforeUpload: func(f File) bool { seen = append(seen, f.Name); return f.Name != "skip.png" },
		OnFiles:      func(f []File) { got = f }}, nil)
	dropCenter(tt, t, paths...)
	tt.Frame()
	if want := []string{"a.png", "b.png"}; !slices.Equal(names(got), want) {
		t.Fatalf("files %v, want %v", names(got), want)
	}
	if len(seen) != 4 {
		t.Errorf("BeforeUpload saw %v, want every file", seen)
	}
}

func TestUploadNothingAcceptedCallsNothing(t *testing.T) {
	paths := tempFiles(t, "b.txt")
	called := false
	tt := uploadTester(UploadProps{Dragger: true, Accept: "image/*", OnFiles: func([]File) { called = true }}, nil)
	dropCenter(tt, t, paths...)
	tt.Frame()
	if called {
		t.Error("OnFiles called with no accepted file")
	}
}

func TestUploadDroppedFolderGivesItsFiles(t *testing.T) {
	paths := tempFiles(t, "pics/a.png", "pics/sub/b.png", "pics/.DS_Store", "pics/.git/x.png")
	var got []File
	tt := uploadTester(UploadProps{Dragger: true, Multiple: true, OnFiles: func(f []File) { got = f }}, nil)
	dropCenter(tt, t, filepath.Dir(paths[0]))
	tt.Frame()
	if want := []string{"a.png", "b.png"}; !slices.Equal(names(got), want) {
		t.Fatalf("files %v, want %v", names(got), want)
	}
}

func TestUploadTriggerTakesNoDrop(t *testing.T) {
	paths := tempFiles(t, "a.png")
	tt := uploadTester(UploadProps{OnFiles: func([]File) {}}, func(c *ui.Context) { ui.Text(c, "Choose") })
	if dropCenter(tt, t, paths...) {
		t.Error("a plain trigger took a drop")
	}
}

type fakePicker struct {
	mu    sync.Mutex
	opts  []PickOptions
	files []File
}

func (p *fakePicker) pick(o PickOptions) ([]File, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.opts = append(p.opts, o)
	return p.files, nil
}

func (p *fakePicker) calls() []PickOptions {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.opts)
}

func settleUntil(tt *ui.Tester, done func() bool) bool {
	for range 200 {
		tt.Frame()
		if done() {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return false
}

func clickCenter(tt *ui.Tester, t *testing.T) {
	t.Helper()
	info, ok := tt.Access("upload")
	if !ok {
		t.Fatal("no upload")
	}
	b := info.Bounds
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
}

func TestUploadClickPicksFiles(t *testing.T) {
	fp := &fakePicker{files: []File{{Name: "a.png", Type: "image/png", Data: []byte("x")}, {Name: "b.txt", Type: "text/plain"}}}
	var got []File
	tt := uploadTester(UploadProps{Accept: "image/*,.md", Multiple: true, Picker: fp.pick, OnFiles: func(f []File) { got = f }},
		func(c *ui.Context) { ui.Text(c, "Choose") })
	clickCenter(tt, t)
	if !settleUntil(tt, func() bool { return got != nil }) {
		t.Fatal("picked files never arrived")
	}
	if want := []string{"a.png"}; !slices.Equal(names(got), want) {
		t.Fatalf("files %v, want %v", names(got), want)
	}
	if data, _ := got[0].ReadAll(); string(data) != "x" {
		t.Errorf("ReadAll %q, want the picked data", data)
	}
	opts := fp.calls()
	if len(opts) != 1 || opts[0] != (PickOptions{Accept: "image/*,.md", Multiple: true}) {
		t.Errorf("picker options %+v", opts)
	}
}

func TestUploadKeyboardPicks(t *testing.T) {
	fp := &fakePicker{files: []File{{Name: "a.png"}}}
	var got []File
	tt := uploadTester(UploadProps{Dragger: true, Directory: true, Picker: fp.pick, OnFiles: func(f []File) { got = f }}, nil)
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if !tt.Focused("upload") {
		t.Fatal("upload not focusable")
	}
	tt.Key(0, ui.KeyEnter)
	if !settleUntil(tt, func() bool { return got != nil }) {
		t.Fatal("Enter did not pick")
	}
	if opts := fp.calls(); len(opts) != 1 || !opts[0].Directory {
		t.Errorf("picker options %+v, want Directory", opts)
	}
}

func TestUploadDisabledAndNoClick(t *testing.T) {
	paths := tempFiles(t, "a.png")
	for _, p := range []UploadProps{{Dragger: true, Disabled: true}, {Dragger: true, NoClick: true}} {
		fp := &fakePicker{files: []File{{Name: "a.png"}}}
		p.Picker = fp.pick
		p.OnFiles = func([]File) {}
		tt := uploadTester(p, nil)
		clickCenter(tt, t)
		settleUntil(tt, func() bool { return false })
		if n := len(fp.calls()); n != 0 {
			t.Errorf("%+v: picker opened %d times", p, n)
		}
		if took := dropCenter(tt, t, paths...); took == p.Disabled {
			t.Errorf("Disabled=%v: drop taken %v", p.Disabled, took)
		}
	}
}

func TestUploadDraggerDefaultTexts(t *testing.T) {
	tt := uploadTester(UploadProps{Dragger: true, Description: "PNG or JPG, up to 5 MB"}, nil)
	for _, s := range []string{"Click or drag file to this area to upload", "PNG or JPG, up to 5 MB"} {
		if !tt.HasText(s) {
			t.Errorf("texts %q, want %q", tt.Texts(), s)
		}
	}
	tt = uploadTester(UploadProps{Dragger: true, Title: "Drop a cover"}, nil)
	if !tt.HasText("Drop a cover") || tt.HasText("Click or drag") {
		t.Errorf("texts %q, want only the custom title", tt.Texts())
	}
}

func TestUploadOpensOnePickerAtATime(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	calls := 0
	pick := func(PickOptions) ([]File, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		<-release
		return nil, nil
	}
	tt := uploadTester(UploadProps{Dragger: true, Picker: pick, OnFiles: func([]File) {}}, nil)
	clickCenter(tt, t)
	tt.Frame()
	clickCenter(tt, t)
	tt.Frame()
	close(release)
	time.Sleep(20 * time.Millisecond)
	tt.Frame()
	clickCenter(tt, t)
	settleUntil(tt, func() bool { mu.Lock(); defer mu.Unlock(); return calls == 2 })
	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Errorf("picker opened %d times, want once while open and again after", calls)
	}
}
