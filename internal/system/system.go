package system

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type System interface {
	// RequireCommand requires that a command exists on the system, or errors if it doesn't
	RequireCommand(command string) error
	// ExecCommand executes a system terminal command
	ExecCommand(name string, arg ...string) *exec.Cmd
	// RequireFilesInWd requires that the files exist in the current working directory, or errors if they don't
	RequireFilesInWd(filenames ...string) error
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

func (s *DefaultSystem) RequireFilesInWd(filenames ...string) error {
	if len(filenames) == 0 {
		return nil
	}

	cwd, err := s.stdlib.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var missingFiles []string
	for _, file := range filenames {
		path := fmt.Sprintf("%s/%s", cwd, file)
		if _, err := s.stdlib.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missingFiles = append(missingFiles, file)
			} else {
				// Some other error returned (permission denied, etc.)
				return fmt.Errorf("failed to check file %s: %w", path, err)
			}
		}
	}

	if len(missingFiles) > 0 {
		return fmt.Errorf("required filenames not found: %s", strings.Join(missingFiles, ", "))
	}

	return nil
}
