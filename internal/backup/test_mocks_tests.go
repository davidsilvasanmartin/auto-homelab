package backup

import "github.com/davidsilvasanmartin/auto-homelab/internal/system"

type mockFilesHandler struct {
	createDirIfNotExists func(path string) error
	ensureDirExists      func(path string) error
	copyDir              func(srcPath string, dstPath string) error
	getAbsPath           func(path string) (string, error)
}

func (m *mockFilesHandler) CreateDirIfNotExists(path string) error {
	if m.createDirIfNotExists != nil {
		return m.createDirIfNotExists(path)
	}
	return nil
}
func (m *mockFilesHandler) EnsureFilesInWD(filenames ...string) error { return nil }
func (m *mockFilesHandler) EnsureDirExists(path string) error {
	if m.ensureDirExists != nil {
		return m.ensureDirExists(path)
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
func (m *mockFilesHandler) GetAbsPath(path string) (string, error) {
	if m.getAbsPath != nil {
		return m.getAbsPath(path)
	}
	return path, nil
}

type mockTextFormatter struct{}

func (m *mockTextFormatter) WrapLines(_ string, _ uint) []string                     { return nil }
func (m *mockTextFormatter) FormatDotenvKeyValue(_ string, _ string) (string, error) { return "", nil }
func (m *mockTextFormatter) QuoteForPOSIXShell(key string) string {
	// Mock the functionality of this function. We need to keep this into account in individual tests
	return "'" + key + "'"
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
