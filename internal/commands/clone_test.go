package commands

import "testing"

func TestIsSafeItemName(t *testing.T) {
	safe := []string{"chat", "my-template", "my_template", "file.ts", "README.md"}
	for _, name := range safe {
		if !isSafeItemName(name) {
			t.Errorf("expected %q to be considered safe", name)
		}
	}

	unsafe := []string{"", ".", "..", "../escape", "a/b", "a\\b", "/etc/passwd"}
	for _, name := range unsafe {
		if isSafeItemName(name) {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}

func TestHTTPClientHasTimeout(t *testing.T) {
	if httpClient.Timeout <= 0 {
		t.Fatal("expected httpClient to have a positive timeout configured")
	}
}
