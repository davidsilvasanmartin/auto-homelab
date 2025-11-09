package backup

import (
	"errors"
	"testing"

	"github.com/davidsilvasanmartin/auto-homelab/internal/system"
)

type mockFilesHandler struct {
	createDirIfNotExists func(path string) error
	requireDir           func(path string) error
	copyDir              func(srcPath string, dstPath string) error
}

func (m *mockFilesHandler) CreateDirIfNotExists(path string) error {
	if m.createDirIfNotExists != nil {
		return m.createDirIfNotExists(path)
	}
	return nil
}
func (m *mockFilesHandler) RequireFilesInWd(filenames ...string) error { return nil }
func (m *mockFilesHandler) RequireDir(path string) error {
	if m.requireDir != nil {
		return m.requireDir(path)
	}
	return nil
}
func (m *mockFilesHandler) EmptyDir(path string) error { return nil }
func (m *mockFilesHandler) CopyDir(srcPath string, dstPath string) error {
	if m.copyDir != nil {
		return m.copyDir(srcPath, dstPath)
	}
	return nil
}
func (m *mockFilesHandler) Getwd() (dir string, err error)           { return "", nil }
func (m *mockFilesHandler) WriteFile(path string, data []byte) error { return nil }

type mockCommands struct {
	execShellCommand func(cmd string) system.RunnableCommand
}

func (m *mockCommands) ExecCommand(name string, arg ...string) system.RunnableCommand { return nil }
func (m *mockCommands) ExecShellCommand(cmd string) system.RunnableCommand {
	if m.execShellCommand != nil {
		return m.execShellCommand(cmd)
	}
	return nil
}

type mockRunnableCommand struct {
	runFunc func() error
}

func (m *mockRunnableCommand) Run() error {
	if m.runFunc != nil {
		return m.runFunc()
	}
	return nil
}

type mockDockerRunner struct {
	containerExec                      func(containerName string, cmd string) error
	waitUntilContainerExecIsSuccessful func(containerName string, cmd string) error
}

func (m *mockDockerRunner) ComposeStart(serviceNames []string) error {
	return nil
}
func (m *mockDockerRunner) ComposeStop(serviceNames []string) error {
	return nil
}
func (m *mockDockerRunner) WaitUntilContainerExecIsSuccessful(containerName string, cmd string) error {
	if m.waitUntilContainerExecIsSuccessful != nil {
		return m.waitUntilContainerExecIsSuccessful(containerName, cmd)
	}
	return nil
}
func (m *mockDockerRunner) ContainerExec(containerName string, cmd string) error {
	if m.containerExec != nil {
		return m.containerExec(containerName, cmd)
	}
	return nil
}

type mockTextFormatter struct{}

func (m *mockTextFormatter) WrapLines(_ string, _ uint) []string                     { return nil }
func (m *mockTextFormatter) FormatDotenvKeyValue(_ string, _ string) (string, error) { return "", nil }
func (m *mockTextFormatter) QuoteForPOSIXShell(key string) string {
	// Mock the functionality of this function. We need to keep this into account in individual tests
	return "'" + key + "'"
}

func TestDirectoryLocalBackup_Run_Success(t *testing.T) {
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		commands:   &mockCommands{},
		srcPath:    "/src",
		preCommand: "",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDirectoryLocalBackup_Run_SuccessWithPreCommand(t *testing.T) {
	var capturedPreCmd string
	var preCmdExecuted bool
	preCmd := "docker exec container-name some-command"
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				capturedPreCmd = cmd
				return &mockRunnableCommand{
					runFunc: func() error {
						preCmdExecuted = true
						return nil
					},
				}
			},
		},
		srcPath:    "/src",
		preCommand: preCmd,
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !preCmdExecuted {
		t.Errorf("expected pre-command to be executed")
	}
	if capturedPreCmd != preCmd {
		t.Errorf("expected pre-command %q, got: %q", preCmd, capturedPreCmd)
	}
}

