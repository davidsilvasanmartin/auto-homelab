package system

import (
	"errors"
	"fmt"

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
}

func NewDefaultEnv() *DefaultEnv {
	return &DefaultEnv{
		ViperConfig: dotenv.GetViper,
	}
}

func (d *DefaultEnv) GetEnv(varName string) (string, bool) {
	if d.ViperConfig == nil {
		panic("viper config should be defined")
	}
	if v := d.ViperConfig(); v != nil && v.IsSet(varName) {
		return v.GetString(varName), true
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
