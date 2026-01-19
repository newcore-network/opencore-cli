package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

type Builder struct {
	config          *config.Config
	resourceBuilder *ResourceBuilder
	deployer        *Deployer
}

func buildSideOptionsFromConfig(cfg *config.BuildSideConfig) *BuildSideOptions {
	if cfg == nil {
		return nil
	}
	return &BuildSideOptions{
		Platform:   cfg.Platform,
		Format:     cfg.Format,
		Target:     cfg.Target,
		External:   cfg.External,
		Minify:     cfg.Minify,
		SourceMaps: cfg.SourceMaps,
	}
}

func mergeBuildSideConfig(base *config.BuildSideConfig, override *config.BuildSideConfig) *config.BuildSideConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	merged := &config.BuildSideConfig{}

	merged.Platform = base.Platform
	if override.Platform != "" {
		merged.Platform = override.Platform
	}

	merged.Format = base.Format
	if override.Format != "" {
		merged.Format = override.Format
	}

	merged.Target = base.Target
	if override.Target != "" {
		merged.Target = override.Target
	}

	merged.External = base.External
	// nil means "not specified"; empty slice means "specified as empty"
	if override.External != nil {
		merged.External = override.External
	}

	merged.Minify = base.Minify
	if override.Minify != nil {
		merged.Minify = override.Minify
	}

	merged.SourceMaps = base.SourceMaps
	if override.SourceMaps != nil {
		merged.SourceMaps = override.SourceMaps
	}

	return merged
}

func buildSideValue(enabled bool, cfg *config.BuildSideConfig) SideConfigValue {
	if !enabled {
		return SideConfigValue{Enabled: false, Options: nil}
	}
	return SideConfigValue{Enabled: true, Options: buildSideOptionsFromConfig(cfg)}
}

func New(cfg *config.Config) *Builder {
	return &Builder{
		config:          cfg,
		resourceBuilder: NewResourceBuilder("."),
		deployer:        NewDeployer(cfg),
	}
}

func (b *Builder) CollectTasks() []BuildTask {
	return b.collectAllTasks()
}

// Build executes the full build process
func (b *Builder) Build() error {
	// Cleanup embedded script on exit
	defer b.resourceBuilder.Cleanup()

	// Collect all build tasks
	tasks := b.collectAllTasks()

	if len(tasks) == 0 {
		return fmt.Errorf("no resources to build")
	}

	// Clean only the resources we are about to build
	uniqueResources := make(map[string]struct{})
	for _, task := range tasks {
		// ResourceName can be "core" or "myresource/ui"
		baseResource := strings.Split(task.ResourceName, "/")[0]
		uniqueResources[baseResource] = struct{}{}
	}

	for baseResource := range uniqueResources {
		if err := b.cleanResourceOutputDir(baseResource); err != nil {
			return fmt.Errorf("failed to clean resource output directory: %w", err)
		}
	}

	// Determine number of workers
	workers := b.config.Build.MaxWorkers
	if workers == 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(tasks) {
		workers = len(tasks)
	}

	// Build with parallel or sequential mode
	var results []BuildResult
	var err error

	if b.config.Build.Parallel && len(tasks) > 1 {
		results, err = b.buildParallel(tasks, workers)
	} else {
		results, err = b.buildSequential(tasks)
	}

	if err != nil {
		return err
	}

	// Deploy to destination if configured and necessary
	if b.deployer.ShouldDeploy() {
		fmt.Printf("\n%s Deploying to %s...\n", ui.Info("→"), b.config.Destination)
		if err := b.deployer.Deploy(); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
		fmt.Println(ui.Success("Deployed successfully!"))
	}

	// Show final summary
	b.showSummary(results)

	return nil
}

func (b *Builder) BuildTasks(tasks []BuildTask) ([]BuildResult, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	// Cleanup embedded script on exit
	defer b.resourceBuilder.Cleanup()

	uniqueResources := make(map[string]struct{})
	for _, task := range tasks {
		baseResource := strings.Split(task.ResourceName, "/")[0]
		uniqueResources[baseResource] = struct{}{}
	}

	for baseResource := range uniqueResources {
		if err := b.cleanResourceOutputDir(baseResource); err != nil {
			return nil, fmt.Errorf("failed to clean resource output directory: %w", err)
		}
	}

	results, err := b.buildSequential(tasks)
	if err != nil {
		return results, err
	}

	if b.deployer.ShouldDeploy() {
		for baseResource := range uniqueResources {
			if err := b.deployer.DeployResource(baseResource); err != nil {
				return results, fmt.Errorf("deployment failed for %s: %w", baseResource, err)
			}
		}
	}

	return results, nil
}

