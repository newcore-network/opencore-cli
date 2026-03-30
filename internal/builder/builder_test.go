package builder

import (
	"os"
	"path/filepath"
	"strings"
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
			Server:     &config.BuildSideConfig{Target: "ES2020"},
			Client:     &config.BuildSideConfig{Target: "ES2020"},
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
				Path:         "./core/views",
				Framework:    "react",
				BuildCommand: "pnpm astro build",
				OutputDir:    "dist",
			},
			Build: &config.BuildConfig{
				ServerBinaries:       []string{"bin"},
				ServerBinaryPlatform: "win32",
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

	if viewsTask.Options.BuildCommand != "pnpm astro build" {
		t.Errorf("Expected buildCommand 'pnpm astro build', got '%s'", viewsTask.Options.BuildCommand)
	}

	if viewsTask.Options.OutputDir != "dist" {
		t.Errorf("Expected outputDir 'dist', got '%s'", viewsTask.Options.OutputDir)
	}

	if len(tasks[0].Options.ServerBinaries) != 1 {
		t.Errorf("Expected core serverBinaries to be set")
	}

	if tasks[0].Options.ServerBinaryPlatform != "win32" {
		t.Errorf("Expected serverBinaryPlatform 'win32'")
	}

	// Views should inherit core's custom compiler
	if viewsTask.CustomCompiler != "./scripts/core-build.js" {
		t.Errorf("Expected views custom compiler './scripts/core-build.js', got '%s'", viewsTask.CustomCompiler)
	}
}

