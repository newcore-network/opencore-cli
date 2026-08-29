//go:build windows

package watcher

import (
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}

func stopProcessTree(cmd *exec.Cmd, _ string) error {
	return killProcessTree(cmd)
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
