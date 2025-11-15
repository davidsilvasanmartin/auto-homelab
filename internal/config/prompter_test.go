package config

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type mockWriter struct {
	writeFn func(p []byte) (n int, err error)
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	if m.writeFn != nil {
		return m.writeFn(p)
	}
	return len(p), nil
}

type mockReader struct {
	readFn func(p []byte) (n int, err error)
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.readFn != nil {
		return m.readFn(p)
	}
	return 0, io.EOF
}

func TestConsolePrompter_Prompt_Success(t *testing.T) {
	input := "user input value\n"
	reader := bufio.NewReader(strings.NewReader(input))
	var output bytes.Buffer
	prompter := &ConsolePrompter{
		reader: reader,
		writer: &output,
	}

	result, err := prompter.Prompt("Enter value: ")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	expectedResult := "user input value"
	if result != expectedResult {
		t.Errorf("expected result %q, got %q", expectedResult, result)
	}
	expectedOutput := "Enter value: "
	if output.String() != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, output.String())
	}
}

func TestConsolePrompter_Prompt_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "leading spaces",
			input:    "   value\n",
			expected: "value",
		},
		{
			name:     "trailing spaces",
			input:    "value   \n",
			expected: "value",
		},
		{
			name:     "leading and trailing spaces",
			input:    "   value   \n",
			expected: "value",
		},
		{
			name:     "tabs and spaces",
			input:    "\t  value  \t\n",
			expected: "value",
		},
		{
			name:     "empty with whitespace",
			input:    "   \n",
			expected: "",
		},
		{
			name:     "just newline",
			input:    "\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			var output bytes.Buffer
			prompter := &ConsolePrompter{
				reader: reader,
				writer: &output,
			}

			result, err := prompter.Prompt("Prompt: ")

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected result %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConsolePrompter_Prompt_ReadError(t *testing.T) {
	expectedErr := errors.New("read failed")
	mockReader := &mockReader{
		readFn: func(p []byte) (n int, err error) {
			return 0, expectedErr
		},
	}
	reader := bufio.NewReader(mockReader)
	var output bytes.Buffer
	prompter := &ConsolePrompter{
		reader: reader,
		writer: &output,
	}

	_, err := prompter.Prompt("Enter value: ")

	if err == nil {
		t.Fatal("expected error when read fails, got nil")
	}
	if !errors.Is(err, ErrPrompterRead) {
		t.Errorf("expected ErrPrompterRead, got: %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestConsolePrompter_Info_Success(t *testing.T) {
	var output bytes.Buffer
	prompter := &ConsolePrompter{
		reader: bufio.NewReader(strings.NewReader("")),
		writer: &output,
	}

	prompter.Info("Information message")

	expectedOutput := "Information message\n"
	if output.String() != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, output.String())
	}
}

func TestConsolePrompter_Info_EmptyMessage(t *testing.T) {
	var output bytes.Buffer
	prompter := &ConsolePrompter{
		reader: bufio.NewReader(strings.NewReader("")),
		writer: &output,
	}

	prompter.Info("")

	expectedOutput := "\n"
	if output.String() != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, output.String())
	}
}
