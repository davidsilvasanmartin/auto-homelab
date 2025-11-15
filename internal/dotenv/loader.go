package dotenv

import (
	"errors"
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// LoadDotEnv loads the .env file from the current working directory.
// It uses Viper to read the .env file and loads all variables into the environment.
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

	for key, value := range v.AllSettings() {
		if strValue, ok := value.(string); ok {
			if err := os.Setenv(key, strValue); err != nil {
				slog.Warn("Failed to set environment variable from .env file",
					"key", key,
					"error", err.Error())
			}
		}
	}

	slog.Info("Successfully loaded .env file from current directory",
		"file", v.ConfigFileUsed())
}
