package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupLocalCmd)
	backupCmd.AddCommand(backupCloudCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create backups (local or cloud)",
	Long:  "Commands to create backups of services locally or sync them to the cloud.",
}

var backupLocalCmd = &cobra.Command{
	Use:   "local",
	Short: "Create a local backup of all services' data",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Creating local backup...")
	},
}

var backupCloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Sync the local backup to the cloud",
	Long:  "Syncs the local backup to the configured cloud bucket. Assumes a local backup exists.",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Syncing backup to cloud...")
	},
}
