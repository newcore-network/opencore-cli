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
  resource    Create a new framework-connected module in resources/ folder
              • Can use DI container, database, exports, events
              • Part of the OpenCore ecosystem
              • Best for: gameplay features, admin systems, economy, etc.

  standalone  Create an independent script in standalones/ folder
              • Self-contained, no framework dependencies
              • Faster to develop, smaller bundle size
              • Best for: simple utilities, libraries, legacy scripts

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
