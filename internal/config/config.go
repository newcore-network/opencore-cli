package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	Name        string            `json:"name"`
	OutDir      string            `json:"outDir"`
	Destination string            `json:"destination,omitempty"`
	Core        CoreConfig        `json:"core"`
	Resources   ResourcesConfig   `json:"resources"`
	Standalone  *StandaloneConfig `json:"standalone,omitempty"`
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
	Server     *bool `json:"server,omitempty"`
	Client     *bool `json:"client,omitempty"`
	NUI        *bool `json:"nui,omitempty"`
	Minify     *bool `json:"minify,omitempty"`
	SourceMaps *bool `json:"sourceMaps,omitempty"`
}

type StandaloneConfig struct {
	Include  []string           `json:"include"`
	Explicit []ExplicitResource `json:"explicit,omitempty"`
}

type ViewsConfig struct {
	Path      string `json:"path"`
	Framework string `json:"framework,omitempty"`
}

type BuildConfig struct {
	Minify     bool   `json:"minify"`
	SourceMaps bool   `json:"sourceMaps"`
	Target     string `json:"target,omitempty"`
	Parallel   bool   `json:"parallel"`
	MaxWorkers int    `json:"maxWorkers,omitempty"`
}

// Load reads and transpiles opencore.config.ts to Config
func Load() (*Config, error) {
	configPath := "opencore.config.ts"

	// Check if Node.js is installed
	if _, err := exec.LookPath("node"); err != nil {
		return nil, fmt.Errorf("Node.js is not installed. Please install Node.js 18+ and try again")
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("opencore.config.ts not found in current directory")
	}

	// Create temporary transpiler script
	transpilerScript := `
const { pathToFileURL } = require('url');
const path = require('path');

(async () => {
  try {
    // Use tsx to run TypeScript directly
    const configPath = path.resolve(process.argv[2]);

    // Try to require tsx or ts-node
    let result;
    try {
      require('tsx/cjs');
      result = require(configPath);
    } catch (e) {
      // Fallback: try to use esbuild-register
      try {
        require('esbuild-register/dist/node').register();
        result = require(configPath);
      } catch (e2) {
        // Last resort: assume it's already transpiled or use plain require
        result = require(configPath);
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
		return nil, fmt.Errorf("failed to create transpiler script: %w", err)
	}
	defer os.Remove(tmpFile)

	// Execute transpiler script
	cmd := exec.Command("node", tmpFile, configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to transpile config: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON output
	var config Config
	if err := json.Unmarshal(output, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w\nOutput: %s", err, string(output))
	}

	// Set defaults
	if config.OutDir == "" {
		config.OutDir = "./build"
	}
	if config.Build.Target == "" {
		config.Build.Target = "ES2020"
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

	return &config, nil
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
	if c.Standalone == nil {
		return nil
	}

	var paths []string

	// Add explicit standalone resources
	for _, res := range c.Standalone.Explicit {
		paths = append(paths, res.Path)
	}

	// Add standalone matching include glob patterns
	for _, pattern := range c.Standalone.Include {
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
	if c.Standalone == nil {
		return true
	}

	for _, res := range c.Standalone.Explicit {
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
	if c.Standalone != nil {
		for _, res := range c.Standalone.Explicit {
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
	if c.Standalone == nil {
		return nil
	}
	for i := range c.Standalone.Explicit {
		if c.Standalone.Explicit[i].Path == path {
			return &c.Standalone.Explicit[i]
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
	if c.Standalone != nil {
		for _, res := range c.Standalone.Explicit {
			if res.Path == resourcePath {
				return res.CustomCompiler
			}
		}
	}

	return "" // Use embedded compiler
}