// findViewsPath searches for a views directory in a resource path
func findViewsPath(resourcePath string) string {
	viewDirs := []string{"ui", "view", "views", "web", "html"}
	for _, dir := range viewDirs {
		path := filepath.Join(resourcePath, dir)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return ""
}

// collectAllTasks gathers all build tasks from config
func (b *Builder) collectAllTasks() []BuildTask {
	var tasks []BuildTask

	// Core task
	coreBuildCfg := &b.config.Build
	if b.config.Core.Build != nil {
		coreBuildCfg = b.config.Core.Build
	}

	coreServerCfg := mergeBuildSideConfig(b.config.Build.Server, coreBuildCfg.Server)
	coreClientCfg := mergeBuildSideConfig(b.config.Build.Client, coreBuildCfg.Client)

	// Determine log level
	logLevel := b.config.Build.LogLevel
	if coreBuildCfg.LogLevel != "" {
		logLevel = coreBuildCfg.LogLevel
	}
	if logLevel == "" {
		logLevel = "INFO"
	}

	coreTask := BuildTask{
		Path:           b.config.Core.Path,
		ResourceName:   b.config.Core.ResourceName,
		Type:           TypeCore,
		OutDir:         filepath.Join(b.config.OutDir, b.config.Core.ResourceName),
		CustomCompiler: b.config.Core.CustomCompiler,
		Options: BuildOptions{
			Server:     buildSideValue(true, coreServerCfg),
			Client:     buildSideValue(true, coreClientCfg),
			Minify:     b.config.Build.Minify,
			SourceMaps: b.config.Build.SourceMaps,
			LogLevel:   logLevel,
			Target:     b.config.Build.Target,
			Compile:    true,
		},
	}

	// Add entry points if configured
	if b.config.Core.EntryPoints != nil {
		coreTask.Options.EntryPoints = &EntryPoints{
			Server: b.config.Core.EntryPoints.Server,
			Client: b.config.Core.EntryPoints.Client,
		}
	}

	tasks = append(tasks, coreTask)

	// Core views if configured
	if b.config.Core.Views != nil {
		tasks = append(tasks, BuildTask{
			Path:           b.config.Core.Views.Path,
			ResourceName:   b.config.Core.ResourceName + "/ui",
			Type:           TypeViews,
			OutDir:         filepath.Join(b.config.OutDir, b.config.Core.ResourceName, "ui"),
			CustomCompiler: b.config.Core.CustomCompiler, // Use core's custom compiler for views too
			Options: BuildOptions{
				Framework:    b.config.Core.Views.Framework,
				Minify:       b.config.Build.Minify,
				SourceMaps:   b.config.Build.SourceMaps,
				ViewEntry:    b.config.Core.Views.EntryPoint,
				Ignore:       b.config.Core.Views.Ignore,
				ForceInclude: b.config.Core.Views.ForceInclude,
			},
		})
	}

	// Resources from glob patterns
	for _, pattern := range b.config.Resources.Include {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || !info.IsDir() {
				continue
			}

			resourceName := filepath.Base(match)

			// Skip if it's the core path
			if match == b.config.Core.Path {
				continue
			}

			// Check for explicit override
			explicit := b.config.GetExplicitResource(match)

			// Auto-discover views if not explicitly configured
			var viewsConfig *config.ViewsConfig
			if explicit != nil && explicit.Views != nil {
				viewsConfig = explicit.Views
			} else {
				viewsPath := findViewsPath(match)
				if viewsPath != "" {
					// Convert to relative path if possible for consistency
					relPath, err := filepath.Rel(".", viewsPath)
					if err == nil {
						viewsPath = "./" + filepath.ToSlash(relPath)
					}
					viewsConfig = &config.ViewsConfig{
						Path:      viewsPath,
						Framework: detectFramework(viewsPath),
					}
				}
			}

			// Determine log level
			resourceLogLevel := b.config.Build.LogLevel
			if explicit != nil && explicit.Build != nil && explicit.Build.LogLevel != "" {
				resourceLogLevel = explicit.Build.LogLevel
			}
			if resourceLogLevel == "" {
				resourceLogLevel = "INFO"
			}

			task := BuildTask{
				Path:         match,
				ResourceName: resourceName,
				Type:         TypeResource,
				OutDir:       filepath.Join(b.config.OutDir, resourceName),
				Options: BuildOptions{
					Server:     buildSideValue(true, b.config.Build.Server),
					Client:     buildSideValue(b.hasClientCode(match), b.config.Build.Client),
					Minify:     b.config.Build.Minify,
					SourceMaps: b.config.Build.SourceMaps,
					LogLevel:   resourceLogLevel,
					Target:     b.config.Build.Target,
					Compile:    true,
				},
			}

			// Apply explicit overrides
			if explicit != nil {
				if explicit.ResourceName != "" {
					task.ResourceName = explicit.ResourceName
				}
				if explicit.CustomCompiler != "" {
					task.CustomCompiler = explicit.CustomCompiler
				}
				if explicit.EntryPoints != nil {
					task.Options.EntryPoints = &EntryPoints{
						Server: explicit.EntryPoints.Server,
						Client: explicit.EntryPoints.Client,
					}
				}
				if explicit.Build != nil {
					if explicit.Build.Server != nil {
						task.Options.Server = buildSideValue(*explicit.Build.Server, b.config.Build.Server)
					}
					if explicit.Build.Client != nil {
						task.Options.Client = buildSideValue(*explicit.Build.Client, b.config.Build.Client)
					}
					if explicit.Build.NUI != nil {
						task.Options.NUI = *explicit.Build.NUI
					}
					if explicit.Build.Minify != nil {
						task.Options.Minify = *explicit.Build.Minify
					}
					if explicit.Build.SourceMaps != nil {
						task.Options.SourceMaps = *explicit.Build.SourceMaps
					}
				}

				// Add views task if configured or discovered
				if viewsConfig != nil {
					// Auto-detect framework if not explicitly set
					framework := viewsConfig.Framework
					if framework == "" {
						framework = detectFramework(viewsConfig.Path)
					}

					tasks = append(tasks, BuildTask{
						Path:           viewsConfig.Path,
						ResourceName:   task.ResourceName + "/ui",
						Type:           TypeViews,
						OutDir:         filepath.Join(b.config.OutDir, task.ResourceName, "ui"),
						CustomCompiler: explicit.CustomCompiler, // Use same compiler for views
						Options: BuildOptions{
							Framework:    framework,
							Minify:       b.config.Build.Minify,
							SourceMaps:   b.config.Build.SourceMaps,
							ViewEntry:    viewsConfig.EntryPoint,
							Ignore:       viewsConfig.Ignore,
							ForceInclude: viewsConfig.ForceInclude,
							LogLevel:     resourceLogLevel,
						},
					})
				}
			} else if viewsConfig != nil {
				// Discovery for non-explicit resources
				tasks = append(tasks, BuildTask{
					Path:         viewsConfig.Path,
					ResourceName: task.ResourceName + "/ui",
					Type:         TypeViews,
					OutDir:       filepath.Join(b.config.OutDir, task.ResourceName, "ui"),
					Options: BuildOptions{
						Framework:    viewsConfig.Framework,
						Minify:       b.config.Build.Minify,
						SourceMaps:   b.config.Build.SourceMaps,
						ForceInclude: viewsConfig.ForceInclude,
						LogLevel:     resourceLogLevel,
					},
				})
			}

			tasks = append(tasks, task)
		}
	}

	// Explicit resources
	for _, res := range b.config.Resources.Explicit {
		// Skip if already added via glob
		alreadyAdded := false
		for _, t := range tasks {
			if t.Path == res.Path {
				alreadyAdded = true
				break
			}
		}
		if alreadyAdded {
			continue
		}

		resourceName := res.ResourceName
		if resourceName == "" {
			resourceName = filepath.Base(res.Path)
		}

		// Determine log level
		resourceLogLevel := b.config.Build.LogLevel
		if res.Build != nil && res.Build.LogLevel != "" {
			resourceLogLevel = res.Build.LogLevel
		}
		if resourceLogLevel == "" {
			resourceLogLevel = "INFO"
		}

		task := BuildTask{
			Path:           res.Path,
			ResourceName:   resourceName,
			Type:           TypeResource,
			OutDir:         filepath.Join(b.config.OutDir, resourceName),
			CustomCompiler: res.CustomCompiler,
			Options: BuildOptions{
				Server:     buildSideValue(true, b.config.Build.Server),
				Client:     buildSideValue(true, b.config.Build.Client),
				Minify:     b.config.Build.Minify,
				SourceMaps: b.config.Build.SourceMaps,
				LogLevel:   resourceLogLevel,
				Target:     b.config.Build.Target,
				Compile:    true,
			},
		}

		// Apply entryPoints if configured
		if res.EntryPoints != nil {
			task.Options.EntryPoints = &EntryPoints{
				Server: res.EntryPoints.Server,
				Client: res.EntryPoints.Client,
			}
		}

		if res.Build != nil {
			if res.Build.Server != nil {
				task.Options.Server = buildSideValue(*res.Build.Server, b.config.Build.Server)
			}
			if res.Build.Client != nil {
				task.Options.Client = buildSideValue(*res.Build.Client, b.config.Build.Client)
			}
			if res.Build.NUI != nil {
				task.Options.NUI = *res.Build.NUI
			}
		}

		// Resources are always compiled, so we can always check for views
		var viewsConfig *config.ViewsConfig
		if res.Views != nil {
			viewsConfig = res.Views
		} else {
			viewsPath := findViewsPath(res.Path)
			if viewsPath != "" {
				relPath, err := filepath.Rel(".", viewsPath)
				if err == nil {
					viewsPath = "./" + filepath.ToSlash(relPath)
				}
				viewsConfig = &config.ViewsConfig{
					Path:      viewsPath,
					Framework: detectFramework(viewsPath),
				}
			}
		}

		tasks = append(tasks, task)

		// Add views task if configured or discovered
		if viewsConfig != nil {
			// Auto-detect framework if not explicitly set
			framework := viewsConfig.Framework
			if framework == "" {
				framework = detectFramework(viewsConfig.Path)
			}

			tasks = append(tasks, BuildTask{
				Path:           viewsConfig.Path,
				ResourceName:   resourceName + "/ui",
				Type:           TypeViews,
				OutDir:         filepath.Join(b.config.OutDir, resourceName, "ui"),
				CustomCompiler: res.CustomCompiler,
				Options: BuildOptions{
					Framework:    framework,
					Minify:       b.config.Build.Minify,
					SourceMaps:   b.config.Build.SourceMaps,
					ViewEntry:    viewsConfig.EntryPoint,
					Ignore:       viewsConfig.Ignore,
					ForceInclude: viewsConfig.ForceInclude,
					LogLevel:     resourceLogLevel,
				},
			})
		}
	}

	// Standalone resources
	if b.config.Standalones != nil {
		// From glob patterns
		for _, pattern := range b.config.Standalones.Include {
			matches, _ := filepath.Glob(pattern)
			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil || !info.IsDir() {
					continue
				}

				resourceName := filepath.Base(match)
				shouldCompile := b.config.ShouldCompile(match)

				// Check for explicit override to get CustomCompiler and EntryPoints
				explicit := b.config.GetExplicitStandalone(match)
				customCompiler := ""
				var entryPoints *EntryPoints
				if explicit != nil {
					customCompiler = explicit.CustomCompiler
					if explicit.EntryPoints != nil {
						entryPoints = &EntryPoints{
							Server: explicit.EntryPoints.Server,
							Client: explicit.EntryPoints.Client,
						}
					}
				}

				taskType := TypeStandalone
				if !shouldCompile {
					taskType = TypeCopy
				}

				// Determine log level
				standaloneLogLevel := b.config.Build.LogLevel
				if explicit != nil && explicit.Build != nil && explicit.Build.LogLevel != "" {
					standaloneLogLevel = explicit.Build.LogLevel
				}
				if standaloneLogLevel == "" {
					standaloneLogLevel = "INFO"
				}

				tasks = append(tasks, BuildTask{
					Path:           match,
					ResourceName:   resourceName,
					Type:           taskType,
					OutDir:         filepath.Join(b.config.OutDir, resourceName),
					CustomCompiler: customCompiler,
					Options: BuildOptions{
						Server:      buildSideValue(true, b.config.Build.Server),
						Client:      buildSideValue(b.hasClientCode(match), b.config.Build.Client),
						Minify:      b.config.Build.Minify,
						SourceMaps:  b.config.Build.SourceMaps,
						LogLevel:    standaloneLogLevel,
						Target:      b.config.Build.Target,
						Compile:     shouldCompile,
						EntryPoints: entryPoints,
					},
				})
			}
		}

		// Explicit standalone
		for _, res := range b.config.Standalones.Explicit {
			resourceName := res.ResourceName
			if resourceName == "" {
				resourceName = filepath.Base(res.Path)
			}

			shouldCompile := true
			if res.Compile != nil {
				shouldCompile = *res.Compile
			}

			taskType := TypeStandalone
			if !shouldCompile {
				taskType = TypeCopy
			}

			// Determine log level
			standaloneLogLevel := b.config.Build.LogLevel
			if res.Build != nil && res.Build.LogLevel != "" {
				standaloneLogLevel = res.Build.LogLevel
			}
			if standaloneLogLevel == "" {
				standaloneLogLevel = "INFO"
			}

			task := BuildTask{
				Path:           res.Path,
				ResourceName:   resourceName,
				Type:           taskType,
				OutDir:         filepath.Join(b.config.OutDir, resourceName),
				CustomCompiler: res.CustomCompiler,
				Options: BuildOptions{
					Server:     buildSideValue(true, b.config.Build.Server),
					Client:     buildSideValue(b.hasClientCode(res.Path), b.config.Build.Client),
					Minify:     b.config.Build.Minify,
					SourceMaps: b.config.Build.SourceMaps,
					LogLevel:   standaloneLogLevel,
					Target:     b.config.Build.Target,
					Compile:    shouldCompile,
				},
			}

			// Apply entryPoints if configured
			if res.EntryPoints != nil {
				task.Options.EntryPoints = &EntryPoints{
					Server: res.EntryPoints.Server,
					Client: res.EntryPoints.Client,
				}
			}

			tasks = append(tasks, task)

			// Add views task if configured
			if res.Views != nil {
				tasks = append(tasks, BuildTask{
					Path:           res.Views.Path,
					ResourceName:   resourceName + "/ui",
					Type:           TypeViews,
					OutDir:         filepath.Join(b.config.OutDir, resourceName, "ui"),
					CustomCompiler: res.CustomCompiler,
					Options: BuildOptions{
						Framework:    res.Views.Framework,
						Minify:       b.config.Build.Minify,
						SourceMaps:   b.config.Build.SourceMaps,
						ViewEntry:    res.Views.EntryPoint,
						Ignore:       res.Views.Ignore,
						ForceInclude: res.Views.ForceInclude,
					},
				})
			}
		}
	}

	return tasks
}

