package docker

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type mockSystem struct {
	requireCommand func(command string) error
	execCommand    func(name string, arg ...string) *exec.Cmd
}

func (m *mockSystem) RequireCommand(command string) error {
	if m.requireCommand != nil {
		return m.requireCommand(command)
	}
	return nil
}

func (m *mockSystem) ExecCommand(name string, arg ...string) *exec.Cmd {
	if m.execCommand != nil {
		return m.execCommand(name, arg...)
	}
	return nil
}

type mockFiles struct {
	createDirIfNotExists func(path string) error
	requireFilesInWd     func(filenames ...string) error
	requireDir           func(path string) error
}

func (m *mockFiles) CreateDirIfNotExists(path string) error {
	if m.createDirIfNotExists != nil {
		return m.createDirIfNotExists(path)
	}
	return nil
}

func (m *mockFiles) RequireFilesInWd(filenames ...string) error {
	if m.requireFilesInWd != nil {
		return m.requireFilesInWd(filenames...)
	}
	return nil
}

func (m *mockFiles) RequireDir(path string) error {
	if m.requireDir(path) != nil {
		return m.requireDir(path)
	}
	return nil
}

// Test that ComposeStart with no services runs: docker compose up -d
func TestSystemRunner_ComposeStart_NoServices(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	system := &mockSystem{
		execCommand: func(name string, arg ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = arg
			// This creates a command but doesn't execute it
			// When Run() is called, it just runs "echo" which is harmless and fast
			return exec.Command("echo", "test")
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		system: system,
		files:  files,
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
	system := &mockSystem{
		execCommand: func(name string, arg ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = arg
			return exec.Command("echo", "test")
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		system: system,
		files:  files,
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
	mock := &mockSystem{
		execCommand: func(name string, arg ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = arg
			return exec.Command("echo", "test")
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		system: mock,
		files:  files,
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
	mock := &mockSystem{
		execCommand: func(name string, arg ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = arg
			// This creates a command but doesn't execute it
			// When Run() is called, it just runs "echo" which is harmless and fast
			return exec.Command("echo", "test")
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		system: mock,
		files:  files,
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
	mock := &mockSystem{
		execCommand: func(name string, arg ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = arg
			return exec.Command("echo", "test")
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		system: mock,
		files:  files,
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
	mock := &mockSystem{
		execCommand: func(name string, arg ...string) *exec.Cmd {
			capturedName = name
			capturedArgs = arg
			return exec.Command("echo", "test")
		},
	}
	files := &mockFiles{}
	runner := &SystemRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		system: mock,
		files:  files,
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
