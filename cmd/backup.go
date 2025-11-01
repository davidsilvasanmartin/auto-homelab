package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/davidsilvasanmartin/auto-homelab/internal/backup"
	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
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
	RunE:  runBackupLocal,
}

var backupCloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Sync the local backup to the cloud",
	Long:  "Syncs the local backup to the configured cloud bucket. Assumes a local backup exists.",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Syncing backup to cloud...")
	},
}

func runBackupLocal(cmd *cobra.Command, args []string) error {
	slog.Info("Creating local backup...")
	sys := system.NewDefaultSystem()
	stdout := os.Stdout
	stderr := os.Stderr

	// Get the main backup directory path
	mainBackupDir, err := backup.GetRequiredEnv("HOMELAB_BACKUP_PATH")
	if err != nil {
		return fmt.Errorf("failed to get backup path: %w", err)
	}

	// Prepare the main backup directory
	if err := backup.PrepareBackupDirectory(mainBackupDir, stdout); err != nil {
		return fmt.Errorf("failed to prepare backup directory: %w", err)
	}

	// Define backup operations
	backupOperations, err := createBackupOperations(mainBackupDir, sys, stdout, stderr)
	if err != nil {
		return fmt.Errorf("failed to create backup operations: %w", err)
	}

	// Run all backup operations
	for _, operation := range backupOperations {
		if _, err := operation.Run(); err != nil {
			return fmt.Errorf("backup operation failed: %w", err)
		}
	}

	return nil
}

func createBackupOperations(mainBackupDir string, sys system.System, stdout, stderr *os.File) ([]backup.Backup, error) {
	// Helper function to get required env variables with better error context
	getEnv := func(key string) (string, error) {
		val, err := backup.GetRequiredEnv(key)
		if err != nil {
			return "", fmt.Errorf("failed to get %s: %w", key, err)
		}
		return val, nil
	}

	// Get all required environment variables
	calibreLibraryPath, err := getEnv("HOMELAB_CALIBRE_LIBRARY_PATH")
	if err != nil {
		return nil, err
	}

	calibreConfPath, err := getEnv("HOMELAB_CALIBRE_CONF_PATH")
	if err != nil {
		return nil, err
	}

	paperlessExportPath, err := getEnv("HOMELAB_PAPERLESS_WEB_EXPORT_PATH")
	if err != nil {
		return nil, err
	}
	//
	//immichDBContainer, err := getEnv("HOMELAB_IMMICH_DB_CONTAINER_NAME")
	//if err != nil {
	//	return nil, err
	//}
	//
	//immichDBName, err := getEnv("HOMELAB_IMMICH_DB_DATABASE")
	//if err != nil {
	//	return nil, err
	//}
	//
	//immichDBUser, err := getEnv("HOMELAB_IMMICH_DB_USER")
	//if err != nil {
	//	return nil, err
	//}
	//
	//immichDBPassword, err := getEnv("HOMELAB_IMMICH_DB_PASSWORD")
	//if err != nil {
	//	return nil, err
	//}
	//
	//immichUploadPath, err := getEnv("HOMELAB_IMMICH_WEB_UPLOAD_PATH")
	//if err != nil {
	//	return nil, err
	//}
	//
	//fireflyDBContainer, err := getEnv("HOMELAB_FIREFLY_DB_CONTAINER_NAME")
	//if err != nil {
	//	return nil, err
	//}
	//
	//fireflyDBName, err := getEnv("HOMELAB_FIREFLY_DB_DATABASE")
	//if err != nil {
	//	return nil, err
	//}
	//
	//fireflyDBUser, err := getEnv("HOMELAB_FIREFLY_DB_USER")
	//if err != nil {
	//	return nil, err
	//}
	//
	//fireflyDBPassword, err := getEnv("HOMELAB_FIREFLY_DB_PASSWORD")
	//if err != nil {
	//	return nil, err
	//}

	operations := []backup.Backup{
		backup.NewDirectoryBackup(
			calibreLibraryPath,
			filepath.Join(mainBackupDir, "calibre-web-automated-calibre-library"),
			"",
			"",
			sys,
			stdout,
			stderr,
		),

		backup.NewDirectoryBackup(
			calibreConfPath,
			filepath.Join(mainBackupDir, "calibre-web-automated-config"),
			"docker compose stop calibre",
			"docker compose start calibre",
			sys,
			stdout,
			stderr,
		),

		backup.NewDirectoryBackup(
			paperlessExportPath,
			filepath.Join(mainBackupDir, "paperless-ngx-webserver-export"),
			"docker compose start paperless-redis paperless-db paperless && docker compose exec -T paperless document_exporter -d ../export",
			"", // no post-command
			sys,
			stdout,
			stderr,
		),

		// TODO
		//backup.NewPostgreSQLBackup(
		//	immichDBContainer,
		//	immichDBName,
		//	immichDBUser,
		//	immichDBPassword,
		//	filepath.Join(mainBackupDir, "immich-db"),
		//	sys,
		//	stdout,
		//	stderr,
		//),
		//
		//backup.NewDirectoryBackup(
		//	immichUploadPath,
		//	filepath.Join(mainBackupDir, "immich-library"),
		//	"", // no pre-command
		//	"", // no post-command
		//	sys,
		//	stdout,
		//	stderr,
		//),
		//
		//backup.NewMariaDBBackup(
		//	fireflyDBContainer,
		//	fireflyDBName,
		//	fireflyDBUser,
		//	fireflyDBPassword,
		//	filepath.Join(mainBackupDir, "firefly-db"),
		//	sys,
		//	stdout,
		//	stderr,
		//),
	}

	return operations, nil
}
