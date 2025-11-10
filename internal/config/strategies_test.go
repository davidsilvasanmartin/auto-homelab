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

func TestGeneratedStrategy_Acquire_ShowsCorrectSuccessMessage(t *testing.T) {
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

func TestIPStrategy_Acquire_AlreadySetInEnv(t *testing.T) {
	existingIP := "192.168.1.100"
	strategy := &IPStrategy{
		prompter: &mockPrompter{},
		env: &mockEnv{
			getEnvFunc: func(varName string) (string, bool) {
				return existingIP, true
			},
		},
	}

	result, err := strategy.Acquire("SERVER_IP", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != existingIP {
		t.Errorf("expected result %q, got %q", existingIP, result)
	}
}

func TestIPStrategy_Acquire_PromptsCorrectMessage(t *testing.T) {
	validIP := "192.168.1.100"
	var capturedPrompt string
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				capturedPrompt = message
				return validIP, nil
			},
		},
		env: &mockEnv{},
	}

	_, err := strategy.Acquire("TEST_IP", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedPrompt := "Enter value for TEST_IP (IP): "
	if capturedPrompt != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, capturedPrompt)
	}
}

func TestIPStrategy_Acquire_ValidIpv4(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"public", "1.2.3.4"},
		{"loopback", "127.0.0.1"},
		{"all_zeros", "0.0.0.0"},
		{"broadcast", "255.255.255.255"},
		{"private_10", "10.0.0.1"},
		{"private_172", "172.16.0.1"},
		{"private_192", "192.168.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &IPStrategy{
				prompter: &mockPrompter{
					promptFunc: func(message string) (string, error) {
						return tt.ip, nil
					},
				},
				env: &mockEnv{},
			}

			result, err := strategy.Acquire("IP_VAR", nil)

			if err != nil {
				t.Fatalf("expected no error for %s, got %v", tt.ip, err)
			}
			if result != tt.ip {
				t.Errorf("expected result %q, got %q", tt.ip, result)
			}
		})
	}
}

func TestIPStrategy_Acquire_ValidIPv6(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"full_ipv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"compressed_ipv6", "2001:db8:85a3::8a2e:370:7334"},
		{"loopback_ipv6", "::1"},
		{"ipv6_all_zeros", "::"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &IPStrategy{
				prompter: &mockPrompter{
					promptFunc: func(message string) (string, error) {
						return tt.ip, nil
					},
				},
				env: &mockEnv{},
			}

			result, err := strategy.Acquire("IP_VAR", nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != tt.ip {
				t.Errorf("expected result %q, got %q", tt.ip, result)
			}
		})
	}
}

func TestIPStrategy_Acquire_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"leading_space", " 192.168.1.1", "192.168.1.1"},
		{"trailing_space", "192.168.1.1 ", "192.168.1.1"},
		{"both_spaces", "  192.168.1.1  ", "192.168.1.1"},
		{"tabs", "\t192.168.1.1\t", "192.168.1.1"},
		{"newlines", "\n192.168.1.1\n", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &IPStrategy{
				prompter: &mockPrompter{
					promptFunc: func(message string) (string, error) {
						return tt.input, nil
					},
				},
				env: &mockEnv{},
			}

			result, err := strategy.Acquire("IP_VAR", nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected result %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIPStrategy_Acquire_PrompterError(t *testing.T) {
	expectedErr := errors.New("prompter read failed")
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return "", expectedErr
			},
		},
		env: &mockEnv{},
	}

	_, err := strategy.Acquire("IP_VAR", nil)

	if err == nil {
		t.Fatal("expected error when prompter fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestIPStrategy_Acquire_EmptyInput_RetriesUntilValid(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				if callCount == 1 {
					return "", nil
				}
				if callCount == 2 {
					return "  ", nil
				}
				return "192.168.1.1", nil
			},
			infoFunc: func(message string) error {
				capturedInfoMessages = append(capturedInfoMessages, message)
				return nil
			},
		},
		env: &mockEnv{},
	}

	result, err := strategy.Acquire("IP_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "192.168.1.1" {
		t.Errorf("expected result %q, got %q", "192.168.1.1", result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 prompt calls, got %d", callCount)
	}
	if len(capturedInfoMessages) != 2 {
		t.Fatalf("expected 2 info messages, got %d", len(capturedInfoMessages))
	}
	expectedMessage := "IP address cannot be empty. Please try again."
	for i, msg := range capturedInfoMessages {
		if msg != expectedMessage {
			t.Errorf("info message %d: expected %q, got %q", i, expectedMessage, msg)
		}
	}
}

func TestIPStrategy_Acquire_InfoErrorOnEmptyInput(t *testing.T) {
	expectedErr := errors.New("info write failed")
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return "", nil
			},
			infoFunc: func(message string) error {
				return expectedErr
			},
		},
		env: &mockEnv{},
	}

	_, err := strategy.Acquire("IP_VAR", nil)

	if err == nil {
		t.Fatal("expected error when info fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestIPStrategy_Acquire_InvalidIP_RetriesUntilValid(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				switch callCount {
				case 1:
					return "not-an-ip", nil
				case 2:
					return "192.168.1.256", nil
				case 3:
					return "192.168.1", nil
				case 4:
					return "192.168.1.1:8080", nil
				case 5:
					return "http://192.168.1.1", nil
				default:
					return "10.0.0.1", nil
				}
			},
			infoFunc: func(message string) error {
				capturedInfoMessages = append(capturedInfoMessages, message)
				return nil
			},
		},
		env: &mockEnv{},
	}

	result, err := strategy.Acquire("IP_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "10.0.0.1" {
		t.Errorf("expected result %q, got %q", "10.0.0.1", result)
	}
	if callCount != 6 {
		t.Errorf("expected 6 prompt calls, got %d", callCount)
	}
	if len(capturedInfoMessages) != 5 {
		t.Fatalf("expected 5 info messages, got %d", len(capturedInfoMessages))
	}
	expectedMessage := "Invalid IP address. Please enter a valid IPv4 or IPv6 address."
	for i, msg := range capturedInfoMessages {
		if msg != expectedMessage {
			t.Errorf("info message %d: expected %q, got %q", i, expectedMessage, msg)
		}
	}
}

func TestIPStrategy_Acquire_InfoErrorOnInvalidIP(t *testing.T) {
	expectedErr := errors.New("info write failed")
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return "invalid-ip", nil
			},
			infoFunc: func(message string) error {
				return expectedErr
			},
		},
		env: &mockEnv{},
	}

	_, err := strategy.Acquire("IP_VAR", nil)

	if err == nil {
		t.Fatal("expected error when info fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestIPStrategy_Acquire_IgnoresDefaultSpec(t *testing.T) {
	defaultSpec := "10.0.0.1"
	promptedValue := "192.168.1.1"
	strategy := &IPStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return promptedValue, nil
			},
		},
		env: &mockEnv{},
	}

	result, err := strategy.Acquire("IP_VAR", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != promptedValue {
		t.Errorf("expected result %q, got %q", promptedValue, result)
	}
}
