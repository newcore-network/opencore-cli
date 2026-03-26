package commands

import (
	"fmt"
	"os"

	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/newcore-network/opencore-cli/internal/updater"
	"github.com/spf13/cobra"
)

func NewUpdateCommand() *cobra.Command {
	var force bool
	var channel string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update OpenCore CLI from a release channel",
		Run: func(cmd *cobra.Command, args []string) {
			version := cmd.Root().Version
			if version == "" {
				version = "0.0.0"
			}

			channel = updater.NormalizeChannel(channel)

			if force {
				fmt.Println(ui.Info(fmt.Sprintf("Checking for updates on the %s channel (forced)...", channel)))
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Checking for updates on the %s channel...", channel)))
			}

			info, err := updater.CheckForUpdate(version, force, channel)
			if err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to check for updates: %v", err)))
				return
			}

			if !updater.NeedsUpdate(cmd.Root().Version, info.LatestVersion) {
				fmt.Println(ui.Success(fmt.Sprintf("OpenCore CLI is already up to date on the %s channel (%s)", channel, cmd.Root().Version)))
				return
			}

			fmt.Println(ui.Info(fmt.Sprintf("New %s release available: %s (current: %s)", channel, info.LatestVersion, cmd.Root().Version)))

			if updater.IsNPMInstallation() {
				fmt.Println(ui.Warning("It looks like you installed OpenCore CLI via NPM."))
				fmt.Println(ui.Info("Please run the following command to update:"))
				npmTarget := "@open-core/cli"
				if channel != updater.ChannelStable {
					npmTarget = fmt.Sprintf("@open-core/cli@%s", channel)
				}
				fmt.Println(ui.Info(fmt.Sprintf("  npm install -g %s", npmTarget)))
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

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force update check, ignoring cache")
	cmd.Flags().StringVar(&channel, "channel", updater.GetConfiguredChannel(), "Update channel to use (stable|beta)")

	return cmd
}