// buildParallel executes builds in parallel using worker pool with TUI
func (b *Builder) buildParallel(tasks []BuildTask, workers int) ([]BuildResult, error) {
	pool := NewWorkerPool(workers)
	pool.Start(b.resourceBuilder.Build)

	// Submit all tasks
	pool.SubmitAll(tasks)

	// Run TUI
	m := newBuildModel(tasks, pool.Results())
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		pool.Cancel()
		return nil, err
	}

	pool.Close()

	model := finalModel.(buildModel)

	// Check for failures
	failCount := 0
	for _, r := range model.results {
		if !r.Success {
			failCount++
		}
	}

	if failCount > 0 {
		return model.results, fmt.Errorf("%d resource(s) failed to build", failCount)
	}

	return model.results, nil
}

// buildSequential executes builds one by one
func (b *Builder) buildSequential(tasks []BuildTask) ([]BuildResult, error) {
	results := make([]BuildResult, 0, len(tasks))

	for _, task := range tasks {
		fmt.Printf("%s Building %s...\n", ui.Info("→"), task.ResourceName)

		result := b.resourceBuilder.Build(task)
		results = append(results, result)

		if result.Success {
			fmt.Println(ui.Success(fmt.Sprintf("[%s] compiled (%s)", task.ResourceName, result.Duration.Round(time.Millisecond))))
		} else {
			fmt.Println(ui.Error(fmt.Sprintf("[%s] failed: %v", task.ResourceName, result.Error)))
			if result.Output != "" {
				fmt.Println(ui.Muted("Build output:"))
				fmt.Println(result.Output)
			}
			return results, fmt.Errorf("build failed for %s", task.ResourceName)
		}
	}

	return results, nil
}

