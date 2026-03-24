package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

type adapterPackageManifest struct {
	Name             string                     `json:"name"`
	Dependencies     map[string]string          `json:"dependencies,omitempty"`
	DevDependencies  map[string]string          `json:"devDependencies,omitempty"`
	PeerDependencies map[string]string          `json:"peerDependencies,omitempty"`
	Exports          map[string]json.RawMessage `json:"exports,omitempty"`
}

type adapterSideReport struct {
	Name            string   `json:"name"`
	Expected        bool     `json:"expected"`
	FactoryPath     string   `json:"factoryPath,omitempty"`
	BaselinePath    string   `json:"baselinePath,omitempty"`
	Registered      []string `json:"registered,omitempty"`
	Required        []string `json:"required,omitempty"`
	Optional        []string `json:"optional,omitempty"`
	MissingRequired []string `json:"missingRequired,omitempty"`
	MissingOptional []string `json:"missingOptional,omitempty"`
	Extra           []string `json:"extra,omitempty"`
	Status          string   `json:"status"`
	Message         string   `json:"message,omitempty"`
}

type adapterCheckReport struct {
	ProjectRoot   string             `json:"projectRoot"`
	PackageName   string             `json:"packageName"`
	FrameworkRoot string             `json:"frameworkRoot"`
	Server        *adapterSideReport `json:"server,omitempty"`
	Client        *adapterSideReport `json:"client,omitempty"`
}

var (
	adapterBindPattern             = regexp.MustCompile(`ctx\.(?:bindSingleton|bindInstance|bindFactory)\(\s*([A-Za-z_][A-Za-z0-9_]*)`)
	adapterTransportPattern        = regexp.MustCompile(`ctx\.bindMessagingTransport\s*\(`)
	adapterUseRuntimeBridgePattern = regexp.MustCompile(`ctx\.useRuntimeBridge\s*\(`)
)

var adapterOptionalTokens = map[string]map[string]struct{}{
	"server": {
		"IPedAppearanceServer": {},
	},
	"client": {
		"IClientLogConsole": {},
	},
}

func NewAdapterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adapter",
		Short: "Validate external adapter implementations",
		Long:  "Tools for checking that OpenCore external adapters implement the framework contract coverage expected by the runtime.",
	}

	cmd.AddCommand(newAdapterCheckCommand())
	return cmd
}

func newAdapterCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check adapter contract coverage",
		Long:  "Inspect the current adapter package and compare its registered framework contracts against the OpenCore runtime baseline.",
		RunE:  runAdapterCheck,
	}

	cmd.Flags().Bool("strict", false, "Treat optional contract gaps as failures")
	cmd.Flags().Bool("json", false, "Print machine-readable JSON output")

	return cmd
}

func runAdapterCheck(cmd *cobra.Command, args []string) error {
	report, err := inspectAdapterProject(".")
	if err != nil {
		return err
	}

	strict, _ := cmd.Flags().GetBool("strict")
	asJSON, _ := cmd.Flags().GetBool("json")

	if asJSON {
		payload, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
	} else {
		renderAdapterCheckReport(report, strict)
	}

	if report.hasFailures(strict) {
		return fmt.Errorf("adapter contract check failed")
	}

	return nil
}

func inspectAdapterProject(projectRoot string) (*adapterCheckReport, error) {
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}

	manifest, err := readAdapterPackageManifest(projectRoot)
	if err != nil {
		return nil, err
	}

	if !looksLikeAdapterPackage(manifest.Name) {
		return nil, fmt.Errorf("current directory is not an adapter package (package name: %q)", manifest.Name)
	}

	frameworkRoot, err := resolveFrameworkRoot(projectRoot, manifest)
	if err != nil {
		return nil, err
	}

	report := &adapterCheckReport{
		ProjectRoot:   projectRoot,
		PackageName:   manifest.Name,
		FrameworkRoot: frameworkRoot,
	}

	report.Server = inspectAdapterSide(projectRoot, frameworkRoot, "server", exportExpected(manifest, "./server"))
	report.Client = inspectAdapterSide(projectRoot, frameworkRoot, "client", exportExpected(manifest, "./client"))

	return report, nil
}

func readAdapterPackageManifest(projectRoot string) (*adapterPackageManifest, error) {
	path := filepath.Join(projectRoot, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var manifest adapterPackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &manifest, nil
}

func looksLikeAdapterPackage(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return strings.Contains(name, "adapter")
}

func resolveFrameworkRoot(projectRoot string, manifest *adapterPackageManifest) (string, error) {
	candidates := []string{
		filepath.Join(projectRoot, "node_modules", "@open-core", "framework"),
		filepath.Join(projectRoot, "..", "opencore-framework"),
	}

	for _, depMap := range []map[string]string{manifest.DevDependencies, manifest.Dependencies, manifest.PeerDependencies} {
		if depMap == nil {
			continue
		}
		if ref, ok := depMap["@open-core/framework"]; ok {
			if strings.HasPrefix(ref, "file:") {
				candidates = append(candidates, filepath.Join(projectRoot, strings.TrimPrefix(ref, "file:")))
			}
		}
	}

	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, "package.json")); err == nil {
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not locate @open-core/framework from %s", projectRoot)
}

