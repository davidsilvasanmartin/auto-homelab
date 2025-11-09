package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidsilvasanmartin/auto-homelab/internal/format"
	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// Orchestration logic for the config feature

type Configurer interface {
	// LoadConfig loads the configuration from a file
	LoadConfig(configFilePath string) (*ConfigRoot, error)
	// ProcessConfig processes the configuration and retrieves variable values
	ProcessConfig(configRoot *ConfigRoot) (*EnvVarRoot, error)
	// WriteConfig writes the processed configuration into a timestamped generated .env file
	WriteConfig(envVarRoot *EnvVarRoot) (string, error)
}

var (
	ErrConfigFileRead  = errors.New("failed to read config file")
	ErrConfigFileParse = errors.New("failed to parse config file")
	ErrVarType         = errors.New("error processing variable type")
	ErrVarAcquireVal   = errors.New("error acquiring value for variable")
	ErrConfigFileWrite = errors.New("failed to write config file")
)

type DefaultConfigurer struct {
	prompter         Prompter
	strategyRegistry *StrategyRegistry
	textFormatter    format.TextFormatter
	files            system.FilesHandler
}

func NewDefaultConfigurer() *DefaultConfigurer {
	return &DefaultConfigurer{
		prompter:         NewConsolePrompter(),
		strategyRegistry: NewStrategyRegistry(),
		textFormatter:    format.NewDefaultTextFormatter(),
		files:            system.NewDefaultFilesHandler(),
	}
}

func (c *DefaultConfigurer) LoadConfig(configFilePath string) (*ConfigRoot, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrConfigFileRead, configFilePath)
	}

	var configRoot ConfigRoot
	if err := json.Unmarshal(data, &configRoot); err != nil {
		return nil, fmt.Errorf("%w: %q", ErrConfigFileParse, configFilePath)
	}

	return &configRoot, nil
}

func (c *DefaultConfigurer) ProcessConfig(configRoot *ConfigRoot) (*EnvVarRoot, error) {
	root := &EnvVarRoot{
		Prefix:   configRoot.Prefix,
		Sections: make([]EnvVarSection, 0, len(configRoot.Sections)),
	}

	for _, configSection := range configRoot.Sections {
		sectionName := fmt.Sprintf("%s_%s", root.Prefix, configSection.Name)
		section := EnvVarSection{
			Name:        sectionName,
			Description: configSection.Description,
			Vars:        make([]EnvVar, 0, len(configSection.Variables)),
		}

		if err := c.prompter.Info(fmt.Sprintf("\n\n>>>>>>>>>> Section: %s", section.Name)); err != nil {
			return nil, err
		}
		if err := c.prompter.Info(section.Description); err != nil {
			return nil, err
		}

		for _, configVar := range configSection.Variables {
			varName := fmt.Sprintf("%s_%s", section.Name, configVar.Name)

			if err := c.prompter.Info(fmt.Sprintf("\n> %s: %s", varName, configVar.Description)); err != nil {
				return nil, err
			}

			// Get the appropriate strategy
			strategy, err := c.strategyRegistry.Get(configVar.Type)
			if err != nil {
				return nil, fmt.Errorf("%w %q (var name %q): %w", ErrVarType, configVar.Type, varName, err)
			}

			// Acquire the value using the strategy
			value, err := strategy.Acquire(varName, configVar.Value)
			if err != nil {
				return nil, fmt.Errorf("%w %q: %w", ErrVarAcquireVal, varName, err)
			}

			envVar := EnvVar{
				Name:        varName,
				Description: configVar.Description,
				Value:       value,
			}
			section.Vars = append(section.Vars, envVar)
		}

		root.Sections = append(root.Sections, section)
	}

	return root, nil
}

// TODO REVIEW AND TEST BELOW
func (c *DefaultConfigurer) WriteConfig(envVarRoot *EnvVarRoot) (string, error) {
	builder := newDotenvBuilder(c.textFormatter)
	for _, section := range envVarRoot.Sections {
		builder.addSection(section)
	}
	content := builder.build()

	// Generate timestamped filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf(".env.generated.%d", timestamp)

	// Get current working directory to save the file in project root
	wd, err := c.files.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	outputPath := filepath.Join(wd, filename)

	// Write the file
	if err := c.files.WriteFile(outputPath, []byte(content)); err != nil {
		return "", fmt.Errorf("%w %q: %w", ErrConfigFileWrite, outputPath, err)
	}

	return outputPath, nil
}

// dotenvBuilder builds a .env file from the parsed configuration
type dotenvBuilder struct {
	lines         []string
	totalVars     int
	textFormatter format.TextFormatter
}

func newDotenvBuilder(formatter format.TextFormatter) *dotenvBuilder {
	return &dotenvBuilder{
		lines:         []string{},
		totalVars:     0,
		textFormatter: formatter,
	}
}

// addVar adds a single variable to the .env file
func (b *dotenvBuilder) addVar(envVar EnvVar) error {
	// Wrap and add description lines as comments
	wrappedLines := b.textFormatter.WrapLines(envVar.Description, 120)
	for _, line := range wrappedLines {
		b.lines = append(b.lines, fmt.Sprintf("# %s", line))
	}

	// Add the key-value pair
	formatted, err := b.textFormatter.FormatDotenvKeyValue(envVar.Name, envVar.Value)
	if err != nil {
		return fmt.Errorf("failed to format variable %q: %w", envVar.Name, err)
	}
	b.lines = append(b.lines, formatted)

	b.totalVars++
	return nil
}

// addSection adds a section to the .env file
func (b *dotenvBuilder) addSection(section EnvVarSection) error {
	// Add section header
	b.lines = append(b.lines, strings.Repeat("#", 120))
	b.lines = append(b.lines, fmt.Sprintf("# %s", section.Name))
	wrappedLines := b.textFormatter.WrapLines(section.Description, 120)
	for _, line := range wrappedLines {
		b.lines = append(b.lines, fmt.Sprintf("# %s", line))
	}
	b.lines = append(b.lines, strings.Repeat("#", 120))

	// Add all variables in the section
	for _, envVar := range section.Vars {
		if err := b.addVar(envVar); err != nil {
			return err
		}
	}

	// Add blank line after section
	b.lines = append(b.lines, "")

	return nil
}

// build returns the final .env file content
func (b *dotenvBuilder) build() string {
	// Join all lines with newline
	content := strings.Join(b.lines, "\n")
	// Remove trailing whitespace and ensure file ends with single newline
	content = strings.TrimRight(content, "\n")
	return content + "\n"
}
