package system

import (
	"os"
	"time"
)

// mockRunnableCommand is a simple mock for RunnableCommand
type mockRunnableCommand struct {
	runFunc func() error
}

func (m *mockRunnableCommand) Run() error {
	if m.runFunc != nil {
		return m.runFunc()
	}
	return nil
}

// mockStdlib is a mock implementation of the stdlib interface
type mockStdlib struct {
	getwd        func() (string, error)
	stat         func(name string) (os.FileInfo, error)
	execCommand  func(name string, arg ...string) RunnableCommand
	execLookPath func(file string) (string, error)
	mkdirAll     func(path string, mode os.FileMode) error
}

func (m *mockStdlib) Getwd() (string, error) {
	if m.getwd != nil {
		return m.getwd()
	}
	return "", nil
}

func (m *mockStdlib) Stat(name string) (os.FileInfo, error) {
	if m.stat != nil {
		return m.stat(name)
	}
	return nil, nil
}

func (m *mockStdlib) ExecCommand(name string, arg ...string) RunnableCommand {
	if m.execCommand != nil {
		return m.execCommand(name, arg...)
	}
	return nil
}

func (m *mockStdlib) ExecLookPath(file string) (string, error) {
	if m.execLookPath != nil {
		return m.execLookPath(file)
	}
	return "", nil
}

func (m *mockStdlib) MkdirAll(path string, perm os.FileMode) error {
	if m.mkdirAll != nil {
		return m.mkdirAll(path, perm)
	}
	return nil
}

// mockFileInfo is a mock implementation of os.FileInfo for testing
type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }
