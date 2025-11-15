package config

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// Strategies for variable acquisition

// AcquireStrategy defines the interface for acquiring environment variable values
type AcquireStrategy interface {
	Acquire(varName string, defaultSpec *string) (string, error)
}

var (
	ErrConstantVarHasNoDefault = errors.New("constant variable has no default value")
	ErrCantParseDefaultSpec    = errors.New("unable to parse default spec")
	ErrCantGenerateSecret      = errors.New("unable to generate secret")
)

// ConstantStrategy returns a constant value
type ConstantStrategy struct {
	prompter Prompter
}

func NewConstantStrategy() *ConstantStrategy {
	return &ConstantStrategy{prompter: NewConsolePrompter()}
}

func (s *ConstantStrategy) Acquire(varName string, defaultSpec *string) (string, error) {
	// We don't care about previous values of varName here (the value read from .env),
	// we will override it
	if defaultSpec == nil {
		return "", fmt.Errorf("%w: %q", ErrConstantVarHasNoDefault, varName)
	}
	s.prompter.Info(fmt.Sprintf("Defaulting to: %s", *defaultSpec))
	return *defaultSpec, nil
}

// GeneratedStrategy generates a random secret value
type GeneratedStrategy struct {
	prompter Prompter
	env      system.Env
}

func NewGeneratedStrategy() *GeneratedStrategy {
	return &GeneratedStrategy{prompter: NewConsolePrompter(), env: system.NewDefaultEnv()}
}

var charsetPools = map[string]string{
	"ALL":   "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789%&*+-.:<>^_|~",
	"ALPHA": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
}

func (s *GeneratedStrategy) Acquire(varName string, defaultSpec *string) (string, error) {
	// TODO check defaultSpec NOT NIL
	if val, exists := s.env.GetEnv(varName); exists == true {
		s.prompter.Info("Not overriding already existing environment variable " + varName)
		return val, nil
	}

	charsetName, length, err := parseGeneratedSpec(*defaultSpec)
	if err != nil {
		return "", fmt.Errorf("%w %q: %w", ErrCantParseDefaultSpec, varName, err)
	}

	pool := charsetPools[charsetName]
	generated, err := generateSecret(pool, length)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCantGenerateSecret, err)
	}

	s.prompter.Info(fmt.Sprintf("Generated a secret value of length %d for %s.", length, varName))
	return generated, nil
}

func parseGeneratedSpec(spec string) (string, int, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", 0, errors.New("invalid format, expected SET:LENGTH")
	}

	charsetName := strings.TrimSpace(strings.ToUpper(parts[0]))
	if _, ok := charsetPools[charsetName]; !ok {
		return "", 0, fmt.Errorf("invalid charset %q, must be ALL or ALPHA", charsetName)
	}

	length, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", 0, fmt.Errorf("invalid length: %w", err)
	}
	if length <= 0 || length > 1024 {
		return "", 0, fmt.Errorf("length must be between 1 and 1024, got %d", length)
	}

	return charsetName, length, nil
}

func generateSecret(charset string, length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// IPStrategy prompts the user for a valid IP address
type IPStrategy struct {
	prompter Prompter
	env      system.Env
}

func NewIPStrategy() *IPStrategy {
	return &IPStrategy{prompter: NewConsolePrompter(), env: system.NewDefaultEnv()}
}

func (s *IPStrategy) Acquire(varName string, _ *string) (string, error) {
	if val, exists := s.env.GetEnv(varName); exists == true {
		s.prompter.Info("Not overriding already existing environment variable " + varName)
		return val, nil
	}

	for {
		input, err := s.prompter.Prompt(fmt.Sprintf("Enter value for %s (IP): ", varName))
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			s.prompter.Info("IP address cannot be empty. Please try again.")
			continue
		}

		if net.ParseIP(input) == nil {
			s.prompter.Info("Invalid IP address. Please enter a valid IPv4 or IPv6 address.")
			continue
		}

		return input, nil
	}
}

// StringStrategy prompts the user for a non-empty string
type StringStrategy struct {
	prompter Prompter
	env      system.Env
}

func NewStringStrategy() *StringStrategy {
	return &StringStrategy{prompter: NewConsolePrompter(), env: system.NewDefaultEnv()}
}

func (s *StringStrategy) Acquire(varName string, _ *string) (string, error) {
	// Check if already set in environment
	if val, exists := s.env.GetEnv(varName); exists == true {
		s.prompter.Info("Not overriding already existing environment variable " + varName)
		return val, nil
	}

	for {
		input, err := s.prompter.Prompt(fmt.Sprintf("Enter value for %s (STRING): ", varName))
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			s.prompter.Info("Value cannot be empty. Please enter a non-empty string.")
			continue
		}

		return input, nil
	}
}

// PathStrategy prompts the user for a directory path, creating it if needed
type PathStrategy struct {
	prompter Prompter
	env      system.Env
	files    system.FilesHandler
}

func NewPathStrategy() *PathStrategy {
	return &PathStrategy{prompter: NewConsolePrompter(), env: system.NewDefaultEnv(), files: system.NewDefaultFilesHandler()}
}

func (s *PathStrategy) Acquire(varName string, _ *string) (string, error) {
	if val, exists := s.env.GetEnv(varName); exists == true {
		s.prompter.Info("Not overriding already existing environment variable " + varName)
		return val, nil
	}

	for {
		input, err := s.prompter.Prompt(fmt.Sprintf("Enter value for %s (PATH): ", varName))
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			s.prompter.Info("Path cannot be empty. Please enter a directory path.")
			continue
		}

		if strings.HasPrefix(input, "~") {
			s.prompter.Info("Homedir ('~') expansion is not supported. Please enter a valid directory path.")
			continue
		}

		absPath, err := s.files.GetAbsPath(input)
		if err != nil {
			s.prompter.Info(fmt.Sprintf("Invalid path: %v. Please try again.", err))
			continue
		}

		// Check if the directory exists. If it does, we continue, and if it doesn't, we try to create it. If directory
		// creation fails, it can be due to an error such as insufficient permissions, so we let the user try again
		// with another directory
		if err := s.files.EnsureDirExists(absPath); err != nil {
			if err := s.files.CreateDirIfNotExists(absPath); err != nil {
				s.prompter.Info(fmt.Sprintf("Invalid path: %v. Please try again.", err))
				continue
			}
			s.prompter.Info(fmt.Sprintf("Created directory: %s", absPath))
		} else {
			s.prompter.Info(fmt.Sprintf("Directory exists: %s", absPath))
		}

		return absPath, nil
	}
}
