package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateStarterProjectWithFiveMAdapter(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-project")

	err := GenerateStarterProject(
		targetPath,
		"demo-project",
		false,
		"fivem",
		false,
		"C:/FXServer/resources",
		"pnpm@10.0.0",
	)
	if err != nil {
		t.Fatalf("GenerateStarterProject() error = %v", err)
	}

	configContent, err := os.ReadFile(filepath.Join(targetPath, "opencore.config.ts"))
	if err != nil {
		t.Fatalf("failed to read opencore.config.ts: %v", err)
	}

	configText := string(configContent)
	if !strings.Contains(configText, "FiveMServerAdapter") {
		t.Fatal("expected generated config to import FiveMServerAdapter")
	}
	if !strings.Contains(configText, "adapter: {") {
		t.Fatal("expected generated config to include central adapter block")
	}
	if !strings.Contains(configText, "client: FiveMClientAdapter()") {
		t.Fatal("expected generated config to include FiveM client adapter")
	}
	if strings.Contains(configText, "modules:") {
		t.Fatal("expected generated config to leave modules configuration out of starter template")
	}

	packageContent, err := os.ReadFile(filepath.Join(targetPath, "package.json"))
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}

	if !strings.Contains(string(packageContent), "\"@open-core/fivem-adapter\": \"latest\"") {
		t.Fatal("expected generated package.json to include @open-core/fivem-adapter")
	}
	if !strings.Contains(string(packageContent), "\"vite\": \"^7.1.0\"") {
		t.Fatal("expected generated package.json to include vite")
	}
	if !strings.Contains(string(packageContent), "\"postcss\": \"^8.5.6\"") {
		t.Fatal("expected generated package.json to include postcss")
	}

	if _, err := os.Stat(filepath.Join(targetPath, "core", "src", "features")); err != nil {
		t.Fatalf("expected core/src/features directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetPath, "core", "src", "server.ts")); err != nil {
		t.Fatalf("expected core/src/server.ts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetPath, "core", "src", "server", "main.ts")); !os.IsNotExist(err) {
		t.Fatal("did not expect legacy server/main.ts in starter")
	}
	if _, err := os.Stat(filepath.Join(targetPath, "vite.config.ts")); err != nil {
		t.Fatalf("expected root vite.config.ts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetPath, "postcss.config.mjs")); err != nil {
		t.Fatalf("expected root postcss.config.mjs: %v", err)
	}
}

func TestGenerateStarterProjectWithRageMPAdapter(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-ragemp")

	err := GenerateStarterProject(
		targetPath,
		"demo-ragemp",
		false,
		"ragemp",
		false,
		"C:/ragemp-server",
		"pnpm@10.0.0",
	)
	if err != nil {
		t.Fatalf("GenerateStarterProject() error = %v", err)
	}

	configContent, err := os.ReadFile(filepath.Join(targetPath, "opencore.config.ts"))
	if err != nil {
		t.Fatalf("failed to read opencore.config.ts: %v", err)
	}
	configText := string(configContent)
	if !strings.Contains(configText, "RageMPServerAdapter") {
		t.Fatal("expected generated config to import RageMPServerAdapter")
	}
	if !strings.Contains(configText, "server: {") || !strings.Contains(configText, "target: 'node14'") {
		t.Fatal("expected generated config to default server target to node14 for RageMP")
	}
	if !strings.Contains(configText, "client: {") || !strings.Contains(configText, "target: 'es2020'") {
		t.Fatal("expected generated config to default client target to es2020 for RageMP")
	}

	tsconfigContent, err := os.ReadFile(filepath.Join(targetPath, "tsconfig.json"))
	if err != nil {
		t.Fatalf("failed to read tsconfig.json: %v", err)
	}
	tsconfigText := string(tsconfigContent)
	if !strings.Contains(tsconfigText, "\"target\": \"es2020\"") {
		t.Fatal("expected RageMP tsconfig target es2020")
	}
	if !strings.Contains(tsconfigText, "\"module\": \"preserve\"") {
		t.Fatal("expected RageMP tsconfig module preserve")
	}
	if strings.Contains(tsconfigText, "@citizenfx") {
		t.Fatal("did not expect CitizenFX types in RageMP tsconfig")
	}
	if !strings.Contains(tsconfigText, "@ragempcommunity/types-server") || !strings.Contains(tsconfigText, "@ragempcommunity/types-client") {
		t.Fatal("expected RageMP tsconfig to include RageMP type packages")
	}

	packageContent, err := os.ReadFile(filepath.Join(targetPath, "package.json"))
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	packageText := string(packageContent)
	if !strings.Contains(packageText, "\"@open-core/ragemp-adapter\": \"latest\"") {
		t.Fatal("expected generated package.json to include @open-core/ragemp-adapter")
	}
	if strings.Contains(packageText, "@citizenfx/client") || strings.Contains(packageText, "@citizenfx/server") {
		t.Fatal("did not expect CitizenFX dependencies in RageMP starter")
	}
	if !strings.Contains(packageText, "\"@types/node\": \"^14.18.63\"") {
		t.Fatal("expected RageMP starter to include Node 14 type definitions")
	}
	if !strings.Contains(packageText, "\"@ragempcommunity/types-server\": \"latest\"") {
		t.Fatal("expected RageMP starter to include RageMP server types")
	}
	if !strings.Contains(packageText, "\"@ragempcommunity/types-client\": \"latest\"") {
		t.Fatal("expected RageMP starter to include RageMP client types")
	}

	if _, err := os.Stat(filepath.Join(targetPath, "core", "fxmanifest.lua")); !os.IsNotExist(err) {
		t.Fatal("did not expect core/fxmanifest.lua in RageMP starter")
	}
	if _, err := os.Stat(filepath.Join(targetPath, "core", "src", "features")); err != nil {
		t.Fatalf("expected core/src/features directory: %v", err)
	}
}

func TestGenerateResourceWithFiveMRuntime(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-resource")

	err := GenerateResource(targetPath, "demo-resource", true, false, ScaffoldRuntimeOptions{
		Runtime:      "fivem",
		ManifestKind: "fxmanifest",
	})
	if err != nil {
		t.Fatalf("GenerateResource() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetPath, "fxmanifest.lua")); err != nil {
		t.Fatalf("expected fxmanifest.lua for FiveM resource: %v", err)
	}

	packageContent, err := os.ReadFile(filepath.Join(targetPath, "package.json"))
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	packageText := string(packageContent)
	if !strings.Contains(packageText, "\"@citizenfx/server\": \"latest\"") || !strings.Contains(packageText, "\"@citizenfx/client\": \"latest\"") {
		t.Fatal("expected FiveM resource package.json to include CitizenFX dev dependencies")
	}
	if strings.Contains(packageText, "@ragempcommunity/types-server") {
		t.Fatal("did not expect RageMP types in FiveM resource package.json")
	}
}

func TestGenerateResourceWithRageMPRuntime(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-resource-ragemp")

	err := GenerateResource(targetPath, "demo-resource-ragemp", true, false, ScaffoldRuntimeOptions{
		Runtime:      "ragemp",
		ManifestKind: "none",
	})
	if err != nil {
		t.Fatalf("GenerateResource() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetPath, "fxmanifest.lua")); !os.IsNotExist(err) {
		t.Fatal("did not expect fxmanifest.lua for RageMP resource")
	}

	packageContent, err := os.ReadFile(filepath.Join(targetPath, "package.json"))
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	packageText := string(packageContent)
	if !strings.Contains(packageText, "\"@types/node\": \"^14.18.63\"") {
		t.Fatal("expected RageMP resource package.json to include Node types")
	}
	if !strings.Contains(packageText, "\"@ragempcommunity/types-server\": \"latest\"") || !strings.Contains(packageText, "\"@ragempcommunity/types-client\": \"latest\"") {
		t.Fatal("expected RageMP resource package.json to include RageMP type dependencies")
	}
	if strings.Contains(packageText, "@citizenfx") {
		t.Fatal("did not expect CitizenFX dependencies in RageMP resource package.json")
	}
}

func TestGenerateResourceWithRedMRuntime(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-resource-redm")

	err := GenerateResource(targetPath, "demo-resource-redm", true, false, ScaffoldRuntimeOptions{
		Runtime:      "redm",
		ManifestKind: "fxmanifest",
	})
	if err != nil {
		t.Fatalf("GenerateResource() error = %v", err)
	}

	manifestContent, err := os.ReadFile(filepath.Join(targetPath, "fxmanifest.lua"))
	if err != nil {
		t.Fatalf("failed to read fxmanifest.lua: %v", err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "game 'rdr3'") {
		t.Fatal("expected RedM resource manifest to target rdr3")
	}
	if !strings.Contains(manifestText, "rdr3_warning 'I acknowledge that this is a prerelease build of RedM") {
		t.Fatal("expected RedM resource manifest to include rdr3_warning")
	}
}

func TestGenerateStandaloneWithFiveMRuntime(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-standalone")

	err := GenerateStandalone(targetPath, "demo-standalone", true, false, ScaffoldRuntimeOptions{
		Runtime:      "fivem",
		ManifestKind: "fxmanifest",
	})
	if err != nil {
		t.Fatalf("GenerateStandalone() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetPath, "fxmanifest.lua")); err != nil {
		t.Fatalf("expected fxmanifest.lua for FiveM standalone: %v", err)
	}

	packageContent, err := os.ReadFile(filepath.Join(targetPath, "package.json"))
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	packageText := string(packageContent)
	if !strings.Contains(packageText, "\"@open-core/framework\": \"latest\"") {
		t.Fatal("expected standalone package.json to include @open-core/framework")
	}
	if !strings.Contains(packageText, "\"@citizenfx/server\": \"latest\"") || !strings.Contains(packageText, "\"@citizenfx/client\": \"latest\"") {
		t.Fatal("expected FiveM standalone package.json to include CitizenFX dev dependencies")
	}
}

func TestGenerateStandaloneWithRageMPRuntime(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-standalone-ragemp")

	err := GenerateStandalone(targetPath, "demo-standalone-ragemp", true, false, ScaffoldRuntimeOptions{
		Runtime:      "ragemp",
		ManifestKind: "none",
	})
	if err != nil {
		t.Fatalf("GenerateStandalone() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetPath, "fxmanifest.lua")); !os.IsNotExist(err) {
		t.Fatal("did not expect fxmanifest.lua for RageMP standalone")
	}
}

func TestGenerateStandaloneWithRedMRuntime(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-standalone-redm")

	err := GenerateStandalone(targetPath, "demo-standalone-redm", true, false, ScaffoldRuntimeOptions{
		Runtime:      "redm",
		ManifestKind: "fxmanifest",
	})
	if err != nil {
		t.Fatalf("GenerateStandalone() error = %v", err)
	}

	manifestContent, err := os.ReadFile(filepath.Join(targetPath, "fxmanifest.lua"))
	if err != nil {
		t.Fatalf("failed to read fxmanifest.lua: %v", err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "game 'rdr3'") {
		t.Fatal("expected RedM standalone manifest to target rdr3")
	}
	if !strings.Contains(manifestText, "rdr3_warning 'I acknowledge that this is a prerelease build of RedM") {
		t.Fatal("expected RedM standalone manifest to include rdr3_warning")
	}
}
