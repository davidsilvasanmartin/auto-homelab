package format

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/go-wordwrap"
)

type TextFormatter interface {
	WrapLines(text string, width uint) []string
	FormatDotenvKeyValue(key string, value string) (string, error)
	QuoteForPOSIXShell(text string) string
}

var (
	ErrEmptyKey = errors.New("key must not be empty")
)

type DefaultTextFormatter struct{}

func NewDefaultTextFormatter() *DefaultTextFormatter {
	return &DefaultTextFormatter{}
}

// WrapLines wraps text to the specified width, preserving paragraph breaks.
func (d *DefaultTextFormatter) WrapLines(text string, width uint) []string {
	if strings.TrimSpace(text) == "" {
		return []string{""}
	}

	var lines []string
	paragraphs := strings.Split(text, "\n")

	for _, paragraph := range paragraphs {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		wrapped := wordwrap.WrapString(paragraph, width)
		wrappedLines := strings.Split(wrapped, "\n")

		lines = append(lines, wrappedLines...)
	}

	return lines
}

// FormatDotenvKeyValue formats a key-value pair for a .env file as: KEY="VALUE"
// Double quotes inside VALUE are escaped as \"
func (d *DefaultTextFormatter) FormatDotenvKeyValue(key string, value string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("%w", ErrEmptyKey)
	}

	value = strings.ReplaceAll(value, `"`, `\"`)

	return fmt.Sprintf(`%s="%s"`, key, value), nil
}

// QuoteForPOSIXShell Wraps a string in single quotes for POSIX shells, escaping any embedded single
// quotes safely. This transforms p'q into 'p'"'"'q' which the shell interprets as a single literal string.
func (d *DefaultTextFormatter) QuoteForPOSIXShell(text string) string {
	return "'" + strings.ReplaceAll(text, "'", "'\"'\"'") + "'"
}
