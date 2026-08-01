package builder

import (
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

// identifierPattern matches a bare JavaScript identifier, used to spot alias references inside a rendered type expression.
var identifierPattern = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

const viewCheckerScriptName = "viewtypes.js"
const viewCheckerTimeout = 90 * time.Second

type viewCheckerResource struct {
	ResourcePath string   `json:"resourcePath"`
	ViewPath     string   `json:"viewPath"`
	OutDir       string   `json:"outDir"`
	ProjectRoot  string   `json:"projectRoot"`
	Files        []string `json:"files"`
}

type viewCheckerRequest struct {
	Resources []viewCheckerResource `json:"resources"`
}

type viewCheckerPayload struct {
	Event   string `json:"event"`
	Payload string `json:"payload"`
}

type viewCheckerWarning struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type viewCheckerResult struct {
	ResourcePath string               `json:"resourcePath"`
	Failed       bool                 `json:"failed"`
	Reason       string               `json:"reason"`
	UiReceives   []viewCheckerPayload `json:"uiReceives"`
	Warnings     []viewCheckerWarning `json:"warnings"`
}

type viewCheckerResponse struct {
	OK        bool                `json:"ok"`
	Reason    string              `json:"reason"`
	Resources []viewCheckerResult `json:"resources"`
}

func (rb *ResourceBuilder) resolveViewPayloads(
	resourcePath string,
	viewPath string,
	outDir string,
	files []string,
) ([]viewEntry, []SourceValidationIssue, bool) {
	if len(files) == 0 {
		return nil, nil, false
	}

	if cached, found := rb.cachedViewPayloads(resourcePath, files); found {
		return cached.entries, cached.warnings, cached.ok
	}

	entries, warnings, ok := rb.runViewChecker(resourcePath, viewPath, outDir, files)
	rb.storeViewPayloads(resourcePath, files, entries, warnings, ok)
	return entries, warnings, ok
}

func (rb *ResourceBuilder) runViewChecker(
	resourcePath string,
	viewPath string,
	outDir string,
	files []string,
) ([]viewEntry, []SourceValidationIssue, bool) {
	checkerPath, found := rb.viewCheckerScriptPath()
	if !found {
		return nil, nil, false
	}

	request := viewCheckerRequest{Resources: []viewCheckerResource{{
		ResourcePath: resourcePath,
		ViewPath:     viewPath,
		OutDir:       outDir,
		ProjectRoot:  absOrSelf(rb.projectPath),
		Files:        mapSlice(files, absOrSelf),
	}}}

	result, ok := rb.invokeViewChecker(checkerPath, request)
	if !ok {
		return nil, nil, false
	}

	return checkerEntries(result.UiReceives), checkerWarnings(result.Warnings), true
}

func (rb *ResourceBuilder) viewCheckerScriptPath() (string, bool) {
	scriptPath, err := rb.ensureEmbeddedScript()
	if err != nil {
		return "", false
	}
	checkerPath := filepath.Join(filepath.Dir(scriptPath), viewCheckerScriptName)
	if _, err := os.Stat(checkerPath); err != nil {
		return "", false
	}
	return checkerPath, true
}

func (rb *ResourceBuilder) invokeViewChecker(
	checkerPath string,
	request viewCheckerRequest,
) (viewCheckerResult, bool) {
	payload, err := json.Marshal(request)
	if err != nil {
		return viewCheckerResult{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), viewCheckerTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", checkerPath, string(payload))
	cmd.Dir = rb.projectPath

	output, err := cmd.Output()
	if err != nil {
		return viewCheckerResult{}, false
	}

	var response viewCheckerResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return viewCheckerResult{}, false
	}
	if !response.OK || len(response.Resources) == 0 || response.Resources[0].Failed {
		return viewCheckerResult{}, false
	}

	return response.Resources[0], true
}

func checkerEntries(received []viewCheckerPayload) []viewEntry {
	var entries []viewEntry
	for _, message := range received {
		if message.Event != "" && message.Payload != "" {
			entries = append(entries, viewEntry{
				EventName:   message.Event,
				PayloadType: message.Payload,
			})
		}
	}
	return entries
}

func checkerWarnings(warnings []viewCheckerWarning) []SourceValidationIssue {
	return mapSlice(warnings, func(w viewCheckerWarning) SourceValidationIssue {
		return SourceValidationIssue{File: w.File, Line: w.Line, Message: w.Message}
	})
}

// absOrSelf makes a path absolute, keeping it as-is when that is not possible.
func absOrSelf(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func mapSlice[In, Out any](in []In, convert func(In) Out) []Out {
	if len(in) == 0 {
		return nil
	}
	out := make([]Out, 0, len(in))
	for _, item := range in {
		out = append(out, convert(item))
	}
	return out
}

type viewPayloadCacheEntry struct {
	fingerprint string
	entries     []viewEntry
	warnings    []SourceValidationIssue
	ok          bool
}

func (rb *ResourceBuilder) cachedViewPayloads(resourcePath string, files []string) (viewPayloadCacheEntry, bool) {
	fingerprint := fingerprintFiles(files)
	if fingerprint == "" {
		return viewPayloadCacheEntry{}, false
	}

	rb.viewPayloadMutex.Lock()
	defer rb.viewPayloadMutex.Unlock()

	cached, found := rb.viewPayloadCache[resourcePath]
	if !found || cached.fingerprint != fingerprint {
		return viewPayloadCacheEntry{}, false
	}
	return cached, true
}

func (rb *ResourceBuilder) storeViewPayloads(
	resourcePath string,
	files []string,
	entries []viewEntry,
	warnings []SourceValidationIssue,
	ok bool,
) {
	fingerprint := fingerprintFiles(files)
	if fingerprint == "" {
		return
	}

	rb.viewPayloadMutex.Lock()
	defer rb.viewPayloadMutex.Unlock()

	if rb.viewPayloadCache == nil {
		rb.viewPayloadCache = map[string]viewPayloadCacheEntry{}
	}
	rb.viewPayloadCache[resourcePath] = viewPayloadCacheEntry{
		fingerprint: fingerprint,
		entries:     entries,
		warnings:    warnings,
		ok:          ok,
	}
}

func fingerprintFiles(files []string) string {
	hash := sha256.New()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return ""
		}
		fmt.Fprintf(hash, "%s:%d:", filepath.ToSlash(file), len(content))
		hash.Write(content)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (rb *ResourceBuilder) refineViewPayloads(
	resourcePath string,
	viewPath string,
	outDir string,
	files []string,
	uiReceives []viewEntry,
	warnings []SourceValidationIssue,
) ([]viewEntry, []SourceValidationIssue) {
	if !scanLostSomething(uiReceives, warnings) {
		return uiReceives, warnings
	}

	resolved, checkerWarnings, ok := rb.resolveViewPayloads(resourcePath, viewPath, outDir, files)
	if !ok {
		return uiReceives, warnings
	}

	return resolved, append(withoutSupersededWarnings(warnings), checkerWarnings...)
}

func scanLostSomething(uiReceives []viewEntry, warnings []SourceValidationIssue) bool {
	lostPayload := slices.ContainsFunc(uiReceives, func(e viewEntry) bool {
		return e.PayloadType == "unknown"
	})
	lostName := slices.ContainsFunc(warnings, func(w SourceValidationIssue) bool {
		return strings.HasPrefix(w.Message, viewSendNameUnresolvedPrefix)
	})
	return lostPayload || lostName
}

func withoutSupersededWarnings(warnings []SourceValidationIssue) []SourceValidationIssue {
	return slices.DeleteFunc(slices.Clone(warnings), func(w SourceValidationIssue) bool {
		return strings.HasPrefix(w.Message, viewPayloadUnresolvedPrefix) ||
			strings.HasPrefix(w.Message, viewSendNameUnresolvedPrefix)
	})
}

func pruneUnusedControllerImports(
	controllerImports map[string]string,
	entries ...[]viewEntry,
) map[string]string {
	referenced := map[string]bool{}
	for _, entry := range slices.Concat(entries...) {
		for _, identifier := range identifierPattern.FindAllString(entry.PayloadType, -1) {
			referenced[identifier] = true
		}
	}

	pruned := make(map[string]string, len(controllerImports))
	for alias, importPath := range controllerImports {
		if referenced[alias] {
			pruned[alias] = importPath
		}
	}
	return pruned
}
