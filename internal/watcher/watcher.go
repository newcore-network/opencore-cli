package watcher

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/newcore-network/opencore-cli/internal/builder"
	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

type Watcher struct {
	config  *config.Config
	builder *builder.Builder
	watcher *fsnotify.Watcher
	debounce map[string]time.Time
}

func New(cfg *config.Config) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		config:   cfg,
		builder:  builder.New(cfg),
		watcher:  w,
		debounce: make(map[string]time.Time),
	}, nil
}

func (w *Watcher) Watch() error {
	// Add paths to watch
	paths := w.config.GetResourcePaths()
	for _, path := range paths {
		srcPath := filepath.Join(path, "src")
		if err := w.watcher.Add(srcPath); err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to watch %s: %v", srcPath, err)))
		} else {
			fmt.Println(ui.Info(fmt.Sprintf("Watching: %s", srcPath)))
		}
	}

	fmt.Println()
	fmt.Println(ui.Success("Development mode started!"))
	fmt.Println(ui.Muted("Watching for changes... (Press Ctrl+C to stop)"))
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
				// Debounce - only rebuild if file hasn't changed in last 300ms
				now := time.Now()
				if lastChange, exists := w.debounce[event.Name]; exists {
					if now.Sub(lastChange) < 300*time.Millisecond {
						continue
					}
				}
				w.debounce[event.Name] = now

				fmt.Println(ui.Info(fmt.Sprintf("File changed: %s", filepath.Base(event.Name))))
				if err := w.builder.Build(); err != nil {
					fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", err)))
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

func (w *Watcher) Close() error {
	return w.watcher.Close()
}

