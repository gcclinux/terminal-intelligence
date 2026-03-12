//go:build windows

package agentic

import (
	"os"
	"os/exec"
)

func setProcGroupAttr(cmd *exec.Cmd) {
	// No-op on Windows; process group management is handled differently
	_ = cmd
}

func killProcessGroup(pid int) {
	p, err := os.FindProcess(pid)
	if err == nil {
		_ = p.Kill()
	}
}
