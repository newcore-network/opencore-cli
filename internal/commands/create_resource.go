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
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Create New Resource"))
	fmt.Println()

	var resourceName string

	// Get resource name from args or prompt
	if len(args) > 0 {
		resourceName = args[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Resource Name").
					Description("Name for your resource (e.g., chat, admin)").
					Value(&resourceName).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("resource name cannot be empty")
						}
						if strings.Contains(s, " ") {
							return fmt.Errorf("resource name cannot contain spaces")
						}
						return nil
					}),
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

	featuresMsg := "Features:\n  • Server-side code"
	if withClient {
		featuresMsg += "\n  • Client-side code"
	}
	if withNUI {
		featuresMsg += "\n  • NUI (UI)"
	}

	fmt.Println(ui.BoxStyle.Render(
		fmt.Sprintf("📁 Location: %s\n\n", resourcePath) +
			featuresMsg + "\n\n" +
			"Next steps:\n" +
			fmt.Sprintf("  cd %s\n", resourcePath) +
			"  pnpm install",
	))
	fmt.Println()

	return nil
}