func exportExpected(manifest *adapterPackageManifest, key string) bool {
	if manifest == nil || len(manifest.Exports) == 0 {
		return true
	}
	_, ok := manifest.Exports[key]
	return ok
}

func inspectAdapterSide(projectRoot, frameworkRoot, side string, expected bool) *adapterSideReport {
	report := &adapterSideReport{Name: side, Expected: expected, Status: "pass"}
	if !expected {
		report.Message = "package does not export this side"
		return report
	}

	baselinePath, err := existingPath(frameworkBaselineCandidates(frameworkRoot, side)...)
	if err != nil {
		report.Status = "fail"
		report.Message = err.Error()
		return report
	}
	report.BaselinePath = baselinePath
	baselineSource, err := os.ReadFile(baselinePath)
	if err != nil {
		report.Status = "fail"
		report.Message = fmt.Sprintf("failed to read framework baseline: %v", err)
		return report
	}

	factoryPath, err := adapterFactoryPath(projectRoot, side)
	if err != nil {
		report.Status = "fail"
		report.Message = err.Error()
		report.Required, report.Optional = classifyTokens(side, parseRegisteredTokens(string(baselineSource)))
		return report
	}
	report.FactoryPath = factoryPath

	adapterSource, err := os.ReadFile(factoryPath)
	if err != nil {
		report.Status = "fail"
		report.Message = fmt.Sprintf("failed to read adapter %s factory: %v", side, err)
		return report
	}

	baselineTokens := parseRegisteredTokens(string(baselineSource))
	adapterTokens := parseRegisteredTokens(string(adapterSource))

	report.Registered = adapterTokens
	report.Required, report.Optional = classifyTokens(side, baselineTokens)
	report.MissingRequired = diffTokens(report.Required, adapterTokens)
	report.MissingOptional = diffTokens(report.Optional, adapterTokens)
	report.Extra = diffTokens(adapterTokens, baselineTokens)
	report.Status = adapterSideStatus(report)
	report.Message = adapterSideMessage(report)

	return report
}

func frameworkBaselineCandidates(frameworkRoot, side string) []string {
	return []string{
		filepath.Join(frameworkRoot, "src", "runtime", side, "adapter", fmt.Sprintf("node-%s-adapter.ts", side)),
		filepath.Join(frameworkRoot, "dist", "runtime", side, "adapter", fmt.Sprintf("node-%s-adapter.js", side)),
	}
}

func adapterFactoryPath(projectRoot, side string) (string, error) {
	patterns := []string{
		filepath.Join(projectRoot, "src", side, fmt.Sprintf("create-*-%s-adapter.ts", side)),
		filepath.Join(projectRoot, "dist", side, fmt.Sprintf("create-*-%s-adapter.js", side)),
	}

	for _, pattern := range patterns {
		found, err := filepath.Glob(pattern)
		if err != nil {
			return "", err
		}
		if len(found) > 0 {
			sort.Strings(found)
			return found[0], nil
		}
	}
	return "", fmt.Errorf("could not find %s adapter factory under %s", side, filepath.Join(projectRoot, "src", side))
}

func existingPath(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not locate framework baseline in any expected path")
}

func parseRegisteredTokens(source string) []string {
	set := map[string]struct{}{}
	matches := adapterBindPattern.FindAllStringSubmatch(source, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		set[match[1]] = struct{}{}
	}

	if adapterTransportPattern.MatchString(source) {
		set["MessagingTransport"] = struct{}{}
		set["EventsAPI"] = struct{}{}
		set["RpcAPI"] = struct{}{}
	}

	if adapterUseRuntimeBridgePattern.MatchString(source) {
		set["IClientRuntimeBridge"] = struct{}{}
	}

	return sortedTokenSet(set)
}

func classifyTokens(side string, baseline []string) (required []string, optional []string) {
	optionalSet := adapterOptionalTokens[side]
	for _, token := range baseline {
		if _, ok := optionalSet[token]; ok {
			optional = append(optional, token)
			continue
		}
		required = append(required, token)
	}
	return required, optional
}

func diffTokens(left, right []string) []string {
	rightSet := map[string]struct{}{}
	for _, token := range right {
		rightSet[token] = struct{}{}
	}

	var diff []string
	for _, token := range left {
		if _, ok := rightSet[token]; !ok {
			diff = append(diff, token)
		}
	}
	return diff
}

