package watcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	"github.com/newcore-network/opencore-cli/internal/watcher/txadmin"
)

type Watcher struct {
	config         *config.Config
	builder        *builder.Builder
	watcher        *fsnotify.Watcher
	debounceTimers map[string]*time.Timer
	txAdminClient  *txadmin.Client
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
		buildingSet:    make(map[string]bool),
	}

	// Initialize txAdmin client if configured
	if cfg.Dev.IsTxAdminConfigured() {
		client, err := txadmin.NewClient(
			cfg.Dev.TxAdminURL,
			cfg.Dev.TxAdminUser,
			cfg.Dev.TxAdminPassword,
		)
		if err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to create txAdmin client: %v", err)))
		} else {
			watcher.txAdminClient = client
		}
	}

	return watcher, nil
}

func (w *Watcher) Watch() error {
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

	// Start log bridge
	go w.startLogBridge()

	fmt.Println()

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#4F46E5")).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	// Count unique resources (not build tasks - a resource can have multiple tasks like server, client, views)
	uniqueResources := make(map[string]struct{})
	for _, task := range allTasks {
		baseResource := strings.Split(task.ResourceName, "/")[0]
		uniqueResources[baseResource] = struct{}{}
	}

	fmt.Printf("%s %s\n",
		headerStyle.Render(" DEV MODE "),
		statusStyle.Render(fmt.Sprintf("Project: %s | Resources: %d", w.config.Name, len(uniqueResources))))

	// Show hot-reload mode
	if w.txAdminClient != nil {
		fmt.Println(ui.Info(fmt.Sprintf("Hot-reload: txAdmin (%s)", w.config.Dev.TxAdminURL)))
		// Attempt initial login
		fmt.Println(ui.Muted("Authenticating with txAdmin..."))
		if err := w.txAdminClient.Login(); err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("txAdmin login failed: %v", err)))
			fmt.Println(ui.Warning("Hot-reload via txAdmin will not work. Check your credentials."))
		} else {
			fmt.Println(ui.Success("Connected to txAdmin"))
		}
	} else {
		fmt.Println(ui.Muted("Hot-reload: Internal HTTP (configure txAdmin for CORE hot-reload)"))
	}

	fmt.Println(ui.Muted("Watching for changes... (Ctrl+C to stop)"))
	fmt.Println()

	// Build once at start
	if err := w.builder.Build(); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Initial build failed: %v", err)))
	}

	// Watch for changes
	for {
		select {
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
					// Handle config file change
					if filepath.Base(fileName) == "opencore.config.ts" {
						fmt.Println(ui.Info("Configuration changed, reloading..."))
						newCfg, err := config.Load()
						if err != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Failed to reload config: %v", err)))
							return
						}
						w.config = newCfg
						w.builder = builder.New(newCfg)
						allTasks = w.builder.CollectTasks()

						// Re-add all paths (fsnotify handles duplicates)
						w.registerPaths()

						fmt.Println(ui.Info("Config reloaded, triggering full build..."))
						if err := w.builder.Build(); err != nil {
							fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", err)))
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
					results, err := w.builder.BuildTasks(affected)

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
					w.notifyFramework(affected, results)

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
					newCfg, _ := config.Load()
					if newCfg != nil {
						w.config = newCfg
						w.builder = builder.New(newCfg)
						allTasks = w.builder.CollectTasks()
						fmt.Println(ui.Info(fmt.Sprintf("New resource detected: %s", filepath.Base(event.Name))))
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
	if w.config.Standalone != nil {
		for _, pattern := range w.config.Standalone.Include {
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
				// Skip node_modules and build output directories
				name := d.Name()
				if name == "node_modules" || name == "dist" || name == "build" || name == ".git" {
					return filepath.SkipDir
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
	return w.watcher.Close()
}

// notifyFramework tells the framework to restart the affected resources
func (w *Watcher) notifyFramework(tasks []builder.BuildTask, results []builder.BuildResult) {
	// Find unique resources that were successfully built
	uniqueResources := make(map[string]bool)
	for _, r := range results {
		if r.Success {
			// Get base resource name (e.g., "core" instead of "core/ui")
			baseName := strings.Split(r.Task.ResourceName, "/")[0]
			uniqueResources[baseName] = true
		}
	}

	// Get core resource name from config
	coreResourceName := w.config.Core.ResourceName
	if coreResourceName == "" {
		coreResourceName = "core"
	}

	// Notify framework for each resource
	for resourceName := range uniqueResources {
		// Check if we need txAdmin for this resource
		isCoreResource := resourceName == coreResourceName

		if w.txAdminClient != nil {
			// Use txAdmin API (works for all resources including CORE)
			go w.notifyViaTxAdmin(resourceName)
		} else if isCoreResource {
			// CORE resource without txAdmin - show warning
			fmt.Println(ui.Warning(fmt.Sprintf(
				"Cannot hot-reload '%s' (CORE resource cannot restart itself). Configure txAdmin in opencore.config.ts for CORE hot-reload.",
				resourceName,
			)))
		} else {
			// Non-CORE resource - use internal HTTP
			go w.notifyViaHTTP(resourceName)
		}
	}
}

// notifyViaTxAdmin uses txAdmin API to restart a resource
func (w *Watcher) notifyViaTxAdmin(resourceName string) {
	// Helper function to check if error is auth-related
	isAuthError := func(err error) bool {
		if err == nil {
			return false
		}
		errMsg := err.Error()
		return strings.Contains(errMsg, "authentication") ||
			strings.Contains(errMsg, "unauthorized") ||
			strings.Contains(errMsg, "status 401") ||
			strings.Contains(errMsg, "status 403")
	}

	// Helper function to re-authenticate
	reauth := func() error {
		fmt.Println(ui.Warning("txAdmin session expired, re-authenticating..."))
		if err := w.txAdminClient.Login(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		fmt.Println(ui.Success("Re-authenticated with txAdmin"))
		return nil
	}

	// First refresh resources to ensure txAdmin sees the updated files
	if err := w.txAdminClient.RefreshResources(); err != nil {
		if isAuthError(err) {
			if reauthErr := reauth(); reauthErr != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Re-authentication failed: %v", reauthErr)))
				return
			}
			// Retry refresh after re-auth
			if err := w.txAdminClient.RefreshResources(); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to refresh resources: %v", err)))
				return
			}
		} else {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to refresh resources: %v", err)))
		}
	}

	// Then restart the resource
	if err := w.txAdminClient.RestartResource(resourceName); err != nil {
		if isAuthError(err) {
			if reauthErr := reauth(); reauthErr != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Re-authentication failed: %v", reauthErr)))
				return
			}
			// Retry restart after re-auth
			if err := w.txAdminClient.RestartResource(resourceName); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Hot-reload failed for %s: %v", resourceName, err)))
				return
			}
		} else {
			fmt.Println(ui.Error(fmt.Sprintf("Hot-reload failed for %s: %v", resourceName, err)))
			return
		}
	}

	fmt.Println(ui.Success(fmt.Sprintf("Hot-reload triggered for %s (via txAdmin)", resourceName)))
}

// notifyViaHTTP uses internal HTTP server to restart a resource (fallback)
func (w *Watcher) notifyViaHTTP(resourceName string) {
	port := w.config.Dev.Port
	endpoint := fmt.Sprintf("http://localhost:%d/restart?resource=%s", port, url.QueryEscape(resourceName))

	resp, err := http.Post(endpoint, "application/json", nil)
	if err != nil {
		// Don't show error if framework is not running (silent failure is better here)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println(ui.Success(fmt.Sprintf("Hot-reload triggered for %s", resourceName)))
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

type LogResponse struct {
	Logs      []LogMessage `json:"logs"`
	Timestamp int64        `json:"timestamp"`
}

func (w *Watcher) startLogBridge() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		port := w.config.Dev.Port
		endpoint := fmt.Sprintf("http://localhost:%d/logs", port)

		resp, err := http.Get(endpoint)
		if err != nil {
			continue // Framework probably not running
		}

		var logResp LogResponse
		if err := json.NewDecoder(resp.Body).Decode(&logResp); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, log := range logResp.Logs {
			w.displayLog(log)
		}
	}
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
