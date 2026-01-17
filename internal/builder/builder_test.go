package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestCollectAllTasks_CoreOnly(t *testing.T) {
	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:           "./core",
			ResourceName:   "[core]",
			CustomCompiler: "./scripts/core-build.js",
		},
		Resources: config.ResourcesConfig{},
		Build: config.BuildConfig{
			Minify:     true,
			SourceMaps: true,
			Target:     "ES2020",
		},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Type != TypeCore {
		t.Errorf("Expected TypeCore, got %v", task.Type)
	}

	if task.ResourceName != "[core]" {
		t.Errorf("Expected resource name '[core]', got '%s'", task.ResourceName)
	}

	if task.CustomCompiler != "./scripts/core-build.js" {
		t.Errorf("Expected custom compiler './scripts/core-build.js', got '%s'", task.CustomCompiler)
	}

	if !task.Options.Minify {
		t.Error("Expected Minify to be true")
	}
}

func TestCollectAllTasks_WithCoreViews(t *testing.T) {
	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:           "./core",
			ResourceName:   "[core]",
			CustomCompiler: "./scripts/core-build.js",
			Views: &config.ViewsConfig{
				Path:      "./core/views",
				Framework: "react",
			},
		},
		Resources: config.ResourcesConfig{},
		Build:     config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks (core + views), got %d", len(tasks))
	}

	// Check views task
	var viewsTask *BuildTask
	for i := range tasks {
		if tasks[i].Type == TypeViews {
			viewsTask = &tasks[i]
			break
		}
	}

	if viewsTask == nil {
		t.Fatal("Expected views task")
	}

	if viewsTask.Options.Framework != "react" {
		t.Errorf("Expected framework 'react', got '%s'", viewsTask.Options.Framework)
	}

	// Views should inherit core's custom compiler
	if viewsTask.CustomCompiler != "./scripts/core-build.js" {
		t.Errorf("Expected views custom compiler './scripts/core-build.js', got '%s'", viewsTask.CustomCompiler)
	}
}

func TestCollectAllTasks_WithExplicitResources(t *testing.T) {
	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "[core]",
		},
		Resources: config.ResourcesConfig{
			Explicit: []config.ExplicitResource{
				{
					Path:           "./resources/admin",
					ResourceName:   "admin-panel",
					CustomCompiler: "./scripts/admin-build.js",
					Views: &config.ViewsConfig{
						Path:      "./resources/admin/ui",
						Framework: "vue",
					},
				},
			},
		},
		Build: config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	// Should have: core, admin resource, admin views
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Find admin resource task
	var adminTask *BuildTask
	for i := range tasks {
		if tasks[i].ResourceName == "admin-panel" {
			adminTask = &tasks[i]
			break
		}
	}

	if adminTask == nil {
		t.Fatal("Expected admin-panel task")
	}

	if adminTask.Type != TypeResource {
		t.Errorf("Expected TypeResource, got %v", adminTask.Type)
	}

	if adminTask.CustomCompiler != "./scripts/admin-build.js" {
		t.Errorf("Expected custom compiler './scripts/admin-build.js', got '%s'", adminTask.CustomCompiler)
	}

	// Find admin views task
	var adminViewsTask *BuildTask
	for i := range tasks {
		if tasks[i].ResourceName == "admin-panel/ui" {
			adminViewsTask = &tasks[i]
			break
		}
	}

	if adminViewsTask == nil {
		t.Fatal("Expected admin-panel/ui task")
	}

	if adminViewsTask.Options.Framework != "vue" {
		t.Errorf("Expected framework 'vue', got '%s'", adminViewsTask.Options.Framework)
	}
}

func TestCollectAllTasks_WithStandalone(t *testing.T) {
	falseVal := false
	trueVal := true

	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "[core]",
		},
		Resources: config.ResourcesConfig{},
		Standalones: &config.StandaloneConfig{
			Explicit: []config.ExplicitResource{
				{
					Path:           "./standalones/utils",
					Compile:        &trueVal,
					CustomCompiler: "./scripts/utils-build.js",
				},
				{
					Path:    "./standalones/legacy",
					Compile: &falseVal, // Copy only
				},
			},
		},
		Build: config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	// Should have: core, utils standalone, legacy copy
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Find utils task
	var utilsTask *BuildTask
	for i := range tasks {
		if tasks[i].Path == "./standalones/utils" {
			utilsTask = &tasks[i]
			break
		}
	}

	if utilsTask == nil {
		t.Fatal("Expected utils task")
	}

	if utilsTask.Type != TypeStandalone {
		t.Errorf("Expected TypeStandalone, got %v", utilsTask.Type)
	}

	if utilsTask.CustomCompiler != "./scripts/utils-build.js" {
		t.Errorf("Expected custom compiler './scripts/utils-build.js', got '%s'", utilsTask.CustomCompiler)
	}

	// Find legacy task
	var legacyTask *BuildTask
	for i := range tasks {
		if tasks[i].Path == "./standalones/legacy" {
			legacyTask = &tasks[i]
			break
		}
	}

	if legacyTask == nil {
		t.Fatal("Expected legacy task")
	}

	if legacyTask.Type != TypeCopy {
		t.Errorf("Expected TypeCopy for compile:false, got %v", legacyTask.Type)
	}
}

