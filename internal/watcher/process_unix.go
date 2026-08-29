//go:build !windows

package watcher

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func stopProcessTree(cmd *exec.Cmd, signalName string) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	var signal syscall.Signal
	switch strings.ToUpper(strings.TrimSpace(signalName)) {
	case "SIGINT", "INT":
		signal = syscall.SIGINT
	case "SIGKILL", "KILL":
		signal = syscall.SIGKILL
	default:
		signal = syscall.SIGTERM
	}
	return signalProcessGroup(cmd.Process.Pid, signal)
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return signalProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
}

func signalProcessGroup(pid int, signal syscall.Signal) error {
	err := syscall.Kill(-pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
