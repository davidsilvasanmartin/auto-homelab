package backup

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/davidsilvasanmartin/auto-homelab/internal/docker"
	"github.com/davidsilvasanmartin/auto-homelab/internal/format"
	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

// LocalBackup is the interface for all backup operations
type LocalBackup interface {
	// Run executes the backup operation
	Run() error
}

// baseLocalBackup contains common backup functionality
type baseLocalBackup struct {
	dstPath string
	files   system.FilesHandler
}

// newBaseLocalBackup creates a new base backup instance
func newBaseLocalBackup(
	dstPath string,
	files system.FilesHandler,
) *baseLocalBackup {
	return &baseLocalBackup{
		dstPath: dstPath,
		files:   files,
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////
///// SPECIFIC BACKUPS below
///////////////////////////////////////////////////////////////////////////////////////////////////////

// DirectoryLocalBackup handles directory copy operations
type DirectoryLocalBackup struct {
	*baseLocalBackup
	commands system.Commands
	srcPath  string
	// TODO the design of this could be better. The preCommand will typically be "docker exec something",
	//  and at that point we have missed all the nice abstractions we have made on top of Docker with the
	//  docker.Runner interface
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
			system.NewDefaultFilesHandler(),
		),
		commands:   system.NewDefaultCommands(),
		srcPath:    srcPath,
		preCommand: preCommand,
	}
}

// Run executes the directory backup operation
func (d *DirectoryLocalBackup) Run() error {
	slog.Info("Running directory local backup", "srcPath", d.srcPath, "dstPath", d.dstPath)
	if err := d.files.CreateDirIfNotExists(d.dstPath); err != nil {
		return err
	}

	if d.preCommand != "" {
		slog.Info("Running pre-command", "preCommand", d.preCommand)
		cmd := d.commands.ExecShellCommand(d.preCommand)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pre-command failed: %w", err)
		}
		slog.Info("Successfully ran pre-command", "preCommand", d.preCommand)
	}

	if err := d.files.RequireDir(d.srcPath); err != nil {
		return err
	}

	if err := d.files.CopyDir(d.srcPath, d.dstPath); err != nil {
		return err
	}

	slog.Info("Directory local backup ran successfully", "srcPath", d.srcPath, "dstPath", d.dstPath)
	return nil
}

// PostgreSQLLocalBackup handles PostgreSQL database backups using docker exec
type PostgreSQLLocalBackup struct {
	*baseLocalBackup
	dockerRunner  docker.Runner
	textFormatter format.TextFormatter
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
			system.NewDefaultFilesHandler(),
		),
		dockerRunner:  docker.NewSystemRunner(),
		textFormatter: format.NewDefaultTextFormatter(),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the PostgreSQL backup
func (p *PostgreSQLLocalBackup) Run() error {
	slog.Info("Running PostgreSQL local backup", "containerName", p.containerName, "dbName", p.dbName, "dstPath", p.dstPath)
	if err := p.files.CreateDirIfNotExists(p.dstPath); err != nil {
		return err
	}

	backupFile := filepath.Join(p.dstPath, p.dbName+".sql")

	if err := p.dockerRunner.WaitUntilContainerExecIsSuccessful(p.containerName, "pg_isready -q"); err != nil {
		return fmt.Errorf("PostgreSQL database %s not ready: %w", p.dbName, err)
	}

	quotedPassword := p.textFormatter.QuoteForPOSIXShell(p.password)
	containerCmd := fmt.Sprintf(
		`/bin/bash -c "PGPASSWORD=%s pg_dump --username %s %s" > %s`,
		quotedPassword,
		p.username,
		p.dbName,
		backupFile,
	)
	err := p.dockerRunner.ContainerExec(p.containerName, containerCmd)
	if err != nil {
		return fmt.Errorf("error backing up PostgreSQL database %s: %w", p.dbName, err)
	}

	slog.Info("PostgreSQL local backup ran successfully", "containerName", p.containerName, "dbName", p.dbName, "dstPath", p.dstPath, "backupFile", backupFile)
	return nil
}

