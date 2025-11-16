package docker

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
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
	execShellCommand func(cmd string) system.RunnableCommand
}

func (m *mockCommands) ExecCommand(name string, arg ...string) system.RunnableCommand {
	return nil
}
func (m *mockCommands) ExecShellCommand(cmd string) system.RunnableCommand {
	if m.execShellCommand != nil {
		return m.execShellCommand(cmd)
	}
	return nil
}

type mockFiles struct {
	ensureFilesInWD func(filenames ...string) error
}

func (m *mockFiles) CreateDirIfNotExists(path string) error {
	return nil
}
func (m *mockFiles) EnsureFilesInWD(filenames ...string) error {
	if m.ensureFilesInWD != nil {
		return m.ensureFilesInWD(filenames...)
	}
	return nil
}
func (m *mockFiles) EnsureDirExists(path string) error {
	return nil
}
func (m *mockFiles) EmptyDir(path string) error {
	return nil
}
func (m *mockFiles) CopyDir(srcPath string, dstPath string) error {
	return nil
}
func (m *mockFiles) Getwd() (dir string, err error)           { return "", nil }
func (m *mockFiles) WriteFile(path string, data []byte) error { return nil }
func (m *mockFiles) GetAbsPath(path string) (string, error)   { return "", nil }

type mockTime struct{}

func (t *mockTime) Sleep(d time.Duration) {
	// noop
}

func mockBuildDockerComposeCommandStr(cmd string) string {
	return "docker compose " + cmd
}

func TestSystemRunner_ComposeStart_NoServices(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ComposeStart([]string{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "docker compose up -d"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got %q", expectedCmd, capturedCmd)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || len(s) > len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSystemRunner_ComposeStart_OneService(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ComposeStart([]string{"service"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "docker compose up -d service"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestSystemRunner_ComposeStart_MultipleServices(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ComposeStart([]string{"service1", "service2"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedCmd := "docker compose up -d service1 service2"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestSystemRunner_ComposeStop_NoServices(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ComposeStop([]string{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "docker compose stop"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestSystemRunner_ComposeStop_OneService(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ComposeStop([]string{"service"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "docker compose stop service"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestSystemRunner_ComposeStop_MultipleServices(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ComposeStop([]string{"service1", "service2"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedCmd := "docker compose stop service1 service2"
	if capturedCmd != expectedCmd {
		t.Errorf("expected command to be %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestSystemRunner_ContainerExec_ExecutesCorrectCommand(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			capturedCmd = cmd
			return &mockRunnableCommand{}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.ContainerExec("cont", "echo hello")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "docker container exec cont echo hello"
	if capturedCmd != expectedCmd {
		t.Errorf("wrong command issued %q, expected %q", capturedCmd, expectedCmd)
	}
}

func TestSystemRunner_WaitUntilContainerExecIsSuccessful_SucceedsImmediately(t *testing.T) {
	var callCount int
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			callCount++
			return &mockRunnableCommand{
				runFunc: func() error {
					return nil // Success on the first try
				},
			}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.WaitUntilContainerExecIsSuccessful("test-container", "test-cmd")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 exec call, got: %d", callCount)
	}
}

func TestSystemRunner_WaitUntilContainerExecIsSuccessful_SucceedsAfterRetries(t *testing.T) {
	var callCount int
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			callCount++
			return &mockRunnableCommand{
				runFunc: func() error {
					if callCount < 5 {
						return fmt.Errorf("not ready yet")
					}
					return nil // Success on the 5th try
				},
			}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.WaitUntilContainerExecIsSuccessful("test-container", "test-cmd")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 5 {
		t.Errorf("expected 5 exec calls, got: %d", callCount)
	}
}

func TestSystemRunner_WaitUntilContainerExecIsSuccessful_ExhaustsRetries(t *testing.T) {
	var callCount int
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			callCount++
			return &mockRunnableCommand{
				runFunc: func() error {
					return fmt.Errorf("always fails")
				},
			}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.WaitUntilContainerExecIsSuccessful("test-container", "test-cmd")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTooManyRetries) {
		t.Errorf("expected ErrTooManyRetries, got: %v", err)
	}
	if callCount != 30 {
		t.Errorf("expected %d exec calls, got: %d", 30, callCount)
	}
}

func TestSystemRunner_WaitUntilContainerExecIsSuccessful_ExecutesCorrectCommand(t *testing.T) {
	var capturedCmd string
	commands := &mockCommands{
		execShellCommand: func(cmd string) system.RunnableCommand {
			return &mockRunnableCommand{
				runFunc: func() error {
					capturedCmd = cmd
					return nil // Success on the first try
				},
			}
		},
	}
	runner := &SystemRunner{
		commands:                     commands,
		files:                        &mockFiles{},
		time:                         &mockTime{},
		buildDockerComposeCommandStr: mockBuildDockerComposeCommandStr,
	}

	err := runner.WaitUntilContainerExecIsSuccessful("test-container", "test-cmd")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedCmd := "docker container exec test-container test-cmd"
	if capturedCmd != expectedCmd {
		t.Errorf("wrong command issued %q, expected %q", capturedCmd, expectedCmd)
	}
}
