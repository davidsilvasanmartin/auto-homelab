package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// User interaction abstraction

// Prompter defines the interface for user interaction (prompting and displaying info)
type Prompter interface {
	Prompt(message string) (string, error)
	Info(message string) error
}

var (
	ErrPrompterWrite = errors.New("unable to write")
	ErrPrompterRead  = errors.New("unable to read")
)

// ConsolePrompter implements Prompter using stdin/stdout
type ConsolePrompter struct {
	reader *bufio.Reader
	writer io.Writer
}

// NewConsolePrompter creates a new console-based prompter
func NewConsolePrompter() *ConsolePrompter {
	return &ConsolePrompter{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Prompt displays a message and reads user input
func (p *ConsolePrompter) Prompt(message string) (string, error) {
	if _, err := fmt.Fprint(p.writer, message); err != nil {
		return "", fmt.Errorf("%w: %w", ErrPrompterWrite, err)
	}
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrPrompterRead, err)
	}
	return strings.TrimSpace(input), nil
}

// Info displays an informational message
func (p *ConsolePrompter) Info(message string) error {
	if _, err := fmt.Fprintln(p.writer, message); err != nil {
		return fmt.Errorf("%w: %w", ErrPrompterWrite, err)
	}
	return nil
}
