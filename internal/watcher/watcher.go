package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/newcore-network/opencore-cli/internal/builder"
	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

type Watcher struct {
	config         *config.Config
	builder        *builder.Builder
	watcher        *fsnotify.Watcher
	debounceTimers map[string]*time.Timer
	restarter      restarter
	logQueue       chan LogMessage
	buildingMutex  sync.Mutex
	buildingSet    map[string]bool // Track which resources are currently being built
}

func New(cfg *config.Config) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		config:         cfg,
		builder:        builder.New(cfg),
		watcher:        w,
		debounceTimers: make(map[string]*time.Timer),
		logQueue:       make(chan LogMessage, 256),
		buildingSet:    make(map[string]bool),
	}

	restarter, err := newRestarter(cfg)
	if err != nil {
		return nil, err
	}
	watcher.restarter = restarter

	return watcher, nil
}

func (w *Watcher) Watch(ctx context.Context) error {
	allTasks := w.builder.CollectTasks()

	// Watch config file for dynamic updates
	configPath := "opencore.config.ts"
	if _, err := os.Stat(configPath); err == nil {
		if err := w.watcher.Add(configPath); err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to watch %s: %v", configPath, err)))
		} else {
			fmt.Println(ui.Info(fmt.Sprintf("Watching configuration: %s", configPath)))
		}
	}

	// Add paths to watch recursively
	w.registerPaths()

	if err := w.startBridgeServer(ctx); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Failed to start dev bridge: %v", err)))
	}
	go w.startLogPrinter(ctx)

	fmt.Println()

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#4F46E5")).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	// Count unique resources and standalones separately
	uniqueResources := make(map[string]struct{})
	uniqueStandalones := make(map[string]struct{})
	for _, task := range allTasks {
		baseResource := strings.Split(task.ResourceName, "/")[0]
		// Skip counting views separately (they're part of a resource)
		if strings.HasSuffix(task.ResourceName, "/ui") {
			continue
		}
		if task.Type == "standalone" || task.Type == "copy" {
			uniqueStandalones[baseResource] = struct{}{}
		} else {
			uniqueResources[baseResource] = struct{}{}
		}
	}

	// Build status string
	var statusParts []string
	statusParts = append(statusParts, fmt.Sprintf("Project: %s", w.config.Name))
	if len(uniqueResources) > 0 && len(uniqueStandalones) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Resources: %d | Standalones: %d", len(uniqueResources), len(uniqueStandalones)))
	} else if len(uniqueResources) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Resources: %d", len(uniqueResources)))
	} else if len(uniqueStandalones) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Standalones: %d", len(uniqueStandalones)))
	}

	fmt.Printf("%s %s\n",
		headerStyle.Render(" DEV MODE "),
		statusStyle.Render(strings.Join(statusParts, " | ")))

	fmt.Println(ui.Muted(fmt.Sprintf("Bridge: http://localhost:%d/logs", w.config.Dev.BridgePort())))
	switch w.restarter.Mode() {
	case "txadmin":
		fmt.Println(ui.Info(fmt.Sprintf("Restart mode: txAdmin (%s)", w.config.Dev.TxAdmin.URL)))
		fmt.Println(ui.Muted("Authenticating with txAdmin..."))
		if err := w.restarter.Start(ctx); err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("txAdmin login failed: %v", err)))
			fmt.Println(ui.Warning("Automatic restarts via txAdmin are disabled until credentials work."))
			w.restarter = &noopRestarter{}
		} else {
			fmt.Println(ui.Success("Connected to txAdmin"))
		}
	case "process":
		fmt.Println(ui.Info(fmt.Sprintf("Restart mode: managed process (%s)", w.config.Dev.Process.Command)))
	case "none":
		fmt.Println(ui.Muted("Restart mode: build only"))
	}

	fmt.Println(ui.Muted("Watching for changes... (Ctrl+C to stop)"))
	fmt.Println()

	// Build once at start
	if err := w.builder.BuildWithOutputContext(ctx, builder.OutputModeAuto); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Initial build failed: %v", err)))
	} else if err := w.restarter.Start(ctx); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to start dev runtime: %v", err)))
	}

	// Watch for changes
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				// Debounce using timer - wait for 500ms of silence before processing
				fileName := event.Name

				// Cancel existing timer for this file if any
				if timer, exists := w.debounceTimers[fileName]; exists {
					timer.Stop()
				}

				// Create new timer that will execute after 500ms of silence
				w.debounceTimers[fileName] = time.AfterFunc(500*time.Millisecond, func() {
					if ctx.Err() != nil {
						delete(w.debounceTimers, fileName)
						return
					}
					// Handle config file change
					if filepath.Base(fileName) == "opencore.config.ts" {
						fmt.Println(ui.Info("Configuration changed, reloading..."))
						newCfg, root, err := config.LoadWithProjectRoot()
						if err != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Failed to reload config: %v", err)))
							return
						}
						if err := os.Chdir(root); err != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Failed to switch to project root: %v", err)))
							return
						}
						w.config = newCfg
						w.builder = builder.New(newCfg)
						newRestarter, restarterErr := newRestarter(newCfg)
						if restarterErr != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Failed to configure restart mode: %v", restarterErr)))
							return
						}
						if w.restarter != nil {
							_ = w.restarter.Stop()
						}
						w.restarter = newRestarter
						allTasks = w.builder.CollectTasks()

						// Re-add all paths (fsnotify handles duplicates)
						w.registerPaths()

						fmt.Println(ui.Info("Config reloaded, triggering full build..."))
						if err := w.builder.BuildWithOutputContext(ctx, builder.OutputModeAuto); err != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", err)))
						} else if err := w.restarter.Start(ctx); err != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Failed to start dev runtime: %v", err)))
						}
						return
					}

					affected := w.tasksForChangedFile(allTasks, fileName)
					if len(affected) == 0 {
						fmt.Println(ui.Muted(fmt.Sprintf("File changed (ignored): %s", filepath.Base(fileName))))
						return
					}

					// Get unique base resources from affected tasks
					affectedResources := make(map[string]bool)
					for _, task := range affected {
						baseResource := strings.Split(task.ResourceName, "/")[0]
						affectedResources[baseResource] = true
					}

					// Check if any of these resources are already being built
					w.buildingMutex.Lock()
					shouldSkip := false
					for resource := range affectedResources {
						if w.buildingSet[resource] {
							shouldSkip = true
							break
						}
					}

					if shouldSkip {
						w.buildingMutex.Unlock()
						fmt.Println(ui.Muted(fmt.Sprintf("Build already in progress for %s, skipping...", filepath.Base(fileName))))
						delete(w.debounceTimers, fileName)
						return
					}

					// Mark all affected resources as being built
					for resource := range affectedResources {
						w.buildingSet[resource] = true
					}
					w.buildingMutex.Unlock()

					// Perform the build
					fmt.Println(ui.Info(fmt.Sprintf("File changed: %s", filepath.Base(fileName))))
					results, err := w.builder.BuildTasksContext(ctx, affected)

					// Unmark resources as being built
					w.buildingMutex.Lock()
					for resource := range affectedResources {
						delete(w.buildingSet, resource)
					}
					w.buildingMutex.Unlock()

					if err != nil {
						fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", err)))
						delete(w.debounceTimers, fileName)
						return
					}

					// Notify framework for hot reload
					w.notifyFramework(results)

					// Clean up timer reference
					delete(w.debounceTimers, fileName)
				})
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					// Automatically watch new directories
					w.watcher.Add(event.Name)

					// Re-collect tasks to include new resource if it matches globs
					newCfg, root, _ := config.LoadWithProjectRoot()
					if newCfg != nil {
						_ = os.Chdir(root)
						w.config = newCfg
						w.builder = builder.New(newCfg)
						allTasks = w.builder.CollectTasks()
					}
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Println(ui.Error(fmt.Sprintf("Watcher error: %v", err)))
		}
	}
}

