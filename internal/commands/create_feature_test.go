package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCreateFeatureUsesDefaultCoreFeaturesPath(t *testing.T) {
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

	if err := os.MkdirAll(filepath.Join(tmpDir, "core", "src", "features"), 0755); err != nil {
		t.Fatalf("failed to create core/src/features: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	if err := runCreateFeature(nil, []string{"banking"}, ""); err != nil {
		t.Fatalf("runCreateFeature() error = %v", err)
	}

	featureDir := filepath.Join(tmpDir, "core", "src", "features", "banking")
	for _, file := range []string{"banking.controller.ts", "banking.service.ts", "index.ts"} {
		if _, err := os.Stat(filepath.Join(featureDir, file)); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "core", "src", "modules", "banking")); !os.IsNotExist(err) {
		t.Fatal("did not expect legacy modules path to be used")
	}
}
