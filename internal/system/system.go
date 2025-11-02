package system

import (
	"fmt"
	"os/exec"
)

type System interface {
	// RequireCommand requires that a command exists on the system, or errors if it doesn't
	RequireCommand(command string) error
	// ExecCommand executes a system terminal command
	ExecCommand(name string, arg ...string) *exec.Cmd
}

type DefaultSystem struct {
	stdlib stdlib
}

func NewDefaultSystem() *DefaultSystem {
	return &DefaultSystem{
		stdlib: newGoStdlib(),
	}
}

func (s *DefaultSystem) RequireCommand(command string) error {
	if _, err := s.stdlib.ExecLookPath(command); err != nil {
		return fmt.Errorf("%q command not found: %w (is %q installed in PATH?)", command, err, command)
	}
	return nil
}

func (s *DefaultSystem) ExecCommand(name string, arg ...string) *exec.Cmd {
	return s.stdlib.ExecCommand(name, arg...)
}
