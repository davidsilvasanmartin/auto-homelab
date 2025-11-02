package backup

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// TODO think about prefixing things with "Local"

// Backup is the interface for all backup operations
type Backup interface {
	// Run executes the backup operation and returns the path to the backup
	Run() (string, error)
}

// BaseBackup contains common backup functionality
type BaseBackup struct {
	outputPath string
	system     system.System
	files      system.FilesHandler
	stdout     io.Writer
	stderr     io.Writer
}

// NewBaseBackup creates a new base backup instance
func NewBaseBackup(outputPath string, sys system.System, files system.FilesHandler, stdout, stderr io.Writer) *BaseBackup {
	return &BaseBackup{
		outputPath: outputPath,
		system:     sys,
		files:      files,
		stdout:     stdout,
		stderr:     stderr,
	}
}

// prepareOutputDirectory ensures the output directory exists
func (b *BaseBackup) prepareOutputDirectory() error {
	if err := os.MkdirAll(b.outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", b.outputPath, err)
	}
	return nil
}

// DirectoryBackup handles directory copy operations
type DirectoryBackup struct {
	*BaseBackup
	sourcePath  string
	preCommand  string
	postCommand string
}

// NewDirectoryBackup creates a new directory backup instance
func NewDirectoryBackup(sourcePath, outputPath string, preCommand, postCommand string, sys system.System, files system.FilesHandler, stdout, stderr io.Writer) *DirectoryBackup {
	return &DirectoryBackup{
		BaseBackup:  NewBaseBackup(outputPath, sys, files, stdout, stderr),
		sourcePath:  sourcePath,
		preCommand:  preCommand,
		postCommand: postCommand,
	}
}

// runShellCommand executes a shell command
func (d *DirectoryBackup) runShellCommand(command string) error {
	cmd := d.system.ExecCommand("sh", "-c", command)
	cmd.Stdout = d.stdout
	cmd.Stderr = d.stderr
	return cmd.Run()
}

// Run executes the directory backup operation
func (d *DirectoryBackup) Run() (string, error) {
	if err := d.prepareOutputDirectory(); err != nil {
		return "", err
	}

	// Run pre-command if provided
	if d.preCommand != "" {
		slog.Info("Running pre-command", "preCommand", d.preCommand)
		if err := d.runShellCommand(d.preCommand); err != nil {
			return "", fmt.Errorf("pre-command failed: %w", err)
		}
		slog.Info("Successfully ran pre-command")
	}

	// Check if source exists and is a directory
	if err := d.files.RequireDir(d.sourcePath); err != nil {
		return "", err
	}

	slog.Info("Copying directory", "sourcePath", d.sourcePath, "outputPath", d.outputPath)
	if err := copyDir(d.sourcePath, d.outputPath); err != nil {
		return "", fmt.Errorf("failed to copy directory: %w", err)
	}
	slog.Info("Successfully copied directory", "sourcePath", d.sourcePath, "outputPath", d.outputPath)

	if d.postCommand != "" {
		slog.Info("Running post-command", "postCommand", d.postCommand)
		if err := d.runShellCommand(d.postCommand); err != nil {
			return "", fmt.Errorf("post-command failed: %w", err)
		}
		slog.Info("Successfully ran post-command")
	}

	return "", nil
}

