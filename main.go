package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/commands"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

var (
	version = "0.2.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "opencore",
		Short: "OpenCore CLI - Official tooling for OpenCore Framework",
		Long: ui.Logo() + "\n\n" +
			"OpenCore CLI is the official command-line tool for creating,\n" +
			"managing, and building FiveM servers with the OpenCore Framework.",
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(ui.Error(err.Error()))
		os.Exit(1)
	}
}
