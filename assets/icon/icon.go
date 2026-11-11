// Package icon embeds Aion's shared vector artwork.
package icon

import _ "embed"

// SVG is the app icon, shared by the sidebar and project documentation.
//
//go:embed aion-icon.svg
var SVG string

// PNG is the rasterized app icon, shared by native dialogs and update windows.
//
//go:embed aion-icon.png
var PNG []byte
