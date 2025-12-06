package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
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

	// Detect project architecture
	arch := config.DetectArchitecture(".")

	fmt.Println(ui.Info(fmt.Sprintf("Detected architecture: %s", arch)))
	fmt.Println(ui.Info(fmt.Sprintf("Creating feature: %s", featureName)))
	fmt.Println()

	var featurePath string
	var filesCreated []string

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

	fmt.Println()
	fmt.Println(ui.Success("Feature created successfully!"))
	fmt.Println()

	filesList := ""
	for _, file := range filesCreated {
		filesList += fmt.Sprintf("  • %s\n", file)
	}

	fmt.Println(ui.BoxStyle.Render(
		fmt.Sprintf("📁 Location: %s\n\n", featurePath) +
			"Files created:\n" +
			filesList + "\n" +
			"Next: Import your feature in the appropriate bootstrap file",
	))
	fmt.Println()

	return nil
}
