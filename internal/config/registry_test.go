package config

import (
	"errors"
	"strings"
	"testing"
)

func TestStrategyRegistry_Register_Success(t *testing.T) {
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}
	strategy := &mockStrategy{}

	registry.Register("TEST", strategy)

	if len(registry.strategies) != 1 {
		t.Errorf("expected 1 strategy registered, got %d", len(registry.strategies))
	}
	if registry.strategies["TEST"] != strategy {
		t.Error("expected strategy to be registered with key 'TEST'")
	}
}

func TestStrategyRegistry_Register_ConvertsToUppercase(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected string
	}{
		{"lowercase", "test", "TEST"},
		{"uppercase", "TEST", "TEST"},
		{"mixed_case", "TeSt", "TEST"},
		{"with_underscore", "test_type", "TEST_TYPE"},
		{"with_hyphen", "test-type", "TEST-TYPE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &DefaultStrategyRegistry{
				strategies: make(map[string]AcquireStrategy),
			}
			strategy := &mockStrategy{}

			registry.Register(tt.typeName, strategy)

			if registry.strategies[tt.expected] != strategy {
				t.Errorf("expected strategy to be registered with key %q", tt.expected)
			}
		})
	}
}

func TestStrategyRegistry_Register_ReplacesExistingStrategy(t *testing.T) {
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}
	firstStrategy := &mockStrategy{
		acquireFunc: func(varName string, defaultSpec *string) (string, error) {
			return "first", nil
		},
	}
	secondStrategy := &mockStrategy{
		acquireFunc: func(varName string, defaultSpec *string) (string, error) {
			return "second", nil
		},
	}

	registry.Register("TEST", firstStrategy)
	registry.Register("TEST", secondStrategy)

	if len(registry.strategies) != 1 {
		t.Errorf("expected 1 strategy registered, got %d", len(registry.strategies))
	}
	if registry.strategies["TEST"] != secondStrategy {
		t.Error("expected second strategy to replace first strategy")
	}

	// Verify the replaced strategy works correctly
	result, err := registry.strategies["TEST"].Acquire("VAR", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "second" {
		t.Errorf("expected result 'second', got %q", result)
	}
}

func TestStrategyRegistry_Register_DifferentCasesReplaceEachOther(t *testing.T) {
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}
	firstStrategy := &mockStrategy{}
	secondStrategy := &mockStrategy{}

	registry.Register("test", firstStrategy)
	registry.Register("TEST", secondStrategy)

	if len(registry.strategies) != 1 {
		t.Errorf("expected 1 strategy registered (same key), got %d", len(registry.strategies))
	}
	if registry.strategies["TEST"] != secondStrategy {
		t.Error("expected second strategy to replace first strategy")
	}
}

func TestStrategyRegistry_Get_ReturnsRegisteredStrategy(t *testing.T) {
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}
	strategy := &mockStrategy{}
	registry.Register("TEST", strategy)

	result, err := registry.Get("TEST")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != strategy {
		t.Error("expected to get the registered strategy")
	}
}

func TestStrategyRegistry_Get_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name       string
		registerAs string
		retrieveAs string
	}{
		{"exact_match", "TEST", "TEST"},
		{"lowercase_retrieve", "TEST", "test"},
		{"uppercase_retrieve", "test", "TEST"},
		{"mixed_case_retrieve", "TEST", "TeSt"},
		{"mixed_case_register", "TeSt", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &DefaultStrategyRegistry{
				strategies: make(map[string]AcquireStrategy),
			}
			strategy := &mockStrategy{}
			registry.Register(tt.registerAs, strategy)

			result, err := registry.Get(tt.retrieveAs)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != strategy {
				t.Error("expected to get the registered strategy")
			}
		})
	}
}

func TestStrategyRegistry_Get_UnregisteredType_ReturnsError(t *testing.T) {
	unregisteredType := "NONEXISTENT"
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}

	_, err := registry.Get(unregisteredType)

	if err == nil {
		t.Fatal("expected error for unregistered type, got nil")
	}
	if !errors.Is(err, ErrVarTypeNotSupported) {
		t.Errorf("expected ErrVarTypeNotSupported, got: %v", err)
	}
	if !strings.Contains(err.Error(), unregisteredType) {
		t.Errorf("expected error message to contain %q, got %q", unregisteredType, err.Error())
	}
}

func TestStrategyRegistry_Get_MultipleStrategies(t *testing.T) {
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}
	strategy1 := &mockStrategy{
		acquireFunc: func(varName string, defaultSpec *string) (string, error) {
			return "value1", nil
		},
	}
	strategy2 := &mockStrategy{
		acquireFunc: func(varName string, defaultSpec *string) (string, error) {
			return "value2", nil
		},
	}
	strategy3 := &mockStrategy{
		acquireFunc: func(varName string, defaultSpec *string) (string, error) {
			return "value3", nil
		},
	}

	registry.Register("TYPE1", strategy1)
	registry.Register("TYPE2", strategy2)
	registry.Register("TYPE3", strategy3)

	// Test all three strategies can be retrieved
	result1, err := registry.Get("TYPE1")
	if err != nil {
		t.Fatalf("expected no error for TYPE1, got %v", err)
	}
	val1, _ := result1.Acquire("VAR", nil)
	if val1 != "value1" {
		t.Errorf("expected value1, got %q", val1)
	}

	result2, err := registry.Get("type2")
	if err != nil {
		t.Fatalf("expected no error for type2, got %v", err)
	}
	val2, _ := result2.Acquire("VAR", nil)
	if val2 != "value2" {
		t.Errorf("expected value2, got %q", val2)
	}

	result3, err := registry.Get("TyPe3")
	if err != nil {
		t.Fatalf("expected no error for TyPe3, got %v", err)
	}
	val3, _ := result3.Acquire("VAR", nil)
	if val3 != "value3" {
		t.Errorf("expected value3, got %q", val3)
	}
}
