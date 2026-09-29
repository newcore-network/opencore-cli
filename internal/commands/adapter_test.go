package commands

import (
	"os"
	"path/filepath"
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
	projectRoot := newAdapterProjectFixture(t, "@open-core/fivem-adapter", "client", `
ctx.bindSingleton(IClientRuntimeBridge, RuntimeBridge)
ctx.bindSingleton(IClientLogConsole, LogConsole)
`, `
ctx.bindSingleton(IClientRuntimeBridge, RuntimeBridge)
`)
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
	projectRoot := newAdapterProjectFixture(t, "@open-core/ragemp-adapter", "server", `
ctx.bindSingleton(IRageMPServerAdapter, ServerAdapter)
ctx.bindSingleton(IPedAppearanceServer, PedAppearanceServer)
`, `
ctx.bindSingleton(IRageMPServerAdapter, ServerAdapter)
`)
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

func newAdapterProjectFixture(t *testing.T, packageName, side, baseline, registered string) string {
	t.Helper()
	projectRoot := t.TempDir()
	writeAdapterTestFile(t, projectRoot, "package.json", `{"name":"`+packageName+`","exports":{"./`+side+`":"./`+side+`.js"}}`)
	frameworkRoot := filepath.Join(projectRoot, "node_modules", "@open-core", "framework")
	writeAdapterTestFile(t, frameworkRoot, "package.json", `{"name":"@open-core/framework"}`)
	writeAdapterTestFile(t, frameworkRoot,
		filepath.Join("src", "runtime", side, "adapter", "node-"+side+"-adapter.ts"), baseline)
	writeAdapterTestFile(t, projectRoot,
		filepath.Join("src", side, "create-fixture-"+side+"-adapter.ts"), registered)
	return projectRoot
}

func writeAdapterTestFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}
}
