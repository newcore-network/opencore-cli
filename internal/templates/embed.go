package templates

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed all:starter-project
//go:embed all:resource
//go:embed all:standalone
//go:embed all:feature
var templatesFS embed.FS

var scaffoldNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidateName applies the single naming policy used by scaffold commands and manifests.
func ValidateName(name string) error {
	if !scaffoldNamePattern.MatchString(name) {
		return fmt.Errorf("name must match %s", scaffoldNamePattern.String())
	}
	return nil
}

var templateFunctions = template.FuncMap{
	"jsonString": func(value string) (string, error) {
		encoded, err := json.Marshal(value)
		return string(encoded), err
	},
	"luaString": func(value string) string {
		replacer := strings.NewReplacer(
			`\`, `\\`,
			`'`, `\'`,
			"\r", `\r`,
			"\n", `\n`,
			"\x00", `\0`,
		)
		return "'" + replacer.Replace(value) + "'"
	},
}

type ProjectConfig struct {
	ProjectName          string
	InstallIdentity      bool
	Adapter              string
	InstallFiveMAdapter  bool
	InstallRageMPAdapter bool
	UseMinify            bool
	Destination          string
	PackageManager       string
	ManifestGame         string
	AddRedMWarning       bool
}

type ResourceConfig struct {
	ResourceName       string
	HasClient          bool
	HasNUI             bool
	Runtime            string
	ManifestKind       string
	GenerateManifest   bool
	UseNodeTypes       bool
	UseCitizenFXTypes  bool
	UseRageMPTypes     bool
	TSConfigTarget     string
	TSConfigModule     string
	TSModuleResolution string
	ManifestGame       string
	AddRedMWarning     bool
}

type StandaloneConfig struct {
	StandaloneName     string
	HasClient          bool
	HasNUI             bool
	Runtime            string
	ManifestKind       string
	GenerateManifest   bool
	UseNodeTypes       bool
	UseCitizenFXTypes  bool
	UseRageMPTypes     bool
	TSConfigTarget     string
	TSConfigModule     string
	TSModuleResolution string
	ManifestGame       string
	AddRedMWarning     bool
}

type ScaffoldRuntimeOptions struct {
	Runtime      string
	ManifestKind string
}

func normalizeScaffoldRuntimeOptions(opts ScaffoldRuntimeOptions) ScaffoldRuntimeOptions {
	runtime := strings.ToLower(strings.TrimSpace(opts.Runtime))
	if runtime == "" {
		runtime = "fivem"
	}

	manifestKind := strings.ToLower(strings.TrimSpace(opts.ManifestKind))
	if manifestKind == "" {
		if runtime == "ragemp" {
			manifestKind = "none"
		} else {
			manifestKind = "fxmanifest"
		}
	}

	return ScaffoldRuntimeOptions{
		Runtime:      runtime,
		ManifestKind: manifestKind,
	}
}

func resourceTemplateConfig(resourceName string, hasClient, hasNUI bool, opts ScaffoldRuntimeOptions) ResourceConfig {
	normalized := normalizeScaffoldRuntimeOptions(opts)
	config := ResourceConfig{
		ResourceName:       resourceName,
		HasClient:          hasClient,
		HasNUI:             hasNUI,
		Runtime:            normalized.Runtime,
		ManifestKind:       normalized.ManifestKind,
		GenerateManifest:   normalized.ManifestKind == "fxmanifest",
		TSConfigTarget:     "ES2022",
		TSConfigModule:     "preserve",
		TSModuleResolution: "bundler",
		ManifestGame:       "gta5",
	}

	if normalized.Runtime == "ragemp" {
		config.UseNodeTypes = true
		config.UseRageMPTypes = true
		config.TSConfigTarget = "es2020"
		config.TSConfigModule = "preserve"
		config.TSModuleResolution = "node"
	} else {
		config.UseCitizenFXTypes = true
	}
	if normalized.Runtime == "redm" {
		config.ManifestGame = "rdr3"
		config.AddRedMWarning = true
	}

	return config
}

