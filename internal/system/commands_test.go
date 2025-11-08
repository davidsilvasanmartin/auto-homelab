package system

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestExecCommand_SetsCommandProperties tests that ExecCommand properly configures the command
func TestExecCommand_SetsCommandProperties(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	commands := &DefaultCommands{
		stdlib: std,
	}

	cmd := commands.ExecCommand("docker", "compose", "up", "-d", "service1")

	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "up", "-d", "service1"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

// TestExecCommand_NoArgs tests ExecCommand with no arguments
func TestExecCommand_NoArgs(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	commands := &DefaultCommands{stdlib: std}

	cmd := commands.ExecCommand("ls")
	if capturedName != "ls" {
		t.Errorf("expected command name %q, got %q", "ls", capturedName)
	}
	if len(capturedArgs) != 0 {
		t.Errorf("expected no args, got %v", capturedArgs)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

// TestExecShellCommand tests that ExecShellCommand calls ExecCommand correctly
func TestExecShellCommand(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	commands := &DefaultCommands{stdlib: std}

	cmd := commands.ExecShellCommand("echo hello && ls -la")

	if capturedName != "sh" {
		t.Errorf("expected command name %q, got %q", "sh", capturedName)
	}
	expectedArgs := []string{"-c", "echo hello && ls -la"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

// TestExecShellCommand_EmptyString tests ExecShellCommand with an empty string
func TestExecShellCommand_EmptyString(t *testing.T) {
	var capturedArgs []string
	std := &mockStdlib{
		execCommand: func(name string, arg ...string) RunnableCommand {
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	commands := &DefaultCommands{stdlib: std}

	commands.ExecShellCommand("")

	expectedArgs := []string{"-c", ""}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
}

// TestNewDefaultCommands tests that the constructor creates proper defaults
func TestNewDefaultCommands(t *testing.T) {
	commands := NewDefaultCommands()

	if commands == nil {
		t.Fatal("expected non-nil DefaultCommands")
	}

	if commands.stdlib == nil {
		t.Error("expected non-nil stdlib")
	}
}
