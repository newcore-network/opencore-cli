package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	fmt.Println(ui.Logo())

	// Cleanup embedded script on exit
	defer b.resourceBuilder.Cleanup()

	// Clean output directory before build
	if err := b.cleanOutputDir(); err != nil {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	// Collect all build tasks
	tasks := b.collectAllTasks()

	if len(tasks) == 0 {
		return fmt.Errorf("no resources to build")
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

	// Deploy to destination if configured
	if b.deployer.HasDestination() {
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

	if b.deployer.HasDestination() {
		for baseResource := range uniqueResources {
			if err := b.deployer.DeployResource(baseResource); err != nil {
				return results, fmt.Errorf("deployment failed for %s: %w", baseResource, err)
			}
		}
	}

	return results, nil
}

// collectAllTasks gathers all build tasks from config
func (b *Builder) collectAllTasks() []BuildTask {
	var tasks []BuildTask

	// Core task
	coreTask := BuildTask{
		Path:           b.config.Core.Path,
		ResourceName:   b.config.Core.ResourceName,
		Type:           TypeCore,
		OutDir:         b.config.OutDir,
		CustomCompiler: b.config.Core.CustomCompiler,
		Options: BuildOptions{
			Server:     true,
			Client:     true,
			Minify:     b.config.Build.Minify,
			SourceMaps: b.config.Build.SourceMaps,
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
				Framework:  b.config.Core.Views.Framework,
				Minify:     b.config.Build.Minify,
				SourceMaps: b.config.Build.SourceMaps,
				ViewEntry:  b.config.Core.Views.EntryPoint,
				Ignore:     b.config.Core.Views.Ignore,
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

			task := BuildTask{
				Path:         match,
				ResourceName: resourceName,
				Type:         TypeResource,
				OutDir:       b.config.OutDir,
				Options: BuildOptions{
					Server:     true,
					Client:     b.hasClientCode(match),
					Minify:     b.config.Build.Minify,
					SourceMaps: b.config.Build.SourceMaps,
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
						task.Options.Server = *explicit.Build.Server
					}
					if explicit.Build.Client != nil {
						task.Options.Client = *explicit.Build.Client
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

				// Add views task if configured
				if explicit.Views != nil {
					tasks = append(tasks, BuildTask{
						Path:           explicit.Views.Path,
						ResourceName:   task.ResourceName + "/ui",
						Type:           TypeViews,
						OutDir:         filepath.Join(b.config.OutDir, task.ResourceName, "ui"),
						CustomCompiler: explicit.CustomCompiler, // Use same compiler for views
						Options: BuildOptions{
							Framework:  explicit.Views.Framework,
							Minify:     b.config.Build.Minify,
							SourceMaps: b.config.Build.SourceMaps,
							ViewEntry:  explicit.Views.EntryPoint,
							Ignore:     explicit.Views.Ignore,
						},
					})
				}
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

		task := BuildTask{
			Path:           res.Path,
			ResourceName:   resourceName,
			Type:           TypeResource,
			OutDir:         b.config.OutDir,
			CustomCompiler: res.CustomCompiler,
			Options: BuildOptions{
				Server:     true,
				Client:     true,
				Minify:     b.config.Build.Minify,
				SourceMaps: b.config.Build.SourceMaps,
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
				task.Options.Server = *res.Build.Server
			}
			if res.Build.Client != nil {
				task.Options.Client = *res.Build.Client
			}
			if res.Build.NUI != nil {
				task.Options.NUI = *res.Build.NUI
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
					Framework:  res.Views.Framework,
					Minify:     b.config.Build.Minify,
					SourceMaps: b.config.Build.SourceMaps,
					ViewEntry:  res.Views.EntryPoint,
					Ignore:     res.Views.Ignore,
				},
			})
		}
	}

	// Standalone resources
	if b.config.Standalone != nil {
		// From glob patterns
		for _, pattern := range b.config.Standalone.Include {
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

				tasks = append(tasks, BuildTask{
					Path:           match,
					ResourceName:   resourceName,
					Type:           taskType,
					OutDir:         b.config.OutDir,
					CustomCompiler: customCompiler,
					Options: BuildOptions{
						Server:      true,
						Client:      b.hasClientCode(match),
						Minify:      b.config.Build.Minify,
						SourceMaps:  b.config.Build.SourceMaps,
						Target:      b.config.Build.Target,
						Compile:     shouldCompile,
						EntryPoints: entryPoints,
					},
				})
			}
		}

		// Explicit standalone
		for _, res := range b.config.Standalone.Explicit {
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

			task := BuildTask{
				Path:           res.Path,
				ResourceName:   resourceName,
				Type:           taskType,
				OutDir:         b.config.OutDir,
				CustomCompiler: res.CustomCompiler,
				Options: BuildOptions{
					Server:     true,
					Client:     b.hasClientCode(res.Path),
					Minify:     b.config.Build.Minify,
					SourceMaps: b.config.Build.SourceMaps,
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
						Framework:  res.Views.Framework,
						Minify:     b.config.Build.Minify,
						SourceMaps: b.config.Build.SourceMaps,
						ViewEntry:  res.Views.EntryPoint,
						Ignore:     res.Views.Ignore,
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
	clientPath := filepath.Join(resourcePath, "src", "client")
	info, err := os.Stat(clientPath)
	return err == nil && info.IsDir()
}

// cleanOutputDir removes the output directory before building
func (b *Builder) cleanOutputDir() error {
	outDir := b.config.OutDir
	if outDir == "" {
		outDir = "./build"
	}

	// Check if directory exists
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return nil // Nothing to clean
	}

	fmt.Printf("%s Cleaning %s...\n", ui.Info("→"), outDir)
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}

	return nil
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

// getResourceSizes calculates the size of compiled files for each resource
func (b *Builder) getResourceSizes(results []BuildResult) []ResourceSize {
	var sizes []ResourceSize
	seen := make(map[string]bool)

	for _, r := range results {
		if !r.Success {
			continue
		}

		resourceName := r.Task.ResourceName
		// Skip if already processed (e.g., views are part of a resource)
		if seen[resourceName] || strings.HasSuffix(resourceName, "/ui") {
			continue
		}
		seen[resourceName] = true

		resourceDir := filepath.Join(b.config.OutDir, resourceName)

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

	return sizes
}

// showSummary displays the build summary
func (b *Builder) showSummary(results []BuildResult) {
	// Count unique resources, not build tasks (a resource can have multiple tasks)
	successResources := make(map[string]struct{})
	failedResources := make(map[string]struct{})
	totalDuration := time.Duration(0)

	for _, r := range results {
		baseResource := strings.Split(r.Task.ResourceName, "/")[0]
		if r.Success {
			successResources[baseResource] = struct{}{}
			totalDuration += r.Duration
		} else {
			failedResources[baseResource] = struct{}{}
		}
	}

	successCount := len(successResources)
	failCount := len(failedResources)

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
		boxContent.WriteString(fmt.Sprintf("Resources: %d\n", successCount))
		boxContent.WriteString(fmt.Sprintf("Time: %s\n", totalDuration.Round(time.Millisecond)))
		boxContent.WriteString(fmt.Sprintf("Output: %s\n", b.config.OutDir))

		if b.deployer.HasDestination() {
			boxContent.WriteString(fmt.Sprintf("Deployed: %s\n", b.config.Destination))
		}

		// Add bundle sizes
		if len(sizes) > 0 {
			boxContent.WriteString("\n")
			boxContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")).Render("--- Bundle Sizes ---"))
			boxContent.WriteString("\n")
			for _, s := range sizes {
				serverStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("-")
				clientStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("-")
				if s.ServerSize > 0 {
					serverStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Render(formatSize(s.ServerSize))
				}
				if s.ClientSize > 0 {
					clientStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Render(formatSize(s.ClientSize))
				}
				nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
				boxContent.WriteString(fmt.Sprintf("%s  Server: %s  Client: %s\n",
					nameStyle.Render(fmt.Sprintf("%-14s", s.Name)), serverStr, clientStr))
			}
			totalStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B"))
			boxContent.WriteString(fmt.Sprintf("\nTotal: %s", totalStyle.Render(formatSize(grandTotal))))
		}

		fmt.Println(ui.SuccessBoxStyle.Render(boxContent.String()))
	} else {
		boxContent := fmt.Sprintf(
			"Build completed with errors\n\n"+
				"Success: %d\n"+
				"Failed: %d",
			successCount,
			failCount,
		)
		fmt.Println(ui.ErrorBoxStyle.Render(boxContent))
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
		default:
			b.WriteString(fmt.Sprintf("○ [%s] skipped\n", ts.task.ResourceName))
		}
	}

	return b.String()
}
