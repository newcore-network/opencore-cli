//go:build !windows

package watcher

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/newcore-network/opencore-cli/internal/config"
)

func TestProcessRestarterStopsProcessGroup(t *testing.T) {
	r := newProcessRestarter(config.DevProcessConfig{
		Command:       "sh",
		Args:          []string{"-c", "sleep 30 & wait"},
		StopTimeoutMs: 500,
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	pid := r.cmd.Process.Pid

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		err := syscall.Kill(-pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("process group %d still exists: %v", pid, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
