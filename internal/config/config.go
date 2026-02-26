package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	Name        string            `json:"name"`
	OutDir      string            `json:"outDir"`
	Destination string            `json:"destination,omitempty"`
	Core        CoreConfig        `json:"core"`
	Resources   ResourcesConfig   `json:"resources"`
	Standalones *StandaloneConfig `json:"standalones,omitempty"`
	Modules     []string          `json:"modules"`
	Build       BuildConfig       `json:"build"`
	Dev         DevConfig         `json:"dev"`
}

type DevConfig struct {
	Port            int    `json:"port"`
	TxAdminURL      string `json:"txAdminUrl,omitempty"`
	TxAdminUser     string `json:"txAdminUser,omitempty"`
	TxAdminPassword string `json:"txAdminPassword,omitempty"`
}

// IsTxAdminConfigured returns true if txAdmin credentials are fully configured
func (d *DevConfig) IsTxAdminConfigured() bool {
	return d.TxAdminURL != "" && d.TxAdminUser != "" && d.TxAdminPassword != ""
}

type CoreConfig struct {
	Path           string       `json:"path"`
	ResourceName   string       `json:"resourceName"`
	EntryPoints    *EntryPoints `json:"entryPoints,omitempty"`
	Views          *ViewsConfig `json:"views,omitempty"`
	Build          *BuildConfig `json:"build,omitempty"`
	CustomCompiler string       `json:"customCompiler,omitempty"` // Path to custom build script
}

type EntryPoints struct {
	Server string `json:"server"`
	Client string `json:"client"`
}

type ResourcesConfig struct {
	Include  []string           `json:"include"`
	Explicit []ExplicitResource `json:"explicit"`
}

type ExplicitResource struct {
	Path           string               `json:"path"`
	ResourceName   string               `json:"resourceName,omitempty"`
	Type           string               `json:"type,omitempty"`
	Compile        *bool                `json:"compile,omitempty"`
	EntryPoints    *EntryPoints         `json:"entryPoints,omitempty"`
	Build          *ResourceBuildConfig `json:"build,omitempty"`
	Views          *ViewsConfig         `json:"views,omitempty"`
	CustomCompiler string               `json:"customCompiler,omitempty"` // Path to custom build script
}

type ResourceBuildConfig struct {
	Server               *bool    `json:"server,omitempty"`
	Client               *bool    `json:"client,omitempty"`
	NUI                  *bool    `json:"nui,omitempty"`
	Minify               *bool    `json:"minify,omitempty"`
	SourceMaps           *bool    `json:"sourceMaps,omitempty"`
	ServerBinaries       []string `json:"serverBinaries,omitempty"`
	ServerBinaryPlatform string   `json:"serverBinaryPlatform,omitempty"`
	LogLevel             string   `json:"logLevel,omitempty"`
}

type StandaloneConfig struct {
	Include  []string           `json:"include"`
	Explicit []ExplicitResource `json:"explicit,omitempty"`
}

type ViewsConfig struct {
	Path         string   `json:"path"`
	Framework    string   `json:"framework,omitempty"`
	EntryPoint   string   `json:"entryPoint,omitempty"`   // Optional: explicit entry point (e.g., "main.ng.ts")
	Ignore       []string `json:"ignore,omitempty"`       // Optional: ignore patterns (e.g., ["*.config.ts", "test/**"])
	ForceInclude []string `json:"forceInclude,omitempty"` // Optional: force include static files by name
	BuildCommand string   `json:"buildCommand,omitempty"` // Optional: custom build command for static frameworks (e.g. Astro)
	OutputDir    string   `json:"outputDir,omitempty"`    // Optional: output directory for static frameworks (e.g. Astro)
}

type BuildSideConfig struct {
	Platform   string   `json:"platform,omitempty"`
	Format     string   `json:"format,omitempty"`
	Target     string   `json:"target,omitempty"`
	External   []string `json:"external,omitempty"`
	Minify     *bool    `json:"minify,omitempty"`
	SourceMaps *bool    `json:"sourceMaps,omitempty"`
}

