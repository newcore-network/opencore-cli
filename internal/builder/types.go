package builder

import "time"

// ResourceType defines the type of resource for build configuration
type ResourceType string

const (
	TypeCore       ResourceType = "core"
	TypeResource   ResourceType = "resource"
	TypeStandalone ResourceType = "standalone"
	TypeViews      ResourceType = "views"
	TypeCopy       ResourceType = "copy" // standalone without compilation
)

// BuildTask represents a single build task
type BuildTask struct {
	Path           string
	ResourceName   string
	Type           ResourceType
	OutDir         string
	Options        BuildOptions
	CustomCompiler string // Path to custom compiler, empty = use embedded
}

// BuildOptions contains build configuration for a resource
type BuildOptions struct {
	Server      bool         `json:"server"`
	Client      bool         `json:"client"`
	NUI         bool         `json:"nui"`
	Minify      bool         `json:"minify"`
	SourceMaps  bool         `json:"sourceMaps"`
	Target      string       `json:"target"`
	EntryPoints *EntryPoints `json:"entryPoints,omitempty"`
	Framework   string       `json:"framework,omitempty"` // react, vue, svelte
	Compile     bool         `json:"compile"`             // for standalone resources
	ViewEntry   string       `json:"viewEntry,omitempty"` // explicit entry point for views (e.g., "main.ng.ts")
	Ignore      []string     `json:"ignore,omitempty"`    // ignore patterns for views (e.g., ["*.config.ts"])
}

// EntryPoints defines entry points for core builds
type EntryPoints struct {
	Server string `json:"server"`
	Client string `json:"client"`
}

// BuildResult represents the result of a build task
type BuildResult struct {
	Task     BuildTask
	Success  bool
	Duration time.Duration
	Error    error
	Output   string
}

// BuildProgress represents build progress for UI
type BuildProgress struct {
	Total     int
	Completed int
	Current   string
	Results   []BuildResult
}
