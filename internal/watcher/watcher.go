package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/newcore-network/opencore-cli/internal/builder"
	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

type Watcher struct {
	config    *config.Config
	builder   *builder.Builder
	watcher   *fsnotify.Watcher
	restarter restarter
	logQueue  chan LogMessage
	closeOnce sync.Once
	closeErr  error
}

const (
	debounceDelay = 500 * time.Millisecond
	maxLogBody    = 1 << 20
	maxLogDomain  = 256
	maxLogMessage = 16 << 10
	maxLogStack   = 64 << 10
)

type buildRequest struct {
	generation   uint64
	builder      *builder.Builder
	tasks        []builder.BuildTask
	full         bool
	startRuntime bool
}

type buildOutcome struct {
	request buildRequest
	results []builder.BuildResult
	err     error
}

type debounceState map[string]time.Time

func (d debounceState) add(path string, now time.Time) {
	d[path] = now.Add(debounceDelay)
}

func (d debounceState) takeDue(now time.Time) []string {
	var due []string
	for path, deadline := range d {
		if !deadline.After(now) {
			due = append(due, path)
			delete(d, path)
		}
	}
	sort.Strings(due)
	return due
}

func (d debounceState) next(now time.Time) time.Duration {
	var earliest time.Time
	for _, deadline := range d {
		if earliest.IsZero() || deadline.Before(earliest) {
			earliest = deadline
		}
	}
	if earliest.IsZero() {
		return -1
	}
	if wait := earliest.Sub(now); wait > 0 {
		return wait
	}
	return 0
}

func New(cfg *config.Config) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		config:   cfg,
		builder:  builder.New(cfg),
		watcher:  w,
		logQueue: make(chan LogMessage, 256),
	}

	restarter, err := newRestarter(cfg)
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	watcher.restarter = restarter

	return watcher, nil
}

func (w *Watcher) Watch(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	allTasks := w.builder.CollectTasks()
	restarter := w.restarter
	defer func() { _ = restarter.Stop() }()

	// Watch config file for dynamic updates
	configPath := "opencore.config.ts"
	if _, err := os.Stat(configPath); err == nil {
		if err := w.watcher.Add(filepath.Dir(configPath)); err != nil {
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
	switch restarter.Mode() {
	case "txadmin":
		fmt.Println(ui.Info(fmt.Sprintf("Restart mode: txAdmin (%s)", w.config.Dev.TxAdmin.URL)))
		fmt.Println(ui.Muted("Authenticating with txAdmin..."))
		if err := restarter.Start(ctx); err != nil {
			_ = w.Close()
			return fmt.Errorf("txAdmin login failed: %w", err)
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

	buildRequests := make(chan buildRequest, 1)
	buildResults := make(chan buildOutcome, 1)
	go runBuildWorker(ctx, buildRequests, buildResults)

	var generation uint64
	building := false
	pendingFull := false
	tasksDirty := false
	pendingPaths := make(map[string]struct{})
	debounces := make(debounceState)
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	var timerC <-chan time.Time

	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		wait := debounces.next(time.Now())
		if wait < 0 {
			timerC = nil
			return
		}
		timer.Reset(wait)
		timerC = timer.C
	}

	startBuild := func(paths map[string]struct{}, full, startRuntime bool) bool {
		if building {
			if full {
				pendingFull = true
			}
			for path := range paths {
				pendingPaths[path] = struct{}{}
			}
			return false
		}

		request := buildRequest{generation: generation, builder: w.builder, full: full, startRuntime: startRuntime}
		if tasksDirty {
			allTasks = w.builder.CollectTasks()
			tasksDirty = false
		}
		if full {
		} else {
			seen := make(map[string]struct{})
			for path := range paths {
				for _, task := range w.tasksForChangedFile(allTasks, path) {
					key := task.Path + "\x00" + task.ResourceName
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						request.tasks = append(request.tasks, task)
					}
				}
			}
			if len(request.tasks) == 0 {
				return false
			}
		}
		building = true
		buildRequests <- request
		return true
	}

	startBuild(nil, true, true)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timerC:
			due := debounces.takeDue(time.Now())
			resetTimer()
			paths := make(map[string]struct{}, len(due))
			for _, path := range due {
				if filepath.Base(path) == "opencore.config.ts" {
					fmt.Println(ui.Info("Configuration changed, reloading..."))
					newCfg, root, err := config.LoadWithProjectRoot()
					if err != nil {
						fmt.Println(ui.Error(fmt.Sprintf("Failed to reload config: %v", err)))
						continue
					}
					if err := os.Chdir(root); err != nil {
						fmt.Println(ui.Error(fmt.Sprintf("Failed to switch to project root: %v", err)))
						continue
					}
					newRestarter, err := newRestarter(newCfg)
					if err != nil {
						fmt.Println(ui.Error(fmt.Sprintf("Failed to configure restart mode: %v", err)))
						continue
					}
					if err := restarter.Stop(); err != nil {
						_ = w.Close()
						return fmt.Errorf("failed to stop previous dev runtime: %w", err)
					}
					restarter = newRestarter
					w.config = newCfg
					w.builder = builder.New(newCfg)
					generation++
					tasksDirty = true
					w.registerPaths()
					pendingFull = true
					fmt.Println(ui.Info("Config reloaded, triggering full build..."))
					continue
				}
				paths[path] = struct{}{}
			}
			if pendingFull {
				if startBuild(nil, true, true) {
					pendingFull = false
					pendingPaths = make(map[string]struct{})
				}
			} else {
				startBuild(paths, false, false)
			}
		case outcome := <-buildResults:
			building = false
			if outcome.request.generation == generation {
				if outcome.err != nil {
					fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", outcome.err)))
				} else if outcome.request.startRuntime {
					if err := restarter.Start(ctx); err != nil {
						_ = w.Close()
						return fmt.Errorf("failed to start dev runtime: %w", err)
					}
				} else if err := w.notifyFramework(restarter, outcome.results); err != nil {
					_ = w.Close()
					return err
				}
			}

			if pendingFull {
				pendingFull = false
				pendingPaths = make(map[string]struct{})
				startBuild(nil, true, true)
			} else if len(pendingPaths) > 0 {
				paths := pendingPaths
				pendingPaths = make(map[string]struct{})
				startBuild(paths, false, false)
			}
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if w.shouldIgnorePath(event.Name) {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.registerDirectory(event.Name)
				}
			}
			if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				tasksDirty = true
			}
			debounces.add(event.Name, time.Now())
			resetTimer()
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Println(ui.Error(fmt.Sprintf("Watcher error: %v", err)))
		}
	}
}

