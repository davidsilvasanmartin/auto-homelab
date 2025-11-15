package config

import (
	"errors"
	"strings"
	"testing"
)

func TestConstantStrategy_Acquire_Success(t *testing.T) {
	defaultValue := "test-value"
	var capturedMessage string
	strategy := &ConstantStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) {
				capturedMessage = message
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
	if !errors.Is(err, ErrNilDefaultSpec) {
		t.Errorf("expected ErrNilDefaultSpec, got: %v", err)
	}
	if !strings.Contains(err.Error(), "TEST_VAR") {
		t.Errorf("expected error message to contain variable name %q, got %q", "TEST_VAR", err.Error())
	}
}

func TestConstantStrategy_Acquire_EmptyDefault(t *testing.T) {
	defaultValue := ""
	var capturedMessage string
	strategy := &ConstantStrategy{
		prompter: &mockPrompter{
			infoFunc: func(message string) {
				capturedMessage = message
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

func TestGeneratedStrategy_Acquire_NoDefault(t *testing.T) {
	strategy := &GeneratedStrategy{prompter: &mockPrompter{}}

	_, err := strategy.Acquire("TEST_VAR", nil)

	if err == nil {
		t.Fatal("expected error when defaultSpec is nil, got nil")
	}
	if !errors.Is(err, ErrNilDefaultSpec) {
		t.Errorf("expected ErrNilDefaultSpec, got: %v", err)
	}
	if !strings.Contains(err.Error(), "TEST_VAR") {
		t.Errorf("expected error message to contain variable name %q, got %q", "TEST_VAR", err.Error())
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
			infoFunc: func(message string) {
				capturedMessage = message
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
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
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
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
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

func TestStringStrategy_Acquire_AlreadySetInEnv(t *testing.T) {
	existingValue := "val"
	strategy := &StringStrategy{
		prompter: &mockPrompter{},
		env: &mockEnv{
			getEnvFunc: func(varName string) (string, bool) {
				return existingValue, true
			},
		},
	}

	result, err := strategy.Acquire("VAR_NAME", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != existingValue {
		t.Errorf("expected result %q, got %q", existingValue, result)
	}
}

func TestStringStrategy_Acquire_PrompterError(t *testing.T) {
	expectedError := errors.New("prompter read failed")
	strategy := &StringStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return "", expectedError
			},
		},
		env: &mockEnv{},
	}

	_, err := strategy.Acquire("VAR_NAME", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedError) {
		t.Errorf("expected error to be %v, got: %v", expectedError, err)
	}
}

func TestStringStrategy_Acquire_EmptyInput_RetriesUntilValid(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	validValue := "val"
	strategy := &StringStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				if callCount == 1 {
					return "", nil
				}
				if callCount == 2 {
					return "  ", nil
				}
				return validValue, nil
			},
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
			},
		},
		env: &mockEnv{},
	}

	result, err := strategy.Acquire("VAR_NAME", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != validValue {
		t.Errorf("expected result %q, got %q", validValue, result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 prompt calls, got %d", callCount)
	}
	expectedMessageWhenValueIsEmpty := "Value cannot be empty"
	emptyMessages := 0
	for _, msg := range capturedInfoMessages {
		if strings.Contains(msg, expectedMessageWhenValueIsEmpty) {
			emptyMessages++
		}
	}
	if emptyMessages != 2 {
		t.Errorf("expected 2 empty value messages, got %d", emptyMessages)
	}
}

func TestStringStrategy_Acquire_Success(t *testing.T) {
	validValue := "val"
	var ocapturedPrompt string
	strategy := &StringStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				ocapturedPrompt = message
				return validValue, nil
			},
		},
		env: &mockEnv{},
	}

	result, err := strategy.Acquire("VAR_NAME", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != validValue {
		t.Errorf("expected result to be %v, got %v", validValue, result)
	}
	expectedPrompt := "Enter value for VAR_NAME (STRING): "
	if ocapturedPrompt != expectedPrompt {
		t.Errorf("expected prompt to be %v, got %v", expectedPrompt, ocapturedPrompt)
	}
}

func TestStringStrategy_Acquire_InputIsTrimmed(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "spaces", in: "    a     ", out: "a"},
		{name: "tabs and newline", in: "\t\n a \t\n", out: "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := &StringStrategy{
				prompter: &mockPrompter{
					promptFunc: func(message string) (string, error) {
						return tt.in, nil
					},
				},
				env: &mockEnv{},
			}

			result, err := strategy.Acquire("VAR_NAME", nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != tt.out {
				t.Errorf("expected result to be %v, got %v", tt.out, result)
			}
		})
	}

}

