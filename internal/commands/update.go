package commands

import (
	"fmt"
	"os"

	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/newcore-network/opencore-cli/internal/updater"
	"github.com/spf13/cobra"
)

func NewUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update OpenCore CLI to the latest version",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Root().Flags().GetString("version")
			if version == "" {
				// Fallback to a default if not found, though main.go sets it
				version = "0.0.0"
			}

			fmt.Println(ui.Info("Checking for updates..."))

			info, err := updater.CheckForUpdate(version)
			if err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to check for updates: %v", err)))
				return
			}

			if !updater.NeedsUpdate(cmd.Root().Version, info.LatestVersion) {
				fmt.Println(ui.Success(fmt.Sprintf("OpenCore CLI is already up to date (%s)", cmd.Root().Version)))
				return
			}

			fmt.Println(ui.Info(fmt.Sprintf("New version available: %s (current: %s)", info.LatestVersion, cmd.Root().Version)))

			if updater.IsNPMInstallation() {
				fmt.Println(ui.Warning("It looks like you installed OpenCore CLI via NPM."))
				fmt.Println(ui.Info("Please run the following command to update:"))
				fmt.Println(ui.Info("  npm install -g @open-core/cli"))
				return
			}

			fmt.Println(ui.Info("Updating..."))
			err = updater.Update(info.LatestVersion)
			if err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to update: %v", err)))
				os.Exit(1)
			}

			fmt.Println(ui.Success(fmt.Sprintf("Successfully updated to %s!", info.LatestVersion)))
		},
	}
}
