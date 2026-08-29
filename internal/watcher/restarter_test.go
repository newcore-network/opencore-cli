package watcher

import (
	"context"
	"errors"
	"testing"

	"github.com/newcore-network/opencore-cli/internal/builder"
	"github.com/newcore-network/opencore-cli/internal/config"
)

type failingRestarter struct {
	err error
}

func (r *failingRestarter) Mode() string                { return "process" }
func (r *failingRestarter) Start(context.Context) error { return nil }
func (r *failingRestarter) Restart([]string) error      { return r.err }
func (r *failingRestarter) Stop() error                 { return nil }

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

func TestNotifyFrameworkReturnsRestartFailure(t *testing.T) {
	w := &Watcher{}
	want := errors.New("restart unavailable")
	err := w.notifyFramework(&failingRestarter{err: want}, []builder.BuildResult{{
		Task:    builder.BuildTask{ResourceName: "core"},
		Success: true,
	}})
	if !errors.Is(err, want) {
		t.Fatalf("notifyFramework error = %v, want %v", err, want)
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