func TestDirectoryLocalBackup_Run_CreateDirError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files: &mockFilesHandler{
				createDirIfNotExists: func(path string) error {
					return expectedErr
				},
			},
		},
		commands:   &mockCommands{},
		srcPath:    "/src",
		preCommand: "",
	}

	err := backup.Run()

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestDirectoryLocalBackup_Run_PreCommandError(t *testing.T) {
	expectedErr := errors.New("pre-command failed")
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				return &mockRunnableCommand{
					runFunc: func() error {
						return expectedErr
					},
				}
			},
		},
		srcPath:    "/src",
		preCommand: "docker exec container-name failing-command",
	}

	err := backup.Run()

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestDirectoryLocalBackup_Run_RequireDirError(t *testing.T) {
	expectedErr := errors.New("source directory not found")
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files: &mockFilesHandler{
				requireDir: func(path string) error {
					return expectedErr
				},
			},
		},
		commands:   &mockCommands{},
		srcPath:    "/src",
		preCommand: "",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestDirectoryLocalBackup_Run_CopyDirError(t *testing.T) {
	expectedErr := errors.New("copy failed")
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/backup/destination",
			files: &mockFilesHandler{
				copyDir: func(srcPath string, dstPath string) error {
					return expectedErr
				},
			},
		},
		commands:   &mockCommands{},
		srcPath:    "/home/user/data",
		preCommand: "",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestDirectoryLocalBackup_Run_CorrectPathsUsed(t *testing.T) {
	var createdDstPath string
	var requiredSrcPath string
	var copiedSrcPath, copiedDstPath string
	srcPath := "/src"
	dstPath := "/dst"
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: dstPath,
			files: &mockFilesHandler{
				createDirIfNotExists: func(path string) error {
					createdDstPath = path
					return nil
				},
				requireDir: func(path string) error {
					requiredSrcPath = path
					return nil
				},
				copyDir: func(srcPath string, dstPath string) error {
					copiedSrcPath = srcPath
					copiedDstPath = dstPath
					return nil
				},
			},
		},
		commands:   &mockCommands{},
		srcPath:    srcPath,
		preCommand: "",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if createdDstPath != dstPath {
		t.Errorf("expected CreateDirIfNotExists to be called with %q, got %q", dstPath, createdDstPath)
	}
	if requiredSrcPath != srcPath {
		t.Errorf("expected RequireDir to be called with %q, got %q", srcPath, requiredSrcPath)
	}
	if copiedSrcPath != srcPath {
		t.Errorf("expected CopyDir source to be %q, got %q", srcPath, copiedSrcPath)
	}
	if copiedDstPath != dstPath {
		t.Errorf("expected CopyDir destination to be %q, got %q", dstPath, copiedDstPath)
	}
}

func TestDirectoryLocalBackup_Run_PreCommandNotExecutedWhenEmpty(t *testing.T) {
	var preCommandCalled bool
	backup := &DirectoryLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		commands: &mockCommands{
			execShellCommand: func(cmd string) system.RunnableCommand {
				preCommandCalled = true
				return &mockRunnableCommand{}
			},
		},
		srcPath:    "/src",
		preCommand: "", // Empty pre-command
	}

	err := backup.Run()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if preCommandCalled {
		t.Error("expected pre-command not to be called when empty")
	}
}

