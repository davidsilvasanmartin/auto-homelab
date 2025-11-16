package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"

	"github.com/davidsilvasanmartin/auto-homelab/internal/backup"
	"github.com/davidsilvasanmartin/auto-homelab/internal/docker"
	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupLocalCmd)
	backupCmd.AddCommand(backupCloudCmd)

	// Add cloud backup subcommands
	backupCloudCmd.AddCommand(backupCloudInitCmd)
	backupCloudCmd.AddCommand(backupCloudCheckCmd)
	backupCloudCmd.AddCommand(backupCloudListCmd)
	backupCloudCmd.AddCommand(backupCloudPruneCmd)
	backupCloudCmd.AddCommand(backupCloudRestoreCmd)
	backupCloudCmd.AddCommand(backupCloudListFilesCmd)
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
	Short: "Manage cloud backups using restic and Backblaze B2",
	Long:  "Commands to manage cloud backups. Run without subcommands to perform a full backup (init, backup, prune).",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		return cloudBackup.RunFullBackup()
	},
}

var backupCloudInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the cloud backup repository",
	Long:  "Initializes a new restic repository in Backblaze B2 if it doesn't already exist.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		return cloudBackup.Init()
	},
}

var backupCloudCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check cloud backup repository integrity",
	Long:  "Verifies the integrity of the cloud backup repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		return cloudBackup.Check()
	},
}

var backupCloudListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cloud backup snapshots",
	Long:  "Lists all snapshots in the cloud backup repository.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		return cloudBackup.ListSnapshots()
	},
}

var backupCloudPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old cloud backup snapshots",
	Long:  "Removes old snapshots according to the configured retention policy.",
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		return cloudBackup.Prune()
	},
}

var backupCloudRestoreCmd = &cobra.Command{
	Use:   "restore [target-directory]",
	Short: "Restore the latest cloud backup snapshot",
	Long:  "Restores the latest snapshot from the cloud backup to the specified directory.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		targetDir := args[0]
		return cloudBackup.Restore(targetDir)
	},
}

var backupCloudListFilesCmd = &cobra.Command{
	Use:   "ls-files [snapshot-id]",
	Short: "List files in a specific cloud backup snapshot",
	Long:  "Lists all files contained in a specific snapshot.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env := system.NewDefaultEnv()
		config, err := getCloudBackupConfig(env)
		if err != nil {
			return err
		}
		cloudBackup := backup.NewCloudBackup(config)
		snapshotID := args[0]
		return cloudBackup.ListFiles(snapshotID)
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
	localBackupList.Add(backup.NewMariaDBLocalBackup(
		fireflyDBContainer,
		fireflyDBName,
		fireflyDBUser,
		fireflyDBPassword,
		filepath.Join(mainBackupDir, "firefly-db"),
	))

	return localBackupList, nil
}

// getCloudBackupConfig loads cloud backup configuration from environment variables
func getCloudBackupConfig(env system.Env) (backup.ResticConfig, error) {
	repositoryURL, err := env.GetRequiredEnv("HOMELAB_BACKUP_RESTIC_REPOSITORY")
	if err != nil {
		return backup.ResticConfig{}, err
	}

	b2KeyID, err := env.GetRequiredEnv("HOMELAB_BACKUP_B2_KEY_ID")
	if err != nil {
		return backup.ResticConfig{}, err
	}

	b2ApplicationKey, err := env.GetRequiredEnv("HOMELAB_BACKUP_B2_APPLICATION_KEY")
	if err != nil {
		return backup.ResticConfig{}, err
	}

	resticPassword, err := env.GetRequiredEnv("HOMELAB_BACKUP_RESTIC_PASSWORD")
	if err != nil {
		return backup.ResticConfig{}, err
	}

	backupPath, err := env.GetRequiredEnv("HOMELAB_BACKUP_PATH")
	if err != nil {
		return backup.ResticConfig{}, err
	}

	retentionDaysStr, err := env.GetRequiredEnv("HOMELAB_BACKUP_RETENTION_DAYS")
	if err != nil {
		return backup.ResticConfig{}, err
	}

	retentionDays, err := strconv.Atoi(retentionDaysStr)
	if err != nil {
		return backup.ResticConfig{}, fmt.Errorf("invalid retention days value: %w", err)
	}

	return backup.ResticConfig{
		RepositoryURL:    repositoryURL,
		B2KeyID:          b2KeyID,
		B2ApplicationKey: b2ApplicationKey,
		ResticPassword:   resticPassword,
		BackupPath:       backupPath,
		RetentionDays:    retentionDays,
	}, nil
}
