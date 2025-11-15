package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// The configurations below contain all the cases we want to test. I left them as global
// variables so that they can be seen and modified together.
var configJSON string = `{
	"prefix": "TEST",
	"sections": [
		{
			"name": "DATABASE",
			"description": "Database configuration",
			"vars": [
				{
					"name": "HOST",
					"type": "STRING",
					"description": "Database host",
					"value": "127.0.0.1"
				},
				{
					"name": "PASSWORD",
					"type": "STRING",
					"description": "Database password"
				}
			]
		},
		{
			"name": "SERVER",
			"description": "Server configuration",
			"vars": [
				{
					"name": "NAME",
					"type": "STRING",
					"description": "Server name",
					"value": "MyServer"
				}
			]
		}
	]
}`
var configDbHostValue string = "127.0.0.1"
var configServerNameValue string = "MyServer"
var configRoot *ConfigRoot = &ConfigRoot{
	Prefix: "TEST",
	Sections: []ConfigSection{
		{
			Name:        "DATABASE",
			Description: "Database configuration",
			Vars: []ConfigVar{
				{
					Name:        "HOST",
					Type:        "STRING",
					Description: "Database host",
					Value:       &configDbHostValue,
				},
				{
					Name:        "PASSWORD",
					Type:        "STRING",
					Description: "Database password",
					// Testing a nil value here
					Value: nil,
				},
			},
		},
		{
			Name:        "SERVER",
			Description: "Server configuration",
			Vars: []ConfigVar{
				{
					Name:        "NAME",
					Type:        "STRING",
					Description: "Server name",
					Value:       &configServerNameValue,
				},
			},
		},
	},
}
var testStrategy AcquireStrategy = &mockStrategy{
	acquireFunc: func(varName string, defaultSpec *string) (string, error) {
		// This code produces the values that I wrote on envVarRoot
		if defaultSpec != nil {
			return fmt.Sprintf("%s#%s#value", varName, *defaultSpec), nil
		}
		return fmt.Sprintf("%s##value", varName), nil
	},
}
var testStrategyRegistry StrategyRegistry = &mockStrategyRegistry{
	getFunc: func(varType string) (AcquireStrategy, error) {
		return testStrategy, nil
	},
}
var envVarRoot *EnvVarRoot = &EnvVarRoot{
	Sections: []EnvVarSection{
		{
			Name:        "TEST_DATABASE",
			Description: "Database configuration",
			Vars: []EnvVar{
				{
					Name:        "TEST_DATABASE_HOST",
					Description: "Database host",
					Value:       "TEST_DATABASE_HOST#127.0.0.1#value",
				},
				{
					Name:        "TEST_DATABASE_PASSWORD",
					Description: "Database password",
					Value:       "TEST_DATABASE_PASSWORD##value",
				},
			},
		},
		{
			Name:        "TEST_SERVER",
			Description: "Server configuration",
			Vars: []EnvVar{
				{
					Name:        "TEST_SERVER_NAME",
					Description: "Server name",
					Value:       "TEST_SERVER_NAME#MyServer#value",
				},
			},
		},
	},
}

func TestDefaultConfigurer_LoadConfig_Success(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	configurer := &DefaultConfigurer{
		prompter:         &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{},
		textFormatter:    &mockTextFormatter{},
		files:            &mockFiles{},
	}

	result, err := configurer.LoadConfig(configPath)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if diff := cmp.Diff(configRoot, result); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultConfigurer_LoadConfig_FileNotFound(t *testing.T) {
	configurer := &DefaultConfigurer{
		prompter:         &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{},
		textFormatter:    &mockTextFormatter{},
		files:            &mockFiles{},
	}
	nonExistentPath := "/path/that/does/not/exist/config.json"

	_, err := configurer.LoadConfig(nonExistentPath)

	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !errors.Is(err, ErrConfigFileRead) {
		t.Errorf("expected ErrConfigFileRead, got: %v", err)
	}
	if !strings.Contains(err.Error(), nonExistentPath) {
		t.Errorf("expected error message to contain path %q, got %q", nonExistentPath, err.Error())
	}
}

func TestDefaultConfigurer_LoadConfig_InvalidJSON(t *testing.T) {
	invalidJSON := `{
		"prefix": "TEST",
		"sections": [
			{
				"name": "DATABASE"
				"missing_comma": true
			}
		]
	}`
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	configurer := &DefaultConfigurer{
		prompter:         &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{},
		textFormatter:    &mockTextFormatter{},
		files:            &mockFiles{},
	}

	_, err := configurer.LoadConfig(configPath)

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !errors.Is(err, ErrConfigFileParse) {
		t.Errorf("expected ErrConfigFileParse, got: %v", err)
	}
	if !strings.Contains(err.Error(), configPath) {
		t.Errorf("expected error message to contain path %q, got %q", configPath, err.Error())
	}
}

func TestDefaultConfigurer_LoadConfig_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty.json")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	configurer := &DefaultConfigurer{
		prompter:         &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{},
		textFormatter:    &mockTextFormatter{},
		files:            &mockFiles{},
	}

	_, err := configurer.LoadConfig(configPath)

	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
	if !errors.Is(err, ErrConfigFileParse) {
		t.Errorf("expected ErrConfigFileParse, got: %v", err)
	}
}

