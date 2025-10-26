package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "auto-homelab",
	Short: "auto-homelab is a CLI to manage your homelab services and backups",
	Long:  "A Cobra CLI for starting services, managing backups, and restoring instances in your homelab.",
	CompletionOptions: cobra.CompletionOptions{
		// See https://github.com/spf13/cobra/issues/1507
		HiddenDefaultCmd: true,
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
