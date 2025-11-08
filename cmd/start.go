package cmd

import (
	"log/slog"

	"github.com/davidsilvasanmartin/auto-homelab/internal/docker"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start [service1 service2 ...]",
	Short: "Start services (or all services if none specified)",
	Long:  "Starts services in your homelab. If no service is provided, this would start all services.",
	RunE: func(cmd *cobra.Command, args []string) error {
		runner := docker.NewSystemRunner()
		return startServices(runner, args...)
	},
}

// startServices starts services by using docker compose.
// If the service is empty, starts all services
func startServices(dockerRunner docker.Runner, services ...string) error {
	if len(services) == 0 {
		slog.Info("Starting all services...")
	} else {
		slog.Info("Starting services", "services", services)
	}

	err := dockerRunner.ComposeStart(services)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		slog.Info("Successfully started all services")
	} else {
		slog.Info("Successfully started services", "services", services)
	}

	return nil
}
