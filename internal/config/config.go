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
	Adapter     *AdapterConfig    `json:"adapter,omitempty"`
	Core        CoreConfig        `json:"core"`
	Resources   ResourcesConfig   `json:"resources"`
	Standalones *StandaloneConfig `json:"standalones,omitempty"`
	Modules     []string          `json:"modules"`
	Build       BuildConfig       `json:"build"`
	Dev         DevConfig         `json:"dev"`
}

type AdapterConfig struct {
	Server *AdapterBinding `json:"server,omitempty"`
	Client *AdapterBinding `json:"client,omitempty"`
}

type AdapterBinding struct {
	Name      string                 `json:"name,omitempty"`
	Valid     bool                   `json:"valid"`
	Message   string                 `json:"message,omitempty"`
	Package   string                 `json:"package,omitempty"`
	EntryPath string                 `json:"entryPath,omitempty"`
	Runtime   *AdapterRuntimeBinding `json:"runtime,omitempty"`
}

type AdapterRuntimeBinding struct {
	Runtime  string                   `json:"runtime,omitempty"`
	Server   *AdapterRuntimeSideHints `json:"server,omitempty"`
	Client   *AdapterRuntimeSideHints `json:"client,omitempty"`
	Manifest *AdapterManifestBinding  `json:"manifest,omitempty"`
}

type AdapterRuntimeSideHints struct {
	Platform    string `json:"platform,omitempty"`
	Target      string `json:"target,omitempty"`
	Format      string `json:"format,omitempty"`
	OutFileName string `json:"outFileName,omitempty"`
	OutputRoot  string `json:"outputRoot,omitempty"`
}

type AdapterManifestBinding struct {
	Kind string `json:"kind,omitempty"`
}

type DevConfig struct {
	Bridge          DevBridgeConfig  `json:"bridge,omitempty"`
	Restart         DevRestartConfig `json:"restart,omitempty"`
	TxAdmin         DevTxAdminConfig `json:"txAdmin,omitempty"`
	Process         DevProcessConfig `json:"process,omitempty"`
	Port            int              `json:"port,omitempty"`
	TxAdminURL      string           `json:"txAdminUrl,omitempty"`
	TxAdminUser     string           `json:"txAdminUser,omitempty"`
	TxAdminPassword string           `json:"txAdminPassword,omitempty"`
}

type DevBridgeConfig struct {
	Port int `json:"port,omitempty"`
}

type DevRestartConfig struct {
	Mode string `json:"mode,omitempty"`
}

type DevTxAdminConfig struct {
	URL      string `json:"url,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

type DevProcessConfig struct {
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Cwd           string            `json:"cwd,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	StopSignal    string            `json:"stopSignal,omitempty"`
	StopTimeoutMs int               `json:"stopTimeoutMs,omitempty"`
}

// IsTxAdminConfigured returns true if txAdmin credentials are fully configured
func (d *DevConfig) IsTxAdminConfigured() bool {
	return strings.TrimSpace(d.TxAdmin.URL) != "" && strings.TrimSpace(d.TxAdmin.User) != "" && strings.TrimSpace(d.TxAdmin.Password) != ""
}

func (d *DevConfig) BridgePort() int {
	if d == nil {
		return 3847
	}
	if d.Bridge.Port > 0 {
		return d.Bridge.Port
	}
	if d.Port > 0 {
		return d.Port
	}
	return 3847
}

func (d *DevConfig) RestartMode() string {
	if d == nil {
		return "auto"
	}
	mode := strings.ToLower(strings.TrimSpace(d.Restart.Mode))
	if mode == "" {
		return "auto"
	}
	return mode
}

func (d *DevConfig) HasManagedProcess() bool {
	if d == nil {
		return false
	}
	return strings.TrimSpace(d.Process.Command) != ""
}

