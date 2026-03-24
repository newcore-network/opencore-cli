package watcher

import (
	"testing"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestNewRestarterUsesProcessInAutoMode(t *testing.T) {
	cfg := &config.Config{
		Dev: config.DevConfig{
			Restart: config.DevRestartConfig{Mode: "auto"},
			Process: config.DevProcessConfig{Command: "./server"},
		},
	}
	cfg.Dev.Normalize()

	restarter, err := newRestarter(cfg)
	if err != nil {
		t.Fatalf("newRestarter failed: %v", err)
	}
	if restarter.Mode() != "process" {
		t.Fatalf("expected process mode, got %s", restarter.Mode())
	}
}

func TestNewRestarterUsesTxAdminForFiveM(t *testing.T) {
	cfg := &config.Config{
		Adapter: &config.AdapterConfig{Server: &config.AdapterBinding{Name: "fivem"}},
		Dev: config.DevConfig{
			Restart: config.DevRestartConfig{Mode: "auto"},
			TxAdmin: config.DevTxAdminConfig{
				URL:      "http://localhost:40120",
				User:     "admin",
				Password: "secret",
			},
		},
	}
	cfg.Dev.Normalize()

	restarter, err := newRestarter(cfg)
	if err != nil {
		t.Fatalf("newRestarter failed: %v", err)
	}
	if restarter.Mode() != "txadmin" {
		t.Fatalf("expected txadmin mode, got %s", restarter.Mode())
	}
}

func TestNewRestarterFallsBackToNone(t *testing.T) {
	cfg := &config.Config{}
	cfg.Dev.Normalize()

	restarter, err := newRestarter(cfg)
	if err != nil {
		t.Fatalf("newRestarter failed: %v", err)
	}
	if restarter.Mode() != "none" {
		t.Fatalf("expected none mode, got %s", restarter.Mode())
	}
}
