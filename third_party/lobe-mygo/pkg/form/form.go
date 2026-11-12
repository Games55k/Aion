package form

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type ValidateOn int

const (
	// ValidateAuto checks a field as it loses the focus once it was edited or the form submitted,
	// and as it changes once it shows an error.
	ValidateAuto ValidateOn = iota
	ValidateChange
	ValidateBlur
	ValidateSubmit
)

type trigger int

const (
	onChange trigger = iota
	onBlur
	onSubmit
	onDeps
)

const requiredText = "This field is required"

// Form keeps what a form knows besides its values: the values it started from, which fields were
// touched, their errors and the checks still running. The app edits Values directly, binding its
// controls to its fields.
type Form[T any] struct {
	Values   T
	baseline T
	host
}

// New returns a form whose Values and baseline, for IsDirty and Reset, are initial.
func New[T any](initial T) *Form[T] {
	return &Form[T]{Values: deepCopy(initial), baseline: deepCopy(initial), host: host{fields: map[string]*fieldState{}}}
}

// IsDirty reports whether Values differ from the values the form started from or last submitted.
func (f *Form[T]) IsDirty() bool { return !reflect.DeepEqual(f.Values, f.baseline) }

// Reset puts back the values the form started from or last submitted, and forgets its errors.
func (f *Form[T]) Reset() {
	f.Values = deepCopy(f.baseline)
	f.clear()
}

// ResetTo makes values the form's new baseline and Values.
func (f *Form[T]) ResetTo(values T) {
	f.baseline = deepCopy(values)
	f.Reset()
}

type asyncResult struct {
	name  string
	token int
	err   string
}

type fieldState struct {
	p                FieldProps
	last             any
	seen, registered bool
	err, server      string
	touched, edited  bool
	focused          bool
	validating       bool
	token            int
	cancel           context.CancelFunc
}

type host struct {
	view          viewSettings
	fields        map[string]*fieldState
	order         []string
	submitCount   int
	submitting    bool
	changed       bool
	changePending bool
	groupFields   int
	wake          func()
	mu            sync.Mutex
	results       []asyncResult
	dirty         func() bool
	reset         func()
	submit        func()
}

type viewSettings struct {
	variant      Variant
	layout       Layout
	itemMinWidth float32
	validateOn   ValidateOn
}

func (h *host) formHost() *host { return h }

// Error returns the error the field named name shows, or "".
func (h *host) Error(name string) string {
	if fs := h.fields[name]; fs != nil {
		if fs.server != "" {
			return fs.server
		}
		return fs.err
	}
	return ""
}

// SetError shows err under the field named name until its value changes, as a server's answer.
func (h *host) SetError(name, err string) {
	fs := h.field(name)
	fs.server = err
}

// Validating reports whether a check of the field named name is still running.
func (h *host) Validating(name string) bool {
	fs := h.fields[name]
	return fs != nil && fs.validating
}

func (h *host) field(name string) *fieldState {
	fs := h.fields[name]
	if fs == nil {
		fs = &fieldState{}
		h.fields[name] = fs
	}
	return fs
}

func (h *host) clear() {
	for _, fs := range h.fields {
		if fs.cancel != nil {
			fs.cancel()
		}
	}
	h.fields = map[string]*fieldState{}
	h.submitCount, h.submitting = 0, false
}

func (h *host) shouldValidate(fs *fieldState, t trigger) bool {
	mode := h.view.validateOn
	if fs.p.ValidateOn != ValidateAuto {
		mode = fs.p.ValidateOn
	}
	switch {
	case t == onSubmit || t == onDeps:
		return true
	case mode == ValidateChange:
		return true
	case mode == ValidateBlur:
		return t == onBlur
	case mode == ValidateSubmit:
		return false
	case fs.err != "":
		return true
	}
	return t == onBlur && (fs.edited || h.submitCount > 0)
}

func (h *host) schedule(name string, fs *fieldState, t trigger) {
	if h.shouldValidate(fs, t) {
		h.run(name, fs)
	}
}