func (d *DevConfig) Normalize() {
	if d == nil {
		return
	}

	if d.Bridge.Port == 0 {
		d.Bridge.Port = d.Port
	}
	if d.Bridge.Port == 0 {
		d.Bridge.Port = 3847
	}

	if strings.TrimSpace(d.TxAdmin.URL) == "" {
		d.TxAdmin.URL = strings.TrimSpace(d.TxAdminURL)
	}
	if strings.TrimSpace(d.TxAdmin.User) == "" {
		d.TxAdmin.User = strings.TrimSpace(d.TxAdminUser)
	}
	if strings.TrimSpace(d.TxAdmin.Password) == "" {
		d.TxAdmin.Password = d.TxAdminPassword
	}

	if d.Port == 0 {
		d.Port = d.Bridge.Port
	}
	if strings.TrimSpace(d.TxAdminURL) == "" {
		d.TxAdminURL = d.TxAdmin.URL
	}
	if strings.TrimSpace(d.TxAdminUser) == "" {
		d.TxAdminUser = d.TxAdmin.User
	}
	if strings.TrimSpace(d.TxAdminPassword) == "" {
		d.TxAdminPassword = d.TxAdmin.Password
	}

	if d.Process.StopTimeoutMs <= 0 {
		d.Process.StopTimeoutMs = 5000
	}
	if strings.TrimSpace(d.Process.StopSignal) == "" {
		d.Process.StopSignal = "SIGTERM"
	}
}

func (c *Config) RuntimeKind() string {
	if c == nil || c.Adapter == nil {
		return "fivem"
	}

	if c.Adapter.Server != nil {
		if runtime := inferRuntimeKind(c.Adapter.Server); runtime != "" {
			return runtime
		}
	}
	if c.Adapter.Client != nil {
		if runtime := inferRuntimeKind(c.Adapter.Client); runtime != "" {
			return runtime
		}
	}

	return "fivem"
}

func inferRuntimeKind(binding *AdapterBinding) string {
	if binding == nil {
		return ""
	}
	if binding.Runtime != nil && strings.TrimSpace(binding.Runtime.Runtime) != "" {
		return strings.ToLower(strings.TrimSpace(binding.Runtime.Runtime))
	}
	name := strings.ToLower(strings.TrimSpace(binding.Name))
	switch name {
	case "fivem", "redm", "ragemp", "node":
		return name
	default:
		return ""
	}
}

func (c *Config) UsesSplitRuntimeLayout() bool {
	return c.RuntimeKind() == "ragemp"
}

func (c *Config) ensureBuildSideConfigs() {
	if c.Build.Server == nil {
		c.Build.Server = &BuildSideConfig{}
	}
	if c.Build.Client == nil {
		c.Build.Client = &BuildSideConfig{}
	}
}

