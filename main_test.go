package main

import "testing"

func TestShouldCheckForUpdatesDisabledByEnv(t *testing.T) {
	t.Setenv("OPENCORE_DISABLE_UPDATE_CHECK", "1")

	if shouldCheckForUpdates([]string{"opencore", "build"}) {
		t.Fatalf("expected update check to be disabled by env")
	}
}

func TestShouldCheckForUpdatesSkipsVersionAndUpdate(t *testing.T) {
	t.Setenv("OPENCORE_DISABLE_UPDATE_CHECK", "")

	if shouldCheckForUpdates([]string{"opencore", "update"}) {
		t.Fatalf("expected update command to skip update checks")
	}

	if shouldCheckForUpdates([]string{"opencore", "--version"}) {
		t.Fatalf("expected --version command to skip update checks")
	}
}
