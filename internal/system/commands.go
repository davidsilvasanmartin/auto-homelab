package system

import "log/slog"

type Commands interface {
	// ExecCommand executes a system terminal command
	ExecCommand(name string, arg ...string) RunnableCommand
	// ExecShellCommand executes a full shell command. The full command must be passed as a
	// string rather than as a slice of its arguments
	ExecShellCommand(command string) RunnableCommand
}

// DefaultCommands is the default implementation of the Commands interface
type DefaultCommands struct {
	stdlib stdlib
}

// NewDefaultCommands provides a default implementation of the Commands interface.
// It uses the system's os.Stdout and os.Stderr. It uses the current working directory
// as the directory where the commands are run from
func NewDefaultCommands() *DefaultCommands {
	return &DefaultCommands{
		stdlib: newGoStdlib(),
	}
}

func (s *DefaultCommands) ExecCommand(name string, arg ...string) RunnableCommand {
	slog.Debug("Executing command", "command", name, "arg", arg)
	return s.stdlib.ExecCommand(name, arg...)
}

func (s *DefaultCommands) ExecShellCommand(command string) RunnableCommand {
	return s.ExecCommand("sh", "-c", command)
}