func (c *Config) adapterSideTarget(side string) string {
	if c == nil || c.Adapter == nil {
		return ""
	}

	var binding *AdapterBinding
	switch side {
	case "server":
		binding = c.Adapter.Server
	case "client":
		binding = c.Adapter.Client
	default:
		return ""
	}

	if binding == nil || binding.Runtime == nil {
		return ""
	}

	var hints *AdapterRuntimeSideHints
	if side == "server" {
		hints = binding.Runtime.Server
	} else {
		hints = binding.Runtime.Client
	}

	if hints == nil {
		return ""
	}

	return strings.TrimSpace(hints.Target)
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
const os = require('os');
const fs = require('fs');
const { createRequire } = require('module');

// Make sure module resolution happens from the project root (cwd).
const requireFromProject = createRequire(process.cwd() + path.sep);

function inspectAdapterBinding(binding, pkgName, entryPath) {
  if (!binding) {
    return undefined;
  }

  const name = typeof binding.name === 'string' ? binding.name : '';
  const hasRegister = typeof binding.register === 'function';
  const issues = [];

  if (!name) {
    issues.push('missing adapter name');
  }
  if (!hasRegister) {
    issues.push('missing register()');
  }

  return {
    name,
    valid: issues.length === 0,
    message: issues.length > 0 ? issues.join(', ') : undefined,
    package: pkgName,
    entryPath,
    runtime: inspectRuntimeHints(binding.runtime),
  };
}

function inspectRuntimeHints(hints) {
  if (!hints || typeof hints !== 'object') {
    return undefined;
  }

  return {
    runtime: typeof hints.runtime === 'string' ? hints.runtime : undefined,
    server: inspectRuntimeSide(hints.server),
    client: inspectRuntimeSide(hints.client),
    manifest: hints.manifest && typeof hints.manifest === 'object'
      ? { kind: typeof hints.manifest.kind === 'string' ? hints.manifest.kind : undefined }
      : undefined,
  };
}

function inspectRuntimeSide(side) {
  if (!side || typeof side !== 'object') {
    return undefined;
  }

  return {
    platform: typeof side.platform === 'string' ? side.platform : undefined,
    target: typeof side.target === 'string' ? side.target : undefined,
    format: typeof side.format === 'string' ? side.format : undefined,
    outFileName: typeof side.outFileName === 'string' ? side.outFileName : undefined,
    outputRoot: typeof side.outputRoot === 'string' ? side.outputRoot : undefined,
  };
}

async function loadConfig(configPath) {
  const esbuild = requireFromProject('esbuild');
  const outfile = path.join(
    os.tmpdir(),
    'opencore-config-' + process.pid + '-' + Date.now() + '-' + Math.random().toString(16).slice(2) + '.cjs'
  );

  try {
    try {
      requireFromProject('reflect-metadata');
    } catch (_) {}

    await esbuild.build({
      entryPoints: [configPath],
      outfile,
      bundle: true,
      platform: 'node',
      format: 'cjs',
      target: ['node18'],
      absWorkingDir: process.cwd(),
      write: true,
      logLevel: 'silent',
    });

    const result = require(outfile);
    return result.default || result;
  } finally {
    try {
      fs.unlinkSync(outfile);
    } catch (_) {}
  }
}

(async () => {
  try {
    const configPath = path.resolve(process.argv[2]);
    const config = await loadConfig(configPath);
    const serialized = {
      ...config,
      adapter: {
        server: inspectAdapterBinding(config?.adapter?.server, '@open-core/fivem-adapter', '@open-core/fivem-adapter/server'),
        client: inspectAdapterBinding(config?.adapter?.client, '@open-core/fivem-adapter', '@open-core/fivem-adapter/client'),
      },
    };

    if (!serialized.adapter.server && !serialized.adapter.client) {
      delete serialized.adapter;
    }

    console.log(JSON.stringify(serialized, null, 2));
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

	runtimeKind := config.RuntimeKind()
	category := config.Name
	if !isBracketFolderName(category) {
		category = fmt.Sprintf("[%s]", config.Name)
	}

	if strings.TrimSpace(config.Destination) != "" {
		if runtimeKind == "ragemp" {
			config.Destination = strings.TrimSpace(config.Destination)
			config.OutDir = config.Destination
		} else {
			config.Destination = filepath.Join(strings.TrimSpace(config.Destination), category)
			config.OutDir = config.Destination
		}
	} else {
		outBase := strings.TrimSpace(config.OutDir)
		if outBase == "" {
			outBase = "build"
		}
		if runtimeKind == "ragemp" {
			config.OutDir = outBase
		} else {
			config.OutDir = filepath.Join(outBase, category)
		}
		config.Destination = ""
	}

	config.ensureBuildSideConfigs()
	legacyTarget := strings.TrimSpace(config.Build.Target)
	if strings.TrimSpace(config.Build.Server.Target) == "" {
		if adapterTarget := config.adapterSideTarget("server"); adapterTarget != "" {
			config.Build.Server.Target = adapterTarget
		} else if legacyTarget != "" {
			config.Build.Server.Target = legacyTarget
		} else if runtimeKind == "ragemp" {
			config.Build.Server.Target = "node14"
		} else {
			config.Build.Server.Target = "ES2020"
		}
	}
	if strings.TrimSpace(config.Build.Client.Target) == "" {
		if adapterTarget := config.adapterSideTarget("client"); adapterTarget != "" {
			config.Build.Client.Target = adapterTarget
		} else if legacyTarget != "" {
			config.Build.Client.Target = legacyTarget
		} else {
			config.Build.Client.Target = "ES2020"
		}
	}
	if config.Build.LogLevel == "" {
		config.Build.LogLevel = "INFO"
	}

	// Environment variables override config file (higher priority)
	if envURL := os.Getenv("OPENCORE_TXADMIN_URL"); envURL != "" {
		config.Dev.TxAdmin.URL = envURL
	}
	if envUser := os.Getenv("OPENCORE_TXADMIN_USER"); envUser != "" {
		config.Dev.TxAdmin.User = envUser
	}
	if envPass := os.Getenv("OPENCORE_TXADMIN_PASSWORD"); envPass != "" {
		config.Dev.TxAdmin.Password = envPass
	}
	config.Dev.Normalize()

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
