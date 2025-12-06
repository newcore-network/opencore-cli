package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new OpenCore project",
		Long:  "Create a new OpenCore project with the recommended structure and configuration.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.Logo())
	fmt.Println(ui.TitleStyle.Render("Initialize New Project"))
	fmt.Println()

	var projectName string
	var architecture string
	var installIdentity bool
	var useMinify bool

	// If project name not provided as arg, prompt for it
	if len(args) > 0 {
		projectName = args[0]
	}

	// Always show interactive form for configuration
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project Name").
				Description("Name of your OpenCore server project").
				Value(&projectName).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("project name cannot be empty")
					}
					if strings.Contains(s, " ") {
						return fmt.Errorf("project name cannot contain spaces")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project Architecture").
				Description("Choose how to organize your code").
				Options(
					huh.NewOption("Domain-Driven (Recommended for large projects)", "domain-driven"),
					huh.NewOption("Layer-Based (For large teams)", "layer-based"),
					huh.NewOption("Feature-Based (Simple, for small projects)", "feature-based"),
					huh.NewOption("Hybrid (Flexible, evolving projects)", "hybrid"),
				).
				Value(&architecture),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Install @open-core/identity?").
				Description("Official identity and authentication module").
				Value(&installIdentity),
			huh.NewConfirm().
				Title("Enable minification in production?").
				Value(&useMinify).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Create project directory
	projectPath := filepath.Join(".", projectName)
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", projectName)
	}

	fmt.Println(ui.Info(fmt.Sprintf("Creating project: %s", projectName)))
	fmt.Println()

	// Generate project from template
	if err := templates.GenerateStarterProject(projectPath, projectName, architecture, installIdentity, useMinify); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success("Project created successfully!"))
	fmt.Println()
	fmt.Println(ui.BoxStyle.Render(
		fmt.Sprintf("📁 Project: %s\n\n", projectName) +
			"Next steps:\n" +
			fmt.Sprintf("  cd %s\n", projectName) +
			"  pnpm install\n" +
			"  opencore dev",
	))
	fmt.Println()

	return nil
}