func TestDefaultConfigurer_ProcessConfig_Success(t *testing.T) {
	var capturedInfoMessages []string
	configurer := &DefaultConfigurer{
		prompter: &mockPrompter{
			infoFunc: func(message string) {
				capturedInfoMessages = append(capturedInfoMessages, message)
			},
		},
		strategyRegistry: testStrategyRegistry,
		textFormatter:    &mockTextFormatter{},
		files:            &mockFiles{},
	}

	result, err := configurer.ProcessConfig(configRoot)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if diff := cmp.Diff(envVarRoot, result); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
	if len(capturedInfoMessages) < 2 {
		t.Errorf("expected at least 2 info messages, got %d", len(capturedInfoMessages))
	}
}

func TestDefaultConfigurer_ProcessConfig_EmptyConfig(t *testing.T) {
	configurer := &DefaultConfigurer{
		prompter:         &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{},
		textFormatter:    &mockTextFormatter{},
		files:            &mockFiles{},
	}
	emptyConfigRoot := &ConfigRoot{
		Prefix:   "EMPTY",
		Sections: []ConfigSection{},
	}

	result, err := configurer.ProcessConfig(emptyConfigRoot)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedEnvVarRoot := &EnvVarRoot{
		Sections: []EnvVarSection{},
	}
	if diff := cmp.Diff(expectedEnvVarRoot, result); diff != "" {
		t.Errorf("args mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultConfigurer_ProcessConfig_ErrorWhenGettingStrategyForVarType(t *testing.T) {
	var configRootWithUnknownVarType *ConfigRoot = &ConfigRoot{
		Prefix: "TEST",
		Sections: []ConfigSection{
			{
				Name:        "DATABASE",
				Description: "Database configuration",
				Vars: []ConfigVar{
					{
						Name:        "HOST",
						Type:        "UNKNOWN_TYPE",
						Description: "Database host",
					},
				},
			},
		},
	}
	expectedErr := errors.New("env var type not supported")
	configurer := &DefaultConfigurer{
		prompter: &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{
			getFunc: func(varType string) (AcquireStrategy, error) {
				return nil, expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		files:         &mockFiles{},
	}

	_, err := configurer.ProcessConfig(configRootWithUnknownVarType)

	if err == nil {
		t.Fatal("expected error for unknown variable type, got nil")
	}
	if !errors.Is(err, ErrVarType) {
		t.Errorf("expected ErrVarType, got: %v", err)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected ErrVarType, got: %v", err)
	}
	if !strings.Contains(err.Error(), "UNKNOWN_TYPE") {
		t.Errorf("expected error message to contain type %q, got %q", "UNKNOWN_TYPE", err.Error())
	}
	if !strings.Contains(err.Error(), "TEST_DATABASE_HOST") {
		t.Errorf("expected error message to contain var name %q, got %q", "TEST_DATABASE_HOST", err.Error())
	}
}

func TestDefaultConfigurer_ProcessConfig_ErrorWhenAcquiringValue(t *testing.T) {
	expectedErr := errors.New("acquisition failed")
	mockStrategyReturningErr := &mockStrategy{
		acquireFunc: func(varName string, defaultSpec *string) (string, error) {
			return "", expectedErr
		},
	}
	configurer := &DefaultConfigurer{
		prompter: &mockPrompter{},
		strategyRegistry: &mockStrategyRegistry{
			getFunc: func(varType string) (AcquireStrategy, error) {
				return mockStrategyReturningErr, nil
			},
		},
		textFormatter: &mockTextFormatter{},
		files:         &mockFiles{},
	}
	testConfigRoot := &ConfigRoot{
		Prefix: "TEST",
		Sections: []ConfigSection{
			{
				Name:        "SECTION1",
				Description: "Description",
				Vars: []ConfigVar{
					{Name: "VAR1", Type: "STRING", Description: "Var description"},
				},
			},
		},
	}

	_, err := configurer.ProcessConfig(testConfigRoot)

	if err == nil {
		t.Fatal("expected error when strategy fails, got nil")
	}
	if !errors.Is(err, ErrVarAcquireVal) {
		t.Errorf("expected ErrVarAcquireVal, got: %v", err)
	}
	if !strings.Contains(err.Error(), "TEST_SECTION1_VAR1") {
		t.Errorf("expected error message to contain var name %q, got %q", "TEST_SECTION1_VAR1", err.Error())
	}
}
