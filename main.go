package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/commands"
	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/newcore-network/opencore-cli/internal/updater"
)

var (
	version = "1.2.3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			pm, _ := cmd.Flags().GetString("usePackageManager")
			pm = strings.TrimSpace(pm)
			if pm != "" {
				os.Setenv("OPENCORE_PACKAGE_MANAGER", pm)
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().String("usePackageManager", "", "Package manager to use (pnpm|yarn|npm|auto)")

	// Set version template
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Add commands
	rootCmd.AddCommand(commands.NewInitCommand())
	rootCmd.AddCommand(commands.NewCreateCommand())
	rootCmd.AddCommand(commands.NewBuildCommand())
	rootCmd.AddCommand(commands.NewDevCommand())
	rootCmd.AddCommand(commands.NewDoctorCommand())
	rootCmd.AddCommand(commands.NewCloneCommand())
	rootCmd.AddCommand(commands.NewAdapterCommand())
	rootCmd.AddCommand(commands.NewUpdateCommand())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(ui.Error(err.Error()))
		os.Exit(1)
	}

	// Check for updates in the background after command execution
	if shouldCheckForUpdates(os.Args) {
		channel := updater.GetConfiguredChannel()
		if info, err := updater.CheckForUpdate(version, false, channel); err == nil {
			if updater.NeedsUpdate(version, info.LatestVersion) {
				fmt.Println()
				fmt.Println(ui.Info(fmt.Sprintf("New %s version available: %s -> %s", channel, version, info.LatestVersion)))
				fmt.Println(ui.Info(fmt.Sprintf("Run 'opencore update --channel %s' to update.", channel)))
			}
		}
	}
}

func shouldCheckForUpdates(args []string) bool {
	if len(args) <= 1 {
		return false
	}

	if args[1] == "update" || args[1] == "--version" || args[1] == "-v" {
		return false
	}

	if ui.IsUpdateCheckDisabled() || ui.IsNonInteractiveSession() {
		return false
	}

	return true
}
