package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

func newCreateFeatureCommand() *cobra.Command {
	var resourceName string

	cmd := &cobra.Command{
		Use:   "feature [name]",
		Short: "Create a new feature in the core or a resource",
		Long: `Generate a new feature with controller and service.

By default, features are created in core/src/features/ (or modules/ for domain-driven).
Use -r to create the feature inside a specific resource instead.

Examples:
  opencore create feature banking           # Creates in core
  opencore create feature chat -r myserver  # Creates in resources/myserver/`,
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
		// Create in core - detect project architecture
		arch := config.DetectArchitecture(".")

		fmt.Println(ui.Info(fmt.Sprintf("Detected architecture: %s", arch)))
		fmt.Println(ui.Info(fmt.Sprintf("Creating feature: %s", featureName)))
		fmt.Println()

		// Generate based on architecture
		switch arch {
		case config.ArchitectureDomainDriven:
			// Domain-Driven: create module with client/server/shared
			featurePath = filepath.Join("core", "src", "modules", featureName)
			if err := templates.GenerateModuleDomainDriven(featurePath, featureName); err != nil {
				return fmt.Errorf("failed to generate module: %w", err)
			}
			filesCreated = []string{
				"client/" + featureName + ".controller.ts",
				"client/" + featureName + ".ui.ts",
				"server/" + featureName + ".controller.ts",
				"server/" + featureName + ".service.ts",
				"server/" + featureName + ".repository.ts",
				"shared/" + featureName + ".types.ts",
				"shared/" + featureName + ".events.ts",
			}

		case config.ArchitectureLayerBased:
			// Layer-Based: create in controllers and services directories
			clientPath := filepath.Join("core", "src", "client", "controllers")
			serverPath := filepath.Join("core", "src", "server", "controllers")
			servicePath := filepath.Join("core", "src", "server", "services")

			if err := templates.GenerateLayerBased(clientPath, serverPath, servicePath, featureName); err != nil {
				return fmt.Errorf("failed to generate layer-based feature: %w", err)
			}
			featurePath = "core/src/"
			filesCreated = []string{
				"client/controllers/" + featureName + ".controller.ts",
				"client/services/" + featureName + ".client.service.ts",
				"server/controllers/" + featureName + ".controller.ts",
				"server/services/" + featureName + ".service.ts",
			}

		case config.ArchitectureNo:
			// No-Architecture: simple server.ts and client.ts in core/src
			featurePath = filepath.Join("core", "src")
			if err := templates.GenerateNoArchitecture(featurePath, featureName); err != nil {
				return fmt.Errorf("failed to generate no-architecture feature: %w", err)
			}
			filesCreated = []string{
				featureName + ".server.ts",
				featureName + ".client.ts",
			}
			fmt.Println(ui.Info("Note: Don't forget to import these files in your server.ts and client.ts bootstrap files."))

		case config.ArchitectureFeatureBased, config.ArchitectureHybrid:
			// Feature-Based or Hybrid: use features directory
			featurePath = filepath.Join("core", "src", "features", featureName)
			if err := templates.GenerateFeature(featurePath, featureName); err != nil {
				return fmt.Errorf("failed to generate feature: %w", err)
			}
			filesCreated = []string{
				featureName + ".controller.ts",
				featureName + ".service.ts",
				"index.ts",
			}

		default:
			// Unknown: fallback to feature-based
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
