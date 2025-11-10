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

// GetEnv assumes by default that the variable does NOT exist
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
		t.Fatalf("expected no error, got %v", err)
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
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
	expectedMessage := "Defaulting to: "
	if capturedMessage != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, capturedMessage)
	}
}

func TestGeneratedStrategy_Acquire_AlreadySetInEnv(t *testing.T) {
	existingValue := "existing-secret-value"
	strategy := &GeneratedStrategy{
		prompter: &mockPrompter{},
		env: &mockEnv{
			getEnvFunc: func(varName string) (string, bool) {
				return existingValue, true
			},
		},
	}
	defaultSpec := "ALL:32"

	result, err := strategy.Acquire("TEST_SECRET", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != existingValue {
		t.Errorf("expected result %q, got %q", existingValue, result)
	}
}

func TestGeneratedStrategy_Acquire_PromptsCorrectMessage(t *testing.T) {
	var capturedMessage string
	strategy := &GeneratedStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) error {
				capturedMessage = message
				return nil
			},
		},
		env: &mockEnv{},
	}
	defaultSpec := "ALPHA:32"

	_, err := strategy.Acquire("TEST_VAR", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedMessage := "Generated a secret value of length 32 for TEST_VAR."
	if capturedMessage != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, capturedMessage)
	}
}

func TestGeneratedStrategy_Acquire_GeneratesAlphaSecret(t *testing.T) {
	strategy := &GeneratedStrategy{
		prompter: &mockPrompter{},
		env:      &mockEnv{},
	}
	defaultSpec := "ALPHA:32"

	result, err := strategy.Acquire("TEST_VAR", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 32 {
		t.Errorf("expected result length 32, got %d", len(result))
	}
	for _, ch := range result {
		if !strings.ContainsRune(charsetPools["ALPHA"], ch) {
			t.Errorf("expected only alphanumeric characters, found %q", ch)
		}
	}
}

func TestGeneratedStrategy_Acquire_GeneratesAllSecret(t *testing.T) {
	strategy := &GeneratedStrategy{
		prompter: &mockPrompter{},
		env:      &mockEnv{},
	}
	defaultSpec := "ALL:64"

	result, err := strategy.Acquire("TEST_VAR", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 64 {
		t.Errorf("expected result length 64, got %d", len(result))
	}
	for _, ch := range result {
		if !strings.ContainsRune(charsetPools["ALL"], ch) {
			t.Errorf("unexpected character %q in generated secret", ch)
		}
	}
}

func TestGeneratedStrategy_Acquire_DifferentLengths(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected int
	}{
		{"length_1", "ALPHA:1", 1},
		{"length_16", "ALL:16", 16},
		{"length_128", "ALPHA:128", 128},
		{"length_1024", "ALL:1024", 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &GeneratedStrategy{
				prompter: &mockPrompter{},
				env:      &mockEnv{},
			}

			result, err := strategy.Acquire("VAR", &tt.spec)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("expected length %d, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestGeneratedStrategy_Acquire_InvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{"no_colon", "ALPHA32"},
		{"empty", ""},
		{"only_charset", "ALPHA:"},
		{"only_length", ":32"},
		{"multiple_colons", "ALPHA:32:extra"},
		{"charset_invalid", "INVALID:32"},
		{"length_zero", "ALPHA:0"},
		{"length_negative", "ALL:-5"},
		{"length_too_large", "ALPHA:1025"},
		{"length_not_a_number", "ALL:abc"},
		{"length_float_number", "ALPHA:32.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &GeneratedStrategy{
				prompter: &mockPrompter{},
				env:      &mockEnv{},
			}

			_, err := strategy.Acquire("VAR", &tt.spec)

			if err == nil {
				t.Fatal("expected error for invalid spec, got nil")
			}
			if !errors.Is(err, ErrCantParseDefaultSpec) {
				t.Errorf("expected ErrCantParseDefaultSpec, got: %v", err)
			}
			if !strings.Contains(err.Error(), "VAR") {
				t.Errorf("expected error message to contain var name %q, got %q", "VAR", err.Error())
			}
		})
	}
}

func TestGeneratedStrategy_Acquire_CaseInsensitiveCharset(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{"lowercase_alpha", "alpha:16"},
		{"lowercase_all", "all:16"},
		{"mixed_case", "AlPhA:16"},
		{"uppercase", "ALL:16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &GeneratedStrategy{
				prompter: &mockPrompter{},
				env:      &mockEnv{},
			}

			result, err := strategy.Acquire("VAR", &tt.spec)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(result) != 16 {
				t.Errorf("expected length 16, got %d", len(result))
			}
		})
	}
}

func TestGeneratedStrategy_Acquire_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{"spaces_around", " ALPHA : 32 "},
		{"tabs", "\tALL\t:\t16\t"},
		{"mixed_whitespace", "  ALL  :  64  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &GeneratedStrategy{
				prompter: &mockPrompter{},
				env:      &mockEnv{},
			}

			result, err := strategy.Acquire("VAR", &tt.spec)

			if err != nil {
				t.Fatalf("expected no error with whitespace handling, got %v", err)
			}
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestGeneratedStrategy_Acquire_PrompterError(t *testing.T) {
	expectedErr := errors.New("prompter failed")
	strategy := &GeneratedStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) error {
				return expectedErr
			},
		},
		env: &mockEnv{},
	}
	defaultSpec := "ALPHA:32"

	_, err := strategy.Acquire("VAR", &defaultSpec)

	if err == nil {
		t.Fatal("expected error when prompter fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestGeneratedStrategy_Acquire_GeneratesUniqueValues(t *testing.T) {
	strategy := &GeneratedStrategy{
		prompter: &mockPrompter{},
		env:      &mockEnv{},
	}
	defaultSpec := "ALL:32"

	// Generate multiple secrets and check they're different
	results := make(map[string]bool)
	for i := 0; i < 10; i++ {
		result, err := strategy.Acquire("VAR", &defaultSpec)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if results[result] {
			t.Errorf("generated duplicate secret: %q", result)
		}
		results[result] = true
	}
}
