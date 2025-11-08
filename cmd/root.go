package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "auto-homelab",
	Short: "auto-homelab is a CLI to manage your homelab services and backups",
	Long:  "A Cobra CLI app for starting services, managing backups, and restoring instances in your homelab.",
	// Tell Cobra to NOT show the usage/help text after an error
	SilenceUsage: true,
	CompletionOptions: cobra.CompletionOptions{
		// See https://github.com/spf13/cobra/issues/1507
		HiddenDefaultCmd: true,
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initLogger(logLevel)
	},
}

// initLogger initializes the global logger with the specified level
func initLogger(level string) error {
	var logLevelVar slog.Level

	switch level {
	case "debug":
		logLevelVar = slog.LevelDebug
	case "info":
		logLevelVar = slog.LevelInfo
	case "warn":
		logLevelVar = slog.LevelWarn
	case "error":
		logLevelVar = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", level)
	}

	opts := &slog.HandlerOptions{
		Level: logLevelVar,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&logLevel, "log-level", "info",
		"Set the logging level (debug, info, warn, error)",
	)
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