// MySQLLocalBackup handles MySQL database backups using docker exec
type MySQLLocalBackup struct {
	*baseLocalBackup
	dockerRunner  docker.Runner
	textFormatter format.TextFormatter
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMySQLLocalBackup creates a new MySQL backup instance
func NewMySQLLocalBackup(containerName, dbName, username, password, dstPath string) *MySQLLocalBackup {
	return &MySQLLocalBackup{
		baseLocalBackup: newBaseLocalBackup(
			dstPath,
			system.NewDefaultFilesHandler(),
		),
		dockerRunner:  docker.NewSystemRunner(),
		textFormatter: format.NewDefaultTextFormatter(),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the MySQL backup
func (m *MySQLLocalBackup) Run() error {
	slog.Info("Running MySQL local backup", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath)
	if err := m.files.CreateDirIfNotExists(m.dstPath); err != nil {
		return err
	}

	backupFile := filepath.Join(m.dstPath, m.dbName+".sql")

	if err := m.dockerRunner.WaitUntilContainerExecIsSuccessful(m.containerName, "mysqladmin ping -h localhost --silent"); err != nil {
		return fmt.Errorf("MySQL database %s not ready: %w", m.dbName, err)
	}

	quotedPassword := m.textFormatter.QuoteForPOSIXShell(m.password)
	containerCmd := fmt.Sprintf(
		`/bin/bash -c "MYSQL_PWD=%s mysqldump --user %s %s" > %s`,
		quotedPassword,
		m.username,
		m.dbName,
		backupFile,
	)
	if err := m.dockerRunner.ContainerExec(m.containerName, containerCmd); err != nil {
		return fmt.Errorf("error backing up MySQL database %s: %w", m.dbName, err)
	}

	slog.Info("MySQL local backup ran successfully", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath, "backupFile", backupFile)
	return nil
}

// MariaDBLocalBackup handles MariaDB database backups using docker exec
type MariaDBLocalBackup struct {
	*baseLocalBackup
	dockerRunner  docker.Runner
	textFormatter format.TextFormatter
	containerName string
	dbName        string
	username      string
	password      string
}

// NewMariaDBLocalBackup creates a new MariaDB backup instance
func NewMariaDBLocalBackup(containerName, dbName, username, password, dstPath string) *MariaDBLocalBackup {
	return &MariaDBLocalBackup{
		baseLocalBackup: newBaseLocalBackup(
			dstPath,
			system.NewDefaultFilesHandler(),
		),
		dockerRunner:  docker.NewSystemRunner(),
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}
}

// Run executes the MariaDB backup
func (m *MariaDBLocalBackup) Run() error {
	slog.Info("Running MariaDB local backup", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath)
	if err := m.files.CreateDirIfNotExists(m.dstPath); err != nil {
		return err
	}

	backupFile := filepath.Join(m.dstPath, m.dbName+".sql")

	if err := m.dockerRunner.WaitUntilContainerExecIsSuccessful(m.containerName, "mariadb-admin ping -h localhost --silent"); err != nil {
		return fmt.Errorf("MariaDB database %s not ready: %w", m.dbName, err)
	}

	quotedPassword := m.textFormatter.QuoteForPOSIXShell(m.password)
	containerCmd := fmt.Sprintf(
		`/bin/bash -c "MYSQL_PWD=%s mariadb-dump --user %s %s" > %s`,
		quotedPassword,
		m.username,
		m.dbName,
		backupFile,
	)
	if err := m.dockerRunner.ContainerExec(m.containerName, containerCmd); err != nil {
		return fmt.Errorf("error backing up MariaDB database %s: %w", m.dbName, err)
	}

	slog.Info("MariaDB local backup ran successfully", "containerName", m.containerName, "dbName", m.dbName, "dstPath", m.dstPath, "backupFile", backupFile)
	return nil
}