func TestStringStrategy_Acquire_IgnoresDefaultSpec(t *testing.T) {
	defaultSpec := "default"
	promptedValue := "val"
	strategy := &StringStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return promptedValue, nil
			},
		},
		env: &mockEnv{},
	}

	result, err := strategy.Acquire("VAR_NAME", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != promptedValue {
		t.Errorf("expected result %q (not default), got %q", promptedValue, result)
	}
}

func TestPathStrategy_Acquire_AlreadySetInEnv(t *testing.T) {
	existingPath := "/home/user/data"
	strategy := &PathStrategy{
		prompter: &mockPrompter{},
		env: &mockEnv{
			getEnvFunc: func(varName string) (string, bool) {
				return existingPath, true
			},
		},
		files: &mockFiles{},
	}

	result, err := strategy.Acquire("DATA_PATH", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != existingPath {
		t.Errorf("expected result %q, got %q", existingPath, result)
	}
}

func TestPathStrategy_Acquire_PrompterError(t *testing.T) {
	expectedErr := errors.New("prompter read failed")
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return "", expectedErr
			},
		},
		env:   &mockEnv{},
		files: &mockFiles{},
	}

	_, err := strategy.Acquire("PATH_VAR", nil)

	if err == nil {
		t.Fatal("expected error when prompter fails, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestPathStrategy_Acquire_EmptyInput_RetriesUntilValid(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	validPath := "/home/user/valid"
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				if callCount == 1 {
					return "", nil
				}
				if callCount == 2 {
					return "  ", nil
				}
				return validPath, nil
			},
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				// This function will be called just once, with the non-empty path
				return validPath, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != validPath {
		t.Errorf("expected result %q, got %q", validPath, result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 prompt calls, got %d", callCount)
	}
	expectedMessageWhenPathIsEmpty := "Path cannot be empty"
	emptyMessages := 0
	for _, msg := range capturedInfoMessages {
		if strings.Contains(msg, expectedMessageWhenPathIsEmpty) {
			emptyMessages++
		}
	}
	if emptyMessages != 2 {
		t.Errorf("expected 2 empty path messages, got %d", emptyMessages)
	}
}

func TestPathStrategy_Acquire_HomedirExpansionNotSupported(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	validPath := "/home/user/documents"
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				if callCount == 1 {
					return "~/documents", nil
				}
				return validPath, nil
			},
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				return validPath, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != validPath {
		t.Errorf("expected result %q, got %q", validPath, result)
	}
	if callCount != 2 {
		t.Errorf("expected 2 prompt calls, got %d", callCount)
	}
	expectedMessageWhenHomedirCharIntroduced := "Homedir ('~') expansion is not supported"
	found := false
	for _, msg := range capturedInfoMessages {
		if strings.Contains(msg, expectedMessageWhenHomedirCharIntroduced) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find homedir expansion message in info messages")
	}
}

func TestPathStrategy_Acquire_RelativePathConvertsToAbsolute(t *testing.T) {
	relativePath := "relative/path"
	absolutePath := "/home/user/relative/path"
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return relativePath, nil
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				return absolutePath, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != absolutePath {
		t.Errorf("expected absolute path, got %q", result)
	}
}