// hasClientCode checks if a resource has client code
func (b *Builder) hasClientCode(resourcePath string) bool {
	patterns := []string{
		"src/client.ts",
		"src/client/main.ts",
		"src/client/index.ts",
	}

	for _, pattern := range patterns {
		if _, err := os.Stat(filepath.Join(resourcePath, pattern)); err == nil {
			return true
		}
	}

	// Also check if src/client is a directory (legacy/fallback)
	clientDir := filepath.Join(resourcePath, "src", "client")
	if info, err := os.Stat(clientDir); err == nil && info.IsDir() {
		return true
	}

	return false
}

func (b *Builder) cleanResourceOutputDir(resourceName string) error {
	outDir := b.config.OutDir
	if outDir == "" {
		outDir = "./build"
	}

	resourceDir := filepath.Join(outDir, resourceName)
	if _, err := os.Stat(resourceDir); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(resourceDir); err != nil {
		return err
	}

	return nil
}

// ResourceSize holds size information for a compiled resource
type ResourceSize struct {
	Name       string
	ServerSize int64
	ClientSize int64
	TotalSize  int64
	IsViews    bool   // true if this is a views/UI bundle
	Framework  string // framework used (react, vue, svelte, vanilla) - only for views
}

// detectFramework detects the framework used in a views directory
func detectFramework(viewPath string) string {
	// Check for framework-specific files
	hasReact := false
	hasVue := false
	hasSvelte := false

	_ = filepath.WalkDir(viewPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		switch ext {
		case ".tsx", ".jsx":
			hasReact = true
		case ".vue":
			hasVue = true
		case ".svelte":
			hasSvelte = true
		}
		return nil
	})

	// Return detected framework (prioritize by specificity)
	if hasSvelte {
		return "svelte"
	}
	if hasVue {
		return "vue"
	}
	if hasReact {
		return "react"
	}
	return "vanilla"
}

// formatSize formats bytes into human readable format (KB/MB)
func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	}
}

// getDirSize calculates total size of all files in a directory recursively
func getDirSize(dirPath string) int64 {
	var total int64
	_ = filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

// getResourceSizes calculates the size of compiled files for each resource
func (b *Builder) getResourceSizes(results []BuildResult) []ResourceSize {
	var sizes []ResourceSize
	seen := make(map[string]bool)

	for _, r := range results {
		if !r.Success {
			continue
		}

		resourceName := r.Task.ResourceName
		if seen[resourceName] {
			continue
		}
		seen[resourceName] = true

		resourceDir := filepath.Join(b.config.OutDir, resourceName)

		// Handle views separately - calculate total directory size
		if r.Task.Type == TypeViews {
			totalSize := getDirSize(resourceDir)
			if totalSize > 0 {
				// Detect framework from source path
				framework := detectFramework(r.Task.Path)
				sizes = append(sizes, ResourceSize{
					Name:      resourceName,
					TotalSize: totalSize,
					IsViews:   true,
					Framework: framework,
				})
			}
			continue
		}

		var serverSize, clientSize int64

		// Check server.js
		serverPath := filepath.Join(resourceDir, "server.js")
		if info, err := os.Stat(serverPath); err == nil {
			serverSize = info.Size()
		}

		// Check client.js
		clientPath := filepath.Join(resourceDir, "client.js")
		if info, err := os.Stat(clientPath); err == nil {
			clientSize = info.Size()
		}

		if serverSize > 0 || clientSize > 0 {
			sizes = append(sizes, ResourceSize{
				Name:       resourceName,
				ServerSize: serverSize,
				ClientSize: clientSize,
				TotalSize:  serverSize + clientSize,
			})
		}
	}

	// Sort sizes to group related resources together (core, core/ui, xchat, xchat/ui, etc.)
	sort.Slice(sizes, func(i, j int) bool {
		// Extract base names (before any /)
		baseI := strings.Split(sizes[i].Name, "/")[0]
		baseJ := strings.Split(sizes[j].Name, "/")[0]

		// If different base names, sort alphabetically by base
		if baseI != baseJ {
			return baseI < baseJ
		}

		// Same base: main resource comes before sub-resources (e.g., core before core/ui)
		return len(sizes[i].Name) < len(sizes[j].Name)
	})

	return sizes
}

// showSummary displays the build summary
func (b *Builder) showSummary(results []BuildResult) {
	// Count unique resources, standalones, and UIs separately
	successResources := make(map[string]struct{})
	successStandalones := make(map[string]struct{})
	successUIs := make(map[string]struct{})
	failedResources := make(map[string]struct{})
	failedStandalones := make(map[string]struct{})
	totalDuration := time.Duration(0)

	for _, r := range results {
		baseResource := strings.Split(r.Task.ResourceName, "/")[0]
		isStandalone := r.Task.Type == TypeStandalone || r.Task.Type == TypeCopy

		if r.Success {
			if isStandalone {
				successStandalones[baseResource] = struct{}{}
			} else if r.Task.Type == TypeViews {
				successUIs[r.Task.ResourceName] = struct{}{}
			} else {
				successResources[baseResource] = struct{}{}
			}
			totalDuration += r.Duration
		} else {
			if isStandalone {
				failedStandalones[baseResource] = struct{}{}
			} else if r.Task.Type != TypeViews {
				failedResources[baseResource] = struct{}{}
			}
		}
	}

	successResourceCount := len(successResources)
	successStandaloneCount := len(successStandalones)
	successUICount := len(successUIs)
	failResourceCount := len(failedResources)
	failStandaloneCount := len(failedStandalones)
	failCount := failResourceCount + failStandaloneCount

	fmt.Println()

	// Get resource sizes
	sizes := b.getResourceSizes(results)
	var grandTotal int64
	for _, s := range sizes {
		grandTotal += s.TotalSize
	}

	if failCount == 0 {
		var boxContent strings.Builder
		boxContent.WriteString("Build completed successfully!\n\n")

		// Show counts based on what's present
		var countParts []string
		if successResourceCount > 0 {
			countParts = append(countParts, fmt.Sprintf("Resources: %d", successResourceCount))
		}
		if successUICount > 0 {
			countParts = append(countParts, fmt.Sprintf("UIs: %d", successUICount))
		}
		if successStandaloneCount > 0 {
			countParts = append(countParts, fmt.Sprintf("Standalones: %d", successStandaloneCount))
		}
		if len(countParts) > 0 {
			boxContent.WriteString(strings.Join(countParts, " | ") + "\n")
		}

		boxContent.WriteString(fmt.Sprintf("Time: %s\n", totalDuration.Round(time.Millisecond)))

		if b.deployer.HasDestination() {
			boxContent.WriteString(fmt.Sprintf("Deployed: %s\n", b.config.Destination))
		}

		// Add bundle sizes
		if len(sizes) > 0 {
			boxContent.WriteString("\n")
			boxContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")).Render("--- Bundle Sizes ---"))
			boxContent.WriteString("\n")
			for _, s := range sizes {
				nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
				if s.IsViews {
					// Views show only total size (includes JS, CSS, HTML, assets) + framework
					totalStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#E879F9")).Render(formatSize(s.TotalSize))
					frameworkStr := ""
					if s.Framework != "" {
						frameworkStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(fmt.Sprintf(" (%s)", s.Framework))
					}
					boxContent.WriteString(fmt.Sprintf("%s  Total: %s%s\n",
						nameStyle.Render(fmt.Sprintf("%-14s", s.Name)), totalStr, frameworkStr))
				} else {
					serverStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("-")
					clientStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("-")
					if s.ServerSize > 0 {
						serverStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Render(formatSize(s.ServerSize))
					}
					if s.ClientSize > 0 {
						clientStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Render(formatSize(s.ClientSize))
					}
					boxContent.WriteString(fmt.Sprintf("%s  Server: %s  Client: %s\n",
						nameStyle.Render(fmt.Sprintf("%-14s", s.Name)), serverStr, clientStr))
				}
			}
			totalStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B"))
			boxContent.WriteString(fmt.Sprintf("\nTotal: %s", totalStyle.Render(formatSize(grandTotal))))
		}

		fmt.Println(ui.SuccessBoxStyle.Render(boxContent.String()))
	} else {
		var boxContent strings.Builder
		boxContent.WriteString("Build completed with errors\n\n")

		// Show success counts
		if successResourceCount > 0 || successStandaloneCount > 0 || successUICount > 0 {
			var successParts []string
			if successResourceCount > 0 {
				successParts = append(successParts, fmt.Sprintf("Resources: %d", successResourceCount))
			}
			if successUICount > 0 {
				successParts = append(successParts, fmt.Sprintf("UIs: %d", successUICount))
			}
			if successStandaloneCount > 0 {
				successParts = append(successParts, fmt.Sprintf("Standalones: %d", successStandaloneCount))
			}
			boxContent.WriteString(fmt.Sprintf("✓ Success: %s\n", strings.Join(successParts, " | ")))
		}

		// Show fail counts
		if failResourceCount > 0 && failStandaloneCount > 0 {
			boxContent.WriteString(fmt.Sprintf("✗ Failed: Resources: %d | Standalones: %d", failResourceCount, failStandaloneCount))
		} else if failResourceCount > 0 {
			boxContent.WriteString(fmt.Sprintf("✗ Failed: Resources: %d", failResourceCount))
		} else if failStandaloneCount > 0 {
			boxContent.WriteString(fmt.Sprintf("✗ Failed: Standalones: %d", failStandaloneCount))
		}

		fmt.Println(ui.ErrorBoxStyle.Render(boxContent.String()))
	}
}

// ============================================================================
// BubbleTea Model for Parallel Build TUI
// ============================================================================

type taskStatus int

const (
	statusPending taskStatus = iota
	statusBuilding
	statusDone
	statusFailed
)

type taskState struct {
	task   BuildTask
	status taskStatus
	result *BuildResult
}

type buildModel struct {
	spinner     spinner.Model
	progress    progress.Model
	tasks       []taskState
	results     []BuildResult
	resultsChan <-chan BuildResult
	completed   int
	total       int
	done        bool
}

type resultMsg BuildResult

func newBuildModel(tasks []BuildTask, resultsChan <-chan BuildResult) buildModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	p := progress.New(progress.WithDefaultGradient())

	taskStates := make([]taskState, len(tasks))
	for i, t := range tasks {
		taskStates[i] = taskState{
			task:   t,
			status: statusPending,
		}
	}

	return buildModel{
		spinner:     s,
		progress:    p,
		tasks:       taskStates,
		results:     []BuildResult{},
		resultsChan: resultsChan,
		total:       len(tasks),
	}
}

