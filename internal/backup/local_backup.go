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
	dstPath  string
	commands system.Commands
	files    system.FilesHandler
	env      system.Env
}

// newBaseLocalBackup creates a new base backup instance
func newBaseLocalBackup(
	dstPath string,
	commands system.Commands,
	files system.FilesHandler,
	env system.Env,
) *baseLocalBackup {
	return &baseLocalBackup{
		dstPath:  dstPath,
		commands: commands,
		files:    files,
		env:      env,
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
	srcPath    string
	preCommand string
}

// NewDirectoryLocalBackup creates a new directory backup instance
func NewDirectoryLocalBackup(
	srcPath, dstPath string,
	preCommand string,
) *DirectoryLocalBackup {
	return &DirectoryLocalBackup{
		baseLocalBackup: newBaseLocalBackup(
			dstPath,
			system.NewDefaultCommands(),
			system.NewDefaultFilesHandler(),
			system.NewDefaultEnv(),
		),
		srcPath:    srcPath,
		preCommand: preCommand,
	}
}

// Run executes the directory backup operation
func (d *DirectoryLocalBackup) Run() (string, error) {
	slog.Info("Running directory local backup", "srcPath", d.srcPath, "dstPath", d.dstPath)
	if err := d.files.CreateDirIfNotExists(d.dstPath); err != nil {
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

	if err := d.files.RequireDir(d.srcPath); err != nil {
		return "", err
	}

	if err := d.files.CopyDir(d.srcPath, d.dstPath); err != nil {
		return "", err
	}

	slog.Info("Directory local backup ran successfully", "srcPath", d.srcPath, "dstPath", d.dstPath)
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
func NewPostgreSQLLocalBackup(containerName, dbName, username, password, dstPath string) *PostgreSQLLocalBackup {
	return &PostgreSQLLocalBackup{
		baseLocalBackup: newBaseLocalBackup(
			dstPath,
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
	slog.Info("Running PostgreSQL local backup", "containerName", p.containerName, "dbName", p.dbName, "dstPath", p.dstPath)
	if err := p.files.CreateDirIfNotExists(p.dstPath); err != nil {
		return "", err
	}

	backupFile := filepath.Join(p.dstPath, p.dbName+".sql")

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
		return "", fmt.Errorf("error backing up PostgreSQL database %s: %w", p.dbName, err)
	}

	slog.Info("PostgreSQL local backup ran successfully", "containerName", p.containerName, "dbName", p.dbName, "dstPath", p.dstPath)
	return backupFile, nil
}

// shQuote Wrap a string in single quotes for POSIX shells, escaping any embedded single quotes safely.
// This transforms p'q into 'p'"'"'q' which the shell interprets as a single literal string.
func shQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// MySQLBackup handles MySQL database backups using docker exec
type MySQLBackup struct {
	*baseLocalBackup
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMySQLBackup creates a new MySQL backup instance
func NewMySQLBackup(containerName, dbName, username, password, dstPath string) *MySQLBackup {
	return &MySQLBackup{
		baseLocalBackup: newBaseLocalBackup(
			dstPath,
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

// Run executes the MySQL backup
func (m *MySQLBackup) Run() (string, error) {
	slog.Info("Running MySQL local backup", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath)
	if err := m.files.CreateDirIfNotExists(m.dstPath); err != nil {
		return "", err
	}

	backupFile := filepath.Join(m.dstPath, m.dbName+".sql")

	quotedPassword := shQuote(m.password)
	dockerCommand := fmt.Sprintf(
		`docker exec -i %s /bin/bash -c "MYSQL_PWD=%s mysqldump --user %s %s" > %s`,
		m.containerName,
		quotedPassword,
		m.username,
		m.dbName,
		backupFile,
	)
	cmd := m.commands.ExecShellCommand(dockerCommand)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up MySQL database %s: %w", m.dbName, err)
	}

	slog.Info("MySQL local backup ran successfully", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath)
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
func NewMariaDBBackup(containerName, dbName, username, password, dstPath string) *MariaDBBackup {
	return &MariaDBBackup{
		baseLocalBackup: newBaseLocalBackup(
			dstPath,
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

// Run executes the MariaDB backup
func (m *MariaDBBackup) Run() (string, error) {
	slog.Info("Running MariaDB local backup", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath)
	if err := m.files.CreateDirIfNotExists(m.dstPath); err != nil {
		return "", err
	}

	backupFile := filepath.Join(m.dstPath, m.dbName+".sql")

	quotedPassword := shQuote(m.password)
	dockerCommand := fmt.Sprintf(
		`docker exec -i %s /bin/bash -c "MYSQL_PWD=%s mariadb-dump --user %s %s" > %s`,
		m.containerName,
		quotedPassword,
		m.username,
		m.dbName,
		backupFile,
	)
	cmd := m.commands.ExecShellCommand(dockerCommand)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error backing up MariaDB database %s: %w", m.dbName, err)
	}

	slog.Info("MariaDB local backup ran successfully", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath)
	return backupFile, nil
}
