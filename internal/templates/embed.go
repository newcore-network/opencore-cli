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
//go:embed all:feature
//go:embed all:architectures
var templatesFS embed.FS

type ProjectConfig struct {
	ProjectName     string
	Architecture    string
	InstallIdentity bool
	UseMinify       bool
}

type ResourceConfig struct {
	ResourceName string
	HasClient    bool
	HasNUI       bool
}

type FeatureConfig struct {
	FeatureName       string
	FeatureNamePascal string
}

func GenerateStarterProject(targetPath, projectName, architecture string, installIdentity, useMinify bool) error {
	config := ProjectConfig{
		ProjectName:     projectName,
		Architecture:    architecture,
		InstallIdentity: installIdentity,
		UseMinify:       useMinify,
	}

	// Create base directories
	dirs := []string{
		targetPath,
		filepath.Join(targetPath, "core"),
		filepath.Join(targetPath, "core", "src"),
		filepath.Join(targetPath, "views"),
		filepath.Join(targetPath, "resources"),
	}

	// Add architecture-specific directories
	switch architecture {
	case "domain-driven":
		dirs = append(dirs,
			filepath.Join(targetPath, "core", "src", "modules"),
		)
	case "layer-based":
		// Layer-based needs client/server folders for controllers and services
		dirs = append(dirs,
			filepath.Join(targetPath, "core", "src", "client"),
			filepath.Join(targetPath, "core", "src", "client", "controllers"),
			filepath.Join(targetPath, "core", "src", "client", "services"),
			filepath.Join(targetPath, "core", "src", "server"),
			filepath.Join(targetPath, "core", "src", "server", "controllers"),
			filepath.Join(targetPath, "core", "src", "server", "services"),
			filepath.Join(targetPath, "core", "src", "shared"),
		)
	case "feature-based":
		dirs = append(dirs,
			filepath.Join(targetPath, "core", "src", "features"),
		)
	case "hybrid":
		dirs = append(dirs,
			filepath.Join(targetPath, "core", "src", "core-modules"),
			filepath.Join(targetPath, "core", "src", "features"),
		)
	default:
		// Fallback to feature-based
		dirs = append(dirs,
			filepath.Join(targetPath, "core", "src", "features"),
		)
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
		"core/fxmanifest.lua": filepath.Join(targetPath, "core", "fxmanifest.lua"),
		"tsconfig.json":       filepath.Join(targetPath, "tsconfig.json"),
	}

	// Add bootstrap files based on architecture
	if architecture == "layer-based" {
		// Layer-based uses main.ts inside folders
		files["core/src/server/main.ts"] = filepath.Join(targetPath, "core", "src", "server", "main.ts")
		files["core/src/client/main.ts"] = filepath.Join(targetPath, "core", "src", "client", "main.ts")
	} else {
		// All other architectures use client.ts and server.ts directly in src/
		files["core/src/server.ts"] = filepath.Join(targetPath, "core", "src", "server.ts")
		files["core/src/client.ts"] = filepath.Join(targetPath, "core", "src", "client.ts")
	}

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

func GenerateResource(targetPath, resourceName string, hasClient, hasNUI bool) error {
	config := ResourceConfig{
		ResourceName: resourceName,
		HasClient:    hasClient,
		HasNUI:       hasNUI,
	}

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
		"tsconfig.json":      filepath.Join(targetPath, "tsconfig.json"),
		"fxmanifest.lua":     filepath.Join(targetPath, "fxmanifest.lua"),
		"src/server/main.ts": filepath.Join(targetPath, "src", "server", "main.ts"),
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

type ModuleConfig struct {
	ModuleName       string
	ModuleNamePascal string
}

func GenerateModuleDomainDriven(targetPath, moduleName string) error {
	pascalCase := toPascalCase(moduleName)
	config := ModuleConfig{
		ModuleName:       moduleName,
		ModuleNamePascal: pascalCase,
	}

	// Create module structure
	dirs := []string{
		targetPath,
		filepath.Join(targetPath, "client"),
		filepath.Join(targetPath, "server"),
		filepath.Join(targetPath, "shared"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Generate files
	files := map[string]string{
		"module-client-controller.ts": filepath.Join(targetPath, "client", moduleName+".controller.ts"),
		"module-client-ui.ts":         filepath.Join(targetPath, "client", moduleName+".ui.ts"),
		"module-server-controller.ts": filepath.Join(targetPath, "server", moduleName+".controller.ts"),
		"module-server-service.ts":    filepath.Join(targetPath, "server", moduleName+".service.ts"),
		"module-server-repository.ts": filepath.Join(targetPath, "server", moduleName+".repository.ts"),
		"module-shared-types.ts":      filepath.Join(targetPath, "shared", moduleName+".types.ts"),
		"module-shared-events.ts":     filepath.Join(targetPath, "shared", moduleName+".events.ts"),
	}

	for tplFile, targetFile := range files {
		embedPath := path.Join("architectures", "domain-driven", tplFile)
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

func GenerateLayerBased(clientPath, serverPath, servicePath, featureName string) error {
	pascalCase := toPascalCase(featureName)
	config := FeatureConfig{
		FeatureName:       featureName,
		FeatureNamePascal: pascalCase,
	}

	// Generate files
	files := map[string]string{
		"client-controller.ts": filepath.Join(clientPath, featureName+".controller.ts"),
		"client-service.ts":    filepath.Join(clientPath, "..", "services", featureName+".client.service.ts"),
		"server-controller.ts": filepath.Join(serverPath, featureName+".controller.ts"),
		"server-service.ts":    filepath.Join(servicePath, featureName+".service.ts"),
	}

	for tplFile, targetFile := range files {
		embedPath := path.Join("architectures", "layer-based", tplFile)
		content, err := templatesFS.ReadFile(embedPath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", tplFile, err)
		}

		tmpl, err := template.New(tplFile).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", tplFile, err)
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetFile, err)
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
