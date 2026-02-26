package ui

import "testing"

func TestIsCI(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("BUILD_ID", "")

	if IsCI() {
		t.Fatalf("expected IsCI to be false without CI env vars")
	}

	t.Setenv("CI", "true")
	if !IsCI() {
		t.Fatalf("expected IsCI to be true when CI=true")
	}
}

func TestIsUpdateCheckDisabled(t *testing.T) {
	t.Setenv("OPENCORE_DISABLE_UPDATE_CHECK", "1")
	if !IsUpdateCheckDisabled() {
		t.Fatalf("expected update check to be disabled")
	}

	t.Setenv("OPENCORE_DISABLE_UPDATE_CHECK", "0")
	if IsUpdateCheckDisabled() {
		t.Fatalf("expected update check to be enabled")
	}
}
