package commands

import (
	"errors"
	"testing"
)

func TestUpdateArgs(t *testing.T) {
	tests := []struct {
		manager string
		channel string
		want    []string
	}{
		{"npm", "stable", []string{"update", "--global", "@open-core/cli@latest"}},
		{"pnpm", "beta", []string{"update", "--global", "@open-core/cli@beta"}},
		{"yarn", "stable", []string{"global", "add", "@open-core/cli@latest"}},
	}

	for _, tt := range tests {
		t.Run(tt.manager+"-"+tt.channel, func(t *testing.T) {
			got, err := updateArgs(tt.manager, tt.channel)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("args = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("args = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestPackageManagerFromUserAgent(t *testing.T) {
	tests := map[string]string{
		"npm/11.0.0 node/v24": "npm",
		"pnpm/10.0.0 npm/?":   "pnpm",
		"yarn/1.22.22 npm/?":  "yarn",
		"unknown/1.0.0 npm/?": "",
	}
	for userAgent, want := range tests {
		if got := packageManagerFromUserAgent(userAgent); got != want {
			t.Fatalf("packageManagerFromUserAgent(%q) = %q, want %q", userAgent, got, want)
		}
	}
}

func TestUpdateCommandUsesSelectedPackageManager(t *testing.T) {
	originalFind := findPackageManager
	originalRun := runPackageManager
	t.Cleanup(func() {
		findPackageManager = originalFind
		runPackageManager = originalRun
	})

	findPackageManager = func(name string) (string, error) { return name, nil }
	var gotName string
	var gotArgs []string
	runPackageManager = func(name string, args []string) error {
		gotName = name
		gotArgs = args
		return nil
	}

	cmd := NewUpdateCommand()
	cmd.SetArgs([]string{"--package-manager", "pnpm", "--channel", "beta"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if gotName != "pnpm" {
		t.Fatalf("manager = %q, want pnpm", gotName)
	}
	if len(gotArgs) != 3 || gotArgs[2] != "@open-core/cli@beta" {
		t.Fatalf("args = %v", gotArgs)
	}
}

func TestResolvePackageManagerRejectsUnavailableExplicitManager(t *testing.T) {
	originalFind := findPackageManager
	t.Cleanup(func() { findPackageManager = originalFind })
	findPackageManager = func(string) (string, error) { return "", errors.New("not found") }

	if _, err := resolvePackageManager("yarn"); err == nil {
		t.Fatal("expected unavailable package manager error")
	}
}
