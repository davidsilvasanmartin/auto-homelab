package backup

import (
	"fmt"
	"log/slog"

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
	outputPath               string
	commands                 system.Commands
	files                    system.FilesHandler
	env                      system.Env
	requiresServicesEnabled  []string
	requiresServicesDisabled []string
}

// NewBaseBackup creates a new base backup instance
func NewBaseBackup(
	outputPath string,
	commands system.Commands,
	files system.FilesHandler,
	env system.Env,
	requiresServicesEnabled []string, requiresServicesDisabled []string,
) *BaseBackup {
	return &BaseBackup{
		outputPath:               outputPath,
		commands:                 commands,
		files:                    files,
		env:                      env,
		requiresServicesEnabled:  requiresServicesEnabled,
		requiresServicesDisabled: requiresServicesDisabled,
	}
}

type ServiceState int

const (
	ServiceStateEnabled = iota
	ServiceStateDisabled
)

type BackupList struct {
	Backups                   []BaseBackup
	serviceStatesDuringBackup map[ServiceState]map[string]bool
}

func NewBackupList() *BackupList {
	states := make(map[ServiceState]map[string]bool)
	states[ServiceStateEnabled] = make(map[string]bool)
	states[ServiceStateDisabled] = make(map[string]bool)
	return &BackupList{
		Backups:                   []BaseBackup{},
		serviceStatesDuringBackup: states,
	}
}

func (l *BackupList) Add(backup BaseBackup) {
	l.Backups = append(l.Backups, backup)
}

func (l *BackupList) Prepare() error {
	for _, b := range l.Backups {
		mustEnableServices := b.requiresServicesEnabled
		for _, s := range mustEnableServices {
			disabled := l.serviceStatesDuringBackup[ServiceStateDisabled]
			if _, ok := disabled[s]; ok == true {
				return fmt.Errorf("service %q cannot be enabled and disabled at the same time", s)
			}
			enabled := l.serviceStatesDuringBackup[ServiceStateEnabled]
			enabled[s] = true
		}

		mustDisableServices := b.requiresServicesDisabled
		for _, s := range mustDisableServices {
			enabled := l.serviceStatesDuringBackup[ServiceStateEnabled]
			if _, ok := enabled[s]; ok == true {
				return fmt.Errorf("service %q cannot be enabled and disabled at the same time", s)
			}
			disabled := l.serviceStatesDuringBackup[ServiceStateDisabled]
			disabled[s] = true
		}
	}
	return nil
}

func (l *BackupList) Run() {
	// TODO 1. Enable/disable services as per requirements
	// TODO 2. Run local backups asynchronously (like it's being done now)
	// TODO 3. Turn services that have been disabled back up
}

///////////////////////////////////////////////////////////////////////////////////////////////////////
///// SPECIFIC BACKUPS below
///////////////////////////////////////////////////////////////////////////////////////////////////////

// DirectoryBackup handles directory copy operations
// ⚠️⚠️⚠️ WARNING!! preCommand and postCommand run concurrently. We need to keep this in mind when
// writing Backup operations. E.g., we can't use preCommand="docker compose stop service"
// and postCommand="docker compose start service" on several Backup operations, because
// they will intermix and run in unspecified order
// TODO delete the above warning when done with new code
type DirectoryBackup struct {
	*BaseBackup
	sourcePath string
	preCommand string
}

// NewDirectoryBackup creates a new directory backup instance
func NewDirectoryBackup(
	sourcePath, outputPath string,
	preCommand string,
	commands system.Commands,
	files system.FilesHandler,
	env system.Env,
	requiresServicesEnabled, requiresServicesDisabled []string,
) *DirectoryBackup {
	return &DirectoryBackup{
		BaseBackup: NewBaseBackup(outputPath, commands, files, env, requiresServicesEnabled, requiresServicesDisabled),
		sourcePath: sourcePath,
		preCommand: preCommand,
	}
}

// Run executes the directory backup operation
func (d *DirectoryBackup) Run() (string, error) {
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
		slog.Info("Successfully ran pre-command")
	}

	if err := d.files.RequireDir(d.sourcePath); err != nil {
		return "", err
	}

	if err := d.files.CopyDir(d.sourcePath, d.outputPath); err != nil {
		return "", err
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
func NewPostgreSQLBackup(containerName, dbName, username, password, outputPath string, sys commands.Commands, stdout, stderr io.Writer) *PostgreSQLBackup {
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

	cmd := p.commands.ExecCommand("sh", "-c", dockerCommand)
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
func NewMySQLBackup(containerName, dbName, username, password, outputPath string, sys commands.Commands, stdout, stderr io.Writer) *MySQLBackup {
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
	*BaseBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMariaDBBackup creates a new MariaDB backup instance
func NewMariaDBBackup(containerName, dbName, username, password, outputPath string, sys commands.Commands, stdout, stderr io.Writer) *MariaDBBackup {
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

	cmd := m.commands.ExecCommand("sh", "-c", dockerCommand)
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
