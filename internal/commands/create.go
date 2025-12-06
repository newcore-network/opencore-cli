package commands

import (
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [feature|resource]",
		Short: "Create a new feature or resource",
		Long:  "Create a new feature in the core or a new independent resource.",
	}

	// Add subcommands
	cmd.AddCommand(newCreateFeatureCommand())
	cmd.AddCommand(newCreateResourceCommand())

	return cmd
}