func sortedTokenSet(set map[string]struct{}) []string {
	items := make([]string, 0, len(set))
	for token := range set {
		items = append(items, token)
	}
	sort.Strings(items)
	return items
}

func adapterSideStatus(report *adapterSideReport) string {
	if report == nil || !report.Expected {
		return "pass"
	}
	if len(report.MissingRequired) > 0 {
		return "fail"
	}
	if len(report.MissingOptional) > 0 {
		return "warn"
	}
	return "pass"
}

func adapterSideMessage(report *adapterSideReport) string {
	if report == nil {
		return ""
	}
	if report.Message != "" && len(report.Registered) == 0 {
		return report.Message
	}

	parts := make([]string, 0, 4)
	parts = append(parts, fmt.Sprintf("registered=%d", len(report.Registered)))
	if len(report.MissingRequired) > 0 {
		parts = append(parts, fmt.Sprintf("missing required=%d", len(report.MissingRequired)))
	}
	if len(report.MissingOptional) > 0 {
		parts = append(parts, fmt.Sprintf("missing optional=%d", len(report.MissingOptional)))
	}
	if len(report.Extra) > 0 {
		parts = append(parts, fmt.Sprintf("extra=%d", len(report.Extra)))
	}
	return strings.Join(parts, " | ")
}

func (r *adapterCheckReport) hasFailures(strict bool) bool {
	for _, side := range []*adapterSideReport{r.Server, r.Client} {
		if side == nil || !side.Expected {
			continue
		}
		if len(side.MissingRequired) > 0 {
			return true
		}
		if strict && len(side.MissingOptional) > 0 {
			return true
		}
		if side.Status == "fail" && len(side.MissingRequired) == 0 {
			return true
		}
	}
	return false
}

func renderAdapterCheckReport(report *adapterCheckReport, strict bool) {
	modeLabel := "compat"
	if strict {
		modeLabel = "strict"
	}

	fmt.Println(ui.TitleStyle.Render("Adapter Check"))
	fmt.Println()
	fmt.Println(ui.Info(fmt.Sprintf("Package: %s", report.PackageName)))
	fmt.Println(ui.Muted(fmt.Sprintf("Project: %s", report.ProjectRoot)))
	fmt.Println(ui.Muted(fmt.Sprintf("Framework: %s", report.FrameworkRoot)))
	fmt.Println(ui.Muted(fmt.Sprintf("Mode: %s", modeLabel)))
	fmt.Println()

	renderAdapterSide(report.Server, strict)
	renderAdapterSide(report.Client, strict)

	if report.hasFailures(strict) {
		fmt.Println(ui.ErrorBoxStyle.Render("Adapter contract check failed."))
		return
	}

	if hasWarnings(report) {
		fmt.Println(ui.Warning("Adapter contract check passed with warnings."))
		return
	}

	fmt.Println(ui.SuccessBoxStyle.Render("All adapter contract checks passed."))
}

func renderAdapterSide(report *adapterSideReport, strict bool) {
	if report == nil {
		return
	}

	statusLabel := ui.Success("PASS")
	if report.Status == "warn" {
		if strict {
			statusLabel = ui.Error("FAIL")
		} else {
			statusLabel = ui.Warning("WARN")
		}
	}
	if report.Status == "fail" {
		statusLabel = ui.Error("FAIL")
	}

	if !report.Expected {
		statusLabel = ui.Muted("SKIP")
	}

	fmt.Printf("%s %s\n", statusLabel, strings.ToUpper(report.Name))
	if report.FactoryPath != "" {
		fmt.Println(ui.Muted(fmt.Sprintf("  factory: %s", report.FactoryPath)))
	}
	if report.Message != "" {
		fmt.Println(ui.Muted(fmt.Sprintf("  %s", report.Message)))
	}
	if len(report.MissingRequired) > 0 {
		fmt.Println(ui.Error(fmt.Sprintf("  missing required: %s", strings.Join(report.MissingRequired, ", "))))
	}
	if len(report.MissingOptional) > 0 {
		fmt.Println(ui.Warning(fmt.Sprintf("  missing optional: %s", strings.Join(report.MissingOptional, ", "))))
	}
	if len(report.Extra) > 0 {
		fmt.Println(ui.Muted(fmt.Sprintf("  extra bindings: %s", strings.Join(report.Extra, ", "))))
	}
	fmt.Println()
}

func hasWarnings(report *adapterCheckReport) bool {
	for _, side := range []*adapterSideReport{report.Server, report.Client} {
		if side != nil && len(side.MissingOptional) > 0 {
			return true
		}
	}
	return false
}