/* TODO FROM HERE ********************************************************/
/*********************************

// PostgreSQLBackup handles PostgreSQL database backups using docker exec
type PostgreSQLBackup struct {
	*BaseBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewPostgreSQLBackup creates a new PostgreSQL backup instance
func NewPostgreSQLBackup(containerName, dbName, username, password, outputPath string, sys system.System, stdout, stderr io.Writer) *PostgreSQLBackup {
	return &PostgreSQLBackup{
		BaseBackup:    NewBaseBackup(outputPath, sys, stdout, stderr),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the PostgreSQL backup
func (p *PostgreSQLBackup) Run() (string, error) {
	if err := p.prepareOutputDirectory(); err != nil {
		return "", err
	}

	backupFile := filepath.Join(p.outputPath, p.dbName+".sql")

	// Construct the docker command
	quotedPassword := shQuote(p.password)
	dockerCommand := fmt.Sprintf(
		`docker exec -i %s /bin/bash -c "PGPASSWORD=%s pg_dump --username %s %s" > %s`,
		p.containerName,
		quotedPassword,
		p.username,
		p.dbName,
		backupFile,
	)

	fmt.Fprintf(p.stdout, "Running backup command for database %s in container %s\n", p.dbName, p.containerName)

	cmd := p.system.ExecCommand("sh", "-c", dockerCommand)
	cmd.Stdout = p.stdout
	cmd.Stderr = p.stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up database %s: %w", p.dbName, err)
	}

	fmt.Fprintf(p.stdout, "Successfully backed up %s to %s\n\n", p.dbName, backupFile)
	return backupFile, nil
}

// MySQLBackup handles MySQL database backups using docker exec
type MySQLBackup struct {
	*BaseBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMySQLBackup creates a new MySQL backup instance
func NewMySQLBackup(containerName, dbName, username, password, outputPath string, sys system.System, stdout, stderr io.Writer) *MySQLBackup {
	return &MySQLBackup{
		BaseBackup:    NewBaseBackup(outputPath, sys, stdout, stderr),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the MySQL backup
func (m *MySQLBackup) Run() (string, error) {
	if err := m.prepareOutputDirectory(); err != nil {
		return "", err
	}

	backupFile := filepath.Join(m.outputPath, m.dbName+".sql")

	// Construct the docker command
	quotedPassword := shQuote(m.password)
	dockerCommand := fmt.Sprintf(
		`docker exec -i %s /bin/bash -c "MYSQL_PWD=%s mysqldump --user %s %s" > %s`,
		m.containerName,
		quotedPassword,
		m.username,
		m.dbName,
		backupFile,
	)

	fmt.Fprintf(m.stdout, "Running backup command for database %s in container %s\n", m.dbName, m.containerName)

	cmd := m.system.ExecCommand("sh", "-c", dockerCommand)
	cmd.Stdout = m.stdout
	cmd.Stderr = m.stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up database %s: %w", m.dbName, err)
	}

	fmt.Fprintf(m.stdout, "Successfully backed up %s to %s\n\n", m.dbName, backupFile)
	return backupFile, nil
}

// MariaDBBackup handles MariaDB database backups using docker exec
type MariaDBBackup struct {
	*BaseBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMariaDBBackup creates a new MariaDB backup instance
func NewMariaDBBackup(containerName, dbName, username, password, outputPath string, sys system.System, stdout, stderr io.Writer) *MariaDBBackup {
	return &MariaDBBackup{
		BaseBackup:    NewBaseBackup(outputPath, sys, stdout, stderr),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the MariaDB backup
func (m *MariaDBBackup) Run() (string, error) {
	if err := m.prepareOutputDirectory(); err != nil {
		return "", err
	}

	backupFile := filepath.Join(m.outputPath, m.dbName+".sql")

	// Construct the docker command using mariadb-dump
	quotedPassword := shQuote(m.password)
	dockerCommand := fmt.Sprintf(
		`docker exec -i %s /bin/bash -c "MYSQL_PWD=%s mariadb-dump --user %s %s" > %s`,
		m.containerName,
		quotedPassword,
		m.username,
		m.dbName,
		backupFile,
	)

	fmt.Fprintf(m.stdout, "Running backup command for MariaDB database %s in container %s\n", m.dbName, m.containerName)

	cmd := m.system.ExecCommand("sh", "-c", dockerCommand)
	cmd.Stdout = m.stdout
	cmd.Stderr = m.stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up MariaDB database %s: %w", m.dbName, err)
	}

	fmt.Fprintf(m.stdout, "Successfully backed up %s to %s\n\n", m.dbName, backupFile)
	return backupFile, nil
}

// shQuote wraps a string in single quotes for POSIX shells, escaping any embedded single quotes safely
func shQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

************************/
