package watcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/newcore-network/opencore-cli/internal/config"
	"github.com/newcore-network/opencore-cli/internal/watcher/txadmin"
)

type restarter interface {
	Mode() string
	Start(context.Context) error
	Restart([]string) error
	Stop() error
}

type noopRestarter struct{}

func (r *noopRestarter) Mode() string                { return "none" }
func (r *noopRestarter) Start(context.Context) error { return nil }
func (r *noopRestarter) Restart([]string) error      { return nil }
func (r *noopRestarter) Stop() error                 { return nil }

type txAdminRestarter struct {
	client *txadmin.Client
}

func (r *txAdminRestarter) Mode() string                { return "txadmin" }
func (r *txAdminRestarter) Start(context.Context) error { return r.client.Login() }
func (r *txAdminRestarter) Stop() error                 { return nil }

func (r *txAdminRestarter) Restart(resources []string) error {
	if len(resources) == 0 {
		return nil
	}

	if err := r.client.RefreshResources(); err != nil {
		if !isTxAdminAuthError(err) {
			return err
		}
		if err := r.client.Login(); err != nil {
			return err
		}
		if err := r.client.RefreshResources(); err != nil {
			return err
		}
	}

	for _, resourceName := range resources {
		if err := r.client.RestartResource(resourceName); err != nil {
			if !isTxAdminAuthError(err) {
				return err
			}
			if err := r.client.Login(); err != nil {
				return err
			}
			if err := r.client.RestartResource(resourceName); err != nil {
				return err
			}
		}
	}

	return nil
}

type processRestarter struct {
	config      config.DevProcessConfig
	stdout      *os.File
	stderr      *os.File
	mu          sync.Mutex
	ctx         context.Context
	cancelWatch context.CancelFunc
	cmd         *exec.Cmd
	running     bool
}

func newProcessRestarter(cfg config.DevProcessConfig) *processRestarter {
	return &processRestarter{config: cfg, stdout: os.Stdout, stderr: os.Stderr}
}

func (r *processRestarter) Mode() string { return "process" }

func (r *processRestarter) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}
	r.ctx, r.cancelWatch = context.WithCancel(ctx)
	return r.startLocked()
}

func (r *processRestarter) Restart(_ []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ctx == nil {
		r.ctx = context.Background()
	}

	if err := r.stopLocked(); err != nil {
		return err
	}
	return r.startLocked()
}

func (r *processRestarter) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancelWatch != nil {
		r.cancelWatch()
		r.cancelWatch = nil
	}
	return r.stopLocked()
}

func (r *processRestarter) startLocked() error {
	command := strings.TrimSpace(r.config.Command)
	if command == "" {
		return nil
	}

	cmd := exec.CommandContext(r.ctx, command, r.config.Args...)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	cmd.Stdin = os.Stdin

	if cwd := strings.TrimSpace(r.config.Cwd); cwd != "" {
		if !filepath.IsAbs(cwd) {
			cwd = filepath.Clean(cwd)
		}
		cmd.Dir = cwd
	}

	if len(r.config.Env) > 0 {
		cmd.Env = os.Environ()
		for key, value := range r.config.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	r.cmd = cmd
	r.running = true

	go func() {
		err := cmd.Wait()
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.cmd == cmd {
			r.cmd = nil
			r.running = false
		}
		if err != nil && !errors.Is(err, context.Canceled) && r.ctx.Err() == nil {
			fmt.Fprintf(r.stderr, "[opencore dev] managed process exited: %v\n", err)
		}
	}()

	return nil
}

func (r *processRestarter) stopLocked() error {
	if r.cmd == nil || r.cmd.Process == nil || !r.running {
		r.cmd = nil
		r.running = false
		return nil
	}

	proc := r.cmd.Process
	cmd := r.cmd
	stopTimeout := time.Duration(r.config.StopTimeoutMs) * time.Millisecond
	if stopTimeout <= 0 {
		stopTimeout = 5 * time.Second
	}

	if err := sendStopSignal(proc, r.config.StopSignal); err != nil {
		_ = proc.Kill()
	}

	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if !processRunning(cmd.ProcessState, proc.Pid) {
			r.cmd = nil
			r.running = false
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}

	r.cmd = nil
	r.running = false
	return nil
}

func processRunning(state *os.ProcessState, pid int) bool {
	if state != nil && state.Exited() {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return proc.Signal(syscall.Signal(0)) == nil
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func sendStopSignal(proc *os.Process, signalName string) error {
	if proc == nil {
		return nil
	}

	sig := strings.ToUpper(strings.TrimSpace(signalName))
	switch sig {
	case "", "SIGTERM", "TERM":
		if runtime.GOOS == "windows" {
			return proc.Kill()
		}
		return proc.Signal(syscall.SIGTERM)
	case "SIGINT", "INT":
		if runtime.GOOS == "windows" {
			return proc.Signal(os.Interrupt)
		}
		return proc.Signal(os.Interrupt)
	case "SIGKILL", "KILL":
		return proc.Kill()
	default:
		if runtime.GOOS == "windows" {
			return proc.Kill()
		}
		return proc.Signal(syscall.SIGTERM)
	}
}

func newRestarter(cfg *config.Config) (restarter, error) {
	mode := cfg.Dev.RestartMode()
	runtimeKind := cfg.RuntimeKind()

	switch mode {
	case "none":
		return &noopRestarter{}, nil
	case "process":
		if !cfg.Dev.HasManagedProcess() {
			return nil, fmt.Errorf("dev.process.command is required when dev.restart.mode is 'process'")
		}
		return newProcessRestarter(cfg.Dev.Process), nil
	case "txadmin":
		if !cfg.Dev.IsTxAdminConfigured() {
			return nil, fmt.Errorf("dev.txAdmin.url, dev.txAdmin.user and dev.txAdmin.password are required when dev.restart.mode is 'txadmin'")
		}
		client, err := txadmin.NewClient(cfg.Dev.TxAdmin.URL, cfg.Dev.TxAdmin.User, cfg.Dev.TxAdmin.Password)
		if err != nil {
			return nil, err
		}
		return &txAdminRestarter{client: client}, nil
	case "auto", "":
		if cfg.Dev.HasManagedProcess() {
			return newProcessRestarter(cfg.Dev.Process), nil
		}
		if (runtimeKind == "fivem" || runtimeKind == "redm") && cfg.Dev.IsTxAdminConfigured() {
			client, err := txadmin.NewClient(cfg.Dev.TxAdmin.URL, cfg.Dev.TxAdmin.User, cfg.Dev.TxAdmin.Password)
			if err != nil {
				return nil, err
			}
			return &txAdminRestarter{client: client}, nil
		}
		return &noopRestarter{}, nil
	default:
		return nil, fmt.Errorf("unknown dev.restart.mode %q", mode)
	}
}

func isTxAdminAuthError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "status 401") ||
		strings.Contains(errMsg, "status 403")
}