func TestPostgreSQLLocalBackup_Run_Success(t *testing.T) {
	backup := &PostgreSQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner:  &mockDockerRunner{},
		textFormatter: &mockTextFormatter{},
		containerName: "postgres-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestPostgreSQLLocalBackup_Run_CreateDirError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	backup := &PostgreSQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files: &mockFilesHandler{
				createDirIfNotExists: func(path string) error {
					return expectedErr
				},
			},
		},
		dockerRunner:  &mockDockerRunner{},
		textFormatter: &mockTextFormatter{},
		containerName: "postgres-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestPostgreSQLLocalBackup_Run_WaitUntilContainerExecIsSuccessfulError(t *testing.T) {
	expectedErr := errors.New("database not ready")
	backup := &PostgreSQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			waitUntilContainerExecIsSuccessful: func(containerName string, cmd string) error {
				return expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: "postgres-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestPostgreSQLLocalBackup_Run_ContainerExecError(t *testing.T) {
	expectedErr := errors.New("pg_dump failed")
	backup := &PostgreSQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			containerExec: func(containerName string, cmd string) error {
				return expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: "postgres-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestPostgreSQLLocalBackup_Run_CorrectReadinessCheck(t *testing.T) {
	var capturedContainerName string
	var capturedCmd string
	containerName := "postgres-container"
	backup := &PostgreSQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			waitUntilContainerExecIsSuccessful: func(containerName string, cmd string) error {
				capturedContainerName = containerName
				capturedCmd = cmd
				return nil
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: containerName,
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedContainerName != containerName {
		t.Errorf("expected container name %q, got %q", containerName, capturedContainerName)
	}
	expectedCmd := "pg_isready -q"
	if capturedCmd != expectedCmd {
		t.Errorf("expected readiness check command %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestPostgreSQLLocalBackup_Run_CorrectBackupCommand(t *testing.T) {
	var capturedContainerName string
	var capturedCmd string
	containerName := "postgres-container"
	dbName := "mydb"
	username := "myuser"
	password := "mypass"
	dstPath := "/dst"
	backup := &PostgreSQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: dstPath,
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			containerExec: func(containerName string, cmd string) error {
				capturedContainerName = containerName
				capturedCmd = cmd
				return nil
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedContainerName != containerName {
		t.Errorf("expected container name %q, got %q", containerName, capturedContainerName)
	}
	expectedCmd := `/bin/bash -c "PGPASSWORD='mypass' pg_dump --username myuser mydb" > /dst/mydb.sql`
	if capturedCmd != expectedCmd {
		t.Errorf("expected command:\n%q\ngot:\n%q", expectedCmd, capturedCmd)
	}
}

func TestMySQLLocalBackup_Run_Success(t *testing.T) {
	backup := &MySQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner:  &mockDockerRunner{},
		textFormatter: &mockTextFormatter{},
		containerName: "mysql-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestMySQLLocalBackup_Run_CreateDirError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	backup := &MySQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files: &mockFilesHandler{
				createDirIfNotExists: func(path string) error {
					return expectedErr
				},
			},
		},
		dockerRunner:  &mockDockerRunner{},
		textFormatter: &mockTextFormatter{},
		containerName: "mysql-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestMySQLLocalBackup_Run_WaitUntilContainerExecIsSuccessfulError(t *testing.T) {
	expectedErr := errors.New("database not ready")
	backup := &MySQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			waitUntilContainerExecIsSuccessful: func(containerName string, cmd string) error {
				return expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: "mysql-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestMySQLLocalBackup_Run_ContainerExecError(t *testing.T) {
	expectedErr := errors.New("pg_dump failed")
	backup := &MySQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			containerExec: func(containerName string, cmd string) error {
				return expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: "mysql-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestMySQLLocalBackup_Run_CorrectReadinessCheck(t *testing.T) {
	var capturedContainerName string
	var capturedCmd string
	containerName := "mysql-container"
	backup := &MySQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			waitUntilContainerExecIsSuccessful: func(containerName string, cmd string) error {
				capturedContainerName = containerName
				capturedCmd = cmd
				return nil
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: containerName,
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedContainerName != containerName {
		t.Errorf("expected container name %q, got %q", containerName, capturedContainerName)
	}
	expectedCmd := "mysqladmin ping -h localhost --silent"
	if capturedCmd != expectedCmd {
		t.Errorf("expected readiness check command %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestMySQLLocalBackup_Run_CorrectBackupCommand(t *testing.T) {
	var capturedContainerName string
	var capturedCmd string
	containerName := "mysql-container"
	dbName := "mydb"
	username := "myuser"
	password := "mypass"
	dstPath := "/dst"
	backup := &MySQLLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: dstPath,
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			containerExec: func(containerName string, cmd string) error {
				capturedContainerName = containerName
				capturedCmd = cmd
				return nil
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedContainerName != containerName {
		t.Errorf("expected container name %q, got %q", containerName, capturedContainerName)
	}
	expectedCmd := `/bin/bash -c "MYSQL_PWD='mypass' mysqldump --user myuser mydb" > /dst/mydb.sql`
	if capturedCmd != expectedCmd {
		t.Errorf("expected command:\n%q\ngot:\n%q", expectedCmd, capturedCmd)
	}
}

func TestMariaDBLocalBackup_Run_Success(t *testing.T) {
	backup := &MariaDBLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner:  &mockDockerRunner{},
		textFormatter: &mockTextFormatter{},
		containerName: "mariadb-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestMariaDBLocalBackup_Run_CreateDirError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	backup := &MariaDBLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files: &mockFilesHandler{
				createDirIfNotExists: func(path string) error {
					return expectedErr
				},
			},
		},
		dockerRunner:  &mockDockerRunner{},
		textFormatter: &mockTextFormatter{},
		containerName: "mariadb-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}
}

func TestMariaDBLocalBackup_Run_WaitUntilContainerExecIsSuccessfulError(t *testing.T) {
	expectedErr := errors.New("database not ready")
	backup := &MariaDBLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			waitUntilContainerExecIsSuccessful: func(containerName string, cmd string) error {
				return expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: "mariadb-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestMariaDBLocalBackup_Run_ContainerExecError(t *testing.T) {
	expectedErr := errors.New("pg_dump failed")
	backup := &MariaDBLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			containerExec: func(containerName string, cmd string) error {
				return expectedErr
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: "mariadb-container",
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got: %v", expectedErr, err)
	}
}

func TestMariaDBLocalBackup_Run_CorrectReadinessCheck(t *testing.T) {
	var capturedContainerName string
	var capturedCmd string
	containerName := "mariadb-container"
	backup := &MariaDBLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: "/dst",
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			waitUntilContainerExecIsSuccessful: func(containerName string, cmd string) error {
				capturedContainerName = containerName
				capturedCmd = cmd
				return nil
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: containerName,
		dbName:        "testdb",
		username:      "testuser",
		password:      "testpass",
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedContainerName != containerName {
		t.Errorf("expected container name %q, got %q", containerName, capturedContainerName)
	}
	expectedCmd := "mariadb-admin ping -h localhost --silent"
	if capturedCmd != expectedCmd {
		t.Errorf("expected readiness check command %q, got %q", expectedCmd, capturedCmd)
	}
}

func TestMariaDBLocalBackup_Run_CorrectBackupCommand(t *testing.T) {
	var capturedContainerName string
	var capturedCmd string
	containerName := "mariadb-container"
	dbName := "mydb"
	username := "myuser"
	password := "mypass"
	dstPath := "/dst"
	backup := &MariaDBLocalBackup{
		baseLocalBackup: &baseLocalBackup{
			dstPath: dstPath,
			files:   &mockFilesHandler{},
		},
		dockerRunner: &mockDockerRunner{
			containerExec: func(containerName string, cmd string) error {
				capturedContainerName = containerName
				capturedCmd = cmd
				return nil
			},
		},
		textFormatter: &mockTextFormatter{},
		containerName: containerName,
		dbName:        dbName,
		username:      username,
		password:      password,
	}

	err := backup.Run()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedContainerName != containerName {
		t.Errorf("expected container name %q, got %q", containerName, capturedContainerName)
	}
	expectedCmd := `/bin/bash -c "MYSQL_PWD='mypass' mariadb-dump --user myuser mydb" > /dst/mydb.sql`
	if capturedCmd != expectedCmd {
		t.Errorf("expected command:\n%q\ngot:\n%q", expectedCmd, capturedCmd)
	}
}
