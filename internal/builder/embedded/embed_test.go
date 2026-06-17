package embedded

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedBuildScript(t *testing.T) {
	// 1. Check entry script (build.js)
	entryScript := string(GetBuildScript())
	if !strings.Contains(entryScript, "buildSingle") {
		t.Error("build.js missing buildSingle")
	}
	if !strings.Contains(entryScript, "require('./build_functions')") {
		t.Error("build.js missing build_functions require")
	}

	// 2. Check plugins.js
	pluginsScript, _ := BuildFS.ReadFile("plugins.js")
	pluginsContent := string(pluginsScript)
	requiredPlugins := []string{
		"createSwcPlugin",
		"normalizeSwcTarget",
		"createExcludeNodeAdaptersPlugin",
		"preserveFiveMExportsPlugin",
		"findProjectConfigPath",
		"__openCoreProjectAdapter",
		"__openCoreUseAdapter",
	}
	for _, plugin := range requiredPlugins {
		if !strings.Contains(pluginsContent, plugin) {
			t.Errorf("plugins.js missing required plugin: %s", plugin)
		}
	}
	if !strings.Contains(pluginsContent, "require('esbuild')") {
		t.Error("plugins.js missing esbuild require")
	}
	if !strings.Contains(pluginsContent, "require('@swc/core')") {
		t.Error("plugins.js missing @swc/core require")
	}

	// 3. Check build_functions.js
	buildFuncsScript, _ := BuildFS.ReadFile("build_functions.js")
	buildFuncsContent := string(buildFuncsScript)
	requiredFunctions := []string{
		"buildCore",
		"buildResource",
		"buildStandalone",
		"serverBinaryPlatform",
		"function dependencyPlugin",
		"function optionsWithServerExternals",
		"function esbuildExternals",
		"createEnvironmentAliasPlugin",
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(buildFuncsContent, fn) {
			t.Errorf("build_functions.js missing required function: %s", fn)
		}
	}

	// 4. Check views.js
	viewsScript, _ := BuildFS.ReadFile("views.js")
	viewsContent := string(viewsScript)
	requiredViewsSymbols := []string{
		"forceInclude",
		"buildViteViews",
		"detectViteFramework",
		"supportedFrameworks = new Set(['', 'vite', 'vanilla'])",
		"Framework-specific CLI builders were removed",
	}
	for _, symbol := range requiredViewsSymbols {
		if !strings.Contains(viewsContent, symbol) {
			t.Errorf("views.js missing required symbol: %s", symbol)
		}
	}
	if strings.Contains(viewsContent, "const isVite = explicitFramework !== '' && detectViteFramework(viewPath)") {
		t.Error("views.js still contains the buggy Vite auto-detection condition")
	}
	if !strings.Contains(viewsContent, "const isVite = explicitFramework === 'vite' || (explicitFramework === '' && detectViteFramework(viewPath))") {
		t.Error("views.js missing the guarded Vite detection condition")
	}
	if !strings.Contains(viewsContent, "const absLocalViteConfig = localViteConfig ? path.resolve(localViteConfig).replace(/\\\\/g, '/') : null") {
		t.Error("views.js should normalize local vite config to an absolute path")
	}
	if !strings.Contains(viewsContent, "commandParts.push(`--config \"${absLocalViteConfig}\"`)") {
		t.Error("views.js should pass the absolute local vite config path to vite")
	}

	// 5. Check dependency installer layout for FXServer sandbox compatibility.
	depsScript, _ := BuildFS.ReadFile("dependencies.js")
	depsContent := string(depsScript)
	if !strings.Contains(depsContent, "--config.node-linker=hoisted") {
		t.Error("dependencies.js should force pnpm's hoisted node linker for resource-local transitive dependencies")
	}
	if !strings.Contains(depsContent, "nodeLinker: pm === 'pnpm' ? 'hoisted' : undefined") {
		t.Error("dependencies.js should include pnpm node linker mode in the dependency cache key")
	}
}

func TestBuildScriptNotEmpty(t *testing.T) {
	script := GetBuildScript()

	// The entry script itself is small now, but should still be valid
	minSize := 100 // bytes
	if len(script) < minSize {
		t.Errorf("Build script seems too small: %d bytes (expected at least %d)", len(script), minSize)
	}
}

