package embedded

import (
	_ "embed"
)

//go:embed build.js
var BuildScript []byte

// GetBuildScript returns the embedded build script content
func GetBuildScript() []byte {
	return BuildScript
}
