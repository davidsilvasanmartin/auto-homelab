package cmd

import (
	"log/slog"

	"github.com/davidsilvasanmartin/auto-homelab/internal/docker"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop [service1 service2 ...]",
	Short: "Stops services (or all services if none specified)",
	Long:  "Stops services in your homelab. If no service is provided, this would stop all services.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerRunner := docker.NewSystemRunner()
		return stopServices(dockerRunner, args...)
	},
}

// stopServices stops services by using docker compose.
// If the service is empty, stops all services
func stopServices(dockerRunner docker.Runner, services ...string) error {
	if len(services) == 0 {
		slog.Info("Stopping all services...")
	} else {
		slog.Info("Stopping services", "services", services)
	}

	err := dockerRunner.ComposeStop(services)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		slog.Info("Successfully stopped all services")
	} else {
		slog.Info("Successfully stopped services", "services", services)
	}

	return nil
}
