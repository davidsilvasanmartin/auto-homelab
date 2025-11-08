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

func Test_shQuote(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{in: "qqq", out: "'qqq'"},
		{in: "q\"q\"q", out: "'q\"q\"q'"},
		{in: "q'q'q", out: "'q'\"'\"'q'\"'\"'q'"},
		{in: "'", out: "''\"'\"''"},
		{in: "''", out: "''\"'\"''\"'\"''"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := shQuote(tc.in)
			if got != tc.out {
				t.Errorf("got %q, want %q", got, tc.out)
			}
		})
	}
}
