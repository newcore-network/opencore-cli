package builder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var importMarkPattern = regexp.MustCompile("\x01([^\x01]+)\x01")

type analyzedFile struct {
	Handlers []analyzedHandler `json:"handlers"`
	Views    []analyzedHandler `json:"views"`
	Sends    []analyzedSend    `json:"sends"`
	Warnings []struct {
		Line    int    `json:"line"`
		Message string `json:"message"`
	} `json:"warnings"`
}

type analyzedHandler struct {
	Kind        typegenKind `json:"kind"`
	Event       string      `json:"event"`
	ClassName   string      `json:"className"`
	MethodName  string      `json:"methodName"`
	Description string      `json:"description"`
	Usage       string      `json:"usage"`
	Line        int         `json:"line"`
}

type analyzedSend struct {
	Event   string `json:"event"`
	Payload struct {
		Kind       string `json:"kind"` // "param" (a parameter of the enclosing method) or "type"
		Type       string `json:"type"`
		ClassName  string `json:"className"`
		MethodName string `json:"methodName"`
		Index      int    `json:"index"`
	} `json:"payload"`
}

type analyzedSource struct {
	path   string
	result analyzedFile
}

type resourceAnalysis struct {
	sources []analyzedSource
}

type cachedAnalysis struct {
	fingerprint  string
	dependencies []string
	analysis     *resourceAnalysis
}

// SetTypegenTypeScript overrides the TypeScript installation typegen uses.
func (rb *ResourceBuilder) SetTypegenTypeScript(dir string) {
	rb.typegenTypeScript = dir
}

// analyzeResource is cached until a scanned file, a file they import or the tsconfig changes.
func (rb *ResourceBuilder) analyzeResource(resourcePath string) (*resourceAnalysis, error) {
	resourcePath = filepath.Clean(resourcePath)
	viewPath := rb.viewPathFor(resourcePath)
	var files []string
	if err := walkSourceFiles(resourcePath, viewPath, func(path string) error {
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return &resourceAnalysis{}, nil
	}

	tsconfig := filepath.Join(rb.projectPath, typecheckConfigFile)
	cacheKey := resourcePath + "\x00" + viewPath
	rb.analysisMutex.Lock()
	cached := rb.analysisCache[cacheKey]
	rb.analysisMutex.Unlock()
	if cached != nil && cached.fingerprint == fingerprintFiles(files, cached.dependencies, tsconfig) {
		return cached.analysis, nil
	}

	absFiles := make([]string, len(files))
	for i, file := range files {
		absFiles[i], _ = filepath.Abs(file)
	}
	response, err := rb.runAnalyzer(absFiles, tsconfig)
	if err != nil {
		return nil, err
	}

	analysis := &resourceAnalysis{}
	for i, file := range files {
		analysis.sources = append(analysis.sources, analyzedSource{path: file, result: response.Files[absFiles[i]]})
	}
	entry := &cachedAnalysis{analysis: analysis, dependencies: response.Dependencies}
	entry.fingerprint = fingerprintFiles(files, entry.dependencies, tsconfig)

	rb.analysisMutex.Lock()
	if rb.analysisCache == nil {
		rb.analysisCache = map[string]*cachedAnalysis{}
	}
	rb.analysisCache[cacheKey] = entry
	rb.analysisMutex.Unlock()
	return analysis, nil
}

type analyzerResponse struct {
	Files        map[string]analyzedFile `json:"files"`
	Dependencies []string                `json:"dependencies"`
}

func (rb *ResourceBuilder) runAnalyzer(files []string, tsconfig string) (*analyzerResponse, error) {
	buildScript, err := rb.ensureEmbeddedScript()
	if err != nil {
		return nil, err
	}
	script, err := filepath.Abs(filepath.Join(filepath.Dir(buildScript), "typegen.js"))
	if err != nil {
		return nil, err
	}
	projectPath, err := filepath.Abs(rb.projectPath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(tsconfig); err != nil {
		tsconfig = ""
	} else if tsconfig, err = filepath.Abs(tsconfig); err != nil {
		return nil, err
	}

	request, err := json.Marshal(map[string]any{
		"projectPath": projectPath,
		"tsconfig":    tsconfig,
		"typescript":  rb.typegenTypeScript,
		"files":       files,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", script)
	cmd.Dir = projectPath
	cmd.Stdin = bytes.NewReader(request)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("typegen analyzer failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var response analyzerResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("typegen analyzer returned invalid JSON: %w", err)
	}
	return &response, nil
}

func fingerprintFiles(files []string, dependencies []string, tsconfig string) string {
	hash := sha256.New()
	for _, file := range slices.Concat(files, dependencies, []string{tsconfig}) {
		fmt.Fprintf(hash, "%s\x00", file)
		if info, err := os.Stat(file); err == nil {
			fmt.Fprintf(hash, "%d\x00%d\x00", info.Size(), info.ModTime().UnixNano())
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// renderImportMarks rewrites the analyzer's absolute module paths relative to outDir.
func renderImportMarks(typeExpr string, outDir string) (string, error) {
	outDir, err := filepath.Abs(outDir)
	if err != nil {
		return "", err
	}
	var renderErr error
	rendered := importMarkPattern.ReplaceAllStringFunc(typeExpr, func(mark string) string {
		spec, err := moduleImportPath(outDir, filepath.FromSlash(strings.Trim(mark, "\x01"))+".ts")
		if err != nil {
			renderErr = err
		}
		return quoteTSString(spec)
	})
	return rendered, renderErr
}
