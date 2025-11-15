package config

import (
	"errors"
	"fmt"
	"strings"
)

// Variable acquisition strategy registration and lookup

// StrategyRegistry registers variable acquisition strategies
type StrategyRegistry interface {
	Register(typeName string, strategy AcquireStrategy)
	Get(typeName string) (AcquireStrategy, error)
}

// DefaultStrategyRegistry maintains a registry of acquisition strategies by type name
type DefaultStrategyRegistry struct {
	strategies map[string]AcquireStrategy
}

var (
	ErrVarTypeNotSupported = errors.New("unsupported variable type")
)

// NewDefaultStrategyRegistry creates a new registry with default strategies
func NewDefaultStrategyRegistry() *DefaultStrategyRegistry {
	registry := &DefaultStrategyRegistry{
		strategies: make(map[string]AcquireStrategy),
	}

	// Register default strategies
	registry.Register("CONSTANT", NewConstantStrategy())
	registry.Register("GENERATED", NewGeneratedStrategy())
	registry.Register("IP", NewIPStrategy())
	registry.Register("STRING", NewStringStrategy())
	registry.Register("PATH", NewPathStrategy())

	return registry
}

// Register adds or replaces a strategy for a given type name
func (r *DefaultStrategyRegistry) Register(typeName string, strategy AcquireStrategy) {
	r.strategies[strings.ToUpper(typeName)] = strategy
}

// Get retrieves a strategy by type name
func (r *DefaultStrategyRegistry) Get(typeName string) (AcquireStrategy, error) {
	key := strings.ToUpper(typeName)
	strategy, ok := r.strategies[key]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrVarTypeNotSupported, typeName)
	}
	return strategy, nil
}
