package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func findNodeModulesRoot(t *testing.T) (string, bool) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}

	cur := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(cur, "node_modules")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	return "", false
}

func linkProjectDependency(t *testing.T, nodeModulesRoot string, projectNodeModules string, packagePath string) bool {
	t.Helper()

	sourcePath := filepath.Join(nodeModulesRoot, filepath.FromSlash(packagePath))
	if _, err := os.Stat(sourcePath); err != nil {
		return false
	}

	targetPath := filepath.Join(projectNodeModules, filepath.FromSlash(packagePath))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", packagePath, err)
	}
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		t.Fatalf("failed to symlink %s: %v", packagePath, err)
	}

	return true
}

func TestDetectScaffoldRuntimeOptionsDefaultsWithoutConfig(t *testing.T) {
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

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	options := detectScaffoldRuntimeOptions()
	if options.Runtime != "fivem" {
		t.Fatalf("expected default runtime fivem, got %q", options.Runtime)
	}
	if options.ManifestKind != "fxmanifest" {
		t.Fatalf("expected default manifest fxmanifest, got %q", options.ManifestKind)
	}
}

func TestDetectScaffoldRuntimeOptionsFromRageMPConfig(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("skipping: node is not installed")
	}

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

	configText := `import { defineConfig } from '@open-core/cli'

export default defineConfig({
  name: 'demo',
  destination: './build',
  adapter: {
    server: {
      name: 'ragemp',
      register() {},
      runtime: {
        runtime: 'ragemp',
        manifest: { kind: 'none' },
      },
    },
    client: {
      name: 'ragemp',
      register() {},
      runtime: {
        runtime: 'ragemp',
      },
    },
  },
  core: {
    path: './core',
    resourceName: 'core',
  },
  resources: {
    include: ['./resources/*'],
  },
})
`

	if err := os.WriteFile(filepath.Join(tmpDir, "opencore.config.ts"), []byte(configText), 0644); err != nil {
		t.Fatalf("failed to write opencore.config.ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"demo","private":true}`), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
	sharedNodeModulesRoot, ok := findNodeModulesRoot(t)
	if !ok {
		t.Skip("skipping: node_modules not available for config loading test")
	}

	projectNodeModules := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(filepath.Join(projectNodeModules, "@open-core", "cli"), 0755); err != nil {
		t.Fatalf("failed to create temp node_modules: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectNodeModules, "@open-core", "cli", "package.json"),
		[]byte(`{"name":"@open-core/cli","main":"./index.js"}`),
		0644,
	); err != nil {
		t.Fatalf("failed to write temp @open-core/cli package.json: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectNodeModules, "@open-core", "cli", "index.js"),
		[]byte("module.exports = { defineConfig: (config) => config }\n"),
		0644,
	); err != nil {
		t.Fatalf("failed to write temp @open-core/cli index.js: %v", err)
	}

	if !linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "esbuild") {
		t.Skip("skipping: esbuild not available for config loading test")
	}
	_ = linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "reflect-metadata")
	_ = linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "typescript")
	_ = linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "@swc/core")
	_ = linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "tsconfig-paths")
	if !linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "@esbuild/linux-x64") {
		_ = linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "@esbuild/darwin-arm64")
		_ = linkProjectDependency(t, sharedNodeModulesRoot, projectNodeModules, "@esbuild/darwin-x64")
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	options := detectScaffoldRuntimeOptions()
	if options.Runtime != "ragemp" {
		t.Fatalf("expected runtime ragemp, got %q", options.Runtime)
	}
	if options.ManifestKind != "none" {
		t.Fatalf("expected manifest none, got %q", options.ManifestKind)
	}
}
