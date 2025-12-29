package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestNewDeployer(t *testing.T) {
	cfg := &config.Config{
		OutDir:      "./dist",
		Destination: "C:/FXServer/resources",
	}

	deployer := NewDeployer(cfg)

	if deployer == nil {
		t.Fatal("NewDeployer returned nil")
	}

	if deployer.config != cfg {
		t.Error("Config not properly assigned")
	}
}

func TestHasDestination(t *testing.T) {
	tests := []struct {
		destination string
		expected    bool
	}{
		{"C:/FXServer/resources", true},
		{"./local/resources", true},
		{"", false},
	}

	for _, tt := range tests {
		cfg := &config.Config{
			Destination: tt.destination,
		}
		deployer := NewDeployer(cfg)

		result := deployer.HasDestination()
		if result != tt.expected {
			t.Errorf("HasDestination() with '%s' = %v, expected %v", tt.destination, result, tt.expected)
		}
	}
}

func TestDeploy(t *testing.T) {
	// Create temp directories
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create source structure
	outDir := filepath.Join(srcDir, "dist", "resources")
	coreDir := filepath.Join(outDir, "[core]")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some files
	if err := os.WriteFile(filepath.Join(coreDir, "server.js"), []byte("// server"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "client.js"), []byte("// client"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "fxmanifest.lua"), []byte("fx_version 'cerulean'"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create another resource
	adminDir := filepath.Join(outDir, "admin")
	if err := os.MkdirAll(adminDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adminDir, "server.js"), []byte("// admin server"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		OutDir:      filepath.Join(srcDir, "dist", "resources"),
		Destination: dstDir,
	}

	deployer := NewDeployer(cfg)
	err := deployer.Deploy()

	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify files were copied
	if _, err := os.Stat(filepath.Join(dstDir, "[core]", "server.js")); os.IsNotExist(err) {
		t.Error("core/server.js should be deployed")
	}

	if _, err := os.Stat(filepath.Join(dstDir, "[core]", "fxmanifest.lua")); os.IsNotExist(err) {
		t.Error("core/fxmanifest.lua should be deployed")
	}

	if _, err := os.Stat(filepath.Join(dstDir, "admin", "server.js")); os.IsNotExist(err) {
		t.Error("admin/server.js should be deployed")
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(dstDir, "[core]", "server.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "// server" {
		t.Errorf("server.js content mismatch: got '%s'", string(content))
	}
}

func TestDeployNoDestination(t *testing.T) {
	cfg := &config.Config{
		OutDir:      "./dist",
		Destination: "", // No destination
	}

	deployer := NewDeployer(cfg)
	err := deployer.Deploy()

	// When destination is not set, Deploy should return nil (skip silently)
	if err != nil {
		t.Errorf("Expected nil error when destination is not set, got: %v", err)
	}
}

func TestDeployNonExistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		OutDir:      "/nonexistent/path/dist",
		Destination: tmpDir,
	}

	deployer := NewDeployer(cfg)
	err := deployer.Deploy()

	if err == nil {
		t.Error("Expected error when source directory doesn't exist")
	}
}

func TestDeployCreatesDestination(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "new", "nested", "destination")

	// Create source structure
	outDir := filepath.Join(srcDir, "dist")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file
	if err := os.WriteFile(filepath.Join(outDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		OutDir:      outDir,
		Destination: dstDir,
	}

	deployer := NewDeployer(cfg)
	err := deployer.Deploy()

	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify destination was created
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Error("Destination directory should be created")
	}
}

func TestDeployPreservesStructure(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create nested structure
	outDir := filepath.Join(srcDir, "dist")
	nestedDir := filepath.Join(outDir, "resource", "deep", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(nestedDir, "deep.js"), []byte("// deep"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		OutDir:      outDir,
		Destination: dstDir,
	}

	deployer := NewDeployer(cfg)
	err := deployer.Deploy()

	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify nested structure was preserved
	expectedPath := filepath.Join(dstDir, "resource", "deep", "nested", "deep.js")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Nested file should exist at %s", expectedPath)
	}
}
