package builder

import (
	"encoding/json"
	"time"
)

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

// BuildSideOptions represents per-side build options that are forwarded to build.js.
// In JSON, `server`/`client` can be either:
// - false: skip building that side
// - object: these options
type BuildSideOptions struct {
	Platform   string   `json:"platform,omitempty"`
	Format     string   `json:"format,omitempty"`
	Target     string   `json:"target,omitempty"`
	External   []string `json:"external,omitempty"`
	Minify     *bool    `json:"minify,omitempty"`
	SourceMaps *bool    `json:"sourceMaps,omitempty"`
}

// SideConfigValue marshals as either:
// - false (to skip building that side)
// - an object with BuildSideOptions (to configure that side)
type SideConfigValue struct {
	Enabled bool
	Options *BuildSideOptions
}

func (s SideConfigValue) MarshalJSON() ([]byte, error) {
	if !s.Enabled {
		return []byte("false"), nil
	}
	if s.Options == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(s.Options)
}

func (s *SideConfigValue) UnmarshalJSON(data []byte) error {
	// Accept bool or object
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		s.Enabled = b
		s.Options = nil
		return nil
	}

	var opts BuildSideOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return err
	}
	// Object implies enabled
	s.Enabled = true
	s.Options = &opts
	return nil
}

// BuildOptions contains build configuration for a resource
type BuildOptions struct {
	PackageManager        string          `json:"packageManager,omitempty"`
	Server               SideConfigValue `json:"server"`
	Client               SideConfigValue `json:"client"`
	NUI                  bool            `json:"nui"`
	Minify               bool            `json:"minify"`
	SourceMaps           bool            `json:"sourceMaps"`
	LogLevel             string          `json:"logLevel"`
	Target               string          `json:"target"`
	EntryPoints          *EntryPoints    `json:"entryPoints,omitempty"`
	Framework            string          `json:"framework,omitempty"`            // react, vue, svelte
	Compile              bool            `json:"compile"`                        // for standalone resources
	ViewEntry            string          `json:"viewEntry,omitempty"`            // explicit entry point for views (e.g., "main.ng.ts")
	Ignore               []string        `json:"ignore,omitempty"`               // ignore patterns for views (e.g., ["*.config.ts"])
	ForceInclude         []string        `json:"forceInclude,omitempty"`         // force include static files by name
	BuildCommand         string          `json:"buildCommand,omitempty"`         // custom build command for static frameworks (e.g. Astro)
	OutputDir            string          `json:"outputDir,omitempty"`            // output directory for static frameworks (e.g. Astro)
	ServerBinaries       []string        `json:"serverBinaries,omitempty"`       // server binary files to copy alongside server.js
	ServerBinaryPlatform string          `json:"serverBinaryPlatform,omitempty"` // platform to select binaries from bin/<platform>
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
