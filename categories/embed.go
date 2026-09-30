// Package categories bundles the built-in category definitions into the
// binary via go:embed.
package categories

import "embed"

//go:embed *.json
var FS embed.FS
