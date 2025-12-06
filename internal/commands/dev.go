package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/newcore-network/opencore-cli/internal/watcher"
)

func NewDevCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start development mode with hot-reload",
		Long:  "Watch for file changes and automatically rebuild resources.",
		RunE:  runDev,
	}

	return cmd
}

func runDev(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Development Mode"))
	fmt.Println()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create watcher
	w, err := watcher.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer w.Close()

	// Start watching
	return w.Watch()
}
