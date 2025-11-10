package config

import (
	"errors"
	"strings"
	"testing"
)

// mockPrompter is a mock implementation of Prompter for testing
type mockPrompter struct {
	promptFunc func(message string) (string, error)
	infoFunc   func(message string) error
}

func (m *mockPrompter) Prompt(message string) (string, error) {
	if m.promptFunc != nil {
		return m.promptFunc(message)
	}
	return "", nil
}
func (m *mockPrompter) Info(message string) error {
	if m.infoFunc != nil {
		return m.infoFunc(message)
	}
	return nil
}

type mockEnv struct {
	getEnvFunc func(varName string) (string, bool)
}

func (m *mockEnv) GetEnv(varName string) (string, bool) {
	if m.getEnvFunc != nil {
		return m.getEnvFunc(varName)
	}
	return "", false
}
func (m *mockEnv) GetRequiredEnv(varName string) (string, error) { return "", nil }

type mockFiles struct {
	createDirIfNotExists func(path string) error
	requireFilesInWd     func(filenames ...string) error
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

func TestConstantStrategy_Acquire_Success(t *testing.T) {
	defaultValue := "test-value"
	var capturedMessage string
	strategy := &ConstantStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) error {
				capturedMessage = message
				return nil
			},
		},
	}

	result, err := strategy.Acquire("TEST_VAR", &defaultValue)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != defaultValue {
		t.Errorf("expected result %q, got %q", defaultValue, result)
	}
	expectedMessage := "Defaulting to: test-value"
	if capturedMessage != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, capturedMessage)
	}
}

func TestConstantStrategy_Acquire_NoDefault(t *testing.T) {
	strategy := &ConstantStrategy{prompter: &mockPrompter{}}

	_, err := strategy.Acquire("TEST_VAR", nil)

	if err == nil {
		t.Fatal("expected error when defaultSpec is nil, got nil")
	}
	if !errors.Is(err, ErrConstantVarHasNoDefault) {
		t.Errorf("expected ErrConstantVarHasNoDefault, got: %v", err)
	}
	if !strings.Contains(err.Error(), "TEST_VAR") {
		t.Errorf("expected error message to contain variable name %q, got %q", "TEST_VAR", err.Error())
	}
}

func TestConstantStrategy_Acquire_PrompterError(t *testing.T) {
	defaultValue := "some-value"
	expectedErr := errors.New("prompter failed")
	strategy := &ConstantStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) error {
				return expectedErr
			},
		},
	}

	_, err := strategy.Acquire("MY_VAR", &defaultValue)

	if err == nil {
		t.Fatal("expected error when prompter fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestConstantStrategy_Acquire_EmptyDefault(t *testing.T) {
	defaultValue := ""
	var capturedMessage string
	strategy := &ConstantStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) error {
				capturedMessage = message
				return nil
			},
		},
	}

	result, err := strategy.Acquire("EMPTY_VAR", &defaultValue)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
	expectedMessage := "Defaulting to: "
	if capturedMessage != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, capturedMessage)
	}
}
