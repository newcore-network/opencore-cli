package commands

import (
	"strings"
	"testing"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestCheckAdapterNoConfig(t *testing.T) {
	result := checkAdapter(&config.Config{})

	if !result.Passed {
		t.Fatal("expected missing adapter configuration to pass")
	}
	if !strings.Contains(result.Message, "No adapter configured") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestCheckAdapterValid(t *testing.T) {
	result := checkAdapter(&config.Config{
		Adapter: &config.AdapterConfig{
			Server: &config.AdapterBinding{Name: "fivem", Valid: true},
			Client: &config.AdapterBinding{Name: "fivem", Valid: true},
		},
	})

	if !result.Passed {
		t.Fatal("expected valid adapter configuration to pass")
	}
	if !strings.Contains(result.Message, "server: fivem (valid)") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
	if !strings.Contains(result.Message, "client: fivem (valid)") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestCheckAdapterInvalid(t *testing.T) {
	result := checkAdapter(&config.Config{
		Adapter: &config.AdapterConfig{
			Client: &config.AdapterBinding{Name: "unknown", Valid: false, Message: "missing register()"},
		},
	})

	if result.Passed {
		t.Fatal("expected invalid adapter configuration to fail")
	}
	if !strings.Contains(result.Message, "client: unknown (invalid: missing register())") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}