func TestCollectAllTasks_WithViewsForceInclude(t *testing.T) {
	cfg := &config.Config{
		Name:        "test-project",
		Destination: "./dist",
		OutDir:      "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "[core]",
			Views: &config.ViewsConfig{
				Path:         "./core/views",
				Framework:    "react",
				ForceInclude: []string{"favicon.ico", "*.mp3"},
			},
		},
		Resources: config.ResourcesConfig{
			Explicit: []config.ExplicitResource{
				{
					Path:         "./resources/admin",
					ResourceName: "admin-panel",
					Views: &config.ViewsConfig{
						Path:         "./resources/admin/ui",
						Framework:    "vue",
						ForceInclude: []string{"robots.txt"},
					},
				},
			},
		},
		Build: config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	if len(tasks) != 4 {
		t.Fatalf("Expected 4 tasks (core + core views + admin + admin views), got %d", len(tasks))
	}

	var coreViewsTask *BuildTask
	var adminViewsTask *BuildTask
	for i := range tasks {
		switch tasks[i].ResourceName {
		case "[core]/ui":
			coreViewsTask = &tasks[i]
		case "admin-panel/ui":
			adminViewsTask = &tasks[i]
		}
	}

	if coreViewsTask == nil {
		t.Fatal("Expected core views task")
	}

	if adminViewsTask == nil {
		t.Fatal("Expected admin views task")
	}

	if len(coreViewsTask.Options.ForceInclude) != 2 {
		t.Errorf("Expected 2 core forceInclude entries, got %d", len(coreViewsTask.Options.ForceInclude))
	}

	if len(adminViewsTask.Options.ForceInclude) != 1 {
		t.Errorf("Expected 1 admin forceInclude entry, got %d", len(adminViewsTask.Options.ForceInclude))
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

func TestDetectViewFramework_PrefersViteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	viewDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(filepath.Join(viewDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(viewDir, "vite.config.ts"), []byte("export default {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(viewDir, "src", "app.tsx"), []byte("export const App = () => null\n"), 0644); err != nil {
		t.Fatal(err)
	}

	framework := detectViewFramework(viewDir)
	if framework != "vite" {
		t.Fatalf("Expected framework 'vite', got '%s'", framework)
	}
}

func TestCollectAllTasks_ViewsFrameworkWithoutPathUsesAutodiscovery(t *testing.T) {
	tmpDir := t.TempDir()
	resourceDir := filepath.Join(tmpDir, "resources", "auth")
	viewDir := filepath.Join(resourceDir, "ui")
	if err := os.MkdirAll(viewDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(viewDir, "vite.config.ts"), []byte("export default {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(viewDir, "main.tsx"), []byte("export {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	cfg := &config.Config{
		Name:   "test-project",
		OutDir: "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "core",
		},
		Resources: config.ResourcesConfig{
			Explicit: []config.ExplicitResource{
				{
					Path:         "./resources/auth",
					ResourceName: "auth",
					Views: &config.ViewsConfig{
						Framework: "vite",
					},
				},
			},
		},
		Build: config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	var viewsTask *BuildTask
	for i := range tasks {
		if tasks[i].Type == TypeViews && tasks[i].ResourceName == "auth/ui" {
			viewsTask = &tasks[i]
			break
		}
	}

	if viewsTask == nil {
		t.Fatal("Expected auth/ui views task")
	}

	if viewsTask.Path != "./resources/auth/ui" {
		t.Fatalf("Expected autodetected views path './resources/auth/ui', got '%s'", viewsTask.Path)
	}
	if viewsTask.Options.Framework != "vite" {
		t.Fatalf("Expected views framework 'vite', got '%s'", viewsTask.Options.Framework)
	}
}

func TestCollectAllTasks_ResourcesDefaultViewsFrameworkApplied(t *testing.T) {
	tmpDir := t.TempDir()
	resourceDir := filepath.Join(tmpDir, "resources", "auth")
	viewDir := filepath.Join(resourceDir, "ui")
	if err := os.MkdirAll(viewDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(viewDir, "vite.config.ts"), []byte("export default {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(viewDir, "main.tsx"), []byte("export {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	cfg := &config.Config{
		Name:   "test-project",
		OutDir: "./dist",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "core",
		},
		Resources: config.ResourcesConfig{
			Include: []string{"./resources/*"},
			Views: &config.ViewsConfig{
				Framework: "vite",
			},
		},
		Build: config.BuildConfig{},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()

	var viewsTask *BuildTask
	for i := range tasks {
		if tasks[i].Type == TypeViews && tasks[i].ResourceName == "auth/ui" {
			viewsTask = &tasks[i]
			break
		}
	}

	if viewsTask == nil {
		t.Fatal("Expected auth/ui views task")
	}
	if viewsTask.Options.Framework != "vite" {
		t.Fatalf("Expected views framework 'vite', got '%s'", viewsTask.Options.Framework)
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
			Server:     &config.BuildSideConfig{Target: "ES2021"},
			Client:     &config.BuildSideConfig{Target: "ES2021"},
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

	if clientOnlyTask.Options.Server.Options != nil && clientOnlyTask.Options.Server.Options.Target != "ES2021" {
		t.Errorf("Expected server target 'ES2021', got '%s'", clientOnlyTask.Options.Server.Options.Target)
	}
	if clientOnlyTask.Options.Client.Options != nil && clientOnlyTask.Options.Client.Options.Target != "ES2021" {
		t.Errorf("Expected client target 'ES2021', got '%s'", clientOnlyTask.Options.Client.Options.Target)
	}
}

func TestCollectAllTasks_RageMPLayout(t *testing.T) {
	cfg := &config.Config{
		Name:   "test-project",
		OutDir: "./build",
		Core: config.CoreConfig{
			Path:         "./core",
			ResourceName: "core",
		},
		Resources: config.ResourcesConfig{},
		Build:     config.BuildConfig{},
		Adapter: &config.AdapterConfig{
			Server: &config.AdapterBinding{
				Name:  "ragemp",
				Valid: true,
				Runtime: &config.AdapterRuntimeBinding{
					Runtime:  "ragemp",
					Server:   &config.AdapterRuntimeSideHints{Target: "node14"},
					Client:   &config.AdapterRuntimeSideHints{OutputRoot: "client_packages"},
					Manifest: &config.AdapterManifestBinding{Kind: "none"},
				},
			},
		},
	}

	builder := New(cfg)
	tasks := builder.collectAllTasks()
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.OutDir != filepath.Join("build", "packages", "core") {
		t.Fatalf("Expected RageMP server out dir, got '%s'", task.OutDir)
	}
	if task.Options.Runtime != "ragemp" {
		t.Fatalf("Expected runtime 'ragemp', got '%s'", task.Options.Runtime)
	}
	if task.Options.ServerOutDir != filepath.Join("build", "packages", "core") {
		t.Fatalf("Unexpected server out dir '%s'", task.Options.ServerOutDir)
	}
	if task.Options.ClientOutDir != filepath.Join("build", "client_packages", "core") {
		t.Fatalf("Unexpected client out dir '%s'", task.Options.ClientOutDir)
	}
	if task.Options.ServerOutFile != "index.js" || task.Options.ClientOutFile != "index.js" {
		t.Fatalf("Expected RageMP output files to use index.js")
	}
	if task.Options.ManifestKind != "none" {
		t.Fatalf("Expected RageMP manifest kind 'none', got '%s'", task.Options.ManifestKind)
	}
}

func TestWriteRuntimeArtifactsRageMPBarrels(t *testing.T) {
	outDir := t.TempDir()
	cfg := &config.Config{
		OutDir: outDir,
		Core: config.CoreConfig{
			ResourceName: "core",
		},
		Build: config.BuildConfig{},
		Adapter: &config.AdapterConfig{
			Server: &config.AdapterBinding{Name: "ragemp", Valid: true},
			Client: &config.AdapterBinding{Name: "ragemp", Valid: true},
		},
	}
	builder := New(cfg)

	paths := []string{
		filepath.Join(outDir, "packages", "core", "index.js"),
		filepath.Join(outDir, "packages", "alpha", "index.js"),
		filepath.Join(outDir, "client_packages", "core", "index.js"),
		filepath.Join(outDir, "client_packages", "bravo", "index.js"),
	}
	for _, p := range paths {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("module.exports = {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	results := []BuildResult{
		{Success: true, Task: BuildTask{ResourceName: "alpha", Type: TypeResource, Options: BuildOptions{Server: SideConfigValue{Enabled: true}, Client: SideConfigValue{Enabled: false}}}},
		{Success: true, Task: BuildTask{ResourceName: "core", Type: TypeCore, Options: BuildOptions{Server: SideConfigValue{Enabled: true}, Client: SideConfigValue{Enabled: true}}}},
		{Success: true, Task: BuildTask{ResourceName: "bravo", Type: TypeResource, Options: BuildOptions{Server: SideConfigValue{Enabled: false}, Client: SideConfigValue{Enabled: true}}}},
		{Success: true, Task: BuildTask{ResourceName: "core/ui", Type: TypeViews, Options: BuildOptions{}}},
	}

	if err := builder.writeRuntimeArtifacts(results); err != nil {
		t.Fatalf("writeRuntimeArtifacts() error = %v", err)
	}

	serverBarrel, err := os.ReadFile(filepath.Join(outDir, "packages", "index.js"))
	if err != nil {
		t.Fatalf("failed to read server barrel: %v", err)
	}
	serverText := string(serverBarrel)
	if !strings.Contains(serverText, "require('./core')") || !strings.Contains(serverText, "require('./alpha')") {
		t.Fatalf("unexpected server barrel contents: %s", serverText)
	}
	if strings.Index(serverText, "require('./core')") > strings.Index(serverText, "require('./alpha')") {
		t.Fatal("expected core to be required before alpha in server barrel")
	}
	if strings.Contains(serverText, "bravo") {
		t.Fatal("did not expect client-only resource in server barrel")
	}

	clientBarrel, err := os.ReadFile(filepath.Join(outDir, "client_packages", "index.js"))
	if err != nil {
		t.Fatalf("failed to read client barrel: %v", err)
	}
	clientText := string(clientBarrel)
	if !strings.Contains(clientText, "require('./core')") || !strings.Contains(clientText, "require('./bravo')") {
		t.Fatalf("unexpected client barrel contents: %s", clientText)
	}
	if strings.Contains(clientText, "alpha") {
		t.Fatal("did not expect server-only resource in client barrel")
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
