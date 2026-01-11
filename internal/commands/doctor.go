package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/ui"
)

func NewDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check project health and dependencies",
		Long:  "Validate that all required dependencies and configuration are correct.",
		RunE:  runDoctor,
	}

	return cmd
}

type CheckResult struct {
	Name    string
	Passed  bool
	Message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.TitleStyle.Render("Health Check"))
	fmt.Println()

	checks := []CheckResult{}

	// Check Node.js
	nodeCheck := checkCommand("node", "--version", "Node.js")
	checks = append(checks, nodeCheck)

	// Check pnpm
	pnpmCheck := checkCommand("pnpm", "--version", "pnpm")
	checks = append(checks, pnpmCheck)

	// Check if in OpenCore project
	projectCheck := checkOpenCoreProject()
	checks = append(checks, projectCheck)

	// Check configuration
	if projectCheck.Passed {
		configCheck := checkConfig()
		checks = append(checks, configCheck)
	}

	// Check if dependencies are installed
	if projectCheck.Passed {
		depsCheck := checkDependencies()
		checks = append(checks, depsCheck)
	}

	// Render results table
	renderCheckResults(checks)

	// Determine overall status
	allPassed := true
	for _, check := range checks {
		if !check.Passed {
			allPassed = false
			break
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println(ui.SuccessBoxStyle.Render("✓ All checks passed! Your project is healthy."))
	} else {
		fmt.Println(ui.ErrorBoxStyle.Render("✗ Some checks failed. Please fix the issues above."))
		return fmt.Errorf("health check failed")
	}

	return nil
}

func checkCommand(command string, args string, name string) CheckResult {
	cmd := exec.Command(command, args)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return CheckResult{
			Name:    name,
			Passed:  false,
			Message: "Not found or not working",
		}
	}

	version := strings.TrimSpace(string(output))
	return CheckResult{
		Name:    name,
		Passed:  true,
		Message: version,
	}
}

func checkOpenCoreProject() CheckResult {
	// Check for opencore.config.ts or package.json with @open-core/framework
	configExists := false
	if _, err := os.Stat("opencore.config.ts"); err == nil {
		configExists = true
	}

	packageExists := false
	if _, err := os.Stat("package.json"); err == nil {
		packageExists = true
	}

	coreExists := false
	if info, err := os.Stat("core"); err == nil && info.IsDir() {
		coreExists = true
	}

	if configExists || (packageExists && coreExists) {
		return CheckResult{
			Name:    "OpenCore Project",
			Passed:  true,
			Message: "Valid project structure",
		}
	}

	return CheckResult{
		Name:    "OpenCore Project",
		Passed:  false,
		Message: "Not in an OpenCore project directory",
	}
}

func checkConfig() CheckResult {
	cfg, err := config.Load()
	if err != nil {
		return CheckResult{
			Name:    "Configuration",
			Passed:  false,
			Message: err.Error(),
		}
	}

	if cfg.Destination == "" {
		return CheckResult{
			Name:    "Configuration",
			Passed:  true,
			Message: fmt.Sprintf("Valid configuration (no destination; build output: %s)", cfg.OutDir),
		}
	}

	return CheckResult{
		Name:    "Configuration",
		Passed:  true,
		Message: fmt.Sprintf("Valid configuration (destination: %s)", cfg.Destination),
	}
}

func checkDependencies() CheckResult {
	// Check if node_modules exists
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Dependencies",
			Passed:  false,
			Message: "Run 'pnpm install' to install dependencies",
		}
	}

	// Check if @open-core/framework is installed
	frameworkPath := filepath.Join("node_modules", "@open-core", "framework")
	if _, err := os.Stat(frameworkPath); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Dependencies",
			Passed:  false,
			Message: "@open-core/framework not found",
		}
	}

	return CheckResult{
		Name:    "Dependencies",
		Passed:  true,
		Message: "All dependencies installed",
	}
}

func renderCheckResults(checks []CheckResult) {
	// Table headers
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.PrimaryColor).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	headers := []string{
		headerStyle.Render("Check"),
		headerStyle.Render("Status"),
		headerStyle.Render("Details"),
	}

	rows := [][]string{}
	for _, check := range checks {
		status := ui.Success("PASS")
		if !check.Passed {
			status = ui.Error("FAIL")
		}

		rows = append(rows, []string{
			cellStyle.Render(check.Name),
			cellStyle.Render(status),
			cellStyle.Render(check.Message),
		})
	}

	// Calculate column widths
	widths := []int{20, 10, 50}

	// Print header
	fmt.Println(strings.Repeat("─", widths[0]+widths[1]+widths[2]+6))
	fmt.Printf("%-*s %-*s %-*s\n", widths[0], headers[0], widths[1], headers[1], widths[2], headers[2])
	fmt.Println(strings.Repeat("─", widths[0]+widths[1]+widths[2]+6))

	// Print rows
	for _, row := range rows {
		fmt.Printf("%-*s %-*s %-*s\n", widths[0], row[0], widths[1], row[1], widths[2], row[2])
	}
	fmt.Println(strings.Repeat("─", widths[0]+widths[1]+widths[2]+6))
}
