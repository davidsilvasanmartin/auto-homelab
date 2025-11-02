package docker

import (
	"testing"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
	"github.com/google/go-cmp/cmp"
)

// mockRunnableCommand is a simple mock for RunnableCommand
type mockRunnableCommand struct {
	runFunc func() error
}

func (m *mockRunnableCommand) Run() error {
	if m.runFunc != nil {
		return m.runFunc()
	}
	return nil
}

type mockCommands struct {
	execCommand func(name string, arg ...string) system.RunnableCommand
}

func (m *mockCommands) ExecCommand(name string, arg ...string) system.RunnableCommand {
	if m.execCommand != nil {
		return m.execCommand(name, arg...)
	}
	return nil
}

func (m *mockCommands) ExecShellCommand(command string) system.RunnableCommand {
	return nil
}

type mockFiles struct {
	requireFilesInWd func(filenames ...string) error
}

func (m *mockFiles) CreateDirIfNotExists(path string) error {
	return nil
}

func (m *mockFiles) RequireFilesInWd(filenames ...string) error {
	if m.requireFilesInWd != nil {
		return m.requireFilesInWd(filenames...)
	}
	return nil
}

func (m *mockFiles) RequireDir(path string) error {
	return nil
}

func (m *mockFiles) EmptyDir(path string) error {
	return nil
}

func (m *mockFiles) CopyDir(srcPath string, dstPath string) error {
	return nil
}

// Test that ComposeStart with no services runs: docker compose up -d
func TestSystemRunner_ComposeStart_NoServices(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	commands := &mockCommands{
		execCommand: func(name string, arg ...string) system.RunnableCommand {
			capturedName = name
			capturedArgs = arg
			// This creates a command but doesn't execute it
			// When Run() is called, it just runs "echo" which is harmless and fast
			return &mockRunnableCommand{}
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		commands: commands,
		files:    files,
	}

	err := runner.ComposeStart([]string{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "up", "-d"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}

// Test that ComposeStart with one service runs: docker compose up -d service
func TestSystemRunner_ComposeStart_OneService(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	commands := &mockCommands{
		execCommand: func(name string, arg ...string) system.RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		commands: commands,
		files:    files,
	}

	err := runner.ComposeStart([]string{"service"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "up", "-d", "service"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}

// Test that ComposeStart with multiple services runs: docker compose up -d service1 service2
func TestSystemRunner_ComposeStart_MultipleServices(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	mock := &mockCommands{
		execCommand: func(name string, arg ...string) system.RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		commands: mock,
		files:    files,
	}

	err := runner.ComposeStart([]string{"service1", "service2"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "up", "-d", "service1", "service2"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}

// Test that ComposeStop with no services runs: docker compose stop
func TestSystemRunner_ComposeStop_NoServices(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	mock := &mockCommands{
		execCommand: func(name string, arg ...string) system.RunnableCommand {
			capturedName = name
			capturedArgs = arg
			// This creates a command but doesn't execute it
			// When Run() is called, it just runs "echo" which is harmless and fast
			return &mockRunnableCommand{}
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		commands: mock,
		files:    files,
	}

	err := runner.ComposeStop([]string{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "stop"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}

// Test that ComposeStop with one service runs: docker compose stop service
func TestSystemRunner_ComposeStop_OneService(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	mock := &mockCommands{
		execCommand: func(name string, arg ...string) system.RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		commands: mock,
		files:    files,
	}

	err := runner.ComposeStop([]string{"service"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "stop", "service"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}

// Test that ComposeStop with multiple services runs: docker compose stop service1 service2
func TestSystemRunner_ComposeStop_MultipleServices(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	mock := &mockCommands{
		execCommand: func(name string, arg ...string) system.RunnableCommand {
			capturedName = name
			capturedArgs = arg
			return &mockRunnableCommand{}
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		commands: mock,
		files:    files,
	}

	err := runner.ComposeStop([]string{"service1", "service2"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedName != "docker" {
		t.Errorf("expected command name %q, got %q", "docker", capturedName)
	}
	expectedArgs := []string{"compose", "stop", "service1", "service2"}
	if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
		t.Errorf("args mismatch:\n%s", diff)
	}
}
