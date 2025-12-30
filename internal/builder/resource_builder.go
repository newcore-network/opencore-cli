package builder

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/newcore-network/opencore-cli/internal/builder/embedded"
)

// ResourceBuilder handles building individual resources
type ResourceBuilder struct {
	projectPath         string
	embeddedScriptPath  string
	embeddedScriptMutex sync.Mutex
	embeddedScriptReady bool
}

// NewResourceBuilder creates a new resource builder
func NewResourceBuilder(projectPath string) *ResourceBuilder {
	return &ResourceBuilder{
		projectPath: projectPath,
	}
}

// ensureEmbeddedScript extracts the embedded build script to the project directory
// so it can resolve node_modules properly
func (rb *ResourceBuilder) ensureEmbeddedScript() (string, error) {
	rb.embeddedScriptMutex.Lock()
	defer rb.embeddedScriptMutex.Unlock()

	if rb.embeddedScriptReady && rb.embeddedScriptPath != "" {
		// Check if file still exists
		if _, err := os.Stat(rb.embeddedScriptPath); err == nil {
			return rb.embeddedScriptPath, nil
		}
	}

	// Create script in project's node_modules/.cache directory so it can resolve modules
	cacheDir := filepath.Join(rb.projectPath, "node_modules", ".cache", "opencore")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	scriptPath := filepath.Join(cacheDir, "build.js")

	// Write embedded script to cache directory
	if err := os.WriteFile(scriptPath, embedded.GetBuildScript(), 0644); err != nil {
		return "", fmt.Errorf("failed to extract embedded build script: %w", err)
	}

	rb.embeddedScriptPath = scriptPath
	rb.embeddedScriptReady = true

	return scriptPath, nil
}

// Cleanup removes temporary files created by the builder
func (rb *ResourceBuilder) Cleanup() {
	rb.embeddedScriptMutex.Lock()
	defer rb.embeddedScriptMutex.Unlock()

	if rb.embeddedScriptPath != "" {
		os.Remove(rb.embeddedScriptPath)
		rb.embeddedScriptPath = ""
		rb.embeddedScriptReady = false
	}
}

// getBuildScriptPath returns the build script path for a task
// Uses custom compiler if specified, otherwise uses embedded script
func (rb *ResourceBuilder) getBuildScriptPath(task BuildTask) (string, error) {
	if task.CustomCompiler != "" {
		// Use custom compiler - resolve relative to project path
		customPath := task.CustomCompiler
		if !filepath.IsAbs(customPath) {
			customPath = filepath.Join(rb.projectPath, customPath)
		}

		// Verify custom compiler exists
		if _, err := os.Stat(customPath); os.IsNotExist(err) {
			return "", fmt.Errorf("custom compiler not found: %s", customPath)
		}

		return customPath, nil
	}

	// Use embedded script
	return rb.ensureEmbeddedScript()
}

// Build executes a build task and returns the result
func (rb *ResourceBuilder) Build(task BuildTask) BuildResult {
	start := time.Now()

	var err error
	var output string

	switch task.Type {
	case TypeCore:
		output, err = rb.buildCore(task)
	case TypeResource:
		output, err = rb.buildResource(task)
	case TypeStandalone:
		output, err = rb.buildStandalone(task)
	case TypeViews:
		output, err = rb.buildViews(task)
	case TypeCopy:
		output, err = rb.copyResource(task)
	default:
		err = fmt.Errorf("unknown resource type: %s", task.Type)
	}

	duration := time.Since(start)

	return BuildResult{
		Task:     task,
		Success:  err == nil,
		Duration: duration,
		Error:    err,
		Output:   output,
	}
}

// buildCore builds the core resource
func (rb *ResourceBuilder) buildCore(task BuildTask) (string, error) {
	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		return "", err
	}

	optionsJSON, err := json.Marshal(task.Options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %w", err)
	}

	cmd := exec.Command("node", scriptPath, "single",
		string(TypeCore), task.Path, task.OutDir, string(optionsJSON))
	cmd.Dir = rb.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("core build failed: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
}

// buildResource builds a satellite resource
func (rb *ResourceBuilder) buildResource(task BuildTask) (string, error) {
	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		return "", err
	}

	optionsJSON, err := json.Marshal(task.Options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %w", err)
	}

	cmd := exec.Command("node", scriptPath, "single",
		string(TypeResource), task.Path, task.OutDir, string(optionsJSON))
	cmd.Dir = rb.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("resource build failed: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
}

// buildStandalone builds a standalone resource
func (rb *ResourceBuilder) buildStandalone(task BuildTask) (string, error) {
	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		return "", err
	}

	optionsJSON, err := json.Marshal(task.Options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %w", err)
	}

	cmd := exec.Command("node", scriptPath, "single",
		string(TypeStandalone), task.Path, task.OutDir, string(optionsJSON))
	cmd.Dir = rb.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("standalone build failed: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
}

// buildViews builds views/NUI for a resource
func (rb *ResourceBuilder) buildViews(task BuildTask) (string, error) {
	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		return "", err
	}

	optionsJSON, err := json.Marshal(task.Options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %w", err)
	}

	cmd := exec.Command("node", scriptPath, "single",
		string(TypeViews), task.Path, task.OutDir, string(optionsJSON))
	cmd.Dir = rb.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("views build failed: %w", err)
	}

	return string(output), nil
}

// copyResource copies a resource without compilation (for compile: false)
func (rb *ResourceBuilder) copyResource(task BuildTask) (string, error) {
	srcPath := task.Path
	dstPath := filepath.Join(task.OutDir, filepath.Base(task.Path))

	// Create destination directory
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy all files recursively
	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		// Skip node_modules and other build artifacts
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "dist") {
			return filepath.SkipDir
		}

		targetPath := filepath.Join(dstPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})

	if err != nil {
		return "", fmt.Errorf("failed to copy resource: %w", err)
	}

	return fmt.Sprintf("Copied %s to %s", srcPath, dstPath), nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