func (h *host) run(name string, fs *fieldState) {
	fs.token++
	if fs.cancel != nil {
		fs.cancel()
		fs.cancel = nil
	}
	fs.validating = false
	p := fs.p
	switch {
	case p.Required && isEmpty(p.Value):
		fs.err = requiredText
		if p.RequiredText != "" {
			fs.err = p.RequiredText
		}
		return
	case p.Validate != nil:
		if err := p.Validate(); err != "" {
			fs.err = err
			return
		}
	}
	if p.ValidateAsync == nil {
		fs.err = ""
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	fs.cancel, fs.validating = cancel, true
	token, value, check, wake := fs.token, deepCopy(p.Value), p.ValidateAsync, h.wake
	go func() {
		if p.ValidateDebounce > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.ValidateDebounce):
			}
		}
		err := check(ctx, value)
		if ctx.Err() != nil {
			return
		}
		h.mu.Lock()
		h.results = append(h.results, asyncResult{name: name, token: token, err: err})
		h.mu.Unlock()
		if wake != nil {
			wake()
		}
	}()
}

func (h *host) drain() {
	h.mu.Lock()
	results := h.results
	h.results = nil
	h.mu.Unlock()
	for _, r := range results {
		if fs := h.fields[r.name]; fs != nil && fs.token == r.token {
			fs.err, fs.validating = r.err, false
		}
	}
}

func (h *host) busy() bool {
	for _, fs := range h.fields {
		if fs.validating {
			return true
		}
	}
	return false
}

func (h *host) hasErrors() bool {
	for _, fs := range h.fields {
		if fs.err != "" || fs.server != "" {
			return true
		}
	}
	return false
}

func (h *host) validateAll() {
	h.submitCount++
	for _, name := range h.order {
		fs := h.fields[name]
		fs.touched = true
		h.run(name, fs)
	}
}

func (h *host) observe(name string, p FieldProps, focused bool) *fieldState {
	fs := h.field(name)
	fs.p, fs.registered = p, true
	h.order = append(h.order, name)
	if !fs.seen {
		fs.seen, fs.last, fs.focused = true, deepCopy(p.Value), focused
		return fs
	}
	if !reflect.DeepEqual(p.Value, fs.last) {
		fs.last = deepCopy(p.Value)
		fs.touched, fs.edited, fs.server = true, true, ""
		h.changed = true
		h.schedule(name, fs, onChange)
		for other, ofs := range h.fields {
			if other != name && ofs.touched && contains(ofs.p.Deps, name) {
				h.schedule(other, ofs, onDeps)
			}
		}
	}
	if fs.focused && !focused {
		fs.touched = true
		h.schedule(name, fs, onBlur)
	}
	fs.focused = focused
	return fs
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return r.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return r.IsNil()
	}
	return false
}

func deepCopy[T any](v T) T {
	src := reflect.ValueOf(&v).Elem()
	dst := reflect.New(src.Type()).Elem()
	copyValue(dst, src)
	return dst.Interface().(T)
}

func copyValue(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Slice:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Len()))
		for i := 0; i < src.Len(); i++ {
			copyValue(dst.Index(i), src.Index(i))
		}
	case reflect.Map:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.MakeMapWithSize(src.Type(), src.Len()))
		for _, k := range src.MapKeys() {
			v := reflect.New(src.Type().Elem()).Elem()
			copyValue(v, src.MapIndex(k))
			dst.SetMapIndex(k, v)
		}
	case reflect.Pointer:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.New(src.Type().Elem()))
		copyValue(dst.Elem(), src.Elem())
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			if dst.Field(i).CanSet() {
				copyValue(dst.Field(i), src.Field(i))
			} else {
				dst.Set(src)
				return
			}
		}
	case reflect.Interface:
		if src.IsNil() {
			return
		}
		v := reflect.New(src.Elem().Type()).Elem()
		copyValue(v, src.Elem())
		dst.Set(v)
	default:
		dst.Set(src)
	}
}
