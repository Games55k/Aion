//go:build !js

package field

import (
	"strings"

	"github.com/egoist/mygo"
)

// PickFiles shows the system's open dialog for o and returns the files chosen, nil when canceled.
// It blocks until the dialog closes.
func PickFiles(o PickOptions) ([]File, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Filters: acceptFilters(o.Accept), Multiple: o.Multiple, Directory: o.Directory})
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	files := make([]File, len(paths))
	for i, p := range paths {
		files[i] = fileAt(p)
	}
	return files, nil
}

func acceptFilters(accept string) []mygo.FileFilter {
	patterns := acceptPatterns(accept)
	if len(patterns) == 0 {
		return nil
	}
	f := mygo.FileFilter{Name: "Accepted files"}
	for _, p := range patterns {
		if ext, ok := strings.CutPrefix(p, "."); ok {
			f.Extensions = append(f.Extensions, ext)
		} else {
			f.MIMETypes = append(f.MIMETypes, p)
		}
	}
	return []mygo.FileFilter{f}
}
