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
		"feature-based",
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
}

func TestGenerateStarterProjectWithRageMPAdapter(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "demo-ragemp")

	err := GenerateStarterProject(
		targetPath,
		"demo-ragemp",
		"feature-based",
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
	if !strings.Contains(configText, "target: 'node14'") {
		t.Fatal("expected generated config to default build target to node14 for RageMP")
	}

	tsconfigContent, err := os.ReadFile(filepath.Join(targetPath, "tsconfig.json"))
	if err != nil {
		t.Fatalf("failed to read tsconfig.json: %v", err)
	}
	tsconfigText := string(tsconfigContent)
	if !strings.Contains(tsconfigText, "\"target\": \"es2020\"") {
		t.Fatal("expected RageMP tsconfig target es2020")
	}
	if !strings.Contains(tsconfigText, "\"module\": \"commonjs\"") {
		t.Fatal("expected RageMP tsconfig module commonjs")
	}
	if strings.Contains(tsconfigText, "@citizenfx") {
		t.Fatal("did not expect CitizenFX types in RageMP tsconfig")
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
}