// registerPaths adds all source directories to the watcher
func (w *Watcher) registerPaths() {
	// 1. Watch the project root for config changes (already added in Watch())

	// 2. Watch glob parent directories to detect new resources
	for _, pattern := range w.config.Resources.Include {
		parent := filepath.Dir(pattern)
		if info, err := os.Stat(parent); err == nil && info.IsDir() {
			if err := w.watcher.Add(parent); err == nil {
				fmt.Println(ui.Muted(fmt.Sprintf("Watching directory for new resources: %s", parent)))
			}
		}
	}
	if w.config.Standalones != nil {
		for _, pattern := range w.config.Standalones.Include {
			parent := filepath.Dir(pattern)
			if info, err := os.Stat(parent); err == nil && info.IsDir() {
				if err := w.watcher.Add(parent); err == nil {
					fmt.Println(ui.Muted(fmt.Sprintf("Watching directory for new standalone: %s", parent)))
				}
			}
		}
	}

	// 3. Watch existing resource directories recursively (entire resource path, not just src)
	paths := append([]string{}, w.config.GetResourcePaths()...)
	paths = append(paths, w.config.GetStandalonePaths()...)
	for _, basePath := range paths {
		// Walk entire resource directory recursively to catch all changes
		// (fxmanifest.lua, package.json, src/, views/, etc.)
		err := filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip directories we can't access
			}
			if d.IsDir() {
				// Skip node_modules and other common non-source directories
				name := d.Name()
				if name == "node_modules" || name == "dist" || name == ".git" {
					return filepath.SkipDir
				}

				// Skip the destination directory if it's inside the project
				if w.config.Destination != "" {
					absPath, _ := filepath.Abs(path)
					absDest, _ := filepath.Abs(w.config.Destination)
					if absPath == absDest {
						return filepath.SkipDir
					}
				}
				if watchErr := w.watcher.Add(path); watchErr != nil {
					// Silent fail for duplicates or already watched
				}
			}
			return nil
		})

		if err == nil {
			fmt.Println(ui.Info(fmt.Sprintf("Watching: %s (recursive)", basePath)))
		}
	}
}

