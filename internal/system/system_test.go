package system

import (
	"errors"
	"testing"
)

// Test that a command that exists returns no error
func TestDefaultSystem_RequireCommand_CommandExists(t *testing.T) {
	mock := &mockStdlib{
		execLookPath: func(file string) (string, error) {
			return "/usr/bin/docker", nil
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireCommand("docker")
	if err != nil {
		t.Errorf("expected no error for existing command, got %v", err)
	}
}

// Test that a command that doesn't exist returns an error
func TestDefaultSystem_RequireCommand_CommandNotFound(t *testing.T) {
	lookPathErr := errors.New("executable file not found")
	mock := &mockStdlib{
		execLookPath: func(file string) (string, error) {
			return "", lookPathErr
		},
	}
	sys := &DefaultSystem{stdlib: mock}

	err := sys.RequireCommand("nonexistent-command")

	if err == nil {
		t.Fatal("expected error for missing command, got nil")
	}
	if !errors.Is(err, lookPathErr) {
		t.Errorf("expected error to wrap %v, got: %v", lookPathErr, err)
	}
}
