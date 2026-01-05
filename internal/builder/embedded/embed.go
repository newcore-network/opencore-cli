package embedded

import (
	"embed"
)

//go:embed *.js
var BuildFS embed.FS

// GetBuildScript returns the main build script content
func GetBuildScript() []byte {
	content, _ := BuildFS.ReadFile("build.js")
	return content
}
