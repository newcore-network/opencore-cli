package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestParseRegisteredTokens(t *testing.T) {
	source := `
	ctx.bindMessagingTransport(new CustomTransport())
	ctx.bindSingleton(IHasher as InjectionToken<IHasher>, CustomHasher)
	ctx.bindInstance(IEngineEvents as InjectionToken<IEngineEvents>, resolveEvents())
	ctx.bindFactory(IClientSpawnBridge as InjectionToken<IClientSpawnBridge>, () => thing)
	ctx.useRuntimeBridge(new RuntimeBridge())
	`

	tokens := parseRegisteredTokens(source)
	expected := []string{
		"EventsAPI",
		"IClientRuntimeBridge",
		"IClientSpawnBridge",
		"IEngineEvents",
		"IHasher",
		"MessagingTransport",
		"RpcAPI",
	}

	if !slices.Equal(tokens, expected) {
		t.Fatalf("unexpected tokens\nwant: %#v\ngot:  %#v", expected, tokens)
	}
}

func TestInspectAdapterProjectFiveM(t *testing.T) {
	projectRoot := siblingRepoPath(t, "opencore-fivem-adapter")
	report, err := inspectAdapterProject(projectRoot)
	if err != nil {
		t.Fatalf("inspectAdapterProject failed: %v", err)
	}

	if report.PackageName != "@open-core/fivem-adapter" {
		t.Fatalf("unexpected package name: %s", report.PackageName)
	}
	if report.Client == nil {
		t.Fatal("expected client report")
	}
	if len(report.Client.MissingRequired) != 0 {
		t.Fatalf("expected no required client gaps, got %#v", report.Client.MissingRequired)
	}
	if !slices.Equal(report.Client.MissingOptional, []string{"IClientLogConsole"}) {
		t.Fatalf("unexpected optional client gaps: %#v", report.Client.MissingOptional)
	}
	if report.hasFailures(false) {
		t.Fatal("expected compat mode to pass for fivem adapter")
	}
	if !report.hasFailures(true) {
		t.Fatal("expected strict mode to fail for fivem adapter")
	}
}

func TestInspectAdapterProjectRageMP(t *testing.T) {
	projectRoot := siblingRepoPath(t, "opencore-ragemp-adapter")
	report, err := inspectAdapterProject(projectRoot)
	if err != nil {
		t.Fatalf("inspectAdapterProject failed: %v", err)
	}

	if report.PackageName != "@open-core/ragemp-adapter" {
		t.Fatalf("unexpected package name: %s", report.PackageName)
	}
	if report.Server == nil {
		t.Fatal("expected server report")
	}
	if len(report.Server.MissingRequired) != 0 {
		t.Fatalf("expected no required server gaps, got %#v", report.Server.MissingRequired)
	}
	if !slices.Equal(report.Server.MissingOptional, []string{"IPedAppearanceServer"}) {
		t.Fatalf("unexpected optional server gaps: %#v", report.Server.MissingOptional)
	}
	if report.hasFailures(false) {
		t.Fatal("expected compat mode to pass for ragemp adapter")
	}
	if !report.hasFailures(true) {
		t.Fatal("expected strict mode to fail for ragemp adapter")
	}
}

func siblingRepoPath(t *testing.T, name string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	path := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", name))
	if _, err := os.Stat(path); err != nil {
		t.Skipf("repo %s not available: %v", name, err)
	}
	return path
}
