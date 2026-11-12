//go:build !js

package field

import (
	"slices"
	"testing"
)

func TestUploadAcceptFilters(t *testing.T) {
	got := acceptFilters("image/*, .PDF,application/zip")
	if len(got) != 1 || !slices.Equal(got[0].Extensions, []string{"pdf"}) ||
		!slices.Equal(got[0].MIMETypes, []string{"image/*", "application/zip"}) {
		t.Errorf("filters %+v", got)
	}
	if acceptFilters("") != nil {
		t.Error("filters for no accept")
	}
}
