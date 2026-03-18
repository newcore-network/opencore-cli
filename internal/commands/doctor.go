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
	"github.com/newcore-network/opencore-cli/internal/pkgmgr"
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
	Name     string
	Passed   bool
	Message  string
	Required bool
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.TitleStyle.Render("Health Check"))
	fmt.Println()

	checks := []CheckResult{}

	// Check Node.js
	nodeCheck := checkCommand("node", "--version", "Node.js")
	nodeCheck.Required = true
	checks = append(checks, nodeCheck)

	// Package manager checks
	pnpmCheck := checkCommand("pnpm", "--version", "pnpm")
	pnpmCheck.Required = false
	checks = append(checks, pnpmCheck)
	yarnCheck := checkCommand("yarn", "--version", "yarn")
	yarnCheck.Required = false
	checks = append(checks, yarnCheck)
	npmCheck := checkCommand("npm", "--version", "npm")
	npmCheck.Required = false
	checks = append(checks, npmCheck)

	preference := pkgmgr.EffectivePreference(".")
	resolved, err := pkgmgr.Resolve(preference)
	if err != nil {
		checks = append(checks, CheckResult{Name: "Package Manager", Passed: false, Message: err.Error(), Required: true})
	} else {
		checks = append(checks, CheckResult{Name: "Package Manager", Passed: true, Message: fmt.Sprintf("%s (%s)", resolved.Choice, resolved.Version), Required: true})
	}

	// Check if in OpenCore project
	projectCheck := checkOpenCoreProject()
	projectCheck.Required = true
	checks = append(checks, projectCheck)

	// Check configuration
	var cfg *config.Config
	if projectCheck.Passed {
		var configCheck CheckResult
		cfg, configCheck = checkConfig()
		configCheck.Required = true
		checks = append(checks, configCheck)
		if configCheck.Passed {
			checks = append(checks, CheckResult{Name: "Runtime", Passed: true, Message: cfg.RuntimeKind(), Required: true})
			adapterCheck := checkAdapter(cfg)
			adapterCheck.Required = true
			checks = append(checks, adapterCheck)
		}
	}

	// Check if dependencies are installed
	if projectCheck.Passed {
		depsCheck := checkDependencies(resolved)
		depsCheck.Required = true
		checks = append(checks, depsCheck)
	}

	// Render results table
	renderCheckResults(checks)

	// Determine overall status
	allPassed := true
	for _, check := range checks {
		if check.Required && !check.Passed {
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
			Name:     name,
			Passed:   false,
			Message:  "Not found or not working",
			Required: true,
		}
	}

	version := strings.TrimSpace(string(output))
	return CheckResult{
		Name:     name,
		Passed:   true,
		Message:  version,
		Required: true,
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

func checkConfig() (*config.Config, CheckResult) {
	cfg, root, err := config.LoadWithProjectRoot()
	if err != nil {
		return nil, CheckResult{
			Name:    "Configuration",
			Passed:  false,
			Message: err.Error(),
		}
	}
	if err := os.Chdir(root); err != nil {
		return nil, CheckResult{
			Name:    "Configuration",
			Passed:  false,
			Message: err.Error(),
		}
	}

	if cfg.Destination == "" {
		return cfg, CheckResult{
			Name:    "Configuration",
			Passed:  true,
			Message: fmt.Sprintf("Valid configuration (no destination; build output: %s)", cfg.OutDir),
		}
	}

	return cfg, CheckResult{
		Name:    "Configuration",
		Passed:  true,
		Message: fmt.Sprintf("Valid configuration (destination: %s)", cfg.Destination),
	}
}

func checkAdapter(cfg *config.Config) CheckResult {
	if cfg == nil || cfg.Adapter == nil || (cfg.Adapter.Server == nil && cfg.Adapter.Client == nil) {
		return CheckResult{
			Name:    "Adapter",
			Passed:  true,
			Message: "No adapter configured (framework default adapter resolution)",
		}
	}

	parts := []string{}
	passed := true

	if cfg.Adapter.Server != nil {
		parts = append(parts, formatAdapterBinding("server", cfg.Adapter.Server))
		passed = passed && cfg.Adapter.Server.Valid
	}
	if cfg.Adapter.Client != nil {
		parts = append(parts, formatAdapterBinding("client", cfg.Adapter.Client))
		passed = passed && cfg.Adapter.Client.Valid
	}

	if len(parts) == 0 {
		parts = append(parts, "Adapter block present but empty")
		passed = false
	}

	return CheckResult{
		Name:    "Adapter",
		Passed:  passed,
		Message: strings.Join(parts, "; "),
	}
}

func formatAdapterBinding(side string, binding *config.AdapterBinding) string {
	if binding == nil {
		return fmt.Sprintf("%s: not configured", side)
	}

	name := strings.TrimSpace(binding.Name)
	if name == "" {
		name = "unknown"
	}

	status := "valid"
	if !binding.Valid {
		status = "invalid"
	}

	message := strings.TrimSpace(binding.Message)
	if message != "" {
		return fmt.Sprintf("%s: %s [%s] (%s: %s)", side, name, adapterRuntimeLabel(binding), status, message)
	}

	return fmt.Sprintf("%s: %s [%s] (%s)", side, name, adapterRuntimeLabel(binding), status)
}

func adapterRuntimeLabel(binding *config.AdapterBinding) string {
	if binding == nil || binding.Runtime == nil || strings.TrimSpace(binding.Runtime.Runtime) == "" {
		return "default"
	}
	return strings.TrimSpace(binding.Runtime.Runtime)
}

func checkDependencies(pm pkgmgr.Resolved) CheckResult {
	// Check if node_modules exists
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		install := "pnpm install"
		if pm.Choice != "" {
			install = pm.InstallCmd()
		}
		return CheckResult{
			Name:    "Dependencies",
			Passed:  false,
			Message: fmt.Sprintf("Run '%s' to install dependencies", install),
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
		if !check.Passed && !check.Required {
			status = ui.Warning("WARN")
		}
		if !check.Passed && check.Required {
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