func (m buildModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForResult(m.resultsChan),
	)
}

func waitForResult(ch <-chan BuildResult) tea.Cmd {
	return func() tea.Msg {
		result, ok := <-ch
		if !ok {
			return nil
		}
		return resultMsg(result)
	}
}

func (m buildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultMsg:
		result := BuildResult(msg)
		m.results = append(m.results, result)
		m.completed++

		// Update task state
		for i := range m.tasks {
			if m.tasks[i].task.Path == result.Task.Path &&
				m.tasks[i].task.Type == result.Task.Type {
				if result.Success {
					m.tasks[i].status = statusDone
				} else {
					m.tasks[i].status = statusFailed
				}
				m.tasks[i].result = &result
				break
			}
		}

		// Mark next pending tasks as building
		buildingCount := 0
		for i := range m.tasks {
			if m.tasks[i].status == statusBuilding {
				buildingCount++
			}
		}
		for i := range m.tasks {
			if m.tasks[i].status == statusPending && buildingCount < 4 {
				m.tasks[i].status = statusBuilding
				buildingCount++
			}
		}

		if m.completed >= m.total {
			m.done = true
			return m, tea.Quit
		}

		return m, waitForResult(m.resultsChan)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m buildModel) View() string {
	if m.done {
		return m.renderFinal()
	}

	var b strings.Builder

	b.WriteString(ui.TitleStyle.Render("Building Resources"))
	b.WriteString(fmt.Sprintf(" (%d/%d)\n\n", m.completed, m.total))

	// Show tasks
	for _, ts := range m.tasks {
		switch ts.status {
		case statusPending:
			b.WriteString(fmt.Sprintf("  ○ %s\n", ui.Muted(ts.task.ResourceName)))
		case statusBuilding:
			b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), ts.task.ResourceName))
		case statusDone:
			duration := ""
			if ts.result != nil {
				duration = fmt.Sprintf(" (%s)", ts.result.Duration.Round(time.Millisecond))
			}
			b.WriteString(fmt.Sprintf("  %s %s%s\n", ui.Success("✓"), ts.task.ResourceName, ui.Muted(duration)))
		case statusFailed:
			b.WriteString(fmt.Sprintf("  %s %s\n", ui.Error("✗"), ts.task.ResourceName))
		}
	}

	// Progress bar
	b.WriteString("\n")
	prog := float64(m.completed) / float64(m.total)
	b.WriteString(m.progress.ViewAs(prog))
	b.WriteString("\n")

	return b.String()
}

func (m buildModel) renderFinal() string {
	var b strings.Builder

	b.WriteString("\n")

	for _, ts := range m.tasks {
		switch ts.status {
		case statusDone:
			duration := ""
			if ts.result != nil {
				duration = fmt.Sprintf(" (%s)", ts.result.Duration.Round(time.Millisecond))
			}
			b.WriteString(fmt.Sprintf("%s [%s] compiled%s\n", ui.Success("✓"), ts.task.ResourceName, ui.Muted(duration)))
		case statusFailed:
			errMsg := ""
			if ts.result != nil && ts.result.Error != nil {
				errMsg = fmt.Sprintf(": %v", ts.result.Error)
			}
			b.WriteString(fmt.Sprintf("%s [%s] failed%s\n", ui.Error("✗"), ts.task.ResourceName, errMsg))
			// Show build output for failed tasks (contains detailed error messages)
			if ts.result != nil && ts.result.Output != "" {
				b.WriteString(ui.Muted("Build output:\n"))
				b.WriteString(ts.result.Output)
				b.WriteString("\n")
			}
		default:
			b.WriteString(fmt.Sprintf("○ [%s] skipped\n", ts.task.ResourceName))
		}
	}

	return b.String()
}