func runBuildWorker(ctx context.Context, requests <-chan buildRequest, outcomes chan<- buildOutcome) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-requests:
			outcome := buildOutcome{request: request}
			if request.full {
				outcome.err = request.builder.BuildWithOutputContext(ctx, builder.OutputModeAuto)
			} else {
				regenerateTypes(request.builder, request.tasks)
				outcome.results, outcome.err = request.builder.BuildTasksContext(ctx, request.tasks)
			}
			select {
			case outcomes <- outcome:
			case <-ctx.Done():
				return
			}
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
				if name == "node_modules" || name == "dist" || name == ".git" || name == ".opencore" {
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

func (w *Watcher) registerDirectory(root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}
		if w.shouldIgnorePath(path) {
			return filepath.SkipDir
		}
		_ = w.watcher.Add(path)
		return nil
	})
}

func regenerateTypes(b *builder.Builder, tasks []builder.BuildTask) {
	resourceBuilder := b.ResourceBuilder()
	if resourceBuilder == nil {
		return
	}

	seen := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		if task.Type == builder.TypeViews || seen[task.Path] {
			continue
		}
		seen[task.Path] = true

		if resourceBuilder.RunTypegen(task.Path) {
			fmt.Println(ui.Muted(fmt.Sprintf("  types regenerated: %s", task.ResourceName)))
		}
	}
}

func (w *Watcher) tasksForChangedFile(all []builder.BuildTask, changedFile string) []builder.BuildTask {
	if w.shouldIgnorePath(changedFile) {
		return nil
	}

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

func (w *Watcher) shouldIgnorePath(path string) bool {
	cleanPath := filepath.Clean(path)
	slashPath := filepath.ToSlash(cleanPath)

	if strings.Contains(slashPath, "/node_modules/") ||
		strings.Contains(slashPath, "/dist/") ||
		strings.Contains(slashPath, "/.git/") ||
		strings.Contains(slashPath, "/.opencore/") ||
		strings.HasSuffix(slashPath, "/node_modules") ||
		strings.HasSuffix(slashPath, "/dist") ||
		strings.HasSuffix(slashPath, "/.git") ||
		strings.HasSuffix(slashPath, "/.opencore") {
		return true
	}

	if w == nil || w.config == nil {
		return false
	}

	if isPathWithin(cleanPath, w.config.OutDir) {
		return true
	}

	if isPathWithin(cleanPath, w.config.Destination) {
		return true
	}

	return false
}

func isPathWithin(path string, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}

	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}

	pathAbs = filepath.Clean(pathAbs)
	rootAbs = filepath.Clean(rootAbs)

	if pathAbs == rootAbs {
		return true
	}

	return strings.HasPrefix(pathAbs, rootAbs+string(os.PathSeparator))
}

func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
		w.closeErr = w.watcher.Close()
	})
	return w.closeErr
}

// notifyFramework restarts affected resources or the managed process.
func (w *Watcher) notifyFramework(r restarter, results []builder.BuildResult) error {
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
	sort.Strings(resources)
	if err := r.Restart(resources); err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}

	if len(resources) == 0 || r.Mode() == "none" {
		return nil
	}

	if r.Mode() == "process" {
		fmt.Println(ui.Success("Managed server process restarted"))
		return nil
	}

	for _, resourceName := range resources {
		fmt.Println(ui.Success(fmt.Sprintf("Restart triggered for %s (via %s)", resourceName, r.Mode())))
	}
	return nil
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

	defer r.Body.Close()
	if r.ContentLength > maxLogBody {
		rw.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(rw, r.Body, maxLogBody))
	if err != nil {
		rw.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	var payload struct {
		Type    string       `json:"type"`
		Payload []LogMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, log := range payload.Payload {
		sanitizeLog(&log)
		select {
		case w.logQueue <- log:
		default:
		}
	}

	rw.WriteHeader(http.StatusAccepted)
}

func sanitizeLog(log *LogMessage) {
	log.Domain = sanitizeTerminalText(log.Domain, maxLogDomain)
	log.Message = sanitizeTerminalText(log.Message, maxLogMessage)
	if log.Error != nil {
		log.Error.Name = sanitizeTerminalText(log.Error.Name, maxLogDomain)
		log.Error.Message = sanitizeTerminalText(log.Error.Message, maxLogMessage)
		log.Error.Stack = sanitizeTerminalText(log.Error.Stack, maxLogStack)
	}
}

func sanitizeTerminalText(value string, maxBytes int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
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