func TestPathStrategy_Acquire_CreateDirError_RetriesUntilValid(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	createDirError := errors.New("permission denied")
	restrictedPath := "/restricted/path"
	validPath := "/home/user/valid"
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				if callCount <= 2 {
					return restrictedPath, nil
				}
				return validPath, nil
			},
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return errors.New("directory does not exist")
			},
			createDirIfNotExists: func(path string) error {
				if path == restrictedPath {
					return createDirError
				}
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				return path, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != validPath {
		t.Errorf("expected result %q, got %q", validPath, result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 prompt calls, got %d", callCount)
	}
	errorMessagesShown := 0
	for _, msg := range capturedInfoMessages {
		if strings.Contains(msg, "Invalid path:") {
			errorMessagesShown++
		}
	}
	if errorMessagesShown != 2 {
		t.Errorf("expected 2 error messages, got %d", errorMessagesShown)
	}
}

func TestPathStrategy_Acquire_PathReused_RetriesUntilValid(t *testing.T) {
	callCount := 0
	var capturedInfoMessages []string
	alreadyUsedPath := "/used/path"
	validPath := "/home/user/valid"
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				callCount++
				if callCount <= 2 {
					return alreadyUsedPath, nil
				}
				return validPath, nil
			},
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			getAbsPath: func(path string) (string, error) {
				return path, nil
			},
		},
		alreadyUsedPaths: []string{alreadyUsedPath},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != validPath {
		t.Errorf("expected result %q, got %q", validPath, result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 prompt calls, got %d", callCount)
	}
	pathReusedMessagesShown := 0
	for _, msg := range capturedInfoMessages {
		if strings.Contains(msg, "Path cannot be reused:") {
			pathReusedMessagesShown++
		}
	}
	if pathReusedMessagesShown != 2 {
		t.Errorf("expected 2 path reused messages, got %d", pathReusedMessagesShown)
	}
}

func TestPathStrategy_Acquire_Success_ExistingDirectory(t *testing.T) {
	inputPath := "existing/dir"
	absPath := "/home/user/existing/dir"
	var capturedPrompt string
	var capturedInfoMessage string
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				capturedPrompt = message
				return inputPath, nil
			},
			infoFunc: func(message string) {
				// Continuously override this variable: we only want to keep the last one
				capturedInfoMessage = message
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				return absPath, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != absPath {
		t.Errorf("expected result %q, got %q", absPath, result)
	}
	expectedPrompt := "Enter value for PATH_VAR (PATH): "
	if capturedPrompt != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, capturedPrompt)
	}
	expectedInfo := "Directory exists: " + absPath
	if capturedInfoMessage != expectedInfo {
		t.Errorf("expected info %q, got %q", expectedInfo, capturedInfoMessage)
	}
}

func TestPathStrategy_Acquire_Success_CreatesNonExistingDirectory(t *testing.T) {
	inputPath := "new/dir"
	absPath := "/home/user/new/dir"
	var createdPath string
	var capturedPrompt string
	var capturedInfoMessage string
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				capturedPrompt = message
				return inputPath, nil
			},
			infoFunc: func(message string) {
				// We will keep the last one
				capturedInfoMessage = message
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return errors.New("directory does not exist")
			},
			createDirIfNotExists: func(path string) error {
				createdPath = path
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				return absPath, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != absPath {
		t.Errorf("expected result %q, got %q", absPath, result)
	}
	if createdPath != absPath {
		t.Errorf("expected created path %q, got %q", absPath, createdPath)
	}
	expectedPrompt := "Enter value for PATH_VAR (PATH): "
	if capturedPrompt != expectedPrompt {
		t.Errorf("expected prompt %q, got %q", expectedPrompt, capturedPrompt)
	}
	expectedInfo := "Created directory: " + absPath
	if capturedInfoMessage != expectedInfo {
		t.Errorf("expected info %q, got %q", expectedInfo, capturedInfoMessage)
	}
}

func TestPathStrategy_Acquire_IgnoresDefaultSpec(t *testing.T) {
	defaultSpec := "/default/path"
	promptedValue := "/prompted/path"
	strategy := &PathStrategy{
		prompter: &mockPrompter{
			promptFunc: func(message string) (string, error) {
				return promptedValue, nil
			},
		},
		env: &mockEnv{},
		files: &mockFiles{
			ensureDirExists: func(path string) error {
				return nil
			},
			getAbsPath: func(path string) (string, error) {
				return path, nil
			},
		},
	}

	result, err := strategy.Acquire("PATH_VAR", &defaultSpec)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != promptedValue {
		t.Errorf("expected result %q (not default), got %q", promptedValue, result)
	}
}
