package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	Name      string         `json:"name"`
	OutDir    string         `json:"outDir"`
	Core      CoreConfig     `json:"core"`
	Resources ResourceConfig `json:"resources"`
	Modules   []string       `json:"modules"`
	Build     BuildConfig    `json:"build"`
}

type CoreConfig struct {
	Path         string `json:"path"`
	ResourceName string `json:"resourceName"`
}

type ResourceConfig struct {
	Include  []string `json:"include"`
	Explicit []ExplicitResource `json:"explicit"`
}

type ExplicitResource struct {
	Path         string `json:"path"`
	ResourceName string `json:"resourceName"`
}

type BuildConfig struct {
	Minify     bool `json:"minify"`
	SourceMaps bool `json:"sourceMaps"`
}

// Load reads and transpiles opencore.config.ts to Config
func Load() (*Config, error) {
	configPath := "opencore.config.ts"
	
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
		config.OutDir = "./dist/resources"
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
	
	// TODO: Add glob pattern matching for include
	
	return paths
}

