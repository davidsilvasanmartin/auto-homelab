package system

import (
	"errors"
	"fmt"
	"os"

	"github.com/davidsilvasanmartin/auto-homelab/internal/dotenv"
	"github.com/spf13/viper"
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
	// ViperConfig provides access to the Viper configuration loaded from .env file
	ViperConfig func() *viper.Viper
	// LookupEnv wraps os.LookupEnv for fallback to system environment variables
	LookupEnv func(key string) (string, bool)
}

func NewDefaultEnv() *DefaultEnv {
	return &DefaultEnv{
		ViperConfig: dotenv.GetViper,
		LookupEnv:   os.LookupEnv,
	}
}

func (d *DefaultEnv) GetEnv(varName string) (string, bool) {
	// First, check Viper configuration from .env file
	if d.ViperConfig != nil {
		if v := d.ViperConfig(); v != nil && v.IsSet(varName) {
			return v.GetString(varName), true
		}
	}

	// Fall back to system environment variables
	if d.LookupEnv != nil {
		return d.LookupEnv(varName)
	}

	return "", false
}

func (d *DefaultEnv) GetRequiredEnv(varName string) (string, error) {
	value, exists := d.GetEnv(varName)
	if !exists {
		return "", fmt.Errorf("%w: %q", ErrRequiredEnvNotFound, varName)
	}
	return value, nil
}
