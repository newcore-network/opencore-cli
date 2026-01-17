package commands

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

func newCreateResourceCommand() *cobra.Command {
	var withClient bool
	var withNUI bool

	cmd := &cobra.Command{
		Use:   "resource [name]",
		Short: "Create a new independent resource",
		Long:  "Generate a new resource in resources/ directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateResource(cmd, args, withClient, withNUI)
		},
	}

	cmd.Flags().BoolVar(&withClient, "with-client", false, "Include client-side code")
	cmd.Flags().BoolVar(&withNUI, "with-nui", false, "Include NUI (UI)")

	return cmd
}

func runCreateResource(cmd *cobra.Command, args []string, withClient, withNUI bool) error {
	fmt.Println(ui.TitleStyle.Render("Create New Resource"))
	fmt.Println()

	resourceName, err := getNameFromArgsOrPrompt(args, createNamePrompt{
		Title:       "Resource Name",
		Description: "Name for your resource (e.g., chat, admin)",
		Kind:        "resource",
	})
	if err != nil {
		return err
	}

	// Ask for optional features interactively if user didn't specify a name argument.
	if len(args) == 0 {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Include client-side code?").
					Value(&withClient),
				huh.NewConfirm().
					Title("Include NUI?").
					Value(&withNUI),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
	}

	resourcePath := filepath.Join("resources", resourceName)

	fmt.Println(ui.Info(fmt.Sprintf("Creating resource: %s", resourceName)))
	fmt.Println()

	// Generate resource
	if err := templates.GenerateResource(resourcePath, resourceName, withClient, withNUI); err != nil {
		return fmt.Errorf("failed to generate resource: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success("Resource created successfully!"))
	fmt.Println()

	renderCreateBox(
		fmt.Sprintf("Location: %s\n\n", resourcePath) +
			featuresMessage(withClient, withNUI) + "\n\n" +
			"Next steps:\n" +
			fmt.Sprintf("  cd %s\n", resourcePath) +
			"  pnpm install\n" +
			"// or use workspace node_modules package\n\n" +
			"Remember to add your resource to opencore.config.ts:\n" +
			"  resources: {\n" +
			"    include: ['./resources/*'],\n" +
			"  }",
	)

	return nil
}
