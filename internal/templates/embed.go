package templates

import (
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:starter-project
//go:embed all:resource
//go:embed all:standalone
//go:embed all:feature
var templatesFS embed.FS

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
		config.TSConfigModule = "commonjs"
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

	// Create base directories
	dirs := []string{
		targetPath,
		filepath.Join(targetPath, "core"),
		filepath.Join(targetPath, "core", "src"),
		filepath.Join(targetPath, "core", "src", "features"),
		filepath.Join(targetPath, "views"),
		filepath.Join(targetPath, "resources"),
		filepath.Join(targetPath, "standalones"),
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
		"core/package.json":   filepath.Join(targetPath, "core", "package.json"),
		"tsconfig.json":       filepath.Join(targetPath, "tsconfig.json"),
		".gitignore":          filepath.Join(targetPath, ".gitignore"),
	}
	if adapter != "ragemp" {
		files["core/fxmanifest.lua"] = filepath.Join(targetPath, "core", "fxmanifest.lua")
	}

	files["core/src/server.ts"] = filepath.Join(targetPath, "core", "src", "server.ts")
	files["core/src/client.ts"] = filepath.Join(targetPath, "core", "src", "client.ts")

	for tplFile, targetFile := range files {
		// Use forward slashes for embed.FS (works on all platforms)
		embedPath := path.Join("starter-project", tplFile)
		content, err := templatesFS.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tplFile, err)
		}

		tmpl, err := template.New(tplFile).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tplFile, err)
		}

		f, err := os.Create(targetFile)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetFile, err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, config); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tplFile, err)
		}
	}

	return nil
}

func GenerateResource(targetPath, resourceName string, hasClient, hasNUI bool, opts ScaffoldRuntimeOptions) error {
	config := resourceTemplateConfig(resourceName, hasClient, hasNUI, opts)

	// Create directories
	dirs := []string{
		targetPath,
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
		// Use forward slashes for embed.FS (works on all platforms)
		embedPath := path.Join("resource", tplFile)
		content, err := templatesFS.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tplFile, err)
		}

		tmpl, err := template.New(tplFile).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tplFile, err)
		}

		f, err := os.Create(targetFile)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetFile, err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, config); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tplFile, err)
		}
	}

	return nil
}

// GenerateStandalone generates a new standalone resource from templates.
func GenerateStandalone(targetPath, standaloneName string, hasClient, hasNUI bool, opts ScaffoldRuntimeOptions) error {
	config := standaloneTemplateConfig(standaloneName, hasClient, hasNUI, opts)

	// Create directories
	dirs := []string{
		targetPath,
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
		content, err := templatesFS.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tplFile, err)
		}

		tmpl, err := template.New(tplFile).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tplFile, err)
		}

		f, err := os.Create(targetFile)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetFile, err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, config); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tplFile, err)
		}
	}

	return nil
}

func GenerateFeature(targetPath, featureName string) error {
	pascalCase := toPascalCase(featureName)
	config := FeatureConfig{
		FeatureName:       featureName,
		FeatureNamePascal: pascalCase,
	}

	// Create feature directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	// Generate files
	files := map[string]string{
		"controller.ts": filepath.Join(targetPath, featureName+".controller.ts"),
		"service.ts":    filepath.Join(targetPath, featureName+".service.ts"),
		"index.ts":      filepath.Join(targetPath, "index.ts"),
	}

	for tplFile, targetFile := range files {
		// Use forward slashes for embed.FS (works on all platforms)
		embedPath := path.Join("feature", tplFile)
		content, err := templatesFS.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tplFile, err)
		}

		tmpl, err := template.New(tplFile).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tplFile, err)
		}

		f, err := os.Create(targetFile)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetFile, err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, config); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", tplFile, err)
		}
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
