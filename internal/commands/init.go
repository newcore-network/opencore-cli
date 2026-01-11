package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/templates"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new OpenCore project",
		Long:  "Create a new OpenCore project with the recommended structure and configuration.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}

	cmd.Flags().StringP("dir", "d", ".", "Directory where the project folder will be created")
	cmd.Flags().String("architecture", "", "Project architecture (domain-driven|layer-based|feature-based|hybrid)")
	cmd.Flags().Bool("minify", false, "Enable code minification in production builds")
	cmd.Flags().StringArray("module", nil, "Add a module to install (repeatable)")
	cmd.Flags().String("destination", "", "Build output directory (usually your FiveM resources folder)")
	cmd.Flags().Bool("skip-destination", false, "Do not set destination during init (you can edit opencore.config.ts later)")
	cmd.Flags().Bool("non-interactive", false, "Do not run the interactive wizard; use flags/defaults")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	baseDir, _ := cmd.Flags().GetString("dir")
	architectureFlag, _ := cmd.Flags().GetString("architecture")
	minifyFlag, _ := cmd.Flags().GetBool("minify")
	modulesFlag, _ := cmd.Flags().GetStringArray("module")
	destinationFlag, _ := cmd.Flags().GetString("destination")
	skipDestination, _ := cmd.Flags().GetBool("skip-destination")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	// Define wizard steps
	steps := []ui.WizardStep{
		{
			Title:       "Project Name",
			Description: "Name of your OpenCore server project (no spaces)",
			Type:        ui.StepTypeInput,
			Validate: func(s string) error {
				if s == "" {
					return fmt.Errorf("project name cannot be empty")
				}
				if strings.Contains(s, " ") {
					return fmt.Errorf("project name cannot contain spaces")
				}
				projectDir := filepath.Join(baseDir, s)
				if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
					return fmt.Errorf("directory '%s' already exists", projectDir)
				}
				return nil
			},
		},
		{
			Title:       "Architecture",
			Description: "Choose how to organize your code",
			Type:        ui.StepTypeSelect,
			Options: []ui.WizardOption{
				{
					Label: "Domain-Driven",
					Value: "domain-driven",
					Desc:  "Recommended for large projects with complex business logic",
				},
				{
					Label: "Layer-Based",
					Value: "layer-based",
					Desc:  "Traditional separation by technical layers (controllers, services)",
				},
				{
					Label: "Feature-Based",
					Value: "feature-based",
					Desc:  "Simple structure, good for small projects",
				},
				{
					Label: "Hybrid",
					Value: "hybrid",
					Desc:  "Flexible approach for evolving projects",
				},
			},
		},
		{
			Title:       "Modules",
			Description: "Select official modules to install (space to toggle)",
			Type:        ui.StepTypeMultiSelect,
			Options: []ui.WizardOption{
				{
					Label: "@open-core/identity",
					Value: "@open-core/identity",
					Desc:  "Authentication and player identity management",
				},
				// Future modules can be added here:
				// {
				// 	Label: "@open-core/economy",
				// 	Value: "@open-core/economy",
				// 	Desc:  "In-game economy and currency system",
				// },
				// {
				// 	Label: "@open-core/inventory",
				// 	Value: "@open-core/inventory",
				// 	Desc:  "Player inventory management",
				// },
			},
		},
		{
			Title:       "Minification",
			Description: "Enable code minification in production builds?",
			Type:        ui.StepTypeConfirm,
		},
	}
	if !skipDestination {
		steps = append(steps, ui.WizardStep{
			Title:       "Server Destination",
			Description: "Build output directory (optional; usually your FiveM resources folder)",
			Type:        ui.StepTypeInput,
			Validate: func(s string) error {
				return nil
			},
		})
	}

	// Non-interactive mode: rely on flags/defaults
	if nonInteractive {
		projectName := ""
		if len(args) > 0 {
			projectName = args[0]
		}
		if projectName == "" {
			return fmt.Errorf("project name is required in non-interactive mode")
		}
		architecture := architectureFlag
		if architecture == "" {
			architecture = "feature-based"
		}
		useMinify := minifyFlag
		modules := modulesFlag
		destination := ""
		if !skipDestination {
			destination = destinationFlag
		}

		installIdentity := false
		for _, mod := range modules {
			if mod == "@open-core/identity" {
				installIdentity = true
				break
			}
		}

		projectPath := filepath.Join(baseDir, projectName)
		fmt.Println()
		fmt.Println(ui.Logo())
		fmt.Println()
		fmt.Println(ui.Info(fmt.Sprintf("Creating project: %s", projectName)))
		fmt.Println()
		if err := templates.GenerateStarterProject(projectPath, projectName, architecture, installIdentity, useMinify, destination); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}
		fmt.Println()
		fmt.Println(ui.Success("Project created successfully!"))
		fmt.Println()
		return nil
	}

	// Pre-fill project name if provided as argument
	wizard := ui.NewWizard(steps)
	if len(args) > 0 {
		wizard.GetValues()["Project Name"] = args[0]
	}
	if architectureFlag != "" {
		wizard.GetValues()["Architecture"] = architectureFlag
	}
	if len(modulesFlag) > 0 {
		wizard.GetValues()["Modules"] = modulesFlag
	}
	wizard.GetValues()["Minification"] = minifyFlag
	if destinationFlag != "" {
		wizard.GetValues()["Server Destination"] = destinationFlag
	}

	// Run the wizard
	p := tea.NewProgram(wizard, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	result := finalModel.(*ui.WizardModel)

	// Check if cancelled
	if result.IsCancelled() {
		fmt.Println(ui.Warning("Project creation cancelled."))
		return nil
	}

	// Extract values
	projectName := result.GetStringValue("Project Name")
	architecture := result.GetStringValue("Architecture")
	modules := result.GetStringSliceValue("Modules")
	useMinify := result.GetBoolValue("Minification")
	destination := ""
	if !skipDestination {
		destination = result.GetStringValue("Server Destination")
	}

	// Check if identity module is selected
	installIdentity := false
	for _, mod := range modules {
		if mod == "@open-core/identity" {
			installIdentity = true
			break
		}
	}

	// Create project directory
	projectPath := filepath.Join(baseDir, projectName)

	fmt.Println()
	fmt.Println(ui.Logo())
	fmt.Println()
	fmt.Println(ui.Info(fmt.Sprintf("Creating project: %s", projectName)))
	fmt.Println()

	// Generate project from template
	if err := templates.GenerateStarterProject(projectPath, projectName, architecture, installIdentity, useMinify, destination); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success("Project created successfully!"))
	fmt.Println()

	// Build modules string
	modulesStr := "None"
	if len(modules) > 0 {
		modulesStr = strings.Join(modules, ", ")
	}

	// Summary box
	summaryContent := fmt.Sprintf(
		"Project: %s\n"+
			"Architecture: %s\n"+
			"Modules: %s\n"+
			"Minify: %s\n"+
			"Destination: %s\n\n"+
			"Next steps:\n"+
			"  cd %s\n"+
			"  pnpm install\n"+
			"  opencore dev",
		projectName,
		architecture,
		modulesStr,
		boolToYesNo(useMinify),
		destination,
		projectName,
	)

	fmt.Println(ui.SuccessBoxStyle.Render(summaryContent))
	fmt.Println()

	return nil
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