func (w *Watcher) tasksForChangedFile(all []builder.BuildTask, changedFile string) []builder.BuildTask {
	changedAbs, err := filepath.Abs(changedFile)
	if err != nil {
		changedAbs = changedFile
	}

	// Find best matching task (longest path prefix)
	bestIdx := -1
	bestLen := -1
	for i, t := range all {
		taskAbs, err := filepath.Abs(t.Path)
		if err != nil {
			taskAbs = t.Path
		}

		// Normalize path separators for Windows
		taskAbs = filepath.Clean(taskAbs)
		changedAbs = filepath.Clean(changedAbs)

		if strings.HasPrefix(changedAbs, taskAbs+string(os.PathSeparator)) || changedAbs == taskAbs {
			if len(taskAbs) > bestLen {
				bestLen = len(taskAbs)
				bestIdx = i
			}
		}
	}

	if bestIdx == -1 {
		return nil
	}

	best := all[bestIdx]
	base := strings.Split(best.ResourceName, "/")[0]

	// If the change is in a views task, rebuild only the views task.
	if best.Type == builder.TypeViews || strings.HasSuffix(best.ResourceName, "/ui") {
		return []builder.BuildTask{best}
	}

	// Otherwise rebuild all tasks belonging to the base resource (e.g., resource + its views).
	var affected []builder.BuildTask
	for _, t := range all {
		if strings.Split(t.ResourceName, "/")[0] == base {
			affected = append(affected, t)
		}
	}
	return affected
}

func (w *Watcher) Close() error {
	if w.restarter != nil {
		_ = w.restarter.Stop()
	}
	return w.watcher.Close()
}