func TestCollectAllTasks_WithGlobPatterns(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create resources directory
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(filepath.Join(resourcesDir, "admin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(resourcesDir, "inventory"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create core directory
	coreDir := filepath.Join(tmpDir, "core")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "[core]",
		},
		Resources: config.ResourcesConfig{
			Include: []string{"./resources/*"},
		},
		Build: config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	// Should have: core, admin, inventory
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Verify resource types
	resourceCount := 0
	for _, task := range tasks {
		if task.Type == TypeResource {
			resourceCount++
		}
	}

	if resourceCount != 2 {
		t.Errorf("Expected 2 resource tasks, got %d", resourceCount)
	}
}

func TestCollectAllTasks_BuildOptions(t *testing.T) {
	serverFalse := false

	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "[core]",
		},
		Resources: config.ResourcesConfig{
			Explicit: []config.ExplicitResource{
				{
					Path: "./resources/client-only",
					Build: &config.ResourceBuildConfig{
						Server: &serverFalse,
					},
				},
			},
		},
		Build: config.BuildConfig{
			Minify:     true,
			SourceMaps: false,
			Target:     "ES2021",
		},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	// Find client-only task
	var clientOnlyTask *BuildTask
	for i := range tasks {
		if tasks[i].Path == "./resources/client-only" {
			clientOnlyTask = &tasks[i]
			break
		}
	}

	if clientOnlyTask == nil {
		t.Fatal("Expected client-only task")
	}

	if clientOnlyTask.Options.Server.Enabled {
		t.Error("Expected Server to be false")
	}

	// Build options should be inherited
	if !clientOnlyTask.Options.Minify {
		t.Error("Expected Minify to be true (inherited)")
	}

	if clientOnlyTask.Options.SourceMaps {
		t.Error("Expected SourceMaps to be false (inherited)")
	}

	if clientOnlyTask.Options.Target != "ES2021" {
		t.Errorf("Expected Target 'ES2021', got '%s'", clientOnlyTask.Options.Target)
	}
}

func TestHasClientCode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create resource with client code
	withClient := filepath.Join(tmpDir, "with-client")
	if err := os.MkdirAll(filepath.Join(withClient, "src", "client"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create resource without client code
	withoutClient := filepath.Join(tmpDir, "without-client")
	if err := os.MkdirAll(filepath.Join(withoutClient, "src", "server"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Core: config.CoreConfig{Path: "./core"},
	}
	builder := New(cfg)

	if !builder.hasClientCode(withClient) {
		t.Error("Expected hasClientCode to return true for resource with client dir")
	}

	if builder.hasClientCode(withoutClient) {
		t.Error("Expected hasClientCode to return false for resource without client dir")
	}
}

func TestCollectAllTasks_EntryPoints(t *testing.T) {
	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "[core]",
			EntryPoints: &config.EntryPoints{
				Server: "./core/server/main.ts",
				Client: "./core/client/main.ts",
			},
		},
		Resources: config.ResourcesConfig{},
		Build:     config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Options.EntryPoints == nil {
		t.Fatal("Expected EntryPoints to be set")
	}

	if task.Options.EntryPoints.Server != "./core/server/main.ts" {
		t.Errorf("Expected server entry './core/server/main.ts', got '%s'", task.Options.EntryPoints.Server)
	}

	if task.Options.EntryPoints.Client != "./core/client/main.ts" {
		t.Errorf("Expected client entry './core/client/main.ts', got '%s'", task.Options.EntryPoints.Client)
	}
}

func TestBuilderNew(t *testing.T) {
	cfg := &config.Config{
		Name:        "test",
		OutDir:      "./dist",
		Destination: "C:/FXServer/resources",
		Core: config.CoreConfig{
			Path: "./core",
		},
		Build: config.BuildConfig{
			Parallel:   true,
			MaxWorkers: 8,
		},
	}

	builder := New(cfg)

	if builder == nil {
		t.Fatal("New returned nil")
	}

	if builder.config != cfg {
		t.Error("Config not properly assigned")
	}

	if builder.resourceBuilder == nil {
		t.Error("ResourceBuilder is nil")
	}

	if builder.deployer == nil {
		t.Error("Deployer is nil")
	}
}
