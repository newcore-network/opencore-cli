package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/pkgmgr"
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
	cmd.Flags().Bool("minify", false, "Enable code minification in production builds")
	cmd.Flags().String("adapter", "", "Project adapter (none|fivem|redm|ragemp)")
	cmd.Flags().String("destination", "", "Deployment root path (FiveM resources dir or RageMP server root)")
	cmd.Flags().Bool("non-interactive", false, "Do not run the interactive wizard; use flags/defaults")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	baseDir, _ := cmd.Flags().GetString("dir")
	minifyFlag, _ := cmd.Flags().GetBool("minify")
	adapterFlag, _ := cmd.Flags().GetString("adapter")
	destinationFlag, _ := cmd.Flags().GetString("destination")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	preference := pkgmgr.PreferenceFromEnv()
	usePmFlag, _ := cmd.Flags().GetString("usePackageManager")
	usePmFlag = strings.TrimSpace(usePmFlag)
	if usePmFlag != "" {
		choice, err := pkgmgr.ParseChoice(usePmFlag)
		if err != nil {
			return err
		}
		preference = choice
	}

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
			Title:       "Adapter",
			Description: "Choose the runtime adapter to configure centrally",
			Type:        ui.StepTypeSelect,
			Options: []ui.WizardOption{
				{
					Label: "None",
					Value: "none",
					Desc:  "Use the framework default adapter resolution (NodeJS)",
				},
				{
					Label: "FiveM",
					Value: "fivem",
					Desc:  "Install @open-core/fivem-adapter and wire server/client adapters centrally",
				},
				{
					Label:    "RedM (Coming soon)",
					Value:    "redm",
					Desc:     "Reserved adapter slot for RedM support",
					Disabled: true,
				},
				{
					Label: "RageMP",
					Value: "ragemp",
					Desc:  "Install @open-core/ragemp-adapter and use RageMP build defaults",
				},
			},
		},
		{
			Title:       "Minification",
			Description: "Enable code minification in production builds?",
			Type:        ui.StepTypeConfirm,
		},
	}
	steps = append(steps, ui.WizardStep{
		Title:       "Package Manager",
		Description: "Choose the package manager for this project",
		Type:        ui.StepTypeSelect,
		Options: []ui.WizardOption{
			{Label: "Auto", Value: string(pkgmgr.ChoiceAuto), Desc: "Prefer pnpm, then yarn (v2+), then npm"},
			{Label: "pnpm", Value: string(pkgmgr.ChoicePnpm), Desc: "Fast, disk-efficient"},
			{Label: "yarn (berry)", Value: string(pkgmgr.ChoiceYarn), Desc: "Modern yarn (v2+)"},
			{Label: "npm", Value: string(pkgmgr.ChoiceNpm), Desc: "Default Node.js package manager"},
		},
	})
	steps = append(steps, ui.WizardStep{
		Title:       "Server Destination",
		Description: "Deployment root path, e.g. C:/FXServer/server-data/resources or C:/ragemp-server (optional)",
		Type:        ui.StepTypeInput,
		Validate: func(s string) error {
			return nil
		},
	})

	// Non-interactive mode: rely on flags/defaults
	if nonInteractive {
		projectName := ""
		if len(args) > 0 {
			projectName = args[0]
		}
		if projectName == "" {
			return fmt.Errorf("project name is required in non-interactive mode")
		}
		useMinify := minifyFlag
		adapter := strings.TrimSpace(strings.ToLower(adapterFlag))
		if adapter == "" {
			adapter = "none"
		}
		switch adapter {
		case "none", "fivem", "ragemp":
		case "redm":
			return fmt.Errorf("adapter %q is not available yet", adapter)
		default:
			return fmt.Errorf("invalid adapter %q (expected: none, fivem, redm, or ragemp)", adapter)
		}
		destination := strings.TrimSpace(destinationFlag)

		resolved, err := pkgmgr.Resolve(preference)
		if err != nil {
			return err
		}

		projectPath := filepath.Join(baseDir, projectName)
		fmt.Println()
		fmt.Println(ui.Logo())
		fmt.Println()
		fmt.Println(ui.Info(fmt.Sprintf("Creating project: %s", projectName)))
		fmt.Println()
		if err := templates.GenerateStarterProject(projectPath, projectName, false, adapter, useMinify, destination, fmt.Sprintf("%s@%s", resolved.Choice, resolved.Version)); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}
		fmt.Println()
		fmt.Println(ui.Success("Project created successfully!"))
		fmt.Println()

		summaryContent := fmt.Sprintf(
			"Project: %s\n"+
				"Adapter: %s\n"+
				"Minify: %s\n"+
				"Destination: %s\n\n"+
				"Next steps:\n"+
				"  cd %s\n"+
				"  %s\n"+
				"  opencore dev",
			projectName,
			adapter,
			boolToYesNo(useMinify),
			destination,
			projectName,
			resolved.InstallCmd(),
		)
		fmt.Println(ui.SuccessBoxStyle.Render(summaryContent))
		fmt.Println()
		return nil
	}

	// Pre-fill project name if provided as argument
	wizard := ui.NewWizard(steps)
	if len(args) > 0 {
		wizard.GetValues()["Project Name"] = args[0]
	}
	if adapterFlag != "" {
		wizard.GetValues()["Adapter"] = strings.ToLower(adapterFlag)
	} else {
		wizard.GetValues()["Adapter"] = "none"
	}
	wizard.GetValues()["Minification"] = minifyFlag
	wizard.GetValues()["Package Manager"] = string(preference)
	if destinationFlag != "" {
		wizard.GetValues()["Server Destination"] = destinationFlag
	}

	// Run the wizard
	p := tea.NewProgram(wizard, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	result, ok := finalModel.(ui.WizardModel)
	if !ok {
		return fmt.Errorf("unexpected wizard result type %T", finalModel)
	}

	// Check if cancelled
	if result.IsCancelled() {
		fmt.Println(ui.Warning("Project creation cancelled."))
		return nil
	}

	// Extract values
	projectName := result.GetStringValue("Project Name")
	adapter := result.GetStringValue("Adapter")
	useMinify := result.GetBoolValue("Minification")
	packageManager := result.GetStringValue("Package Manager")
	destination := strings.TrimSpace(result.GetStringValue("Server Destination"))

	pmChoice, err := pkgmgr.ParseChoice(packageManager)
	if err != nil {
		return err
	}
	resolved, err := pkgmgr.Resolve(pmChoice)
	if err != nil {
		return err
	}

	// Create project directory
	projectPath := filepath.Join(baseDir, projectName)

	fmt.Println()
	fmt.Println(ui.Logo())
	fmt.Println()
	fmt.Println(ui.Info(fmt.Sprintf("Creating project: %s", projectName)))
	fmt.Println()

	// Generate project from template
	if err := templates.GenerateStarterProject(projectPath, projectName, false, adapter, useMinify, destination, fmt.Sprintf("%s@%s", resolved.Choice, resolved.Version)); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success("Project created successfully!"))
	fmt.Println()

	// Summary box
	summaryContent := fmt.Sprintf(
		"Project: %s\n"+
			"Adapter: %s\n"+
			"Minify: %s\n"+
			"Destination: %s\n\n"+
			"Next steps:\n"+
			"  cd %s\n"+
			"  %s\n"+
			"  opencore dev",
		projectName,
		adapter,
		boolToYesNo(useMinify),
		destination,
		projectName,
		resolved.InstallCmd(),
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
