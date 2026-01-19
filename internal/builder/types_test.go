package builder

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResourceTypeConstants(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		expected     string
	}{
		{TypeCore, "core"},
		{TypeResource, "resource"},
		{TypeStandalone, "standalone"},
		{TypeViews, "views"},
		{TypeCopy, "copy"},
	}

	for _, tt := range tests {
		if string(tt.resourceType) != tt.expected {
			t.Errorf("ResourceType %v = '%s', expected '%s'", tt.resourceType, string(tt.resourceType), tt.expected)
		}
	}
}

func TestBuildTaskStructure(t *testing.T) {
	task := BuildTask{
		Path:           "./core",
		ResourceName:   "[core]",
		Type:           TypeCore,
		OutDir:         "./dist",
		CustomCompiler: "./scripts/build.js",
		Options: BuildOptions{
			Server:     SideConfigValue{Enabled: true},
			Client:     SideConfigValue{Enabled: true},
			Minify:     true,
			SourceMaps: true,
			Target:     "ES2020",
			Compile:    true,
		},
	}

	if task.Path != "./core" {
		t.Errorf("Expected path './core', got '%s'", task.Path)
	}

	if task.CustomCompiler != "./scripts/build.js" {
		t.Errorf("Expected customCompiler './scripts/build.js', got '%s'", task.CustomCompiler)
	}

	if task.Type != TypeCore {
		t.Errorf("Expected type TypeCore, got %v", task.Type)
	}

	if !task.Options.Server.Enabled {
		t.Error("Expected Options.Server.Enabled to be true")
	}

	if !task.Options.Minify {
		t.Error("Expected Options.Minify to be true")
	}
}

func TestBuildOptionsJSON(t *testing.T) {
	options := BuildOptions{
		Server:         SideConfigValue{Enabled: true, Options: &BuildSideOptions{External: []string{"typeorm"}}},
		Client:         SideConfigValue{Enabled: false},
		NUI:            true,
		Minify:         true,
		SourceMaps:     false,
		Target:         "ES2020",
		ForceInclude:   []string{"favicon.ico"},
		BuildCommand:   "pnpm astro build",
		OutputDir:      "dist",
		ServerBinaries: []string{"bin"},
		EntryPoints: &EntryPoints{
			Server: "./src/server.ts",
			Client: "./src/client.ts",
		},
		Framework: "react",
		Compile:   true,
	}

	// Test JSON marshaling
	data, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("Failed to marshal BuildOptions: %v", err)
	}

	// Test JSON unmarshaling
	var parsed BuildOptions
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal BuildOptions: %v", err)
	}

	if parsed.Server.Enabled != options.Server.Enabled {
		t.Errorf("Server.Enabled mismatch: got %v, expected %v", parsed.Server.Enabled, options.Server.Enabled)
	}
	if parsed.Client.Enabled != options.Client.Enabled {
		t.Errorf("Client.Enabled mismatch: got %v, expected %v", parsed.Client.Enabled, options.Client.Enabled)
	}

	if parsed.Target != options.Target {
		t.Errorf("Target mismatch: got '%s', expected '%s'", parsed.Target, options.Target)
	}

	if len(parsed.ForceInclude) != len(options.ForceInclude) {
		t.Errorf("ForceInclude mismatch: got %d, expected %d", len(parsed.ForceInclude), len(options.ForceInclude))
	}

	if parsed.BuildCommand != options.BuildCommand {
		t.Errorf("BuildCommand mismatch: got '%s', expected '%s'", parsed.BuildCommand, options.BuildCommand)
	}

	if parsed.OutputDir != options.OutputDir {
		t.Errorf("OutputDir mismatch: got '%s', expected '%s'", parsed.OutputDir, options.OutputDir)
	}

	if len(parsed.ServerBinaries) != len(options.ServerBinaries) {
		t.Errorf("ServerBinaries mismatch: got %d, expected %d", len(parsed.ServerBinaries), len(options.ServerBinaries))
	}

	if parsed.EntryPoints == nil {
		t.Error("EntryPoints should not be nil after unmarshaling")
	} else {
		if parsed.EntryPoints.Server != options.EntryPoints.Server {
			t.Errorf("EntryPoints.Server mismatch")
		}
	}
}

func TestBuildResult(t *testing.T) {
	task := BuildTask{
		Path:         "./core",
		ResourceName: "[core]",
		Type:         TypeCore,
	}

	result := BuildResult{
		Task:     task,
		Success:  true,
		Duration: 1500 * time.Millisecond,
		Error:    nil,
		Output:   "Build successful",
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Duration != 1500*time.Millisecond {
		t.Errorf("Expected duration 1.5s, got %v", result.Duration)
	}

	if result.Task.ResourceName != "[core]" {
		t.Errorf("Expected task ResourceName '[core]', got '%s'", result.Task.ResourceName)
	}
}

func TestBuildProgress(t *testing.T) {
	progress := BuildProgress{
		Total:     10,
		Completed: 5,
		Current:   "admin-panel",
		Results: []BuildResult{
			{
				Task:    BuildTask{ResourceName: "core"},
				Success: true,
			},
			{
				Task:    BuildTask{ResourceName: "inventory"},
				Success: true,
			},
		},
	}

	if progress.Total != 10 {
		t.Errorf("Expected Total 10, got %d", progress.Total)
	}

	if progress.Completed != 5 {
		t.Errorf("Expected Completed 5, got %d", progress.Completed)
	}

	if len(progress.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(progress.Results))
	}
}

func TestEntryPoints(t *testing.T) {
	ep := EntryPoints{
		Server: "./src/server.ts",
		Client: "./src/client.ts",
	}

	data, err := json.Marshal(ep)
	if err != nil {
		t.Fatalf("Failed to marshal EntryPoints: %v", err)
	}

	var parsed EntryPoints
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal EntryPoints: %v", err)
	}

	if parsed.Server != ep.Server {
		t.Errorf("Server mismatch: got '%s', expected '%s'", parsed.Server, ep.Server)
	}

	if parsed.Client != ep.Client {
		t.Errorf("Client mismatch: got '%s', expected '%s'", parsed.Client, ep.Client)
	}
}
