package pkgmgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreferenceFromProjectStrict(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"packageManager":"yarn@4.5.1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	choice, err := PreferenceFromProjectStrict(dir)
	if err != nil || choice != ChoiceYarn {
		t.Fatalf("expected Yarn Berry, got %q, %v", choice, err)
	}
}

func TestPreferenceFromProjectStrictRejectsInvalidDeclarations(t *testing.T) {
	for name, contents := range map[string]string{
		"malformed json":      `{`,
		"missing version":     `{"packageManager":"pnpm"}`,
		"unsupported manager": `{"packageManager":"bun@1.0.0"}`,
		"yarn classic":        `{"packageManager":"yarn@1.22.22"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := PreferenceFromProjectStrict(dir); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestYarnBerryCommands(t *testing.T) {
	resolved := Resolved{Choice: ChoiceYarn, Version: "4.5.1"}
	tests := map[string]string{
		resolved.InstallCmd():             "yarn install",
		resolved.AddDevCmd("vitest"):      "yarn add -D vitest",
		resolved.AddCmd("react"):          "yarn add react",
		resolved.ExecCmd("vite", "build"): "yarn dlx vite build",
	}
	for got, want := range tests {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestInvalidResolvedDoesNotFallBackToNpm(t *testing.T) {
	resolved := Resolved{Choice: Choice("bun")}
	if resolved.InstallCmd() != "" || resolved.AddCmd("pkg") != "" || resolved.ExecCmd("bin") != "" {
		t.Fatal("invalid package manager must not silently run npm")
	}
}
