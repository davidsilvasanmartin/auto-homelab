package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.AddCommand(restorePaperlessCmd)
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore data for supported services",
	Long:  "Commands to restore data for supported services from backups.",
}

var restorePaperlessCmd = &cobra.Command{
	Use:   "paperless",
	Short: "Restore the paperless-ngx data",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Restoring paperless-ngx...")
	},
}
