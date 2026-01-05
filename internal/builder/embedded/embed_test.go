package embedded

import (
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
		"createExcludeNodeAdaptersPlugin",
		"preserveFiveMExportsPlugin",
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
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(buildFuncsContent, fn) {
			t.Errorf("build_functions.js missing required function: %s", fn)
		}
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
