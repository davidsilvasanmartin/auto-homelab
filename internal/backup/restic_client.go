package backup

import (
	"fmt"

	"github.com/davidsilvasanmartin/auto-homelab/internal/format"
	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// ResticClient defines the interface for interacting with restic
type ResticClient interface {
	// Init initializes a new restic repository if it doesn't exist
	Init() error
	// Backup creates a new backup snapshot
	Backup(path string, tags []string) error
	// Forget removes snapshots according to retention policy
	Forget(keepWithin string, prune bool) error
	// Check verifies repository integrity
	Check() error
	// Snapshots lists all snapshots
	Snapshots() error
	// ListFiles lists files in a specific snapshot
	ListFiles(snapshotID string) error
	// Restore restores the latest snapshot to a target directory
	Restore(targetDir string) error
}

// ResticConfig holds the configuration for restic operations
type ResticConfig struct {
	RepositoryURL    string
	B2KeyID          string
	B2ApplicationKey string
	ResticPassword   string
	BackupPath       string
	RetentionDays    int
}

// DefaultResticClient is the default implementation of ResticClient
type DefaultResticClient struct {
	commands      system.Commands
	textFormatter format.TextFormatter
	config        ResticConfig
}

// NewDefaultResticClient creates a new restic client with the provided configuration
func NewDefaultResticClient(config ResticConfig) *DefaultResticClient {
	return &DefaultResticClient{
		commands:      system.NewDefaultCommands(),
		textFormatter: format.NewDefaultTextFormatter(),
		config:        config,
	}
}

// execRestic executes a restic command with the configured environment
// It uses shell execution to properly set environment variables
func (r *DefaultResticClient) execRestic(args ...string) error {
	// Build the command with environment variables
	// We need to properly escape the values to prevent shell injection
	envVars := fmt.Sprintf(
		"RESTIC_REPOSITORY=%s B2_ACCOUNT_ID=%s B2_ACCOUNT_KEY=%s RESTIC_PASSWORD=%s",
		r.textFormatter.QuoteForPOSIXShell(r.config.RepositoryURL),
		r.textFormatter.QuoteForPOSIXShell(r.config.B2KeyID),
		r.textFormatter.QuoteForPOSIXShell(r.config.B2ApplicationKey),
		r.textFormatter.QuoteForPOSIXShell(r.config.ResticPassword),
	)

	cmdStr := envVars + " restic"
	for _, arg := range args {
		cmdStr += " " + arg
	}

	cmd := r.commands.ExecShellCommand(cmdStr)
	return cmd.Run()
}

// Init initializes a new restic repository if it doesn't exist
func (r *DefaultResticClient) Init() error {
	// First check if repository exists by running snapshots
	err := r.execRestic("snapshots")
	if err == nil {
		// Repository exists
		return nil
	}

	// Repository doesn't exist, initialize it
	return r.execRestic("init")
}

// Backup creates a new backup snapshot
func (r *DefaultResticClient) Backup(path string, tags []string) error {
	args := []string{"backup", path, "--verbose"}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}
	return r.execRestic(args...)
}

// Forget removes snapshots according to retention policy
func (r *DefaultResticClient) Forget(keepWithin string, prune bool) error {
	args := []string{"forget", "--keep-within", keepWithin}
	if prune {
		args = append(args, "--prune")
	}
	return r.execRestic(args...)
}

// Check verifies repository integrity
func (r *DefaultResticClient) Check() error {
	return r.execRestic("check")
}

// Snapshots lists all snapshots
func (r *DefaultResticClient) Snapshots() error {
	return r.execRestic("snapshots")
}

// ListFiles lists files in a specific snapshot
func (r *DefaultResticClient) ListFiles(snapshotID string) error {
	return r.execRestic("ls", snapshotID)
}

// Restore restores the latest snapshot to a target directory
func (r *DefaultResticClient) Restore(targetDir string) error {
	return r.execRestic("restore", "latest", "--target", r.textFormatter.QuoteForPOSIXShell(targetDir), "--verbose")
}
