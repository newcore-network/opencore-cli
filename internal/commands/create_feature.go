package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

func newCreateFeatureCommand() *cobra.Command {
	var resourceName string

	cmd := &cobra.Command{
		Use:   "feature [name]",
		Short: "Create a new feature in the core or a resource",
		Long: `Generate a new feature with controller and service.

	By default, features are created in core/src/features/.
	Use -r to create the feature inside a specific resource instead.

Examples:
  opencore create feature banking           # Creates in core
  opencore create feature chat -r myresource  # Creates in resources/myserver/`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateFeature(cmd, args, resourceName)
		},
	}

	cmd.Flags().StringVarP(&resourceName, "resource", "r", "", "Create feature inside a resource instead of core")

	return cmd
}

func runCreateFeature(cmd *cobra.Command, args []string, resourceName string) error {
	fmt.Println(ui.TitleStyle.Render("Create New Feature"))
	fmt.Println()

	featureName, err := getNameFromArgsOrPrompt(args, createNamePrompt{
		Title:       "Feature Name",
		Description: "Name for your feature (e.g., banking, jobs)",
		Kind:        "feature",
	})
	if err != nil {
		return err
	}

	var featurePath string
	var filesCreated []string

	// Check if creating in a resource
	if resourceName != "" {
		// Verify resource exists
		resourcePath := filepath.Join("resources", resourceName)
		if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
			return fmt.Errorf("resource '%s' does not exist in resources/", resourceName)
		}

		fmt.Println(ui.Info(fmt.Sprintf("Creating feature '%s' in resource '%s'", featureName, resourceName)))
		fmt.Println()

		// Create feature in resource's src/server/features/
		featurePath = filepath.Join(resourcePath, "src", "server", "features", featureName)
		if err := templates.GenerateFeature(featurePath, featureName); err != nil {
			return fmt.Errorf("failed to generate feature: %w", err)
		}
		filesCreated = []string{
			featureName + ".controller.ts",
			featureName + ".service.ts",
			"index.ts",
		}
	} else {
		fmt.Println(ui.Info(fmt.Sprintf("Creating feature: %s", featureName)))
		fmt.Println()

		featurePath = filepath.Join("core", "src", "features", featureName)
		if err := templates.GenerateFeature(featurePath, featureName); err != nil {
			return fmt.Errorf("failed to generate feature: %w", err)
		}
		filesCreated = []string{
			featureName + ".controller.ts",
			featureName + ".service.ts",
			"index.ts",
		}
	}

	fmt.Println()
	fmt.Println(ui.Success("Feature created successfully!"))
	fmt.Println()

	filesList := ""
	for _, file := range filesCreated {
		filesList += fmt.Sprintf("  • %s\n", file)
	}

	renderCreateBox(
		fmt.Sprintf("Location: %s\n\n", featurePath) +
			"Files created:\n" +
			filesList + "\n" +
			"Next: Import your feature in the appropriate bootstrap file",
	)

	return nil
}
