package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start [service]",
	Short: "Start a service (or all services)",
	Long:  "Starts a service in your homelab. If no service is provided, this would start all services.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return startAllServices()
		}
		return startOneService(args[0])
	},
}

func startAllServices() error {
	slog.Info("Starting all services...")
	return nil
}

func startOneService(service string) error {
	slog.Info("Starting service", "service", service)
	return nil
}
