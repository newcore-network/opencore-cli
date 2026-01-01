package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

func newCreateStandaloneCommand() *cobra.Command {
	var withClient bool
	var withNUI bool

	cmd := &cobra.Command{
		Use:   "standalone [name]",
		Short: "Create a new standalone resource",
		Long: `Generate a new standalone resource in standalone/ directory.

Standalone resources are independent scripts that don't depend on the OpenCore Framework.
They're useful for utilities, legacy scripts, or simple functionality.

Examples:
  opencore create standalone utils
  opencore create standalone admin --with-client`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateStandalone(cmd, args, withClient, withNUI)
		},
	}

	cmd.Flags().BoolVar(&withClient, "with-client", false, "Include client-side code")
	cmd.Flags().BoolVar(&withNUI, "with-nui", false, "Include NUI (UI)")

	return cmd
}

func runCreateStandalone(cmd *cobra.Command, args []string, withClient, withNUI bool) error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Create New Standalone"))
	fmt.Println()

	var standaloneName string

	// Get standalone name from args or prompt
	if len(args) > 0 {
		standaloneName = args[0]
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Standalone Name").
					Description("Name for your standalone resource (e.g., utils, logger)").
					Value(&standaloneName).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("standalone name cannot be empty")
						}
						if strings.Contains(s, " ") {
							return fmt.Errorf("standalone name cannot contain spaces")
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

	standalonePath := filepath.Join("standalone", standaloneName)

	fmt.Println(ui.Info(fmt.Sprintf("Creating standalone: %s", standaloneName)))
	fmt.Println()

	// Generate standalone
	if err := templates.GenerateStandalone(standalonePath, standaloneName, withClient, withNUI); err != nil {
		return fmt.Errorf("failed to generate standalone: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success("Standalone created successfully!"))
	fmt.Println()

	featuresMsg := "Features:\n  - Server-side code"
	if withClient {
		featuresMsg += "\n  - Client-side code"
	}
	if withNUI {
		featuresMsg += "\n  - NUI (UI)"
	}

	fmt.Println(ui.BoxStyle.Render(
		fmt.Sprintf("Location: %s\n\n", standalonePath) +
			featuresMsg + "\n\n" +
			"Next steps:\n" +
			fmt.Sprintf("  cd %s\n", standalonePath) +
			"  pnpm install\n\n" +
			"Remember to add your standalone to opencore.config.ts:\n" +
			"  standalone: {\n" +
			"    include: ['./standalone/*'],\n" +
			"  }",
	))
	fmt.Println()

	return nil
}
