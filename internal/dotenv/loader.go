package dotenv

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/spf13/viper"
)

var (
	viperInstance *viper.Viper
	mu            sync.RWMutex
)

// See the following:
//  https://github.com/spf13/viper?tab=readme-ov-file#should-viper-be-a-global-singleton-or-passed-around
//  https://github.com/spf13/viper?tab=readme-ov-file#is-it-safe-to-concurrently-read-and-write-to-a-viper-instance

// GetViper returns the Viper instance loaded by LoadDotEnv.
// Returns nil if LoadDotEnv has not been called or if loading failed.
func GetViper() *viper.Viper {
	mu.RLock()
	defer mu.RUnlock()
	return viperInstance
}

// LoadDotEnv loads the .env file from the current working directory.
// It uses Viper to read the .env file and stores the configuration in a package-level Viper instance.
// If the .env file is not found, it logs an info message and continues without error.
// If the .env file is found and loaded successfully, it logs an info message.
// Any other errors during loading are logged as warnings.
func LoadDotEnv() {
	v := viper.New()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, &viper.ConfigFileNotFoundError{}) {
			slog.Info(".env file not found in current directory, continuing without it")
			return
		}
		slog.Warn("Error loading .env file, continuing without it", "error", err.Error())
		return
	}

	mu.Lock()
	viperInstance = v
	mu.Unlock()

	slog.Info("Successfully loaded .env file from current directory",
		"file", v.ConfigFileUsed())
}
