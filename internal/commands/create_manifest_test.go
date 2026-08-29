package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveManifestCreateTargetRequiresExactlyOneSelector(t *testing.T) {
	if _, err := resolveManifestCreateTarget("chat", "utils", false); err == nil {
		t.Fatal("expected selector validation error")
	}
	if _, err := resolveManifestCreateTarget("", "", false); err == nil {
		t.Fatal("expected missing selector validation error")
	}
}

func TestResolveManifestCreateTargetRejectsTraversal(t *testing.T) {
	for _, name := range []string{"../chat", "chat/name", `chat\name`, " chat "} {
		if _, err := resolveManifestCreateTarget(name, "", false); err == nil {
			t.Errorf("expected resource name %q to be rejected", name)
		}
	}
}

func TestRunCreateManifestResource(t *testing.T) {
	tmpDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Fatalf("failed to restore wd: %v", chdirErr)
		}
	}()

	if err := os.MkdirAll(filepath.Join(tmpDir, "resources", "chat"), 0755); err != nil {
		t.Fatalf("failed to create resource dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	if err := runCreateManifest("chat", "", false, false); err != nil {
		t.Fatalf("runCreateManifest() error = %v", err)
	}

	manifestPath := filepath.Join(tmpDir, "resources", "chat", ocManifestFileName)
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read generated manifest: %v", err)
	}

	var manifest templateManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("failed to decode generated manifest: %v", err)
	}

	if manifest.Schema != ocManifestSchemaURL {
		t.Fatalf("expected schema URL %q, got %q", ocManifestSchemaURL, manifest.Schema)
	}
	if manifest.Kind != "resource" {
		t.Fatalf("expected resource kind, got %q", manifest.Kind)
	}
	if manifest.Name != "chat" {
		t.Fatalf("expected name chat, got %q", manifest.Name)
	}
	if manifest.Requires == nil || len(manifest.Requires.Templates) != 1 || manifest.Requires.Templates[0] != "core" {
		t.Fatalf("expected resource manifest to require core, got %#v", manifest.Requires)
	}
	if manifest.Compatibility == nil || len(manifest.Compatibility.Runtimes) != 1 || manifest.Compatibility.Runtimes[0] != "fivem" {
		t.Fatalf("expected default fivem runtime, got %#v", manifest.Compatibility)
	}
}

func TestRunCreateManifestCore(t *testing.T) {
	tmpDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Fatalf("failed to restore wd: %v", chdirErr)
		}
	}()

	if err := os.MkdirAll(filepath.Join(tmpDir, "core"), 0755); err != nil {
		t.Fatalf("failed to create core dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	if err := runCreateManifest("", "", true, false); err != nil {
		t.Fatalf("runCreateManifest() error = %v", err)
	}

	manifestPath := filepath.Join(tmpDir, "core", ocManifestFileName)
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read generated manifest: %v", err)
	}

	var manifest templateManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("failed to decode generated manifest: %v", err)
	}

	if manifest.Kind != "core" {
		t.Fatalf("expected core kind, got %q", manifest.Kind)
	}
	if manifest.Requires != nil {
		t.Fatalf("did not expect core manifest requires, got %#v", manifest.Requires)
	}
}

func TestRunCreateManifestDoesNotOverwriteWithoutForce(t *testing.T) {
	tmpDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	target := filepath.Join(tmpDir, "resources", "chat")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(target, ocManifestFileName)
	original := []byte("do not overwrite")
	if err := os.WriteFile(manifestPath, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := runCreateManifest("chat", "", false, false); err == nil {
		t.Fatal("expected existing manifest error")
	}
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(original) {
		t.Fatalf("existing manifest was modified: %q", body)
	}
}
