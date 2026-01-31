package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/builder"
	"github.com/newcore-network/opencore-cli/internal/config"
)

func NewBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build all resources for production",
		Long:  "Compile TypeScript to JavaScript and prepare resources for deployment.",
		RunE:  runBuild,
	}

	return cmd
}

func runBuild(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, root, err := config.LoadWithProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("failed to switch to project root: %w", err)
	}

	// Create builder and build
	b := builder.New(cfg)
	return b.Build()
}
