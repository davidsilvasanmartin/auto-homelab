package backup

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// CloudBackup orchestrates cloud backup operations using restic
type CloudBackup struct {
	client ResticClient
	files  system.FilesHandler
	config ResticConfig
}

// NewCloudBackup creates a new cloud backup instance
func NewCloudBackup(config ResticConfig) *CloudBackup {
	return &CloudBackup{
		client: NewDefaultResticClient(config),
		files:  system.NewDefaultFilesHandler(),
		config: config,
	}
}

// RunFullBackup executes a complete backup workflow: init, backup, and prune
func (c *CloudBackup) RunFullBackup() error {
	slog.Info("Starting full cloud backup workflow")

	slog.Info("Checking if repository exists...")
	if err := c.client.Init(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	slog.Info("Repository ready")

	if err := c.files.EnsureDirExists(c.config.BackupPath); err != nil {
		return fmt.Errorf("backup path does not exist: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	tags := []string{fmt.Sprintf("automatic-%s", timestamp)}
	slog.Info("Creating backup", "path", c.config.BackupPath, "tags", tags)
	if err := c.client.Backup(c.config.BackupPath, tags); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	slog.Info("Backup completed successfully")

	keepWithin := fmt.Sprintf("%dd", c.config.RetentionDays)
	slog.Info("Pruning old backups", "keepWithin", keepWithin)
	if err := c.client.Forget(keepWithin, true); err != nil {
		return fmt.Errorf("failed to prune old backups: %w", err)
	}
	slog.Info("Pruning completed successfully")

	slog.Info("Full cloud backup workflow completed successfully")
	return nil
}

// Init initializes the repository
func (c *CloudBackup) Init() error {
	slog.Info("Initializing repository...")
	if err := c.client.Init(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	slog.Info("Repository initialized successfully")
	return nil
}

// Check verifies repository integrity
func (c *CloudBackup) Check() error {
	slog.Info("Checking repository integrity...")
	if err := c.client.Check(); err != nil {
		return fmt.Errorf("repository check failed: %w", err)
	}
	slog.Info("Repository check completed successfully")
	return nil
}

// ListSnapshots lists all snapshots in the repository
func (c *CloudBackup) ListSnapshots() error {
	slog.Info("Listing snapshots...")
	if err := c.client.Snapshots(); err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}
	return nil
}

// Prune removes old backups according to retention policy
func (c *CloudBackup) Prune() error {
	keepWithin := fmt.Sprintf("%dd", c.config.RetentionDays)
	slog.Info("Pruning old backups", "keepWithin", keepWithin)
	if err := c.client.Forget(keepWithin, true); err != nil {
		return fmt.Errorf("failed to prune old backups: %w", err)
	}
	slog.Info("Pruning completed successfully")
	return nil
}

// Restore restores the latest snapshot to a target directory
func (c *CloudBackup) Restore(targetDir string) error {
	slog.Info("Restoring latest snapshot", "targetDir", targetDir)

	// Ensure target directory exists
	targetDir, err := c.files.GetAbsPath(targetDir)
	if err != nil {
		return fmt.Errorf("failed to convert target directory to an absolute path: %w", err)
	}
	if err := c.files.CreateDirIfNotExists(targetDir); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	if err := c.client.Restore(targetDir); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	slog.Info("Restore completed successfully", "targetDir", targetDir)
	return nil
}

// ListFiles lists files in a specific snapshot
func (c *CloudBackup) ListFiles(snapshotID string) error {
	slog.Info("Listing files in snapshot", "snapshotID", snapshotID)
	if err := c.client.ListFiles(snapshotID); err != nil {
		return fmt.Errorf("failed to list files in snapshot: %w", err)
	}
	return nil
}
