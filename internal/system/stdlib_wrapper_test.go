package system

import (
	"os"
	"os/exec"
)

// mockStdlib is a mock implementation of the stdlib interface
type mockStdlib struct {
	getwd        func() (string, error)
	stat         func(name string) (os.FileInfo, error)
	execCommand  func(name string, arg ...string) *exec.Cmd
	execLookPath func(file string) (string, error)
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

func (m *mockStdlib) ExecCommand(name string, arg ...string) *exec.Cmd {
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
