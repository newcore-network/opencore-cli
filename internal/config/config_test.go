package config

import (
	"encoding/json"
	"testing"
)

func TestConfigParsing(t *testing.T) {
	jsonConfig := `{
		"name": "test-project",
		"outDir": "./dist",
		"destination": "C:/FXServer/resources",
		"core": {
			"path": "./core",
			"resourceName": "[core]",
			"customCompiler": "./scripts/core-build.js",
			"entryPoints": {
				"server": "./core/src/server.ts",
				"client": "./core/src/client.ts"
			},
			"views": {
				"path": "./core/views",
				"framework": "react",
				"forceInclude": ["favicon.ico"]
			}
		},
		"resources": {
			"include": ["./resources/*"],
			"explicit": [
				{
					"path": "./resources/admin",
					"resourceName": "admin-panel",
					"customCompiler": "./scripts/admin-build.js"
				}
			]
		},
		"standalones": {
			"include": ["./standalones/*"],
			"explicit": [
				{
					"path": "./standalones/legacy",
					"compile": false,
					"customCompiler": "./scripts/legacy-build.js"
				}
			]
		},
		"build": {
			"minify": true,
			"sourceMaps": true,
			"target": "ES2020",
			"parallel": true,
			"maxWorkers": 4
		}
	}`

	var cfg Config
	err := json.Unmarshal([]byte(jsonConfig), &cfg)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Test basic fields
	if cfg.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", cfg.Name)
	}

	// OutDir should match Destination as we forced it in Load()
	// But in this raw Unmarshal test it stays as is.
	// The logic for forcing OutDir = Destination is in Load() which we aren't testing here directly with json.Unmarshal.
	if cfg.Destination != "C:/FXServer/resources" {
		t.Errorf("Expected destination 'C:/FXServer/resources', got '%s'", cfg.Destination)
	}

	// Test core config
	if cfg.Core.Path != "./core" {
		t.Errorf("Expected core path './core', got '%s'", cfg.Core.Path)
	}

	if cfg.Core.CustomCompiler != "./scripts/core-build.js" {
		t.Errorf("Expected core customCompiler './scripts/core-build.js', got '%s'", cfg.Core.CustomCompiler)
	}

	if cfg.Core.EntryPoints == nil {
		t.Error("Expected entryPoints to be set")
	} else {
		if cfg.Core.EntryPoints.Server != "./core/src/server.ts" {
			t.Errorf("Expected server entry './core/src/server.ts', got '%s'", cfg.Core.EntryPoints.Server)
		}
	}

	if cfg.Core.Views == nil {
		t.Error("Expected core views to be set")
	} else {
		if len(cfg.Core.Views.ForceInclude) != 1 || cfg.Core.Views.ForceInclude[0] != "favicon.ico" {
			t.Errorf("Expected forceInclude to contain 'favicon.ico'")
		}
	}

	// Test resources config
	if len(cfg.Resources.Include) != 1 {
		t.Errorf("Expected 1 resource include pattern, got %d", len(cfg.Resources.Include))
	}

	if len(cfg.Resources.Explicit) != 1 {
		t.Errorf("Expected 1 explicit resource, got %d", len(cfg.Resources.Explicit))
	} else {
		res := cfg.Resources.Explicit[0]
		if res.CustomCompiler != "./scripts/admin-build.js" {
			t.Errorf("Expected resource customCompiler './scripts/admin-build.js', got '%s'", res.CustomCompiler)
		}
	}

	// Test standalone config
	if cfg.Standalones == nil {
		t.Error("Expected standalones config to be set")
	} else {
		if len(cfg.Standalones.Explicit) != 1 {
			t.Errorf("Expected 1 explicit standalone, got %d", len(cfg.Standalones.Explicit))
		} else {
			standalone := cfg.Standalones.Explicit[0]
			if standalone.Compile == nil || *standalone.Compile != false {
				t.Error("Expected standalone compile to be false")
			}
			if standalone.CustomCompiler != "./scripts/legacy-build.js" {
				t.Errorf("Expected standalone customCompiler './scripts/legacy-build.js', got '%s'", standalone.CustomCompiler)
			}
		}
	}

	// Test build config
	if !cfg.Build.Minify {
		t.Error("Expected minify to be true")
	}
	if !cfg.Build.Parallel {
		t.Error("Expected parallel to be true")
	}
	if cfg.Build.MaxWorkers != 4 {
		t.Errorf("Expected maxWorkers 4, got %d", cfg.Build.MaxWorkers)
	}
}

