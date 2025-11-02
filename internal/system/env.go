package system

import (
	"fmt"
	"os"
)

type Env interface {
	GetRequiredEnv(varName string) (string, error)
}

type DefaultEnv struct {
	// LookupEnv wraps os.LookupEnv
	LookupEnv func(key string) (string, bool)
}

func NewDefaultEnv() *DefaultEnv {
	return &DefaultEnv{
		LookupEnv: os.LookupEnv,
	}
}

// GetRequiredEnv gets a required environment variable or returns an error
func (d *DefaultEnv) GetRequiredEnv(varName string) (string, error) {
	value, exists := d.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("missing required environment variable: %s", varName)
	}
	return value, nil
}