func standaloneTemplateConfig(standaloneName string, hasClient, hasNUI bool, opts ScaffoldRuntimeOptions) StandaloneConfig {
	normalized := normalizeScaffoldRuntimeOptions(opts)
	config := StandaloneConfig{
		StandaloneName:     standaloneName,
		HasClient:          hasClient,
		HasNUI:             hasNUI,
		Runtime:            normalized.Runtime,
		ManifestKind:       normalized.ManifestKind,
		GenerateManifest:   normalized.ManifestKind == "fxmanifest",
		TSConfigTarget:     "ES2022",
		TSConfigModule:     "preserve",
		TSModuleResolution: "bundler",
		ManifestGame:       "gta5",
	}

	if normalized.Runtime == "ragemp" {
		config.UseNodeTypes = true
		config.UseRageMPTypes = true
		config.TSConfigTarget = "es2020"
		config.TSConfigModule = "preserve"
		config.TSModuleResolution = "node"
	} else {
		config.UseCitizenFXTypes = true
	}
	if normalized.Runtime == "redm" {
		config.ManifestGame = "rdr3"
		config.AddRedMWarning = true
	}

	return config
}

type FeatureConfig struct {
	FeatureName       string
	FeatureNamePascal string
}

func GenerateStarterProject(targetPath, projectName string, installIdentity bool, adapter string, useMinify bool, destination string, packageManager string) error {
	if err := ValidateName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}
	if destination != "" {
		// Ensure the generated TypeScript config is safe on Windows.
		// Backslashes can be interpreted as escape sequences in JS/TS strings.
		destination = strings.ReplaceAll(destination, "\\", "/")
	}

	installFiveMAdapter := adapter == "fivem"

	config := ProjectConfig{
		ProjectName:          projectName,
		InstallIdentity:      installIdentity,
		Adapter:              adapter,
		InstallFiveMAdapter:  installFiveMAdapter,
		InstallRageMPAdapter: adapter == "ragemp",
		UseMinify:            useMinify,
		Destination:          destination,
		PackageManager:       packageManager,
		ManifestGame:         "gta5",
	}
	if adapter == "redm" {
		config.ManifestGame = "rdr3"
		config.AddRedMWarning = true
	}
	cleanup, err := createTargetDirectory(targetPath)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			cleanup()
		}
	}()

	// Create base directories
	dirs := []string{
		filepath.Join(targetPath, "core"),
		filepath.Join(targetPath, "core", "src"),
		filepath.Join(targetPath, "core", "src", "features"),
		filepath.Join(targetPath, "views"),
		filepath.Join(targetPath, "resources"),
		filepath.Join(targetPath, "standalones"),
		filepath.Join(targetPath, "environments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Generate files from templates
	files := map[string]string{
		"package.json":        filepath.Join(targetPath, "package.json"),
		"opencore.config.ts":  filepath.Join(targetPath, "opencore.config.ts"),
		"pnpm-workspace.yaml": filepath.Join(targetPath, "pnpm-workspace.yaml"),
		"vite.config.ts":      filepath.Join(targetPath, "vite.config.ts"),
		"core/package.json":   filepath.Join(targetPath, "core", "package.json"),
		"tsconfig.json":       filepath.Join(targetPath, "tsconfig.json"),
		".gitignore":          filepath.Join(targetPath, ".gitignore"),
	}
	if adapter != "ragemp" {
		files["core/fxmanifest.lua"] = filepath.Join(targetPath, "core", "fxmanifest.lua")
	}

	files["core/src/server.ts"] = filepath.Join(targetPath, "core", "src", "server.ts")
	files["core/src/client.ts"] = filepath.Join(targetPath, "core", "src", "client.ts")
	files["environments/environment.model.ts"] = filepath.Join(targetPath, "environments", "environment.model.ts")
	files["environments/environment.development.ts"] = filepath.Join(targetPath, "environments", "environment.development.ts")
	files["environments/environment.production.ts"] = filepath.Join(targetPath, "environments", "environment.production.ts")

	for tplFile, targetFile := range files {
		// Use forward slashes for embed.FS (works on all platforms)
		embedPath := path.Join("starter-project", tplFile)
		if err := renderTemplateFile(embedPath, tplFile, targetFile, config); err != nil {
			return err
		}
	}

	succeeded = true
	return nil
}

func GenerateResource(targetPath, resourceName string, hasClient, hasNUI bool, opts ScaffoldRuntimeOptions) error {
	if err := ValidateName(resourceName); err != nil {
		return fmt.Errorf("invalid resource name: %w", err)
	}
	config := resourceTemplateConfig(resourceName, hasClient, hasNUI, opts)
	cleanup, err := createTargetDirectory(targetPath)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			cleanup()
		}
	}()

	// Create directories
	dirs := []string{
		filepath.Join(targetPath, "src"),
		filepath.Join(targetPath, "src", "server"),
	}

	if hasClient {
		dirs = append(dirs, filepath.Join(targetPath, "src", "client"))
	}

	if hasNUI {
		dirs = append(dirs, filepath.Join(targetPath, "ui"))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Generate files
	files := map[string]string{
		"package.json":       filepath.Join(targetPath, "package.json"),
		"src/server/main.ts": filepath.Join(targetPath, "src", "server", "main.ts"),
	}

	if config.GenerateManifest {
		files["fxmanifest.lua"] = filepath.Join(targetPath, "fxmanifest.lua")
	}

	if hasClient {
		files["src/client/main.ts"] = filepath.Join(targetPath, "src", "client", "main.ts")
	}

	if hasNUI {
		files["ui/tsconfig.json"] = filepath.Join(targetPath, "ui", "tsconfig.json")
	}

	for tplFile, targetFile := range files {
		// Use forward slashes for embed.FS (works on all platforms)
		embedPath := path.Join("resource", tplFile)
		if err := renderTemplateFile(embedPath, tplFile, targetFile, config); err != nil {
			return err
		}
	}

	succeeded = true
	return nil
}