type BuildConfig struct {
	Minify               bool             `json:"minify"`
	SourceMaps           bool             `json:"sourceMaps"`
	LogLevel             string           `json:"logLevel,omitempty"`
	Target               string           `json:"target,omitempty"`
	Parallel             bool             `json:"parallel"`
	MaxWorkers           int              `json:"maxWorkers,omitempty"`
	ServerBinaries       []string         `json:"serverBinaries,omitempty"`
	ServerBinaryPlatform string           `json:"serverBinaryPlatform,omitempty"`
	Server               *BuildSideConfig `json:"server,omitempty"`
	Client               *BuildSideConfig `json:"client,omitempty"`
}

func isBracketFolderName(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]") && len(name) > 2
}

// FindProjectRoot searches upwards from startDir to locate opencore.config.ts.
func FindProjectRoot(startDir string) (string, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, "opencore.config.ts")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("opencore.config.ts not found in current directory or any parent directory")
}

// LoadWithProjectRoot reads and transpiles opencore.config.ts to Config and returns the project root.
func LoadWithProjectRoot() (*Config, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	root, err := FindProjectRoot(wd)
	if err != nil {
		return nil, "", err
	}

	configPathAbs := filepath.Join(root, "opencore.config.ts")

	// Check if Node.js is installed
	if _, err := exec.LookPath("node"); err != nil {
		return nil, "", fmt.Errorf("Node.js is not installed. Please install Node.js 18+ and try again")
	}

	// Create temporary transpiler script
	transpilerScript := `
const path = require('path');
const { createRequire } = require('module');

// Make sure module resolution happens from the project root (cwd).
const requireFromProject = createRequire(process.cwd() + path.sep);

(async () => {
  try {
    // Use tsx to run TypeScript directly
    const configPath = path.resolve(process.argv[2]);

    // Try to require tsx or ts-node
    let result;
    try {
      requireFromProject('tsx/cjs');
      result = requireFromProject(configPath);
    } catch (e) {
      // Fallback: try to use esbuild-register
      try {
        requireFromProject('esbuild-register/dist/node').register();
        result = requireFromProject(configPath);
      } catch (e2) {
        // Last resort: assume it's already transpiled or use plain require
        result = requireFromProject(configPath);
      }
    }

    const config = result.default || result;
    console.log(JSON.stringify(config, null, 2));
  } catch (error) {
    console.error('Failed to load config:', error.message);
    process.exit(1);
  }
})();
`

	// Write transpiler script to temp file
	tmpFile := filepath.Join(os.TempDir(), "opencore-config-loader.js")
	if err := os.WriteFile(tmpFile, []byte(transpilerScript), 0644); err != nil {
		return nil, "", fmt.Errorf("failed to create transpiler script: %w", err)
	}
	defer os.Remove(tmpFile)

	// Execute transpiler script
	cmd := exec.Command("node", tmpFile, configPathAbs)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("failed to transpile config: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON output
	var config Config
	if err := json.Unmarshal(output, &config); err != nil {
		return nil, "", fmt.Errorf("failed to parse config JSON: %w\nOutput: %s", err, string(output))
	}

	if strings.TrimSpace(config.Name) == "" {
		return nil, "", fmt.Errorf("config.name is required")
	}

	category := config.Name
	if !isBracketFolderName(category) {
		category = fmt.Sprintf("[%s]", config.Name)
	}

	if strings.TrimSpace(config.Destination) != "" {
		// Destination is optional; when provided it is the output base directory.
		// The CLI always creates a FiveM category folder derived from config.name.
		config.Destination = filepath.Join(strings.TrimSpace(config.Destination), category)
		config.OutDir = config.Destination
	} else {
		// When destination is not set, build locally.
		outBase := strings.TrimSpace(config.OutDir)
		if outBase == "" {
			outBase = "build"
		}
		config.OutDir = filepath.Join(outBase, category)
		config.Destination = ""
	}

	if config.Build.Target == "" {
		config.Build.Target = "ES2020"
	}
	if config.Build.LogLevel == "" {
		config.Build.LogLevel = "INFO"
	}
	if config.Dev.Port == 0 {
		config.Dev.Port = 3847
	}

	// Environment variables override config file (higher priority)
	if envURL := os.Getenv("OPENCORE_TXADMIN_URL"); envURL != "" {
		config.Dev.TxAdminURL = envURL
	}
	if envUser := os.Getenv("OPENCORE_TXADMIN_USER"); envUser != "" {
		config.Dev.TxAdminUser = envUser
	}
	if envPass := os.Getenv("OPENCORE_TXADMIN_PASSWORD"); envPass != "" {
		config.Dev.TxAdminPassword = envPass
	}

	return &config, root, nil
}

// Load reads and transpiles opencore.config.ts to Config.
func Load() (*Config, error) {
	cfg, _, err := LoadWithProjectRoot()
	return cfg, err
}

// GetResourcePaths returns all resource paths (including core)
func (c *Config) GetResourcePaths() []string {
	paths := []string{c.Core.Path}

	// Add explicit resources
	for _, res := range c.Resources.Explicit {
		paths = append(paths, res.Path)
	}

	// Add resources matching include glob patterns
	for _, pattern := range c.Resources.Include {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			// Invalid pattern, skip it
			continue
		}
		for _, match := range matches {
			// Only include directories (resources are directories)
			info, err := os.Stat(match)
			if err == nil && info.IsDir() {
				// Avoid duplicates
				isDuplicate := false
				for _, existing := range paths {
					if existing == match {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					paths = append(paths, match)
				}
			}
		}
	}

	return paths
}

// GetStandalonePaths returns all standalone resource paths
func (c *Config) GetStandalonePaths() []string {
	if c.Standalones == nil {
		return nil
	}

	var paths []string

	// Add explicit standalone resources
	for _, res := range c.Standalones.Explicit {
		paths = append(paths, res.Path)
	}

	// Add standalone matching include glob patterns
	for _, pattern := range c.Standalones.Include {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && info.IsDir() {
				isDuplicate := false
				for _, existing := range paths {
					if existing == match {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					paths = append(paths, match)
				}
			}
		}
	}

	return paths
}

// ShouldCompile returns whether a standalone resource should be compiled
func (c *Config) ShouldCompile(path string) bool {
	if c.Standalones == nil {
		return true
	}

	for _, res := range c.Standalones.Explicit {
		if res.Path == path {
			if res.Compile != nil {
				return *res.Compile
			}
			return true // default to compile
		}
	}

	return true // default to compile for glob-matched resources
}

// GetResourceViews returns views config for a specific resource path
func (c *Config) GetResourceViews(path string) *ViewsConfig {
	// Check core
	if path == c.Core.Path && c.Core.Views != nil {
		return c.Core.Views
	}

	// Check explicit resources
	for _, res := range c.Resources.Explicit {
		if res.Path == path && res.Views != nil {
			return res.Views
		}
	}

	// Check standalone
	if c.Standalones != nil {
		for _, res := range c.Standalones.Explicit {
			if res.Path == path && res.Views != nil {
				return res.Views
			}
		}
	}

	return nil
}

// GetExplicitResource returns the explicit resource config for a path, if any
func (c *Config) GetExplicitResource(path string) *ExplicitResource {
	for i := range c.Resources.Explicit {
		if c.Resources.Explicit[i].Path == path {
			return &c.Resources.Explicit[i]
		}
	}
	return nil
}

// GetExplicitStandalone returns the explicit standalone config for a path, if any
func (c *Config) GetExplicitStandalone(path string) *ExplicitResource {
	if c.Standalones == nil {
		return nil
	}
	for i := range c.Standalones.Explicit {
		if c.Standalones.Explicit[i].Path == path {
			return &c.Standalones.Explicit[i]
		}
	}
	return nil
}

// GetCustomCompiler returns the custom compiler path for a specific resource, or empty if using default
func (c *Config) GetCustomCompiler(resourcePath string) string {
	// Check core
	if resourcePath == c.Core.Path {
		return c.Core.CustomCompiler
	}

	// Check explicit resources
	for _, res := range c.Resources.Explicit {
		if res.Path == resourcePath {
			return res.CustomCompiler
		}
	}

	// Check standalone
	if c.Standalones != nil {
		for _, res := range c.Standalones.Explicit {
			if res.Path == resourcePath {
				return res.CustomCompiler
			}
		}
	}

	return "" // Use embedded compiler
}
