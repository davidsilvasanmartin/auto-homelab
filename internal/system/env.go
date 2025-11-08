package system

import (
	"errors"
	"fmt"
	"os"
)

type Env interface {
	// GetRequiredEnv gets a required environment variable or returns an error
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

func (d *DefaultEnv) GetRequiredEnv(varName string) (string, error) {
	value, exists := d.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("%w: %q", ErrRequiredEnvNotFound, varName)
	}
	return value, nil
}
