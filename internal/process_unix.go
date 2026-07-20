//go:build !windows

package afr

import (
	"os/exec"
	"syscall"
)

type processTree struct{}

func prepareProcessTree(command *exec.Cmd) (*processTree, error) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processTree{}, nil
}

func (tree *processTree) afterStart(command *exec.Cmd) error {
	return nil
}

func (tree *processTree) interrupt(command *exec.Cmd) error {
	return syscall.Kill(-command.Process.Pid, syscall.SIGINT)
}

func (tree *processTree) killCommand(command *exec.Cmd) error {
	return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
}

func (tree *processTree) close() error {
	return nil
}
