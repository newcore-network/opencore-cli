package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefineViewPayloads_SkippedWhenNothingWasLost(t *testing.T) {
	rb := NewResourceBuilder(".")

	entries := []viewEntry{
		{EventName: "a:one", PayloadType: "Parameters<V0_X['m']>[0]"},
		{EventName: "a:two", PayloadType: "undefined"},
	}

	refined, warnings := rb.refineViewPayloads(
		"resources/a", "resources/a/ui", "resources/a/ui/.opencore",
		[]string{"resources/a/src/client/a.ts"},
		entries, nil,
	)

	if len(refined) != len(entries) || refined[0].PayloadType != entries[0].PayloadType {
		t.Fatalf("entries must be returned untouched, got %v", refined)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	// A populated cache would mean the pass ran.
	if len(rb.viewPayloadCache) != 0 {
		t.Fatal("the checker must not run when every payload is already resolved")
	}
}

func TestRefineViewPayloads_FallsBackWhenCheckerUnavailable(t *testing.T) {
	rb := NewResourceBuilder(t.TempDir())

	entries := []viewEntry{{EventName: "a:one", PayloadType: "unknown"}}
	original := []SourceValidationIssue{{
		File:    "src/client/a.ts",
		Line:    3,
		Message: viewPayloadUnresolvedPrefix + " \"a:one\"; …",
	}}

	// The embedded script is never extracted for this builder, so the pass cannot start.
	refined, warnings := rb.refineViewPayloads(
		"resources/a", "resources/a/ui", "resources/a/ui/.opencore",
		[]string{filepath.Join(t.TempDir(), "missing.ts")},
		entries, original,
	)

	if len(refined) != 1 || refined[0].PayloadType != "unknown" {
		t.Fatalf("expected the regex entries to survive, got %v", refined)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected the original warning to survive, got %v", warnings)
	}
}

func TestPruneUnusedControllerImports(t *testing.T) {
	imports := map[string]string{
		"V0_Used":   "../../src/client/used.controller",
		"V1_Unused": "../../src/client/unused.controller",
	}

	sends := []viewEntry{{EventName: "a", PayloadType: "__Payload<Parameters<V0_Used['onA']>>"}}
	receives := []viewEntry{{EventName: "b", PayloadType: `import("../../src/shared/x").B`}}

	pruned := pruneUnusedControllerImports(imports, sends, receives)

	if _, kept := pruned["V0_Used"]; !kept {
		t.Fatal("an alias a payload refers to must be kept")
	}
	if _, kept := pruned["V1_Unused"]; kept {
		t.Fatal("an alias nothing refers to must be dropped")
	}
}

func TestViewPayloadCache_InvalidatedByContentChange(t *testing.T) {
	rb := NewResourceBuilder(".")

	dir := t.TempDir()
	file := filepath.Join(dir, "a.ts")
	if err := os.WriteFile(file, []byte("const a = 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	entries := []viewEntry{{EventName: "a:one", PayloadType: "Payload"}}
	rb.storeViewPayloads("resources/a", []string{file}, entries, nil, true)

	if _, found := rb.cachedViewPayloads("resources/a", []string{file}); !found {
		t.Fatal("expected a cache hit for unchanged contents")
	}

	if err := os.WriteFile(file, []byte("const a = 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, found := rb.cachedViewPayloads("resources/a", []string{file}); found {
		t.Fatal("expected the cache to miss after the file changed")
	}
}

func TestViewCheckerScriptIsEmbedded(t *testing.T) {
	rb := NewResourceBuilder(t.TempDir())

	scriptPath, err := rb.ensureEmbeddedScript()
	if err != nil {
		t.Fatalf("failed to extract embedded scripts: %v", err)
	}
	defer rb.Cleanup()

	checkerPath := filepath.Join(filepath.Dir(scriptPath), viewCheckerScriptName)
	content, err := os.ReadFile(checkerPath)
	if err != nil {
		t.Fatalf("expected %s to be extracted alongside build.js: %v", viewCheckerScriptName, err)
	}
	if !strings.Contains(string(content), "createProgram") {
		t.Fatal("the extracted script is not the checker pass")
	}
}
