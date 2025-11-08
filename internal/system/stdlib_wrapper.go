package system

import (
	"os"
	"os/exec"
	"time"
)

type RunnableCommand interface {
	Run() error
}

// stdlib defines the os operations we NEED. Wraps go std lib functions
type stdlib interface {
	// Getwd wraps os.Getwd
	Getwd() (string, error)
	// Stat wraps os.Stat
	Stat(name string) (os.FileInfo, error)
	// ExecCommand wraps exec.Cmd
	ExecCommand(name string, arg ...string) RunnableCommand
	// ExecLookPath wraps exec.LookPath
	ExecLookPath(file string) (string, error)
	// MkdirAll wraps os.MkdirAll
	MkdirAll(path string, perm os.FileMode) error
	// RemoveAll wraps os.RemoveAll
	RemoveAll(path string) error
	// Sleep wraps time.Sleep
	Sleep(d time.Duration)
}

// goStdlib implements stdlib by using the real go's std
type goStdlib struct{}

// newGoStdlib creates a new instance of the goStdlib struct
func newGoStdlib() *goStdlib {
	return &goStdlib{}
}

func (*goStdlib) Getwd() (string, error) {
	return os.Getwd()
}

func (*goStdlib) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// ExecCommand uses the system's os.Stdout and os.Stderr. It uses the current working directory
// as the directory where the commands are run from
func (*goStdlib) ExecCommand(name string, arg ...string) RunnableCommand {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "."
	return cmd
}

func (*goStdlib) ExecLookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (*goStdlib) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

func (*goStdlib) RemoveAll(path string) error { return os.RemoveAll(path) }

func (*goStdlib) Sleep(d time.Duration) { time.Sleep(d) }
