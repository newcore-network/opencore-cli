package commands

import (
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <type> [name]",
		Short: "Create a new feature, resource, or standalone",
		Long: `Create scaffolding for different project components.

Types:
  feature     Create a new feature in the core (or resource with -r flag)
  resource    Create a new satellite resource (depends on core)
  standalone  Create a new standalone resource (no dependencies)

Examples:
  opencore create feature banking
  opencore create feature chat -r myserver
  opencore create resource admin --with-client
  opencore create standalone utils`,
	}

	// Add subcommands
	cmd.AddCommand(newCreateFeatureCommand())
	cmd.AddCommand(newCreateResourceCommand())
	cmd.AddCommand(newCreateStandaloneCommand())

	return cmd
}
