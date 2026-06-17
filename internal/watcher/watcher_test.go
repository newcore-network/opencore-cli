package watcher

import (
	"path/filepath"
	"testing"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestShouldIgnorePathForAutoloadArtifacts(t *testing.T) {
	w := &Watcher{}

	if !w.shouldIgnorePath(filepath.Join("resources", "sample-resource", ".opencore", "autoload.client.controllers.ts")) {
		t.Fatalf("expected .opencore autoload file to be ignored")
	}
}

func TestShouldIgnorePathForOutputAndDestination(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "build")
	destDir := filepath.Join(tmp, "deploy")
	w := &Watcher{
		config: &config.Config{
			OutDir:      outDir,
			Destination: destDir,
		},
	}

	if !w.shouldIgnorePath(filepath.Join(outDir, "resource", "core", "server.js")) {
		t.Fatalf("expected outDir path to be ignored")
	}
	if !w.shouldIgnorePath(filepath.Join(destDir, "resource", "core", "client.js")) {
		t.Fatalf("expected destination path to be ignored")
	}
}

func TestShouldIgnorePathDoesNotIgnoreSourceFiles(t *testing.T) {
	tmp := t.TempDir()
	w := &Watcher{
		config: &config.Config{
			OutDir:      filepath.Join(tmp, "build"),
			Destination: filepath.Join(tmp, "deploy"),
		},
	}

	if w.shouldIgnorePath(filepath.Join(tmp, "resources", "sample-resource", "src", "server", "main.ts")) {
		t.Fatalf("expected source file path to be watched")
	}
}
