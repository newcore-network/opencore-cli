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
var templatesFS embed.FS

type ProjectConfig struct {
	ProjectName     string
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

func GenerateStarterProject(targetPath, projectName string, installIdentity, useMinify bool) error {
	config := ProjectConfig{
		ProjectName:     projectName,
		InstallIdentity: installIdentity,
		UseMinify:       useMinify,
	}

	// Create directories
	dirs := []string{
		targetPath,
		filepath.Join(targetPath, "core"),
		filepath.Join(targetPath, "core", "src"),
		filepath.Join(targetPath, "core", "src", "server"),
		filepath.Join(targetPath, "core", "src", "client"),
		filepath.Join(targetPath, "core", "src", "features"),
		filepath.Join(targetPath, "resources"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Generate files from templates
	files := map[string]string{
		"package.json":            filepath.Join(targetPath, "package.json"),
		"opencore.config.ts":      filepath.Join(targetPath, "opencore.config.ts"),
		"pnpm-workspace.yaml":     filepath.Join(targetPath, "pnpm-workspace.yaml"),
		"core/package.json":       filepath.Join(targetPath, "core", "package.json"),
		"core/tsconfig.json":      filepath.Join(targetPath, "core", "tsconfig.json"),
		"core/fxmanifest.lua":     filepath.Join(targetPath, "core", "fxmanifest.lua"),
		"core/src/server/main.ts": filepath.Join(targetPath, "core", "src", "server", "main.ts"),
		"core/src/client/main.ts": filepath.Join(targetPath, "core", "src", "client", "main.ts"),
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

func toPascalCase(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, "")
}