func TestGetCustomCompiler(t *testing.T) {
	cfg := &Config{
		Core: CoreConfig{
			Path:           "./core",
			CustomCompiler: "./scripts/core-build.js",
		},
		Resources: ResourcesConfig{
			Explicit: []ExplicitResource{
				{
					Path:           "./resources/admin",
					CustomCompiler: "./scripts/admin-build.js",
				},
				{
					Path: "./resources/inventory",
					// No custom compiler
				},
			},
		},
		Standalones: &StandaloneConfig{
			Explicit: []ExplicitResource{
				{
					Path:           "./standalones/utils",
					CustomCompiler: "./scripts/utils-build.js",
				},
			},
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"./core", "./scripts/core-build.js"},
		{"./resources/admin", "./scripts/admin-build.js"},
		{"./resources/inventory", ""},
		{"./standalones/utils", "./scripts/utils-build.js"},
		{"./unknown/path", ""},
	}

	for _, tt := range tests {
		result := cfg.GetCustomCompiler(tt.path)
		if result != tt.expected {
			t.Errorf("GetCustomCompiler(%s) = '%s', expected '%s'", tt.path, result, tt.expected)
		}
	}
}

func TestShouldCompile(t *testing.T) {
	trueVal := true
	falseVal := false

	cfg := &Config{
		Standalones: &StandaloneConfig{
			Explicit: []ExplicitResource{
				{
					Path:    "./standalones/compiled",
					Compile: &trueVal,
				},
				{
					Path:    "./standalones/copy-only",
					Compile: &falseVal,
				},
				{
					Path: "./standalones/default",
					// Compile not set, should default to true
				},
			},
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"./standalones/compiled", true},
		{"./standalones/copy-only", false},
		{"./standalones/default", true},
		{"./standalones/glob-matched", true}, // Not in explicit, defaults to true
	}

	for _, tt := range tests {
		result := cfg.ShouldCompile(tt.path)
		if result != tt.expected {
			t.Errorf("ShouldCompile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestGetExplicitResource(t *testing.T) {
	cfg := &Config{
		Resources: ResourcesConfig{
			Explicit: []ExplicitResource{
				{
					Path:         "./resources/admin",
					ResourceName: "admin-panel",
				},
				{
					Path: "./resources/inventory",
				},
			},
		},
	}

	// Test existing resource
	res := cfg.GetExplicitResource("./resources/admin")
	if res == nil {
		t.Error("Expected to find explicit resource for ./resources/admin")
	} else if res.ResourceName != "admin-panel" {
		t.Errorf("Expected resourceName 'admin-panel', got '%s'", res.ResourceName)
	}

	// Test non-existing resource
	res = cfg.GetExplicitResource("./resources/unknown")
	if res != nil {
		t.Error("Expected nil for unknown resource path")
	}
}

func TestGetExplicitStandalone(t *testing.T) {
	falseVal := false

	cfg := &Config{
		Standalones: &StandaloneConfig{
			Explicit: []ExplicitResource{
				{
					Path:    "./standalones/legacy",
					Compile: &falseVal,
				},
			},
		},
	}

	// Test existing standalone
	res := cfg.GetExplicitStandalone("./standalones/legacy")
	if res == nil {
		t.Error("Expected to find explicit standalone for ./standalones/legacy")
	} else if res.Compile == nil || *res.Compile != false {
		t.Error("Expected compile to be false")
	}

	// Test non-existing standalone
	res = cfg.GetExplicitStandalone("./standalones/unknown")
	if res != nil {
		t.Error("Expected nil for unknown standalone path")
	}

	// Test nil standalone config
	cfgNoStandalone := &Config{}
	res = cfgNoStandalone.GetExplicitStandalone("./standalones/any")
	if res != nil {
		t.Error("Expected nil when standalone config is nil")
	}
}

func TestGetResourceViews(t *testing.T) {
	cfg := &Config{
		Core: CoreConfig{
			Path: "./core",
			Views: &ViewsConfig{
				Path:      "./core/views",
				Framework: "react",
			},
		},
		Resources: ResourcesConfig{
			Explicit: []ExplicitResource{
				{
					Path: "./resources/admin",
					Views: &ViewsConfig{
						Path:      "./resources/admin/ui",
						Framework: "vue",
					},
				},
			},
		},
	}

	// Test core views
	views := cfg.GetResourceViews("./core")
	if views == nil {
		t.Error("Expected views for ./core")
	} else {
		if views.Framework != "react" {
			t.Errorf("Expected framework 'react', got '%s'", views.Framework)
		}
	}

	// Test resource views
	views = cfg.GetResourceViews("./resources/admin")
	if views == nil {
		t.Error("Expected views for ./resources/admin")
	} else {
		if views.Framework != "vue" {
			t.Errorf("Expected framework 'vue', got '%s'", views.Framework)
		}
	}

	// Test resource without views
	views = cfg.GetResourceViews("./resources/other")
	if views != nil {
		t.Error("Expected nil views for resource without views config")
	}
}
