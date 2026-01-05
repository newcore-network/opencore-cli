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
		if _, err := os.Stat(rb.embeddedScriptPath); err == nil {
			return rb.embeddedScriptPath, nil
		}
	}

	cacheDir := filepath.Join(rb.projectPath, "node_modules", ".cache", "opencore")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	entries, err := embedded.BuildFS.ReadDir(".")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := embedded.BuildFS.ReadFile(entry.Name())
		if err != nil {
			return "", fmt.Errorf("failed to read embedded file %s: %w", entry.Name(), err)
		}
		destPath := filepath.Join(cacheDir, entry.Name())
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return "", fmt.Errorf("failed to extract embedded file %s: %w", entry.Name(), err)
		}
	}

	rb.embeddedScriptPath = filepath.Join(cacheDir, "build.js")
	rb.embeddedScriptReady = true
	return rb.embeddedScriptPath, nil
}

// Cleanup removes temporary files created by the builder
func (rb *ResourceBuilder) Cleanup() {
	rb.embeddedScriptMutex.Lock()
	defer rb.embeddedScriptMutex.Unlock()

	if rb.embeddedScriptPath != "" {
		cacheDir := filepath.Dir(rb.embeddedScriptPath)
		os.RemoveAll(cacheDir)
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
	scriptPath, err := rb.getBuildScriptPath(task)
	if err != nil {
		return "", err
	}

	optionsJSON, err := json.Marshal(task.Options)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %w", err)
	}

	// Use the embedded script to handle the copy so it also handles dependencies/symlinks
	cmd := exec.Command("node", scriptPath, "single",
		"copy", task.Path, task.OutDir, string(optionsJSON))
	cmd.Dir = rb.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("resource copy failed: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
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
