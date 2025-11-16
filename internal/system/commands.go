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
	// TODO we are often logging passwords here. For example, run the "cloud backup list" command to see it.
	//  We should look into redacting passwords. Look at Navidrome's approach:
	//  https://github.com/navidrome/navidrome/blob/395a36e10f2d3f4af8cccbfa81b0da1e556a0d36/log/log.go#L24
	slog.Debug("Executing command", "command", name, "arg", arg)
	return s.stdlib.ExecCommand(name, arg...)
}

func (s *DefaultCommands) ExecShellCommand(command string) RunnableCommand {
	return s.ExecCommand("sh", "-c", command)
}
