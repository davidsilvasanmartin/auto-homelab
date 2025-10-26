package cmd

import (
	"auto-homelab/internal/require"
	"log/slog"
	"os"
	"os/exec"

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
		return startServices(args...)
	},
}

// startServices starts services by using docker compose.
// If the service is empty, starts all services
func startServices(services ...string) error {
	if len(services) == 0 {
		slog.Info("Starting all services...")
	} else {
		slog.Info("Starting services", "services", services)
	}

	if err := require.RequireDocker(); err != nil {
		return err
	}
	if err := require.RequireFilesInWd("docker-compose.yml", ".env"); err != nil {
		return err
	}

	args := []string{"compose", "up", "-d"}
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
		slog.Info("Successfully started all services")
	} else {
		slog.Info("Successfully started services", "services", services)
	}

	return nil
}

// TODO CHECK THIS CODE
/**
// parseDockerComposeError provides better error messages for common docker compose failures
func parseDockerComposeError(err error, services []string) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Docker compose failed with a non-zero exit code
		stderr := string(exitErr.Stderr)

		// Check for common error patterns
		switch {
		case strings.Contains(stderr, "no such service"):
			if len(services) > 0 {
				return fmt.Errorf("service '%s' not found in docker-compose.yml", services[0])
			}
			return fmt.Errorf("service not found in docker-compose.yml")

		case strings.Contains(stderr, "variable is not set"):
			return fmt.Errorf("missing required environment variable (check your .env file): %w", err)

		case strings.Contains(stderr, "Cannot connect to the Docker daemon"):
			return fmt.Errorf("cannot connect to Docker daemon (is Docker running?): %w", err)

		case strings.Contains(stderr, "docker-compose.yml"):
			return fmt.Errorf("docker-compose.yml file issue: %w", err)

		default:
			return fmt.Errorf("docker compose failed: %w", err)
		}
	}

	return fmt.Errorf("failed to execute docker compose: %w", err)
}
*/
