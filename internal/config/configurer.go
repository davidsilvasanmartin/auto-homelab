package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	WriteConfig(envVarRoot *EnvVarRoot) error
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
	strategyRegistry StrategyRegistry
	textFormatter    format.TextFormatter
	files            system.FilesHandler
}

func NewDefaultConfigurer() *DefaultConfigurer {
	return &DefaultConfigurer{
		prompter:         NewConsolePrompter(),
		strategyRegistry: NewDefaultStrategyRegistry(),
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
		Sections: make([]EnvVarSection, 0, len(configRoot.Sections)),
	}

	for _, configSection := range configRoot.Sections {
		section := EnvVarSection{
			Name:        fmt.Sprintf("%s_%s", configRoot.Prefix, configSection.Name),
			Description: configSection.Description,
			Vars:        make([]EnvVar, 0, len(configSection.Vars)),
		}

		c.prompter.Info(fmt.Sprintf("\n\n>>>>>>>>>> Section: %s", section.Name))
		c.prompter.Info(section.Description)

		for _, configVar := range configSection.Vars {
			varName := fmt.Sprintf("%s_%s", section.Name, configVar.Name)

			c.prompter.Info(fmt.Sprintf("\n> %s: %s", varName, configVar.Description))

			strategy, err := c.strategyRegistry.Get(configVar.Type)
			if err != nil {
				return nil, fmt.Errorf("%w %q (varName=%q): %w", ErrVarType, configVar.Type, varName, err)
			}

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

func (c *DefaultConfigurer) WriteConfig(envVarRoot *EnvVarRoot) error {
	builder := newDotenvBuilder(c.textFormatter)
	for _, section := range envVarRoot.Sections {
		err := builder.addSection(section)
		if err != nil {
			return err
		}
	}
	content := builder.build()

	timestamp := time.Now().Unix()
	// Start name with .env so the file is shown next to other .env files; end file with .env so that
	// we have syntax highlighting when opening it
	filename := fmt.Sprintf(".env.generated.%d.env", timestamp)

	wd, err := c.files.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	outputPath := filepath.Join(wd, filename)

	if err := c.files.WriteFile(outputPath, []byte(content)); err != nil {
		return fmt.Errorf("%w %q: %w", ErrConfigFileWrite, outputPath, err)
	}

	slog.Info("wrote config file", "outputPath", outputPath, "totalVars", builder.totalVars)

	return nil
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
	wrappedLines := b.textFormatter.WrapLines(envVar.Description, 118)
	for _, line := range wrappedLines {
		b.lines = append(b.lines, fmt.Sprintf("# %s", line))
	}

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
	b.lines = append(b.lines, strings.Repeat("#", 120))
	b.lines = append(b.lines, fmt.Sprintf("# %s", section.Name))
	wrappedLines := b.textFormatter.WrapLines(section.Description, 118)
	for _, line := range wrappedLines {
		b.lines = append(b.lines, fmt.Sprintf("# %s", line))
	}
	b.lines = append(b.lines, strings.Repeat("#", 120))

	for _, envVar := range section.Vars {
		if err := b.addVar(envVar); err != nil {
			return err
		}
	}

	b.lines = append(b.lines, "")

	return nil
}

// build returns the final .env file content
func (b *dotenvBuilder) build() string {
	content := strings.Join(b.lines, "\n")
	content = strings.TrimRight(content, "\n")
	return content + "\n"
}
