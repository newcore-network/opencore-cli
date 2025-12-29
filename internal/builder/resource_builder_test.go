package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewResourceBuilder(t *testing.T) {
	rb := NewResourceBuilder("/test/project")

	if rb == nil {
		t.Fatal("NewResourceBuilder returned nil")
	}

	if rb.projectPath != "/test/project" {
		t.Errorf("Expected projectPath '/test/project', got '%s'", rb.projectPath)
	}

	if rb.embeddedScriptReady {
		t.Error("embeddedScriptReady should be false initially")
	}

	if rb.embeddedScriptPath != "" {
		t.Error("embeddedScriptPath should be empty initially")
	}
}

func TestEnsureEmbeddedScript(t *testing.T) {
	rb := NewResourceBuilder(".")

	// First call should extract the script
	scriptPath, err := rb.ensureEmbeddedScript()
	if err != nil {
		t.Fatalf("ensureEmbeddedScript failed: %v", err)
	}

	if scriptPath == "" {
		t.Error("Script path is empty")
	}

	// Verify the script was created
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Errorf("Script file was not created at %s", scriptPath)
	}

	// Read the script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Script file is empty")
	}

	// Second call should return the same path (cached)
	scriptPath2, err := rb.ensureEmbeddedScript()
	if err != nil {
		t.Fatalf("Second ensureEmbeddedScript failed: %v", err)
	}

	if scriptPath != scriptPath2 {
		t.Errorf("Expected cached path '%s', got '%s'", scriptPath, scriptPath2)
	}

	// Cleanup
	rb.Cleanup()

	// Verify cleanup
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("Script file should be deleted after Cleanup")
	}
}

func TestCleanup(t *testing.T) {
	rb := NewResourceBuilder(".")

	// Extract the script
	scriptPath, err := rb.ensureEmbeddedScript()
	if err != nil {
		t.Fatalf("ensureEmbeddedScript failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatal("Script should exist before cleanup")
	}

	// Cleanup
	rb.Cleanup()

	// Verify it's deleted
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("Script should be deleted after cleanup")
	}

	// Verify state is reset
	if rb.embeddedScriptReady {
		t.Error("embeddedScriptReady should be false after cleanup")
	}

	if rb.embeddedScriptPath != "" {
		t.Error("embeddedScriptPath should be empty after cleanup")
	}

	// Cleanup again should not panic
	rb.Cleanup()
}

func TestGetBuildScriptPath_EmbeddedScript(t *testing.T) {
	rb := NewResourceBuilder(".")
	defer rb.Cleanup()

	task := BuildTask{
		Path:           "./core",
		CustomCompiler: "", // Empty = use embedded
	}

	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		t.Fatalf("getBuildScriptPath failed: %v", err)
	}

	if scriptPath == "" {
		t.Error("Script path should not be empty")
	}

	// Verify it's the embedded script
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Error("Embedded script should exist")
	}
}

func TestGetBuildScriptPath_CustomCompiler(t *testing.T) {
	// Create a temporary custom compiler file
	tmpDir := t.TempDir()
	customCompiler := filepath.Join(tmpDir, "custom-build.js")
	if err := os.WriteFile(customCompiler, []byte("// custom build"), 0644); err != nil {
		t.Fatalf("Failed to create custom compiler: %v", err)
	}

	rb := NewResourceBuilder(tmpDir)
	defer rb.Cleanup()

	task := BuildTask{
		Path:           "./core",
		CustomCompiler: "custom-build.js", // Relative path
	}

	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		t.Fatalf("getBuildScriptPath failed: %v", err)
	}

	if scriptPath != customCompiler {
		t.Errorf("Expected custom compiler path '%s', got '%s'", customCompiler, scriptPath)
	}
}

func TestGetBuildScriptPath_CustomCompilerAbsolute(t *testing.T) {
	// Create a temporary custom compiler file
	tmpDir := t.TempDir()
	customCompiler := filepath.Join(tmpDir, "absolute-build.js")
	if err := os.WriteFile(customCompiler, []byte("// custom build"), 0644); err != nil {
		t.Fatalf("Failed to create custom compiler: %v", err)
	}

	rb := NewResourceBuilder(".")
	defer rb.Cleanup()

	task := BuildTask{
		Path:           "./core",
		CustomCompiler: customCompiler, // Absolute path
	}

	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		t.Fatalf("getBuildScriptPath failed: %v", err)
	}

	if scriptPath != customCompiler {
		t.Errorf("Expected custom compiler path '%s', got '%s'", customCompiler, scriptPath)
	}
}

func TestGetBuildScriptPath_CustomCompilerNotFound(t *testing.T) {
	rb := NewResourceBuilder(".")
	defer rb.Cleanup()

	task := BuildTask{
		Path:           "./core",
		CustomCompiler: "./nonexistent/build.js",
	}

	_, err := rb.getBuildScriptPath(task)
	if err == nil {
		t.Error("Expected error for non-existent custom compiler")
	}
}

func TestCopyResource(t *testing.T) {
	// Create source directory with files
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create some files
	if err := os.WriteFile(filepath.Join(srcDir, "file1.lua"), []byte("-- lua script"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.lua"), []byte("-- nested"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create node_modules to test skip
	if err := os.MkdirAll(filepath.Join(srcDir, "node_modules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "node_modules", "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	rb := NewResourceBuilder(".")
	task := BuildTask{
		Path:   srcDir,
		OutDir: outDir,
	}

	output, err := rb.copyResource(task)

	if err != nil {
		t.Fatalf("copyResource failed: %v", err)
	}

	if output == "" {
		t.Error("Expected non-empty output message")
	}

	// Verify files were copied
	dstDir := filepath.Join(outDir, filepath.Base(srcDir))

	if _, err := os.Stat(filepath.Join(dstDir, "file1.lua")); os.IsNotExist(err) {
		t.Error("file1.lua should be copied")
	}

	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "file2.lua")); os.IsNotExist(err) {
		t.Error("subdir/file2.lua should be copied")
	}

	// Verify node_modules was skipped
	if _, err := os.Stat(filepath.Join(dstDir, "node_modules")); !os.IsNotExist(err) {
		t.Error("node_modules should not be copied")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	content := []byte("Hello, World!")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	copied, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(copied) != string(content) {
		t.Errorf("Content mismatch: got '%s', expected '%s'", string(copied), string(content))
	}

	// Verify permissions
	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)

	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Permissions mismatch: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestBuildTaskTypes(t *testing.T) {
	rb := NewResourceBuilder(".")
	defer rb.Cleanup()

	tests := []struct {
		taskType    ResourceType
		description string
	}{
		{TypeCore, "core build"},
		{TypeResource, "resource build"},
		{TypeStandalone, "standalone build"},
		{TypeViews, "views build"},
		{TypeCopy, "copy resource"},
	}

	for _, tt := range tests {
		task := BuildTask{
			Path:         "./test",
			ResourceName: "test",
			Type:         tt.taskType,
			OutDir:       "./dist",
		}

		// Just verify Build doesn't panic for any type
		// Actual execution would fail without Node.js setup
		result := rb.Build(task)

		// For TypeCopy with non-existent path, we expect an error
		if tt.taskType == TypeCopy {
			// Copy of non-existent directory should fail
			if result.Error == nil {
				t.Errorf("Expected error for copy of non-existent directory")
			}
		}

		// All types should return a result
		if result.Task.Type != tt.taskType {
			t.Errorf("Result task type mismatch for %s", tt.description)
		}
	}
}
