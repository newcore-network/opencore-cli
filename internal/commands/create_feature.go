package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCreateFeatureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature [name]",
		Short: "Create a new feature in the core",
		Long:  "Generate a new feature with controller and service in core/src/features/",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCreateFeature,
	}

	return cmd
}

func runCreateFeature(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Create New Feature"))
	fmt.Println()

	var featureName string

	// Get feature name from args or prompt
	if len(args) > 0 {
		featureName = args[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Feature Name").
					Description("Name for your feature (e.g., banking, jobs)").
					Value(&featureName).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("feature name cannot be empty")
						}
						if strings.Contains(s, " ") {
							return fmt.Errorf("feature name cannot contain spaces")
						}
						return nil
					}),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}
	}

	// Check if in core directory
	corePath := "core/src/features"
	featurePath := filepath.Join(corePath, featureName)

	fmt.Println(ui.Info(fmt.Sprintf("Creating feature: %s", featureName)))
	fmt.Println()

	// Generate feature
	if err := templates.GenerateFeature(featurePath, featureName); err != nil {
		return fmt.Errorf("failed to generate feature: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success("Feature created successfully!"))
	fmt.Println()
	fmt.Println(ui.BoxStyle.Render(
		fmt.Sprintf("📁 Location: %s\n\n", featurePath) +
			"Files created:\n" +
			fmt.Sprintf("  • %s.controller.ts\n", featureName) +
			fmt.Sprintf("  • %s.service.ts\n", featureName) +
			"  • index.ts\n\n" +
			"Next: Import your feature in core/src/server/main.ts",
	))
	fmt.Println()

	return nil
}