// notifyFramework restarts affected resources or the managed process.
func (w *Watcher) notifyFramework(results []builder.BuildResult) {
	// Find unique resources that were successfully built
	uniqueResources := make(map[string]struct{})
	for _, r := range results {
		if r.Success {
			// Get base resource name (e.g., "core" instead of "core/ui")
			baseName := strings.Split(r.Task.ResourceName, "/")[0]
			uniqueResources[baseName] = struct{}{}
		}
	}
	resources := make([]string, 0, len(uniqueResources))
	for resourceName := range uniqueResources {
		resources = append(resources, resourceName)
	}
	if err := w.restarter.Restart(resources); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Restart failed: %v", err)))
		return
	}

	if len(resources) == 0 || w.restarter.Mode() == "none" {
		return
	}

	if w.restarter.Mode() == "process" {
		fmt.Println(ui.Success("Managed server process restarted"))
		return
	}

	for _, resourceName := range resources {
		fmt.Println(ui.Success(fmt.Sprintf("Restart triggered for %s (via %s)", resourceName, w.restarter.Mode())))
	}
}

type LogMessage struct {
	Level     int                    `json:"level"`
	Domain    string                 `json:"domain"`
	Message   string                 `json:"message"`
	Timestamp int64                  `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Error     *struct {
		Name    string `json:"name"`
		Message string `json:"message"`
		Stack   string `json:"stack,omitempty"`
	} `json:"error,omitempty"`
}

func (w *Watcher) startLogPrinter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case log := <-w.logQueue:
			w.displayLog(log)
		}
	}
}

func (w *Watcher) startBridgeServer(ctx context.Context) error {
	port := w.config.Dev.BridgePort()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	})
	mux.HandleFunc("/logs", w.handleLogs)

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println(ui.Warning(fmt.Sprintf("Dev bridge stopped: %v", err)))
		}
	}()

	return nil
}

func (w *Watcher) handleLogs(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload struct {
		Type    string       `json:"type"`
		Payload []LogMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, log := range payload.Payload {
		select {
		case w.logQueue <- log:
		default:
		}
	}

	rw.WriteHeader(http.StatusAccepted)
}

func (w *Watcher) displayLog(log LogMessage) {
	timeStr := time.Unix(log.Timestamp/1000, 0).Format("15:04:05")

	levelStyle := lipgloss.NewStyle().Bold(true)
	levelLabel := "INFO"

	switch log.Level {
	case 10: // Trace
		levelStyle = levelStyle.Foreground(lipgloss.Color("#9CA3AF"))
		levelLabel = "TRCE"
	case 20: // Debug
		levelStyle = levelStyle.Foreground(lipgloss.Color("#60A5FA"))
		levelLabel = "DEBG"
	case 30: // Info
		levelStyle = levelStyle.Foreground(lipgloss.Color("#34D399"))
		levelLabel = "INFO"
	case 40: // Warn
		levelStyle = levelStyle.Foreground(lipgloss.Color("#FBBF24"))
		levelLabel = "WARN"
	case 50: // Error
		levelStyle = levelStyle.Foreground(lipgloss.Color("#F87171"))
		levelLabel = "ERR "
	case 60: // Fatal
		levelStyle = levelStyle.Foreground(lipgloss.Color("#EF4444")).Background(lipgloss.Color("#FFFFFF"))
		levelLabel = "FATL"
	}

	domainStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	fmt.Printf("%s %s %s %s\n",
		timeStyle.Render(timeStr),
		levelStyle.Render(levelLabel),
		domainStyle.Render(fmt.Sprintf("[%s]", log.Domain)),
		msgStyle.Render(log.Message),
	)

	if log.Error != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Italic(true)
		fmt.Printf("  %s: %s\n", errStyle.Render(log.Error.Name), errStyle.Render(log.Error.Message))
		if log.Error.Stack != "" {
			stackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563"))
			fmt.Println(stackStyle.Render(log.Error.Stack))
		}
	}
}
