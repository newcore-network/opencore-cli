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
		true,
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