// GenerateStandalone generates a new standalone resource from templates.
func GenerateStandalone(targetPath, standaloneName string, hasClient, hasNUI bool, opts ScaffoldRuntimeOptions) error {
	if err := ValidateName(standaloneName); err != nil {
		return fmt.Errorf("invalid standalone name: %w", err)
	}
	config := standaloneTemplateConfig(standaloneName, hasClient, hasNUI, opts)
	cleanup, err := createTargetDirectory(targetPath)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			cleanup()
		}
	}()

	// Create directories
	dirs := []string{
		filepath.Join(targetPath, "src"),
		filepath.Join(targetPath, "src", "server"),
	}

	if hasClient {
		dirs = append(dirs, filepath.Join(targetPath, "src", "client"))
	}

	if hasNUI {
		dirs = append(dirs, filepath.Join(targetPath, "ui"))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Generate files
	files := map[string]string{
		"package.json":       filepath.Join(targetPath, "package.json"),
		"src/server/main.ts": filepath.Join(targetPath, "src", "server", "main.ts"),
	}

	if config.GenerateManifest {
		files["fxmanifest.lua"] = filepath.Join(targetPath, "fxmanifest.lua")
	}

	if hasClient {
		files["src/client/main.ts"] = filepath.Join(targetPath, "src", "client", "main.ts")
	}

	for tplFile, targetFile := range files {
		embedPath := path.Join("standalone", tplFile)
		if err := renderTemplateFile(embedPath, tplFile, targetFile, config); err != nil {
			return err
		}
	}

	succeeded = true
	return nil
}

func GenerateFeature(targetPath, featureName string) error {
	if err := ValidateName(featureName); err != nil {
		return fmt.Errorf("invalid feature name: %w", err)
	}
	pascalCase := toPascalCase(featureName)
	config := FeatureConfig{
		FeatureName:       featureName,
		FeatureNamePascal: pascalCase,
	}

	cleanup, err := createTargetDirectory(targetPath)
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			cleanup()
		}
	}()

	// Generate files
	files := map[string]string{
		"controller.ts": filepath.Join(targetPath, featureName+".controller.ts"),
		"service.ts":    filepath.Join(targetPath, featureName+".service.ts"),
		"index.ts":      filepath.Join(targetPath, "index.ts"),
	}

	for tplFile, targetFile := range files {
		// Use forward slashes for embed.FS (works on all platforms)
		embedPath := path.Join("feature", tplFile)
		if err := renderTemplateFile(embedPath, tplFile, targetFile, config); err != nil {
			return err
		}
	}

	succeeded = true
	return nil
}

func createTargetDirectory(targetPath string) (func(), error) {
	cleanTarget := filepath.Clean(targetPath)
	if cleanTarget == "." {
		return nil, fmt.Errorf("refusing to generate into the current directory")
	}
	if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directory: %w", err)
	}
	if err := os.Mkdir(cleanTarget, 0755); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("target path %q already exists", cleanTarget)
		}
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}
	return func() { _ = os.RemoveAll(cleanTarget) }, nil
}

func renderTemplateFile(embedPath, templateName, targetFile string, data any) error {
	content, err := templatesFS.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templateName, err)
	}
	tmpl, err := template.New(templateName).Funcs(templateFunctions).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}
	f, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetFile, err)
	}
	if err := tmpl.Execute(f, data); err != nil {
		_ = f.Close()
		_ = os.Remove(targetFile)
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %w", targetFile, err)
	}
	return nil
}

func toPascalCase(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, "")
}
