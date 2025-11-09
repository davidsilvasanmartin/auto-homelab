package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Orchestration logic for the config feature

type Configurer interface {
	// LoadConfig loads the configuration from a file
	LoadConfig(configFilePath string) (*ConfigRoot, error)
	// ProcessConfig processes the configuration and retrieves variable values
	ProcessConfig(configRoot *ConfigRoot) (*EnvVarRoot, error)
}

var (
	ErrConfigFileRead  = errors.New("failed to read config file")
	ErrConfigFileParse = errors.New("failed to parse config file")
	ErrVarType         = errors.New("error processing variable type")
	ErrVarAcquireVal   = errors.New("error acquiring value for variable")
)

type DefaultConfigurer struct {
	prompter         Prompter
	strategyRegistry *StrategyRegistry
}

func NewDefaultConfigurer() *DefaultConfigurer {
	return &DefaultConfigurer{
		prompter:         NewConsolePrompter(),
		strategyRegistry: NewStrategyRegistry(),
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
				Type:        configVar.Type,
				Description: configVar.Description,
				Value:       value,
			}
			section.Vars = append(section.Vars, envVar)
		}

		root.Sections = append(root.Sections, section)
	}

	return root, nil
}
