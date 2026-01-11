package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/commands"
	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/newcore-network/opencore-cli/internal/updater"
)

var (
	version = "0.4.9"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "opencore",
		Short: "OpenCore CLI - Official tooling for OpenCore Framework",
		Long: ui.Logo() + "\n\n" +
			"OpenCore CLI is the official command-line tool for creating,\n" +
			"managing, and building FiveM servers with the OpenCore Framework.\n\n" +
			"Project Structure:\n" +
			"  • Resources:    Framework-connected modules in resources/ folder\n" +
			"                  (can use DI container, exports, events, database, etc.)\n" +
			"  • Standalones:  Independent scripts in standalones/ folder\n" +
			"                  (self-contained, no framework dependencies)",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Set version template
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Add commands
	rootCmd.AddCommand(commands.NewInitCommand())
	rootCmd.AddCommand(commands.NewCreateCommand())
	rootCmd.AddCommand(commands.NewBuildCommand())
	rootCmd.AddCommand(commands.NewDevCommand())
	rootCmd.AddCommand(commands.NewDoctorCommand())
	rootCmd.AddCommand(commands.NewCloneCommand())
	rootCmd.AddCommand(commands.NewUpdateCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(ui.Error(err.Error()))
		os.Exit(1)
	}

	// Check for updates in the background after command execution
	if len(os.Args) > 1 && os.Args[1] != "update" && os.Args[1] != "--version" && os.Args[1] != "-v" {
		if info, err := updater.CheckForUpdate(version, false); err == nil {
			if updater.NeedsUpdate(version, info.LatestVersion) {
				fmt.Println()
				fmt.Println(ui.Info(fmt.Sprintf("New version available: %s -> %s", version, info.LatestVersion)))
				fmt.Println(ui.Info("Run 'opencore update' to update to the latest version."))
			}
		}
	}
}
