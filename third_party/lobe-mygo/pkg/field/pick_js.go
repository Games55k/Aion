//go:build js

package field

import (
	"errors"
	"strings"
	"syscall/js"
)

// PickFiles shows the browser's file chooser for o through a file input and returns the files
// chosen, read into Data, nil when canceled. It blocks until the chooser closes, so it runs off
// the UI's goroutine; browsers open it only shortly after a click or a key press.
func PickFiles(o PickOptions) ([]File, error) {
	doc := js.Global().Get("document")
	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	input.Set("accept", o.Accept)
	input.Set("multiple", o.Multiple || o.Directory)
	if o.Directory {
		input.Set("webkitdirectory", true)
	}
	input.Get("style").Set("display", "none")
	doc.Get("body").Call("appendChild", input)
	defer input.Call("remove")

	done := make(chan js.Value, 1)
	send := func(v js.Value) {
		select {
		case done <- v:
		default:
		}
	}
	onChange := js.FuncOf(func(js.Value, []js.Value) any { send(input.Get("files")); return nil })
	onCancel := js.FuncOf(func(js.Value, []js.Value) any { send(js.Null()); return nil })
	defer onChange.Release()
	defer onCancel.Release()
	input.Call("addEventListener", "change", onChange)
	input.Call("addEventListener", "cancel", onCancel)
	input.Call("click")

	list := <-done
	if list.IsNull() || list.IsUndefined() {
		return nil, nil
	}
	files := make([]File, 0, list.Length())
	for i := range list.Length() {
		f := list.Index(i)
		name := f.Get("name").String()
		if o.Directory && strings.HasPrefix(name, ".") {
			continue
		}
		data, err := readBlob(f)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Name: name, Type: f.Get("type").String(), Size: int64(f.Get("size").Float()), Data: data})
	}
	return files, nil
}

func readBlob(blob js.Value) ([]byte, error) {
	type result struct {
		buf js.Value
		err error
	}
	ch := make(chan result, 1)
	then := js.FuncOf(func(_ js.Value, args []js.Value) any { ch <- result{buf: args[0]}; return nil })
	fail := js.FuncOf(func(_ js.Value, args []js.Value) any {
		ch <- result{err: errors.New(args[0].Call("toString").String())}
		return nil
	})
	defer then.Release()
	defer fail.Release()
	blob.Call("arrayBuffer").Call("then", then, fail)
	r := <-ch
	if r.err != nil {
		return nil, r.err
	}
	bytes := js.Global().Get("Uint8Array").New(r.buf)
	data := make([]byte, bytes.Length())
	js.CopyBytesToGo(data, bytes)
	return data, nil
}