func TestBuildScriptIsValidJS(t *testing.T) {
	content := string(GetBuildScript())

	// Basic syntax checks
	// Check for balanced braces (simple check - not perfect for strings/comments)
	openBraces := strings.Count(content, "{")
	closeBraces := strings.Count(content, "}")

	if openBraces != closeBraces {
		t.Errorf("Unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	// Check that it starts with valid JS (const, let, var, or a require)
	trimmed := strings.TrimSpace(content)
	validStarts := []string{"const ", "let ", "var ", "'use strict'", "\"use strict\""}
	hasValidStart := false
	for _, start := range validStarts {
		if strings.HasPrefix(trimmed, start) {
			hasValidStart = true
			break
		}
	}
	if !hasValidStart {
		t.Errorf("Script doesn't start with a valid JS statement, starts with: %.50s...", trimmed)
	}
}

func TestEmbeddedBuildResourceWithDependencyResolutionAndEnvironmentAliases(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("skipping: node is not installed")
	}

	repoRoot := findRepoRootWithNodeDeps(t)
	scriptDir := t.TempDir()
	entries, err := BuildFS.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read embedded scripts: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := BuildFS.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("failed to read embedded %s: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(scriptDir, entry.Name()), content, 0644); err != nil {
			t.Fatalf("failed to write embedded %s: %v", entry.Name(), err)
		}
	}

	resourceDir := filepath.Join(scriptDir, "resource")
	envPath := filepath.Join(resourceDir, "env.ts")
	if err := os.MkdirAll(filepath.Join(resourceDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "package.json"), []byte(`{"name":"test-resource","version":"1.0.0","private":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte(`export const environment = { name: 'merged-env' }`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "src", "server.ts"), []byte(`
import { environment } from '@opencore/environment'
console.log(environment.name)
`), 0644); err != nil {
		t.Fatal(err)
	}
	stubFiles := map[string]string{
		filepath.Join(scriptDir, "node_modules", "reflect-metadata", "package.json"):        `{"name":"reflect-metadata","version":"0.0.0","main":"index.js"}`,
		filepath.Join(scriptDir, "node_modules", "reflect-metadata", "index.js"):            `module.exports = {}`,
		filepath.Join(scriptDir, "node_modules", "@open-core", "framework", "package.json"): `{"name":"@open-core/framework","version":"0.0.0","exports":{"./server":"./server.js","./client":"./client.js"}}`,
		filepath.Join(scriptDir, "node_modules", "@open-core", "framework", "server.js"):    `exports.useAdapter = function () {}`,
		filepath.Join(scriptDir, "node_modules", "@open-core", "framework", "client.js"):    `exports.useAdapter = function () {}`,
	}
	for filePath, content := range stubFiles {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	runner := filepath.Join(scriptDir, "run.js")
	if err := os.WriteFile(runner, []byte(`
const fs = require('fs')
const path = require('path')
const { buildResource } = require('./build_functions')

async function main() {
  const scriptDir = process.argv[2]
  const resourceDir = path.join(scriptDir, 'resource')
  const outDir = path.join(scriptDir, 'out')
  await buildResource(resourceDir, outDir, {
    runtime: 'fivem',
    manifestKind: 'none',
    serverOutDir: outDir,
    clientOutDir: outDir,
    serverOutFile: 'server.js',
    clientOutFile: 'client.js',
    dependencyResolution: { mode: 'isolated' },
    environmentAliases: { '@opencore/environment': path.join(resourceDir, 'env.ts') },
    server: { platform: 'node', format: 'cjs', target: 'es2020', external: ['unused-external'] },
    client: false,
  })
  const output = fs.readFileSync(path.join(outDir, 'server.js'), 'utf8')
  if (!output.includes('merged-env')) throw new Error('environment alias was not bundled')
  if (fs.existsSync(path.join(outDir, 'node_modules'))) throw new Error('unused external should not install node_modules')
}
main().catch(err => { console.error(err.stack || err.message); process.exit(1) })
`), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("node", runner, scriptDir)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "NODE_PATH="+filepath.Join(scriptDir, "node_modules")+string(os.PathListSeparator)+filepath.Join(repoRoot, "node_modules"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("embedded build smoke test failed: %v\nOutput:\n%s", err, string(output))
	}
}

func findRepoRootWithNodeDeps(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	cur := wd
	for i := 0; i < 8; i++ {
		if info, err := os.Stat(filepath.Join(cur, "node_modules", "esbuild")); err == nil && info.IsDir() {
			return cur
		}
		if info, err := os.Stat(filepath.Join(cur, "node_modules", ".pnpm")); err == nil && info.IsDir() {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	t.Fatalf("could not find repo root with node_modules from %s", wd)
	return ""
}
