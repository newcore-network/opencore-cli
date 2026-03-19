package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/newcore-network/opencore-cli/internal/ui"
)

const ocManifestSchemaURL = "https://opencorejs.dev/schemas/oc-manifest.schema.json"

type manifestCreateTarget struct {
	Kind string
	Name string
	Path string
}

func newCreateManifestCommand() *cobra.Command {
	var resourceName string
	var standaloneName string
	var core bool
	var force bool

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Create an oc.manifest.json example",
		Long: `Create an example oc.manifest.json for an existing core, resource, or standalone.

Examples:
  opencore create manifest --resource xchat
  opencore create manifest --standalone utils
  opencore create manifest --core
  opencore create manifest --resource xchat --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateManifest(resourceName, standaloneName, core, force)
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource", "", "Create a manifest for resources/<name>")
	cmd.Flags().StringVar(&standaloneName, "standalone", "", "Create a manifest for standalones/<name>")
	cmd.Flags().BoolVar(&core, "core", false, "Create a manifest for core/")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite oc.manifest.json if it already exists")

	return cmd
}

func runCreateManifest(resourceName, standaloneName string, core, force bool) error {
	fmt.Println(ui.TitleStyle.Render("Create Manifest"))
	fmt.Println()

	target, err := resolveManifestCreateTarget(resourceName, standaloneName, core)
	if err != nil {
		return err
	}

	if err := ensureManifestParentExists(target.Path); err != nil {
		return err
	}

	manifestPath := filepath.Join(target.Path, ocManifestFileName)
	if !force {
		if _, err := os.Stat(manifestPath); err == nil {
			return fmt.Errorf("%s already exists\n\nUse '--force' to overwrite it", manifestPath)
		}
	}

	manifest := buildExampleManifest(target)
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to build manifest JSON: %w", err)
	}
	body = append(body, '\n')

	if err := os.WriteFile(manifestPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", ocManifestFileName, err)
	}

	fmt.Println(ui.Success("Manifest created successfully!"))
	fmt.Println()
	renderCreateBox(
		fmt.Sprintf("Location: %s\n\n", manifestPath) +
			"The generated file is an example starting point.\n" +
			"Review compatibility and dependencies before publishing.",
	)

	return nil
}

func resolveManifestCreateTarget(resourceName, standaloneName string, core bool) (manifestCreateTarget, error) {
	targets := 0
	if strings.TrimSpace(resourceName) != "" {
		targets++
	}
	if strings.TrimSpace(standaloneName) != "" {
		targets++
	}
	if core {
		targets++
	}

	if targets != 1 {
		return manifestCreateTarget{}, fmt.Errorf("choose exactly one target: --resource <name>, --standalone <name>, or --core")
	}

	if strings.TrimSpace(resourceName) != "" {
		name := strings.TrimSpace(resourceName)
		if err := validateCreateName("resource")(name); err != nil {
			return manifestCreateTarget{}, err
		}
		return manifestCreateTarget{
			Kind: "resource",
			Name: name,
			Path: filepath.Join("resources", name),
		}, nil
	}

	if strings.TrimSpace(standaloneName) != "" {
		name := strings.TrimSpace(standaloneName)
		if err := validateCreateName("standalone")(name); err != nil {
			return manifestCreateTarget{}, err
		}
		return manifestCreateTarget{
			Kind: "standalone",
			Name: name,
			Path: filepath.Join("standalones", name),
		}, nil
	}

	return manifestCreateTarget{
		Kind: "core",
		Name: "core",
		Path: "core",
	}, nil
}

func ensureManifestParentExists(targetPath string) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target path '%s' does not exist", targetPath)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("target path '%s' is not a directory", targetPath)
	}
	return nil
}

func buildExampleManifest(target manifestCreateTarget) *templateManifest {
	compatibility := detectManifestCompatibilityDefaults()

	manifest := &templateManifest{
		Schema:      ocManifestSchemaURL,
		Version:     1,
		Name:        target.Name,
		DisplayName: target.Name,
		Kind:        target.Kind,
		Description: fmt.Sprintf("Example OpenCore %s manifest for %s", target.Kind, target.Name),
		Compatibility: &templateManifestCompatibility{
			Runtimes:     []string{compatibility.Runtime},
			GameProfiles: []string{compatibility.GameProfile},
		},
		Links: &templateManifestLinks{
			Readme: "./README.md",
		},
	}

	if target.Kind == "resource" {
		manifest.Requires = &templateManifestRequires{Templates: []string{"core"}}
	}

	return manifest
}
