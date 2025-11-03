package backup

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// TODO think about prefixing things with "Local"

// LocalBackup is the interface for all backup operations
type LocalBackup interface {
	// Run executes the backup operation and returns the path to the backup
	Run() (string, error)
}

// baseLocalBackup contains common backup functionality
type baseLocalBackup struct {
	outputPath string
	commands   system.Commands
	files      system.FilesHandler
	env        system.Env
}

// newBaseLocalBackup creates a new base backup instance
func newBaseLocalBackup(
	outputPath string,
	commands system.Commands,
	files system.FilesHandler,
	env system.Env,
) *baseLocalBackup {
	return &baseLocalBackup{
		outputPath: outputPath,
		commands:   commands,
		files:      files,
		env:        env,
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////
///// SPECIFIC BACKUPS below
///////////////////////////////////////////////////////////////////////////////////////////////////////

// DirectoryLocalBackup handles directory copy operations
// ⚠️⚠️⚠️ WARNING!! preCommand and postCommand run concurrently. We need to keep this in mind when
// writing LocalBackup operations. E.g., we can't use preCommand="docker compose stop service"
// and postCommand="docker compose start service" on several LocalBackup operations, because
// they will intermix and run in unspecified order
// TODO delete the above warning when done with new code
type DirectoryLocalBackup struct {
	*baseLocalBackup
	sourcePath string
	preCommand string
}

// NewDirectoryLocalBackup creates a new directory backup instance
func NewDirectoryLocalBackup(
	sourcePath, outputPath string,
	preCommand string,
) *DirectoryLocalBackup {
	return &DirectoryLocalBackup{
		baseLocalBackup: newBaseLocalBackup(
			outputPath,
			system.NewDefaultCommands(),
			system.NewDefaultFilesHandler(),
			system.NewDefaultEnv(),
		),
		sourcePath: sourcePath,
		preCommand: preCommand,
	}
}

// Run executes the directory backup operation
func (d *DirectoryLocalBackup) Run() (string, error) {
	slog.Info("Running directory local backup", "srcPath", d.sourcePath, "dstPath", d.outputPath)
	if err := d.files.CreateDirIfNotExists(d.outputPath); err != nil {
		return "", err
	}

	// Run pre-command if provided
	if d.preCommand != "" {
		slog.Info("Running pre-command", "preCommand", d.preCommand)
		cmd := d.commands.ExecShellCommand(d.preCommand)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("pre-command failed: %w", err)
		}
		slog.Info("Successfully ran pre-command", "preCommand", d.preCommand)
	}

	if err := d.files.RequireDir(d.sourcePath); err != nil {
		return "", err
	}

	if err := d.files.CopyDir(d.sourcePath, d.outputPath); err != nil {
		return "", err
	}

	slog.Info("Directory local backup ran successfully", "srcPath", d.sourcePath, "dstPath", d.outputPath)
	return "", nil
}

// PostgreSQLLocalBackup handles PostgreSQL database backups using docker exec
type PostgreSQLLocalBackup struct {
	*baseLocalBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewPostgreSQLLocalBackup creates a new PostgreSQL backup instance
func NewPostgreSQLLocalBackup(containerName, dbName, username, password, outputPath string) *PostgreSQLLocalBackup {
	return &PostgreSQLLocalBackup{
		baseLocalBackup: newBaseLocalBackup(
			outputPath,
			system.NewDefaultCommands(),
			system.NewDefaultFilesHandler(),
			system.NewDefaultEnv(),
		),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the PostgreSQL backup
func (p *PostgreSQLLocalBackup) Run() (string, error) {
	slog.Info("Running PostgreSQL local backup", "containerName", p.containerName, "dbName", p.dbName, "dstPath", p.outputPath)
	if err := p.files.CreateDirIfNotExists(p.outputPath); err != nil {
		return "", err
	}

	backupFile := filepath.Join(p.outputPath, p.dbName+".sql")

	quotedPassword := shQuote(p.password)
	dockerCommand := fmt.Sprintf(
		`docker exec -i %s /bin/bash -c "PGPASSWORD=%s pg_dump --username %s %s" > %s`,
		p.containerName,
		quotedPassword,
		p.username,
		p.dbName,
		backupFile,
	)
	cmd := p.commands.ExecShellCommand(dockerCommand)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up database %s: %w", p.dbName, err)
	}

	slog.Info("PostgreSQL local backup ran successfully", "containerName", p.containerName, "dbName", p.dbName, "dstPath", p.outputPath)
	return backupFile, nil
}

// shQuote Wrap a string in single quotes for POSIX shells, escaping any embedded single quotes safely.
// This transforms p'q into 'p'"'"'q' which the shell interprets as a single literal string.
func shQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

/* TODO FROM HERE ********************************************************/
/*********************************

// MySQLBackup handles MySQL database backups using docker exec
type MySQLBackup struct {
	*baseLocalBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMySQLBackup creates a new MySQL backup instance
func NewMySQLBackup(containerName, dbName, username, password, outputPath string, sys commands.Commands, stdout, stderr io.Writer) *MySQLBackup {
	return &MySQLBackup{
		baseLocalBackup:    newBaseLocalBackup(outputPath, sys, stdout, stderr),
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

	cmd := m.commands.ExecCommand("sh", "-c", dockerCommand)
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
	*baseLocalBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMariaDBBackup creates a new MariaDB backup instance
func NewMariaDBBackup(containerName, dbName, username, password, outputPath string, sys commands.Commands, stdout, stderr io.Writer) *MariaDBBackup {
	return &MariaDBBackup{
		baseLocalBackup:    newBaseLocalBackup(outputPath, sys, stdout, stderr),
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

	cmd := m.commands.ExecCommand("sh", "-c", dockerCommand)
	cmd.Stdout = m.stdout
	cmd.Stderr = m.stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up MariaDB database %s: %w", m.dbName, err)
	}

	fmt.Fprintf(m.stdout, "Successfully backed up %s to %s\n\n", m.dbName, backupFile)
	return backupFile, nil
}
************************/
