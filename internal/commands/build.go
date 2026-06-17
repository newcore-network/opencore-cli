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

	cmd.Flags().String("output", "auto", "Output mode (auto|tui|plain)")
	cmd.Flags().StringP("environment", "e", "", "Environment to build for (e.g. development, production)")

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

	if env, _ := cmd.Flags().GetString("environment"); env != "" {
		cfg.Build.Environment = env
	}

	outputModeValue, _ := cmd.Flags().GetString("output")
	outputMode, err := builder.ParseOutputMode(outputModeValue)
	if err != nil {
		return err
	}

	// Create builder and build
	b := builder.New(cfg)
	return b.BuildWithOutputContext(cmd.Context(), outputMode)
}
