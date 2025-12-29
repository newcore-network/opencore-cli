package embedded

import (
	"strings"
	"testing"
)

func TestEmbeddedBuildScript(t *testing.T) {
	script := GetBuildScript()

	if len(script) == 0 {
		t.Fatal("Embedded build script is empty")
	}

	content := string(script)

	// Check for required functions
	requiredFunctions := []string{
		"buildCore",
		"buildResource",
		"buildStandalone",
		"buildViews",
		"buildSingle",
	}

	for _, fn := range requiredFunctions {
		if !strings.Contains(content, fn) {
			t.Errorf("Embedded script missing required function: %s", fn)
		}
	}

	// Check for required plugins
	requiredPlugins := []string{
		"swcPlugin",
		"excludeNodeAdaptersPlugin",
		"preserveFiveMExportsPlugin",
	}

	for _, plugin := range requiredPlugins {
		if !strings.Contains(content, plugin) {
			t.Errorf("Embedded script missing required plugin: %s", plugin)
		}
	}

	// Check for esbuild import
	if !strings.Contains(content, "require('esbuild')") {
		t.Error("Embedded script missing esbuild require")
	}

	// Check for SWC plugin import
	if !strings.Contains(content, "esbuild-plugin-swc") {
		t.Error("Embedded script missing esbuild-plugin-swc require")
	}

	// Check that it handles the 'single' mode for CLI invocation
	if !strings.Contains(content, "'single'") && !strings.Contains(content, "\"single\"") {
		t.Error("Embedded script missing 'single' mode handling")
	}
}

func TestBuildScriptNotEmpty(t *testing.T) {
	script := BuildScript

	// Script should be at least a few KB
	minSize := 1000 // bytes
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
