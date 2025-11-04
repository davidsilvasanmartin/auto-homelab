package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/davidsilvasanmartin/auto-homelab/internal/backup"
	"github.com/davidsilvasanmartin/auto-homelab/internal/docker"
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
	Short: "Creates a local backup of all services' data",
	Long:  "Creates a local backup of all services' data into a single directory. Running this command will start up all services first. The backup operations run concurrently. It is important that backups are performed in periods of low service usage: for example, we would not want to backup a database that's in the process of updating a large number of records",
	RunE: func(cmd *cobra.Command, args []string) error {
		files := system.NewDefaultFilesHandler()
		env := system.NewDefaultEnv()
		if err := startAllContainers(); err != nil {
			return err
		}
		return runBackupLocal(files, env)
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

// startAllContainers starts all containers. Note that some containers (e.g., databases) need to be running in
// order to perform the backup, because we need to run commands on them (e.g., exporting the database)
func startAllContainers() error {
	dockerRunner := docker.NewSystemRunner()
	if err := dockerRunner.ComposeStart([]string{}); err != nil {
		return fmt.Errorf("failed to start all containers: %w", err)
	}
	return nil
}

func runBackupLocal(files system.FilesHandler, env system.Env) error {
	slog.Info("Creating local backup...")

	// Get the main backup directory path
	mainBackupDir, err := env.GetRequiredEnv("HOMELAB_BACKUP_PATH")
	if err != nil {
		return fmt.Errorf("failed to get backup path: %w", err)
	}

	// Prepare the main backup directory
	if err := files.EmptyDir(mainBackupDir); err != nil {
		return fmt.Errorf("failed to prepare backup directory: %w", err)
	}

	// Define backup operations
	localBackupList, err := buildLocalBackupList(mainBackupDir, env)
	if err != nil {
		return fmt.Errorf("failed to create backup operations: %w", err)
	}

	if err := localBackupList.RunAll(); err != nil {
		return fmt.Errorf("failed running backup operations: %w", err)
	}

	slog.Info("Local backup completed successfully")
	return nil
}

func buildLocalBackupList(mainBackupDir string, env system.Env) (*backup.LocalBackupList, error) {
	localBackupList := backup.NewLocalBackupList()

	calibreLibraryPath, err := env.GetRequiredEnv("HOMELAB_CALIBRE_LIBRARY_PATH")
	if err != nil {
		return nil, err
	}
	localBackupList.Add(backup.NewDirectoryLocalBackup(
		calibreLibraryPath,
		filepath.Join(mainBackupDir, "calibre-web-automated-calibre-library"),
		"",
	))

	calibreConfPath, err := env.GetRequiredEnv("HOMELAB_CALIBRE_CONF_PATH")
	if err != nil {
		return nil, err
	}
	localBackupList.Add(backup.NewDirectoryLocalBackup(
		calibreConfPath,
		filepath.Join(mainBackupDir, "calibre-web-automated-config"),
		"",
	))

	paperlessExportPath, err := env.GetRequiredEnv("HOMELAB_PAPERLESS_WEB_EXPORT_PATH")
	if err != nil {
		return nil, err
	}
	localBackupList.Add(backup.NewDirectoryLocalBackup(
		paperlessExportPath,
		filepath.Join(mainBackupDir, "paperless-ngx-webserver-export"),
		"docker compose exec -T paperless document_exporter -d ../export",
	))

	immichDBContainer, err := env.GetRequiredEnv("HOMELAB_IMMICH_DB_CONTAINER_NAME")
	if err != nil {
		return nil, err
	}
	immichDBName, err := env.GetRequiredEnv("HOMELAB_IMMICH_DB_DATABASE")
	if err != nil {
		return nil, err
	}
	immichDBUser, err := env.GetRequiredEnv("HOMELAB_IMMICH_DB_USER")
	if err != nil {
		return nil, err
	}
	immichDBPassword, err := env.GetRequiredEnv("HOMELAB_IMMICH_DB_PASSWORD")
	if err != nil {
		return nil, err
	}
	localBackupList.Add(backup.NewPostgreSQLLocalBackup(
		immichDBContainer,
		immichDBName,
		immichDBUser,
		immichDBPassword,
		filepath.Join(mainBackupDir, "immich-db"),
	))

	immichUploadPath, err := env.GetRequiredEnv("HOMELAB_IMMICH_WEB_UPLOAD_PATH")
	if err != nil {
		return nil, err
	}
	localBackupList.Add(backup.NewDirectoryLocalBackup(
		immichUploadPath,
		filepath.Join(mainBackupDir, "immich-library"),
		"",
	))

	fireflyDBContainer, err := env.GetRequiredEnv("HOMELAB_FIREFLY_DB_CONTAINER_NAME")
	if err != nil {
		return nil, err
	}
	fireflyDBName, err := env.GetRequiredEnv("HOMELAB_FIREFLY_DB_DATABASE")
	if err != nil {
		return nil, err
	}
	fireflyDBUser, err := env.GetRequiredEnv("HOMELAB_FIREFLY_DB_USER")
	if err != nil {
		return nil, err
	}
	fireflyDBPassword, err := env.GetRequiredEnv("HOMELAB_FIREFLY_DB_PASSWORD")
	if err != nil {
		return nil, err
	}
	localBackupList.Add(backup.NewMariaDBBackup(
		fireflyDBContainer,
		fireflyDBName,
		fireflyDBUser,
		fireflyDBPassword,
		filepath.Join(mainBackupDir, "firefly-db"),
	))

	return localBackupList, nil
}
