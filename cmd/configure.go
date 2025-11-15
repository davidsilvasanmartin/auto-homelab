package cmd

import (
	"log"
	"log/slog"

	"github.com/davidsilvasanmartin/auto-homelab/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	var configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Configure the environment variables for all services",
		Long:  "This utility configures the environment for all services in this project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configurer := config.NewDefaultConfigurer()
			return configure(configurer)
		},
	}
	rootCmd.AddCommand(configureCmd)
}

// configure starts the process of configuring the environment
func configure(configurer config.Configurer) error {
	slog.Info("Initiating configuration...")
	configRoot, err := configurer.LoadConfig("files/config/env.config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	envVars, err := configurer.ProcessConfig(configRoot)
	if err != nil {
		log.Fatalf("Failed to process config: %v", err)
	}

	err = configurer.WriteConfig(envVars)
	if err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}

	slog.Info("Configuration finished successfully")
	return nil
}
