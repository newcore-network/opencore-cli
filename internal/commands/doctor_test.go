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
			Server: &config.AdapterBinding{Name: "fivem", Valid: true, Runtime: &config.AdapterRuntimeBinding{Runtime: "fivem"}},
			Client: &config.AdapterBinding{Name: "fivem", Valid: true, Runtime: &config.AdapterRuntimeBinding{Runtime: "fivem"}},
		},
	})

	if !result.Passed {
		t.Fatal("expected valid adapter configuration to pass")
	}
	if !strings.Contains(result.Message, "server: fivem [fivem] (valid)") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
	if !strings.Contains(result.Message, "client: fivem [fivem] (valid)") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestCheckAdapterInvalid(t *testing.T) {
	result := checkAdapter(&config.Config{
		Adapter: &config.AdapterConfig{
			Client: &config.AdapterBinding{Name: "unknown", Valid: false, Message: "missing register()", Runtime: &config.AdapterRuntimeBinding{Runtime: "ragemp"}},
		},
	})

	if result.Passed {
		t.Fatal("expected invalid adapter configuration to fail")
	}
	if !strings.Contains(result.Message, "client: unknown [ragemp] (invalid: missing register())") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}
