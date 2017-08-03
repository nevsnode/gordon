package utils

import (
	"os/exec"
	"syscall"
)

// ExecCommand is a wrapper arond exec.Command that adds commonly used properties/functonality
func ExecCommand(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)

	// set Setpgid to true, to execute command in different process group,
	// so it won't receive the interrupt-signals sent to the main go-application
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}
