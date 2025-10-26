package cmd

import (
	"auto-homelab/internal/require"
	"log/slog"
	"os"
	"os/exec"

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
		return stopServices(args...)
	},
}

// stopServices stops services by using docker compose.
// If the service is empty, stops all services
func stopServices(services ...string) error {
	if len(services) == 0 {
		slog.Info("Stopping all services...")
	} else {
		slog.Info("Stopping services", "services", services)
	}

	if err := require.RequireDocker(); err != nil {
		return err
	}

	args := []string{"compose", "stop"}
	args = append(args, services...)

	cmd := exec.Command("docker", args...)
	// Capture both stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Debug("Executing docker compose command",
		"command", "docker",
		"args", args,
		"dir", cmd.Dir,
	)

	if err := cmd.Run(); err != nil {
		return err
	}

	if len(services) == 0 {
		slog.Info("Successfully stopped all services")
	} else {
		slog.Info("Successfully stopped services", "services", services)
	}

	return nil
}
