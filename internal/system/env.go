package system

import (
	"errors"
	"fmt"
	"os"
)

type Env interface {
	// GetEnv gets an environment variable and returns a bool to indicate whether it exists
	GetEnv(varName string) (value string, exists bool)
	// GetRequiredEnv returns (value, true) if an environment variable with name varName exists,
	// and ("", false) if it does not exist
	GetRequiredEnv(varName string) (string, error)
}

var (
	ErrRequiredEnvNotFound = errors.New("missing required environment variable")
)

type DefaultEnv struct {
	// LookupEnv wraps os.LookupEnv
	LookupEnv func(key string) (string, bool)
}

func NewDefaultEnv() *DefaultEnv {
	return &DefaultEnv{
		LookupEnv: os.LookupEnv,
	}
}

func (d *DefaultEnv) GetEnv(varName string) (string, bool) {
	return d.LookupEnv(varName)
}

func (d *DefaultEnv) GetRequiredEnv(varName string) (string, error) {
	value, exists := d.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("%w: %q", ErrRequiredEnvNotFound, varName)
	}
	return value, nil
}
